package post

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
)

// fakeMediaResolver returns fixed authoritative metadata for any asset.
type fakeMediaResolver struct {
	kind, mime string
	bytes      int64
	err        error
}

func (f fakeMediaResolver) ResolveMedia(context.Context, uuid.UUID, uuid.UUID) (string, string, int64, error) {
	return f.kind, f.mime, f.bytes, f.err
}

func TestResolveMedia(t *testing.T) {
	ws := uuid.New()

	t.Run("no media -> nil", func(t *testing.T) {
		s := &Service{media: fakeMediaResolver{}}
		out, err := s.resolveMedia(context.Background(), ws, nil)
		if err != nil || out != nil {
			t.Fatalf("expected (nil,nil), got (%v,%v)", out, err)
		}
	})

	t.Run("media attached but pipeline disabled -> validation", func(t *testing.T) {
		s := &Service{media: nil}
		_, err := s.resolveMedia(context.Background(), ws, []MediaMeta{{MediaID: uuid.New()}})
		if apperr.KindOf(err) != apperr.KindValidation {
			t.Fatalf("expected validation, got %v", err)
		}
	})

	t.Run("media without an upload id -> validation", func(t *testing.T) {
		s := &Service{media: fakeMediaResolver{kind: "image", mime: "image/png", bytes: 10}}
		_, err := s.resolveMedia(context.Background(), ws, []MediaMeta{{Kind: "image", Bytes: 1}})
		if apperr.KindOf(err) != apperr.KindValidation {
			t.Fatalf("expected media_unresolved validation, got %v", err)
		}
	})

	t.Run("resolved media overrides client values", func(t *testing.T) {
		s := &Service{media: fakeMediaResolver{kind: "image", mime: "image/png", bytes: 4242}}
		out, err := s.resolveMedia(context.Background(), ws, []MediaMeta{{MediaID: uuid.New(), Kind: "video", MIME: "video/mp4", Bytes: 1}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out) != 1 || out[0].Kind != "image" || out[0].MIME != "image/png" || out[0].Bytes != 4242 {
			t.Fatalf("client values not overridden: %+v", out)
		}
	})

	t.Run("foreign/unknown asset error propagates", func(t *testing.T) {
		s := &Service{media: fakeMediaResolver{err: apperr.NotFound("media_not_found", "nope")}}
		_, err := s.resolveMedia(context.Background(), ws, []MediaMeta{{MediaID: uuid.New()}})
		if apperr.KindOf(err) != apperr.KindNotFound {
			t.Fatalf("expected not-found, got %v", err)
		}
	})
}
