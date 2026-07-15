import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { assignRole, createUser, getUser, listUsers, removeRole, updateUser } from './api'
import { blockUser, resetPassword, unblockUser } from './admin-actions'
import type { CreateUserInput, RoleAssignment, UsersFilter } from './types'
import type { RoleCode } from '@/entities/auth/types'

/** Реестр пользователей — только SUPER_ADMIN (гард на роуте). */
export function useAdminUsers(filter: UsersFilter, enabled = true) {
  return useQuery({
    queryKey: ['admin', 'users', filter],
    queryFn: () => listUsers(filter),
    enabled,
  })
}

/** Один пользователь с ролями (для диалога управления ролями). */
export function useAdminUser(id: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ['admin', 'user', id],
    queryFn: () => getUser(id!),
    enabled: !!id && enabled,
  })
}

export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateUserInput) => createUser(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
}

export function useUpdateUser(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { full_name: string; email?: string; organization?: string }) =>
      updateUser(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
}

/** Блокировка/разблокировка пользователя из реестра. */
export function useUserStatusMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, block }: { userId: string; block: boolean }) =>
      block ? blockUser(userId) : unblockUser(userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
}

/** Сброс пароля (используется и в реестре, и в таблице конкурсантов). */
export function useResetPassword() {
  return useMutation({ mutationFn: (userId: string) => resetPassword(userId) })
}

/** Назначение роли со scope. Инвалидирует карточку пользователя и реестр. */
export function useAssignRole(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { role: RoleCode; scope_type: 'GLOBAL' | 'CONTEST'; scope_id?: string }) =>
      assignRole(userId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'user', userId] })
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}

/** Снятие роли со scope. */
export function useRemoveRole(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (a: RoleAssignment) => removeRole(userId, a.code, a.scope_type, a.scope_id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'user', userId] })
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}
