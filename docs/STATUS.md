# Состояние проекта

Живой документ для передачи контекста между сессиями. Обновлять при заметных изменениях.
Обзор этапов — в [README](../README.md). Спецификация — [SITE.md](../SITE.md).

_Последнее обновление: 2026-07-14 (Этап 5 закрыт)._

**Тулинг:** `admin create-user <login> <password> <role> [full_name] [--must-change]` (идемпотентен по login, роли SUPER_ADMIN|ADMIN|CONTESTANT в GLOBAL-scope). Тестовые юзеры засеяны: `superadmin` / `admin` / `contestant` (см. креды у команды). БД-том переживает рестарт, но users заполняем этой командой.

## Готово

### Этап 0 — инфраструктура
- Monorepo: `backend/` (Go), `frontend/` (React/Vite/TS), `infra/`, `docs/`.
- Docker Compose: postgres (**порт 5433**, не 5432 — см. память server-infra), redis, minio.
- Config, Makefile, линтеры, базовый OpenAPI (`backend/api/openapi.yaml`), envelope-ответы.

### Этап 1 — авторизация (бэкенд + фронт)
Бэкенд (`backend/internal/modules/auth/`):
- login, refresh с ротацией, logout / logout-all, `/me`, change-password, список/отзыв сессий.
- RBAC-middleware, аудит (`modules/audit`), CLI `cmd/admin` (migrate + create-superadmin).
- Блокировка после 5 неудач на 15 мин, min пароль 10 символов.

Фронт (подключён к реальному API, **моки убраны из auth-флоу**):
- `shared/api/client.ts` — in-memory access-токен + `Authorization` + одноразовый refresh на 401.
- `entities/auth/` — api, типы, `roles.ts` (роль→landing), `AuthProvider` (silent-refresh `/me` на старте).
- `app/guards.tsx` — `RequireAuth` (форсит смену пароля), `RequireGuest`.
- Страницы: login (реальная мутация), change-password, forgot-password (инфо-заглушка).

### Админ-панель (фронт на реальном API, 2026-07-14)
Роль-адаптивная админка (`pages/admin/`) подключена к бэкенду Этапа 2, **моки убраны** (`shared/api/mock/admin-data.ts` удалён):
- `AdminLayout` — sidebar с фильтрацией пунктов по роли (`admin-nav.ts`) + хедер с бейджем роли.
- Экраны: `/admin` (дашборд-метрики, разные для ADMIN/SUPER_ADMIN), `/admin/contests` (список + диалог создания для SUPER_ADMIN), `/admin/contests/:id` (карточка + кнопки переходов статуса + таблица конкурсантов с add/remove/reset/block + import/export CSV), `/admin/users` (реестр с пагинацией/поиском/фильтрами в URL, диалог создания, reset/block, гард `RequireRole['SUPER_ADMIN']`).
- Scope теперь **серверный**: `GET /admin/contests` уже отфильтрован по доступу (ADMIN видит только назначенные). `admin_logins` из фронта убран.
- Сущности `entities/contest|contestant|user` — реальные `api.ts` + `queries.ts` (useQuery/useMutation с инвалидацией). Общие действия reset/block/unblock — `entities/user/admin-actions.ts`. `delay`-моки удалены.
- API-клиент (`shared/api/client.ts`): `apiRequestFull` (data+meta для пагинации), `apiRequestText`/`postText` (CSV import/export вне JSON-envelope).
- Временный пароль (create user/contestant, reset) показывается в `TempPasswordNote` с копированием.
- `RequireRole` guard в `app/guards.tsx`.
- ⚠️ Submission-метрики в таблице конкурсантов убраны — их даёт Этап 3 (испытания/формы). Поле `challenges_count` у конкурса тоже отложено до Этапа 3.

**Достройка UI Этапа 2 (2026-07-14):**
- **Редактирование конкурса**: `pages/admin/edit-contest-dialog.tsx` (name, description, start_at/end_at через `datetime-local`, timezone) → `useUpdateContest` (PATCH). Кнопка «Редактировать» на `contest-detail-page` (скрыта для ARCHIVED, доступна всем с доступом — бэкенд `ensureAccess`). Хелперы `isoToLocalInput`/`localInputToIso` в `shared/lib/format.ts`.
- **Управление ролями из UI**: `pages/admin/manage-roles-dialog.tsx` — список назначений со scope (Глобально / Конкурс: имя), снятие (DELETE), форма назначения (роль + область: глобально или конкретный конкурс). Действие (иконка щита) в `users-table`. Хуки `useAdminUser`/`useAssignRole`/`useRemoveRole` в `entities/user/queries.ts`, инвалидируют `['admin','user',id]` + `['admin','users']`.

### Этап 2 — конкурсы и участники (бэкенд, срез, 2026-07-14)
Миграция `0005_contests.up.sql`: таблицы `contests`, `contest_participants` (SITE.md §21.6–21.7).
- `modules/contests/` — CRUD конкурса, переходы статусов publish/finish/archive (матрица допустимых переходов), scoped-list. Участники: список, добавление конкурсанта (создаёт user + роль CONTESTANT scope=CONTEST + participant в одной транзакции, возвращает временный пароль), удаление (soft, `left_at`).
- `modules/useradmin/` — reset-password (временный пароль + must_change + отзыв сессий), block/unblock (при block — отзыв сессий).
- **Scoped-RBAC в сервис-слое**: `HasContestAccess` = SUPER_ADMIN (global) ∨ ADMIN scoped на конкретный конкурс. `RequireRole('ADMIN')` — coarse-гейт на роутах `/api/v1/admin/*`.
- `security.GenerateTempPassword()` (14 симв., без визуально похожих). Аудит на всех мутациях.
- Роуты `/api/v1/admin/contests*` + `/admin/users/:id/{reset-password,block,unblock}`.

**Достройка Этапа 2 (2026-07-14, бэкенд):**
- **Roles API** (`useradmin/service_roles.go`): `POST /admin/users/:id/roles` (assign, scope GLOBAL|CONTEST, идемпотентно), `DELETE /admin/users/:id/roles?role=&scope_type=&scope_id=`. Заменил psql-костыль в смоуке.
- **User CRUD/список** (`useradmin/service_users.go`, `repo.go`, `repo_write.go`): `GET /admin/users` (серверная пагинация limit/offset + поиск по login/ФИО + фильтры role/status, роли догружаются без N+1), `POST /admin/users` (создать + опц. стартовая роль, временный пароль), `GET /admin/users/:id` (с ролями), `PATCH /admin/users/:id` (профиль). **Только SUPER_ADMIN** (подгруппа с `RequireRole('SUPER_ADMIN')`).
- **Import/export конкурсантов** (`contests/service_import.go`): `POST .../contestants/import` (CSV в теле → построчный AddContestant, сводка created/failed/rows), `GET .../contestants/export` (CSV-вложение, экранирование RFC 4180). Скелет: синхронно, без файлов/фона (полноценный экспорт — Этап 8).
- Вне охвата: soft-delete юзера (DELETE), назначение нескольких ролей батчем.

### Этап 3 — испытания и конструктор форм (бэкенд + фронт, 2026-07-14)
Миграция `0007_challenges.up.sql`: `contest_challenges`, `challenge_fields` (soft-delete + версионные окна `schema_version_from/to`), `challenge_schema_versions` (снапшоты) — SITE.md §21.8–21.10.

Бэкенд (`modules/challenges/`):
- CRUD испытания (DRAFT при создании, slug с транслитерацией кириллицы), матрица статусов DRAFT↔PUBLISHED↔CLOSED→ARCHIVED, дубликат (мета+поля).
- Поля: create/update/delete (soft) / reorder (в транзакции), валидация типа по `ValidFieldTypes` (v1-набор из SITE.md §11.1; RICH_TEXT/TIME/DATETIME/MULTISELECT — долг).
- **Версионирование**: при публикации — снапшот схемы; правка полей опубликованного испытания бампит `current_schema_version` + новый снапшот (SITE.md §11.4).
- **Доступ**: админ — через `contests.HasContestAccess` (SUPER_ADMIN ∨ ADMIN scoped); контестант-чтение — через участие (`IsParticipant`), видит только PUBLISHED. Роуты `/admin/contests/:id/challenges`, `/admin/challenges/:cid/*`, читающие `/contests/:id/challenges` + `/challenges/:cid` (авторизован, роль не важна).

Фронт:
- `entities/challenge/admin-{types,api,queries}.ts` — отдельно от контестант-мок-типов (`types.ts` не тронут).
- Конструктор `/admin/challenges/:challengeId` (`challenge-builder-page.tsx`): мета+статусы, вкладки Конструктор/Превью, список полей с ↑↓/edit/delete (`fields-list.tsx`), редактор поля с настройками по типу — options для SELECT/RADIO, extensions/multiple для FILE_GROUP (`field-editor-dialog.tsx`, `options-editor.tsx`), превью переиспользует контестант-`FieldRenderer` в disabled-режиме (`challenge-preview.tsx`).
- Список испытаний на странице конкурса (`challenges-section.tsx`) + диалог создания (`create-challenge-dialog.tsx`).

### Этап 4 — подача ответов (submissions, бэкенд + фронт, 2026-07-14)
Миграция `0008_submissions.up.sql`: `files` (метаданные объектов MinIO), `submissions` (одна работа на пару испытание+конкурсант, `answers_json`, статус DRAFT/SUBMITTED/LOCKED, счётчики версий/ревизий), `submission_revisions` (immutable-снапшоты схемы+ответов+файлов, checksum), `submission_files` — SITE.md §21.11–21.14.

Бэкенд (`modules/submissions/`):
- Черновик: `GET /challenges/:cid/submission` создаёт/открывает работу (проставляет `first_opened_at`), `PUT .../draft` сохраняет ответы без ревизии.
- Отправка `POST .../submit`: валидация обязательных полей (по типам, FILE_GROUP → наличие файла), immutable-ревизия (SUBMIT/RESUBMIT), checksum sha256, `version++` при повторной подаче. Первая — SUBMIT, далее RESUBMIT.
- Окно подачи: проверка PUBLISHED + `open_at`/`deadline_at`; поздняя подача только при `settings.allow_late_submission`; LOCKED → 409.
- Файлы: `POST .../files` (multipart → MinIO через API, ключ `submissions/<contest>/<ch>/<sub>/<reqid>-<safe>`, валидация расширения/размера по настройкам поля), `DELETE .../files/:id` (soft + удаление объекта, только владелец/открытое окно). Откат объекта при сбое БД.
- Админ: `GET /admin/challenges/:cid/submissions` (таблица дирекции §7.6 — ФИО/логин/организация из `users`, число файлов, фильтр по статусу, `meta.total`), `GET /admin/submissions/:id` (ответы+файлы+история ревизий), `GET .../files/:id` (302 на presigned-URL). Доступ — `HasContestAccess`.
- Adapter `challenge_adapter.go` связывает submissions с challenges/contests без обратных зависимостей. Presigner из `storage` подключается в `deps.go` (nil-safe, если MinIO недоступен).
- Новый эндпоинт `GET /my/contests` (`contests.MyContests`) — конкурсы, где пользователь активный участник (для кабинета). `ListForContestant` в challenges добавляет `my_submission_status` в список испытаний.
- **Долг (отложено осознанно)**: `outbox_event submission.submitted` для Telegram-уведомления (TODO в `service_submit.go`) — Этап 5. Правила валидации min/max/regex по полю — как в Этапе 3.

Фронт:
- Контестант **снят с моков** (`shared/api/mock/data.ts` больше не импортируется): `entities/challenge/{api,queries}.ts` + `entities/submission/{api,queries}.ts` — реальные вызовы. Дашборд берёт конкурс из `/my/contests`, статусы работ и метрики (черновики/отправлено/просрочено) — из `my_submission_status`.
- Форма `challenge-form-page.tsx` + `use-submission-form.ts`: автосейв черновика (debounce 800мс), загрузка файлов multipart с оптимистичным UI, submit с подтверждением и валидацией, read-only при LOCKED, «Обновить отправку» для resubmit.
- Админ: вкладка «Ответы» в конструкторе (`submissions-section.tsx` — таблица с фильтром по статусу; `submission-detail-dialog.tsx` — ответы, файлы со скачиванием, история ревизий). API `entities/submission/admin-{api,queries}.ts`.
- Клиент: `apiPostForm` (multipart) добавлен в `shared/api/client.ts`.

### Этап 5 — уведомления (transactional outbox + Telegram, бэкенд, 2026-07-14)
Миграция `0009_outbox.up.sql`: `outbox_events` (event_type, aggregate, payload JSONB, статус PENDING/SENT/DEAD, attempts, available_at для backoff, locked_at/by) — SITE.md §21.16.

Бэкенд (`modules/outbox/`):
- **Транзакционный outbox**: `submissions.Submit` пишет `outbox_event` в той же транзакции, что и ревизия (`repo_write.go`, поля `OutboxEventType`/`OutboxPayload` в `SubmitParams`). Тип — `submission.submitted` (первая) / `submission.resubmitted` (обновление). Черновик события не создаёт (SITE.md §15). Payload минимальный (`submission_id`,`revision`,`action`) — читаемые поля дорезолвит диспетчер.
- **Диспетчер** (`dispatcher.go`) — горутина в API-процессе (по договорённости; `cmd/worker` оставлен тонким скелетом). Поллинг `ClaimPending` через `FOR UPDATE SKIP LOCKED`, exponential backoff (база 10с, до 6 попыток → DEAD), `ReleaseStale` возвращает зависшие блокировки. Стартует из `cmd/api/main.go` через `App.StartBackground`, останавливается по ctx на shutdown.
- **Telegram** (`telegram.go`) — Bot API `sendMessage`, parse_mode HTML. `Enabled()` = флаг И заданы токен+chat. Выключен → события копятся PENDING, не теряются. Шаблон сообщения (`format.go`) по §15: «📥 Новая отправка / 🔄 Обновление отправки», конкурс/испытание/ФИО/организация/ревизия/дата/файлы + ссылка на `/admin/submissions/:id`.
- **Резолвер** (`repo_resolve.go`) — один JOIN submissions→challenge→contest→users для человекочитаемых полей; отсутствие агрегата → `ErrGone` (DEAD без ретраев).
- **`challenges_count` у конкурса** (долг Этапа 2 закрыт): подзапрос non-archived испытаний во всех трёх запросах `contests/repo.go`, поле в JSON + фронт (`AdminContest.challenges_count`, карточка на странице конкурса).
- **Конфиг**: `TELEGRAM_BOT_TOKEN`/`_DEFAULT_CHAT_ID`/`_DEFAULT_THREAD_ID`/`_NOTIFICATIONS_ENABLED` (уже в `.env`-шаблоне, §22). ⚠️ Токен/chat заполняются вручную в `.env` (не в git) — до этого доставка выключена, события копятся.
- Нумерация: в SITE.md §47 это «Этап 6. Telegram и worker», у нас — Этап 5 (по нашей декомпозиции).

## Не проверено / долги
- ✅ Сквозной smoke авторизации пройден (2026-07-14): `backend/scripts/smoke_auth.sh` через https://eazytech.ru — login 3 ролями, /me+роли, refresh по cookie, 401-кейсы. PASS=14. Гонять как регресс (refresh требует https-хост, cookie Secure+Domain).
- ✅ Сквозной smoke Этапа 2 пройден (2026-07-14): `backend/scripts/smoke_contests.sh` — create/scope-invisibility/403 чужого, roles assign/remove через API, ADMIN→/users 403, add+temp-пароль, login конкурсанта, publish/finish + 409, reset/block, import/export CSV, user CRUD+поиск. PASS=33. Требует psql (docker `slc-postgres`) только для генерации уникальных логинов и уборки.
- ✅ Сквозной smoke Этапа 3 пройден (2026-07-14): `backend/scripts/smoke_challenges.sh` — CRUD испытания, дедлайн, валидация типа поля (400) и дубля ключа (409), reorder, schema-preview, publish+снапшот, bump версии при правке PUBLISHED, видимость PUBLISHED-only конкурсанту + 404 на DRAFT, матрица переходов + 409, чужой ADMIN → 403. PASS=30. Регресс contests тоже зелёный (PASS=33).
- ✅ Сквозной smoke Этапа 4 пройден (2026-07-14): `backend/scripts/smoke_submissions.sh` — открытие черновика, сохранение, submit без обязательного → 400, загрузка файла + запрет расширения, submit → ревизия 1, resubmit → ревизия 2 + version bump, удаление файла, админ-таблица (ФИО/организация/фильтр статуса), карточка + история ревизий, просроченный дедлайн → 409 DEADLINE_PASSED, не-участник → 403. **PASS=38**. Регресс: contests=33, challenges=30 — все зелёные (итого 101).
- ✅ Сквозной smoke Этапа 5 пройден (2026-07-14): `backend/scripts/smoke_outbox.sh` — черновик НЕ создаёт событие, submit → PENDING `submission.submitted` (payload revision=1), resubmit (тот же `/submit`) → `submission.resubmitted` (revision=2), резолвер собирает конкурс/испытание/ФИО/организацию. **PASS=11**. Секция доставки в Telegram активируется, если `TELEGRAM_NOTIFICATIONS_ENABLED=true`. Полный регресс: auth=14, contests=33, challenges=30, submissions=38, outbox=11 (итого 126).
- ⚠️ **Миграции применяются вручную, НЕ на старте API**: `./bin/admin migrate` (env из корневого `.env`, юзер БД `slc_app`). При деплое Этапа 3 применены `0006_contest_image` (была пропущена — из-за неё `GET /admin/contests` отдавал 500!) + `0007_challenges`. Этап 4 — `0008_submissions`. Этап 5 — `0009_outbox`. Правило: после `git pull` с новыми миграциями — `admin migrate` перед рестартом.
- ⚠️ **Telegram-доставка требует ручной настройки `.env`**: `TELEGRAM_BOT_TOKEN` + `TELEGRAM_DEFAULT_CHAT_ID` (через @BotFather), затем `TELEGRAM_NOTIFICATIONS_ENABLED=true` и рестарт API. До этого события копятся как PENDING (не теряются). Секреты — только в `.env` (640/www-data), не в git/переписке.
- ⚠️ Флэки тестовый аккаунт `contestant`: при рассинхроне (must_change=f, пароль не совпадает) пересоздать `admin create-user contestant 'Contestant!2026' CONTESTANT '...' --must-change`. Не связано с кодом login.
- **Админ-фронт всё ещё на моках** — теперь есть реальное API конкурсов/участников (Этап 2), но фронт к нему не подключён. Следующий заход: заменить моки в `entities/contest|contestant` на реальные вызовы.
- **Админ-панель целиком на моках** — реального API под конкурсы/конкурсантов/пользователей нет (Этап 2). Действия reset/block/добавить — только тосты-заглушки, CRUD-форм и деталей испытаний/submissions нет (по договорённости — ядро).
- **Frontend lint нежизнеспособен**: скрипт `eslint .` есть, но конфига нет (eslint 9 требует flat-config). Проверка кода — только через `tsc -b`. Долг тулинга.
- Forgot-password — без self-service (сброс делает админ, SITE.md §7).
- Bundle > 500 kB — code-splitting отложен.

## Следующий шаг
Этап 5 (уведомления: outbox + Telegram) закрыт: миграция `0009_outbox`, транзакционный outbox в submit, Telegram-клиент, диспетчер-горутина в API (`FOR UPDATE SKIP LOCKED`, exponential backoff, DEAD после лимита), `challenges_count` у конкурса возвращён. Смоук PASS=11, регресс 126, задеплоено. Дальше:
1. **Включить Telegram** — создать бота (@BotFather), вписать `TELEGRAM_BOT_TOKEN`/`TELEGRAM_DEFAULT_CHAT_ID` в `.env`, `TELEGRAM_NOTIFICATIONS_ENABLED=true`, рестарт API. Затем прогнать `smoke_outbox.sh` — увидеть реальную доставку (SENT) и сообщение в чате.
2. **Ручной прогон в браузере** — сквозной сценарий Этапов 3–5: админ создаёт испытание и форму, конкурсант отправляет ответ с файлом, дирекция видит его во вкладке «Ответы», в Telegram приходит уведомление.
3. **Этап 6 — файлы/справка или hardening** (SITE.md §47): следующий крупный блок по дорожной карте. Определиться на старте сессии.
4. Долг Этапа 3–4: недостающие типы полей (RICH_TEXT/TIME/DATETIME/MULTISELECT), `open_at`/`close_at` в UI, правила валидации (min/max/regex) в редакторе поля и на submit. Долг Этапа 5: напоминания о дедлайне и уведомление о критической ошибке файла (SITE.md §15) — событий пока нет.

## Как запускать
```bash
make up            # postgres/redis/minio (порты на 127.0.0.1)
make api-run       # API :8080
make frontend-dev  # SPA :5173
```
Бэкенд: `make build` → `systemctl restart eazytech-api`. Фронт: `npm run build` в frontend/.
