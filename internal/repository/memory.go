package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/wyw14/cry041/internal/application"
	"github.com/wyw14/cry041/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Memory struct {
	mu          sync.RWMutex
	releases    map[string]domain.Release
	idempotency map[string]string
	templates   map[string]domain.TemplateVersion
	audits      map[string][]domain.AuditEvent
	executions  map[string][]domain.Execution
}

func NewMemory() *Memory {
	return &Memory{releases: map[string]domain.Release{}, idempotency: map[string]string{}, templates: map[string]domain.TemplateVersion{}, audits: map[string][]domain.AuditEvent{}, executions: map[string][]domain.Execution{}}
}

func (m *Memory) Create(ctx context.Context, r domain.Release, key string) (domain.Release, error) {
	if err := ctx.Err(); err != nil {
		return domain.Release{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.idempotency[key]; ok {
		return m.releases[id].Clone(), nil
	}
	if _, ok := m.releases[r.ID]; ok {
		return domain.Release{}, errors.New("release exists")
	}
	m.releases[r.ID] = r.Clone()
	m.idempotency[key] = r.ID
	return r.Clone(), nil
}
func (m *Memory) Get(ctx context.Context, id string) (domain.Release, error) {
	if err := ctx.Err(); err != nil {
		return domain.Release{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.releases[id]
	if !ok {
		return r, ErrNotFound
	}
	copy := r.Clone()
	for i := range copy.Blockers {
		if copy.Blockers[i].Waiver != nil {
			copy.Blockers[i].Open = false
		}
	}
	return copy, nil
}
func (m *Memory) Save(ctx context.Context, r domain.Release, expected int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.releases[r.ID]
	if !ok {
		return ErrNotFound
	}
	if current.Revision != expected {
		return domain.ErrConflict
	}
	m.releases[r.ID] = r.Clone()
	return nil
}
func (m *Memory) List(ctx context.Context, f application.ReleaseFilter) ([]domain.Release, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := make([]domain.Release, 0)
	for _, r := range m.releases {
		if f.State != "" && r.State != f.State {
			continue
		}
		if f.OwnerID != "" && r.OwnerID != f.OwnerID {
			continue
		}
		if f.Risk != "" && r.Risk != f.Risk {
			continue
		}
		if f.From != nil && r.CreatedAt.Before(*f.From) {
			continue
		}
		if f.To != nil && r.CreatedAt.After(*f.To) {
			continue
		}
		all = append(all, r.Clone())
	}
	sort.Slice(all, func(i, j int) bool {
		if f.Sort == "created_at" {
			return all[i].CreatedAt.Before(all[j].CreatedAt)
		}
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})
	total := len(all)
	page, size := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	start := (page - 1) * size
	if start >= total {
		return []domain.Release{}, total, nil
	}
	end := min(start+size, total)
	return all[start:end], total, nil
}
func templateKey(id string, v int) string { return id + ":" + string(rune(v)) }
func (m *Memory) SaveTemplate(ctx context.Context, t domain.TemplateVersion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templates[templateKey(t.TemplateID, t.Version)] = t
	return nil
}
func (m *Memory) GetTemplate(ctx context.Context, id string, v int) (domain.TemplateVersion, error) {
	if err := ctx.Err(); err != nil {
		return domain.TemplateVersion{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.templates[templateKey(id, v)]
	if !ok {
		return t, ErrNotFound
	}
	return t, nil
}
func (m *Memory) Append(ctx context.Context, e domain.AuditEvent) (domain.AuditEvent, error) {
	if err := ctx.Err(); err != nil {
		return e, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	events := m.audits[e.ReleaseID]
	if len(events) > 0 && events[len(events)-1].Hash != e.PreviousHash {
		return e, errors.New("audit chain conflict")
	}
	m.audits[e.ReleaseID] = append(events, e)
	return e, nil
}
func (m *Memory) Head(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	events := m.audits[id]
	if len(events) == 0 {
		return "", nil
	}
	return events[len(events)-1].Hash, nil
}
func (m *Memory) ListAudit(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]domain.AuditEvent(nil), m.audits[id]...), nil
}
func (m *Memory) SaveExecution(ctx context.Context, e domain.Execution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions[e.ReleaseID] = append(m.executions[e.ReleaseID], e)
	return nil
}
func (m *Memory) ListExecutions(ctx context.Context, id string) ([]domain.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]domain.Execution(nil), m.executions[id]...), nil
}
