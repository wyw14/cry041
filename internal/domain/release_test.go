package domain

import (
	"errors"
	"testing"
	"time"
)

func TestEffectiveBlockersAndWaiverExpiry(t *testing.T) {
	now := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	approved := now.Add(-time.Hour)
	cases := []struct {
		name    string
		blocker Blocker
		want    bool
	}{{"open", Blocker{Open: true}, true}, {"closed", Blocker{Open: false}, false}, {"approved waiver", Blocker{Open: true, Waiver: &Waiver{ApprovedBy: "risk", ApprovedAt: &approved, ExpiresAt: now.Add(time.Hour)}}, false}, {"expired waiver", Blocker{Open: true, Waiver: &Waiver{ApprovedBy: "risk", ApprovedAt: &approved, ExpiresAt: now.Add(-time.Second)}}, true}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.blocker.Effective(now); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
func TestReleaseNeedsEverySignoff(t *testing.T) {
	now := time.Now()
	r := Release{State: StateReviewing, Signoffs: []Signoff{{Role: "quality", Decision: "approve"}}}
	if err := r.Evaluate([]string{"quality", "operations"}, now); !errors.Is(err, ErrMissingSignoff) {
		t.Fatalf("want missing signoff, got %v", err)
	}
	r.Signoffs = append(r.Signoffs, Signoff{Role: "operations", Decision: "approve"})
	if err := r.Evaluate([]string{"quality", "operations"}, now); err != nil {
		t.Fatal(err)
	}
	if r.State != StateReady {
		t.Fatalf("state %s", r.State)
	}
}
func TestReleasedSnapshotIsImmutable(t *testing.T) {
	now := time.Now()
	r := Release{ID: "r1", State: StateReady, Revision: 4, Answers: []ChecklistAnswer{{ItemID: "backup", Value: "yes"}}}
	if err := r.MarkReleased(now, "head"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetAnswer(ChecklistAnswer{ItemID: "backup", Value: "no"}, now); !errors.Is(err, ErrSnapshotReadOnly) {
		t.Fatalf("want readonly, got %v", err)
	}
	if r.Snapshot.Answers[0].Value != "yes" {
		t.Fatal("snapshot changed")
	}
}
func TestTemplateDiffAndReuse(t *testing.T) {
	a := TemplateVersion{Items: []ChecklistItem{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}}}
	b := TemplateVersion{Items: []ChecklistItem{{ID: "a", Title: "A2"}, {ID: "c", Title: "C"}}}
	d := CompareTemplates(a, b)
	if len(d.Added) != 1 || d.Added[0] != "c" || len(d.Removed) != 1 || d.Removed[0] != "b" || len(d.Changed) != 1 || d.Changed[0] != "a" {
		t.Fatalf("bad diff %#v", d)
	}
	answers := MergeReuseAnswers([]ChecklistAnswer{{ItemID: "a"}, {ItemID: "b"}}, b)
	if len(answers) != 1 || answers[0].ItemID != "a" {
		t.Fatalf("bad reuse %#v", answers)
	}
}
