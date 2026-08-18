import type { StaffGrant, StaffPermission } from '@/entities/auth/types'
import { apiRequest } from '@/shared/api/client'

export const STAFF_PERMISSION_OPTIONS: Array<{ value: StaffPermission; label: string; hint: string }> = [
  { value: 'event.attendance.scan', label: 'Сканер посещаемости', hint: 'Сканировать QR на лекции' },
  { value: 'event.attendance.manage', label: 'Лекции', hint: 'Создавать и вести расписание лекций' },
  { value: 'event.participants.manage', label: 'Участники мероприятия', hint: 'Список, импорт, блокировка' },
  { value: 'event.tasks.moderate', label: 'Проверка заданий', hint: 'Принимать и отклонять работы' },
  { value: 'event.tasks.manage', label: 'Задания', hint: 'Создавать и менять задания' },
  { value: 'event.merch.orders.manage', label: 'Выдача мерча', hint: 'Выдавать и отклонять заказы' },
  { value: 'event.merch.manage', label: 'Каталог мерча', hint: 'Товары и остатки' },
  { value: 'event.points.manage', label: 'Баллы', hint: 'Ручные корректировки баланса' },
]

export function listStaffPermissions(userId: string): Promise<StaffGrant[]> {
  return apiRequest<StaffGrant[]>(`/admin/users/${userId}/staff-permissions`)
}

export function replaceStaffPermissions(
  userId: string,
  contestId: string,
  permissions: StaffPermission[],
): Promise<void> {
  return apiRequest(`/admin/users/${userId}/staff-permissions`, {
    method: 'PUT',
    body: { contest_id: contestId, permissions },
  })
}

export function clearStaffPermissions(userId: string, contestId: string): Promise<void> {
  return apiRequest(
    `/admin/users/${userId}/staff-permissions?contest_id=${encodeURIComponent(contestId)}`,
    { method: 'DELETE' },
  )
}
