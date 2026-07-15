import { useState } from 'react'
import { KeyRound, Ban, RotateCcw, ShieldCheck } from 'lucide-react'
import { useResetPassword, useUserStatusMutation } from '@/entities/user/queries'
import { ManageRolesDialog } from './manage-roles-dialog'
import { Card } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { useToast } from '@/shared/ui/toast'
import { formatDate } from '@/shared/lib/format'
import type { RoleCode } from '@/entities/auth/types'
import type { AdminUser } from '@/entities/user/types'

const roleBadge: Record<RoleCode, { label: string; tone: 'brand' | 'neutral' | 'success' }> = {
  MEGA_ADMIN: { label: 'Мегаадмин', tone: 'brand' },
  SUPER_ADMIN: { label: 'Суперадмин', tone: 'brand' },
  ADMIN: { label: 'Админ', tone: 'success' },
  CONTESTANT: { label: 'Конкурсант', tone: 'neutral' },
}

/** Уникальные коды ролей пользователя (роли приходят со scope, схлопываем по коду). */
function roleCodes(u: AdminUser): RoleCode[] {
  return [...new Set(u.roles.map((r) => r.code))]
}

export function UsersTable({ users }: { users: AdminUser[] }) {
  return (
    <Card className="overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[560px] text-left text-[14px]">
          <thead className="text-[12px] uppercase tracking-wide text-muted-2">
            <tr className="border-b border-border">
              <th className="px-4 py-2 font-medium">Пользователь</th>
              <th className="px-4 py-2 font-medium">Роли</th>
              <th className="hidden px-4 py-2 font-medium md:table-cell">Вход</th>
              <th className="px-4 py-2 font-medium">Статус</th>
              <th className="px-4 py-2 text-right font-medium">Действия</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <UserRow key={u.id} u={u} />
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}

function UserRow({ u }: { u: AdminUser }) {
  const toast = useToast()
  const reset = useResetPassword()
  const status = useUserStatusMutation()
  const [rolesOpen, setRolesOpen] = useState(false)
  const blocked = u.status === 'BLOCKED'

  function onReset() {
    reset.mutate(u.id, {
      onSuccess: (r) =>
        toast({
          title: `Пароль сброшен: ${u.login}`,
          description: `Временный пароль: ${r.temp_password}`,
          tone: 'success',
        }),
      onError: () => toast({ title: 'Не удалось сбросить пароль', tone: 'error' }),
    })
  }

  function onToggleBlock() {
    status.mutate(
      { userId: u.id, block: !blocked },
      {
        onSuccess: () =>
          toast({ title: blocked ? `Разблокирован: ${u.login}` : `Заблокирован: ${u.login}`, tone: 'info' }),
        onError: () => toast({ title: 'Не удалось изменить статус', tone: 'error' }),
      },
    )
  }

  return (
    <tr className="border-b border-border last:border-0 hover:bg-muted/5">
      <td className="px-4 py-3">
        <p className="font-medium text-ink">{u.full_name}</p>
        <p className="text-[13px] text-muted-2">{u.login}</p>
      </td>
      <td className="px-4 py-3">
        <div className="flex flex-wrap gap-1">
          {roleCodes(u).map((r) => (
            <Badge key={r} tone={roleBadge[r].tone}>
              {roleBadge[r].label}
            </Badge>
          ))}
        </div>
      </td>
      <td className="hidden whitespace-nowrap px-4 py-3 text-muted md:table-cell">
        {u.last_login_at ? formatDate(u.last_login_at) : '—'}
      </td>
      <td className="px-4 py-3">
        {blocked ? <Badge tone="danger">Заблокирован</Badge> : <Badge tone="success">Активен</Badge>}
      </td>
      <td className="px-4 py-3">
        <div className="flex justify-end gap-1">
          <button
            title="Управление ролями"
            onClick={() => setRolesOpen(true)}
            className="rounded-btn p-2 text-muted-2 hover:bg-muted/10 hover:text-brand"
          >
            <ShieldCheck className="h-4 w-4" />
          </button>
          <button
            title="Сбросить пароль"
            onClick={onReset}
            disabled={reset.isPending}
            className="rounded-btn p-2 text-muted-2 hover:bg-muted/10 hover:text-brand disabled:opacity-40"
          >
            <KeyRound className="h-4 w-4" />
          </button>
          <button
            title={blocked ? 'Разблокировать' : 'Заблокировать'}
            onClick={onToggleBlock}
            disabled={status.isPending}
            className="rounded-btn p-2 text-muted-2 hover:bg-muted/10 hover:text-danger disabled:opacity-40"
          >
            {blocked ? <RotateCcw className="h-4 w-4" /> : <Ban className="h-4 w-4" />}
          </button>
          <ManageRolesDialog
            userId={u.id}
            login={u.login}
            open={rolesOpen}
            onOpenChange={setRolesOpen}
          />
        </div>
      </td>
    </tr>
  )
}
