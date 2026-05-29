package auth

import (
	"net/http"

	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/ratelimit"
)

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type resetRequestRequest struct {
	Email string `json:"email"`
}

type resetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) error {
	var req verifyEmailRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := h.svc.VerifyEmail(r.Context(), req.Token); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"message": "email verified"})
	return nil
}

func (h *Handler) requestReset(w http.ResponseWriter, r *http.Request) error {
	var req resetRequestRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	// Always returns nil unless an internal error occurs (no account enumeration).
	if err := h.svc.RequestPasswordReset(r.Context(), req.Email, ratelimit.ClientIPKey(r)); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{
		"message": "if that email is registered, a reset link has been sent",
	})
	return nil
}

func (h *Handler) confirmReset(w http.ResponseWriter, r *http.Request) error {
	var req resetConfirmRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword, ratelimit.ClientIPKey(r)); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"message": "password updated"})
	return nil
}
