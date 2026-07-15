import type { FormField } from '@/entities/challenge/types'

// Демо-схема формы из шаблона ТЗ «Экран / Свет / Звук» (SITE.md §11.2).
// Собрана из секций и полей, а не захардкожена — как требует ТЗ.
export const selfPresentationFields: FormField[] = [
  {
    id: 'f1',
    key: 'intro',
    type: 'INFO_BLOCK',
    label: 'О задании',
    description:
      'Заполните техническое задание для вашего выступления. Все поля со звёздочкой обязательны. Черновик сохраняется автоматически.',
    sort_order: 1,
  },
  {
    id: 'f2',
    key: 'project_name',
    type: 'SHORT_TEXT',
    label: 'Название выступления',
    placeholder: 'Например: «Лидерство через сообщество»',
    help_text: 'Короткое название, которое увидит жюри в программе.',
    required: true,
    sort_order: 2,
  },
  // --- Секция: Экран ---
  { id: 'f3', key: 'sec_screen', type: 'SECTION', label: 'Экран', sort_order: 3 },
  {
    id: 'f4',
    key: 'screen_required',
    type: 'CHECKBOX',
    label: 'Требуется экран',
    help_text: 'Отметьте, если во время выступления нужна проекция на экран.',
    sort_order: 4,
  },
  {
    id: 'f5',
    key: 'screen_resolution',
    type: 'SELECT',
    label: 'Разрешение',
    help_text: 'Соотношение сторон зала — 16:9.',
    sort_order: 5,
    options: [
      { value: '1920x1080', label: '1920×1080 (Full HD)' },
      { value: '1280x720', label: '1280×720 (HD)' },
      { value: '3840x2160', label: '3840×2160 (4K)' },
    ],
  },
  {
    id: 'f6',
    key: 'screen_duration',
    type: 'NUMBER',
    label: 'Длительность видеоряда, сек',
    placeholder: '0',
    help_text: 'Общая длительность всех видеофрагментов.',
    sort_order: 6,
  },
]
