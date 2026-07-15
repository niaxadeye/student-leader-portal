import type { Challenge } from '@/entities/challenge/types'
import { selfPresentationFields } from './challenge-fields'
import { lightSoundFields } from './challenge-fields-2'

export const mockContest = {
  id: 'c1',
  name: 'Студенческий лидер 2026',
  status: 'ACTIVE' as const,
}

export const mockUser = {
  full_name: 'Иванова Анна Сергеевна',
  organization: 'Московский Политех',
  login: 'a.ivanova',
}

// В days-часах от «сейчас», чтобы демо всегда было актуальным.
const inDays = (d: number) => new Date(Date.now() + d * 86_400_000).toISOString()

export const mockChallenges: Challenge[] = [
  {
    id: 'ch1',
    contestId: 'c1',
    title: 'Самопрезентация',
    short_description: 'Техническое задание на выступление: экран, свет, звук.',
    full_description:
      'Подготовьте техническое задание для вашего сценического выступления. Укажите требования к экрану, свету и звуку, загрузите презентацию и видеоматериалы.',
    instructions:
      'Заполните все обязательные поля. Материалы принимаются до дедлайна. После отправки редактирование разрешено.',
    status: 'PUBLISHED',
    deadline_at: inDays(1.5),
    allow_edit_after_submission: true,
    allow_late_submission: false,
    schema_version: 3,
    fields: [...selfPresentationFields, ...lightSoundFields],
  },
  {
    id: 'ch2',
    contestId: 'c1',
    title: 'Эссе о лидерстве',
    short_description: 'Текстовое задание объёмом до 5000 знаков.',
    status: 'PUBLISHED',
    deadline_at: inDays(6),
    allow_edit_after_submission: false,
    allow_late_submission: false,
    schema_version: 1,
    fields: [],
  },
  {
    id: 'ch3',
    contestId: 'c1',
    title: 'Командный проект',
    short_description: 'Загрузка материалов командного проекта.',
    status: 'PUBLISHED',
    deadline_at: inDays(-1),
    allow_edit_after_submission: false,
    allow_late_submission: false,
    schema_version: 2,
    fields: [],
  },
]
