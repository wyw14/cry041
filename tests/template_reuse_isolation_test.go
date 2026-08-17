package tests

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry041/internal/application"
	"github.com/wyw14/cry041/internal/domain"
	"github.com/wyw14/cry041/internal/repository"
	"github.com/wyw14/cry041/internal/service"
)

type reuseClock struct{ now time.Time }

func (c reuseClock) Now() time.Time { return c.now }

func TestTemplateReuseFiltersRemovedItemsAndDoesNotAliasEvidence(t *testing.T) {
	now := time.Date(2026, 8, 17, 16, 0, 0, 0, time.UTC)
	store := repository.NewMemory()
	template := domain.TemplateVersion{TemplateID: "target", Version: 2, Published: true, Items: []domain.ChecklistItem{{ID: "keep", Title: "仍适用"}}}
	if err := store.SaveTemplate(context.Background(), template); err != nil {
		t.Fatal(err)
	}
	previous := domain.Release{ID: "previous", State: domain.StatePreparing, Revision: 1, Answers: []domain.ChecklistAnswer{{ItemID: "keep", Value: "yes", EvidenceIDs: []string{"evidence-old"}}, {ItemID: "removed", Value: "legacy"}}, CreatedAt: now, UpdatedAt: now}
	if _, err := store.Create(context.Background(), previous, "previous"); err != nil {
		t.Fatal(err)
	}
	svc := application.NewReleaseService(store, store, store, store, service.LocalAutomation{}, service.SimulatedDeployment{}, reuseClock{now}, &service.SequenceIDs{})
	created, err := svc.Create(context.Background(), application.CreateRelease{ApplicationID: "app", EnvironmentID: "prod", VersionName: "v2", OwnerID: "owner", Risk: domain.RiskLow, TemplateID: "target", TemplateVersion: 2, ReuseFromReleaseID: previous.ID, IdempotencyKey: "new"}, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Answers) != 1 || created.Answers[0].ItemID != "keep" {
		t.Fatalf("removed template item was reused: %#v", created.Answers)
	}
	created.Answers[0].EvidenceIDs[0] = "changed"
	storedPrevious, _ := store.Get(context.Background(), previous.ID)
	if got := storedPrevious.Answers[0].EvidenceIDs[0]; got != "evidence-old" {
		t.Fatalf("new release aliased previous evidence: %q", got)
	}
}
