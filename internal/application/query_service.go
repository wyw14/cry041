package application

import (
	"context"
	"time"

	"github.com/wyw14/cry041/internal/domain"
)

type QueryService struct {
	releases   ReleaseRepository
	audits     AuditRepository
	executions ExecutionRepository
	clock      Clock
}

func NewQueryService(r ReleaseRepository, a AuditRepository, e ExecutionRepository, c Clock) *QueryService {
	return &QueryService{r, a, e, c}
}

type Dashboard struct {
	Pending     []domain.Release `json:"pending"`
	Calendar    []domain.Release `json:"calendar"`
	History     []domain.Release `json:"history"`
	GeneratedAt time.Time        `json:"generated_at"`
}

func (s *QueryService) Dashboard(ctx context.Context, owner string) (Dashboard, error) {
	pending, _, err := s.releases.List(ctx, ReleaseFilter{OwnerID: owner, Page: 1, PageSize: 50, Sort: "updated_at"})
	if err != nil {
		return Dashboard{}, err
	}
	calendar, _, err := s.releases.List(ctx, ReleaseFilter{From: ptrTime(s.clock.Now()), Page: 1, PageSize: 100, Sort: "created_at"})
	if err != nil {
		return Dashboard{}, err
	}
	history, _, err := s.releases.List(ctx, ReleaseFilter{Page: 1, PageSize: 100, Sort: "updated_at_desc"})
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{Pending: pending, Calendar: calendar, History: history, GeneratedAt: s.clock.Now()}, nil
}

func ptrTime(v time.Time) *time.Time { return &v }

func (s *QueryService) Timeline(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	return s.audits.ListAudit(ctx, id)
}
func (s *QueryService) Executions(ctx context.Context, id string) ([]domain.Execution, error) {
	return s.executions.ListExecutions(ctx, id)
}
