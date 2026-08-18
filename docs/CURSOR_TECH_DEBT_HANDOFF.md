# Handoff для Cursor: аудит состояния и план устранения техдолга

Дата аудита: **2026-08-18**  
Базовый коммит: **`b2fdd6f`** (`master`, синхронизирован с `origin/master`)  
Репозиторий: **`/var/www/student-leader-portal`**

## 1. Назначение документа

Это рабочий контекст для следующего AI-агента. Не повторяй широкий аудит с нуля:
сначала перепроверь конкретные утверждения ниже, затем устраняй проблемы небольшими,
проверяемыми изменениями в порядке приоритета.

Главный вывод аудита: бизнес-функции в основном реализованы, проект собирается и
локально работает, но в текущем виде остаются критичные проблемы с RBAC,
мультиарендной изоляцией и эксплуатационной надёжностью.

## 2. Правила работы

1. Считай каталог репозитория одновременно рабочим каталогом live-развёртывания.
   Не выполняй деплой, миграции production-БД, рестарт PM2/nginx и изменения DNS без
   отдельного разрешения владельца.
2. Перед правкой проверь `git status`; не перезаписывай чужие незакоммиченные изменения.
3. Не выводи содержимое `.env`, токены, пароли и presigned URL в логи или отчёты.
4. Для каждого security-исправления сначала добавь регрессионный тест, который
   воспроизводит запрещённый сценарий.
5. Не объединяй все пункты в один большой рефакторинг. Рекомендуемая единица работы:
   одна проблема, тесты, миграция при необходимости, обновление OpenAPI/документации.
6. Не применяй `npm audit fix --force` вслепую: исправление React Router может
   потребовать major-upgrade и адаптацию API.
7. Если решение меняет утверждённую RBAC-модель, сначала сверяйся с
   `docs/RBAC_MULTITENANCY.md`; не придумывай новую модель доступа.

## 3. Краткая карта проекта

- Backend: Go, `backend/`, модульный монолит, PostgreSQL/pgx, JWT access/refresh.
- Frontend: React + TypeScript + Vite, `frontend/`.
- Файлы: S3-совместимое хранилище Timeweb.
- Runtime: nginx, PM2 (`eazytech-api`, `eazytech-worker`), PostgreSQL в Docker.
- Миграции: `backend/migrations/`, на момент аудита применены `0001`–`0015`.
- Основные архитектурные документы: `docs/architecture.md`,
  `docs/RBAC_MULTITENANCY.md`, `docs/ADR/`.

## 4. P0: критичные задачи

### P0.1. Закрыть захват учётных записей через unscoped user administration

#### Проблема

Маршруты сброса пароля и блокировки находятся в общей ADMIN/SUPER_ADMIN-группе:

- `backend/internal/app/router.go`, блок `/users/{userId}`;
- `backend/internal/modules/useradmin/service.go`, методы `ResetPassword` и `SetStatus`.

Сервис выполняет глобальный `UPDATE users WHERE id=$1`, не проверяя:

- роль и область прав актора;
- `created_by` целевого пользователя;
- принадлежность целевого пользователя конкурсу актора;
- запрет воздействия на равную или более привилегированную роль.

Дополнительно `backend/internal/modules/contests/repo_participants.go` при добавлении
участника использует `INSERT ... ON CONFLICT(login) DO UPDATE ... RETURNING id`.
Это позволяет привязать существующий логин к своему конкурсу, узнать UUID аккаунта,
а затем вызвать глобальный reset-password. Так потенциально захватывается чужая,
в том числе привилегированная, учётная запись.

Есть и более простой случай: ADMIN видит UUID участников через API конкурса и может
вызвать unscoped reset/block, хотя утверждённая модель прямо запрещает ADMIN управлять
участниками.

#### Требуемое направление исправления

1. Убрать reset/block/unblock из зоны доступа `ADMIN` либо поставить обязательный
   service-level guard. Согласно `docs/RBAC_MULTITENANCY.md`, участниками управляют
   только владелец-SUPER_ADMIN и MEGA_ADMIN.
2. Централизовать проверку `CanManageUser(actor, target, action)` в сервисном слое:
   маршрутизация и frontend не считаются границей безопасности.
3. SUPER_ADMIN может управлять только пользователями своей цепочки `created_by` и
   участниками принадлежащих ему конкурсов. MEGA_ADMIN может работать глобально.
4. Запретить SUPER_ADMIN изменять/сбрасывать MEGA_ADMIN, чужого SUPER_ADMIN и чужие
   tenant-аккаунты.
5. Пересмотреть `ON CONFLICT(login)` при добавлении участника. Нельзя молча захватывать
   существующий глобальный аккаунт. Для конфликта должен быть безопасный явный сценарий
   с проверкой владельца и роли либо `409 Conflict`.
6. После изменения пароля, статуса или ролей отзывать активные сессии целевого
   пользователя в той же согласованной операции.

#### Обязательные тесты

- ADMIN получает `403` на reset/block/unblock любого пользователя.
- SUPER_ADMIN A получает `403` для пользователя, созданного SUPER_ADMIN B.
- SUPER_ADMIN не может воздействовать на MEGA_ADMIN или другого SUPER_ADMIN.
- SUPER_ADMIN может управлять собственным конкурсантам согласно спецификации.
- MEGA_ADMIN сохраняет разрешённый глобальный доступ.
- Добавление существующего чужого/привилегированного login не меняет аккаунт и не
  раскрывает пригодный для дальнейшего захвата идентификатор.
- Запрещённые действия не меняют БД и создают корректный security/audit event, если
  такой формат принят в проекте.

### P0.2. Исправить BOLA в реестре пользователей и ролях

#### Проблема

- `backend/internal/modules/useradmin/service_users.go`: `Get` и часть `Update`
  недостаточно ограничивают target по владельцу.
- `backend/internal/modules/useradmin/service_roles.go`: проверяется право на конкурс,
  но не гарантируется, что целевой пользователь принадлежит актору. В UI также можно
  создать/назначить глобальный ADMIN, что противоречит per-contest модели.
- После снятия привилегированной роли существующий access JWT может сохранять старые
  полномочия до истечения TTL.

#### Definition of Done

- Все list/get/update/assign/remove операции фильтруются по actor и `created_by`.
- `ADMIN + CONTEST` требует `scope_id` и `access_level=EDIT|VIEW`.
- Глобальный ADMIN запрещён, если для него нет отдельно утверждённого legacy-сценария.
- Нельзя назначить роль чужому пользователю, даже если актор владеет указанным
  конкурсом.
- При security-sensitive изменении ролей отзываются сессии либо вводится надёжная
  проверка актуальной версии полномочий.
- Покрыты тестами tenant A / tenant B / MEGA / SUPER / ADMIN EDIT / ADMIN VIEW.

### P0.3. Устранить перехват API защитной страницей KillBot

Это преимущественно инфраструктурная задача и может находиться вне репозитория.
На момент аудита:

- `eazytech.ru` резолвился на адреса KillBot;
- `/`, `/health/ready`, `/api/v1/config` возвращали `200 text/html` со страницей
  проверки пользователя вместо ожидаемого JSON;
- `www.eazytech.ru` и прямой origin `201.51.29.205` отвечали корректно.

Нельзя менять DNS/WAF без явного разрешения. Подготовить безопасное предложение:

- bypass challenge для `/api/*`, `/health/*`, `/assets/*`;
- health check должен проверять status, `Content-Type` и небольшой JSON-маркер, а не
  только HTTP 200;
- отдельно проверить CORS, cookie и refresh flow через публичный apex-домен.

### P0.4. Настроить резервное копирование и восстановление

На сервере не обнаружены project-level backup scripts, cron или systemd timer для БД.
PostgreSQL находится в единственном Docker volume.

Требуется отдельное согласование инфраструктурных изменений. Целевое состояние:

- ежедневный зашифрованный `pg_dump`;
- off-host storage и ограниченный доступ;
- retention, мониторинг свежести и ошибок;
- инвентаризация/backup критичных S3-объектов;
- документированный restore runbook и регулярный тест восстановления.

## 5. P1: высокий приоритет

### P1.1. Обновить уязвимые backend-зависимости

`govulncheck` нашёл достижимые уязвимости:

- `github.com/jackc/pgx/v5` 5.7.1 — GO-2026-5004, исправлено в 5.9.2;
- `github.com/golang-jwt/jwt/v5` 5.2.1 — GO-2025-3553, исправлено в 5.2.2;
- `github.com/xuri/excelize/v2` 2.10.0 — GO-2026-5960, исправлено в 2.11.0.

Обновлять по одной группе, изучая changelog. После обновления выполнить полный набор
проверок из раздела 8 и повторить `govulncheck`.

### P1.2. Ограничить JSON request body

Параметры `MAX_JSON_BODY_MB` и `DEFAULT_MAX_SUBMISSION_SIZE_MB` загружаются из config,
но практически не применяются. Многие endpoints используют
`json.NewDecoder(r.Body)` без `http.MaxBytesReader`, а nginx допускает до 1 ГБ на API.

Нужно:

- единообразный middleware/helper для JSON body limits;
- разумные отдельные лимиты для JSON и multipart;
- корректный `413 Request Entity Too Large`;
- при возможности `DisallowUnknownFields` для строгих DTO;
- тесты на boundary, over-limit, malformed JSON и неизвестные поля;
- согласовать лимиты приложения и nginx.

### P1.3. Сделать refresh rotation атомарным

`backend/internal/modules/auth/service_refresh.go` сначала отдельно читает refresh,
затем вызывает rotation. В `repo_sessions.go` нет row lock и условного update по
`used_at IS NULL`; нет уникальности `rotated_from_id`. Два параллельных запроса могут
выдать два валидных дочерних refresh token.

Целевое исправление:

- одна транзакция;
- `SELECT ... FOR UPDATE` или атомарный conditional update;
- ровно один успешный child token;
- reuse detection и отзыв семейства/сессии в соответствии с ADR;
- конкурентный тест с двумя goroutine.

### P1.4. Закрыть file BOLA в submissions

`backend/internal/modules/submissions/service_files.go` проверяет доступ к переданному
`submissionID`, но затем получает произвольный `fileID` без проверки связи файла с этой
заявкой/испытанием/конкурсом.

Запрос к репозиторию должен связывать как минимум `file_id + submission_id` и далее
проверять contest access. Нужен тест, где пользователь с доступом к submission A
пытается скачать file B из другого tenant.

### P1.5. Переделать limiter входа участника

Проблемные места:

- `backend/internal/modules/eventparticipants/handlers.go` формирует ключ из event,
  первого `X-Forwarded-For` и User-Agent;
- заголовок можно подменять при текущей proxy-схеме;
- User-Agent легко менять;
- в `rate_limit.go` очистка map не выполняется для потока новых уникальных ключей;
- состояние локальное и сбрасывается при рестарте/не разделяется между инстансами.

Минимум: доверенная обработка proxy chain, bounded storage и гарантированная очистка.
Предпочтительно: shared limiter/БД и ключ, учитывающий event + нормализованный login,
с отдельным IP/network ограничением. Не логировать пароль.

### P1.6. Сделать feature flags реальными

Сейчас `backend/internal/config/features.go` и `/api/v1/config` фактически только
публикуют значения. Flags не блокируют API routes и почти не управляют UI.

Для каждого flag определить:

- backend boundary и ожидаемый `404`/`403`/`503`;
- скрытие/disable UI;
- тест enabled/disabled;
- поведение background jobs.

### P1.7. Включить S3 в readiness и управлять orphan objects

Backend стартует даже при ошибке инициализации S3, а readiness проверяет только БД.
Файловые функции при этом неработоспособны. Ошибки rollback/delete объекта часто
игнорируются, reconciler отсутствует.

Нужно определить обязательность S3 для readiness, добавить диагностический status,
метрики ошибок и периодическую очистку orphan objects. Для PDF/ZIP/PPTX и других
форматов добавить deep validation/AV policy; сейчас надёжные magic-byte проверки есть
в основном для raster images.

## 6. P2: продуктовый и эксплуатационный техдолг

1. **Данные RBAC:** из пяти конкурсов три имели `owner_user_id IS NULL`; обнаружены
   legacy role assignments с пустым `access_level`. Нужны read-only dry-run отчёт,
   согласованная data migration и post-migration invariants.
2. **Event permissions:** `event_staff_permissions` была пустой; полноценного API/UI
   назначения прав не найдено. Делегирование работает только через owner/MEGA bypass.
3. **Динамические формы:** submit проверяет required, но не полностью проверяет тип,
   option membership, min/max, regex, длину и неизвестные answer keys.
4. **Outbox/Telegram:** `outbox_deliveries`, `user_telegram` и subscriptions добавлены
   миграциями, но не включены в рабочий dispatcher. Отправка всё ещё идёт в общий чат.
   Отдельный PM2 worker является TODO-циклом; фактический dispatcher работает внутри API.
5. **Security headers:** HSTS/CSP и другие headers не применены глобально. Middleware
   используется лишь на части auth/participant routes. Вынести защиту на весь HTTP stack
   и/или nginx, проверить SPA и API.
6. **Runtime hardening:** live API сообщал `APP_ENV=development`; API и worker запущены
   от root. Перевести на отдельного непривилегированного пользователя.
7. **Observability:** env-шаблон обещает OTEL/Prometheus, но config/routes этого не
   реализуют. Нет `/metrics`; access logs малоинформативны.
8. **Deploy:** `deploy.sh` выполняет pull, migration, build и restart в live-каталоге;
   нет lock, atomic release, rollback и обязательных test gates.
9. **Миграции:** runner forward-only, без checksum и advisory lock. `Makefile` содержит
   устаревшие placeholders для migrate/seed.
10. **Frontend artifacts:** Vite использует `emptyOutDir: false`, поэтому старые hashed
    assets накапливаются. Нужна безопасная retention/atomic-release стратегия.
11. **Документация:** `docs/STATUS.md` и `frontend/README.md` местами утверждают, что
    admin UI работает на mocks, хотя production UI использует реальные API.
12. **OpenAPI:** описано заметно меньше операций, чем зарегистрировано в router;
    особенно неполны legacy auth/contest/challenge/submission/user endpoints.

## 7. Дефицит тестов

На момент аудита frontend test script и frontend test files не найдены. Backend tests
проходят, но покрытие ключевых модулей низкое:

| Модуль | Наблюдавшееся покрытие |
|---|---:|
| submissions | 1.7% |
| merch | 7.9% |
| eventtasks | 12.9% |
| lectures | 14.6% |
| points | 29.2% |
| eventparticipants | 37.3% |

У auth, middleware/RBAC, contests, useradmin и outbox тестов практически нет. Первый
набор integration tests должен проверять tenant isolation и отрицательные сценарии,
а не только happy path.

Минимальная security-матрица для каждого resource:

| Actor | Собственный tenant | Чужой tenant | Ожидание |
|---|---|---|---|
| MEGA_ADMIN | да | да | разрешено и аудируется |
| SUPER_ADMIN owner | да | нет | чужой tenant: 403/404 |
| ADMIN EDIT | только назначенный конкурс | нет | мутации контента разрешены |
| ADMIN VIEW | только назначенный конкурс | нет | мутации: 403 |
| CONTESTANT | только свои данные | нет | административные API: 403 |

## 8. Проверки перед передачей каждого изменения

Из корня репозитория:

```bash
git status --short --branch
cd backend && go test -count=1 ./...
cd backend && go test -race -count=1 ./...
cd backend && go vet ./...
cd frontend && npm run lint
cd frontend && npm run build
```

Для dependency/security PR дополнительно:

```bash
cd backend && govulncheck ./...
cd frontend && npm audit --omit=dev
```

Примечание: системный Go на момент аудита был 1.23.4, а `go.mod` требует Go 1.25.0.
Автозагруженный toolchain позволял запускать test/race/vet, но `go test -cover` частично
сломался из-за отсутствующего `covdata`. Это проблема окружения, её нельзя маскировать
изменением требований проекта без отдельного решения.

Также production build Vite предупреждал, что
`event-participant/auth-context.tsx` импортируется одновременно статически и
динамически, поэтому dynamic import не создаёт отдельный chunk. Это не блокер P0/P1,
но стоит исправить при работе над frontend bundle.

## 9. Рекомендуемый порядок выполнения

1. P0.1: reset/block + participant upsert exploit, с regression tests.
2. P0.2: tenant-safe user registry и role assignment, с полной RBAC-матрицей.
3. Подготовить отдельные инфраструктурные изменения для KillBot и backup; не применять
   их без разрешения.
4. P1.1: backend dependency upgrades.
5. P1.2–P1.5: body limits, atomic refresh, file ownership, rate limiter.
6. Data audit/migration для `owner_user_id` и `access_level`.
7. CI gates и frontend/backend integration/E2E tests.
8. Feature flags, S3 readiness/reconciliation, outbox/Telegram.
9. Atomic deploy, observability, docs и OpenAPI.

## 10. Формат отчёта следующего агента

Для каждой выполненной задачи сообщи:

- что было уязвимо или неполно;
- какие файлы изменены;
- какие инварианты доступа теперь соблюдаются;
- какие тесты добавлены и фактически запущены;
- нужны ли миграция, deploy, рестарт или внешнее изменение инфраструктуры;
- какие риски/решения остаются открытыми.

Не отмечай задачу выполненной только потому, что UI скрыл кнопку: security boundary
должна находиться в backend service/repository и подтверждаться отрицательными тестами.
