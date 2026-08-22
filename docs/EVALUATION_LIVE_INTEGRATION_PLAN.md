# Integration plan: live-оценивание конкурсных испытаний

Источник ТЗ: [`cursor_competition_evaluation_live_spec.md`](../cursor_competition_evaluation_live_spec.md).  
Дата плана: **2026-08-21**. Код модуля ещё не писать, пока не начат Phase 2.

Новый функционал — слой поверх существующих `contests` / `contest_challenges` / сдачи файлов.  
Параллельную модель конкурса, испытания, пользователя или конкурсанта не создавать.

---

## Решения, зафиксированные этим планом

| Тема | Решение |
|---|---|
| Модуль | `backend/internal/modules/evaluation/` внутри текущего монолита (ADR 0001) |
| Привязка | `evaluation_schemes.challenge_id → contest_challenges.id` |
| Конкурсант в performance | `contestant_user_id` + `challenge_id`, как в `submissions` |
| COMPETITION_ADMIN | не новая глобальная роль: `HasContestAccess` (MEGA / SUPER / ADMIN EDIT) |
| JURY | новая роль `roles.code = JURY`, scope `CONTEST`; логин через существующий JWT |
| TRIAL_OPERATOR / EXPERT / FOCUS_GROUP_ADMIN | таблица `evaluation_staff_assignments`, не `event_staff_permissions` |
| Фокус-группа (аудитория) | `event_participants` мероприятия, не `users` жюри |
| Audit | существующий `audit_logs`; отдельную `competition_audit_logs` не заводить |
| История баллов | отдельная `score_value_history` (частые мутации) |
| Realtime | **нет** в проекте. Команды — HTTP. Live-контекст — **SSE**. Scores не хранить в сокете |
| Official result | только backend calculators |
| Feature flag | существующий `FEATURE_JURY` / `features.jury` |
| API | `/api/v1/...`, envelope и error codes как сейчас |
| Сдача файлов | `submissions` не трогать по смыслу; маршруты не ломать |

---

## EXISTING ENTITIES TO REUSE

| ТЗ | Проект |
|---|---|
| Competition | `contests` |
| Contest Trial | `contest_challenges` (`backend/internal/modules/challenges`) |
| Contestant / participation | `contest_participants` + `users`; тип `CONTESTANT` уже есть, тип `JURY` в participant_type есть, но **роли жюри в `roles` нет** |
| User | `users` |
| Permissions | `user_roles` (GLOBAL / CONTEST), `event_staff_permissions` — только event-контур |
| Auth | JWT access + refresh cookie (`modules/auth`) |
| Сдача файлов / ТЗ испытания | `submissions`, `files`, `challenge_fields`, `challenge_schema_versions` |
| Audit | `audit_logs` (`modules/audit`) |
| Feature flags | `config.Features.Jury`, `GET /api/v1/config` |
| Event participant | `event_participants` — только focus group, не жюри |
| Idempotency pattern | как `points_ledger.idempotency_key` / merch `Idempotency-Key` |
| Outbox | `outbox_events` — не для scores; при необходимости later для публикации |

Не переиспользовать:

- STAFF мероприятия как оператора испытания;
- `event_tasks` как профтест (другой контур);
- participant-auth (VK/Telegram) как вход жюри.

---

## EXISTING SERVICES TO REUSE

- `contests.HasContestAccess` / `EnsureAccess` — гейт админ-консоли live.
- `challenges` — чтение испытания, статус PUBLISHED/CLOSED; evaluation не подменяет конструктор полей.
- `auth` + `middleware.Authenticator` / `RequireRole` — жюри добавляется в allow-list отдельных групп.
- `audit.Service.Log` / `LogEntry` — live-команды, external scores, window extend, publish.
- `httpserver` envelope + стабильные `error.code`.
- Frontend: `AuthProvider`, `RequireAuth` / `RequireRole`, `apiRequest` / `fetchWithRefresh`, FSD (`entities` / `features` / `pages`).
- TanStack Query — серверное состояние live snapshot; **не** как store оценок жюри.

---

## FILES TO MODIFY

### Backend

- `backend/internal/app/deps.go` — сборка evaluation-сервиса.
- `backend/internal/app/router.go` — `/api/v1/admin/contests/{id}/challenges/{id}/evaluation|live|...`, `/api/v1/jury/...`.
- `backend/internal/middleware/rbac.go` и `useradmin/types.go` — код роли `JURY`.
- `backend/internal/modules/auth` — `/me` отдаёт роль `JURY` (JWT claims как у остальных).
- `backend/api/openapi.yaml`.
- `backend/internal/config/features.go` — флаг уже есть; при необходимости отдельные подфлаги не вводить на старте.
- `.env.example` — комментарий, что `FEATURE_JURY=true` включает модуль.
- nginx (`/etc/nginx/sites-available/eazytech`, **отдельное разрешение**): `proxy_buffering off` для SSE.

### Frontend

- `frontend/src/entities/auth/types.ts` — `RoleCode` += `'JURY'`.
- `frontend/src/entities/auth/roles.ts` — landing `/jury`.
- `frontend/src/app/router.tsx` — маршруты `/jury/*` и admin live.
- `frontend/src/pages/admin/admin-nav.ts` — пункт live при `features.jury`.
- `frontend/src/app/guards.tsx` — `RequireRole(['JURY'])` (или смешанный ADMIN+JURY не смешивать в одном кабинете).
- `frontend/src/shared/config/use-app-config.ts` — уже есть `jury`.

### Документация

- `docs/STATUS.md`, `docs/architecture.md` — после первого рабочего среза, не сейчас.
- ADR: после выбора SSE зафиксировать `docs/ADR/0006-evaluation-realtime.md`.

---

## FILES TO CREATE

### Backend (`modules/evaluation/`)

```text
backend/internal/modules/evaluation/
  types.go
  errors.go
  repo.go
  repo_write.go
  service.go
  service_live.go
  service_scores.go
  service_staff.go
  handlers.go
  handlers_live.go
  handlers_jury.go
  sse.go
  calc/
    criteria.go
    numeric.go
    questions.go
    lives.go
    head_to_head.go
    composite.go
```

Тесты рядом: `service_*_test.go`, smoke `backend/scripts/smoke_evaluation.sh`.

### Frontend

```text
frontend/src/entities/evaluation/     types, api, queries
frontend/src/features/jury-score-sync/  IndexedDB + очередь + indicator
frontend/src/pages/admin/evaluation-scheme-page.tsx
frontend/src/pages/admin/evaluation-live-page.tsx
frontend/src/pages/admin/evaluation-results-page.tsx
frontend/src/pages/jury/jury-layout.tsx
frontend/src/pages/jury/jury-trial-page.tsx
frontend/src/pages/jury/jury-my-scores-page.tsx
```

---

## NEW ENTITIES

Имена таблиц — snake_plural, UUID, `TIMESTAMPTZ`, как в текущих миграциях.

| Таблица | Назначение |
|---|---|
| `evaluation_staff_assignments` | JURY / TRIAL_OPERATOR / EXPERT / FOCUS_GROUP_ADMIN на contest или challenge |
| `evaluation_schemes` | тип двигателя на `contest_challenges` |
| `evaluation_scheme_versions` | snapshot конфигурации после начала |
| `evaluation_criterion_groups` | группы индикаторов (мастер-класс) |
| `evaluation_criteria` | критерии 1–10 и т.п. |
| `criterion_scale_bands` | подсказки диапазонов, не хардкод во фронте |
| `evaluation_components` | JURY / EXTERNAL / FOCUS_GROUP / … |
| `evaluation_audience_windows` | окно редактирования жюри / фокус-группы |
| `evaluation_sessions` | одно live-состояние на испытание, поле `revision` |
| `performances` | выступление конкурсанта |
| `performance_phase_templates` | шаблоны фаз (8+5, 3+5, …) |
| `score_sheets` | UNIQUE(performance_id, evaluator_user_id) |
| `score_values` | UNIQUE(score_sheet_id, criterion_id), `revision` |
| `score_value_history` | спорные ситуации |
| `external_evaluation_results` | UNIQUE(challenge_id, contestant_user_id, component_id) |
| `life_events` | append-only Δ жизней «2 к 1» |
| `competition_rounds` | QUALIFICATION / FINAL |
| `competition_matches` | пара A/B |
| `match_votes` | UNIQUE(match_id, jury_user_id) |
| `focus_group_assignments` | связь с `event_participants` |
| `evaluation_results` | official raw/normalized + `calculation_version` |
| `evaluation_result_snapshots` | воспроизводимость публикации |
| `ranking_results` | место отдельно от балла |
| `evaluation_idempotency_keys` | mutationId / commandId |

Presence жюри **не** писать в БД постоянно: память процесса + SSE heartbeat; после reconnect — HTTP snapshot.

`ScoringCorridor` — либо таблица `scoring_corridors`, либо JSON в scheme.settings до Phase 6; backend всё равно валидирует min/max.

---

## MIGRATIONS

Следующий номер после `0022_participant_social_ids`: **`0023`**. Дробить по фазам, expand-only.

| Миграция | Содержание |
|---|---|
| `0023_evaluation_role_and_staff` | `INSERT roles (JURY)`; `evaluation_staff_assignments` |
| `0024_evaluation_schemes` | schemes, versions, groups, criteria, bands, components, windows |
| `0025_evaluation_live` | sessions, performances, phase templates |
| `0026_evaluation_scores` | sheets, values, history, idempotency |
| `0027_evaluation_mechanics` | external results, life_events, rounds, matches, votes, focus_group |
| `0028_evaluation_results` | results, snapshots, ranking |

Индексы — по ТЗ §66. FK на `contest_challenges`, `contests`, `users`.  
Не удалять и не менять `submissions` / `files`.

---

## REALTIME PLAN

Сейчас realtime-слоя нет (поиск WebSocket/SSE/Socket.IO по репо — пусто).

```text
Admin/Jury HTTP command
        → UPDATE evaluation_sessions.revision
        → audit_logs
        → SSE event (компактный)

Scores / votes / lives
        → только HTTP + идемпотентность
        → SSE максимум «window/state changed», не значения чужих баллов
```

Канал: `GET /api/v1/jury/challenges/{id}/live/stream` и admin-аналог, `text/event-stream`.

События (имена адаптировать к модулю):

- `session.updated`
- `currentContestant.changed`
- `phase.changed`
- `performance.started` / `finished`
- `match.*`
- `twoToOne.lifeChanged` (без PII сверх id)
- `scoring.windowOpened` / `closed`
- `results.published`

Инвариант ТЗ §67: если SSE упал, админ-команды и sync оценок идут HTTP. После reconnect клиент делает `GET .../live` (полный snapshot с `sessionRevision` и `serverTime`), а не replay всех событий.

Таймер: хранить `phase_started_at`, `phase_duration_seconds`, `paused_at`, `accumulated_pause_seconds`. Клиент считает remaining через `serverTime - clientTime`.

Nginx (когда разрешат):

```nginx
location /api/v1/ {
    proxy_buffering off;          # или точечно на /live/stream
    proxy_read_timeout 3600s;
}
```

KillBot на apex по-прежнему может резать long-poll; smoke и жюри в проде вести через `www` / origin, как API.

---

## SECURITY RISKS

1. Жюри меняет чужой `score_sheet` через подмену `evaluatorUserId` / `performanceId`.  
   Защита: sheet всегда `evaluator = auth.user`; performance ∈ challenge ∈ contest assignment.
2. Конкурсант или STAFF мероприятия вызывает jury API.  
   Гейт: роль `JURY` + `evaluation_staff_assignments` на этот contest/challenge.
3. `TRIAL_OPERATOR` публикует результаты или правит scheme.  
   Матрица: operator — только lives (и явно перечисленные команды испытания).
4. Фокус-группа открыта всем participant'ам события.  
   Только `focus_group_assignments.active`.
5. Broadcast индивидуальных баллов по SSE.  
   Админ live видит presence/sync, не чужие scores (ТЗ §58).
6. Правка scheme после первых scores.  
   Versioning; несовместимые изменения — ошибка, не silent.
7. `FEATURE_JURY=false` — маршруты 404, таблиц достаточно создать заранее (флаг режет UI/API, не миграции).
8. CORS / cookie: jury SPA на том же origin `eazytech.ru`; новый subdomain не вводить.

Каждый mutating handler: authn → assignment → permission → принадлежность entity к trial.

Новые error codes (стабильные, в OpenAPI):  
`EVALUATION_REVISION_CONFLICT`, `EVALUATION_WINDOW_CLOSED`, `EVALUATION_CORRIDOR`, `EVALUATION_NOT_ASSIGNED`, `EVALUATION_SCHEME_LOCKED`.

---

## CONCURRENCY RISKS

| Риск | Защита |
|---|---|
| Две админ-консоли, смена текущего конкурсанта | `evaluation_sessions.revision` + `baseRevision` → `409 EVALUATION_REVISION_CONFLICT` |
| Два устройства одного жюри, оценка 8 и 9 | `mutationId` идемпотентен; `baseRevision` на `score_values`; pending local не затирается snapshot (ТЗ §22) |
| Двойной tap «−♥» | `commandId` UNIQUE; `LifeEvent` append-only, не UPDATE счётчика |
| Повтор vote | UNIQUE(match_id, jury_user_id) + mutationId |
| Гонка close voting vs последний vote | close — транзакция: lock match, snapshot votes, winner, revision |
| Recalc во время sync | recalc после ACK батча; `calculation_version` на результате |
| SSE fan-out vs один API-процесс | presence in-memory ок для одного инстанса PM2; при втором инстансе — Redis/pg NOTIFY (не в Phase 3) |

Не использовать клиентский timestamp как истину конфликта.

---

## API (логическая схема → `/api/v1`)

Префикс админки:

```text
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/evaluation
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/live
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/live/start|pause|finish
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/live/current-contestant
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/live/phase
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/external-results
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/two-to-one/...
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/matches
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/results
/api/v1/admin/contests/{contestId}/challenges/{challengeId}/results/recalculate|publish|hide
```

Жюри:

```text
GET  /api/v1/jury/contests
GET  /api/v1/jury/contests/{contestId}/challenges
GET  /api/v1/jury/challenges/{challengeId}/live
GET  /api/v1/jury/challenges/{challengeId}/live/stream
GET  /api/v1/jury/performances/{performanceId}/scores
POST /api/v1/jury/scores/sync
PUT  /api/v1/jury/matches/{matchId}/vote
GET  /api/v1/jury/challenges/{challengeId}/my-scores
```

Sync payload — как ТЗ §71 (`clientId`, `mutations[]` с `mutationId`, `baseRevision`, `localSequence`).

Frontend routes:

```text
/admin/contests/:contestId/challenges/:challengeId/evaluation
/admin/contests/:contestId/challenges/:challengeId/live
/admin/contests/:contestId/challenges/:challengeId/results
/jury
/jury/contests/:contestId/challenges/:challengeId
/jury/contests/:contestId/challenges/:challengeId/my-scores
```

---

## IMPLEMENTATION ORDER

Совпадает с ТЗ §96. После каждого этапа: `go test ./...`, frontend `npm run build` / lint, точечный smoke.

### Phase 1 — исследование

Сделано этим документом. ТЗ в корне репозитория. Массовую генерацию файлов не начинать до Phase 2.

### Phase 2 — scheme + criteria + staff

Миграции `0023`–`0024`. CRUD scheme/criteria/bands/components. Назначение JURY на конкурс. Admin UI конструктора схемы. Flag `FEATURE_JURY`.

### Phase 3 — session + performance + admin live + SSE

Миграция `0025`. Команды current contestant / phase / start / finish. `revision`. Admin console. Snapshot + SSE. Presence in-memory.

### Phase 4 — Jury Workspace + followLive

Маршруты `/jury`. Автоследование за live contestant. Ручной уход + баннер «сейчас выступает X». Без кнопки Submit (поля могут быть ещё read-only до Phase 5, но UX без submit).

### Phase 5 — IndexedDB autosave — **сделано и задеплоено 2026-08-22**

`JuryScoreSyncService`: мгновенный local write, flush ~1 с, `visibilitychange` / `pagehide`, retry, ACK, conflict replay. Индикатор sync. Тесты ТЗ §86–§88.

Реализованы durable IndexedDB queue, coalescing по слоту, exponential backoff,
reconnect/pagehide flush, optimistic `base_revision`, durable server receipts и
автоматический rebase `409`. Chromium E2E покрывает offline → reconnect → reload и
конфликт двух ревизий; PostgreSQL integration покрывает idempotent retry, конфликт и
конкурентные записи с корректным total.

### Phase 6 — «Автопортрет» E2E

`CRITERIA_SCORING`, фазы 8+5 мин, 7 критериев из ТЗ, corridor configurable, seed/reference scheme. Референс для остальных criteria-trial.

### Phase 7 — external + композиты

«Проектирование», «Информационная работа». Ручной ввод заочки админом, audit, recalc.

### Phase 8 — «2 к 1»

`life_events`, TRIAL_OPERATOR UI, рейтинг по выбыванию, restore = +1 event.

### Phase 9 — «Управленческий поединок»

Rounds, matches, votes, скрытие чужих голосов до close, tie configurable.

### Phase 10 — блиц / профтест / правовое

`QUESTION_SCORING`, `NUMERIC_RESULT`; правовое — scheme type настраивается, шкалу не хардкодить.

### Phase 11 — мастер-класс + focus group

15 индикаторов, `PERFORMANCE_WINDOW` + grace, продление с audit, focus group assignments.

### Phase 12 — hardening

History, snapshots, observability, load/offline, permission tests ТЗ §94. Не ломать smoke submissions.

---

## Что сознательно не делать

- Новый `Competition` / `ContestTrial` / `User` для жюри.
- `jury_score` одним полем у конкурсанта.
- Кнопка «Отправить» как обязательный workflow.
- Autosave только в React state.
- WebSocket как единственное хранилище оценок.
- Frontend total как official result.
- Удаление `LifeEvent` при исправлении.
- Hardcode критериев и формул (ТЗ §102 оставить configurable).
- Ломать `submissions` и загрузку файлов.

Приоритет при конфликте (ТЗ §109): сохранность данных → backward compatibility → reuse сущностей → jury scores → server security → concurrency → инварианты ТЗ → conventions репо.

---

## Готовность к Phase 2

Можно стартовать, когда есть явное «начинай Phase 2». Первый коммитный срез: миграция роли `JURY` + пустой модуль `evaluation` с `GET/PUT .../evaluation` и UI-заглушкой схемы на карточке испытания, без live и без IndexedDB.
