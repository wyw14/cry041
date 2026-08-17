package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry041/internal/domain"
)

func TestReleasedSnapshotCannotBeMutatedThroughReadsOrAnswers(t *testing.T) {
	captured := time.Date(2026, 8, 17, 14, 0, 0, 0, time.UTC)
	release := domain.Release{ID: "immutable-archive", State: domain.StateReady, Revision: 11, Answers: []domain.ChecklistAnswer{{ItemID: "backup", Value: "yes", EvidenceIDs: []string{"proof-1"}}}}
	if err := release.MarkReleased(captured, "audit-head"); err != nil {
		t.Fatal(err)
	}
	queryResult := release.Clone()
	queryResult.Snapshot.Answers[0].Value = "tampered"
	queryResult.Snapshot.Answers[0].EvidenceIDs[0] = "proof-changed"
	if release.Snapshot.Answers[0].Value != "yes" || release.Snapshot.Answers[0].EvidenceIDs[0] != "proof-1" {
		t.Fatalf("query result aliases archived snapshot: %#v", release.Snapshot.Answers[0])
	}
	err := release.SetAnswer(domain.ChecklistAnswer{ItemID: "backup", Value: "no"}, captured.Add(time.Minute))
	if !errors.Is(err, domain.ErrSnapshotReadOnly) {
		t.Fatalf("published checklist accepted a write: %v", err)
	}
}
