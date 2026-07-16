import { useState } from 'react'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { toast } from 'sonner'
import { useCreateUser } from '@/entities/user/queries'
import { TempPasswordNote } from './temp-password-note'

/** Диалог создания организатора (SUPER_ADMIN) мегаадмином. Роль фиксирована,
 *  задаётся организация-арендатор (org_name), наследуемая его админами (§2.3, §4). */
export function CreateOrganizerDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const [login, setLogin] = useState('')
  const [fullName, setFullName] = useState('')
  const [orgName, setOrgName] = useState('')
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string>()
  const [temp, setTemp] = useState<{ login: string; password: string }>()
  const create = useCreateUser()

  function reset() {
    setLogin('')
    setFullName('')
    setOrgName('')
    setEmail('')
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
      {
        login: login.trim(),
        full_name: fullName.trim(),
        email: email.trim() || undefined,
        org_name: orgName.trim() || undefined,
        role: 'SUPER_ADMIN',
      },
      {
        onSuccess: (r) => {
          toast.success('Организатор создан')
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
      <DialogContent
        title="Новый организатор"
        description="Создаётся суперадмин с временным паролем. Организация наследуется его админами."
      >
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
            <Field label="Организация">
              {(p) => <Input {...p} value={orgName} onChange={(e) => setOrgName(e.target.value)} placeholder="Напр. МГУ" />}
            </Field>
            <Field label="Email">
              {(p) => <Input {...p} type="email" value={email} onChange={(e) => setEmail(e.target.value)} />}
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
