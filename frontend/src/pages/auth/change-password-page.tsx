import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Field } from '@/shared/ui/field'
import { useToast } from '@/shared/ui/toast'
import { ApiRequestError } from '@/shared/api/client'
import { changePassword } from '@/entities/auth/api'
import { useAuth } from '@/entities/auth/auth-context'
import {
  changePasswordSchema,
  type ChangePasswordValues,
} from '@/features/auth/change-password-schema'

export function ChangePasswordPage() {
  const navigate = useNavigate()
  const toast = useToast()
  const { user, setUser } = useAuth()
  const [formError, setFormError] = useState('')
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ChangePasswordValues>({ resolver: zodResolver(changePasswordSchema) })

  async function onSubmit(values: ChangePasswordValues) {
    setFormError('')
    try {
      await changePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      })
      // Бэкенд отозвал сессии → нужен повторный вход.
      setUser(null)
      toast({ tone: 'success', title: 'Пароль изменён', description: 'Войдите заново.' })
      navigate('/login', { replace: true })
    } catch (e) {
      const msg =
        e instanceof ApiRequestError ? e.message : 'Не удалось изменить пароль'
      setFormError(msg)
    }
  }

  const forced = user?.must_change_password

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-2 px-4">
      <div className="w-full max-w-[400px]">
        <div className="mb-8 text-center">
          <h1 className="text-[28px] font-bold text-ink">Смена пароля</h1>
          <p className="mt-1 text-[15px] text-muted">
            {forced ? 'Задайте постоянный пароль, чтобы продолжить.' : 'Обновите пароль от кабинета.'}
          </p>
        </div>

        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex flex-col gap-4 rounded-card border border-border bg-surface p-6 shadow-subtle"
          noValidate
        >
          <Field label="Текущий пароль" required error={errors.old_password?.message}>
            {(p) => (
              <Input {...p} type="password" autoComplete="current-password" autoFocus {...register('old_password')} />
            )}
          </Field>
          <Field label="Новый пароль" required error={errors.new_password?.message}>
            {(p) => (
              <Input {...p} type="password" autoComplete="new-password" {...register('new_password')} />
            )}
          </Field>
          <Field label="Повторите новый пароль" required error={errors.confirm?.message}>
            {(p) => (
              <Input {...p} type="password" autoComplete="new-password" {...register('confirm')} />
            )}
          </Field>

          {formError && (
            <div role="alert" className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[14px] text-danger">
              {formError}
            </div>
          )}

          <Button type="submit" loading={isSubmitting} className="mt-1 w-full">
            {isSubmitting ? 'Сохранение…' : 'Сменить пароль'}
          </Button>
        </form>
      </div>
    </div>
  )
}
