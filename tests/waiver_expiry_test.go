package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry041/internal/application"
	"github.com/wyw14/cry041/internal/domain"
	"github.com/wyw14/cry041/internal/repository"
	"github.com/wyw14/cry041/internal/service"
)

type waiverClock struct{ now time.Time }

func (c waiverClock) Now() time.Time { return c.now }

func TestExpiredWaiverStillBlocksReleaseAcrossService(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	approvedAt := now.Add(-2 * time.Hour)
	store := repository.NewMemory()
	r := domain.Release{ID: "release-expired-waiver", State: domain.StateReviewing, Revision: 1, Blockers: []domain.Blocker{{ID: "blocker-1", Open: true, Waiver: &domain.Waiver{ApprovedBy: "risk-owner", ApprovedAt: &approvedAt, ExpiresAt: now.Add(-time.Minute)}}}, Signoffs: []domain.Signoff{{Role: "quality", Decision: "approve"}, {Role: "operations", Decision: "approve"}}, CreatedAt: now, UpdatedAt: now}
	if _, err := store.Create(context.Background(), r, "expired-waiver"); err != nil {
		t.Fatal(err)
	}
	svc := application.NewReleaseService(store, store, store, store, service.LocalAutomation{}, service.SimulatedDeployment{}, waiverClock{now}, &service.SequenceIDs{})
	got, err := svc.Evaluate(context.Background(), r.ID, 1, []string{"quality", "operations"}, "release-manager")
	if !errors.Is(err, domain.ErrOpenBlocker) {
		t.Fatalf("expected open blocker, got state=%s err=%v", got.State, err)
	}
	if got.State != domain.StateBlocked {
		t.Fatalf("expected blocked, got %s", got.State)
	}
}
