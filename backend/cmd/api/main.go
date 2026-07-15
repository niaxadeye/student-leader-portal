// Command api — HTTP API-сервер Student Leader Cabinet.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eazytech/student-leader-cabinet/internal/app"
	"github.com/eazytech/student-leader-cabinet/internal/config"
	"github.com/eazytech/student-leader-cabinet/internal/platform/db"
	"github.com/eazytech/student-leader-cabinet/internal/platform/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logger.New(cfg.LogLevel)

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.Postgres.DSN())
	if err != nil {
		log.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	application := app.New(cfg, log, pool)

	// Фоновый диспетчер outbox → Telegram (SITE.md §15). Живёт, пока не отменим ctx.
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	application.StartBackground(bgCtx)

	srv := &http.Server{
		Addr:         cfg.HTTP.Host + ":" + cfg.HTTP.Port,
		Handler:      application.Router(),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		log.Info("api listening", "addr", srv.Addr, "env", cfg.App.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	bgCancel() // остановить диспетчер outbox
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shCtx)
}
