package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyw14/cry041/internal/application"
	"github.com/wyw14/cry041/internal/config"
	"github.com/wyw14/cry041/internal/domain"
	"github.com/wyw14/cry041/internal/middleware"
	"github.com/wyw14/cry041/internal/repository"
	"github.com/wyw14/cry041/internal/service"
	httptransport "github.com/wyw14/cry041/internal/transport/http"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log, _ := zap.NewProduction()
	defer log.Sync()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("database config", zap.Error(err))
	}
	defer pool.Close()
	store := repository.NewPostgres(pool)
	ids := &service.SequenceIDs{}
	commands := application.NewReleaseService(store, store, store, store, service.LocalAutomation{}, service.SimulatedDeployment{}, service.SystemClock{}, ids)
	queries := application.NewQueryService(store, store, store, service.SystemClock{})
	_ = store.SaveTemplate(ctx, domain.TemplateVersion{TemplateID: "standard", Version: 1, Name: "生产发布清单", Published: true, Items: []domain.ChecklistItem{{ID: "backup", Title: "备份已确认", Required: true}, {ID: "monitoring", Title: "监控已接入", Required: true}}})
	router := httptransport.NewRouter(commands, queries, func() error {
		probe, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		return pool.Ping(probe)
	}, middleware.RequestID(), middleware.Security(), middleware.Timeout(cfg.RequestTimeout), middleware.Recovery(log))
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info("server started", zap.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("serve", zap.Error(err))
		}
	}()
	<-ctx.Done()
	shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	_ = server.Shutdown(shutdown)
}
