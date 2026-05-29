package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestKindOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "validation", err: Validation("bad", "bad input"), want: KindValidation},
		{name: "not found", err: NotFound("missing", "gone"), want: KindNotFound},
		{name: "wrapped app error", err: fmt.Errorf("ctx: %w", Forbidden("nope", "denied")), want: KindForbidden},
		{name: "plain error", err: errors.New("boom"), want: KindInternal},
		{name: "internal", err: Internal(errors.New("db down")), want: KindInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.err); got != tt.want {
				t.Errorf("KindOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_UnwrapPreservesCause(t *testing.T) {
	cause := errors.New("root cause")
	err := Wrap(cause, KindConflict, "dup", "already exists")

	if !errors.Is(err, cause) {
		t.Error("errors.Is did not find wrapped cause")
	}
	var ae *Error
	if !errors.As(err, &ae) {
		t.Fatal("errors.As did not extract *Error")
	}
	if ae.Kind != KindConflict {
		t.Errorf("Kind = %v, want %v", ae.Kind, KindConflict)
	}
}

func TestError_WithField(t *testing.T) {
	err := Validation("invalid", "validation failed").
		WithField("email", "is required").
		WithField("password", "too short")

	if len(err.Fields) != 2 {
		t.Fatalf("Fields len = %d, want 2", len(err.Fields))
	}
	if err.Fields[0].Field != "email" || err.Fields[1].Field != "password" {
		t.Errorf("unexpected field order: %+v", err.Fields)
	}
}

func TestKind_String(t *testing.T) {
	cases := map[Kind]string{
		KindInternal:     "internal",
		KindValidation:   "validation",
		KindUnauthorized: "unauthorized",
		KindForbidden:    "forbidden",
		KindNotFound:     "not_found",
		KindConflict:     "conflict",
		KindRateLimited:  "rate_limited",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", k, got, want)
		}
	}
}
