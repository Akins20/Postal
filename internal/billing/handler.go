package billing

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/workspace"
)

// maxWebhookBody bounds payment-webhook payload reads.
const maxWebhookBody = 1 << 20

// ledgerPageSize is the fixed ledger page length.
const ledgerPageSize = 50

// Handler serves wallet endpoints. Workspace routes are capability-gated
// (read for wallet/ledger, manage_workspace for top-ups); webhooks are public
// and authenticated purely by their provider signature over the raw body.
type Handler struct {
	svc      *Service
	wsSvc    *workspace.Service
	stripe   *StripeProvider
	paystack *PaystackProvider
	log      *slog.Logger
}

// NewHandler builds the billing HTTP handler. stripe/paystack may be nil when
// that provider isn't configured (its webhook then 404s).
func NewHandler(svc *Service, wsSvc *workspace.Service, stripe *StripeProvider, paystack *PaystackProvider, log *slog.Logger) *Handler {
	return &Handler{svc: svc, wsSvc: wsSvc, stripe: stripe, paystack: paystack, log: log}
}

// RegisterWorkspaceScoped registers wallet routes onto a
// /workspaces/{workspaceID} router.
func (h *Handler) RegisterWorkspaceScoped(r chi.Router) {
	r.Route("/billing", func(br chi.Router) {
		br.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/wallet", web.Handler(h.log, h.wallet))
		br.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/ledger", web.Handler(h.log, h.ledger))
		br.With(workspace.RequireCapability(h.wsSvc, workspace.CapManageWorkspace, h.log)).Post("/topup", web.Handler(h.log, h.topup))
	})
}

// RegisterPublic registers the payment webhooks on the public API router.
func (h *Handler) RegisterPublic(r chi.Router) {
	r.Post("/billing/webhooks/stripe", web.Handler(h.log, h.stripeWebhook))
	r.Post("/billing/webhooks/paystack", web.Handler(h.log, h.paystackWebhook))
}

func (h *Handler) wallet(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	wallet, err := h.svc.Wallet(r.Context(), wsID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, wallet)
	return nil
}

func (h *Handler) ledger(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	entries, err := h.svc.Ledger(r.Context(), wsID, ledgerPageSize, 0)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, entries)
	return nil
}

func (h *Handler) topup(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	var in struct {
		Provider string `json:"provider"`
		Credits  int64  `json:"credits"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&in); err != nil {
		return apperr.Validation("invalid_json", "could not parse request body")
	}
	url, err := h.svc.CreateCheckout(r.Context(), wsID, in.Provider, in.Credits)
	switch {
	case errors.Is(err, ErrProviderUnavailable):
		return apperr.Validation("provider_unavailable", err.Error())
	case errors.Is(err, ErrBadTopup):
		return apperr.Validation("invalid_topup", err.Error())
	case err != nil:
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"checkout_url": url})
	return nil
}

func (h *Handler) stripeWebhook(w http.ResponseWriter, r *http.Request) error {
	if h.stripe == nil {
		return apperr.NotFound("provider_disabled", "stripe is not configured")
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
	if err != nil {
		return apperr.Validation("unreadable_body", "could not read webhook body")
	}
	evt, err := h.stripe.VerifyWebhook(payload, r.Header.Get("Stripe-Signature"))
	if err != nil {
		return apperr.Unauthorized("bad_signature", "webhook signature verification failed")
	}
	return h.applyTopup(w, r, evt)
}

func (h *Handler) paystackWebhook(w http.ResponseWriter, r *http.Request) error {
	if h.paystack == nil {
		return apperr.NotFound("provider_disabled", "paystack is not configured")
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
	if err != nil {
		return apperr.Validation("unreadable_body", "could not read webhook body")
	}
	evt, err := h.paystack.VerifyWebhook(payload, r.Header.Get("X-Paystack-Signature"))
	if err != nil {
		return apperr.Unauthorized("bad_signature", "webhook signature verification failed")
	}
	return h.applyTopup(w, r, evt)
}

// applyTopup credits a verified top-up event (nil = verified but irrelevant
// event type; acknowledged so the provider stops retrying).
func (h *Handler) applyTopup(w http.ResponseWriter, r *http.Request, evt *TopupEvent) error {
	if evt == nil {
		web.Respond(w, http.StatusOK, map[string]string{"status": "ignored"})
		return nil
	}
	wsID, err := uuid.Parse(evt.WorkspaceID)
	if err != nil {
		return apperr.Validation("bad_workspace", "event carries an invalid workspace id")
	}
	applied, err := h.svc.Credit(r.Context(), wsID, KindTopup, evt.Credits, evt.Reference, "wallet top-up")
	if err != nil {
		return err
	}
	status := "credited"
	if !applied {
		status = "duplicate"
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": status})
	return nil
}
