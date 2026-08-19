package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/modules/audit"
	"github.com/eazytech/student-leader-cabinet/internal/modules/auth"
	"github.com/eazytech/student-leader-cabinet/internal/modules/challenges"
	"github.com/eazytech/student-leader-cabinet/internal/modules/contests"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventparticipants"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventtasks"
	"github.com/eazytech/student-leader-cabinet/internal/modules/lectures"
	"github.com/eazytech/student-leader-cabinet/internal/modules/merch"
	"github.com/eazytech/student-leader-cabinet/internal/modules/points"
	"github.com/eazytech/student-leader-cabinet/internal/modules/submissions"
	"github.com/eazytech/student-leader-cabinet/internal/modules/useradmin"
	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
	"github.com/eazytech/student-leader-cabinet/internal/platform/storage"
)

// deps — собранные зависимости приложения.
type deps struct {
	authHandler              *auth.Handler
	authn                    *middleware.Authenticator
	contestsHandler          *contests.Handler
	challengesHandler        *challenges.Handler
	userAdminHandler         *useradmin.Handler
	staffHandler             *eventpermissions.Handler
	submissionsHandler       *submissions.Handler
	eventParticipantsHandler *eventparticipants.Handler
	participantAuthn         *eventparticipants.Authenticator
	pointsHandler            *points.Handler
	lecturesHandler          *lectures.Handler
	eventTasksHandler        *eventtasks.Handler
	merchHandler             *merch.Handler
}

func (a *App) build() *deps {
	jwtMgr := security.NewJWTManager(
		a.cfg.JWT.AccessSecret, a.cfg.JWT.Issuer, a.cfg.JWT.Audience, a.cfg.JWT.AccessTTL,
	)
	auditSvc := audit.New(a.pool, a.log)
	repo := auth.NewRepo(a.pool)
	authSvc := auth.NewService(repo, jwtMgr, auditSvc, a.cfg.JWT.RefreshTTL)

	cookie := auth.CookieConfig{
		Name:     "slc_refresh",
		Domain:   a.cfg.Cookie.Domain,
		Secure:   a.cfg.Cookie.Secure,
		SameSite: parseSameSite(a.cfg.Cookie.SameSite),
		Path:     "/api/v1/auth",
	}
	contestsRepo := contests.NewRepo(a.pool)
	contestsSvc := contests.NewService(contestsRepo, auditSvc)
	challengesRepo := challenges.NewRepo(a.pool)
	challengesSvc := challenges.NewService(challengesRepo, contestsRepo, auditSvc)
	userAdminSvc := useradmin.NewService(a.pool, auditSvc)
	staffSvc := eventpermissions.NewService(eventpermissions.NewRepo(a.pool), auditSvc)
	authSvc.SetStaffDirectory(staffSvc)
	eventParticipantsSvc := eventparticipants.NewService(
		eventparticipants.NewRepo(a.pool), auditSvc, a.cfg.ParticipantAuth.SessionTTL,
	)
	vkRedirect := strings.TrimSpace(a.cfg.VK.RedirectURL)
	if vkRedirect == "" {
		vkRedirect = strings.TrimRight(a.cfg.App.BaseURL, "/") + "/api/v1/participant-auth/vk/callback"
	}
	eventParticipantsSvc.SetSocialAuth(eventparticipants.SocialAuth{
		TelegramBotToken:    a.cfg.Telegram.BotToken,
		TelegramBotUsername: a.cfg.Telegram.BotUsername,
		VKClientID:          a.cfg.VK.ClientID,
		VKClientSecret:      a.cfg.VK.ClientSecret,
		VKServiceToken:      a.cfg.VK.ServiceToken,
		VKRedirectURL:       vkRedirect,
		PublicBaseURL:       strings.TrimRight(a.cfg.App.BaseURL, "/"),
		StateSecret:         a.cfg.ParticipantAuth.QRSecret,
	}, &http.Client{Timeout: 10 * time.Second})
	pointsRepo := points.NewRepo(a.pool, auditSvc)
	pointsSvc := points.NewService(pointsRepo)
	lectureCodes := lectures.NewCodeManager(a.cfg.ParticipantAuth.QRSecret, a.cfg.ParticipantAuth.QRTTL)
	lecturesSvc := lectures.NewService(
		lectures.NewRepo(a.pool, pointsRepo, auditSvc), lectureCodes, auditSvc,
	)
	maxTaskImageBytes := int64(a.cfg.Limits.MaxFileSizeMB) << 20
	if maxTaskImageBytes <= 0 || maxTaskImageBytes > 20<<20 {
		maxTaskImageBytes = 20 << 20
	}
	participantCookie := eventparticipants.CookieConfig{
		Name: a.cfg.ParticipantAuth.CookieName, Domain: a.cfg.Cookie.Domain,
		Secure: a.cfg.Cookie.Secure, SameSite: parseSameSite(a.cfg.Cookie.SameSite),
		Path: "/api/v1",
	}
	participantLoginLimiter := eventparticipants.NewMemoryRateLimiter(
		a.cfg.ParticipantAuth.RateLimitAttempts, a.cfg.ParticipantAuth.RateLimitWindow,
	)

	// Объектное хранилище — best-effort: если S3 не настроен, обложки/файлы
	// просто не отдаются (handler nil-safe), но запуск API не падает.
	var store *storage.Storage
	if st, err := storage.New(a.cfg.S3); err != nil {
		a.log.Warn("storage init failed; file features disabled", "err", err)
	} else {
		store = st
	}
	// nil-интерфейсы, если хранилище недоступно (иначе typed-nil != nil).
	var imageStore contests.ImageStore
	var fileStore submissions.FileStore
	var taskFileStore eventtasks.FileStore
	var merchFileStore merch.FileStore
	if store != nil {
		imageStore = store
		fileStore = store
		taskFileStore = store
		merchFileStore = store
	}

	submissionsSvc := submissions.NewService(
		submissions.NewRepo(a.pool),
		submissions.NewChallengeAdapter(challengesRepo, contestsRepo),
		auditSvc,
	)
	if store != nil {
		submissionsSvc.SetPresigner(store.PresignGet)
	}
	eventTasksSvc := eventtasks.NewService(
		eventtasks.NewRepo(a.pool, pointsRepo, auditSvc), auditSvc, maxTaskImageBytes,
	)
	if store != nil {
		eventTasksSvc.SetPresigner(store.PresignGet)
	}
	merchSvc := merch.NewService(
		merch.NewRepo(a.pool, pointsRepo, auditSvc), auditSvc, maxTaskImageBytes,
	)
	if store != nil {
		merchSvc.SetPresigner(store.PresignGet)
	}

	return &deps{
		authHandler:              auth.NewHandler(authSvc, cookie),
		authn:                    middleware.NewAuthenticator(jwtMgr, repo),
		contestsHandler:          contests.NewHandler(contestsSvc, imageStore),
		challengesHandler:        challenges.NewHandler(challengesSvc),
		userAdminHandler:         useradmin.NewHandler(userAdminSvc),
		staffHandler:             eventpermissions.NewHandler(staffSvc),
		submissionsHandler:       submissions.NewHandler(submissionsSvc, fileStore, a.cfg.Limits.MaxFileSizeMB),
		eventParticipantsHandler: eventparticipants.NewHandler(eventParticipantsSvc, participantCookie, participantLoginLimiter),
		participantAuthn:         eventparticipants.NewAuthenticator(eventParticipantsSvc, participantCookie.Name),
		pointsHandler:            points.NewHandler(pointsSvc),
		lecturesHandler:          lectures.NewHandler(lecturesSvc),
		eventTasksHandler:        eventtasks.NewHandler(eventTasksSvc, taskFileStore, maxTaskImageBytes),
		merchHandler:             merch.NewHandler(merchSvc, merchFileStore, maxTaskImageBytes),
	}
}

func parseSameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
