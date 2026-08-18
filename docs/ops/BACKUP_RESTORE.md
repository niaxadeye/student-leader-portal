# P0.4. Резервное копирование PostgreSQL

Скрипты лежат в репозитории. **Cron/timer не включались** — это отдельное инфраструктурное разрешение.

## Целевое состояние

- Ежедневный `pg_dump` контейнера `slc-postgres`.
- Сжатие gzip; опционально GnuPG.
- Копия вне хоста (S3/отдельный диск).
- Retention 7–14 дней на хосте.
- Мониторинг свежести: файл младше 36 часов.
- Регулярный restore-тест на копии.

## Что есть сейчас

- БД: Docker volume `pgdata` (`docker-compose.yml`), порт `127.0.0.1:5433`.
- Скрипт дампа: `infra/backup/pg_dump.sh`
- Скрипт restore (на остановленной БД/отдельном инстансе): `infra/backup/pg_restore.sh`

Секреты берутся из `/var/www/student-leader-portal/.env`. Скрипты не печатают пароль.

## Ручной прогон (можно сейчас)

```bash
sudo mkdir -p /var/backups/student-leader-portal
sudo chown root:root /var/backups/student-leader-portal
sudo chmod 700 /var/backups/student-leader-portal
/var/www/student-leader-portal/infra/backup/pg_dump.sh
```

Опционально шифрование: задать `BACKUP_GPG_RECIPIENT` в окружении (fingerprint ключа).

## Предложение по расписанию (не применять без разрешения)

systemd timer, не root-cron с паролем в unit:

```
# /etc/systemd/system/slc-pg-backup.service
[Service]
Type=oneshot
User=root
ExecStart=/var/www/student-leader-portal/infra/backup/pg_dump.sh

# /etc/systemd/system/slc-pg-backup.timer
[Timer]
OnCalendar=*-*-* 03:15:00
Persistent=true
```

Off-host: после дампа копировать `*.sql.gz` / `*.sql.gz.gpg` в отдельный бакет Timeweb (не тот же, что файлы конкурсантов) с отдельными ключами.

S3-объекты конкурса: отдельная политика lifecycle + inventory; не смешивать с pg_dump.

## Restore runbook

1. Остановить API/worker: `pm2 stop eazytech-api eazytech-worker`
2. Восстановить на **копии** или после явного ок на проде:
   ```bash
   /var/www/student-leader-portal/infra/backup/pg_restore.sh /var/backups/student-leader-portal/pg_....sql.gz
   ```
3. `./bin/admin migrate` (идемпотентно, если дамп уже после миграций — no-op)
4. `pm2 start eazytech-api eazytech-worker`
5. `curl -fsS http://127.0.0.1:8080/health/ready` — JSON `status=ready`
6. Логин bootstrap-аккаунтом.

Restore на живой volume без остановки API даст порчу соединений — не делать.

## Тест восстановления

Раз в квартал: поднять второй postgres-контейнер на другом порту, залить последний дамп, проверить `SELECT count(*) FROM users`.
