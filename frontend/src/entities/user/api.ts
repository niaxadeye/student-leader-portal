import { apiRequest, apiRequestFull } from '@/shared/api/client'
import type { RoleCode } from '@/entities/auth/types'
import type {
  AdminUser,
  CreateUserInput,
  CreateUserResult,
  UsersFilter,
  UsersPage,
} from './types'

/** Реестр пользователей с серверной пагинацией/поиском/фильтрами. */
export async function listUsers(f: UsersFilter): Promise<UsersPage> {
  const p = new URLSearchParams()
  if (f.search) p.set('search', f.search)
  if (f.role) p.set('role', f.role)
  if (f.status) p.set('status', f.status)
  if (f.limit != null) p.set('limit', String(f.limit))
  if (f.offset != null) p.set('offset', String(f.offset))
  const qs = p.toString()
  const { data, meta } = await apiRequestFull<AdminUser[]>(`/admin/users${qs ? `?${qs}` : ''}`)
  return {
    users: data,
    total: Number(meta?.total ?? data.length),
    limit: Number(meta?.limit ?? 20),
    offset: Number(meta?.offset ?? 0),
  }
}

export function getUser(id: string): Promise<AdminUser> {
  return apiRequest<AdminUser>(`/admin/users/${id}`)
}

export function createUser(input: CreateUserInput): Promise<CreateUserResult> {
  return apiRequest<CreateUserResult>('/admin/users', { method: 'POST', body: input })
}

export function updateUser(
  id: string,
  input: { full_name: string; email?: string; organization?: string },
): Promise<AdminUser> {
  return apiRequest<AdminUser>(`/admin/users/${id}`, { method: 'PATCH', body: input })
}

export function assignRole(
  userId: string,
  body: { role: RoleCode; scope_type: 'GLOBAL' | 'CONTEST'; scope_id?: string },
): Promise<void> {
  return apiRequest(`/admin/users/${userId}/roles`, { method: 'POST', body })
}

export function removeRole(
  userId: string,
  role: RoleCode,
  scopeType: 'GLOBAL' | 'CONTEST',
  scopeId: string,
): Promise<void> {
  const p = new URLSearchParams({ role, scope_type: scopeType, scope_id: scopeId })
  return apiRequest(`/admin/users/${userId}/roles?${p.toString()}`, { method: 'DELETE' })
}
