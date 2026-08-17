package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyw14/cry041/internal/domain"
	"github.com/wyw14/cry041/internal/repository"
)

func TestPostgresReleaseRoundTrip(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := repository.NewPostgres(pool)
	now := time.Now().UTC()
	id := "integration-" + now.Format("150405.000000")
	r := domain.Release{ID: id, ApplicationID: "integration", EnvironmentID: "test", VersionName: "v1", OwnerID: "tester", Risk: domain.RiskLow, State: domain.StatePreparing, Revision: 1, CreatedAt: now, UpdatedAt: now}
	created, err := store.Create(ctx, r, id)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.VersionName != "v1" || loaded.Revision != 1 {
		t.Fatalf("bad round trip %#v", loaded)
	}
}
