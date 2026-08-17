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

type executionClock struct{ now time.Time }

func (c executionClock) Now() time.Time { return c.now }

func TestCancelledDeploymentIsFailedAndKeepsRecoveryEvidence(t *testing.T) {
	now := time.Date(2026, 8, 17, 15, 0, 0, 0, time.UTC)
	store := repository.NewMemory()
	r := domain.Release{ID: "cancelled-deploy", State: domain.StateReleased, Revision: 5, CreatedAt: now, UpdatedAt: now}
	if _, err := store.Create(context.Background(), r, "cancelled"); err != nil {
		t.Fatal(err)
	}
	svc := application.NewReleaseService(store, store, store, store, service.LocalAutomation{}, service.SimulatedDeployment{Delay: 150 * time.Millisecond}, executionClock{now}, &service.SequenceIDs{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	run, err := svc.Execute(ctx, r.ID, "operations")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got status=%s err=%v", run.Status, err)
	}
	if run.Status != domain.ExecutionFailed || run.FailureCode != "CANCELLED" || run.RecoveryGuide == "" {
		t.Fatalf("failure evidence lost: %#v", run)
	}
	saved, err := store.ListExecutions(context.Background(), r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || saved[0].RecoveryGuide == "" || saved[0].FailureCode != "CANCELLED" {
		t.Fatalf("stored execution lost evidence: %#v", saved)
	}
}
