# Техническое задание
## Личный кабинет студенческого лидера, участника и дирекции конкурса

**Рабочее название:** Student Leader Cabinet  
**Формат:** техническое задание для Claude Code  
**Основной язык интерфейса:** русский  
**Дизайн:** описывается отдельно в `DESIGN.md`  
**Архитектура:** модульный монолит с возможностью дальнейшего выделения сервисов

## 1. Технологический стек

### Frontend

- React;
- Vite;
- TypeScript;
- React Router;
- TanStack Query;
- React Hook Form;
- Zod;
- Zustand только для действительно глобального клиентского состояния;
- типизированный API-клиент на основании OpenAPI;
- UI-компоненты и визуальные правила — по `DESIGN.md`.

### Backend

- Go;
- REST API;
- PostgreSQL;
- `pgx`;
- `sqlc`;
- миграции через `goose`, `tern` или аналог;
- JWT access/refresh;
- S3-совместимое объектное хранилище;
- Redis для распределённого rate limit, locks и фоновых задач при необходимости;
- Telegram Bot API;
- Prometheus;
- OpenTelemetry;
- structured logging.

### Инфраструктура

- Docker;
- Docker Compose для разработки;
- reverse proxy;
- HTTPS;
- CI/CD;
- staging и production;
- MinIO в локальной среде;
- PostgreSQL;
- Redis;
- отдельные процессы `api` и `worker`.

Точные версии зависимостей фиксируются в `go.mod`, `package.json` и lock-файлах.

---

# 2. Назначение системы

Система предназначена для:

1. Организации конкурсных испытаний студенческих лидеров.
2. Сбора материалов и технических требований от конкурсантов.
3. Управления конкурсантами со стороны дирекции.
4. Хранения черновиков и официальных ревизий отправленных форм.
5. Автоматического уведомления дирекции в Telegram.
6. Размещения справочной информации.
7. Последующего расширения до платформы участника с лекциями, баллами, QR-сканированием, магазином мерча и прогнозами.

Первая версия должна быть production-ready, а не демонстрационным прототипом.

---

# 3. Границы первой версии

## 3.1. Обязательный функционал

- авторизация по логину и паролю;
- JWT access token;
- refresh token с ротацией;
- защищённые backend endpoints;
- защищённые frontend routes;
- роли и уровни доступа;
- личный кабинет конкурсанта;
- административный кабинет дирекции;
- создание конкурсов;
- создание конкурсных испытаний;
- динамический конструктор полей;
- подсказка или справка у каждого поля;
- сохранение черновика;
- официальная отправка;
- изменение уже отправленной формы;
- хранение истории ревизий;
- загрузка множества файлов разных типов;
- отдельная справочная страница;
- Telegram-уведомления;
- аудит действий;
- экспорт;
- Docker-окружение;
- миграции;
- unit, integration и E2E-тесты;
- OpenAPI;
- production-конфигурация.

## 3.2. Будущее расширение

Архитектура должна учитывать:

- роль обычного участника;
- лекции и расписание;
- разное количество баллов за каждую лекцию;
- QR-код и штрихкод участника;
- сканирование сотрудником с другого телефона;
- автоматическое начисление баллов;
- балльный кошелёк;
- магазин мерча;
- управление остатками;
- резервы, выдачу и списания;
- прогнозы на победителей;
- роль жюри и оценивание.

Эти модули должны быть предусмотрены архитектурно и описаны в этом ТЗ, но могут быть выключены feature flags в первой версии.

## 3.3. Не входит в первую версию

- онлайн-оплата;
- публичная самостоятельная регистрация;
- мобильное приложение;
- автоматическая оценка работ;
- email и SMS-уведомления, если не будут согласованы отдельно;
- видеоконвертация;
- встроенный редактор презентаций или видео;
- электронная подпись;
- публичный рейтинг.

---

# 4. Термины

**Конкурс** — верхнеуровневая сущность, объединяющая участников, испытания, сроки и справочную информацию.

**Конкурсант** — пользователь, участвующий в конкурсных испытаниях.

**Участник** — пользователь мероприятия, который может посещать лекции и использовать баллы, но не обязательно участвует в конкурсе.

**Конкурсное испытание** — задание с настраиваемой формой.

**Схема формы** — набор полей, типов, справок, ограничений и правил валидации.

**Черновик** — сохранённая, но не отправленная форма.

**Отправка** — официальная передача формы дирекции.

**Ревизия** — неизменяемый снимок формы на момент отправки или обновления.

**Справка к полю** — пояснение, инструкция или пример заполнения.

**Справочная страница** — отдельная страница с регламентами, контактами и инструкциями.

---

# 5. Роли

## 5.1. `SUPER_ADMIN`

- полный доступ ко всей системе;
- создание администраторов;
- управление ролями;
- управление всеми конкурсами;
- управление глобальными настройками;
- просмотр полного аудита;
- блокировка пользователей;
- завершение сессий;
- сброс пароля;
- настройка Telegram;
- архивирование данных.

## 5.2. `ADMIN`

Доступ только к назначенным конкурсам.

- создание и редактирование испытаний;
- управление схемой формы;
- настройка справок;
- управление дедлайнами;
- добавление конкурсантов;
- импорт конкурсантов;
- просмотр заполненных форм;
- просмотр истории ревизий;
- скачивание материалов;
- блокировка и разблокировка формы;
- редактирование справочной информации;
- экспорт данных;
- просмотр аудита своего конкурса.

## 5.3. `CONTESTANT`

- просмотр назначенных конкурсов;
- просмотр доступных испытаний;
- заполнение формы;
- сохранение черновика;
- загрузка и удаление собственных файлов;
- официальная отправка;
- обновление после отправки, если разрешено;
- просмотр своих ревизий;
- просмотр справочной информации;
- смена пароля;
- управление своими сессиями.

## 5.4. Будущие роли

- `PARTICIPANT`;
- `SCANNER`;
- `MERCH_MANAGER`;
- `JURY`.

---

# 6. Разграничение доступа

Использовать RBAC с проверкой scope конкурса.

Для каждого запроса backend проверяет:

1. Авторизацию.
2. Активность пользователя.
3. Активность сессии.
4. Роль.
5. Доступ к конкретному конкурсу.
6. Принадлежность формы.
7. Принадлежность файла.
8. Дедлайн.
9. Блокировку формы.
10. Доступность действия по статусу.

Принципы:

- `ADMIN` конкурса A не видит конкурс B;
- `CONTESTANT` не получает чужую форму по подставленному ID;
- файл не имеет постоянной публичной ссылки;
- временная ссылка выдаётся только после backend-проверки;
- скрытие кнопки во frontend не считается защитой.

---

# 7. Основные сценарии

## 7.1. Добавление конкурсанта

Администратор указывает:

- ФИО;
- уникальный логин;
- email;
- телефон;
- организацию;
- город;
- конкурс;
- статус;
- внутренний номер при необходимости.

Система:

1. Проверяет уникальность.
2. Создаёт пользователя.
3. Назначает роль `CONTESTANT`.
4. Привязывает к конкурсу.
5. Генерирует временный пароль либо принимает заданный администратором.
6. Показывает пароль только один раз.
7. Хранит только хэш.
8. Ставит `must_change_password=true`.
9. Записывает действие в аудит.

## 7.2. Вход

1. Пользователь вводит логин и пароль.
2. Backend проверяет пароль и статус.
3. Создаёт access token.
4. Создаёт refresh token.
5. Сохраняет хэш refresh token.
6. Передаёт refresh token в `HttpOnly` cookie.
7. Возвращает access token.
8. Frontend хранит access token только в памяти.
9. При временном пароле перенаправляет на смену пароля.

## 7.3. Заполнение формы

1. Конкурсант открывает испытание.
2. Frontend получает схему формы.
3. Frontend получает существующий черновик.
4. Конкурсант заполняет поля.
5. Загружает файлы.
6. Нажимает «Сохранить черновик».
7. Backend валидирует структуру.
8. Данные сохраняются.
9. Telegram не уведомляется.
10. Пользователь может вернуться позже.

Разрешается автосохранение с debounce, но должна оставаться явная кнопка сохранения.

## 7.4. Отправка

1. Пользователь нажимает «Отправить».
2. Frontend показывает подтверждение.
3. Backend повторно валидирует обязательные поля.
4. Проверяет завершение загрузок.
5. Создаёт immutable-ревизию.
6. Меняет статус на `SUBMITTED`.
7. Создаёт outbox event.
8. Worker отправляет Telegram-уведомление.
9. Пользователь видит номер ревизии и дату отправки.

## 7.5. Обновление отправленной формы

Разрешено, если:

- испытание открыто;
- дедлайн не истёк либо разрешена поздняя отправка;
- форма не заблокирована;
- администратор разрешил обновление.

При обновлении:

- создаётся следующая ревизия;
- предыдущая сохраняется;
- фиксируются изменённые поля;
- Telegram получает уведомление об обновлении;
- действие записывается в аудит.

## 7.6. Просмотр дирекцией

Администратор видит таблицу:

- конкурсант;
- организация;
- испытание;
- статус;
- дата сохранения;
- дата отправки;
- номер ревизии;
- количество файлов;
- дедлайн;
- блокировка.

Фильтры:

- не начато;
- черновик;
- отправлено;
- обновлено;
- просрочено;
- заблокировано.

---

# 8. Статусы

## Пользователь

```text
INVITED
ACTIVE
BLOCKED
ARCHIVED
```

## Конкурс

```text
DRAFT
ACTIVE
FINISHED
ARCHIVED
```

## Испытание

```text
DRAFT
PUBLISHED
CLOSED
ARCHIVED
```

## Форма

```text
NOT_STARTED
DRAFT
SUBMITTED
LOCKED
```

Вычисляемые состояния:

```text
OVERDUE
UPDATED_AFTER_SUBMISSION
```

## Файл

```text
UPLOADING
UPLOADED
PROCESSING
READY
REJECTED
DELETED
```

## Outbox

```text
PENDING
PROCESSING
SENT
FAILED
DEAD
```

---

# 9. Конкурс

Поля:

- `id`;
- `name`;
- `slug`;
- `description`;
- `status`;
- `start_at`;
- `end_at`;
- `timezone`;
- `settings`;
- `telegram_notifications_enabled`;
- `created_by`;
- `updated_by`;
- `created_at`;
- `updated_at`;
- `archived_at`.

Настройки:

- редактирование после отправки;
- поздняя отправка;
- история ревизий;
- общий лимит файлов;
- лимит одного файла;
- разрешённые типы;
- Telegram chat/thread;
- шаблон уведомлений;
- количество активных сессий;
- срок хранения удалённых файлов.

---

# 10. Конкурсное испытание

Поля:

- `id`;
- `contest_id`;
- `title`;
- `slug`;
- `short_description`;
- `full_description`;
- `instructions`;
- `status`;
- `sort_order`;
- `open_at`;
- `deadline_at`;
- `close_at`;
- `allow_late_submission`;
- `allow_edit_after_submission`;
- `show_revision_history_to_contestant`;
- `is_required`;
- `current_schema_version`;
- `created_by`;
- `updated_by`;
- `created_at`;
- `updated_at`;
- `published_at`;
- `archived_at`.

Администратор может:

- создать;
- дублировать;
- редактировать;
- менять порядок;
- публиковать;
- закрывать;
- архивировать;
- предварительно просматривать;
- назначать всем или отдельным группам;
- настраивать поля и дедлайны.

---

# 11. Конструктор формы

Форма не должна быть жёстко зашита во frontend.

Backend отдаёт схему, frontend динамически рендерит поля.

## 11.1. Типы полей

```text
SHORT_TEXT
LONG_TEXT
RICH_TEXT
NUMBER
URL
EMAIL
PHONE
DATE
TIME
DATETIME
SELECT
MULTISELECT
RADIO
CHECKBOX
FILE
FILE_GROUP
SECTION
INFO_BLOCK
```

Техническое задание на экран, свет и звук рекомендуется собирать из обычных секций и полей, а не хардкодить.

## 11.2. Шаблон технического задания

### Экран

- требуется ли экран;
- формат;
- разрешение;
- соотношение сторон;
- длительность;
- нужен ли звук;
- порядок запуска;
- резервная ссылка;
- подробный комментарий.

### Свет

- общий характер света;
- цвет;
- температура;
- акценты;
- затемнение;
- переходы;
- моменты включения;
- комментарий.

### Звук

- нужен ли микрофон;
- тип;
- количество;
- музыкальное сопровождение;
- стартовая отметка;
- громкость;
- эффекты;
- резервный сценарий;
- комментарий.

## 11.3. Структура поля

```json
{
  "id": "uuid",
  "key": "presentation_files",
  "type": "FILE_GROUP",
  "label": "Презентация и видеоматериалы",
  "description": "Загрузите материалы для испытания",
  "help_text": "Допускаются презентации, видео, изображения, PDF и архивы",
  "required": true,
  "sort_order": 10,
  "settings": {
    "multiple": true,
    "allowed_extensions": ["pdf", "ppt", "pptx", "mp4", "mov", "png", "jpg", "zip"],
    "max_file_size_mb": 2048,
    "max_files": null
  },
  "validation": {},
  "visibility": {
    "roles": ["CONTESTANT"]
  }
}
```

## 11.4. Версионирование схемы

Обязательно:

- хранить `schema_version`;
- сохранять snapshot схемы в ревизии;
- не удалять физически поля с ответами;
- использовать soft delete;
- предупреждать администратора об изменении опубликованной формы;
- требовать подтверждение несовместимых изменений.

Несовместимые изменения:

- смена типа;
- удаление обязательного поля;
- уменьшение лимита файлов;
- удаление уже использованного варианта ответа.

---

# 12. Черновики и ревизии

Для сочетания `contestant + challenge` существует одна текущая форма.

Черновик хранит:

- ответы;
- файлы;
- schema version;
- дату первого открытия;
- дату сохранения;
- optimistic lock version.

Рекомендуемая модель:

- статусы и связи — обычные таблицы;
- динамические ответы — `JSONB`;
- файлы — отдельная таблица;
- ревизии — immutable-записи.

Пример:

```json
{
  "project_name": "Название проекта",
  "presentation_comment": "Комментарий",
  "screen_required": true,
  "screen_resolution": "1920x1080"
}
```

Backend валидирует JSON по серверной схеме.

## Optimistic locking

- форма имеет `version`;
- frontend передаёт текущую версию;
- backend обновляет только при совпадении;
- при конфликте возвращает `409`;
- frontend предлагает перезагрузить или сравнить данные.

## Ревизия

Хранит:

- номер;
- action type;
- schema snapshot;
- answers snapshot;
- files snapshot;
- checksum;
- автора;
- дату.

Типы:

```text
INITIAL_SUBMISSION
RESUBMISSION
ADMIN_CORRECTION
```

Административное изменение, если появится, всегда создаёт новую ревизию.

---

# 13. Файлы

## 13.1. Требования

С точки зрения бизнес-логики количество файлов не ограничено.

Технические лимиты конфигурируются:

- размер одного файла;
- общий размер формы;
- параллельные загрузки;
- количество файлов на поле;
- MIME;
- расширения;
- timeout.

## 13.2. Хранилище

- production: S3-совместимое;
- development: MinIO;
- PostgreSQL хранит только метаданные;
- bucket не публичный.

## 13.3. Загрузка

Для больших файлов:

1. Frontend запрашивает загрузку.
2. Backend проверяет права.
3. Создаёт запись `UPLOADING`.
4. Возвращает presigned URL.
5. Frontend загружает напрямую.
6. Сообщает о завершении.
7. Backend проверяет объект.
8. Worker выполняет дополнительную проверку.
9. Статус становится `READY`.

Предусмотреть multipart upload.

## 13.4. Метаданные

- владелец;
- конкурс;
- испытание;
- форма;
- поле;
- bucket;
- object key;
- оригинальное имя;
- безопасное имя;
- extension;
- MIME;
- размер;
- checksum;
- status;
- timestamps.

## 13.5. Безопасность

- не доверять расширению;
- проверять MIME;
- генерировать object key;
- защищать от path traversal;
- presigned URL с коротким TTL;
- опасные типы отдавать как attachment;
- проверять размер после загрузки;
- предусмотреть антивирус;
- вести аудит административных скачиваний.

## 13.6. Удаление

- soft delete;
- физическое удаление worker;
- период хранения;
- файл из старой ревизии не исчезает;
- object storage versioning или immutable copy для ревизий.

---

# 14. Справочная информация

Отдельная страница содержит:

- регламент;
- инструкции;
- требования к материалам;
- требования к презентации;
- требования к видео;
- технические рекомендации;
- контакты;
- FAQ;
- ссылки;
- вложения.

Администратор может:

- создавать страницы;
- редактировать;
- менять порядок;
- публиковать;
- снимать с публикации;
- задавать видимость;
- привязывать к конкурсам;
- добавлять вложения.

Контент — безопасный Markdown или sanitised rich text.

---

# 15. Telegram

## События

- первая отправка;
- обновление отправленной формы;
- критическая ошибка обработки файла;
- будущие напоминания о дедлайне.

Черновики уведомление не создают.

## Пример

```text
📥 Новая отправка

Конкурс: Студенческий лидер 2026
Испытание: Самопрезентация
Конкурсант: Иванова Анна Сергеевна
Организация: Московский Политех
Ревизия: 1
Отправлено: 12.07.2026 15:30
Файлов: 4

Открыть форму: https://example.ru/admin/submissions/{submissionId}
```

## Outbox pattern

Отправка формы и создание `outbox_event` происходят в одной транзакции.

Worker:

1. выбирает `PENDING`;
2. блокирует через `FOR UPDATE SKIP LOCKED`;
3. отправляет;
4. записывает результат;
5. делает retry с exponential backoff;
6. после лимита переводит в `DEAD`.

Сбой Telegram не отменяет успешную отправку формы.

---

# 16. Авторизация и безопасность

## Пароли

- Argon2id или bcrypt;
- минимум 10 символов;
- временный пароль;
- обязательная смена;
- сброс отзывает все сессии;
- пароли не логируются.

## JWT

Access token:

- короткий TTL;
- `iss`;
- `aud`;
- `sub`;
- `exp`;
- `iat`;
- `nbf`;
- `jti`;
- `role`;
- `session_id`.

Refresh token:

- длинный TTL;
- rotation;
- token family;
- reuse detection;
- в базе только хэш;
- отзыв семейства при повторном использовании старого токена.

Рекомендуемые параметры через environment:

```text
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=30d
```

## Хранение

- access token — только память frontend;
- refresh token — `HttpOnly`, `Secure`, `SameSite` cookie;
- не использовать `localStorage` для refresh token.

## CSRF

- `SameSite=Lax` или `Strict`;
- CSRF token для cookie endpoints;
- проверка `Origin`;
- без wildcard CORS при credentials.

## Middleware

- request ID;
- logging;
- panic recovery;
- CORS;
- security headers;
- body limit;
- authentication;
- authorization;
- contest scope;
- rate limit;
- CSRF;
- timeout;
- audit context;
- metrics;
- tracing;
- idempotency;
- compression при необходимости.

## Дополнительная защита

- HTTPS;
- HSTS;
- CSP;
- защита от clickjacking;
- `X-Content-Type-Options`;
- parameterized SQL;
- sanitization;
- secrets через environment/secret manager;
- минимальные права БД;
- резервные копии;
- dependency scanning;
- аудит.

---

# 17. Backend-архитектура

Модули:

```text
auth
users
roles
contests
contestants
challenges
form_schema
submissions
files
reference
notifications
audit
exports
attendance
points
merch
predictions
```

Структура:

```text
backend/
  cmd/
    api/
      main.go
    worker/
      main.go
  internal/
    app/
    config/
    platform/
      db/
      httpserver/
      logger/
      metrics/
      tracing/
      storage/
      telegram/
      security/
    modules/
      auth/
      users/
      contests/
      challenges/
      submissions/
      files/
      reference/
      notifications/
      audit/
    middleware/
  db/
    migrations/
    queries/
  api/
    openapi.yaml
  tests/
    integration/
    e2e/
  Dockerfile
  go.mod
  go.sum
```

Интерфейсы на внешних границах:

- object storage;
- Telegram sender;
- file scanner;
- clock;
- id generator;
- transaction manager.

Транзакции обязательны для:

- создания пользователя и роли;
- привязки к конкурсу;
- отправки;
- создания ревизии;
- outbox;
- будущего начисления баллов;
- будущей покупки мерча.

---

# 18. Frontend-архитектура

```text
frontend/
  src/
    app/
      router/
      providers/
      guards/
      query-client/
      error-boundary/
    pages/
      auth/
      contestant/
      admin/
      reference/
      profile/
    widgets/
    features/
      auth/
      save-draft/
      submit-form/
      upload-file/
      manage-challenge/
      manage-contestant/
    entities/
      user/
      contest/
      challenge/
      submission/
      file/
    shared/
      api/
      ui/
      lib/
      config/
      types/
      validation/
  public/
  Dockerfile
  package.json
  vite.config.ts
```

Состояние:

- серверные данные — TanStack Query;
- формы — React Hook Form;
- глобальное UI-состояние — Zustand при необходимости;
- фильтры — URL params;
- access token — память.

Frontend обрабатывает:

- `400`;
- `401`;
- `403`;
- `404`;
- `409`;
- `413`;
- `422`;
- `429`;
- `500`;
- сетевые ошибки;
- ошибки загрузки;
- конфликт версий;
- истечение сессии.

---

# 19. Маршруты frontend

## Публичные

```text
/login
/forgot-password
/reset-password
```

## Общие

```text
/profile
/security/sessions
/change-password
/reference
/403
/404
```

## Конкурсант

```text
/contestant
/contestant/contests
/contestant/contests/:contestId
/contestant/challenges/:challengeId
/contestant/challenges/:challengeId/history
```

Dashboard:

- текущий конкурс;
- ближайший дедлайн;
- количество испытаний;
- черновики;
- отправленные формы;
- предупреждения;
- справка.

## Администратор

```text
/admin
/admin/contests
/admin/contests/new
/admin/contests/:contestId
/admin/contests/:contestId/edit
/admin/contests/:contestId/challenges
/admin/challenges/new
/admin/challenges/:challengeId/edit
/admin/challenges/:challengeId/preview
/admin/contests/:contestId/contestants
/admin/contestants/new
/admin/contestants/:userId
/admin/submissions
/admin/submissions/:submissionId
/admin/reference
/admin/audit
/admin/settings/telegram
```

---

# 20. API

Префикс:

```text
/api/v1
```

Успех:

```json
{
  "data": {},
  "meta": {},
  "request_id": "uuid"
}
```

Ошибка:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Проверьте заполнение формы",
    "details": {}
  },
  "request_id": "uuid"
}
```

## Auth

```text
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
POST   /api/v1/auth/logout-all
GET    /api/v1/auth/me
POST   /api/v1/auth/change-password
GET    /api/v1/auth/sessions
DELETE /api/v1/auth/sessions/:sessionId
```

## Пользователи

```text
GET    /api/v1/admin/users
POST   /api/v1/admin/users
GET    /api/v1/admin/users/:userId
PATCH  /api/v1/admin/users/:userId
POST   /api/v1/admin/users/:userId/block
POST   /api/v1/admin/users/:userId/unblock
POST   /api/v1/admin/users/:userId/reset-password
POST   /api/v1/admin/users/:userId/roles
DELETE /api/v1/admin/users/:userId/roles/:role
```

## Конкурсы

```text
GET    /api/v1/contests
GET    /api/v1/contests/:contestId
GET    /api/v1/admin/contests
POST   /api/v1/admin/contests
GET    /api/v1/admin/contests/:contestId
PATCH  /api/v1/admin/contests/:contestId
POST   /api/v1/admin/contests/:contestId/publish
POST   /api/v1/admin/contests/:contestId/finish
POST   /api/v1/admin/contests/:contestId/archive
```

## Конкурсанты

```text
GET    /api/v1/admin/contests/:contestId/contestants
POST   /api/v1/admin/contests/:contestId/contestants
DELETE /api/v1/admin/contests/:contestId/contestants/:userId
POST   /api/v1/admin/contests/:contestId/contestants/import
GET    /api/v1/admin/contests/:contestId/contestants/export
```

## Испытания

```text
GET    /api/v1/contests/:contestId/challenges
GET    /api/v1/challenges/:challengeId
GET    /api/v1/admin/contests/:contestId/challenges
POST   /api/v1/admin/contests/:contestId/challenges
GET    /api/v1/admin/challenges/:challengeId
PATCH  /api/v1/admin/challenges/:challengeId
POST   /api/v1/admin/challenges/:challengeId/duplicate
POST   /api/v1/admin/challenges/:challengeId/publish
POST   /api/v1/admin/challenges/:challengeId/close
POST   /api/v1/admin/challenges/:challengeId/archive
PATCH  /api/v1/admin/challenges/reorder
```

## Поля

```text
GET    /api/v1/admin/challenges/:challengeId/fields
POST   /api/v1/admin/challenges/:challengeId/fields
PATCH  /api/v1/admin/challenges/:challengeId/fields/:fieldId
DELETE /api/v1/admin/challenges/:challengeId/fields/:fieldId
PATCH  /api/v1/admin/challenges/:challengeId/fields/reorder
GET    /api/v1/admin/challenges/:challengeId/schema-preview
```

## Формы

```text
GET    /api/v1/challenges/:challengeId/submission
PUT    /api/v1/challenges/:challengeId/submission/draft
POST   /api/v1/challenges/:challengeId/submission/submit
POST   /api/v1/challenges/:challengeId/submission/resubmit
GET    /api/v1/challenges/:challengeId/submission/revisions
GET    /api/v1/challenges/:challengeId/submission/revisions/:revisionId
```

## Административный просмотр

```text
GET    /api/v1/admin/submissions
GET    /api/v1/admin/submissions/:submissionId
GET    /api/v1/admin/submissions/:submissionId/revisions
GET    /api/v1/admin/submissions/:submissionId/revisions/:revisionId
POST   /api/v1/admin/submissions/:submissionId/lock
POST   /api/v1/admin/submissions/:submissionId/unlock
GET    /api/v1/admin/submissions/:submissionId/download
GET    /api/v1/admin/submissions/export
```

## Файлы

```text
POST   /api/v1/files/init
POST   /api/v1/files/:fileId/complete
POST   /api/v1/files/:fileId/abort
GET    /api/v1/files/:fileId
GET    /api/v1/files/:fileId/download
DELETE /api/v1/files/:fileId
```

## Справка

```text
GET    /api/v1/reference/pages
GET    /api/v1/reference/pages/:slug
GET    /api/v1/admin/reference/pages
POST   /api/v1/admin/reference/pages
PATCH  /api/v1/admin/reference/pages/:pageId
POST   /api/v1/admin/reference/pages/:pageId/publish
POST   /api/v1/admin/reference/pages/:pageId/unpublish
DELETE /api/v1/admin/reference/pages/:pageId
```

## Аудит и health

```text
GET /api/v1/admin/audit
GET /api/v1/admin/audit/:eventId
GET /health/live
GET /health/ready
GET /metrics
```


---

# 21. Схема базы данных

Использовать UUID. Все даты хранить в `TIMESTAMPTZ`.

Основные таблицы первой версии:

```text
users
roles
user_roles
auth_sessions
refresh_tokens
contests
contest_admins
contest_participants
contest_challenges
challenge_fields
challenge_schema_versions
submissions
submission_revisions
files
submission_files
reference_pages
reference_page_contests
outbox_events
notification_deliveries
audit_logs
idempotency_keys
```

Будущие таблицы:

```text
lectures
lecture_sessions
attendance_records
participant_codes
point_wallets
point_transactions
merch_products
merch_variants
inventory_locations
inventory_balances
inventory_movements
merch_orders
merch_order_items
prediction_events
prediction_options
prediction_entries
prediction_settlements
```

## 21.1. `users`

```text
id UUID PK
login VARCHAR UNIQUE NOT NULL
password_hash TEXT NOT NULL
full_name TEXT NOT NULL
email CITEXT NULL
phone TEXT NULL
organization TEXT NULL
city TEXT NULL
status VARCHAR NOT NULL
must_change_password BOOLEAN NOT NULL DEFAULT TRUE
last_login_at TIMESTAMPTZ NULL
password_changed_at TIMESTAMPTZ NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
deleted_at TIMESTAMPTZ NULL
```

## 21.2. `roles`

```text
id UUID PK
code VARCHAR UNIQUE NOT NULL
name TEXT NOT NULL
created_at TIMESTAMPTZ NOT NULL
```

## 21.3. `user_roles`

```text
user_id UUID FK users
role_id UUID FK roles
scope_type VARCHAR NULL
scope_id UUID NULL
created_at TIMESTAMPTZ NOT NULL
PRIMARY KEY (user_id, role_id, scope_type, scope_id)
```

`scope_type` позволяет ограничить роль конкретным конкурсом или мероприятием.

## 21.4. `auth_sessions`

```text
id UUID PK
user_id UUID FK users
token_family_id UUID NOT NULL
user_agent TEXT NULL
ip_hash TEXT NULL
last_used_at TIMESTAMPTZ NOT NULL
expires_at TIMESTAMPTZ NOT NULL
revoked_at TIMESTAMPTZ NULL
revoke_reason TEXT NULL
created_at TIMESTAMPTZ NOT NULL
```

## 21.5. `refresh_tokens`

```text
id UUID PK
session_id UUID FK auth_sessions
jti UUID UNIQUE NOT NULL
token_hash TEXT NOT NULL
rotated_from_id UUID NULL
expires_at TIMESTAMPTZ NOT NULL
used_at TIMESTAMPTZ NULL
revoked_at TIMESTAMPTZ NULL
created_at TIMESTAMPTZ NOT NULL
```

## 21.6. `contests`

```text
id UUID PK
name TEXT NOT NULL
slug CITEXT UNIQUE NOT NULL
description TEXT NULL
status VARCHAR NOT NULL
start_at TIMESTAMPTZ NULL
end_at TIMESTAMPTZ NULL
timezone TEXT NOT NULL
settings JSONB NOT NULL DEFAULT '{}'
created_by UUID FK users
updated_by UUID FK users
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
archived_at TIMESTAMPTZ NULL
```

## 21.7. `contest_participants`

Универсальная таблица для конкурсантов, обычных участников, сотрудников и жюри.

```text
id UUID PK
contest_id UUID FK contests
user_id UUID FK users
participant_type VARCHAR NOT NULL
participant_code CITEXT NULL
metadata JSONB NOT NULL DEFAULT '{}'
joined_at TIMESTAMPTZ NOT NULL
left_at TIMESTAMPTZ NULL
UNIQUE (contest_id, user_id)
```

Типы:

```text
CONTESTANT
PARTICIPANT
STAFF
JURY
```

## 21.8. `contest_challenges`

```text
id UUID PK
contest_id UUID FK contests
title TEXT NOT NULL
slug CITEXT NOT NULL
short_description TEXT NULL
full_description TEXT NULL
instructions TEXT NULL
status VARCHAR NOT NULL
sort_order INT NOT NULL
open_at TIMESTAMPTZ NULL
deadline_at TIMESTAMPTZ NULL
close_at TIMESTAMPTZ NULL
settings JSONB NOT NULL DEFAULT '{}'
current_schema_version INT NOT NULL DEFAULT 1
created_by UUID FK users
updated_by UUID FK users
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
published_at TIMESTAMPTZ NULL
archived_at TIMESTAMPTZ NULL
UNIQUE (contest_id, slug)
```

## 21.9. `challenge_fields`

```text
id UUID PK
challenge_id UUID FK contest_challenges
field_key CITEXT NOT NULL
field_type VARCHAR NOT NULL
label TEXT NOT NULL
description TEXT NULL
help_text TEXT NULL
placeholder TEXT NULL
required BOOLEAN NOT NULL DEFAULT FALSE
sort_order INT NOT NULL
settings JSONB NOT NULL DEFAULT '{}'
validation JSONB NOT NULL DEFAULT '{}'
visibility JSONB NOT NULL DEFAULT '{}'
schema_version_from INT NOT NULL
schema_version_to INT NULL
created_by UUID FK users
updated_by UUID FK users
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
deleted_at TIMESTAMPTZ NULL
UNIQUE (challenge_id, field_key, schema_version_from)
```

## 21.10. `challenge_schema_versions`

```text
id UUID PK
challenge_id UUID FK contest_challenges
version INT NOT NULL
schema_json JSONB NOT NULL
change_summary TEXT NULL
created_by UUID FK users
created_at TIMESTAMPTZ NOT NULL
UNIQUE (challenge_id, version)
```

## 21.11. `submissions`

```text
id UUID PK
challenge_id UUID FK contest_challenges
contestant_user_id UUID FK users
status VARCHAR NOT NULL
answers_json JSONB NOT NULL DEFAULT '{}'
schema_version INT NOT NULL
version INT NOT NULL DEFAULT 1
current_revision_number INT NOT NULL DEFAULT 0
first_opened_at TIMESTAMPTZ NULL
last_saved_at TIMESTAMPTZ NULL
submitted_at TIMESTAMPTZ NULL
last_resubmitted_at TIMESTAMPTZ NULL
locked_at TIMESTAMPTZ NULL
locked_by UUID NULL
lock_reason TEXT NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
UNIQUE (challenge_id, contestant_user_id)
```

## 21.12. `submission_revisions`

```text
id UUID PK
submission_id UUID FK submissions
revision_number INT NOT NULL
action_type VARCHAR NOT NULL
schema_version INT NOT NULL
schema_snapshot JSONB NOT NULL
answers_snapshot JSONB NOT NULL
files_snapshot JSONB NOT NULL
checksum TEXT NOT NULL
created_by UUID FK users
created_at TIMESTAMPTZ NOT NULL
UNIQUE (submission_id, revision_number)
```

## 21.13. `files`

```text
id UUID PK
owner_user_id UUID FK users
contest_id UUID FK contests
challenge_id UUID NULL
submission_id UUID NULL
field_id UUID NULL
bucket TEXT NOT NULL
object_key TEXT UNIQUE NOT NULL
original_name TEXT NOT NULL
safe_name TEXT NOT NULL
extension TEXT NULL
mime_type TEXT NULL
size_bytes BIGINT NULL
checksum TEXT NULL
status VARCHAR NOT NULL
metadata JSONB NOT NULL DEFAULT '{}'
uploaded_at TIMESTAMPTZ NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
deleted_at TIMESTAMPTZ NULL
```

## 21.14. `submission_files`

```text
submission_id UUID FK submissions
file_id UUID FK files
field_id UUID NULL
sort_order INT NOT NULL DEFAULT 0
created_at TIMESTAMPTZ NOT NULL
PRIMARY KEY (submission_id, file_id)
```

## 21.15. `reference_pages`

```text
id UUID PK
slug CITEXT UNIQUE NOT NULL
title TEXT NOT NULL
content TEXT NOT NULL
content_format VARCHAR NOT NULL
status VARCHAR NOT NULL
sort_order INT NOT NULL
visible_from TIMESTAMPTZ NULL
visible_to TIMESTAMPTZ NULL
created_by UUID FK users
updated_by UUID FK users
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
published_at TIMESTAMPTZ NULL
deleted_at TIMESTAMPTZ NULL
```

## 21.16. `outbox_events`

```text
id UUID PK
event_type VARCHAR NOT NULL
aggregate_type VARCHAR NOT NULL
aggregate_id UUID NOT NULL
payload JSONB NOT NULL
status VARCHAR NOT NULL
attempts INT NOT NULL DEFAULT 0
available_at TIMESTAMPTZ NOT NULL
locked_at TIMESTAMPTZ NULL
locked_by TEXT NULL
last_error TEXT NULL
created_at TIMESTAMPTZ NOT NULL
processed_at TIMESTAMPTZ NULL
```

## 21.17. `audit_logs`

```text
id UUID PK
actor_user_id UUID NULL
action VARCHAR NOT NULL
entity_type VARCHAR NOT NULL
entity_id UUID NULL
contest_id UUID NULL
request_id UUID NULL
ip_hash TEXT NULL
user_agent TEXT NULL
before_json JSONB NULL
after_json JSONB NULL
metadata JSONB NOT NULL DEFAULT '{}'
created_at TIMESTAMPTZ NOT NULL
```

Аудит append-only.

## 21.18. Индексы

Обязательно:

```text
users(login)
users(email)
contest_participants(contest_id, participant_type)
contest_challenges(contest_id, status, sort_order)
submissions(challenge_id, status)
submissions(contestant_user_id, updated_at)
submission_revisions(submission_id, revision_number DESC)
files(submission_id, status)
files(owner_user_id, created_at DESC)
outbox_events(status, available_at)
audit_logs(contest_id, created_at DESC)
audit_logs(actor_user_id, created_at DESC)
auth_sessions(user_id, revoked_at, expires_at)
refresh_tokens(session_id, revoked_at, expires_at)
```

Для поиска ФИО можно добавить trigram index.

---

# 22. Аудит действий

Фиксировать:

- успешный и неуспешный вход;
- logout;
- refresh;
- смену и сброс пароля;
- создание и изменение пользователя;
- назначение роли;
- блокировку;
- создание и изменение конкурса;
- создание и изменение испытания;
- изменение схемы;
- публикацию;
- сохранение черновика;
- отправку;
- повторную отправку;
- блокировку формы;
- загрузку и удаление файла;
- административное скачивание;
- экспорт;
- изменение Telegram-настроек.

Не сохранять:

- пароль;
- токен;
- cookie;
- секрет;
- полное содержимое файла.

---

# 23. Экспорт

Администратор может экспортировать:

- конкурсантов;
- статусы испытаний;
- ответы;
- сводную таблицу;
- список файлов;
- журнал отправок.

Форматы:

- CSV;
- XLSX;
- ZIP;
- JSON.

Большие выгрузки выполняются worker. Для первой версии допускается синхронная выгрузка малых объёмов.

---

# 24. Будущий модуль участника

## 24.1. Назначение

Обычный участник получает:

- личный кабинет;
- персональный QR-код;
- штрихкод;
- баланс баллов;
- историю посещений;
- магазин мерча;
- историю заказов;
- доступ к прогнозам при включённом модуле.

## 24.2. Лекции

Сущность `lecture` содержит:

- название;
- описание;
- спикера;
- место;
- дату;
- начало;
- окончание;
- количество баллов;
- лимит;
- период сканирования;
- статус;
- возможность повторного посещения.

## 24.3. Сканирование

1. Сотрудник входит под ролью `SCANNER`.
2. Выбирает лекцию.
3. Открывает камеру.
4. Сканирует код участника.
5. Frontend отправляет код backend.
6. Backend проверяет код.
7. Проверяет лекцию, права, время и отсутствие посещения.
8. В одной транзакции создаёт посещение и начисляет баллы.
9. Сканер видит ФИО, фотографию при наличии и результат.

Уникальное ограничение:

```text
UNIQUE (lecture_id, participant_user_id)
```

Повторное сканирование не начисляет баллы.

## 24.4. Код участника

Первая версия будущего модуля может использовать статический код.

Архитектура должна поддерживать динамический QR:

- timestamp;
- nonce;
- подпись;
- короткий TTL;
- защита от повторного использования.

Не привязывать бизнес-логику посещаемости к одному формату кода.

Пример интерфейса:

```go
type ParticipantCodeVerifier interface {
    Verify(ctx context.Context, rawCode string) (ParticipantIdentity, error)
}
```

---

# 25. Балльная система

Баллы хранятся через ledger.

Таблицы:

```text
point_wallets
point_transactions
```

Типы операций:

```text
ATTENDANCE_REWARD
ADMIN_ADJUSTMENT
MERCH_PURCHASE
MERCH_REFUND
PREDICTION_STAKE
PREDICTION_WIN
PREDICTION_REFUND
EXPIRATION
```

Требования:

- операции immutable;
- баланс не отрицательный;
- каждая операция имеет `idempotency_key`;
- корректировка создаётся компенсирующей транзакцией;
- администратор не редактирует историю;
- все операции аудируются;
- для критичных операций используется row lock.

---

# 26. Магазин мерча

## 26.1. Сущности

```text
merch_products
merch_variants
inventory_locations
inventory_balances
inventory_movements
merch_orders
merch_order_items
merch_redemptions
```

## 26.2. Товар

- название;
- описание;
- изображения;
- цена в баллах;
- статус;
- категория;
- варианты;
- размер;
- цвет;
- остаток;
- лимит на участника;
- период доступности;
- порядок.

## 26.3. Складские операции

```text
RECEIPT
RESERVATION
ISSUE
WRITE_OFF
RETURN
CORRECTION
RELEASE_RESERVATION
```

## 26.4. Покупка

Атомарная транзакция:

1. Проверить баланс.
2. Проверить остаток.
3. Создать заказ.
4. Зарезервировать товар.
5. Списать баллы.
6. Создать ledger transaction.
7. Создать audit event.

При ошибке всё откатывается.

## 26.5. Выдача

`MERCH_MANAGER`:

- открывает заказ;
- сканирует участника;
- подтверждает выдачу;
- система переводит резерв в `ISSUE`.

## 26.6. Администрирование

- CRUD товара;
- варианты;
- остатки;
- приход;
- списание;
- инвентаризация;
- возврат;
- отмена;
- экспорт движений.

---

# 27. Прогнозы на победителей

Пользовательское название: «Прогнозы».

Используются только внутренние невыводимые баллы.

Запрещается:

- покупка баллов за деньги;
- вывод;
- денежный выигрыш;
- обмен на деньги.

Перед production-запуском требуется отдельная юридическая проверка.

Сущности:

```text
prediction_events
prediction_options
prediction_entries
prediction_settlements
```

Сценарий:

1. Администратор создаёт событие.
2. Добавляет варианты.
3. Указывает время закрытия.
4. Участник выбирает вариант и сумму внутренних баллов.
5. Баллы атомарно списываются.
6. После конкурса администратор фиксирует результат.
7. Система рассчитывает награду.
8. Баллы начисляются через ledger.
9. При отмене возвращаются.

Требования:

- settlement идемпотентный;
- закрытое событие нельзя менять;
- результат нельзя незаметно исправить;
- модуль выключен по умолчанию.

---

# 28. Feature flags

```text
FEATURE_REFERENCE_CMS=true
FEATURE_EMAIL_NOTIFICATIONS=false
FEATURE_PARTICIPANT_CABINET=false
FEATURE_ATTENDANCE=false
FEATURE_POINTS=false
FEATURE_MERCH=false
FEATURE_PREDICTIONS=false
FEATURE_JURY=false
```

Backend является источником истины. Frontend получает flags через endpoint конфигурации.

---

# 29. Конфигурация

Пример `.env.example`:

```dotenv
APP_ENV=development
APP_NAME=student-leader-cabinet
APP_BASE_URL=http://localhost:5173
API_BASE_URL=http://localhost:8080

HTTP_HOST=0.0.0.0
HTTP_PORT=8080
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=30s
HTTP_IDLE_TIMEOUT=60s

POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=student_leader
POSTGRES_USER=student_leader
POSTGRES_PASSWORD=change_me
POSTGRES_SSLMODE=disable

JWT_ISSUER=student-leader-cabinet
JWT_AUDIENCE=student-leader-web
JWT_ACCESS_SECRET=change_me
JWT_REFRESH_SECRET=change_me
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h

COOKIE_DOMAIN=localhost
COOKIE_SECURE=false
COOKIE_SAMESITE=lax

S3_ENDPOINT=http://minio:9000
S3_REGION=us-east-1
S3_BUCKET=student-leader-files
S3_ACCESS_KEY=minio
S3_SECRET_KEY=change_me
S3_USE_PATH_STYLE=true
S3_PRESIGN_TTL=15m

TELEGRAM_BOT_TOKEN=
TELEGRAM_DEFAULT_CHAT_ID=
TELEGRAM_DEFAULT_THREAD_ID=
TELEGRAM_NOTIFICATIONS_ENABLED=false

REDIS_URL=redis://redis:6379/0

MAX_JSON_BODY_MB=2
DEFAULT_MAX_FILE_SIZE_MB=2048
DEFAULT_MAX_SUBMISSION_SIZE_MB=10240

LOG_LEVEL=info
OTEL_ENABLED=false
PROMETHEUS_ENABLED=true

FEATURE_REFERENCE_CMS=true
FEATURE_PARTICIPANT_CABINET=false
FEATURE_ATTENDANCE=false
FEATURE_POINTS=false
FEATURE_MERCH=false
FEATURE_PREDICTIONS=false
```

Секреты не коммитить.

---

# 30. Docker Compose

Сервисы:

```text
frontend
backend-api
backend-worker
postgres
redis
minio
minio-init
clamav
prometheus
grafana
```

После запуска:

- ожидать PostgreSQL;
- выполнить миграции;
- создать роли;
- создать bucket;
- создать первого superadmin безопасной CLI-командой или bootstrap-переменными.

Не создавать production superadmin с публичным паролем.

---

# 31. CI/CD

Pipeline:

1. Frontend format check.
2. ESLint.
3. TypeScript check.
4. Frontend tests.
5. Frontend build.
6. Go fmt check.
7. Go vet.
8. Staticcheck.
9. Unit tests.
10. Integration tests.
11. Проверка миграций.
12. OpenAPI validation.
13. Dependency audit.
14. Docker build.
15. Image security scan.
16. Push registry.
17. Deploy staging.
18. Smoke tests.
19. Ручное подтверждение.
20. Deploy production.
21. Healthcheck.

Production-миграции выполняются отдельным job.

---

# 32. Observability

## Логи

JSON:

- timestamp;
- level;
- service;
- environment;
- request_id;
- user_id;
- route;
- method;
- status;
- duration;
- error_code.

Не логировать секреты и полное содержимое форм.

## Метрики

- HTTP count/latency;
- error rate;
- login failures;
- active sessions;
- uploads;
- upload failures;
- submissions;
- resubmissions;
- Telegram success/failure;
- outbox backlog;
- DB pool;
- worker duration.

## Алерты

- рост `5xx`;
- недоступность PostgreSQL;
- недоступность storage;
- outbox backlog;
- Telegram failures;
- ошибки миграций;
- brute force.

---

# 33. Backup

- ежедневный PostgreSQL backup;
- point-in-time recovery по возможности;
- object storage versioning;
- шифрование backup;
- документированный restore;
- регулярная проверка восстановления.

Пример хранения:

- ежедневные — 14 дней;
- еженедельные — 8 недель;
- ежемесячные — 12 месяцев.

---

# 34. Тестирование

## Unit

- password hashing;
- JWT;
- refresh rotation;
- reuse detection;
- RBAC;
- contest scope;
- schema validation;
- deadline checks;
- optimistic locking;
- revision creation;
- Telegram formatting;
- future points ledger;
- future inventory operations.

## Integration

С реальным PostgreSQL через containers:

- миграции;
- login/refresh/logout;
- revoke;
- contests;
- challenge schema;
- draft;
- submit;
- resubmit;
- outbox;
- file access;
- transactional rollback.

## E2E

1. Superadmin создаёт admin.
2. Admin создаёт конкурс.
3. Admin создаёт испытание.
4. Admin добавляет конкурсанта.
5. Конкурсант входит.
6. Меняет пароль.
7. Сохраняет черновик.
8. Загружает файлы.
9. Отправляет форму.
10. Telegram mock получает событие.
11. Admin открывает форму.
12. Конкурсант обновляет.
13. Создаётся ревизия 2.
14. Admin блокирует форму.
15. Конкурсант не может изменить.
16. Другой пользователь не получает чужие данные.

## Security

- IDOR;
- role escalation;
- подмена contest ID;
- SQL injection;
- XSS;
- CSRF;
- refresh replay;
- dangerous upload;
- path traversal;
- oversized body;
- brute force;
- CORS;
- прямой S3-доступ;
- двойной submit;
- race condition.

---

# 35. Производительность

Цели:

- обычный API p95 до 500 мс без upload;
- login p95 до 800 мс;
- список испытаний до 2 секунд;
- сохранение черновика до 1 секунды;
- presigned URL до 500 мс;
- Telegram обычно до 30 секунд;
- минимум 500 активных пользователей;
- горизонтальное масштабирование.

Состояние не хранить только в памяти одного экземпляра.

---

# 36. Accessibility и адаптивность

Обязательно независимо от `DESIGN.md`:

- mobile-first;
- iOS Safari;
- Android Chrome;
- desktop;
- keyboard navigation;
- focus state;
- label и aria;
- достаточный контраст;
- понятные ошибки;
- upload progress;
- retry;
- отсутствие потери текста при краткой ошибке сети.

---

# 37. Время и локализация

- основной язык — русский;
- хранить даты в UTC;
- отображать в timezone конкурса;
- timezone в формате IANA;
- формат интерфейса `ДД.ММ.ГГГГ ЧЧ:ММ`;
- архитектура готова к i18n.

---

# 38. UX-правила

1. Черновик не является отправкой.
2. Пользователь видит номер ревизии.
3. Незавершённая загрузка блокирует отправку.
4. Конфликт версии не должен уничтожать текст.
5. После дедлайна форма read-only, если late submission запрещён.
6. Нельзя удалить испытание с ответами — только архивировать.
7. Нельзя физически удалить пользователя с историей.
8. Нельзя удалить ревизию.
9. Telegram failure не отменяет submit.
10. Двойной запрос не создаёт две ревизии.
11. Критичные кнопки имеют loading и защиту от повторного клика.

---

# 39. Идемпотентность

Обязательна для:

- submit;
- resubmit;
- complete upload;
- Telegram delivery;
- импорт;
- будущая покупка мерча;
- посещение;
- settlement прогнозов.

Заголовок:

```text
Idempotency-Key: UUID
```

Backend сохраняет:

- key;
- user;
- endpoint;
- request hash;
- response.

Одинаковый повтор возвращает прежний результат. Тот же key с другим body возвращает `409`.

---

# 40. Конкурентный доступ

Использовать:

- транзакции;
- optimistic locking;
- unique constraints;
- `SELECT FOR UPDATE`;
- `SKIP LOCKED` для worker;
- advisory locks только при необходимости.

Не использовать глобальный mutex как основную distributed-защиту.

---

# 41. OpenAPI

Создать `backend/api/openapi.yaml`.

Он должен содержать:

- endpoints;
- DTO;
- responses;
- error codes;
- security schemes;
- pagination;
- examples;
- enums;
- upload flow;
- idempotency;
- request ID.

Frontend API client желательно генерировать из OpenAPI.

---

# 42. Seed и миграции

Seed:

- роли;
- permissions;
- шаблон справки;
- шаблон технического задания;
- development users только в development;
- безопасный bootstrap superadmin.

Миграции:

- хранятся в Git;
- имеют timestamp;
- тестируются на пустой и актуальной БД;
- опасные изменения — expand/contract;
- без разрушительных изменений в одной версии;
- forward-first подход.


---

# 43. Административные таблицы и фильтры

Все административные таблицы должны поддерживать:

- серверную пагинацию;
- сортировку;
- поиск;
- фильтрацию;
- хранение фильтров в URL;
- выбор числа строк;
- экспорт текущей выборки;
- loading skeleton;
- empty state;
- error state;
- адаптивное отображение.

Основные таблицы:

- конкурсы;
- конкурсные испытания;
- конкурсанты;
- формы;
- ревизии;
- файлы;
- аудит;
- Telegram deliveries;
- будущие лекции;
- будущие складские операции;
- будущие балльные транзакции.

---

# 44. Внутренние уведомления интерфейса

Показывать понятные уведомления:

- черновик сохранён;
- автосохранение выполнено;
- файл загружен;
- файл отклонён;
- форма отправлена;
- форма обновлена;
- создана новая ревизия;
- возник конфликт версии;
- сессия истекла;
- форма заблокирована;
- дедлайн приближается;
- Telegram-уведомление не влияет на успешность отправки.

В будущем можно добавить таблицу `in_app_notifications`.

---

# 45. Политика архивирования

Soft delete использовать для:

- пользователей;
- конкурсов;
- испытаний;
- полей;
- справочных страниц;
- файлов.

Не удалять физически:

- submission revisions;
- audit logs;
- attendance records;
- point transactions;
- inventory movements;
- prediction settlements.

При необходимости удаления персональных данных предусмотреть анонимизацию без разрушения связанной истории.

---

# 46. Требования к `DESIGN.md`

Этот документ не определяет финальный визуальный стиль.

Claude Code обязан:

1. Проверить наличие `DESIGN.md`.
2. Прочитать его до начала frontend-реализации.
3. Использовать его как источник истины по:
   - цветам;
   - типографике;
   - spacing;
   - grid;
   - компонентам;
   - состояниям;
   - адаптивности;
   - анимациям;
   - иконкам;
   - tone of voice.
4. Не придумывать визуальные решения, противоречащие `DESIGN.md`.
5. При отсутствии решения использовать нейтральный доступный компонент.
6. Не смешивать UI tokens с бизнес-логикой.
7. Создать единый слой design tokens.
8. Не дублировать стили по страницам.
9. Обеспечить единые состояния `hover`, `focus`, `disabled`, `loading`, `error`, `success`.

---

# 47. План реализации

## Этап 0. Инициализация

- создать monorepo;
- добавить frontend;
- добавить backend;
- Docker Compose;
- PostgreSQL;
- Redis;
- MinIO;
- конфигурацию;
- Makefile;
- линтеры;
- CI;
- базовый OpenAPI;
- README.

## Этап 1. Авторизация

- users;
- roles;
- scoped permissions;
- login;
- JWT access;
- refresh rotation;
- session management;
- logout;
- logout all;
- change password;
- must change password;
- middleware;
- frontend guards;
- audit.

## Этап 2. Конкурсы и конкурсанты

- CRUD конкурса;
- scoped admin access;
- contest participants;
- добавление конкурсанта;
- reset password;
- блокировка;
- списки;
- фильтры;
- import/export skeleton.

## Этап 3. Испытания и конструктор

- challenge CRUD;
- dynamic fields;
- help text;
- field validation;
- schema versioning;
- reorder;
- preview;
- publish;
- deadlines;
- archive.

## Этап 4. Формы

- draft;
- server validation;
- optimistic locking;
- submit;
- resubmit;
- immutable revisions;
- history;
- admin view;
- lock/unlock;
- status filters.

## Этап 5. Файлы

- MinIO;
- presigned upload;
- multipart upload;
- progress;
- complete/abort;
- MIME validation;
- access control;
- download;
- soft delete;
- scanner interface.

## Этап 6. Telegram и worker

- outbox;
- worker;
- retry;
- dead letter;
- templates;
- Telegram settings;
- delivery history;
- metrics.

## Этап 7. Справочная информация

- CMS;
- Markdown или rich text;
- sanitization;
- публикация;
- привязка к конкурсу;
- вложения.

## Этап 8. Аудит и экспорт

- audit endpoints;
- audit UI;
- filters;
- CSV;
- XLSX;
- ZIP;
- background export.

## Этап 9. Hardening

- security tests;
- E2E;
- load tests;
- monitoring;
- backup;
- restore test;
- staging;
- production documentation.

## Этап 10. Будущие модули

- participant cabinet;
- lectures;
- attendance scanner;
- point ledger;
- merch;
- predictions;
- jury.

---

# 48. Definition of Done

Функция завершена только если:

- backend реализован;
- frontend реализован;
- backend authorization добавлена;
- серверная валидация добавлена;
- миграция создана;
- OpenAPI обновлён;
- unit tests добавлены;
- integration tests добавлены;
- E2E обновлён при необходимости;
- аудит добавлен;
- ошибки обработаны;
- loading state добавлен;
- empty state добавлен;
- адаптивность проверена;
- accessibility проверена;
- документация обновлена;
- secrets отсутствуют в Git;
- CI проходит;
- staging smoke test проходит.

---

# 49. Acceptance criteria

## 49.1. Авторизация

- пользователь входит по логину и паролю;
- неправильный пароль возвращает нейтральную ошибку;
- access token имеет короткий TTL;
- refresh token ротируется;
- старый refresh token нельзя использовать повторно;
- reuse detection отзывает token family;
- logout завершает текущую сессию;
- logout all завершает все сессии;
- заблокированный пользователь не входит;
- временный пароль требует смены;
- protected API недоступен без авторизации;
- protected frontend route не открывается без сессии.

## 49.2. Роли

- конкурсант не открывает admin endpoints;
- администратор видит только назначенные конкурсы;
- конкурсант видит только свои формы;
- подмена ID не даёт доступ;
- superadmin имеет полный доступ;
- доступ к файлу проверяется отдельно от знания URL.

## 49.3. Испытания

- можно создать любое количество испытаний;
- можно создать любое количество полей;
- у каждого поля есть справка;
- поля можно сортировать;
- типы полей динамические;
- есть preview;
- схема версионируется;
- опубликованное испытание отображается конкурсанту;
- архивное не отображается;
- дедлайн учитывается backend.

## 49.4. Черновики

- конкурсант сохраняет форму;
- данные сохраняются между сессиями;
- можно продолжить позже;
- server validation работает;
- автосохранение не создаёт Telegram notification;
- конфликт вкладок возвращает `409`;
- пользователь не теряет локальные данные при конфликте.

## 49.5. Отправка и ревизии

- обязательные поля проверяются;
- незавершённые файлы блокируют submit;
- первая отправка создаёт revision 1;
- обновление создаёт revision 2;
- revision 1 остаётся доступной;
- checksum сохраняется;
- одинаковый idempotent request не создаёт дубль;
- Telegram failure не откатывает submit;
- admin видит историю.

## 49.6. Файлы

- поддерживается несколько файлов;
- поддерживаются разные типы;
- отображается progress;
- неуспешную загрузку можно повторить;
- dangerous file обрабатывается безопасно;
- размер проверяется;
- чужой файл недоступен;
- bucket закрыт;
- presigned URL ограничен по времени;
- файл прошлой ревизии сохраняется.

## 49.7. Telegram

- отправка создаёт событие;
- обновление создаёт событие;
- черновик не создаёт событие;
- worker повторяет попытки;
- событие не дублируется;
- неуспешное событие видно администратору;
- после лимита попыток статус `DEAD`.

## 49.8. Справочная страница

- конкурсант открывает отдельную страницу;
- admin редактирует контент;
- draft не виден;
- published виден;
- вложения защищены;
- HTML sanitised.

## 49.9. Будущее расширение

Архитектура считается пригодной к расширению, если:

- один пользователь может иметь несколько scoped roles;
- конкурсант может позднее стать участником;
- point ledger не зависит от одной причины начисления;
- attendance создаёт балльную транзакцию атомарно;
- merch использует общий wallet;
- predictions используют общий ledger;
- модули включаются feature flags;
- существующие contest modules не нужно переписывать.

---

# 50. API-ошибки

Стабильные коды:

```text
VALIDATION_ERROR
AUTH_INVALID_CREDENTIALS
AUTH_SESSION_EXPIRED
AUTH_REFRESH_REUSED
AUTH_ACCOUNT_BLOCKED
AUTH_PASSWORD_CHANGE_REQUIRED
FORBIDDEN
CONTEST_ACCESS_DENIED
RESOURCE_NOT_FOUND
SUBMISSION_LOCKED
SUBMISSION_DEADLINE_PASSED
SUBMISSION_VERSION_CONFLICT
SUBMISSION_ALREADY_SUBMITTED
UPLOAD_NOT_COMPLETED
FILE_TYPE_NOT_ALLOWED
FILE_TOO_LARGE
IDEMPOTENCY_CONFLICT
RATE_LIMIT_EXCEEDED
TELEGRAM_DELIVERY_FAILED
INTERNAL_ERROR
```

Frontend не должен зависеть от текста сообщения. Логика строится по `error.code`.

---

# 51. Pagination

Формат запроса:

```text
?page=1&page_size=20&sort=updated_at:desc
```

Ответ:

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 134,
    "total_pages": 7
  },
  "request_id": "uuid"
}
```

Установить максимальный `page_size`.

Для очень больших журналов допускается cursor pagination.

---

# 52. Формат сохранения формы

Пример запроса:

```json
{
  "version": 7,
  "schema_version": 3,
  "answers": {
    "project_name": "Проект",
    "comment": "Обширный комментарий",
    "screen_required": true,
    "screen_resolution": "1920x1080",
    "sound_notes": "Запустить трек после выхода участника"
  },
  "file_ids": [
    "f9078f9e-9b24-4c68-a71e-a2d9d4587aaa",
    "e4ea4b55-3f38-412f-ae1c-4d87a9022bbb"
  ]
}
```

Ответ:

```json
{
  "data": {
    "submission_id": "uuid",
    "status": "DRAFT",
    "version": 8,
    "schema_version": 3,
    "last_saved_at": "2026-07-12T13:45:00Z"
  },
  "request_id": "uuid"
}
```

---

# 53. События домена

Предусмотреть domain/application events:

```text
UserCreated
UserBlocked
PasswordReset
ContestCreated
ChallengePublished
DraftSaved
SubmissionSubmitted
SubmissionResubmitted
SubmissionLocked
FileUploaded
FileRejected
ReferencePagePublished
AttendanceRecorded
PointsGranted
MerchOrderCreated
MerchIssued
PredictionPlaced
PredictionSettled
```

Не обязательно внедрять сложный event bus в первой версии. Достаточно транзакционного outbox для внешних side effects.

---

# 54. Структура репозитория

```text
student-leader-cabinet/
  README.md
  REQUIREMENTS.md
  DESIGN.md
  docker-compose.yml
  .env.example
  .gitignore
  Makefile

  docs/
    architecture.md
    deployment.md
    backup-restore.md
    security.md
    API.md
    ADR/
      0001-modular-monolith.md
      0002-jwt-session-model.md
      0003-dynamic-form-storage.md
      0004-file-storage.md
      0005-outbox.md

  backend/
    cmd/
      api/
      worker/
    internal/
      app/
      config/
      middleware/
      platform/
      modules/
    db/
      migrations/
      queries/
    api/
      openapi.yaml
    tests/
      integration/
      e2e/
    Dockerfile
    go.mod
    go.sum

  frontend/
    src/
      app/
      pages/
      widgets/
      features/
      entities/
      shared/
    public/
    tests/
    Dockerfile
    package.json
    package-lock.json
    vite.config.ts
    tsconfig.json

  infra/
    nginx/
    prometheus/
    grafana/
    scripts/

  .github/
    workflows/
      ci.yml
      deploy-staging.yml
      deploy-production.yml
```

---

# 55. Команды

Предусмотреть:

```bash
make dev
make up
make down
make logs
make migrate-up
make migrate-down
make seed
make test
make test-unit
make test-integration
make test-e2e
make lint
make format
make build
make openapi
make create-superadmin
make backup
make restore
```

---

# 56. Инструкции для Claude Code

## 56.1. Перед началом

Claude Code должен:

1. Полностью прочитать `REQUIREMENTS.md`.
2. Полностью прочитать `DESIGN.md`.
3. Проверить существующий код.
4. Не удалять рабочую реализацию без необходимости.
5. Создать implementation plan.
6. Зафиксировать архитектурные решения в ADR.
7. Реализовывать этапами.
8. После каждого этапа запускать проверки.

## 56.2. Правила реализации

- Не делать mock вместо production-кода в основном приложении.
- Не хранить бизнес-данные только во frontend.
- Не доверять frontend validation.
- Не использовать `localStorage` для refresh token.
- Не делать bucket публичным.
- Не хранить файлы в PostgreSQL.
- Не логировать токены и пароли.
- Не хардкодить роли по всему проекту.
- Не создавать God object.
- Не использовать `panic` для обычных ошибок.
- Возвращать стабильные error codes.
- Критичные операции выполнять транзакционно.
- Administrative actions писать в audit.
- Permissions проверять на backend.
- Даты хранить в UTC.
- API документировать в OpenAPI.
- Новые модули покрывать тестами.
- Не подключать будущие модули к production UI до включения feature flag.
- Не создавать таблицы будущих модулей без необходимости, но сохранить границы модулей и спецификацию.
- Не менять `DESIGN.md` без отдельного запроса.
- Не использовать незафиксированные зависимости.
- Не оставлять `TODO` в критической безопасности.
- Не делать silent fallback для ошибок авторизации.
- Не считать успешным upload до server-side complete.
- Не позволять admin незаметно менять конкурсантскую ревизию.
- Не удалять audit и revisions.

## 56.3. Порядок генерации

1. Monorepo.
2. Docker Compose.
3. Backend skeleton.
4. Frontend skeleton.
5. PostgreSQL migrations.
6. Auth.
7. RBAC и scopes.
8. Contests.
9. Contestants.
10. Challenges.
11. Dynamic schema builder.
12. Submissions.
13. Revisions.
14. Files.
15. Outbox.
16. Telegram.
17. Reference CMS.
18. Admin UI.
19. Contestant UI.
20. Audit.
21. Export.
22. Tests.
23. OpenAPI.
24. CI/CD.
25. Production docs.

## 56.4. Формат отчёта после этапа

После каждого этапа Claude должен перечислить:

- добавленные файлы;
- изменённые файлы;
- миграции;
- endpoints;
- UI-страницы;
- тесты;
- команды запуска;
- известные ограничения;
- следующий этап.

## 56.5. Запрет на генерацию всего проекта одним непроверяемым блоком

Claude должен создавать проект итеративно:

- небольшой логический этап;
- запуск lint;
- запуск tests;
- исправление;
- краткий отчёт;
- следующий этап.

---

# 57. Начальный prompt для Claude Code

Можно использовать следующий prompt после помещения этого файла в корень проекта:

```text
Прочитай полностью REQUIREMENTS.md и DESIGN.md.

Твоя задача — реализовать production-ready веб-приложение Student Leader Cabinet поэтапно.

Сначала:
1. Проанализируй требования.
2. Проверь текущую структуру репозитория.
3. Создай docs/architecture.md.
4. Создай краткие ADR для модульного монолита, JWT session model, динамических форм, файлового хранилища и outbox.
5. Составь детальный implementation plan по этапам.
6. Не начинай будущие модули attendance, points, merch и predictions, пока не завершён основной конкурсный модуль.
7. Не меняй визуальную концепцию из DESIGN.md.
8. Не создавай весь проект одним большим ответом.

После плана начни с этапа 0:
- monorepo;
- Docker Compose;
- PostgreSQL;
- Redis;
- MinIO;
- backend skeleton;
- frontend skeleton;
- Makefile;
- .env.example;
- healthchecks;
- базовый CI.

После каждого этапа:
- запускай форматирование;
- запускай lint;
- запускай тесты;
- перечисляй изменённые файлы;
- указывай команды проверки;
- не переходи дальше при падающих проверках.
```

---

# 58. Итоговый архитектурный принцип

Первая версия должна быть простой для поддержки, но не одноразовой.

Ключевые решения:

- модульный монолит;
- React + Vite + TypeScript;
- Go backend;
- PostgreSQL;
- scoped RBAC;
- JWT access/refresh с rotation;
- динамические формы;
- schema versioning;
- immutable revisions;
- S3 object storage;
- presigned upload;
- outbox для Telegram;
- audit trail;
- OpenAPI contract;
- Docker-first development;
- feature flags;
- ledger для будущих баллов;
- транзакционный склад;
- единый пользователь для ролей конкурсанта, участника, сотрудника и жюри.

Система должна развиваться без создания нескольких несовместимых личных кабинетов и без необходимости переписывать основной конкурсный модуль.
