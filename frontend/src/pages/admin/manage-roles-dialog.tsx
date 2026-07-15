import { useState } from 'react'
import { Trash2, Plus } from 'lucide-react'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Button } from '@/shared/ui/button'
import { Badge } from '@/shared/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'
import { Skeleton, ErrorState } from '@/shared/ui/states'
import { useToast } from '@/shared/ui/toast'
import { useAdminUser, useAssignRole, useRemoveRole } from '@/entities/user/queries'
import { useAdminContests } from '@/entities/contest/queries'
import { ApiRequestError } from '@/shared/api/client'
import type { RoleCode } from '@/entities/auth/types'
import type { AccessLevel, RoleAssignment } from '@/entities/user/types'

const roleLabels: Record<RoleCode, string> = {
  MEGA_ADMIN: 'Мегаадмин',
  SUPER_ADMIN: 'Суперадмин',
  ADMIN: 'Админ',
  CONTESTANT: 'Конкурсант',
}

// Роли, назначаемые через этот диалог (мега/супер создаются иначе).
const assignableRoles: RoleCode[] = ['ADMIN', 'CONTESTANT']

export function ManageRolesDialog({
  userId,
  login,
  open,
  onOpenChange,
}: {
  userId: string
  login: string
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title="Роли пользователя" description={login}>
        {open && <RolesBody userId={userId} />}
      </DialogContent>
    </Dialog>
  )
}

function scopeLabel(a: RoleAssignment, contestName?: string): string {
  if (a.scope_type === 'GLOBAL') return 'Глобально'
  const base = contestName ? `Конкурс: ${contestName}` : `Конкурс ${a.scope_id.slice(0, 8)}`
  const lvl = a.access_level === 'EDIT' ? 'EDIT' : a.access_level === 'VIEW' ? 'VIEW' : ''
  return lvl ? `${base} · ${lvl}` : base
}

function RolesBody({ userId }: { userId: string }) {
  const { data: user, isLoading, isError, refetch } = useAdminUser(userId)
  const contests = useAdminContests()
  const assign = useAssignRole(userId)
  const remove = useRemoveRole(userId)
  const toast = useToast()

  const [role, setRole] = useState<RoleCode>('ADMIN')
  const [scopeId, setScopeId] = useState<string>('')
  const [accessLevel, setAccessLevel] = useState<AccessLevel>('EDIT')
  const [error, setError] = useState<string>()

  // Уровень доступа задаётся только для ADMIN на конкретный конкурс (§3.4).
  const needsAccessLevel = role === 'ADMIN' && scopeId !== ''

  const contestName = (id: string) => contests.data?.find((c) => c.id === id)?.name

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError || !user) return <ErrorState onRetry={() => refetch()} />

  function onRemove(a: RoleAssignment) {
    remove.mutate(a, {
      onSuccess: () => toast({ title: 'Роль снята', tone: 'info' }),
      onError: () => toast({ title: 'Не удалось снять роль', tone: 'error' }),
    })
  }

  function onAssign() {
    setError(undefined)
    const scopeType = scopeId ? 'CONTEST' : 'GLOBAL'
    assign.mutate(
      {
        role,
        scope_type: scopeType,
        scope_id: scopeId || undefined,
        access_level: needsAccessLevel ? accessLevel : undefined,
      },
      {
        onSuccess: () => {
          toast({ title: 'Роль назначена', tone: 'success' })
          setScopeId('')
        },
        onError: (err) =>
          setError(
            err instanceof ApiRequestError && err.code === 'VALIDATION_ERROR'
              ? 'Проверьте роль и область.'
              : 'Не удалось назначить роль.',
          ),
      },
    )
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-2">
        {user.roles.length === 0 ? (
          <p className="text-[14px] text-muted">Ролей пока нет.</p>
        ) : (
          user.roles.map((a) => (
            <div
              key={`${a.code}:${a.scope_type}:${a.scope_id}`}
              className="flex items-center justify-between rounded-[10px] border border-border px-3 py-2"
            >
              <div className="flex items-center gap-2">
                <Badge tone="brand">{roleLabels[a.code]}</Badge>
                <span className="text-[13px] text-muted">{scopeLabel(a, contestName(a.scope_id))}</span>
              </div>
              <button
                title="Снять роль"
                onClick={() => onRemove(a)}
                disabled={remove.isPending}
                className="rounded-btn p-1.5 text-muted-2 hover:bg-muted/10 hover:text-danger disabled:opacity-40"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))
        )}
      </div>

      <div className="border-t border-border pt-4">
        <p className="mb-3 text-[14px] font-medium text-ink">Назначить роль</p>
        <div className="flex flex-col gap-3">
          <Field label="Роль" error={error}>
            {(p) => (
              <Select value={role} onValueChange={(v) => setRole(v as RoleCode)}>
                <SelectTrigger id={p.id}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {assignableRoles.map((r) => (
                    <SelectItem key={r} value={r}>
                      {roleLabels[r]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </Field>
          <Field label="Область" helpText="Пусто — глобально. Иначе роль действует в выбранном конкурсе.">
            {(p) => (
              <Select value={scopeId || 'GLOBAL'} onValueChange={(v) => setScopeId(v === 'GLOBAL' ? '' : v)}>
                <SelectTrigger id={p.id}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="GLOBAL">Глобально</SelectItem>
                  {contests.data?.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </Field>
          {needsAccessLevel && (
            <Field label="Уровень доступа" helpText="EDIT — правка контента; VIEW — только просмотр. Участниками управляет только владелец.">
              {(p) => (
                <Select value={accessLevel} onValueChange={(v) => setAccessLevel(v as AccessLevel)}>
                  <SelectTrigger id={p.id}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="EDIT">Редактирование</SelectItem>
                    <SelectItem value="VIEW">Только просмотр</SelectItem>
                  </SelectContent>
                </Select>
              )}
            </Field>
          )}
          <div className="flex justify-end">
            <Button size="sm" onClick={onAssign} loading={assign.isPending}>
              <Plus className="h-4 w-4" /> Назначить
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}