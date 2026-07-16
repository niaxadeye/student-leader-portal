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
import { TempPasswordNote } from './temp-password-note'
import type { RoleCode } from '@/entities/auth/types'

const allRoleOptions: Array<{ value: RoleCode; label: string }> = [
  { value: 'SUPER_ADMIN', label: 'Суперадмин' },
  { value: 'ADMIN', label: 'Админ' },
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
  // SUPER_ADMIN не может создавать SUPER_ADMIN — только мега (§3.3).
  const roleOptions = isMega(user)
    ? allRoleOptions
    : allRoleOptions.filter((o) => o.value !== 'SUPER_ADMIN')
  const [login, setLogin] = useState('')
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<RoleCode>('ADMIN')
  const [error, setError] = useState<string>()
  const [temp, setTemp] = useState<{ login: string; password: string }>()
  const create = useCreateUser()

  function reset() {
    setLogin('')
    setFullName('')
    setEmail('')
    setRole('ADMIN')
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
    create.mutate(
      { login: login.trim(), full_name: fullName.trim(), email: email.trim() || undefined, role },
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
            <Field label="Роль" required>
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
