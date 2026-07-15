# Frontend — Student Leader Cabinet (UI-фаза)

Прототип интерфейса на боевом стеке (React + Vite + TS + Tailwind + shadcn-подход).
Работает на mock-данных — бэкенд подключается позже без переписывания UI.

## Запуск

```bash
cd frontend
npm install
npm run dev      # http://localhost:5173
```

## Проверки

```bash
npm run build    # tsc + vite build
npm run lint
npm run format
```

## Что реализовано (3 экрана для утверждения)

- `/login` — вход, нейтральная ошибка, редирект на смену пароля (пароль `temp`)
- `/contestant` — дашборд: конкурс, дедлайн, статистика, список испытаний
- `/contestant/challenges/:id` — динамическая форма (Экран/Свет/Звук), автосейв,
  загрузка файлов с прогрессом, подтверждение отправки, ревизии, состояния

Design tokens (DESIGN.md) — `src/app/styles/tokens.css` + `tailwind.config.ts`.
UI-кит — `src/shared/ui/`. Демо-схема формы — `src/shared/api/mock/`.
