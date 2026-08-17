# Техническое задание: модуль платформы проведения мероприятий

## 1. Контекст задачи

Нужно интегрировать новый функционал в **уже существующую информационную систему**, в которой уже есть:

- мероприятия;
- группы;
- существующие пользователи системы;
- существующая авторизация сотрудников/администраторов;
- существующая ролевая модель и/или permissions;
- существующий frontend/backend;
- существующая база данных;
- существующие conventions проекта.

Нельзя создавать параллельные дублирующие сущности `User`, `Event`, `Group`, если они уже существуют в проекте.

Новый функционал должен быть встроен в текущую архитектуру максимально аккуратно и backward-compatible.

---

# 2. Главная архитектурная идея

У существующей системы остаются её текущие пользователи — сотрудники, администраторы, организаторы и модераторы.

Для участников мероприятий необходимо создать **отдельную сущность**, которая не является обычным `User`.

Основная новая сущность:

```text
EventParticipant
```

Каждый участник всегда принадлежит конкретному мероприятию:

```text
Event
 ├── EventParticipant
 ├── Lecture
 ├── Task
 └── MerchProduct
```

На текущем этапе один и тот же физический человек, зарегистрированный на двух разных мероприятиях, может существовать как две разные записи `EventParticipant`.

Не нужно автоматически объединять участников между мероприятиями.

---

# 3. Что необходимо сделать перед написанием кода

Перед внесением изменений необходимо сначала исследовать существующий проект.

Нужно определить:

1. существующую модель `User`;
2. существующую модель `Event`;
3. существующую модель `Group`;
4. текущий механизм authentication;
5. текущий механизм authorization;
6. роли и permissions;
7. используемый ORM;
8. структуру базы данных;
9. текущие API conventions;
10. текущий механизм загрузки файлов;
11. текущий механизм генерации slug;
12. структуру frontend;
13. структуру backend;
14. существующие middleware / guards;
15. существующие conventions ошибок API;
16. существующую систему audit/logging, если есть.

После исследования проекта:

1. показать найденные точки интеграции;
2. перечислить существующие модели, которые будут использованы;
3. перечислить новые модели;
4. перечислить существующие файлы, которые потребуется изменить;
5. перечислить новые файлы;
6. предложить migration plan;
7. только после этого переходить к реализации.

Не переписывать проект целиком.

---

# 4. EventParticipant

Создать отдельную сущность участника мероприятия.

Пример:

```ts
EventParticipant {
    id: UUID

    eventId: UUID

    fullName: string
    fullNameNormalized: string

    birthDate: date

    unionCardNumber: string | null
    sksBarcode: string | null

    status:
        ACTIVE
        BLOCKED
        ARCHIVED

    createdAt
    updatedAt
}
```

`eventId` должен ссылаться на существующую сущность `Event`.

## 4.1. Уникальность

Если значение указано, необходимо обеспечить уникальность:

```text
eventId + unionCardNumber
```

и:

```text
eventId + sksBarcode
```

Связка:

```text
fullName + birthDate
```

не должна быть уникальной на уровне БД, так как теоретически могут существовать разные люди с одинаковыми ФИО и датой рождения.

---

# 5. Импорт участников

Для каждого мероприятия должен существовать свой список участников.

Минимальные данные:

- ФИО;
- дата рождения;
- номер профсоюзного билета — optional;
- barcode СКС РФ — optional.

Нужен раздел:

```text
/admin/events/:eventId/participants
```

Функции:

- просмотр участников;
- поиск;
- фильтрация;
- добавление вручную;
- редактирование;
- блокировка;
- архивирование;
- импорт CSV/XLSX;
- экспорт.

После импорта необходимо показать результат:

```text
Добавлено: N
Обновлено: N
Ошибок: N
Дубликатов: N
```

Ошибочные строки не должны молча игнорироваться.

Нужно возвращать причины ошибок по строкам.

---

# 6. Авторизация участников

Участник авторизуется только внутри конкретного мероприятия.

Пример страницы:

```text
/event/:eventSlug/login
```

Участник может войти одним из трёх способов.

## 6.1. ФИО + дата рождения

Проверяется:

```text
eventId
+
normalizedFullName
+
birthDate
```

ФИО необходимо нормализовать перед сравнением:

- trim;
- множественные пробелы → один пробел;
- привести к единому регистру;
- желательно учитывать `ё` и `е`.

Если найден один участник — авторизация успешна.

Если найдено несколько совпадений — не авторизовать автоматически.

Показать сообщение с предложением использовать:

- номер профсоюзного билета;
- barcode СКС РФ.

## 6.2. Номер профсоюзного билета

Поиск:

```text
eventId + unionCardNumber
```

## 6.3. Barcode СКС РФ

Поиск:

```text
eventId + sksBarcode
```

Поле barcode должно поддерживать:

- ручной ввод;
- USB scanner в HID keyboard режиме.

---

# 7. Participant authentication

Не использовать существующую авторизацию `User` для участников.

Должно существовать два независимых auth flow:

```text
StaffAuth
ParticipantAuth
```

После успешной авторизации участника сервер создаёт participant session.

Пример:

```ts
ParticipantSession {
    id
    eventParticipantId
    eventId
    createdAt
    expiresAt
    lastActivityAt
}
```

Предпочтительно использовать secure session cookie:

```text
HttpOnly
Secure
SameSite
```

Не хранить sensitive session token в localStorage, если архитектура проекта позволяет избежать этого.

---

# 8. Личный кабинет участника

Пример route:

```text
/event/:eventSlug/me
```

Участник должен видеть:

- ФИО;
- текущий баланс баллов;
- зарезервированные баллы;
- доступные баллы;
- кнопку «Мой QR»;
- активные задания;
- выполненные задания;
- задания на проверке;
- магазин;
- выбранный товар, на который он копит;
- свои заказы;
- историю посещённых лекций;
- при необходимости историю операций по баллам.

Пример:

```text
Баланс: 1500
В резерве: 500
Доступно: 1000
```

---

# 9. Система баллов

Не использовать только поле:

```text
participant.points
```

как единственный источник истины.

Основной источник данных — immutable журнал операций:

```text
PointsLedger
```

Пример:

```ts
PointsLedger {
    id

    eventParticipantId
    eventId

    amount: int

    type:
        LECTURE_ATTENDANCE
        TASK_REWARD
        MERCH_PURCHASE
        ADMIN_ADJUSTMENT
        REFUND

    sourceType
    sourceId

    description

    createdByUserId | null

    idempotencyKey

    createdAt
}
```

Примеры:

```text
+100  Лекция "Медиа будущего"
+250  Задание "Сделать публикацию"
-500  Покупка футболки
```

Баланс должен рассчитываться на основании ledger.

Допускается кеширование вычисленного баланса для производительности, но `PointsLedger` остаётся источником истины.

---

# 10. Резервирование баллов

Для заказов мерча необходимо реализовать:

```text
PointsHold
```

Пример:

```ts
PointsHold {
    id

    eventParticipantId
    orderId

    amount

    status:
        ACTIVE
        CAPTURED
        RELEASED

    createdAt
    updatedAt
}
```

Доступные баллы:

```text
availablePoints =
ledgerBalance - activeHolds
```

Пример:

```text
Баланс:              1500
Зарезервировано:      500
Доступно:            1000
```

Это необходимо, чтобы пользователь не смог одновременно оформить несколько заказов на одни и те же баллы.

---

# 11. Лекции

Создать сущность:

```ts
Lecture {
    id
    eventId

    title
    description | null

    points

    startsAt | null
    endsAt | null

    attendanceStartsAt | null
    attendanceEndsAt | null

    status:
        DRAFT
        ACTIVE
        FINISHED

    createdAt
    updatedAt
}
```

Лекция обязательно принадлежит мероприятию.

За посещение лекции участнику начисляется заданное количество баллов.

---

# 12. QR участника

У участника должна быть страница:

```text
/event/:eventSlug/me/qr
```

Не помещать в QR обычный `participantId`.

Плохой вариант:

```text
https://site.ru/checkin/12345
```

Нужно использовать короткоживущий signed token.

Пример payload:

```json
{
  "eventParticipantId": "...",
  "eventId": "...",
  "exp": 1786940000,
  "nonce": "..."
}
```

Token должен:

- подписываться сервером;
- иметь короткий срок жизни;
- проверяться сервером;
- быть связан с participant и event.

Рекомендуемый TTL:

```text
30–60 секунд
```

QR на странице участника должен автоматически обновляться.

Не доверять данным из QR без серверной проверки подписи и срока жизни.

---

# 13. Сканирование посещения

Организатор должен иметь scanner page:

```text
/admin/events/:eventId/lectures/:lectureId/scanner
```

Необходимо поддерживать два способа сканирования.

## 13.1. Камера телефона

Организатор открывает scanner page на телефоне.

Web-приложение получает доступ к камере и считывает QR.

После успешного чтения token отправляется на backend:

```http
POST /api/events/:eventId/lectures/:lectureId/attendance/scan
```

Пример:

```json
{
    "token": "SIGNED_QR_TOKEN"
}
```

Успех:

```text
✓ Иванов Иван Иванович
+100 баллов
```

Повторное сканирование:

```text
⚠ Уже отмечен
Посещение зарегистрировано ранее
```

## 13.2. USB 2D QR scanner

Scanner должен поддерживать стандартный HID keyboard mode.

На desktop scanner page должно быть поле ввода, которое:

- постоянно получает focus;
- принимает строку от сканера;
- после `Enter` автоматически отправляет token;
- очищается после обработки;
- возвращает focus;
- готово к следующему сканированию.

Не требуется специальный native driver, если устройство работает как HID keyboard.

---

# 14. LectureAttendance

Создать:

```ts
LectureAttendance {
    id

    lectureId
    eventParticipantId

    scannedByUserId

    scannerType:
        CAMERA
        USB
        MANUAL

    pointsAwarded

    createdAt
}
```

Обязательно создать DB-level unique constraint:

```text
UNIQUE(lectureId, eventParticipantId)
```

Участник может получить баллы за одну лекцию только один раз.

Создание attendance и начисление баллов должно выполняться одной DB transaction.

Пример:

```text
BEGIN

create LectureAttendance
create PointsLedger +100

COMMIT
```

При повторном запросе не должно происходить повторное начисление.

---

# 15. Задания

Создать:

```ts
Task {
    id
    eventId

    title
    description

    image | null
    icon | null

    points

    startsAt | null
    endsAt | null

    status:
        DRAFT
        ACTIVE
        DISABLED
        ARCHIVED

    sortOrder

    createdAt
    updatedAt
}
```

У задания:

- название;
- картинка;
- иконка;
- описание;
- количество баллов;
- optional дата начала;
- optional дата окончания.

Допустимы варианты:

```text
startsAt = null
endsAt = null
```

или:

```text
startsAt != null
endsAt = null
```

или:

```text
startsAt = null
endsAt != null
```

или обе даты.

---

# 16. Подтверждение выполнения задания

Участник может подтверждать выполнение:

- одним изображением;
- несколькими изображениями;
- скриншотом;
- серией скриншотов;
- ссылкой;
- изображениями + ссылкой.

У задания желательно предусмотреть конфигурацию допустимых типов submission.

Пример:

```ts
allowedSubmissionTypes: [
    IMAGE,
    LINK
]
```

---

# 17. TaskSubmission

Создать:

```ts
TaskSubmission {
    id

    taskId
    eventParticipantId

    status:
        PENDING
        APPROVED
        REJECTED

    participantComment | null
    moderatorComment | null

    reviewedByUserId | null

    submittedAt
    reviewedAt | null
}
```

Файлы / ссылки вынести отдельно:

```ts
TaskSubmissionAsset {
    id

    submissionId

    type:
        IMAGE
        LINK

    url
    sortOrder
}
```

Не создавать поля вида:

```text
image1
image2
image3
```

---

# 18. Модерация заданий

Создать staff page:

```text
/admin/events/:eventId/tasks/moderation
```

Модератор видит очередь `PENDING`.

Нужно показывать:

- участника;
- название задания;
- время отправки;
- изображения;
- ссылки;
- комментарий участника.

Действия:

```text
Принять
Отклонить
```

При `APPROVED`:

```text
TaskSubmission:
PENDING -> APPROVED

PointsLedger:
+N TASK_REWARD
```

Операция должна выполняться транзакционно.

Повторное одобрение не должно повторно начислять баллы.

---

# 19. Повторная отправка задания

Если submission имеет статус:

```text
REJECTED
```

участник должен иметь возможность выполнить повторную отправку.

При отклонении модератор может оставить комментарий.

Пример:

```text
Отклонено

Причина:
На скриншоте не видно публикацию.

[Отправить заново]
```

После успешного `APPROVED` участник не может повторно получить reward за то же задание.

Необходимо обеспечить это DB-level ограничением / idempotency mechanism.

---

# 20. Магазин мерча

Создать сущность:

```ts
MerchProduct {
    id: UUID

    eventId

    title
    slug

    description

    pricePoints
    discountPricePoints | null

    stockQuantity
    reservedQuantity

    status:
        DRAFT
        ACTIVE
        HIDDEN
        SOLD_OUT

    createdAt
    updatedAt
}
```

Поля:

- название;
- id — автогенерация;
- slug — автогенерация;
- стоимость в баллах;
- стоимость со скидкой;
- изображение/изображения;
- описание;
- общее количество;
- количество в резерве;
- количество желающих.

`slug` должен автоматически генерироваться с учётом существующих conventions проекта.

Если slug уже существует — обеспечить уникальный вариант.

---

# 21. Изображения товара

Создать отдельную таблицу:

```ts
MerchProductImage {
    id
    productId
    url
    sortOrder
}
```

Это позволит иметь произвольное количество изображений.

---

# 22. Остатки товара

Основные поля:

```text
stockQuantity
reservedQuantity
```

Доступный остаток:

```text
availableQuantity =
stockQuantity - reservedQuantity
```

Пример:

```text
На складе:        20
В резерве:         7
Можно заказать:   13
```

Нельзя резервировать больше:

```text
stockQuantity - reservedQuantity
```

---

# 23. Товар, на который пользователь копит

Участник может выбрать **только одну** позицию магазина как цель накопления.

Создать:

```ts
MerchSavingTarget {
    id

    eventParticipantId
    productId

    createdAt
}
```

Ограничение:

```text
UNIQUE(eventParticipantId)
```

Если участник выбирает другую позицию, предыдущая цель заменяется.

В личном кабинете можно показывать:

```text
Вы копите на:
Худи мероприятия

Цена:
1500

Ваш баланс:
900

Осталось:
600

Прогресс:
60%
```

---

# 24. Количество желающих

Не хранить как вручную редактируемое число.

Количество желающих должно рассчитываться как:

```text
COUNT(MerchSavingTarget WHERE productId = ...)
```

Допускается кеширование для производительности.

Источник истины — `MerchSavingTarget`.

---

# 25. Заказы мерча

Создать:

```ts
MerchOrder {
    id

    eventId
    eventParticipantId

    status:
        RESERVED
        ISSUED
        REJECTED
        CANCELLED

    pointsTotal

    rejectionReason | null

    createdAt
    issuedAt | null
    rejectedAt | null
    cancelledAt | null

    issuedByUserId | null
    rejectedByUserId | null
}
```

Товары заказа:

```ts
MerchOrderItem {
    id

    orderId
    productId

    quantity

    pricePoints
    totalPoints
}
```

Цена должна сохраняться в `MerchOrderItem` на момент оформления.

Изменение цены товара позднее не должно менять уже созданный заказ.

---

# 26. Создание заказа

Оформление заказа является критической операцией и должно выполняться одной DB transaction.

Алгоритм:

```text
BEGIN TRANSACTION

1. Проверить participant session.
2. Проверить event.
3. Заблокировать необходимые stock records на время транзакции.
4. Проверить доступный остаток товаров.
5. Рассчитать актуальную стоимость.
6. Рассчитать доступные баллы:
   ledgerBalance - activePointsHolds
7. Проверить достаточность баллов.
8. Создать MerchOrder со статусом RESERVED.
9. Создать MerchOrderItem.
10. Увеличить reservedQuantity.
11. Создать PointsHold со статусом ACTIVE.

COMMIT
```

Если какого-либо товара не хватает — не создавать частичный заказ, если специально не реализован режим partial fulfillment.

По умолчанию заказ должен быть atomic: либо резервируются все позиции, либо ни одна.

Не доверять:

- цене с frontend;
- discountPrice с frontend;
- quantity availability с frontend;
- points balance с frontend.

Всё пересчитывать сервером.

---

# 27. Выдача заказа

Staff page:

```text
/admin/events/:eventId/merch/orders
```

Администратор видит:

- номер заказа;
- участника;
- товары;
- количество;
- стоимость;
- статус;
- время оформления.

Действия:

```text
[Выдано]
[Отказ]
```

---

# 28. Статус «Выдано»

При нажатии:

```text
RESERVED -> ISSUED
```

выполнить в одной transaction:

```text
Order:
RESERVED -> ISSUED

Для каждого товара:
stockQuantity -= quantity
reservedQuantity -= quantity

PointsHold:
ACTIVE -> CAPTURED

PointsLedger:
-N MERCH_PURCHASE
```

Не допускается повторное списание баллов при повторном запросе.

---

# 29. Статус «Отказ»

Причина отказа обязательна или configurable в зависимости от conventions проекта.

Пример:

```text
Фактический остаток товара не совпал с системой
```

Transaction:

```text
Order:
RESERVED -> REJECTED

Для каждого товара:
reservedQuantity -= quantity

PointsHold:
ACTIVE -> RELEASED
```

Баллы снова становятся доступны участнику.

---

# 30. Отмена заказа участником

Если требуется поддержать отмену до получения, можно использовать:

```text
RESERVED -> CANCELLED
```

При отмене:

```text
reservedQuantity -= quantity
PointsHold ACTIVE -> RELEASED
```

Не разрешать отмену после `ISSUED`.

---

# 31. Permissions сотрудников

Использовать существующую permission/role систему проекта.

Не создавать новую систему ролей без необходимости.

Пример permissions:

```text
event.participants.manage

event.attendance.scan
event.attendance.manage

event.tasks.manage
event.tasks.moderate

event.merch.manage
event.merch.orders.manage

event.points.manage
```

Пример назначения:

```text
Организатор:
event.attendance.scan

Модератор:
event.tasks.moderate

Сотрудник мерч-зоны:
event.merch.orders.manage

Администратор:
все permissions
```

Permissions должны проверяться server-side.

Также обязательно проверять, имеет ли сотрудник доступ именно к конкретному `eventId`.

---

# 32. Participant routes

Предлагаемая структура:

```text
/event/:eventSlug/login

/event/:eventSlug/me

/event/:eventSlug/me/qr

/event/:eventSlug/tasks

/event/:eventSlug/tasks/:taskId

/event/:eventSlug/shop

/event/:eventSlug/shop/:slug

/event/:eventSlug/orders

/event/:eventSlug/orders/:orderId
```

Адаптировать под conventions текущего проекта.

---

# 33. Staff routes

Пример:

```text
/admin/events/:eventId/participants

/admin/events/:eventId/lectures

/admin/events/:eventId/lectures/:lectureId

/admin/events/:eventId/lectures/:lectureId/scanner

/admin/events/:eventId/tasks

/admin/events/:eventId/tasks/moderation

/admin/events/:eventId/merch

/admin/events/:eventId/merch/orders

/admin/events/:eventId/points
```

---

# 34. Пример API

Адаптировать под текущую архитектуру API проекта.

## Participant auth

```http
POST /api/events/:eventId/participant-auth/fio
POST /api/events/:eventId/participant-auth/union-card
POST /api/events/:eventId/participant-auth/sks

POST /api/participant/logout
GET  /api/participant/me
```

## Participant QR

```http
GET /api/participant/qr
```

## Attendance

```http
POST /api/events/:eventId/lectures/:lectureId/attendance/scan
GET  /api/events/:eventId/lectures/:lectureId/attendance
```

## Tasks

```http
GET  /api/participant/tasks
GET  /api/participant/tasks/:taskId

POST /api/participant/tasks/:taskId/submissions

GET  /api/events/:eventId/task-submissions

POST /api/events/:eventId/task-submissions/:id/approve
POST /api/events/:eventId/task-submissions/:id/reject
```

## Merch

```http
GET /api/participant/merch
GET /api/participant/merch/:slug

PUT    /api/participant/merch-saving-target
DELETE /api/participant/merch-saving-target

POST /api/participant/orders
GET  /api/participant/orders
GET  /api/participant/orders/:id
```

## Staff merch

```http
GET /api/events/:eventId/merch/orders

POST /api/events/:eventId/merch/orders/:id/issue
POST /api/events/:eventId/merch/orders/:id/reject
```

---

# 35. Конкурентность и idempotency

Критические операции должны быть защищены не только frontend-проверками.

Необходимо учитывать следующие случаи:

## 35.1. Два организатора одновременно сканируют одного участника

Баллы за лекцию должны начислиться один раз.

Использовать:

```text
UNIQUE(lectureId, eventParticipantId)
```

и transaction.

## 35.2. Два модератора одновременно одобряют submission

Баллы должны начислиться один раз.

Нужна transaction + conditional state transition + DB constraint / idempotency.

## 35.3. Два пользователя покупают последний товар

Нельзя создать два резерва на одну последнюю единицу.

Использовать row locking / transactional atomic update / механизм, подходящий текущему ORM и БД.

## 35.4. Повторная отправка HTTP-запроса

Критические mutation endpoints должны быть idempotent либо устойчивы к retry.

Особенно:

- attendance scan;
- task approval;
- order create;
- order issue;
- order reject;
- points adjustment.

---

# 36. Работа с файлами заданий

Скриншоты и изображения участников не должны без необходимости становиться публичными.

Желательно:

```text
private storage
+
signed temporary URLs
```

Проверять:

- MIME type;
- расширение;
- размер;
- количество файлов;
- реальные content type / magic bytes, если инфраструктура позволяет.

Минимально поддержать:

```text
JPEG
PNG
WEBP
```

PDF добавить при необходимости.

Нужно ограничить максимальный размер файла и количество файлов на submission.

---

# 37. URL submissions

Если участник отправляет ссылку:

- валидировать URL;
- разрешить только `http://` и `https://`;
- не выполнять сервером произвольные unsafe requests к URL без отдельной необходимости;
- учитывать SSRF risks, если когда-либо появится server-side preview/fetch.

---

# 38. Audit log

Если в проекте уже есть audit log — использовать его.

Если нет — предусмотреть сущность вида:

```ts
AuditLog {
    id

    eventId

    actorUserId | null
    eventParticipantId | null

    action

    entityType
    entityId

    oldValue | null
    newValue | null

    createdAt
}
```

Особенно логировать:

- ручное изменение баллов;
- ручное добавление attendance;
- удаление / отмену attendance;
- approval задания;
- reject задания;
- создание/изменение товара;
- выдачу заказа;
- отказ заказа;
- отмену заказа;
- изменение stockQuantity.

---

# 39. Ручная корректировка баллов

Для администратора желательно предусмотреть:

```text
/admin/events/:eventId/points
```

Можно вручную:

```text
+100
-50
```

Но нельзя напрямую редактировать баланс.

Создавать:

```text
PointsLedger
type = ADMIN_ADJUSTMENT
```

Обязательно хранить:

- кто сделал изменение;
- сколько;
- причину;
- время.

---

# 40. Безопасность

Обязательно:

1. Все authorization checks выполнять server-side.
2. Не доверять `eventId`, пришедшему с frontend, без проверки доступа.
3. Не доверять стоимости товара с frontend.
4. Не доверять количеству баллов с frontend.
5. Не доверять stock с frontend.
6. Не доверять роли/permission с frontend.
7. Проверять ownership participant session.
8. Проверять связь всех entities с одним event.
9. Использовать CSRF protection, если это требуется текущей auth архитектурой.
10. Использовать rate limit на participant login endpoints.
11. Не раскрывать лишние данные при ошибке авторизации.
12. Не включать дату рождения / номер профбилета / barcode в QR.
13. Не использовать предсказуемые идентификаторы как credential.

---

# 41. Rate limiting participant auth

Поскольку участник может входить по персональным данным, endpoint авторизации необходимо защитить от brute force / enumeration.

Добавить rate limit по комбинации:

- IP;
- event;
- session/device — при возможности.

Ошибка не должна позволять легко определять:

```text
существует ли такой профсоюзный билет
```

или:

```text
существует ли barcode
```

---

# 42. Работа со временем

Все даты в БД хранить согласно conventions проекта, предпочтительно UTC.

Frontend отображает timezone мероприятия / пользователя.

Особенно важно для:

- startsAt;
- endsAt;
- attendanceStartsAt;
- attendanceEndsAt;
- сроков заданий;
- QR expiration;
- createdAt;
- reviewedAt.

---

# 43. Удаление данных

По возможности использовать soft-delete / archive для сущностей, которые уже участвовали в операциях.

Например нельзя физически удалять товар, если существуют:

```text
MerchOrderItem
```

Лучше:

```text
status = ARCHIVED / HIDDEN
```

Нельзя физически удалять:

- `PointsLedger`;
- завершённые `LectureAttendance`;
- завершённые `MerchOrder`;
- важные audit records.

---

# 44. Индексы БД

После определения реальных query patterns добавить индексы минимум для:

```text
EventParticipant(eventId)
EventParticipant(eventId, unionCardNumber)
EventParticipant(eventId, sksBarcode)
EventParticipant(eventId, fullNameNormalized, birthDate)

Lecture(eventId)

LectureAttendance(lectureId, eventParticipantId)
LectureAttendance(eventParticipantId)

Task(eventId)
TaskSubmission(taskId, eventParticipantId)
TaskSubmission(status)

PointsLedger(eventParticipantId, createdAt)

PointsHold(eventParticipantId, status)

MerchProduct(eventId, slug)
MerchSavingTarget(eventParticipantId)
MerchSavingTarget(productId)

MerchOrder(eventParticipantId)
MerchOrder(eventId, status)
MerchOrderItem(orderId)
```

Адаптировать индексы под используемую СУБД.

---

# 45. Пользовательский интерфейс участника

Интерфейс должен быть mobile-first.

Основные экраны:

## Главная

```text
Иванов Иван Иванович

★ 1350 баллов

[Мой QR]

Задания
Магазин
Мои заказы
Мои посещения
```

## Задания

Карточка:

```text
[ICON] [IMAGE]

Сделать публикацию

До 18 августа

+300 баллов

[Подробнее]
```

Статусы:

```text
Доступно
На проверке
Выполнено
Отклонено
Завершено
```

## Магазин

```text
Худи

1500 баллов
1200 баллов со скидкой

Доступно: 5

[Копить]
[Добавить в заказ]
```

## Моя цель

```text
Вы копите на:
Худи мероприятия

1200 / 1500

Осталось 300
```

## Заказы

```text
Заказ #...

Футболка × 1

500 баллов

Статус:
Ждёт получения
```

---

# 46. Staff UI для scanner

Scanner UI должен быть очень быстрым.

На экране постоянно отображать:

- название лекции;
- количество уже отмеченных участников;
- состояние камеры / scanner input;
- результат последнего сканирования.

Успех:

```text
✓ Иванов Иван Иванович
+100 баллов
```

Повтор:

```text
⚠ Уже отмечен
14:03
```

Ошибка:

```text
✕ QR недействителен
```

Желательно:

- визуальный feedback;
- короткий звук success/error, если допустимо;
- автоматическая готовность к следующему scan без дополнительных кликов.

---

# 47. Staff UI модерации

Очередь должна позволять быстро обрабатывать submissions.

Фильтры:

- мероприятие;
- задание;
- статус;
- дата;
- участник.

Карточка submission:

```text
Иванов Иван Иванович

Задание:
Опубликовать пост

[изображения]

https://...

Комментарий участника

[Принять]
[Отклонить]
```

При отклонении показать поле причины.

---

# 48. Staff UI магазина

Раздел товара:

- название;
- slug;
- изображения;
- описание;
- базовая цена;
- скидочная цена;
- количество;
- в резерве;
- доступно;
- количество желающих;
- статус.

Количество желающих рассчитывается автоматически.

Раздел заказов:

- RESERVED;
- ISSUED;
- REJECTED;
- CANCELLED.

Фильтры по статусам.

---

# 49. Рекомендуемая модель связей

Высокоуровнево:

```text
EXISTING SYSTEM

User ──────────────── Event ─────────────── Group
                         │
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
 EventParticipant      Lecture           Task
        │                │                │
        │          LectureAttendance   TaskSubmission
        │                                 │
        │                         TaskSubmissionAsset
        │
        ├────────────── PointsLedger
        │
        ├────────────── PointsHold
        │
        ├────────────── MerchSavingTarget
        │
        └────────────── MerchOrder
                              │
                              ▼
                       MerchOrderItem
                              │
                              ▼
                         MerchProduct
                              │
                              ▼
                      MerchProductImage
```

---

# 50. Основные бизнес-инварианты

Эти правила должны соблюдаться всегда, независимо от frontend.

1. `EventParticipant` принадлежит ровно одному `Event`.
2. Participant не является обычным `User`.
3. Все participant операции scoped по `eventId`.
4. Баллы за одну лекцию начисляются максимум один раз.
5. Баллы за одно задание начисляются максимум один раз.
6. Баланс не редактируется напрямую.
7. Все изменения баллов проходят через `PointsLedger`.
8. Активные merch заказы используют `PointsHold`.
9. Нельзя потратить зарезервированные баллы повторно.
10. Нельзя зарезервировать товара больше доступного stock.
11. Stock reservation и PointsHold создаются атомарно.
12. Order issue и списание баллов происходят атомарно.
13. Order rejection освобождает товар и баллы атомарно.
14. Цена заказа фиксируется на момент создания заказа.
15. Пользователь может иметь только одну active saving target.
16. Количество желающих рассчитывается по saving targets.
17. QR не содержит открытые персональные данные.
18. QR token короткоживущий и подписанный.
19. Все staff actions проходят server-side authorization.
20. Все критические действия логируются.

---

# 51. Migration requirements

Необходимо:

- добавить новые таблицы через migrations;
- не удалять существующие данные;
- не менять существующие primary keys без крайней необходимости;
- не ломать существующие API;
- не ломать существующую auth систему;
- обеспечить backward compatibility.

Перед применением migration проверить:

- nullable/default значения;
- foreign keys;
- indexes;
- unique constraints;
- cascade rules.

Не использовать опасные cascade delete для финансово значимых сущностей.

---

# 52. Testing requirements

Добавить unit/integration tests согласно текущему стеку проекта.

Минимально проверить:

## Participant auth

- вход по ФИО + дате;
- вход по профбилету;
- вход по barcode;
- неправильные данные;
- duplicate ФИО + дата;
- blocked participant.

## Attendance

- успешное посещение;
- повторное сканирование;
- expired QR;
- invalid QR;
- QR другого event;
- concurrent scan.

## Tasks

- submission;
- approval;
- rejection;
- resubmission;
- duplicate approval;
- concurrent approval;
- начисление reward только один раз.

## Merch

- успешный reserve;
- недостаточно баллов;
- недостаточно товара;
- concurrent purchase последней единицы;
- issue;
- reject;
- release points;
- release stock;
- повторный issue;
- повторный reject.

## Permissions

- staff без permission;
- staff другого event;
- participant попытался вызвать staff API.

---

# 53. Порядок реализации

Рекомендуется выполнять поэтапно.

## Этап 1. Исследование проекта

- модели;
- auth;
- permissions;
- event architecture;
- file storage;
- DB;
- API conventions.

## Этап 2. EventParticipant + participant auth

- модель;
- migration;
- импорт;
- login;
- session;
- кабинет.

## Этап 3. PointsLedger

- ledger;
- balance service;
- admin adjustments.

## Этап 4. Lectures + attendance

- Lecture;
- QR;
- mobile camera scanner;
- USB scanner;
- attendance;
- rewards.

## Этап 5. Tasks

- Task;
- submission;
- assets;
- moderation;
- reward.

## Этап 6. Merch

- products;
- images;
- saving target;
- orders;
- stock reservation;
- PointsHold;
- issue/reject.

## Этап 7. Audit + hardening

- audit;
- concurrency;
- rate limit;
- permissions;
- security;
- tests.

---

# 54. Требование к процессу работы Codex

Не начинать с массовой генерации файлов.

Сначала:

1. исследовать существующую кодовую базу;
2. определить стек;
3. найти существующие сущности;
4. найти auth flow;
5. найти event ownership/access logic;
6. найти file upload/storage;
7. найти conventions проекта;
8. предоставить краткий integration plan;
9. после этого вносить изменения небольшими логическими блоками.

После каждого крупного этапа:

- проверить build;
- проверить lint;
- проверить typecheck;
- запустить релевантные tests;
- исправить ошибки перед переходом дальше.

Не оставлять систему в состоянии, где старый функционал перестал работать.

---

# 55. Важные запреты

Не делать следующее без явной необходимости:

- не создавать второй `Event`;
- не создавать второй staff `User`;
- не переносить всех существующих пользователей на новую auth систему;
- не переписывать весь backend;
- не переписывать весь frontend;
- не хранить только число `points`;
- не хранить balance как единственный источник истины;
- не списывать баллы окончательно при обычном reserve;
- не доверять price/points/stock с frontend;
- не использовать participant ID как публичный QR credential;
- не начислять reward только frontend-логикой;
- не делать уникальность только application-level;
- не использовать public URLs для приватных submission файлов без необходимости;
- не удалять финансово значимые записи физически;
- не выполнять критические multi-step операции без transaction.

---

# 56. Возможные дальнейшие расширения

Архитектура должна позволять позднее добавить без полного переписывания:

- рейтинги участников;
- рейтинги групп/команд;
- достижения;
- уровни;
- бейджи;
- задания для отдельных групп;
- задания для отдельных категорий участников;
- расписание;
- push notifications;
- NFC check-in;
- бейджи с QR;
- динамические ограничения выдачи мерча;
- лимиты товаров на участника;
- варианты товара, размеры, цвета;
- промокоды;
- достижения за серии посещений;
- leaderboard;
- экспорт статистики;
- dashboard мероприятия.

Не реализовывать эти возможности сейчас, если их нет в задаче, но не проектировать текущую архитектуру так, чтобы они стали невозможны.

---

# 57. Definition of Done

Работа считается завершённой, когда:

1. новый функционал интегрирован в существующую систему;
2. существующие Event/User/Group не продублированы;
3. создан EventParticipant;
4. работает participant auth тремя способами;
5. работает participant session;
6. работает личный кабинет;
7. работает динамический QR;
8. работает camera scan;
9. работает USB scanner flow;
10. attendance начисляет баллы один раз;
11. работает PointsLedger;
12. работает Task + submission;
13. работает moderation;
14. task reward начисляется один раз;
15. работает merch catalog;
16. работает saving target;
17. работает stock reservation;
18. работает PointsHold;
19. заказ можно выдать;
20. заказ можно отклонить;
21. при reject освобождаются товар и баллы;
22. критические операции транзакционны;
23. permissions проверяются server-side;
24. реализованы необходимые DB constraints;
25. migrations безопасны;
26. build/typecheck/lint проходят;
27. релевантные tests проходят;
28. старый функционал продолжает работать.

---

# 58. Финальная инструкция Codex

Сначала исследуй существующий проект и не предполагай стек или структуру заранее.

Используй текущие архитектурные conventions проекта везде, где это возможно.

Не создавай новую параллельную архитектуру без необходимости.

После исследования сначала предоставь короткий список:

```text
Existing entities/services to reuse
New entities to add
Existing files to modify
New files to create
Migrations
Security/concurrency risks
Implementation order
```

После этого приступай к реализации по этапам.

При любых расхождениях между данным ТЗ и реальной архитектурой существующего проекта отдавай приоритет:

1. сохранению существующей архитектуры;
2. backward compatibility;
3. DB integrity;
4. transaction safety;
5. server-side authorization;
6. описанным бизнес-инвариантам.
