package media

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/workspace"
)

// multipartMemory bounds in-memory multipart buffering before spilling to disk.
const multipartMemory = 16 << 20

// Handler serves media endpoints under /workspaces/{workspaceID}/media. Upload
// needs the upload capability, read for list/download, delete for delete.
type Handler struct {
	svc       *Service
	wsSvc     *workspace.Service
	log       *slog.Logger
	maxUpload int64
}

// NewHandler builds the media HTTP handler.
func NewHandler(svc *Service, wsSvc *workspace.Service, log *slog.Logger, maxUpload int64) *Handler {
	return &Handler{svc: svc, wsSvc: wsSvc, log: log, maxUpload: maxUpload}
}

// RegisterWorkspaceScoped registers media routes onto a /workspaces/{workspaceID} router.
func (h *Handler) RegisterWorkspaceScoped(r chi.Router) {
	r.Route("/media", func(mr chi.Router) {
		mr.With(workspace.RequireCapability(h.wsSvc, workspace.CapUpload, h.log)).Post("/", web.Handler(h.log, h.upload))
		mr.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/", web.Handler(h.log, h.list))
		mr.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/{mediaID}/download", web.Handler(h.log, h.download))
		mr.With(workspace.RequireCapability(h.wsSvc, workspace.CapDelete, h.log)).Delete("/{mediaID}", web.Handler(h.log, h.delete))
	})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	// Bound the request body (multipart overhead + the file).
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload+(1<<20))
	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		return apperr.Validation("invalid_upload", "could not parse multipart upload")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return apperr.Validation("missing_file", "expected a 'file' form field")
	}
	defer func() { _ = file.Close() }()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		return apperr.Validation("missing_content_type", "the uploaded file must declare a Content-Type")
	}
	asset, err := h.svc.Upload(r.Context(), wsID, contentType, file, header.Size)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusCreated, asset)
	return nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	assets, err := h.svc.List(r.Context(), wsID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, assets)
	return nil
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	mediaID, err := web.PathUUID(r, "mediaID")
	if err != nil {
		return err
	}
	rc, asset, err := h.svc.Download(r.Context(), wsID, mediaID)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	w.Header().Set("Content-Type", asset.MIME)
	w.Header().Set("Content-Length", strconv.FormatInt(asset.Bytes, 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc) // headers already sent; a copy error can't be reported
	return nil
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	mediaID, err := web.PathUUID(r, "mediaID")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(r.Context(), wsID, mediaID); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"message": "media deleted"})
	return nil
}
