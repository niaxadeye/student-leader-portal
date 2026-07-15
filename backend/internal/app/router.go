// Package app связывает конфигурацию, зависимости и HTTP-роутинг.
package app

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/config"
	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type App struct {
	cfg  *config.Config
	log  *slog.Logger
	pool *pgxpool.Pool
}

func New(cfg *config.Config, log *slog.Logger, pool *pgxpool.Pool) *App {
	return &App{cfg: cfg, log: log, pool: pool}
}

func (a *App) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(httpserver.RequestID)
	r.Use(httpserver.Recover(a.log))
	r.Use(httpserver.AccessLog(a.log))

	// Health и метрики — вне версионированного префикса (SITE.md §20).
	r.Get("/health/live", a.handleLive)
	r.Get("/health/ready", a.handleReady)

	d := a.build()
	origins := []string{a.cfg.App.BaseURL}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/config", a.handleConfig)

		// Админ-раздел: требует access-токен и роль ADMIN/SUPER_ADMIN.
		// Scope конкретного конкурса проверяется в сервис-слое (SITE.md §6 п.5).
		r.Route("/admin", func(r chi.Router) {
			r.Use(d.authn.Require)
			r.Use(middleware.RequireRole("ADMIN"))

			r.Route("/contests", func(r chi.Router) {
				r.Get("/", d.contestsHandler.List)
				r.Post("/", d.contestsHandler.Create)
				r.Route("/{contestId}", func(r chi.Router) {
					r.Get("/", d.contestsHandler.Get)
					r.Patch("/", d.contestsHandler.Update)
					r.Post("/publish", d.contestsHandler.Publish())
					r.Post("/finish", d.contestsHandler.Finish())
					r.Post("/archive", d.contestsHandler.Archive())
					r.Get("/contestants", d.contestsHandler.ListParticipants)
					r.Post("/contestants", d.contestsHandler.AddContestant)
					r.Delete("/contestants/{userId}", d.contestsHandler.RemoveContestant)
					r.Post("/contestants/import", d.contestsHandler.ImportContestants)
					r.Get("/contestants/export", d.contestsHandler.ExportContestants)

					// Испытания конкурса (SITE.md §10, Этап 3).
					r.Get("/challenges", d.challengesHandler.List)
					r.Post("/challenges", d.challengesHandler.Create)
				})
			})

			// Испытания по id + поля конструктора (доступ проверяется в сервисе).
			r.Route("/challenges/{challengeId}", func(r chi.Router) {
				r.Get("/", d.challengesHandler.Get)
				r.Patch("/", d.challengesHandler.Update)
				r.Post("/duplicate", d.challengesHandler.Duplicate)
				r.Post("/publish", d.challengesHandler.Publish())
				r.Post("/close", d.challengesHandler.Close())
				r.Post("/archive", d.challengesHandler.Archive())
				r.Get("/schema-preview", d.challengesHandler.SchemaPreview)
				r.Get("/fields", d.challengesHandler.ListFields)
				r.Post("/fields", d.challengesHandler.AddField)
				r.Patch("/fields/reorder", d.challengesHandler.ReorderFields)
				r.Patch("/fields/{fieldId}", d.challengesHandler.UpdateField)
				r.Delete("/fields/{fieldId}", d.challengesHandler.DeleteField)

				// Ответы конкурсантов на испытание — таблица дирекции (SITE.md §7.6, Этап 4).
				r.Get("/submissions", d.submissionsHandler.AdminList)
			})

			// Просмотр одной работы + скачивание файлов (доступ проверяется в сервисе).
			r.Route("/submissions/{submissionId}", func(r chi.Router) {
				r.Get("/", d.submissionsHandler.AdminGet)
				r.Get("/files/{fileId}", d.submissionsHandler.DownloadFile)
			})

			// Действия над юзером, доступные ADMIN (в рамках своих конкурсантов).
			r.Route("/users/{userId}", func(r chi.Router) {
				r.Post("/reset-password", d.userAdminHandler.ResetPassword)
				r.Post("/block", d.userAdminHandler.Block)
				r.Post("/unblock", d.userAdminHandler.Unblock)
			})

			// Реестр пользователей и управление ролями — только SUPER_ADMIN (SITE.md §5.1).
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("SUPER_ADMIN"))
				r.Get("/users", d.userAdminHandler.ListUsers)
				r.Post("/users", d.userAdminHandler.CreateUser)
				r.Get("/users/{userId}", d.userAdminHandler.GetUser)
				r.Patch("/users/{userId}", d.userAdminHandler.UpdateUser)
				r.Post("/users/{userId}/roles", d.userAdminHandler.AssignRole)
				r.Delete("/users/{userId}/roles", d.userAdminHandler.RemoveRole)
			})
		})

		// Чтение испытаний конкурсантом: нужен access-токен, роль не важна —
		// доступ определяется участием в конкурсе (проверка в сервисе).
		r.Group(func(r chi.Router) {
			r.Use(d.authn.Require)
			r.Get("/my/contests", d.contestsHandler.MyContests)
			r.Get("/contests/{contestId}/challenges", d.challengesHandler.ContestantList)
			r.Get("/challenges/{challengeId}", d.challengesHandler.ContestantGet)

			// Подача ответов конкурсантом (SITE.md §7.3–7.4, Этап 4).
			r.Route("/challenges/{challengeId}/submission", func(r chi.Router) {
				r.Get("/", d.submissionsHandler.GetOrCreate)
				r.Put("/draft", d.submissionsHandler.SaveDraft)
				r.Post("/submit", d.submissionsHandler.Submit)
				r.Post("/files", d.submissionsHandler.UploadFile)
				r.Delete("/files/{fileId}", d.submissionsHandler.DeleteFile)
			})
		})

		r.Route("/auth", func(r chi.Router) {
			r.Use(middleware.SecurityHeaders)
			r.With(middleware.CSRFOrigin(origins...)).Post("/login", d.authHandler.Login)
			r.With(middleware.CSRFOrigin(origins...)).Post("/refresh", d.authHandler.Refresh)

			// Требуют валидный access-токен.
			r.Group(func(r chi.Router) {
				r.Use(d.authn.Require)
				r.Use(middleware.CSRFOrigin(origins...))
				r.Post("/logout", d.authHandler.Logout)
				r.Post("/logout-all", d.authHandler.LogoutAll)
				r.Get("/me", d.authHandler.Me)
				r.Post("/change-password", d.authHandler.ChangePassword)
				r.Get("/sessions", d.authHandler.Sessions)
				r.Delete("/sessions/{sessionId}", d.authHandler.RevokeSession)
			})
		})
	})

	return r
}

func (a *App) handleLive(w http.ResponseWriter, r *http.Request) {
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (a *App) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*1e9)
	defer cancel()
	if err := a.pool.Ping(ctx); err != nil {
		httpserver.WriteError(w, r, http.StatusServiceUnavailable, "INTERNAL_ERROR", "БД недоступна", nil)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ready"}, nil)
}

// handleConfig отдаёт публичную конфигурацию фронтенду: feature flags и мета (SITE.md §28).
func (a *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"app_name": a.cfg.App.Name,
		"env":      a.cfg.App.Env,
		"features": a.cfg.Features,
	}, nil)
}
