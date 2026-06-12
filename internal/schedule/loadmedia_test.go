package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/publish"
)

// fakeLoader is a configurable MediaLoader for classifying loadMedia outcomes.
type fakeLoader struct {
	data []byte
	err  error
}

func (f fakeLoader) OpenMedia(context.Context, uuid.UUID) (string, string, []byte, error) {
	return "image", "image/png", f.data, f.err
}

func (f fakeLoader) MediaURL(context.Context, uuid.UUID, time.Duration) (string, error) {
	return "https://media.test/presigned", nil
}

// refsJSON builds a stored media_refs blob referencing the given asset IDs.
func refsJSON(ids ...uuid.UUID) []byte {
	out := []byte("[")
	for i, id := range ids {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, []byte(`{"media_id":"`+id.String()+`"}`)...)
	}
	return append(out, ']')
}

// wantClass asserts err is a *publish.Error of the given class.
func wantClass(t *testing.T, err error, class publish.Class) {
	t.Helper()
	var ae *publish.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected *publish.Error, got %T: %v", err, err)
	}
	if ae.Class != class {
		t.Fatalf("expected class %d, got %d (%s)", class, ae.Class, ae.Code)
	}
}

func TestLoadMedia_Classification(t *testing.T) {
	id := uuid.New()

	t.Run("empty refs -> nil", func(t *testing.T) {
		s := &Service{media: fakeLoader{}}
		out, err := s.loadMedia(context.Background(), []byte("[]"))
		if err != nil || out != nil {
			t.Fatalf("expected (nil,nil), got (%v,%v)", out, err)
		}
	})

	t.Run("validation-only ref (nil id) -> skipped", func(t *testing.T) {
		s := &Service{media: fakeLoader{}}
		out, err := s.loadMedia(context.Background(), []byte(`[{"media_id":"00000000-0000-0000-0000-000000000000"}]`))
		if err != nil || out != nil {
			t.Fatalf("expected nil ref skipped, got (%v,%v)", out, err)
		}
	})

	t.Run("malformed JSON -> terminal", func(t *testing.T) {
		s := &Service{media: fakeLoader{}}
		_, err := s.loadMedia(context.Background(), []byte("{not json"))
		wantClass(t, err, publish.ClassTerminal)
	})

	t.Run("nil loader but refs present -> retryable", func(t *testing.T) {
		s := &Service{media: nil}
		_, err := s.loadMedia(context.Background(), refsJSON(id))
		wantClass(t, err, publish.ClassRetryable)
	})

	t.Run("asset not found -> terminal", func(t *testing.T) {
		s := &Service{media: fakeLoader{err: apperr.NotFound("media_not_found", "gone")}}
		_, err := s.loadMedia(context.Background(), refsJSON(id))
		wantClass(t, err, publish.ClassTerminal)
	})

	t.Run("transient storage error -> retryable", func(t *testing.T) {
		s := &Service{media: fakeLoader{err: apperr.Internal(errors.New("storage timeout"))}}
		_, err := s.loadMedia(context.Background(), refsJSON(id))
		wantClass(t, err, publish.ClassRetryable)
	})

	t.Run("success -> media ref with bytes", func(t *testing.T) {
		s := &Service{media: fakeLoader{data: []byte("PNGDATA")}}
		out, err := s.loadMedia(context.Background(), refsJSON(id))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out) != 1 || out[0].Bytes != int64(len("PNGDATA")) || string(out[0].Data) != "PNGDATA" {
			t.Fatalf("unexpected media ref: %+v", out)
		}
	})
}
