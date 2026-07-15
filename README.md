# Student Leader Cabinet

Личный кабинет и административная панель конкурса «Студенческий лидер».
Модульный монолит: **Go** (API + worker) + **React/Vite/TS** (SPA) + **PostgreSQL / Redis / MinIO**.

Полная спецификация — [SITE.md](./SITE.md). Визуальные правила — [DESIGN.md](./DESIGN.md).

## Стек и рантайм (этот сервер)

- Зависимости (Postgres, Redis, MinIO) — в Docker Compose, порты только на `127.0.0.1`.
- Go `api` и `worker` — нативные бинарники под systemd (`infra/systemd/`).
- nginx раздаёт `frontend/dist` и проксирует `/api/` на `127.0.0.1:8080`.
- Домен: **eazytech.ru** (SSL через Certbot).

## Быстрый старт (разработка)

```bash
cp .env.example .env      # заполнить секреты
make up                   # поднять postgres/redis/minio
make api-run              # запустить API (:8080)
make frontend-dev         # фронтенд (:5173)
```

## Структура

```
backend/    Go: cmd/{api,worker}, internal/{app,config,platform,modules}, api/openapi.yaml, db/migrations
frontend/   React SPA (см. frontend/README.md)
infra/      systemd-юниты, nginx, prometheus, скрипты
docs/       архитектура, ADR, деплой
```

## Команды

`make help` — полный список. Основные: `up`, `down`, `build`, `api-run`, `worker-run`, `frontend-dev`, `lint`, `fmt`.

## Этапы

Разработка идёт по этапам из [SITE.md §47](./SITE.md). Статус ведём здесь — обновлять по мере работы.

| Этап | Что | Статус |
|------|-----|--------|
| 0 | Инфраструктура и скелеты | ✅ готово |
| 1 | Авторизация (JWT, refresh, сессии, RBAC, аудит) | ✅ готово |
| 2 | Конкурсы и конкурсанты | ✅ готово |
| 3 | Испытания и конструктор форм | ✅ готово |
| 4 | Подача ответов (submissions, черновики, ревизии, файлы) | ✅ готово |
| 5 | Уведомления (outbox + Telegram) | ✅ готово |
| 6–10 | Справка, аудит/экспорт, hardening, будущие модули | ⬜ не начато |

**Текущий фокус:** ручной прогон в браузере + подключение реального Telegram-бота (токен в `.env`). Дальше — Этап 6 (файлы/справка по дорожной карте SITE.md §47).

Детальное состояние и точки входа между сессиями — [docs/STATUS.md](./docs/STATUS.md).
