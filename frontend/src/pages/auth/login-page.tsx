import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Field } from '@/shared/ui/field'
import { ApiRequestError } from '@/shared/api/client'
import { login, fetchMe } from '@/entities/auth/api'
import { useAuth } from '@/entities/auth/auth-context'
import { landingPath } from '@/entities/auth/roles'
import { loginSchema, type LoginValues } from '@/features/auth/login-schema'

// Нейтральное сообщение об ошибке входа (SITE.md §49.1): не раскрываем деталей.
function authMessage(e: unknown): string {
  if (e instanceof ApiRequestError) {
    if (e.code === 'AUTH_ACCOUNT_BLOCKED') return 'Учётная запись заблокирована. Обратитесь к дирекции.'
    if (e.code === 'RATE_LIMIT_EXCEEDED') return 'Слишком много попыток. Попробуйте позже.'
    if (e.code === 'AUTH_INVALID_CREDENTIALS') return 'Неверный логин или пароль'
  }
  return 'Не удалось выполнить вход. Попробуйте ещё раз.'
}

export function LoginPage() {
  const navigate = useNavigate()
  const { setUser } = useAuth()
  const [authError, setAuthError] = useState('')
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({ resolver: zodResolver(loginSchema) })

  async function onSubmit(values: LoginValues) {
    setAuthError('')
    try {
      await login(values)
      const me = await fetchMe()
      setUser(me)
      if (me.must_change_password) {
        navigate('/change-password', { replace: true })
        return
      }
      navigate(landingPath(me.roles), { replace: true })
    } catch (e) {
      setAuthError(authMessage(e))
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-2 px-4">
      <div className="w-full max-w-[400px]">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-btn bg-brand text-[20px] font-bold text-white">
            SL
          </div>
          <h1 className="text-[28px] font-bold text-ink">Вход в кабинет</h1>
          <p className="mt-1 text-[15px] text-muted">Студенческий лидер 2026</p>
        </div>

        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex flex-col gap-4 rounded-card border border-border bg-surface p-6 shadow-subtle"
          noValidate
        >
          <Field label="Логин" required error={errors.login?.message}>
            {(p) => <Input {...p} autoComplete="username" autoFocus {...register('login')} />}
          </Field>
          <Field label="Пароль" required error={errors.password?.message}>
            {(p) => (
              <Input {...p} type="password" autoComplete="current-password" {...register('password')} />
            )}
          </Field>

          {authError && (
            <div
              role="alert"
              className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[14px] text-danger"
            >
              {authError}
            </div>
          )}

          <Button type="submit" loading={isSubmitting} className="mt-1 w-full">
            {isSubmitting ? 'Вход…' : 'Войти'}
          </Button>

          <button
            type="button"
            onClick={() => navigate('/forgot-password')}
            className="text-center text-[14px] text-brand hover:text-brand-dark"
          >
            Забыли пароль?
          </button>
        </form>
      </div>
    </div>
  )
}
