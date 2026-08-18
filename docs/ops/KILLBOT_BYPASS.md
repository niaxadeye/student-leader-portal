# P0.3. KillBot перехватывает apex `eazytech.ru`

Дата проверки: **2026-08-18**. DNS и WAF **не менялись** — это предложение, не инструкция к немедленному применению.

## Наблюдение

| URL | DNS A | Ответ |
|---|---|---|
| `https://eazytech.ru/health/ready` | `31.192.108.184`, `109.236.57.58` | `200 text/html` (страница проверки, не API) |
| `https://eazytech.ru/api/v1/config` | те же | `200 text/html` |
| `https://www.eazytech.ru/health/ready` | `201.51.29.205` | `200 application/json` `{"data":{"status":"ready"}}` |
| `http://127.0.0.1:8080/health/ready` | localhost | JSON, как ожидается |

Apex-домен резолвится на адреса KillBot/WAF. `www` и origin сервера отвечают корректно. Клиенты и smoke-скрипты, ходящие на `https://eazytech.ru/api/*`, получают HTML с кодом 200 — ложный «успех».

## Что не делать без явного разрешения

- Менять DNS A/AAAA/CNAME у регистратора.
- Отключать KillBot целиком на весь домен.
- Менять cookie-domain / CORS вслепую.

## Рекомендуемый bypass (согласовать с владельцем DNS/WAF)

1. В панели KillBot/WAF исключить из JS-challenge:
   - `/api/`
   - `/health/`
   - `/assets/` (hashed SPA)
   - при необходимости `/.well-known/acme-challenge/`
2. Оставить challenge на `/` (HTML-оболочка SPA), если защита лендинга нужна.
3. После bypass проверить с внешней сети:
   ```bash
   ./infra/ops/check-public-api.sh https://eazytech.ru
   ./infra/ops/check-public-api.sh https://www.eazytech.ru
   ```
4. Отдельно прогнать login + refresh на apex: cookie `Domain=eazytech.ru`, CORS `APP_BASE_URL`, SameSite.

## Nginx на origin (этот сервер)

Проксировать не только `/api/`, но и `/health/` — health живёт вне `/api/v1`:

```nginx
location /health/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
}

location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    client_max_body_size 1024m;
}
```

Конфиг nginx в репозитории не хранится. Правка на сервере — отдельное разрешение.

## Критерий готовности

Публичный `GET /health/ready` и `GET /api/v1/config` с apex:

- HTTP 200
- `Content-Type: application/json`
- тело содержит `"status":"ready"` / объект `features`
- не HTML
