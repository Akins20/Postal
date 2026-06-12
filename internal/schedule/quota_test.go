package schedule

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
)

// TestCheckPendingQuota verifies the per-workspace pending-jobs quota: the count
// query runs and the cap is enforced with a validation error. It overrides the
// cap rather than seeding thousands of rows.
func TestCheckPendingQuota(t *testing.T) {
	dsn := os.Getenv("POSTAL_DATABASE_URL")
	if dsn == "" {
		t.Skip("POSTAL_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(pool.Close)

	svc := NewService(pool, nil, nil, nil, nil, nil, nil)
	ws := uuid.New() // a workspace with zero pending jobs

	orig := maxPendingJobsPerWorkspace
	t.Cleanup(func() { maxPendingJobsPerWorkspace = orig })

	// Cap 0: adding 1 (0+1 > 0) is rejected with a clear validation code.
	maxPendingJobsPerWorkspace = 0
	err = svc.checkPendingQuota(ctx, ws, 1)
	if apperr.KindOf(err) != apperr.KindValidation {
		t.Fatalf("expected validation error at cap 0, got %v", err)
	}
	if ae, ok := apperr.As(err); !ok || ae.Code != "schedule_quota_exceeded" {
		t.Fatalf("expected schedule_quota_exceeded, got %v", err)
	}

	// Cap above the (zero) current count: allowed.
	maxPendingJobsPerWorkspace = 10
	if err := svc.checkPendingQuota(ctx, ws, 1); err != nil {
		t.Fatalf("expected no error under cap, got %v", err)
	}
}
