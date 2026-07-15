// Package auth реализует аутентификацию, сессии и refresh-ротацию (SITE.md §16).
package auth

import (
	"errors"
	"time"
)

// Доменные ошибки — маппятся на стабильные error codes API (SITE.md §50).
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountBlocked     = errors.New("account blocked")
	ErrAccountLocked      = errors.New("account temporarily locked")
	ErrSessionExpired     = errors.New("session expired")
	ErrRefreshReused      = errors.New("refresh token reused")
	ErrPasswordTooShort   = errors.New("password too short")
	ErrWrongOldPassword   = errors.New("wrong old password")
)

const (
	StatusActive  = "ACTIVE"
	StatusBlocked = "BLOCKED"

	maxFailedLogins = 5
	lockDuration    = 15 * time.Minute
	minPasswordLen  = 10
)

// User — пользователь с ролью для выпуска токена.
type User struct {
	ID                 string
	Login              string
	PasswordHash       string
	FullName           string
	Status             string
	MustChangePassword bool
	FailedLoginCount   int
	LockedUntil        *time.Time
}

// Role с областью действия (глобально или в рамках конкурса).
type Role struct {
	Code      string
	ScopeType string
	ScopeID   string
}

// Session — активная сессия пользователя.
type Session struct {
	ID          string
	UserID      string
	UserAgent   string
	IPHash      string
	LastUsedAt  time.Time
	ExpiresAt   time.Time
	CreatedAt   time.Time
	RevokedAt   *time.Time
	Current     bool
}

// TokenPair — результат логина/refresh.
type TokenPair struct {
	AccessToken  string
	AccessExp    time.Time
	RefreshToken string
	RefreshExp   time.Time
	SessionID    string
}
