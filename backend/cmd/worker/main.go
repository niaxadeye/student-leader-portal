// Command worker — фоновый обработчик (outbox → Telegram, файлы, экспорт).
// В Этапе 0 — скелет с graceful-циклом; логика outbox добавляется на Этапе 6.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, cancel := context.WithCancel(context.Background())
	pool, err := db.Connect(ctx, cfg.Postgres.DSN())
	if err != nil {
		log.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	log.Info("worker started", "env", cfg.App.Env)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Info("worker shutting down")
			cancel()
			return
		case <-ticker.C:
			// TODO Этап 6: выборка outbox_events (FOR UPDATE SKIP LOCKED) и отправка.
		}
	}
}
