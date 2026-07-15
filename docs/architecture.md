# Архитектура — Student Leader Cabinet

## Обзор

Модульный монолит на Go с отдельным worker-процессом и SPA-фронтендом.
Первая версия — простая для поддержки, но расширяемая (SITE.md §58).

```
Браузер ──HTTPS──▶ nginx ──┬─▶ frontend/dist (статика SPA)
                           └─▶ /api/ ─▶ Go API (127.0.0.1:8080)
                                             │
                        ┌────────────────────┼────────────────────┐
                        ▼                     ▼                    ▼
                    PostgreSQL             Redis               MinIO (S3)
                        ▲
                   Go worker (outbox → Telegram, файлы, экспорт)
```

## Слои backend

- `cmd/api`, `cmd/worker` — точки входа.
- `internal/config` — конфигурация из окружения (§29).
- `internal/platform` — инфраструктура: `db`, `httpserver`, `logger`, далее `storage`, `telegram`, `security`.
- `internal/app` — сборка зависимостей и роутинг.
- `internal/modules/*` — бизнес-модули (auth, users, contests, …) — добавляются по этапам.
- `internal/middleware` — сквозные middleware.

## Ключевые решения

См. ADR в `docs/ADR/`:
1. Модульный монолит.
2. JWT session-модель (access в памяти, refresh в HttpOnly-cookie, ротация).
3. Хранение динамических форм (JSONB + версионирование схемы + immutable-ревизии).
4. Файловое хранилище (S3/MinIO, presigned upload, метаданные в БД).
5. Transactional outbox для внешних side effects (Telegram).

## Контракт API

Единый envelope (§20), стабильные error codes (§50), версионированный префикс `/api/v1`.
Источник истины — `backend/api/openapi.yaml`.

## Безопасность

RBAC со scope конкурса, проверка прав только на backend, presigned URL с коротким TTL,
bucket закрыт, секреты через окружение. Подробно — §16, `docs/security.md` (добавится на Этапе 9).
