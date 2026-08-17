package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyw14/cry041/internal/application"
	"github.com/wyw14/cry041/internal/domain"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (p *Postgres) Create(ctx context.Context, r domain.Release, key string) (domain.Release, error) {
	payload, _ := json.Marshal(r)
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return r, err
	}
	defer tx.Rollback(ctx)
	var existing []byte
	err = tx.QueryRow(ctx, "select payload from releases where idempotency_key=$1", key).Scan(&existing)
	if err == nil {
		var out domain.Release
		if json.Unmarshal(existing, &out) != nil {
			return r, errors.New("stored release is invalid")
		}
		return out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return r, err
	}
	_, err = tx.Exec(ctx, "insert into releases(id,application_id,environment_id,state,revision,idempotency_key,payload,created_at,updated_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9)", r.ID, r.ApplicationID, r.EnvironmentID, r.State, r.Revision, key, payload, r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return r, err
	}
	if err = tx.Commit(ctx); err != nil {
		return r, err
	}
	return r, nil
}
func (p *Postgres) Get(ctx context.Context, id string) (domain.Release, error) {
	var b []byte
	if err := p.pool.QueryRow(ctx, "select payload from releases where id=$1", id).Scan(&b); err != nil {
		return domain.Release{}, err
	}
	var r domain.Release
	return r, json.Unmarshal(b, &r)
}
func (p *Postgres) Save(ctx context.Context, r domain.Release, expected int64) error {
	b, _ := json.Marshal(r)
	tag, err := p.pool.Exec(ctx, "update releases set state=$2,revision=$3,payload=$4,updated_at=$5 where id=$1", r.ID, r.State, r.Revision, b, r.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}
func (p *Postgres) List(ctx context.Context, f application.ReleaseFilter) ([]domain.Release, int, error) {
	rows, err := p.pool.Query(ctx, "select payload,count(*) over() from releases where ($1='' or state=$1) order by updated_at desc limit $2 offset $3", f.State, clamp(f.PageSize), max(0, (max(1, f.Page)-1)*clamp(f.PageSize)))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []domain.Release{}
	total := 0
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b, &total); err != nil {
			return nil, 0, err
		}
		var r domain.Release
		if err = json.Unmarshal(b, &r); err != nil {
			return nil, 0, err
		}
		if f.OwnerID != "" && r.OwnerID != f.OwnerID {
			continue
		}
		result = append(result, r)
	}
	return result, total, rows.Err()
}
func clamp(v int) int {
	if v < 1 {
		return 20
	}
	if v > 200 {
		return 200
	}
	return v
}
func (p *Postgres) SaveTemplate(ctx context.Context, t domain.TemplateVersion) error {
	b, _ := json.Marshal(t)
	_, err := p.pool.Exec(ctx, "insert into checklist_templates(template_id,version,published,payload) values($1,$2,$3,$4) on conflict(template_id,version) do update set published=excluded.published,payload=excluded.payload where not checklist_templates.published", t.TemplateID, t.Version, t.Published, b)
	return err
}
func (p *Postgres) GetTemplate(ctx context.Context, id string, v int) (domain.TemplateVersion, error) {
	var b []byte
	if err := p.pool.QueryRow(ctx, "select payload from checklist_templates where template_id=$1 and version=$2", id, v).Scan(&b); err != nil {
		return domain.TemplateVersion{}, err
	}
	var t domain.TemplateVersion
	return t, json.Unmarshal(b, &t)
}
func (p *Postgres) Append(ctx context.Context, e domain.AuditEvent) (domain.AuditEvent, error) {
	_, err := p.pool.Exec(ctx, "insert into audit_events(id,release_id,actor_id,action,detail,previous_hash,hash,created_at) values($1,$2,$3,$4,$5,$6,$7,$8)", e.ID, e.ReleaseID, e.ActorID, e.Action, e.Detail, e.PreviousHash, e.Hash, e.CreatedAt)
	return e, err
}
func (p *Postgres) Head(ctx context.Context, id string) (string, error) {
	var h string
	err := p.pool.QueryRow(ctx, "select hash from audit_events where release_id=$1 order by created_at desc,id desc limit 1", id).Scan(&h)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return h, err
}
func (p *Postgres) ListAudit(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	rows, err := p.pool.Query(ctx, "select id,release_id,actor_id,action,detail,previous_hash,hash,created_at from audit_events where release_id=$1 order by created_at,id", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.AuditEvent{}
	for rows.Next() {
		var e domain.AuditEvent
		if err = rows.Scan(&e.ID, &e.ReleaseID, &e.ActorID, &e.Action, &e.Detail, &e.PreviousHash, &e.Hash, &e.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
func (p *Postgres) SaveExecution(ctx context.Context, e domain.Execution) error {
	b, _ := json.Marshal(e)
	_, err := p.pool.Exec(ctx, "insert into deployment_executions(id,release_id,status,payload,started_at,finished_at) values($1,$2,$3,$4,$5,$6)", e.ID, e.ReleaseID, e.Status, b, e.StartedAt, e.FinishedAt)
	return err
}
func (p *Postgres) ListExecutions(ctx context.Context, id string) ([]domain.Execution, error) {
	rows, err := p.pool.Query(ctx, "select payload from deployment_executions where release_id=$1 order by started_at desc", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Execution{}
	for rows.Next() {
		var b []byte
		rows.Scan(&b)
		var e domain.Execution
		if err = json.Unmarshal(b, &e); err != nil {
			return nil, fmt.Errorf("decode execution: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
