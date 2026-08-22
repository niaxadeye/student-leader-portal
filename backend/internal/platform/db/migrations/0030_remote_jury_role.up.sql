-- Заочное жюри: отдельная роль на конкурс, не входит в список live-жюри.
INSERT INTO roles (code, name) VALUES ('REMOTE_JURY', 'Заочное жюри')
    ON CONFLICT (code) DO NOTHING;

-- Кто уже назначен на заочные испытания — в пул заочного жюри этого конкурса.
INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
SELECT DISTINCT a.user_id, r.id, 'CONTEST', a.contest_id
FROM evaluation_staff_assignments a
JOIN evaluation_schemes s ON s.challenge_id = a.challenge_id AND s.active AND s.type = 'REMOTE_CRITERIA'
JOIN roles r ON r.code = 'REMOTE_JURY'
WHERE a.active AND a.role = 'JURY'
ON CONFLICT DO NOTHING;
