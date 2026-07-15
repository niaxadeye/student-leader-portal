import { apiRequest } from '@/shared/api/client'

/** Сброс пароля пользователя — возвращает временный пароль (показать один раз). */
export function resetPassword(userId: string): Promise<{ temp_password: string }> {
  return apiRequest<{ temp_password: string }>(`/admin/users/${userId}/reset-password`, {
    method: 'POST',
  })
}

export function blockUser(userId: string): Promise<{ status: string }> {
  return apiRequest(`/admin/users/${userId}/block`, { method: 'POST' })
}

export function unblockUser(userId: string): Promise<{ status: string }> {
  return apiRequest(`/admin/users/${userId}/unblock`, { method: 'POST' })
}
