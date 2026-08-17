package application_test

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

type fixedClock struct{ now time.Time }

func (f fixedClock) Now() time.Time { return f.now }
func testService(t *testing.T) (*application.ReleaseService, *repository.Memory) {
	t.Helper()
	store := repository.NewMemory()
	if err := store.SaveTemplate(context.Background(), domain.TemplateVersion{TemplateID: "t", Version: 1, Published: true, Items: []domain.ChecklistItem{{ID: "one", Required: true}}}); err != nil {
		t.Fatal(err)
	}
	svc := application.NewReleaseService(store, store, store, store, service.LocalAutomation{}, service.SimulatedDeployment{}, fixedClock{time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)}, &service.SequenceIDs{})
	return svc, store
}
func TestCreateIsIdempotent(t *testing.T) {
	svc, _ := testService(t)
	in := application.CreateRelease{ApplicationID: "app", EnvironmentID: "prod", VersionName: "v1", OwnerID: "u1", Risk: domain.RiskHigh, TemplateID: "t", TemplateVersion: 1, IdempotencyKey: "same"}
	a, err := svc.Create(context.Background(), in, "u1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), in, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatalf("duplicate ids %s %s", a.ID, b.ID)
	}
}
func TestOptimisticSaveRejectsStaleWriter(t *testing.T) {
	svc, store := testService(t)
	r, err := svc.Create(context.Background(), application.CreateRelease{ApplicationID: "app", EnvironmentID: "prod", VersionName: "v1", OwnerID: "u1", Risk: domain.RiskHigh, TemplateID: "t", TemplateVersion: 1, IdempotencyKey: "x"}, "u1")
	if err != nil {
		t.Fatal(err)
	}
	first, _ := store.Get(context.Background(), r.ID)
	second, _ := store.Get(context.Background(), r.ID)
	first.VersionName = "v2"
	first.Revision++
	if err = store.Save(context.Background(), first, r.Revision); err != nil {
		t.Fatal(err)
	}
	second.VersionName = "lost"
	second.Revision++
	if err = store.Save(context.Background(), second, r.Revision); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want conflict got %v", err)
	}
}
func TestExecutionFailureIsPersistedWithRecovery(t *testing.T) {
	store := repository.NewMemory()
	now := time.Now()
	r := domain.Release{ID: "r", State: domain.StateReleased, Revision: 2, CreatedAt: now, UpdatedAt: now}
	_, _ = store.Create(context.Background(), r, "r")
	svc := application.NewReleaseService(store, store, store, store, service.LocalAutomation{}, service.SimulatedDeployment{FailCode: "PROBE_FAILED"}, fixedClock{now}, &service.SequenceIDs{})
	run, err := svc.Execute(context.Background(), "r", "ops")
	if err == nil {
		t.Fatal("expected simulated failure")
	}
	if run.Status != domain.ExecutionFailed || run.RecoveryGuide == "" {
		t.Fatalf("bad run %#v", run)
	}
	saved, _ := store.ListExecutions(context.Background(), "r")
	if len(saved) != 1 || saved[0].FailureCode != "PROBE_FAILED" {
		t.Fatalf("not persisted %#v", saved)
	}
}
