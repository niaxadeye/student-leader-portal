-- Роли и назначения со scope конкурса (SITE.md §21.2–21.3, §5, §6).
CREATE TABLE roles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(64) UNIQUE NOT NULL,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- scope_type/scope_id ограничивают роль конкретным конкурсом (ADMIN конкурса A).
-- Для глобальных ролей scope_type='GLOBAL', scope_id — нулевой UUID (часть PK не может быть NULL).
CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    scope_type VARCHAR(32) NOT NULL DEFAULT 'GLOBAL', -- GLOBAL | CONTEST
    scope_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id, scope_type, scope_id)
);

CREATE INDEX idx_user_roles_user ON user_roles (user_id);

-- Сид базовых ролей (SITE.md §5).
INSERT INTO roles (code, name) VALUES
    ('SUPER_ADMIN', 'Суперадминистратор'),
    ('ADMIN',       'Администратор конкурса'),
    ('CONTESTANT',  'Конкурсант');
