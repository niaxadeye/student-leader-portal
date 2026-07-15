# ADR 0002. JWT session-модель

## Статус
Принято.

## Контекст
Нужна безопасная авторизация с возможностью отзыва сессий и защитой от кражи токенов (§16).

## Решение
- Access token: короткий TTL (15m), хранится только в памяти frontend.
- Refresh token: длинный TTL, в HttpOnly/Secure/SameSite cookie, в БД только хэш.
- Ротация refresh + token family + reuse detection: повторное использование старого
  токена отзывает всю семью.
- Сессии в `auth_sessions`, токены в `refresh_tokens`.

## Последствия
- XSS не даёт доступ к refresh (не в localStorage).
- Возможен logout-all и отзыв отдельных сессий.
- Требуется CSRF-защита для cookie-endpoints (SameSite + токен + проверка Origin).
