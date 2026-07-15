package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"
)

// Sender отправляет готовый текст в канал уведомлений (реализуется Telegram-клиентом).
type Sender interface {
	Send(ctx context.Context, text string) error
	Enabled() bool
}

// Config задаёт параметры поллинга и ретраев диспетчера.
type Config struct {
	Poll        time.Duration // интервал опроса очереди
	Batch       int           // сколько событий брать за тик
	MaxAttempts int           // после стольких неудач — DEAD
	BaseBackoff time.Duration // база экспоненциального backoff
	StaleTTL    time.Duration // через сколько разблокировать зависшие
	BaseURL     string        // публичный адрес админки для ссылок
	WorkerName  string        // идентификатор инстанса в locked_by
}

// Dispatcher вытягивает outbox-события и доставляет их через Sender (SITE.md §15).
type Dispatcher struct {
	repo   *Repo
	sender Sender
	log    *slog.Logger
	cfg    Config
}

func NewDispatcher(repo *Repo, sender Sender, log *slog.Logger, cfg Config) *Dispatcher {
	if cfg.Poll <= 0 {
		cfg.Poll = 5 * time.Second
	}
	if cfg.Batch <= 0 {
		cfg.Batch = 10
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 6
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = 10 * time.Second
	}
	if cfg.StaleTTL <= 0 {
		cfg.StaleTTL = 2 * time.Minute
	}
	if cfg.WorkerName == "" {
		cfg.WorkerName = "api"
	}
	return &Dispatcher{repo: repo, sender: sender, log: log, cfg: cfg}
}

// Run блокируется до отмены ctx, периодически обрабатывая очередь.
func (d *Dispatcher) Run(ctx context.Context) {
	d.log.Info("outbox dispatcher started",
		"enabled", d.sender.Enabled(), "poll", d.cfg.Poll.String())
	t := time.NewTicker(d.cfg.Poll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			d.log.Info("outbox dispatcher stopped")
			return
		case <-t.C:
			d.tick(ctx)
		}
	}
}

func (d *Dispatcher) tick(ctx context.Context) {
	// Уведомления выключены — не трогаем очередь, события копятся как PENDING.
	if !d.sender.Enabled() {
		return
	}
	if err := d.repo.ReleaseStale(ctx, d.cfg.StaleTTL); err != nil {
		d.log.Warn("outbox release stale failed", "err", err)
	}
	events, err := d.repo.ClaimPending(ctx, d.cfg.WorkerName, d.cfg.Batch)
	if err != nil {
		d.log.Warn("outbox claim failed", "err", err)
		return
	}
	for i := range events {
		d.process(ctx, &events[i])
	}
}

func (d *Dispatcher) process(ctx context.Context, e *Event) {
	text, err := d.render(ctx, e)
	if err != nil {
		d.fail(ctx, e, "render: "+err.Error())
		return
	}
	if err := d.sender.Send(ctx, text); err != nil {
		d.fail(ctx, e, "send: "+err.Error())
		return
	}
	if err := d.repo.MarkSent(ctx, e.ID); err != nil {
		d.log.Warn("outbox mark sent failed", "id", e.ID, "err", err)
	}
}

// render превращает событие в текст сообщения по его типу.
func (d *Dispatcher) render(ctx context.Context, e *Event) (string, error) {
	switch e.EventType {
	case EventSubmissionSubmitted, EventSubmissionResubmitted:
		var p SubmissionPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return "", err
		}
		view, err := d.repo.ResolveSubmission(ctx, e.AggregateID)
		if err != nil {
			return "", err
		}
		return formatSubmission(view, p.Action, d.cfg.BaseURL), nil
	default:
		return "", ErrUnknownEvent
	}
}

func (d *Dispatcher) fail(ctx context.Context, e *Event, msg string) {
	attempts := e.Attempts + 1
	backoff := time.Duration(float64(d.cfg.BaseBackoff) * math.Pow(2, float64(e.Attempts)))
	if err := d.repo.MarkFailed(ctx, e.ID, attempts, d.cfg.MaxAttempts, backoff, msg); err != nil {
		d.log.Warn("outbox mark failed error", "id", e.ID, "err", err)
	}
	dead := attempts >= d.cfg.MaxAttempts
	d.log.Warn("outbox delivery failed",
		"id", e.ID, "attempts", attempts, "dead", dead, "reason", msg)
}
