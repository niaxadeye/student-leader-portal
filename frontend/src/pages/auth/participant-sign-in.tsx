import { useEffect, useMemo, useRef, useState, type ComponentType } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { ChevronDown, IdCard, ScanLine, Send, UserRound } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  fetchLoginOptions,
  loginParticipantByName,
  loginParticipantBySKS,
  loginParticipantByTelegramWebApp,
  loginParticipantByUnionCard,
  loginParticipantByVKToken,
  socialAuthStartURL,
  type LoginOptions,
} from '@/entities/event-participant/api'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import type { ParticipantSession } from '@/entities/event-participant/types'
import {
  participantIdentifierLoginSchema,
  participantNameLoginSchema,
  type ParticipantIdentifierLoginValues,
  type ParticipantNameLoginValues,
} from '@/features/participant-auth/login-schema'
import { BackButton } from '@/pages/auth/login-back-button'
import { ApiRequestError } from '@/shared/api/client'
import { telegramWebApp } from '@/shared/lib/telegram-webapp'
import { exchangeVkOneTapCode, renderVkOneTap } from '@/shared/lib/vkid'
import { Button } from '@/shared/ui/button'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'

type BackupMethod = 'name' | 'union' | 'sks'

const backupMethods: Array<{
  id: BackupMethod
  label: string
  icon: ComponentType<{ className?: string }>
}> = [
  { id: 'name', label: 'ФИО и дата рождения', icon: UserRound },
  { id: 'union', label: 'Профбилет', icon: IdCard },
  { id: 'sks', label: 'Barcode СКС', icon: ScanLine },
]

function loginErrorMessage(error: unknown): string {
  if (error instanceof ApiRequestError) {
    switch (error.code) {
      case 'PARTICIPANT_IDENTITY_AMBIGUOUS':
        return 'Найдено несколько совпадений. Войдите резервным способом.'
      case 'RATE_LIMIT_EXCEEDED':
        return 'Слишком много попыток. Подождите несколько минут и попробуйте снова.'
      case 'PARTICIPANT_AUTH_FAILED':
        return 'Участник с такими данными не найден.'
      case 'SOCIAL_AUTH_UNAVAILABLE':
        return 'Вход через соцсеть ещё не настроен. Используйте резервный способ.'
      case 'NETWORK_ERROR':
        return 'Нет связи с сервером. Проверьте подключение и попробуйте снова.'
    }
  }
  return 'Не удалось выполнить вход. Попробуйте ещё раз.'
}

function socialErrorFromQuery(code: string | null): string {
  if (code === 'rate') return 'Слишком много попыток. Подождите несколько минут и попробуйте снова.'
  if (code === 'unavailable') return 'Вход через соцсеть ещё не настроен.'
  if (code === 'social') return 'Не удалось войти через соцсеть. Проверьте, что вы есть в списке участников.'
  return ''
}

export function ParticipantSignIn({ onBack }: { onBack: () => void }) {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const { acceptSession, session, status } = useParticipantAuth()
  const [options, setOptions] = useState<LoginOptions | null>(null)
  const [authError, setAuthError] = useState(socialErrorFromQuery(params.get('error')))
  const [backupOpen, setBackupOpen] = useState(false)
  const [backupMethod, setBackupMethod] = useState<BackupMethod>('name')
  const [webAppBusy, setWebAppBusy] = useState(false)
  const attempted = useRef('')
  const eventSlug = params.get('event')?.trim() ?? ''

  const eventName = useMemo(
    () => options?.events.find((event) => event.slug === eventSlug)?.name ?? eventSlug,
    [eventSlug, options],
  )

  useEffect(() => {
    void fetchLoginOptions()
      .then(setOptions)
      .catch(() => setOptions({ telegram: { enabled: false }, vk: { enabled: false }, events: [] }))
  }, [])

  useEffect(() => {
    const app = telegramWebApp()
    if (!app) return
    app.ready()
    app.expand()
    const startEvent = app.initDataUnsafe?.start_param?.trim()
    if (startEvent && startEvent !== eventSlug) {
      const next = new URLSearchParams(params)
      next.set('event', startEvent)
      setParams(next, { replace: true })
    }
  }, [eventSlug, params, setParams])

  useEffect(() => {
    if (session?.event.slug && session.event.slug === eventSlug) {
      navigate(`/event/${encodeURIComponent(eventSlug)}/me`, { replace: true })
    }
  }, [eventSlug, navigate, session])

  useEffect(() => {
    const app = telegramWebApp()
    if (!app || !eventSlug || status === 'loading') return
    if (attempted.current === eventSlug) return
    attempted.current = eventSlug
    setWebAppBusy(true)
    void completeLogin(() => loginParticipantByTelegramWebApp(eventSlug, app.initData)).finally(() => {
      setWebAppBusy(false)
    })
    // Mini App: один автологин на выбранное мероприятие.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [eventSlug, status])

  async function completeLogin(request: () => Promise<ParticipantSession>) {
    setAuthError('')
    try {
      const next = await request()
      acceptSession(next)
      navigate(`/event/${encodeURIComponent(next.event.slug)}/me`, { replace: true })
    } catch (error) {
      setAuthError(loginErrorMessage(error))
    }
  }

  function chooseEvent(slug: string) {
    const next = new URLSearchParams(params)
    next.set('event', slug)
    next.delete('error')
    setParams(next, { replace: true })
    setAuthError('')
  }

  const needEvent = !eventSlug
  const telegramOn = options?.telegram.enabled ?? false
  const vkOn = options?.vk.enabled ?? false

  return (
    <div className="flex flex-col gap-4 rounded-card border border-border bg-surface p-6 shadow-subtle">
      {needEvent ? (
        <EventPicker events={options?.events ?? []} loading={!options} onPick={chooseEvent} />
      ) : (
        <>
          <p className="text-[13px] font-medium text-brand">Мероприятие · {eventName || eventSlug}</p>
          {webAppBusy ? (
            <p className="text-[14px] text-muted">Входим через Telegram Mini App…</p>
          ) : (
            <div className="flex flex-col gap-3">
              <VkOneTapButton
                enabled={vkOn}
                appId={options?.vk.app_id}
                redirectUrl={options?.vk.redirect_url}
                onToken={(token) => completeLogin(() => loginParticipantByVKToken(eventSlug, token))}
                onError={() => setAuthError('Не удалось войти через VK. Попробуйте ещё раз.')}
              />
              <Button
                type="button"
                className="w-full bg-[#229ED9] hover:bg-[#1b8dc3]"
                disabled={!telegramOn}
                onClick={() => {
                  window.location.href = socialAuthStartURL(eventSlug, 'telegram')
                }}
              >
                <Send className="h-4 w-4" aria-hidden />
                Войти через Telegram
              </Button>
              {!vkOn && !telegramOn && (
                <p className="text-[13px] text-muted">
                  Социальный вход ещё не подключён. Используйте резервный способ ниже.
                </p>
              )}
            </div>
          )}
        </>
      )}

      {authError && (
        <div role="alert" className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[14px] text-danger">
          {authError}
        </div>
      )}

      <div className="border-t border-border pt-3">
        <button
          type="button"
          onClick={() => setBackupOpen((open) => !open)}
          className="flex w-full items-center justify-between text-[13px] font-medium text-muted hover:text-ink"
        >
          Резервный вход
          <ChevronDown className={`h-4 w-4 transition-transform ${backupOpen ? 'rotate-180' : ''}`} />
        </button>
        {backupOpen && (
          <div className="mt-4 flex flex-col gap-4">
            {!eventSlug && (
              <p className="text-[13px] text-muted">Сначала выберите мероприятие.</p>
            )}
            {eventSlug && (
              <>
                <div className="grid grid-cols-3 gap-1 rounded-[12px] bg-surface-2 p-1">
                  {backupMethods.map(({ id, label, icon: Icon }) => (
                    <button
                      key={id}
                      type="button"
                      onClick={() => {
                        setBackupMethod(id)
                        setAuthError('')
                      }}
                      className={`flex min-h-11 flex-col items-center justify-center gap-1 rounded-[9px] px-1 text-[11px] font-medium sm:text-[12px] ${
                        backupMethod === id ? 'bg-surface text-brand shadow-micro' : 'text-muted'
                      }`}
                    >
                      <Icon className="h-4 w-4" aria-hidden />
                      {label}
                    </button>
                  ))}
                </div>
                {backupMethod === 'name' && (
                  <NameLoginForm
                    error=""
                    onSubmit={(values) => completeLogin(() => loginParticipantByName(eventSlug, values))}
                  />
                )}
                {backupMethod === 'union' && (
                  <IdentifierLoginForm
                    method="union"
                    error=""
                    onSubmit={(values) =>
                      completeLogin(() => loginParticipantByUnionCard(eventSlug, values))
                    }
                  />
                )}
                {backupMethod === 'sks' && (
                  <IdentifierLoginForm
                    method="sks"
                    error=""
                    onSubmit={(values) => completeLogin(() => loginParticipantBySKS(eventSlug, values))}
                  />
                )}
              </>
            )}
          </div>
        )}
      </div>

      <BackButton onClick={onBack} />
    </div>
  )
}

function VkOneTapButton({
  enabled,
  appId,
  redirectUrl,
  onToken,
  onError,
}: {
  enabled: boolean
  appId?: string
  redirectUrl?: string
  onToken: (accessToken: string) => void
  onError: () => void
}) {
  const container = useRef<HTMLDivElement>(null)
  const onTokenRef = useRef(onToken)
  const onErrorRef = useRef(onError)
  const [failed, setFailed] = useState(false)
  onTokenRef.current = onToken
  onErrorRef.current = onError

  useEffect(() => {
    const node = container.current
    if (!enabled || !appId || !redirectUrl || !node) return
    let disposed = false
    let cleanup: (() => void) | undefined
    void renderVkOneTap({
      container: node,
      appId,
      redirectUrl,
      onCode: (code, deviceId) => {
        void exchangeVkOneTapCode(code, deviceId)
          .then((token) => {
            if (!disposed) onTokenRef.current(token)
          })
          .catch(() => {
            if (!disposed) {
              setFailed(true)
              onErrorRef.current()
            }
          })
      },
      onError: () => {
        if (!disposed) {
          setFailed(true)
          onErrorRef.current()
        }
      },
    })
      .then((destroy) => {
        cleanup = destroy
      })
      .catch(() => {
        if (!disposed) setFailed(true)
      })
    return () => {
      disposed = true
      cleanup?.()
    }
  }, [appId, enabled, redirectUrl])

  if (!enabled || !appId || !redirectUrl) return null
  return (
    <div className="w-full">
      <div ref={container} className="min-h-[44px] w-full overflow-hidden" />
      {failed && (
        <p className="mt-2 text-[13px] text-muted">Не удалось показать кнопку VK. Обновите страницу.</p>
      )}
    </div>
  )
}

function EventPicker({
  events,
  loading,
  onPick,
}: {
  events: Array<{ slug: string; name: string }>
  loading: boolean
  onPick: (slug: string) => void
}) {
  if (loading) {
    return <p className="text-[14px] text-muted">Загружаем список мероприятий…</p>
  }
  if (!events.length) {
    return <p className="text-[14px] text-muted">Сейчас нет активных мероприятий для входа.</p>
  }
  return (
    <div className="flex flex-col gap-2">
      <p className="text-[14px] font-medium text-ink">Выберите мероприятие</p>
      {events.map((event) => (
        <button
          key={event.slug}
          type="button"
          onClick={() => onPick(event.slug)}
          className="rounded-[12px] border border-border px-3.5 py-3 text-left hover:border-brand/40 hover:bg-brand-subtle/40"
        >
          <span className="block font-medium text-ink">{event.name}</span>
          <span className="mt-0.5 block text-[12px] text-muted">{event.slug}</span>
        </button>
      ))}
    </div>
  )
}

function NameLoginForm({
  error,
  onSubmit,
}: {
  error: string
  onSubmit: (values: ParticipantNameLoginValues) => Promise<void>
}) {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ParticipantNameLoginValues>({ resolver: zodResolver(participantNameLoginSchema) })

  return (
    <form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)} noValidate>
      <Field label="ФИО" required error={errors.full_name?.message}>
        {(props) => (
          <Input {...props} autoComplete="name" placeholder="Иванов Иван Иванович" {...register('full_name')} />
        )}
      </Field>
      <Field label="Дата рождения" required error={errors.birth_date?.message}>
        {(props) => (
          <Input
            {...props}
            type="date"
            autoComplete="bday"
            max={new Date().toISOString().slice(0, 10)}
            {...register('birth_date')}
          />
        )}
      </Field>
      {error ? (
        <div role="alert" className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[14px] text-danger">
          {error}
        </div>
      ) : null}
      <Button type="submit" loading={isSubmitting} className="w-full">
        Войти резервным способом
      </Button>
    </form>
  )
}

function IdentifierLoginForm({
  method,
  error,
  onSubmit,
}: {
  method: 'union' | 'sks'
  error: string
  onSubmit: (values: ParticipantIdentifierLoginValues) => Promise<void>
}) {
  const isScanner = method === 'sks'
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ParticipantIdentifierLoginValues>({
    resolver: zodResolver(participantIdentifierLoginSchema),
  })

  return (
    <form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)} noValidate>
      <Field
        label={isScanner ? 'Barcode СКС РФ' : 'Номер профсоюзного билета'}
        required
        error={errors.value?.message}
      >
        {(props) => (
          <Input
            {...props}
            autoComplete="off"
            placeholder={isScanner ? 'Отсканируйте или введите barcode' : 'Номер профбилета'}
            {...register('value')}
          />
        )}
      </Field>
      {error ? (
        <div role="alert" className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[14px] text-danger">
          {error}
        </div>
      ) : null}
      <Button type="submit" loading={isSubmitting} className="w-full">
        Войти резервным способом
      </Button>
    </form>
  )
}
