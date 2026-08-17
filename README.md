# Student Leader Portal

Веб-платформа конкурса «Студенческий лидер» и мероприятий: административная
панель, кабинет конкурсанта и отдельный кабинет участника мероприятия.

Production: [eazytech.ru](https://eazytech.ru)

## Возможности

### Административная панель

- JWT-аутентификация, refresh-сессии, RBAC и аудит действий;
- конкурсы, конкурсанты, испытания и конструктор динамических форм;
- черновики, отправка работ, immutable-ревизии и приватные файлы;
- списки участников мероприятий, CSV/XLSX-импорт, экспорт и блокировка;
- PointsLedger с идемпотентными начислениями и корректировками;
- лекции, динамические QR-коды, camera/USB/manual attendance scanner;
- задания, подтверждения выполнения и очередь модерации;
- каталог мерча, резервирование остатков и баллов, выдача и отмена заказов;
- transactional outbox и Telegram-уведомления.

### Кабинет участника мероприятия

- вход по ФИО и дате рождения, профсоюзному билету или barcode СКС;
- баланс и история начислений;
- короткоживущий QR-код посещения;
- задания, повторная отправка после отклонения и приватные вложения;
- магазин, цель накопления и история заказов.

## Архитектура

Модульный монолит:

- `backend` — Go, Chi, pgx, PostgreSQL, AWS SDK for Go v2;
- `frontend` — React 18, TypeScript, Vite, TanStack Query, Tailwind CSS;
- `PostgreSQL` — единственный локальный infrastructure-контейнер;
- `Timeweb Cloud S3` — внешнее приватное файловое хранилище;
- `nginx` — HTTPS, раздача SPA и проксирование API.

MinIO и Redis проекту не требуются.

## Быстрый старт

Понадобятся Go toolchain из [`backend/go.mod`](./backend/go.mod), Node.js, npm и
Docker Compose.

1. Создайте локальную конфигурацию и заполните секреты:

   ```bash
   cp .env.example .env
   ```

2. Запустите PostgreSQL:

   ```bash
   docker compose up -d postgres
   ```

3. Примените миграции и запустите API:

   ```bash
   set -a
   . ./.env
   set +a
   (cd backend && go run ./cmd/admin migrate)
   make api-run
   ```

4. В другом терминале запустите frontend:

   ```bash
   cd frontend
   npm ci
   npm run dev
   ```

По умолчанию API слушает `127.0.0.1:8080`, Vite — `127.0.0.1:5173`.

## Конфигурация

Все runtime-настройки читаются из окружения. Полный безопасный шаблон находится
в [`.env.example`](./.env.example). Основные группы переменных:

- `POSTGRES_*` — подключение к PostgreSQL;
- `JWT_*`, `COOKIE_*` — staff-аутентификация;
- `PARTICIPANT_*` — participant session, rate limit и QR;
- `S3_*` — Timeweb Cloud S3 и presigned URL;
- `TELEGRAM_*` — доставка уведомлений;
- `FEATURE_*` — модули приложения.

Файл `.env` игнорируется Git. Реальные пароли, JWT-ключи и S3 credentials нельзя
добавлять в репозиторий.

## Проверки

Backend:

```bash
cd backend
go test -race ./...
go vet ./...
```

Frontend:

```bash
cd frontend
npm run lint
npm run build
```

Production smoke-сценарии находятся в [`backend/scripts`](./backend/scripts).
Они создают и изменяют данные, поэтому запускать их следует только в подготовленном
окружении.

## Структура репозитория

```text
backend/
  api/                 OpenAPI-контракт
  cmd/                 API, admin CLI и worker
  internal/modules/    бизнес-модули
  internal/platform/   DB, migrations, security, S3, HTTP
  scripts/             smoke-сценарии
frontend/
  src/app/             router и guards
  src/entities/        API-клиенты, типы и queries
  src/pages/           staff, contestant и participant UI
infra/                 PM2/systemd-конфигурация
docs/                  архитектура, ADR и эксплуатационные документы
```

## Документация

- [SITE.md](./SITE.md) — основная техническая спецификация;
- [codex_event_platform_spec.md](./codex_event_platform_spec.md) — требования платформы мероприятий;
- [docs/STATUS.md](./docs/STATUS.md) — текущее состояние проекта;
- [work_user_func.md](./work_user_func.md) — журнал реализации и следующий шаг;
- [backend/api/openapi.yaml](./backend/api/openapi.yaml) — OpenAPI 3.0;
- [deploy_server.md](./deploy_server.md) — развёртывание и обслуживание сервера;
- [DESIGN.md](./DESIGN.md) — визуальные правила интерфейса.

## Текущий статус

Основное бизнес-ядро развёрнуто: участники, баллы, лекции и attendance, задания,
мерч и Timeweb S3. Ближайший этап — release hardening, browser acceptance и
расширение интеграционных тестов. Детали и известный технический долг ведутся в
[`work_user_func.md`](./work_user_func.md).
