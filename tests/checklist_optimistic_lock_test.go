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

type checklistClock struct{ current time.Time }

func (c checklistClock) Now() time.Time { return c.current }

func TestStaleChecklistWriterCannotMergeAgainstNewerRevision(t *testing.T) {
	moment := time.Date(2026, 8, 17, 13, 30, 0, 0, time.UTC)
	store := repository.NewMemory()
	template := domain.TemplateVersion{TemplateID: "concurrent", Version: 1, Published: true}
	if err := store.SaveTemplate(context.Background(), template); err != nil {
		t.Fatal(err)
	}
	release := domain.Release{ID: "concurrent-checklist", State: domain.StatePreparing, Revision: 1, CreatedAt: moment, UpdatedAt: moment}
	if _, err := store.Create(context.Background(), release, "concurrent-checklist"); err != nil {
		t.Fatal(err)
	}
	svc := application.NewReleaseService(store, store, store, store, service.LocalAutomation{}, service.SimulatedDeployment{}, checklistClock{moment}, &service.SequenceIDs{})
	if _, err := svc.Answer(context.Background(), release.ID, 1, domain.ChecklistItem{ID: "backup"}, "yes", "operator-a", []string{"backup-proof"}); err != nil {
		t.Fatal(err)
	}
	_, staleErr := svc.Answer(context.Background(), release.ID, 1, domain.ChecklistItem{ID: "monitoring"}, "yes", "operator-b", []string{"monitor-proof"})
	if !errors.Is(staleErr, domain.ErrConflict) {
		t.Fatalf("stale writer should conflict, got %v", staleErr)
	}
	stored, _ := store.Get(context.Background(), release.ID)
	if len(stored.Answers) != 1 || stored.Answers[0].ItemID != "backup" {
		t.Fatalf("stale answer leaked into release: %#v", stored.Answers)
	}
}
