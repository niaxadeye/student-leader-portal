import { z } from 'zod'

// Мин. длина 10 — синхронно с бэкендом (auth.minPasswordLen).
export const changePasswordSchema = z
  .object({
    old_password: z.string().min(1, 'Введите текущий пароль'),
    new_password: z.string().min(10, 'Минимум 10 символов'),
    confirm: z.string().min(1, 'Повторите пароль'),
  })
  .refine((v) => v.new_password === v.confirm, {
    path: ['confirm'],
    message: 'Пароли не совпадают',
  })

export type ChangePasswordValues = z.infer<typeof changePasswordSchema>
