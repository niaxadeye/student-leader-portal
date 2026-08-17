# Рабочий журнал: платформа мероприятий

Этот файл фиксирует ход внедрения требований из `codex_event_platform_spec.md`,
принятые архитектурные решения, выполненные проверки и обнаруженные ограничения.
Обновлять его после каждого законченного логического блока.

## 1. Исходное состояние

- Дата начала: 2026-08-17.
- Базовый commit: `2f14ee8` (`master`, синхронизирован с `origin/master`).
- Стек backend: Go, chi, pgx, PostgreSQL, внешний Timeweb Cloud S3.
- Стек frontend: React, TypeScript, Vite, React Router, TanStack Query, Zod.
- Архитектура: модульный монолит, API с префиксом `/api/v1`.
- Миграции: forward-only SQL, выполняются отдельной admin-командой.
- До начала разработки успешно собраны API, worker и frontend.
- Существующий frontend lint не запускается из-за отсутствующего `eslint.config.*`.
- Автоматических Go-тестов в исходном проекте пока нет; присутствуют smoke-скрипты.

Локальные изменения, существовавшие до начала внедрения:

- `docs/STATUS.md` изменён пользователем;
- `codex_event_platform_spec.md` добавлен пользователем.

Эти файлы нельзя перезаписывать или удалять без отдельной необходимости.

## 2. Ключевые архитектурные решения

### 2.1. Мероприятие

Отдельной модели `Event` в проекте нет. Существующий `Contest` уже содержит UUID,
slug, timezone, статусы, владельца и scoped access сотрудников.

Принятое соответствие:

```text
Event из нового ТЗ = Contest в существующем проекте
eventId = contestId
```

Новую таблицу `events` не создавать, чтобы не дублировать агрегат мероприятия.
Во внешних participant routes допустимо использовать термин `event`, но все FK в БД
должны ссылаться на `contests.id`.

### 2.2. Участник мероприятия

Существующие `contest_participants` привязаны к обычным `users` и обслуживают старый
кабинет конкурсанта. Они не соответствуют новому требованию.

Нужно создать отдельную сущность `EventParticipant`, не связанную с staff-auth и не
требующую строки в `users`.

Старый user-based contestant flow должен продолжить работать без изменений.

### 2.3. Группы

Сущность `Group` в текущем проекте не найдена. В текущий scope она не входит, поэтому
новую модель группы пока не создаём. Возможность добавить связь с группой позднее
должна сохраняться.

### 2.4. Авторизация

- Staff auth: существующие JWT access tokens и refresh cookie, без переписывания.
- Participant auth: отдельная opaque session cookie и отдельный middleware/principal.
- Participant token нельзя хранить в `localStorage`.
- Заблокированный или архивный participant не должен проходить auth и использовать
  ранее созданную сессию.

### 2.5. Баллы и критические операции

- `PointsLedger` — единственный источник истины баланса.
- Баланс нельзя редактировать напрямую.
- Резерв мерча использует `PointsHold`.
- Attendance, task approval, order reserve/issue/reject выполняются транзакционно.
- Защита от повторов обеспечивается одновременно conditional update, unique constraints
  и idempotency keys.

## 3. Существующие компоненты для переиспользования

- `contests` и `contests.Repo.AccessLevel`;
- `users`, `roles`, `user_roles` только для сотрудников;
- server-side OWNER/EDIT/VIEW access;
- `audit_logs` и модуль `audit`;
- закрытый S3 и presigned download URLs;
- API envelope `{data, meta, request_id}`;
- CSRF Origin middleware;
- PostgreSQL transactions через pgx;
- transactional outbox;
- feature flags `participant_cabinet`, `attendance`, `points`, `merch`;
- frontend UI-kit и TanStack Query conventions.

## 4. План миграций

- [x] `0011_event_participants.up.sql` (применена к рабочей БД)
  - event participants;
  - participant sessions;
  - scoped staff permissions;
  - необходимые расширения audit log;
  - unique constraints и индексы поиска.
- [x] `0012_points_ledger.up.sql` (применена к рабочей БД)
  - immutable ledger;
  - idempotency constraints;
  - индексы расчёта баланса.
- [x] `0013_lectures_attendance.up.sql` (применена к рабочей БД)
  - lectures;
  - attendance;
  - уникальность посещения.
- [x] `0014_event_tasks.up.sql` (применена к рабочей БД)
  - tasks;
  - submission attempts;
  - private assets;
  - ограничения повторного reward.
- [x] `0015_merch.up.sql` (применена к рабочей БД)
  - products and images;
  - saving target;
  - orders and items;
  - points holds;
  - stock and retry constraints.

Все миграции должны быть additive. Для финансовых и исторических данных не применять
опасные cascade delete.

## 5. Этапы реализации

### Этап 1. EventParticipant и participant auth

- [x] Уточнить схему таблиц и DB constraints.
- [x] Добавить миграцию `0011`.
- [x] Реализовать backend CRUD, поиск, фильтры, block/archive.
- [x] Реализовать нормализацию ФИО (`trim`, пробелы, регистр, `ё` → `е`).
- [x] Реализовать вход по ФИО и дате рождения.
- [x] Реализовать вход по профсоюзному билету.
- [x] Реализовать вход по barcode СКС.
- [x] Реализовать participant session cookie и logout.
- [x] Реализовать rate limiting для текущего single-process deployment.
- [x] Реализовать CSV/XLSX import и экспорт с построчными ошибками.
- [x] Добавить participant login/me frontend routes.
- [x] Добавить admin UI списка, поиска, импорта и управления участниками.
- [x] Добавить тесты и выполнить проверки этапа.

### Этап 2. PointsLedger

- [x] Добавить миграцию `0012`.
- [x] Реализовать balance service.
- [x] Реализовать idempotent ledger writes.
- [x] Реализовать admin adjustments с обязательной причиной.
- [x] Добавить audit и тесты.

### Этап 3. Лекции и attendance

- [x] Добавить миграцию `0013`.
- [x] CRUD лекций.
- [x] Короткоживущий signed QR.
- [x] Camera scanner.
- [x] USB HID scanner flow.
- [x] Транзакционное attendance + reward.
- [x] Concurrency и retry tests.

### Этап 4. Задания

- [x] Добавить миграцию `0014`.
- [x] CRUD заданий.
- [x] Submission images/links.
- [x] Private file access.
- [x] Moderation queue.
- [x] Approval/rejection/resubmission.
- [x] Однократный reward и concurrency tests.

### Этап 5. Мерч

- [x] Добавить миграцию `0015`.
- [x] Products, images и уникальные slug.
- [x] Saving target.
- [x] Atomic order reservation.
- [x] Stock reservation и `PointsHold`.
- [x] Issue/reject/cancel transitions.
- [x] Concurrency и retry tests.

### Этап 6. Hardening

- [x] Полный audit критических действий.
- [x] File MIME/magic-byte validation.
- [x] Permissions matrix.
- [x] OpenAPI contract.
- [x] Regression smoke tests.
- [x] Build, vet, typecheck и lint/tooling review.

## 6. Обязательные инварианты реализации

1. Participant всегда принадлежит одному `Contest`.
2. Participant не является `User`.
3. Все participant entities принадлежат одному contest; cross-contest связи запрещены.
4. Ledger является append-only.
5. Attendance reward начисляется один раз.
6. Task reward начисляется один раз.
7. Активные holds вычитаются из доступного баланса.
8. Заказ и резерв товара/баллов создаются атомарно.
9. Выдача заказа и списание баллов выполняются атомарно.
10. Reject/cancel освобождает stock и hold атомарно.
11. Цена заказа фиксируется в item при создании.
12. Все staff permissions проверяются на backend и scoped по contest.
13. QR не содержит персональные данные или публичный participant ID.
14. Приватные submission-файлы не получают постоянные public URLs.

## 7. Известные риски и технический долг

- Participant login защищён in-memory fixed-window limiter по event + IP + User-Agent;
  при горизонтальном масштабировании потребуется внешняя распределённая реализация
  того же интерфейса.
- Magic-byte validation включена для JPEG/PNG/WEBP/GIF. Произвольные нерастровые
  вложения старого challenge flow (PDF/PPT/PPTX/MP4/MOV/ZIP) по-прежнему проверяются
  по настройкам поля, расширению и размеру; глубокая проверка каждого контейнерного
  формата требует отдельного специализированного валидатора.
- Slug generation различается между модулями и не создаёт suffix при конфликте.
- Обычный `audit.Log` выполняется best-effort; для финансовых операций добавлен
  tx-aware writer, и admin points adjustment пишет audit атомарно с ledger.
- OpenAPI описывает 50 маршрутов, включая весь новый event platform flow; часть старых
  auth/contest/challenge/submission/user endpoints пока остаётся в legacy-документации.
- Camera scanner использует нативный `BarcodeDetector`; для браузеров без него
  остаются USB HID и ручной ввод. Кроссбраузерный WASM/JS decoder можно добавить
  позднее отдельным lazy-loaded chunk.
- После `npm ci` зафиксировано 6 dependency vulnerabilities: 3 moderate, 3 high.
- Корневой `.env` недоступен текущему build-процессу, поэтому `make build` завершается
  до выполнения целей; прямые команды `go build` и `npm run build` работают.

## 8. Журнал работ

### 2026-08-17 — подготовка

- Изучено техническое задание `codex_event_platform_spec.md`.
- Исследованы backend, frontend, DB migrations, auth, RBAC, storage, audit и API conventions.
- Принято решение использовать `Contest` как `Event`.
- Выполнен `git pull --ff-only`, получен commit `2f14ee8`.
- Синхронизированы frontend dependencies через `npm ci`.
- Успешно собраны API, worker и frontend.
- Создан этот рабочий журнал.

### 2026-08-17 — этап 1, backend core

- Добавлена additive-миграция `0011_event_participants.up.sql`.
- Созданы таблицы `event_participants`, `participant_sessions`,
  `event_staff_permissions` и participant-связь в `audit_logs`.
- Уникальность профбилета и barcode обеспечена partial unique indexes в рамках конкурса.
- Создан backend-модуль `internal/modules/eventparticipants`.
- Реализованы scoped admin list/get/create/update/block/unblock/archive.
- Реализована нормализация ФИО и три независимых login flow.
- Participant session использует криптостойкий opaque token; в БД сохраняется SHA-256 hash.
- Middleware при каждом запросе проверяет session, статус participant и статус contest.
- BLOCKED/ARCHIVED транзакционно отзывают активные participant sessions.
- Добавлены participant-aware audit entries без создания фиктивного `User`.
- Добавлены API routes:
  - `/api/v1/admin/contests/:contestId/participants`;
  - `/api/v1/events/:eventSlug/participant-auth/*`;
  - `/api/v1/participant/me` и `/api/v1/participant/logout`.
- Добавлены unit-тесты нормализации, ambiguous identity, трёх login flow,
  blocked participant, inactive contest и hash-only session storage.
- Добавлен fixed-window rate limiter: по умолчанию 10 попыток за 5 минут на комбинацию
  event + IP + User-Agent; параметры вынесены в env.
- Добавлен unit-тест блокировки и сброса окна rate limiter.
- Проверки: `go test ./...` — PASS, `go vet ./...` — PASS,
  `npm run build` — PASS с прежним предупреждением о размере bundle.

### 2026-08-17 — этап 1, import/export участников

- Добавлен единый импорт CSV/XLSX с обязательными колонками `full_name`,
  `birth_date` и optional-колонками `union_card_number`, `sks_barcode`.
- Поддержаны английские и русские названия колонок, CSV-разделители `,`, `;`, tab,
  даты `YYYY-MM-DD`, `DD.MM.YYYY`, `DD/MM/YYYY`, RFC3339 и Excel date serial.
- Порядок сопоставления строк: профбилет/barcode как сильные идентификаторы, затем
  единственное совпадение нормализованных ФИО + даты рождения.
- Если профбилет и barcode указывают на разных участников, строка возвращается с
  ошибкой; неоднозначные ФИО + дата и записи без изменений считаются дубликатами.
- Пустые optional-ячейки при обновлении не стирают уже сохранённые идентификаторы.
- Каждая строка получает статус `added`, `updated`, `error` или `duplicate` и причину;
  API также возвращает итоговые счётчики из ТЗ.
- Добавлены admin endpoints:
  - `POST /api/v1/admin/contests/:contestId/participants/import`;
  - `GET /api/v1/admin/contests/:contestId/participants/export?format=csv|xlsx`.
- Для XLSX добавлена прямая зависимость `github.com/xuri/excelize/v2`.
- Добавлены тесты CSV с русскими заголовками, XLSX round-trip, обязательных колонок,
  добавления, обновления, конфликта идентификаторов и неоднозначных дублей.
- Проверки: `go test ./...` — PASS, `go vet ./...` — PASS, API build — PASS,
  `npm run build` — PASS с прежним предупреждением о размере bundle.

### 2026-08-17 — этап 1, frontend participant auth и кабинет

- Добавлены frontend routes `/event/:eventSlug/login` и `/event/:eventSlug/me`.
- Создан отдельный participant API client: он использует только HttpOnly participant
  cookie и намеренно не подключён к staff access-token/refresh flow.
- Создан отдельный `ParticipantAuthProvider` с восстановлением сессии через
  `/api/v1/participant/me`, logout и in-memory состоянием профиля.
- Route guards проверяют не только наличие participant session, но и совпадение
  `eventSlug`, поэтому сессия одного мероприятия не открывает кабинет другого.
- Реализованы три формы входа: ФИО + дата рождения, профбилет и barcode СКС.
- Для barcode поддержаны ручной ввод и HID USB scanner flow с отправкой по Enter.
- Для неоднозначных ФИО показывается предложение войти по профбилету или barcode;
  отдельно обработаны неверные данные, rate limit и отсутствие сети.
- Кабинет показывает мероприятие, ФИО, дату рождения и optional-идентификаторы.
- Баланс, QR, задания, лекции и магазин обозначены как последующие модули без
  фиктивных данных; кнопка QR неактивна до появления signed QR backend API.
- Календарная дата рождения форматируется без timezone-сдвига.
- Проверка: `npm run build` — PASS. Сохраняется прежнее предупреждение Vite о
  bundle крупнее 500 КБ; текущий основной JS bundle — около 640 КБ.

### 2026-08-17 — этап 1, admin UI участников мероприятия

- В карточку мероприятия добавлен отдельный раздел «Участники мероприятия»;
  существующие User-конкурсанты переименованы в UI в «Конкурсные аккаунты», чтобы
  две модели не смешивались.
- Добавлен staff-auth API client для EventParticipant CRUD, status actions,
  multipart import и бинарного CSV/XLSX export.
- Список использует серверные поиск, фильтр `ACTIVE/BLOCKED/ARCHIVED` и пагинацию
  по 25 записей.
- Реализовано ручное создание и редактирование ФИО, даты рождения, профбилета и
  barcode СКС с обработкой конфликта уникальных идентификаторов.
- Реализованы block/unblock/archive. Перед блокировкой UI предупреждает об отзыве
  активных participant sessions; архивирование подтверждается как финальное действие.
- Импорт принимает `.csv` и `.xlsx` до 16 МиБ и показывает отдельный построчный
  отчёт со статусами added/updated/error/duplicate и сводными счётчиками.
- Экспорт доступен в CSV и XLSX; пустой CSV можно использовать как шаблон импорта.
- Раздел показывается владельцу мероприятия/MEGA_ADMIN согласно текущему
  `access_level`; backend дополнительно проверяет scoped permission.
- Проверка: `npm run build` — PASS. Основной JS bundle вырос примерно до 658 КБ;
  предупреждение Vite о необходимости code splitting остаётся некритичным.

### 2026-08-17 — этап 2, PointsLedger

- Добавлена additive-миграция `0012_points_ledger.up.sql` с типами операций,
  обязательными actor/reason для `ADMIN_ADJUSTMENT` и безопасными foreign keys.
- Ledger защищён от `UPDATE/DELETE` PostgreSQL trigger: исправления выполняются
  только новой компенсирующей операцией.
- Добавлены уникальный `(contest_id, idempotency_key)` и дополнительная уникальность
  participant + type + source для будущих attendance/task/order операций.
- Реализованы balance service и история операций. Источник баланса — только
  `SUM(points_ledger.amount)`; прямого mutable-поля balance нет.
- API баланса сразу возвращает `ledger_balance`, `reserved_points` и
  `available_points`. До появления `PointsHold` в merch-этапе reserved равен нулю.
- Добавлены обычный и transaction-aware idempotent writers. `AppendTx` позволит
  следующим модулям атомарно создавать attendance/submission transition и reward.
- Повтор той же операции с тем же ключом возвращает исходную запись без повторного
  начисления; другой payload с занятым ключом возвращает conflict.
- Ручная корректировка требует ненулевую целую сумму, причину, staff actor,
  `event.points.manage` и idempotency key.
- Ledger entry и критический audit `EVENT_POINTS_ADMIN_ADJUSTED` создаются в одной
  PostgreSQL-транзакции; ошибка audit приводит к rollback корректировки.
- Добавлены endpoints:
  - `GET /api/v1/participant/points`;
  - `GET /api/v1/admin/contests/:contestId/participants/:participantId/points`;
  - `POST /api/v1/admin/contests/:contestId/participants/:participantId/points/adjustments`.
- Кабинет участника показывает реальный общий, зарезервированный и доступный баланс,
  а также последние операции. React Query cache scoped по event + participant.
- В admin UI участника добавлен диалог баланса, истории и ручной корректировки.
  Browser-generated idempotency key сохраняется при повторе после сетевой ошибки.
- Unit-тесты покрывают idempotent retry, конфликт payload, permission/validation,
  отсутствующего участника, balance contract и ограничения типов/знаков операций.
- Проверки: `go test -count=1 ./...` — PASS, `go vet ./...` — PASS, API/worker
  builds — PASS, `npm run build` — PASS. Основной JS bundle — около 667 КБ.

### 2026-08-17 — выкладка этапов 1–2 и исправление добавления участника

- Ошибка в admin UI воспроизведена по рабочим логам: `GET/POST .../participants`
  возвращали `404`, потому что frontend был собран раньше перезапуска backend.
- В рабочей БД отсутствовали новые таблицы: были применены миграции только до
  `0010_multitenancy`.
- Пересобраны `bin/api`, `bin/worker` и `bin/admin`; backend-тесты прошли.
- Успешно применены миграции `0011_event_participants` и `0012_points_ledger`.
- Выяснено, что рабочие API и worker управляются PM2, несмотря на наличие старых
  disabled systemd units. Запуск systemd API конфликтует с PM2 за порт `8080`.
- Штатный PM2-процесс `eazytech-api` перезапущен на новом бинарнике; конфликтующий
  systemd-процесс остановлен и оставлен disabled.
- После выкладки `/health/ready` возвращает `200`, а новый participants route без
  авторизации возвращает ожидаемый `401` вместо прежнего `404`.
- Правило следующих выкладок: собрать также `bin/admin`, применить новые миграции,
  затем перезапустить `eazytech-api` через PM2 (не через systemd).

### 2026-08-17 — этап 3, лекции и attendance

- Добавлена additive-миграция `0013_lectures_attendance.up.sql`: `lectures`,
  `participant_qr_codes`, `lecture_attendance`, contest-scoped foreign keys,
  `UNIQUE(lecture_id, event_participant_id)` и уникальность использованного QR nonce.
- Реализован backend-модуль `internal/modules/lectures`: CRUD, переходы
  `DRAFT → ACTIVE → FINISHED`, scoped permissions `event.attendance.manage|scan`,
  список посещений и participant history.
- QR token живёт по умолчанию 45 секунд, подписан HMAC-SHA256 и не содержит
  participant/event ID или персональные данные. В БД хранится только SHA-256 nonce;
  использованный код атомарно помечается в транзакции attendance.
- Scan transaction блокирует QR code, проверяет статус мероприятия/участника/лекции
  и окно регистрации, создаёт attendance, начисляет `LECTURE_ATTENDANCE` через
  `points.AppendTx` и пишет критический audit. Повтор той же лекции возвращает
  `already_checked` без второго начисления; повтор кода для другой лекции отклоняется.
- Добавлены admin API/UI: список и редактор лекций в карточке мероприятия,
  activate/finish/delete draft, scanner page и история посещений.
- Scanner page поддерживает камеру через `BarcodeDetector`, USB 2D scanner в HID
  keyboard mode (автоотправка по Enter, очистка и возврат focus) и ручной ввод.
- В participant cabinet добавлены `/event/:eventSlug/me/qr`, автообновление QR,
  countdown и список активных/завершённых лекций с отметкой посещения.
- Тесты покрывают подпись/tamper/expiry QR, validation/permissions и 32 конкурентных
  повтора скана: создаётся ровно одно attendance. `go test -count=1 ./...`,
  `go vet ./...`, API/admin/worker builds и `npm run build` — PASS.
- `npm run lint` по-прежнему не запускается из-за отсутствующего flat-config ESLint 9;
  это существующий tooling debt. Frontend bundle около 691 KB, QR renderer вынесен
  Vite в отдельный lazy-loaded chunk около 26 KB.
- Миграция `0013` применена к рабочей БД, таблицы проверены read-only запросом;
  API перезапущен через PM2, `/health/ready` — 200, новый lectures route без auth —
  401, с SUPER_ADMIN token — 200. Ошибок запуска в PM2 log нет.

### 2026-08-17 — этап 4, задания и модерация

- Добавлена и применена additive-миграция `0014_event_tasks.up.sql`: задания,
  одна логическая отправка на task + participant, immutable attempts и отдельные
  assets для `IMAGE|LINK`. Исторические связи используют `ON DELETE RESTRICT`.
- Реализован backend-модуль `internal/modules/eventtasks`: CRUD, переходы
  `DRAFT|DISABLED → ACTIVE → DISABLED|ARCHIVED`, расписание и конфигурация допустимых
  типов подтверждения. Обложки и изображения ограничены 20 МБ на файл.
- Participant flow поддерживает до десяти изображений и десяти HTTP(S)-ссылок,
  комментарий, историю попыток и повторную отправку только после `REJECTED`.
- Объекты изображений остаются приватными. Участник получает presigned URL только
  для assets собственной отправки; staff — только после contest-scoped проверки
  `event.tasks.moderate|manage` и совпадения submission.
- Модерация показывает очереди `PENDING|APPROVED|REJECTED`, материалы и комментарии.
  Reject требует причину. Approval в одной PostgreSQL-транзакции блокирует submission,
  переводит текущую попытку в `APPROVED`, добавляет `TASK_REWARD` через
  `points.AppendTx` и пишет audit.
- Повторный approval является replay: unique task + participant, reward state constraint,
  `FOR UPDATE`, уникальный ledger source и idempotency key не допускают второго reward.
- В admin UI добавлены карточки/редактор заданий и очередь проверки. В participant
  cabinet добавлены `/event/:eventSlug/tasks` и `/:taskId`, отправка материалов,
  статусы проверки, причина отказа и история попыток.
- Тесты покрывают validation ссылок/изображений/расписания, допустимые переходы
  отправки, scoped asset paths и 32 конкурентных approval: новое решение одно,
  остальные ответы replay. `go test -count=1 ./...`, `go test -race`, `go vet ./...`,
  API/admin/worker builds и `npm run build` — PASS.
- Миграция и ключевые DB constraints проверены read-only запросом. API перезапущен
  через PM2; `/health/ready` — 200, новые admin/participant routes без auth — 401,
  ошибок запуска в PM2 log нет. Frontend bundle около 721 КБ; code splitting остаётся
  в hardening/tooling debt.

### 2026-08-17 — participant regression hotfix

- Исправлен `500` при `BLOCKED/ARCHIVED`: PostgreSQL не мог вывести единый тип
  параметра статуса, который использовался и как `varchar`, и как `text` в `CASE`.
  В `eventparticipants.Repo.SetStatus` добавлен явный `varchar(16)` cast.
- Добавлен production smoke `scripts/smoke_event_participants.sh`. Он создаёт временного
  участника, проверяет create/update, вход по ФИО, participant session, block с отзывом
  сессии, запрет входа заблокированного, unblock, повторный вход и archive, затем
  удаляет тестовые данные. Результат: `PASS=11 FAIL=0`.
- Фактический slug рабочего мероприятия — `lpb-2026`. Запросы на переставленный
  `lbp-2026` корректно возвращают participant auth failure, так как такого события нет.

### 2026-08-17 — этап 5, магазин мерча

- Добавлена и применена additive-миграция `0015_merch.up.sql`: товары и несколько
  изображений, единственная цель накопления участника, заказы со snapshot цены,
  позиции заказа и `points_holds`. Исторические связи используют `ON DELETE RESTRICT`.
- Реализован backend-модуль `internal/modules/merch` со scoped permissions
  `event.merch.manage` и `event.merch.orders.manage`, уникальными slug с suffix,
  статусами `DRAFT|ACTIVE|HIDDEN|SOLD_OUT` и контролем остатка относительно резерва.
- Резерв заказа выполняется одной PostgreSQL-транзакцией: блокируются participant
  и товары в стабильном порядке, проверяются активность мероприятия/участника,
  доступный остаток и `ledger - active holds`, затем создаются order/items/hold и
  увеличивается `reserved_quantity`.
- Contest-wide idempotency key защищён advisory transaction lock. Повтор идентичного
  запроса возвращает исходный заказ; тот же ключ с другим набором товаров возвращает
  `409 IDEMPOTENCY_CONFLICT`.
- Выдача атомарно уменьшает физический stock и резерв, переводит hold в `CAPTURED`,
  однократно добавляет отрицательную запись `MERCH_PURCHASE` и пишет audit.
  Reject/cancel атомарно освобождают stock reservation и hold без ложной ledger-записи.
- Balance API теперь вычитает сумму активных holds и возвращает фактические
  `reserved_points` и `available_points`.
- Добавлены admin API/UI каталога, изображений, остатков и очереди заказов с выдачей
  или обязательной причиной отказа. В заголовке мероприятия появились открытие и
  копирование ссылки participant login.
- В participant cabinet добавлены `/event/:eventSlug/shop`, карточка товара,
  цель накопления, резерв количества и `/orders` с отменой ожидающего заказа.
  Навигация доступна и на мобильном экране.
- Добавлены unit-тесты validation, slug/canonical fingerprint, MIME и баланса с holds,
  а также production smoke `scripts/smoke_merch.sh` с автоматической очисткой.
  Smoke проверяет create/activate/login/target, idempotent reserve и conflict,
  cancel, issue replay, reject, ledger/hold/stock, историю и восемь конкурентных
  резервов последних двух единиц без oversell: `PASS=30 FAIL=0`.
- `go test ./...`, `go vet ./...` и `npm run build` — PASS. API пересобран и
  перезапущен через PM2, `/health/ready` — 200, ошибок нового запуска в PM2 log нет.
  Production frontend отдаёт asset `index-C-N7fgy1.js`.
- `npm run lint` всё ещё блокируется отсутствующим `eslint.config.*`; основной
  frontend bundle около 750 КБ и требует code splitting на hardening-этапе.

### 2026-08-17 — этап 6, audit и hardening

- Добавлен общий `internal/platform/filevalidation`: JPEG/PNG/WEBP/GIF принимаются
  только при совпадении расширения, заявленного MIME и бинарной сигнатуры. Прочитанный
  заголовок возвращается в поток, поэтому S3 получает исходный файл целиком.
  Проверка подключена к изображениям заданий, мерча и raster-вложениям старого
  challenge submission flow; spoofed payload возвращает `400` до записи объекта.
- Scoped permissions сведены в тестируемую матрицу `eventpermissions`: только
  `event.attendance.manage -> event.attendance.scan` и
  `event.tasks.manage -> event.tasks.moderate` являются иерархическими; permissions
  участников, баллов, каталога и заказов независимы. Все пять репозиториев используют
  одну матрицу и обязательно ограничивают доступ конкретным `contest_id`.
- Проверены критические audit-события участников, attendance, points, tasks и merch.
  Добавлен отсутствовавший `PARTICIPANT_LOGOUT`; финансовые/конкурентные переходы
  продолжают писать audit в той же PostgreSQL-транзакции.
- `backend/api/openapi.yaml` обновлён до `0.6.0`: описано 50 фактических маршрутов
  event platform, bearer и participant-cookie auth, permissions, idempotency,
  multipart uploads, payload schemas и error codes. YAML parse — PASS,
  дубликатов `operationId` нет.
- Добавлен ESLint 9 flat config. Исправлены warnings в lifecycle камеры и QR countdown;
  `npm run lint` — PASS без warnings.
- Все страницы React Router переведены на route-level lazy loading. Основной JS chunk
  уменьшен примерно с 750 КБ до 295 КБ (gzip около 93 КБ); тяжёлые admin/participant
  страницы вынесены в отдельные chunks. `npm run build` — PASS.
- Полный regression: `go test -count=1 ./...`, `go test -race ./...`, `go vet ./...`,
  API/admin/worker builds — PASS. Production smoke участников — `PASS=11 FAIL=0`;
  smoke мерча, включая magic-byte reject/accept/delete и 8 конкурентных резервов —
  `PASS=33 FAIL=0`. После cleanup временных participants/products осталось `0 | 0`.
- API пересобран и перезапущен через PM2; `/health/ready` — 200, новый запуск без
  ошибок. Production frontend собран с основным asset `index-C3mvA6o6.js`.
- Runtime `.env` вручную синхронизирован владельцем: `participant_cabinet`,
  `attendance`, `points` и `merch` подтверждены как `true` через production `/config`;
  после рестарта readiness — 200, ошибок запуска нет.

### 2026-08-17 — переход на Timeweb S3 и удаление локальных сервисов

- Runtime переключён на `https://s3.twcstorage.ru`, регион `ru-1`, bucket
  `3babd9c9-82a4-4c5f-b382-11f24effd682`, path-style включён.
- Клиент object storage переведён на официальный AWS SDK for Go v2; отдельный
  публичный endpoint больше не нужен, presigned URLs подписываются для Timeweb S3.
- Неиспользуемый Redis-конфиг удалён из backend и `.env`; Compose теперь поднимает
  только PostgreSQL. Старые Redis и локальный object storage удалены из исходников,
  примеров окружения, Makefile и эксплуатационной документации.
- После успешного production smoke удалены контейнеры `slc-minio`,
  `slc-minio-init`, `slc-redis`, data volumes и Docker images. PostgreSQL и его
  volume не затронуты. Четыре старых локальных объекта удалены без переноса.
- Финальный production smoke после физического удаления сервисов проверяет upload,
  presigned download, побайтовое совпадение, delete и весь merch flow:
  `PASS=35 FAIL=0`. `/health/ready` — 200.

### 2026-08-17 — кросс-браузерный QR-сканер

- Удалена зависимость scanner UI от нативного `window.BarcodeDetector`, из-за
  которой камера не распознавала QR в Safari/iPhone и Firefox.
- Подключён `@yudiel/react-qr-scanner` с ZXing/WebAssembly fallback, rear-camera
  constraints, выбором камеры, torch/zoom controls, debounce повторных кадров и
  понятными ошибками permission/no-camera/in-use/HTTPS.
- Добавлено чтение QR из PNG/JPEG/WEBP как резерв для устройства без камеры.
- Camera UI ограничен coarse-touch мобильными устройствами до 1023 px; на ПК
  остаются только USB HID и ручной ввод. Мобильный preview увеличен до 72svh,
  finder-рамка скрыта, автофокус input отключён, а при запущенной камере input disabled.
- `zxing_reader.wasm` включён в production assets и обслуживается локально, без
  runtime-зависимости от CDN; nginx настроен на `application/wasm`.
- `npm run lint`, `npm run build` и автономный decode сгенерированного QR через
  локальный WASM — PASS. Новый frontend опубликован на `eazytech.ru`.

## 9. Следующий шаг

Основная реализация требований `codex_event_platform_spec.md`, hardening и runtime
feature flags завершены. Следующий эксплуатационный шаг — browser acceptance основных
staff/participant сценариев, особенно dialogs/dropdowns участников и мобильного
кабинета. После этого можно брать отдельный backlog: распределённый limiter для
нескольких API-инстансов, глубокая проверка контейнерных файлов и OpenAPI legacy-модулей.
