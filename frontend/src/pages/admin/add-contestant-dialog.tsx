import { useState } from 'react'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { toast } from 'sonner'
import { useAddContestant } from '@/entities/contestant/queries'
import { TempPasswordNote } from './temp-password-note'

export function AddContestantDialog({
  contestId,
  open,
  onOpenChange,
}: {
  contestId: string
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const [login, setLogin] = useState('')
  const [fullName, setFullName] = useState('')
  const [organization, setOrganization] = useState('')
  const [error, setError] = useState<string>()
  const [temp, setTemp] = useState<{ login: string; password: string }>()
  const add = useAddContestant(contestId)

  function reset() {
    setLogin('')
    setFullName('')
    setOrganization('')
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
    add.mutate(
      { login: login.trim(), full_name: fullName.trim(), organization: organization.trim() || undefined },
      {
        onSuccess: (r) => {
          toast.success('Конкурсант добавлен')
          setTemp({ login: r.login, password: r.temp_password })
        },
        onError: () => setError('Не удалось добавить. Возможно, логин занят.'),
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
      <DialogContent title="Добавить конкурсанта" description="Создаётся учётная запись с временным паролем.">
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
              {(p) => <Input {...p} value={organization} onChange={(e) => setOrganization(e.target.value)} />}
            </Field>
            <div className="mt-1 flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
                Отмена
              </Button>
              <Button type="submit" loading={add.isPending}>
                Добавить
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
