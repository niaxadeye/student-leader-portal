-- Роль STAFF: сотрудник мероприятия без прав ADMIN.
-- Операционные права выдаются строками event_staff_permissions на конкретный конкурс.
INSERT INTO roles (code, name) VALUES ('STAFF', 'Сотрудник мероприятия')
    ON CONFLICT (code) DO NOTHING;
