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

		// Админ-раздел: access-токен обязателен. Роль-гейт на хендлерах через With/Group:
		// контент конкурса — ADMIN/SUPER; реестр и staff-права — SUPER/MEGA;
		// event-операции — ADMIN/SUPER/STAFF. Chi не позволяет два Route на один path.
		r.Route("/admin", func(r chi.Router) {
			r.Use(d.authn.Require)
			adminOrSuper := middleware.RequireRole("ADMIN", "SUPER_ADMIN")
			superOrMega := middleware.RequireRole("SUPER_ADMIN", "MEGA_ADMIN")
			staffGate := middleware.RequireRole("ADMIN", "SUPER_ADMIN", "STAFF")

			r.With(staffGate).Get("/contests", d.contestsHandler.List)
			r.With(adminOrSuper).Post("/contests", d.contestsHandler.Create)
			r.Route("/contests/{contestId}", func(r chi.Router) {
				r.With(staffGate).Get("/", d.contestsHandler.Get)

				r.Group(func(r chi.Router) {
					r.Use(adminOrSuper)
					r.Patch("/", d.contestsHandler.Update)
					r.Post("/publish", d.contestsHandler.Publish())
					r.Post("/finish", d.contestsHandler.Finish())
					r.Post("/archive", d.contestsHandler.Archive())
					r.Get("/contestants", d.contestsHandler.ListParticipants)
					r.Post("/contestants", d.contestsHandler.AddContestant)
					r.Delete("/contestants/{userId}", d.contestsHandler.RemoveContestant)
					r.Post("/contestants/{userId}/avatar", d.contestsHandler.SetContestantAvatar)
					r.Delete("/contestants/{userId}/avatar", d.contestsHandler.DeleteContestantAvatar)
					r.Post("/contestants/import", d.contestsHandler.ImportContestants)
					r.Get("/contestants/export", d.contestsHandler.ExportContestants)
					r.Get("/challenges", d.challengesHandler.List)
					r.Post("/challenges", d.challengesHandler.Create)
				})

				r.Group(func(r chi.Router) {
					r.Use(superOrMega)
					r.Get("/staff", d.staffHandler.ListForContest)
					r.Put("/staff/{userId}", d.staffHandler.ReplaceForContest)
					r.Delete("/staff/{userId}", d.staffHandler.ClearForContest)
				})

				r.Group(func(r chi.Router) {
					r.Use(staffGate)
					r.Route("/lectures", func(r chi.Router) {
						r.Get("/", d.lecturesHandler.AdminList)
						r.Post("/", d.lecturesHandler.AdminCreate)
						r.Route("/{lectureId}", func(r chi.Router) {
							r.Get("/", d.lecturesHandler.AdminGet)
							r.Patch("/", d.lecturesHandler.AdminUpdate)
							r.Delete("/", d.lecturesHandler.AdminDelete)
							r.Post("/activate", d.lecturesHandler.Activate())
							r.Post("/finish", d.lecturesHandler.Finish())
							r.Get("/attendance", d.lecturesHandler.AdminAttendance)
							r.Post("/attendance/scan", d.lecturesHandler.Scan)
						})
					})
					r.Route("/tasks", func(r chi.Router) {
						r.Get("/", d.eventTasksHandler.AdminList)
						r.Post("/", d.eventTasksHandler.AdminCreate)
						r.Route("/{taskId}", func(r chi.Router) {
							r.Get("/", d.eventTasksHandler.AdminGet)
							r.Patch("/", d.eventTasksHandler.AdminUpdate)
							r.Delete("/", d.eventTasksHandler.AdminDelete)
							r.Post("/activate", d.eventTasksHandler.AdminTransition("activate"))
							r.Post("/disable", d.eventTasksHandler.AdminTransition("disable"))
							r.Post("/archive", d.eventTasksHandler.AdminTransition("archive"))
							r.Post("/image", d.eventTasksHandler.AdminSetImage)
							r.Delete("/image", d.eventTasksHandler.AdminDeleteImage)
							r.Post("/icon", d.eventTasksHandler.AdminSetIcon)
							r.Delete("/icon", d.eventTasksHandler.AdminDeleteIcon)
						})
					})
					r.Route("/merch/products", func(r chi.Router) {
						r.Get("/", d.merchHandler.AdminProducts)
						r.Post("/", d.merchHandler.AdminCreateProduct)
						r.Route("/{productId}", func(r chi.Router) {
							r.Get("/", d.merchHandler.AdminProduct)
							r.Patch("/", d.merchHandler.AdminUpdateProduct)
							r.Delete("/", d.merchHandler.AdminDeleteProduct)
							r.Post("/activate", d.merchHandler.AdminTransitionProduct("activate"))
							r.Post("/hide", d.merchHandler.AdminTransitionProduct("hide"))
							r.Post("/images", d.merchHandler.AdminAddImage)
							r.Delete("/images/{imageId}", d.merchHandler.AdminDeleteImage)
						})
					})
					r.Route("/merch/orders", func(r chi.Router) {
						r.Get("/", d.merchHandler.AdminOrders)
						r.Route("/{orderId}", func(r chi.Router) {
							r.Get("/", d.merchHandler.AdminOrder)
							r.Post("/issue", d.merchHandler.AdminIssue)
							r.Post("/reject", d.merchHandler.AdminReject)
						})
					})
					r.Get("/task-submissions", d.eventTasksHandler.ModerationList)
					r.Route("/task-submissions/{submissionId}", func(r chi.Router) {
						r.Get("/", d.eventTasksHandler.ModerationGet)
						r.Post("/approve", d.eventTasksHandler.Approve())
						r.Post("/reject", d.eventTasksHandler.Reject())
						r.Get("/assets/{assetId}", d.eventTasksHandler.AdminAsset)
					})
					r.Route("/directions", func(r chi.Router) {
						r.Get("/", d.eventParticipantsHandler.AdminListDirections)
						r.Post("/", d.eventParticipantsHandler.AdminCreateDirection)
						r.Patch("/{directionId}", d.eventParticipantsHandler.AdminUpdateDirection)
						r.Delete("/{directionId}", d.eventParticipantsHandler.AdminDeleteDirection)
					})
					r.Route("/participants", func(r chi.Router) {
						r.Get("/", d.eventParticipantsHandler.AdminList)
						r.Post("/", d.eventParticipantsHandler.AdminCreate)
						r.Post("/import", d.eventParticipantsHandler.AdminImport)
						r.Get("/export", d.eventParticipantsHandler.AdminExport)
						r.Route("/{participantId}", func(r chi.Router) {
							r.Get("/", d.eventParticipantsHandler.AdminGet)
							r.Patch("/", d.eventParticipantsHandler.AdminUpdate)
							r.Post("/block", d.eventParticipantsHandler.AdminBlock())
							r.Post("/unblock", d.eventParticipantsHandler.AdminUnblock())
							r.Post("/archive", d.eventParticipantsHandler.AdminArchive())
							r.Get("/points", d.pointsHandler.AdminOverview)
							r.Post("/points/adjustments", d.pointsHandler.AdminAdjustment)
						})
					})
				})
			})

			r.Group(func(r chi.Router) {
				r.Use(adminOrSuper)
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
					r.Get("/briefing", d.challengesHandler.AdminGetBriefing)
					r.Put("/briefing", d.challengesHandler.AdminSaveBriefing)
					r.Post("/briefing/files", d.challengesHandler.AdminUploadBriefingFile)
					r.Delete("/briefing/files/{fileId}", d.challengesHandler.AdminDeleteBriefingFile)
					r.Put("/briefing/contestants/{userId}", d.challengesHandler.AdminSaveOverride)
					r.Delete("/briefing/contestants/{userId}", d.challengesHandler.AdminClearOverride)
					r.Post("/briefing/contestants/{userId}/files", d.challengesHandler.AdminUploadOverrideFile)
					r.Get("/submissions", d.submissionsHandler.AdminList)
					r.Get("/evaluation", d.evaluationHandler.Get)
					r.Put("/evaluation", d.evaluationHandler.Put)
					r.Get("/evaluation/scores", d.evaluationHandler.AdminScoreboard)
					r.Put("/evaluation/numeric-results", d.evaluationHandler.SetNumericResult)
					r.Put("/evaluation/score-override", d.evaluationHandler.OverrideScore)
					r.Post("/evaluation/reset-results", d.evaluationHandler.ResetResults)
					r.Post("/evaluation/replace-jury", d.evaluationHandler.ReplaceJury)
					r.Put("/evaluation/remote-jury", d.evaluationHandler.PutRemoteJury)
					r.Get("/evaluation/remote-jury", d.evaluationHandler.SearchRemoteJury)
					r.Put("/evaluation/stage-link", d.evaluationHandler.PutStageLink)
					r.Post("/evaluation/criteria", d.evaluationHandler.AddCriterion)
					r.Patch("/evaluation/criteria/reorder", d.evaluationHandler.ReorderCriteria)
					r.Patch("/evaluation/criteria/{criterionId}", d.evaluationHandler.UpdateCriterion)
					r.Delete("/evaluation/criteria/{criterionId}", d.evaluationHandler.DeleteCriterion)
					r.Get("/live", d.evaluationHandler.AdminLive)
					r.Get("/live/stream", d.evaluationHandler.AdminLiveStream)
					r.Post("/live/start", d.evaluationHandler.AdminStart)
					r.Post("/live/pause", d.evaluationHandler.AdminPause)
					r.Post("/live/finish", d.evaluationHandler.AdminFinish)
					r.Post("/live/restart", d.evaluationHandler.AdminRestart)
					r.Post("/live/restart-timer", d.evaluationHandler.AdminRestartTimer)
					r.Post("/live/complete-contestant", d.evaluationHandler.AdminCompleteContestant)
					r.Post("/live/end-speech", d.evaluationHandler.AdminEndSpeech)
					r.Post("/live/durations", d.evaluationHandler.AdminSetDurations)
					r.Post("/live/current-contestant", d.evaluationHandler.AdminSetContestant)
					r.Post("/live/phase", d.evaluationHandler.AdminSetPhase)
					r.Post("/live/question", d.evaluationHandler.AdminStepQuestion)
					r.Post("/live/correct-answer", d.evaluationHandler.AdminSetCorrectAnswer)
					r.Put("/live/question-plan", d.evaluationHandler.AdminSetQuestionPlan)
					r.Post("/live/draw", d.evaluationHandler.AdminShuffleDraw)
					r.Put("/live/draw", d.evaluationHandler.AdminReorderDraw)
				})
				r.Route("/submissions/{submissionId}", func(r chi.Router) {
					r.Get("/", d.submissionsHandler.AdminGet)
					r.Get("/files/{fileId}", d.submissionsHandler.DownloadFile)
				})
			})

			r.Group(func(r chi.Router) {
				r.Use(superOrMega)
				r.Get("/users", d.userAdminHandler.ListUsers)
				r.Post("/users", d.userAdminHandler.CreateUser)
				r.Get("/users/{userId}", d.userAdminHandler.GetUser)
				r.Patch("/users/{userId}", d.userAdminHandler.UpdateUser)
				r.Post("/users/{userId}/reset-password", d.userAdminHandler.ResetPassword)
				r.Post("/users/{userId}/block", d.userAdminHandler.Block)
				r.Post("/users/{userId}/unblock", d.userAdminHandler.Unblock)
				r.Post("/users/{userId}/roles", d.userAdminHandler.AssignRole)
				r.Delete("/users/{userId}/roles", d.userAdminHandler.RemoveRole)
				r.Get("/users/{userId}/staff-permissions", d.staffHandler.ListForUser)
				r.Put("/users/{userId}/staff-permissions", d.staffHandler.ReplaceForUser)
				r.Delete("/users/{userId}/staff-permissions", d.staffHandler.ClearForUser)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(d.authn.Require)
			r.Use(middleware.RequireRole("JURY"))
			r.Get("/jury/contests", d.evaluationHandler.JuryContests)
			r.Get("/jury/challenges/{challengeId}/live", d.evaluationHandler.JuryLive)
			r.Get("/jury/challenges/{challengeId}/live/stream", d.evaluationHandler.JuryLiveStream)
			r.Get("/jury/challenges/{challengeId}/scorecard", d.evaluationHandler.JuryScorecard)
			r.Put("/jury/challenges/{challengeId}/scorecard", d.evaluationHandler.JurySetScore)
			r.Get("/jury/challenges/{challengeId}/briefing", d.challengesHandler.JuryGetBriefing)
			r.Get("/jury/challenges/{challengeId}/submissions", d.submissionsHandler.JuryList)
			r.Get("/jury/submissions/{submissionId}", d.submissionsHandler.JuryGet)
			r.Get("/jury/submissions/{submissionId}/files/{fileId}", d.submissionsHandler.JuryDownloadFile)
			r.Post("/jury/challenges/{challengeId}/lives/wrong", d.evaluationHandler.JuryWrongAnswer)
			r.Post("/jury/challenges/{challengeId}/lives/restore", d.evaluationHandler.JuryRestoreLife)
			r.Post("/jury/challenges/{challengeId}/lives/answer", d.evaluationHandler.JurySetAnswer)
		})

		// Чтение испытаний конкурсантом: нужен access-токен, роль не важна —
		// доступ определяется участием в конкурсе (проверка в сервисе).
		r.Group(func(r chi.Router) {
			r.Use(d.authn.Require)
			r.Get("/my/contests", d.contestsHandler.MyContests)
			r.Get("/contests/{contestId}/challenges", d.challengesHandler.ContestantList)
			r.Get("/contests/{contestId}/draws", d.evaluationHandler.ContestantContestDraws)
			r.Get("/challenges/{challengeId}", d.challengesHandler.ContestantGet)
			r.Get("/challenges/{challengeId}/briefing", d.challengesHandler.ContestantGetBriefing)
			r.Get("/challenges/{challengeId}/draw", d.evaluationHandler.ContestantDraw)

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
			r.Get("/login-options", d.eventParticipantsHandler.LoginOptions)
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

		// Независимый participant auth flow, scoped по slug мероприятия.
		r.Route("/events/{eventSlug}/participant-auth", func(r chi.Router) {
			r.Use(middleware.SecurityHeaders)
			r.Get("/vk/start", d.eventParticipantsHandler.VKStart)
			r.Group(func(r chi.Router) {
				r.Use(middleware.CSRFOrigin(origins...))
				r.Post("/fio", d.eventParticipantsHandler.LoginByName)
				r.Post("/union-card", d.eventParticipantsHandler.LoginByUnionCard)
				r.Post("/sks", d.eventParticipantsHandler.LoginBySKSBarcode)
				r.Post("/telegram", d.eventParticipantsHandler.LoginByTelegram)
				r.Post("/telegram/webapp", d.eventParticipantsHandler.LoginByTelegramWebApp)
				r.Post("/vk", d.eventParticipantsHandler.LoginByVKToken)
			})
		})
		r.Route("/participant-auth", func(r chi.Router) {
			r.Use(middleware.SecurityHeaders)
			r.Get("/vk/start", d.eventParticipantsHandler.VKStart)
			r.Get("/vk/callback", d.eventParticipantsHandler.VKCallback)
			r.Group(func(r chi.Router) {
				r.Use(middleware.CSRFOrigin(origins...))
				r.Post("/telegram", d.eventParticipantsHandler.LoginByTelegram)
				r.Post("/telegram/webapp", d.eventParticipantsHandler.LoginByTelegramWebApp)
				r.Post("/vk", d.eventParticipantsHandler.LoginByVKToken)
				r.Post("/continue", d.eventParticipantsHandler.ContinueSocialLogin)
			})
		})

		r.Route("/participant", func(r chi.Router) {
			r.Use(middleware.SecurityHeaders)
			r.Use(d.participantAuthn.Require)
			r.Get("/me", d.eventParticipantsHandler.Me)
			r.Get("/points", d.pointsHandler.ParticipantOverview)
			r.Get("/lectures", d.lecturesHandler.ParticipantLectures)
			r.Get("/tasks", d.eventTasksHandler.ParticipantList)
			r.Get("/tasks/{taskId}", d.eventTasksHandler.ParticipantGet)
			r.With(middleware.CSRFOrigin(origins...)).Post("/tasks/{taskId}/submissions", d.eventTasksHandler.ParticipantSubmit)
			r.Get("/task-assets/{assetId}", d.eventTasksHandler.ParticipantAsset)
			r.Get("/merch", d.merchHandler.ParticipantProducts)
			r.Get("/merch/{productSlug}", d.merchHandler.ParticipantProduct)
			r.With(middleware.CSRFOrigin(origins...)).Put("/merch-saving-target", d.merchHandler.ParticipantSetTarget)
			r.With(middleware.CSRFOrigin(origins...)).Delete("/merch-saving-target", d.merchHandler.ParticipantDeleteTarget)
			r.Get("/orders", d.merchHandler.ParticipantOrders)
			r.With(middleware.CSRFOrigin(origins...)).Post("/orders", d.merchHandler.ParticipantReserve)
			r.Get("/orders/{orderId}", d.merchHandler.ParticipantOrder)
			r.With(middleware.CSRFOrigin(origins...)).Post("/orders/{orderId}/cancel", d.merchHandler.ParticipantCancel)
			r.With(middleware.CSRFOrigin(origins...)).Post("/qr", d.lecturesHandler.ParticipantCode)
			r.With(middleware.CSRFOrigin(origins...)).Post("/logout", d.eventParticipantsHandler.Logout)
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
