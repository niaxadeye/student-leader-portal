import type { FormField } from '@/entities/challenge/types'

// Секции «Свет» и «Звук» + загрузка материалов (SITE.md §11.2).
export const lightSoundFields: FormField[] = [
  { id: 'f7', key: 'sec_light', type: 'SECTION', label: 'Свет', sort_order: 7 },
  {
    id: 'f8',
    key: 'light_mood',
    type: 'RADIO',
    label: 'Общий характер света',
    help_text: 'Базовая световая атмосфера сцены на время выступления.',
    required: true,
    sort_order: 8,
    options: [
      { value: 'bright', label: 'Яркий равномерный' },
      { value: 'warm', label: 'Тёплый приглушённый' },
      { value: 'spot', label: 'Акцент на спикере' },
    ],
  },
  {
    id: 'f9',
    key: 'light_notes',
    type: 'LONG_TEXT',
    label: 'Комментарий по свету',
    placeholder: 'Опишите моменты включения, затемнения, переходы…',
    help_text: 'Укажите тайминги, если свет меняется по ходу выступления.',
    sort_order: 9,
  },
  { id: 'f10', key: 'sec_sound', type: 'SECTION', label: 'Звук', sort_order: 10 },
  {
    id: 'f11',
    key: 'mic_type',
    type: 'SELECT',
    label: 'Тип микрофона',
    required: true,
    help_text: 'Петличный удобен при движении по сцене.',
    sort_order: 11,
    options: [
      { value: 'lavalier', label: 'Петличный' },
      { value: 'handheld', label: 'Ручной' },
      { value: 'headset', label: 'Головная гарнитура' },
    ],
  },
  {
    id: 'f12',
    key: 'backup_link',
    type: 'URL',
    label: 'Резервная ссылка на материалы',
    placeholder: 'https://…',
    help_text: 'Облачная ссылка на случай сбоя. Доступ должен быть открыт.',
    sort_order: 12,
  },
  {
    id: 'f13',
    key: 'materials',
    type: 'FILE_GROUP',
    label: 'Презентация и видеоматериалы',
    description: 'Загрузите материалы для выступления.',
    help_text: 'Презентации, видео, изображения, PDF и архивы.',
    required: true,
    sort_order: 13,
    settings: {
      multiple: true,
      allowed_extensions: ['pdf', 'ppt', 'pptx', 'mp4', 'mov', 'png', 'jpg', 'zip'],
      max_file_size_mb: 2048,
    },
  },
]
