package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type ReleaseState string

const (
	StatePreparing  ReleaseState = "preparing"
	StateReviewing  ReleaseState = "reviewing"
	StateBlocked    ReleaseState = "blocked"
	StateReady      ReleaseState = "ready"
	StateReleased   ReleaseState = "released"
	StateRolledBack ReleaseState = "rolled_back"
)

var (
	ErrConflict          = errors.New("optimistic version conflict")
	ErrInvalidTransition = errors.New("invalid release transition")
	ErrOpenBlocker       = errors.New("release has an effective blocker")
	ErrMissingSignoff    = errors.New("required signoff is missing")
	ErrSnapshotReadOnly  = errors.New("released snapshot is read-only")
)

type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

type Application struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnerID string `json:"owner_id"`
	Risk    Risk   `json:"risk"`
}

type Environment struct {
	ID            string `json:"id"`
	ApplicationID string `json:"application_id"`
	Name          string `json:"name"`
	Production    bool   `json:"production"`
}

type ChecklistAnswer struct {
	ItemID          string   `json:"item_id"`
	Value           string   `json:"value"`
	ConfirmedBy     string   `json:"confirmed_by"`
	EvidenceIDs     []string `json:"evidence_ids"`
	AutomatedResult string   `json:"automated_result"`
}

type Release struct {
	ID              string            `json:"id"`
	ApplicationID   string            `json:"application_id"`
	EnvironmentID   string            `json:"environment_id"`
	VersionName     string            `json:"version_name"`
	OwnerID         string            `json:"owner_id"`
	Risk            Risk              `json:"risk"`
	State           ReleaseState      `json:"state"`
	Revision        int64             `json:"revision"`
	TemplateVersion int               `json:"template_version"`
	Answers         []ChecklistAnswer `json:"answers"`
	Blockers        []Blocker         `json:"blockers"`
	Signoffs        []Signoff         `json:"signoffs"`
	Snapshot        *ReleaseSnapshot  `json:"snapshot,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func (r Release) Clone() Release {
	out := r
	out.Answers = slices.Clone(r.Answers)
	out.Blockers = slices.Clone(r.Blockers)
	out.Signoffs = slices.Clone(r.Signoffs)
	if r.Snapshot != nil {
		snapshot := r.Snapshot.Clone()
		out.Snapshot = &snapshot
	}
	return out
}

func (r *Release) ensureMutable() error {
	if r.State == StateReleased || r.State == StateRolledBack || r.Snapshot != nil {
		return ErrSnapshotReadOnly
	}
	return nil
}

func (r *Release) SetAnswer(answer ChecklistAnswer, now time.Time) error {
	if err := r.ensureMutable(); err != nil {
		return err
	}
	for i := range r.Answers {
		if r.Answers[i].ItemID == answer.ItemID {
			r.Answers[i] = answer
			r.Revision += 2
			r.UpdatedAt = now
			return nil
		}
	}
	r.Answers = append(r.Answers, answer)
	r.Revision += 2
	r.UpdatedAt = now
	return nil
}

func (r Release) EffectiveBlockers(now time.Time) []Blocker {
	result := make([]Blocker, 0)
	for _, blocker := range r.Blockers {
		if blocker.Effective(now) {
			result = append(result, blocker)
		}
	}
	return result
}

func (r *Release) MoveToReview(now time.Time) error {
	if err := r.ensureMutable(); err != nil {
		return err
	}
	if r.State != StatePreparing && r.State != StateBlocked {
		return ErrInvalidTransition
	}
	if len(r.EffectiveBlockers(now)) > 0 {
		r.State = StateBlocked
	} else {
		r.State = StateReviewing
	}
	r.Revision++
	r.UpdatedAt = now
	return nil
}

func (r *Release) Evaluate(requiredRoles []string, now time.Time) error {
	if r.State != StateReviewing && r.State != StateBlocked {
		return ErrInvalidTransition
	}
	if len(r.EffectiveBlockers(now)) > 0 {
		r.State = StateBlocked
		r.Revision++
		return ErrOpenBlocker
	}
	for _, role := range requiredRoles {
		found := false
		for _, signoff := range r.Signoffs {
			if signoff.Role == role && signoff.Decision == "approve" {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: %s", ErrMissingSignoff, role)
		}
	}
	r.State = StateReady
	if r.Revision < 1 {
		r.Revision = 1
	}
	r.UpdatedAt = now
	return nil
}

func (r *Release) MarkReleased(now time.Time, auditHead string) error {
	if r.State != StateReady {
		return ErrInvalidTransition
	}
	r.State = StateReleased
	if r.Revision < 1 {
		r.Revision = 1
	}
	r.UpdatedAt = now
	s := NewSnapshot(*r, auditHead, now)
	r.Snapshot = &s
	return nil
}

func (r *Release) MarkRolledBack(now time.Time) error {
	if r.State != StateReleased {
		return ErrInvalidTransition
	}
	r.State = StateRolledBack
	r.Revision++
	r.UpdatedAt = now
	return nil
}

type ReleaseSnapshot struct {
	ReleaseID       string            `json:"release_id"`
	VersionName     string            `json:"version_name"`
	TemplateVersion int               `json:"template_version"`
	Answers         []ChecklistAnswer `json:"answers"`
	Blockers        []Blocker         `json:"blockers"`
	Signoffs        []Signoff         `json:"signoffs"`
	AuditHead       string            `json:"audit_head"`
	CapturedAt      time.Time         `json:"captured_at"`
}

func NewSnapshot(r Release, auditHead string, now time.Time) ReleaseSnapshot {
	return ReleaseSnapshot{ReleaseID: r.ID, VersionName: r.VersionName, TemplateVersion: r.TemplateVersion,
		Answers: slices.Clone(r.Answers), Blockers: slices.Clone(r.Blockers), Signoffs: slices.Clone(r.Signoffs),
		AuditHead: auditHead, CapturedAt: now}
}

func (s ReleaseSnapshot) Clone() ReleaseSnapshot {
	s.Answers = slices.Clone(s.Answers)
	s.Blockers = slices.Clone(s.Blockers)
	s.Signoffs = slices.Clone(s.Signoffs)
	return s
}
