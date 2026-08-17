import { z } from 'zod'

export const participantNameLoginSchema = z.object({
  full_name: z.string().trim().min(3, 'Введите ФИО').max(200, 'ФИО слишком длинное'),
  birth_date: z
    .string()
    .min(1, 'Укажите дату рождения')
    .refine((value) => {
      const date = new Date(`${value}T00:00:00`)
      return !Number.isNaN(date.getTime()) && date <= new Date()
    }, 'Проверьте дату рождения'),
})

export const participantIdentifierLoginSchema = z.object({
  value: z.string().trim().min(1, 'Заполните поле').max(160, 'Значение слишком длинное'),
})

export type ParticipantNameLoginValues = z.infer<typeof participantNameLoginSchema>
export type ParticipantIdentifierLoginValues = z.infer<typeof participantIdentifierLoginSchema>
