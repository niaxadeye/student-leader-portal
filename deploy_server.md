# Развёртывание Student Leader Cabinet на новом сервере

Перенос прода: домен **eazytech.ru** переезжает на этот сервер. Старый сервер
пока продолжает работать — DNS переключается последним шагом. Установка
**с чистой базой** — переноса данных с старого сервера нет, миграции
применяются с нуля.

Репозиторий: `https://github.com/niaxadeye/student-leader-portal`
(если приватный — потребуется deploy-ключ/токен на этом сервере).

## 0. Предпосылки

- Debian/Ubuntu-сервер с root/sudo, публичный IP.
- DNS: A-запись `eazytech.ru` (и при необходимости `www.eazytech.ru`) должна
  указывать на IP **этого** сервера — без этого certbot и CORS/cookie-домен
  не заработают. Если DNS ещё не переключён, шаги 1–8 можно выполнить и
  проверить локально (см. §9 «Проверка до переключения DNS»), а переключение
  сделать последним.
- Порты 80/443 открыты снаружи. Порты Postgres/Redis/MinIO — только на
  `127.0.0.1` (так уже настроено в `docker-compose.yml`, наружу не открывать).

Установить системные зависимости:

```bash
# Docker + Docker Compose plugin
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker "$USER"   # перелогиниться после этого

# Go >= 1.25 (см. backend/go.mod)
# скачать нужный архив с https://go.dev/dl/ и распаковать в /usr/local,
# либо через менеджер версий (goenv/asdf) — на выбор исполнителя.
go version   # должно быть >= 1.25.0

# Node.js >= 20 + npm
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# pm2 (менеджер процессов для api/worker — см. ecosystem.config.js)
sudo npm install -g pm2

# nginx + certbot
sudo apt-get install -y nginx certbot python3-certbot-nginx
```

## 1. Код

```bash
sudo mkdir -p /var/www/student-leader-portal
sudo chown "$USER":"$USER" /var/www/student-leader-portal
git clone https://github.com/niaxadeye/student-leader-portal /var/www/student-leader-portal
cd /var/www/student-leader-portal
```

Путь **обязательно** `/var/www/student-leader-portal` — он захардкожен в
`ecosystem.config.js`, `infra/pm2/run-{api,worker}.sh` и `deploy.sh`. Если
нужен другой путь, поправь эти файлы соответственно.

## 2. Секреты (`.env`)

```bash
cp .env.example .env
```

Секреты копируются со старого сервера (значения не должны попадать в git,
чат или markdown-файлы) — перенеси их напрямую между серверами, например:

```bash
# со старого сервера на новый, по SSH (пример)
scp old-server:/var/www/student-leader-portal/.env /var/www/student-leader-portal/.env
```

Если прямого scp между серверами нет — скопируй `.env` через промежуточный
защищённый канал (не через issue/чат/PR). Ключи, которые обязательно должны
совпадать со старым сервером (иначе отвалятся активные сессии/шифрование):

- `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`
- `POSTGRES_PASSWORD`
- `S3_ACCESS_KEY`, `S3_SECRET_KEY`
- `TELEGRAM_BOT_TOKEN`, `TELEGRAM_DEFAULT_CHAT_ID`, `TELEGRAM_DEFAULT_THREAD_ID`
- `BOOTSTRAP_SUPERADMIN_LOGIN`, `BOOTSTRAP_SUPERADMIN_PASSWORD`

Проверь остальные значения — при переносе на тот же домен они не должны
меняться относительно `.env.example`:

- `APP_BASE_URL=https://eazytech.ru`, `API_BASE_URL=https://eazytech.ru`
- `COOKIE_DOMAIN=eazytech.ru`, `COOKIE_SECURE=true`
- `POSTGRES_PORT=5433` — только если на сервере system-Postgres уже занял
  5432 (как на старом сервере). Если 5432 свободен, можно использовать его
  и убрать кастомный порт — но тогда поменяй и `docker-compose.yml`.

## 3. Инфраструктура (Postgres/Redis/MinIO)

```bash
docker compose up -d
docker compose ps   # все три сервиса должны быть healthy
```

Если порт 5432 или 6379 на сервере уже занят (система БД/Redis) — создай
`docker-compose.override.yml` по образцу локальной разработки (файл в
`.gitignore`, на сервере создаётся вручную), меняя маппинг портов, и
синхронизируй `POSTGRES_PORT`/`REDIS_URL` в `.env`.

`minio-init` в составе `docker compose up` создаёт S3-бакет автоматически —
отдельный шаг не нужен.

## 4. Сборка backend

```bash
cd backend
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker
go build -o bin/admin ./cmd/admin
```

## 5. Миграции + bootstrap-аккаунт

```bash
cd /var/www/student-leader-portal
set -a; . ./.env; set +a
./backend/bin/admin migrate
./backend/bin/admin create-megaadmin   # логин/пароль из BOOTSTRAP_SUPERADMIN_*
```

`create-megaadmin` идемпотентна — при повторном запуске обновит пароль того
же логина, не создаст дубликат.

## 6. Сборка frontend

```bash
cd frontend
npm ci
npm run build
cd ..
```

Собранная статика — `frontend/dist/`, её раздаёт nginx (см. §7).

## 7. nginx + TLS

Конфиг nginx не хранится в репозитории — создаётся на сервере вручную.
Пример (`/etc/nginx/sites-available/eazytech.ru`):

```nginx
server {
    listen 80;
    server_name eazytech.ru www.eazytech.ru;

    root /var/www/student-leader-portal/frontend/dist;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/eazytech.ru /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# TLS — только после того, как DNS указывает на этот сервер
sudo certbot --nginx -d eazytech.ru -d www.eazytech.ru
```

## 8. Запуск api/worker (pm2)

```bash
cd /var/www/student-leader-portal
pm2 startOrRestart ecosystem.config.js
pm2 save
pm2 startup   # включить автозапуск pm2 при перезагрузке сервера, выполнить команду, которую он выведет
```

Проверка:

```bash
pm2 status                      # eazytech-api и eazytech-worker — online
curl -fsS http://127.0.0.1:8080/health/ready
```

## 9. Проверка до переключения DNS

Пока DNS ещё указывает на старый сервер, проверить новый можно так:

```bash
curl -fsS -H "Host: eazytech.ru" http://<IP-нового-сервера>/api/v1/health/ready 2>&1 || true
curl -fsS http://127.0.0.1:8080/health/ready   # напрямую на сервере
```

HTTPS/certbot на этом шаге не проверить (сертификат ещё не выпущен без DNS) —
это нормально, certbot запускается после переключения DNS.

## 10. Переключение DNS и финальная проверка

1. Обновить A-запись `eazytech.ru` на IP нового сервера (у регистратора/DNS-провайдера, вне зоны ответственности этого сервера).
2. Подождать распространения DNS (`dig eazytech.ru` с разных резолверов).
3. Выпустить TLS-сертификат (§7, `certbot --nginx`), если не сделан заранее.
4. Полная проверка:
   - `https://eazytech.ru` открывается, фронт грузится.
   - Логин через UI под bootstrap-аккаунтом (`BOOTSTRAP_SUPERADMIN_LOGIN`).
   - `pm2 logs eazytech-api --lines 50` — без ошибок.
5. Остановить сервисы на старом сервере (`pm2 stop ecosystem.config.js` там),
   но не удалять — оставить как резерв на случай отката DNS.

## Дальнейшие обновления

После первого разворачивания повторные деплои делаются через `./deploy.sh`
из корня репозитория на сервере — он сам делает `git pull`, пересборку,
миграции и `pm2 startOrRestart` с health-check.
