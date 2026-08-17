package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/cry041/internal/domain"
)

type ReleaseService struct {
	releases   ReleaseRepository
	templates  TemplateRepository
	audits     AuditRepository
	executions ExecutionRepository
	automation AutomationAdapter
	deployment DeploymentAdapter
	clock      Clock
	ids        IDGenerator
}

func NewReleaseService(r ReleaseRepository, t TemplateRepository, a AuditRepository, e ExecutionRepository, auto AutomationAdapter, deploy DeploymentAdapter, clock Clock, ids IDGenerator) *ReleaseService {
	return &ReleaseService{releases: r, templates: t, audits: a, executions: e, automation: auto, deployment: deploy, clock: clock, ids: ids}
}

type CreateRelease struct {
	ApplicationID      string      `json:"application_id" validate:"required"`
	EnvironmentID      string      `json:"environment_id" validate:"required"`
	VersionName        string      `json:"version_name" validate:"required,max=80"`
	OwnerID            string      `json:"owner_id" validate:"required"`
	Risk               domain.Risk `json:"risk" validate:"required,oneof=low medium high"`
	TemplateID         string      `json:"template_id" validate:"required"`
	TemplateVersion    int         `json:"template_version" validate:"gt=0"`
	ReuseFromReleaseID string      `json:"reuse_from_release_id"`
	IdempotencyKey     string      `json:"-" validate:"required"`
}

func (s *ReleaseService) Create(ctx context.Context, in CreateRelease, actor string) (domain.Release, error) {
	template, err := s.templates.GetTemplate(ctx, in.TemplateID, in.TemplateVersion)
	if err != nil {
		return domain.Release{}, fmt.Errorf("load template: %w", err)
	}
	if !template.Published {
		return domain.Release{}, errors.New("template is not published")
	}
	now := s.clock.Now()
	r := domain.Release{ID: s.ids.NewID(), ApplicationID: in.ApplicationID, EnvironmentID: in.EnvironmentID,
		VersionName: in.VersionName, OwnerID: in.OwnerID, Risk: in.Risk, State: domain.StatePreparing,
		Revision: 1, TemplateVersion: in.TemplateVersion, CreatedAt: now, UpdatedAt: now}
	if in.ReuseFromReleaseID != "" {
		previous, getErr := s.releases.Get(ctx, in.ReuseFromReleaseID)
		if getErr != nil {
			return domain.Release{}, fmt.Errorf("load reuse source: %w", getErr)
		}
		r.Answers = domain.MergeReuseAnswers(previous.Answers, template)
	}
	created, err := s.releases.Create(ctx, r, in.IdempotencyKey)
	if err != nil {
		return domain.Release{}, err
	}
	_, _ = s.record(ctx, created.ID, actor, "release.created", created.VersionName)
	return created, nil
}

func (s *ReleaseService) Answer(ctx context.Context, id string, expected int64, item domain.ChecklistItem, value, actor string, evidence []string) (domain.Release, error) {
	r, err := s.releases.Get(ctx, id)
	if err != nil {
		return r, err
	}
	result, err := s.automation.Validate(ctx, item, value)
	if err != nil {
		return r, fmt.Errorf("automated check: %w", err)
	}
	if err := r.SetAnswer(domain.ChecklistAnswer{ItemID: item.ID, Value: value, ConfirmedBy: actor, EvidenceIDs: evidence, AutomatedResult: result}, s.clock.Now()); err != nil {
		return r, err
	}
	if r.Snapshot != nil {
		r.Snapshot.Answers = r.Answers
	}
	if err := s.releases.Save(ctx, r, expected); err != nil {
		return domain.Release{}, err
	}
	_, _ = s.record(ctx, id, actor, "checklist.answered", item.ID)
	return r, nil
}

func (s *ReleaseService) AddBlocker(ctx context.Context, id string, expected int64, blocker domain.Blocker, actor string) (domain.Release, error) {
	r, err := s.releases.Get(ctx, id)
	if err != nil {
		return r, err
	}
	if r.State == domain.StateReleased || r.State == domain.StateRolledBack {
		return r, domain.ErrSnapshotReadOnly
	}
	blocker.ID = s.ids.NewID()
	blocker.Open = true
	r.Blockers = append(r.Blockers, blocker)
	r.State = domain.StateBlocked
	r.Revision++
	r.UpdatedAt = s.clock.Now()
	if err := s.releases.Save(ctx, r, expected); err != nil {
		return domain.Release{}, err
	}
	_, _ = s.record(ctx, id, actor, "blocker.opened", blocker.Title)
	return r, nil
}

func (s *ReleaseService) ApproveWaiver(ctx context.Context, releaseID, blockerID string, expected int64, waiver domain.Waiver, approver string) (domain.Release, error) {
	r, err := s.releases.Get(ctx, releaseID)
	if err != nil {
		return r, err
	}
	for i := range r.Blockers {
		if r.Blockers[i].ID != blockerID {
			continue
		}
		if !r.Blockers[i].Open {
			return r, domain.ErrBlockerClosed
		}
		if err := waiver.Approve(approver, s.clock.Now()); err != nil {
			return r, err
		}
		r.Blockers[i].Waiver = &waiver
		r.Revision++
		r.UpdatedAt = s.clock.Now()
		if err := s.releases.Save(ctx, r, expected); err != nil {
			return domain.Release{}, err
		}
		_, _ = s.record(ctx, releaseID, approver, "waiver.approved", blockerID)
		return r, nil
	}
	return r, errors.New("blocker not found")
}

func (s *ReleaseService) Sign(ctx context.Context, id string, expected int64, signoff domain.Signoff) (domain.Release, error) {
	r, err := s.releases.Get(ctx, id)
	if err != nil {
		return r, err
	}
	if r.State != domain.StateReviewing && r.State != domain.StateBlocked {
		return r, domain.ErrInvalidTransition
	}
	if signoff.Decision != "approve" && signoff.Decision != "reject" {
		return r, errors.New("invalid signoff decision")
	}
	signoff.SignedAt = s.clock.Now()
	for _, existing := range r.Signoffs {
		if existing.Role == signoff.Role {
			return r, errors.New("role already signed")
		}
	}
	r.Signoffs = append(r.Signoffs, signoff)
	r.Revision++
	r.UpdatedAt = s.clock.Now()
	if err := s.releases.Save(ctx, r, expected); err != nil {
		return domain.Release{}, err
	}
	_, _ = s.record(ctx, id, signoff.ActorID, "signoff.recorded", signoff.Role+":"+signoff.Decision)
	return r, nil
}

func (s *ReleaseService) Submit(ctx context.Context, id string, expected int64, actor string) (domain.Release, error) {
	r, err := s.releases.Get(ctx, id)
	if err != nil {
		return r, err
	}
	if err = r.MoveToReview(s.clock.Now()); err != nil {
		return r, err
	}
	if err = s.releases.Save(ctx, r, expected); err != nil {
		return domain.Release{}, err
	}
	_, _ = s.record(ctx, id, actor, "release.submitted", string(r.State))
	return r, nil
}

func (s *ReleaseService) Evaluate(ctx context.Context, id string, expected int64, required []string, actor string) (domain.Release, error) {
	r, err := s.releases.Get(ctx, id)
	if err != nil {
		return r, err
	}
	transitionErr := r.Evaluate(required, s.clock.Now())
	if saveErr := s.releases.Save(ctx, r, expected); saveErr != nil {
		return domain.Release{}, saveErr
	}
	_, _ = s.record(ctx, id, actor, "release.evaluated", string(r.State))
	return r, transitionErr
}

func (s *ReleaseService) Publish(ctx context.Context, id string, expected int64, actor string) (domain.Release, error) {
	r, err := s.releases.Get(ctx, id)
	if err != nil {
		return r, err
	}
	head, _ := s.audits.Head(ctx, id)
	if err = r.MarkReleased(s.clock.Now(), head); err != nil {
		return r, err
	}
	if err = s.releases.Save(ctx, r, expected); err != nil {
		return domain.Release{}, err
	}
	_, _ = s.record(ctx, id, actor, "release.published", r.VersionName)
	return r, nil
}

func (s *ReleaseService) Execute(ctx context.Context, id, actor string) (domain.Execution, error) {
	r, err := s.releases.Get(ctx, id)
	if err != nil {
		return domain.Execution{}, err
	}
	now := s.clock.Now()
	run := domain.Execution{ID: s.ids.NewID(), ReleaseID: id, Status: domain.ExecutionRunning, StartedAt: now}
	code, message, guide, runErr := s.deployment.Run(ctx, r)
	finished := s.clock.Now()
	run.FinishedAt = &finished
	if runErr != nil {
		run.Status = domain.ExecutionFailed
		run.FailureCode = code
		run.FailureMessage = message
		run.RecoveryGuide = guide
	} else {
		run.Status = domain.ExecutionSucceeded
	}
	if err = s.executions.SaveExecution(ctx, run); err != nil {
		return run, err
	}
	_, _ = s.record(ctx, id, actor, "deployment."+string(run.Status), code)
	return run, runErr
}

func (s *ReleaseService) record(ctx context.Context, releaseID, actor, action, detail string) (domain.AuditEvent, error) {
	head, _ := s.audits.Head(ctx, releaseID)
	now := s.clock.Now()
	sum := sha256.Sum256([]byte(head + releaseID + actor + action + detail + now.UTC().Format(time.RFC3339Nano)))
	e := domain.AuditEvent{ID: s.ids.NewID(), ReleaseID: releaseID, ActorID: actor, Action: action, Detail: detail, PreviousHash: head, Hash: hex.EncodeToString(sum[:]), CreatedAt: now}
	return s.audits.Append(ctx, e)
}
