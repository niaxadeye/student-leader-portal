// Конфиг pm2 для api/worker на проде (заменяет infra/systemd/*.service).
// Запуск: pm2 startOrRestart ecosystem.config.js && pm2 save
// pm2 v7 `env_file` не подхватывает переменные для fork-процессов — .env грузим
// сами через обёртки infra/pm2/run-{api,worker}.sh (source .env + exec бинаря).
module.exports = {
  apps: [
    {
      name: 'eazytech-api',
      cwd: '/var/www/student-leader-portal/backend',
      script: '/var/www/student-leader-portal/infra/pm2/run-api.sh',
      autorestart: true,
      max_restarts: 10,
      restart_delay: 5000,
    },
    {
      name: 'eazytech-worker',
      cwd: '/var/www/student-leader-portal/backend',
      script: '/var/www/student-leader-portal/infra/pm2/run-worker.sh',
      autorestart: true,
      max_restarts: 10,
      restart_delay: 5000,
    },
  ],
}
