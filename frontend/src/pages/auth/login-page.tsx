import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { GraduationCap, Shield, Users } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Field } from '@/shared/ui/field'
import { ApiRequestError } from '@/shared/api/client'
import { login, fetchMe } from '@/entities/auth/api'
import { useAuth } from '@/entities/auth/auth-context'
import { landingPath } from '@/entities/auth/roles'
import { loginSchema, type LoginValues } from '@/features/auth/login-schema'
import { FullscreenLoader } from '@/app/guards'
import { BackButton } from '@/pages/auth/login-back-button'
import { ParticipantSignIn } from '@/pages/auth/participant-sign-in'
import { telegramWebApp, maybeTelegramMiniApp, waitForTelegramWebApp, ensureTelegramWebAppScript } from '@/shared/lib/telegram-webapp'

export type LoginAudience = 'admin' | 'contestant' | 'participant'

function authMessage(e: unknown): string {
  if (e instanceof ApiRequestError) {
    if (e.code === 'AUTH_ACCOUNT_BLOCKED') return 'Учётная запись заблокирована. Обратитесь к дирекции.'
    if (e.code === 'RATE_LIMIT_EXCEEDED') return 'Слишком много попыток. Попробуйте позже.'
    if (e.code === 'AUTH_INVALID_CREDENTIALS') return 'Неверный логин или пароль'
  }
  return 'Не удалось выполнить вход. Попробуйте ещё раз.'
}

const roles: Array<{
  id: LoginAudience
  title: string
  description: string
  icon: typeof Shield
}> = [
  {
    id: 'admin',
    title: 'Администратор',
    description: 'Кабинет организаторов и сотрудников мероприятия',
    icon: Shield,
  },
  {
    id: 'contestant',
    title: 'Конкурсант',
    description: 'Испытания, черновики и отправка работ',
    icon: GraduationCap,
  },
  {
    id: 'participant',
    title: 'Участник',
    description: 'Вход в платформу мероприятия через VK или Telegram',
    icon: Users,
  },
]

export function LoginPage() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const { status, user, setUser } = useAuth()
  const [audience, setAudienceState] = useState<LoginAudience | null>(
    parseAudience(params.get('as')) ?? (telegramWebApp() || maybeTelegramMiniApp() ? 'participant' : null),
  )
  const [miniAppReady, setMiniAppReady] = useState(() => Boolean(telegramWebApp()))

  function setAudience(next: LoginAudience | null) {
    setAudienceState(next)
    const copy = new URLSearchParams(params)
    if (next) copy.set('as', next)
    else copy.delete('as')
    setParams(copy, { replace: true })
  }

  useEffect(() => {
    if (maybeTelegramMiniApp()) ensureTelegramWebAppScript()
    if (telegramWebApp()) {
      setMiniAppReady(true)
      if (audience !== 'participant') setAudience('participant')
      return
    }
    if (!maybeTelegramMiniApp()) return
    void waitForTelegramWebApp().then((app) => {
      if (!app) return
      setMiniAppReady(true)
      setAudience('participant')
    })
    // Mini App always enters as a participant.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (status === 'authenticated' && user && audience !== 'participant') {
      navigate(user.must_change_password ? '/change-password' : landingPath(user.roles), {
        replace: true,
      })
    }
  }, [audience, navigate, status, user])

  if (status === 'loading' && !miniAppReady && !maybeTelegramMiniApp()) return <FullscreenLoader />

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-2 px-4 py-8">
      <div className="w-full max-w-[440px]">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-btn bg-brand text-[20px] font-bold text-white">
            SL
          </div>
          <h1 className="text-[28px] font-bold text-ink">Вход в кабинет</h1>
          <p className="mt-1 text-[15px] text-muted">
            {audience ? subtitleFor(audience) : 'Сначала выберите, как вы входите'}
          </p>
        </div>

        {!audience ? (
          <RolePicker onPick={setAudience} />
        ) : audience === 'participant' ? (
          <ParticipantSignIn onBack={() => setAudience(null)} />
        ) : (
          <PasswordLogin
            audience={audience}
            onBack={() => setAudience(null)}
            onLoggedIn={async () => {
              const me = await fetchMe()
              setUser(me)
              if (me.must_change_password) {
                navigate('/change-password', { replace: true })
                return
              }
              navigate(landingPath(me.roles), { replace: true })
            }}
          />
        )}
      </div>
    </div>
  )
}

function RolePicker({ onPick }: { onPick: (role: LoginAudience) => void }) {
  return (
    <div className="flex flex-col gap-3">
      {roles.map(({ id, title, description, icon: Icon }) => (
        <button
          key={id}
          type="button"
          onClick={() => onPick(id)}
          className="flex items-start gap-4 rounded-card border border-border bg-surface p-4 text-left shadow-subtle transition-colors hover:border-brand/40 hover:bg-brand-subtle/40"
        >
          <span className="mt-0.5 flex h-11 w-11 shrink-0 items-center justify-center rounded-[12px] bg-brand-subtle text-brand">
            <Icon className="h-5 w-5" aria-hidden />
          </span>
          <span>
            <span className="block text-[16px] font-semibold text-ink">{title}</span>
            <span className="mt-1 block text-[13px] text-muted">{description}</span>
          </span>
        </button>
      ))}
    </div>
  )
}

function PasswordLogin({
  audience,
  onBack,
  onLoggedIn,
}: {
  audience: 'admin' | 'contestant'
  onBack: () => void
  onLoggedIn: () => Promise<void>
}) {
  const navigate = useNavigate()
  const [authError, setAuthError] = useState('')
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({ resolver: zodResolver(loginSchema) })

  async function onSubmit(values: LoginValues) {
    setAuthError('')
    try {
      await login({ ...values, audience })
      await onLoggedIn()
    } catch (e) {
      setAuthError(authMessage(e))
    }
  }

  return (
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
        <div role="alert" className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[14px] text-danger">
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
      <BackButton onClick={onBack} />
    </form>
  )
}

function parseAudience(value: string | null): LoginAudience | null {
  if (value === 'admin' || value === 'staff') return 'admin'
  if (value === 'contestant' || value === 'participant') return value
  return null
}

function subtitleFor(audience: LoginAudience): string {
  if (audience === 'admin') return 'Вход администратора и сотрудников'
  if (audience === 'contestant') return 'Вход конкурсанта'
  return 'Вход участника мероприятия'
}
