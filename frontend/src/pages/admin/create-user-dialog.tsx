import { useState } from 'react'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'
import { toast } from 'sonner'
import { useAuth } from '@/entities/auth/auth-context'
import { isMega } from '@/entities/auth/roles'
import { useCreateUser } from '@/entities/user/queries'
import { useAdminContests } from '@/entities/contest/queries'
import { useAppConfig } from '@/shared/config/use-app-config'
import { TempPasswordNote } from './temp-password-note'
import type { RoleCode } from '@/entities/auth/types'
import type { AccessLevel } from '@/entities/user/types'

const allRoleOptions: Array<{ value: RoleCode; label: string }> = [
  { value: 'SUPER_ADMIN', label: 'Суперадмин' },
  { value: 'ADMIN', label: 'Админ' },
  { value: 'STAFF', label: 'Сотрудник' },
  { value: 'JURY', label: 'Жюри (весь конкурс)' },
  { value: 'REMOTE_JURY', label: 'Заочное жюри' },
  { value: 'CONTESTANT', label: 'Конкурсант' },
]

export function CreateUserDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { user } = useAuth()
  const contests = useAdminContests()
  const { data: appConfig } = useAppConfig()
  const visibleRoles = appConfig?.features.jury
    ? allRoleOptions
    : allRoleOptions.filter((o) => o.value !== 'JURY' && o.value !== 'REMOTE_JURY')
  const roleOptions = isMega(user)
    ? visibleRoles
    : visibleRoles.filter((o) => o.value !== 'SUPER_ADMIN')
  const [login, setLogin] = useState('')
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<RoleCode>('ADMIN')
  const [scopeId, setScopeId] = useState('')
  const [accessLevel, setAccessLevel] = useState<AccessLevel>('EDIT')
  const [error, setError] = useState<string>()
  const [temp, setTemp] = useState<{ login: string; password: string }>()
  const create = useCreateUser()
  const needsContest = role === 'ADMIN' || role === 'JURY' || role === 'REMOTE_JURY'
  const needsAccessLevel = role === 'ADMIN'

  function reset() {
    setLogin('')
    setFullName('')
    setEmail('')
    setRole('ADMIN')
    setScopeId('')
    setAccessLevel('EDIT')
    setError(undefined)
    setTemp(undefined)
  }

  function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(undefined)
    if (!login.trim() || !fullName.trim()) {
      setError('Логин и ФИО обязательны.')
      return
    }
    if (needsContest && !scopeId) {
      setError(
        role === 'ADMIN'
          ? 'Админу нужен конкурс и уровень EDIT или VIEW.'
          : 'Жюри нужен конкурс.',
      )
      return
    }
    create.mutate(
      {
        login: login.trim(),
        full_name: fullName.trim(),
        email: email.trim() || undefined,
        role,
        scope_type: needsContest ? 'CONTEST' : 'GLOBAL',
        scope_id: needsContest ? scopeId : undefined,
        access_level: needsAccessLevel ? accessLevel : undefined,
      },
      {
        onSuccess: (r) => {
          toast.success('Пользователь создан')
          setTemp({ login: r.login, password: r.temp_password })
        },
        onError: () => setError('Не удалось создать. Возможно, логин занят.'),
      },
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) reset()
        onOpenChange(v)
      }}
    >
      <DialogContent title="Новый пользователь" description="Создаётся с временным паролем и ролью.">
        {temp ? (
          <div className="flex flex-col gap-4">
            <TempPasswordNote login={temp.login} password={temp.password} />
            <div className="flex justify-end">
              <Button onClick={() => { reset(); onOpenChange(false) }}>Готово</Button>
            </div>
          </div>
        ) : (
          <form onSubmit={submit} className="flex flex-col gap-4">
            <Field label="Логин" required error={error}>
              {(p) => <Input {...p} value={login} onChange={(e) => setLogin(e.target.value)} autoFocus />}
            </Field>
            <Field label="ФИО" required>
              {(p) => <Input {...p} value={fullName} onChange={(e) => setFullName(e.target.value)} />}
            </Field>
            <Field label="Email">
              {(p) => <Input {...p} type="email" value={email} onChange={(e) => setEmail(e.target.value)} />}
            </Field>
            <Field
              label="Роль"
              required
              helpText={
                role === 'REMOTE_JURY'
                  ? 'Не входит в жюри всего конкурса. После создания отметьте человека в схеме заочного испытания.'
                  : undefined
              }
            >
              {(p) => (
                <Select value={role} onValueChange={(v) => setRole(v as RoleCode)}>
                  <SelectTrigger id={p.id} aria-invalid={p['aria-invalid']}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {roleOptions.map((o) => (
                      <SelectItem key={o.value} value={o.value}>
                        {o.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </Field>
            {needsContest && (
              <>
                <Field
                  label="Конкурс"
                  required
                  helpText={
                    role === 'JURY'
                      ? 'Live-жюри всего конкурса. Заочных назначайте ролью «Заочное жюри».'
                      : role === 'REMOTE_JURY'
                        ? 'Пул заочного жюри этого конкурса. Они не видят live-испытания.'
                        : 'Глобальный ADMIN запрещён — доступ только к выбранному конкурсу.'
                  }
                >
                  {(p) => (
                    <Select value={scopeId || ''} onValueChange={setScopeId}>
                      <SelectTrigger id={p.id}>
                        <SelectValue placeholder="Выберите конкурс" />
                      </SelectTrigger>
                      <SelectContent>
                        {(contests.data ?? []).map((c) => (
                          <SelectItem key={c.id} value={c.id}>
                            {c.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </Field>
                {needsAccessLevel && (
                <Field label="Уровень доступа">
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
              </>
            )}
            <div className="mt-1 flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
                Отмена
              </Button>
              <Button type="submit" loading={create.isPending}>
                Создать
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
