package app

import (
	"net/http"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/modules/audit"
	"github.com/eazytech/student-leader-cabinet/internal/modules/auth"
	"github.com/eazytech/student-leader-cabinet/internal/modules/challenges"
	"github.com/eazytech/student-leader-cabinet/internal/modules/contests"
	"github.com/eazytech/student-leader-cabinet/internal/modules/submissions"
	"github.com/eazytech/student-leader-cabinet/internal/modules/useradmin"
	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
	"github.com/eazytech/student-leader-cabinet/internal/platform/storage"
)

// deps — собранные зависимости приложения.
type deps struct {
	authHandler       *auth.Handler
	authn             *middleware.Authenticator
	contestsHandler    *contests.Handler
	challengesHandler  *challenges.Handler
	userAdminHandler   *useradmin.Handler
	submissionsHandler *submissions.Handler
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

	// Объектное хранилище — best-effort: если MinIO недоступен, обложки/файлы
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
	if store != nil {
		imageStore = store
		fileStore = store
	}

	submissionsSvc := submissions.NewService(
		submissions.NewRepo(a.pool),
		submissions.NewChallengeAdapter(challengesRepo, contestsRepo),
		auditSvc,
	)
	if store != nil {
		submissionsSvc.SetPresigner(store.PresignGet)
	}

	return &deps{
		authHandler:        auth.NewHandler(authSvc, cookie),
		authn:              middleware.NewAuthenticator(jwtMgr, repo),
		contestsHandler:    contests.NewHandler(contestsSvc, imageStore),
		challengesHandler:  challenges.NewHandler(challengesSvc),
		userAdminHandler:   useradmin.NewHandler(userAdminSvc),
		submissionsHandler: submissions.NewHandler(submissionsSvc, fileStore, a.cfg.Limits.MaxFileSizeMB),
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
