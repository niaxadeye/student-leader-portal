import { useEffect, useMemo, useRef, useState, type ComponentType } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { ChevronDown, IdCard, Loader2, ScanLine, UserRound } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  continueSocialLogin,
  fetchLoginOptions,
  isAuthenticatedSession,
  loginParticipantByName,
  loginParticipantBySKS,
  loginParticipantByTelegramWebApp,
  loginParticipantByUnionCard,
  loginParticipantByVKToken,
  socialAuthStartURL,
  type LoginOptions,
  type PublicEvent,
  type SocialLoginResult,
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
import { miniAppEventSlug, telegramWebApp } from '@/shared/lib/telegram-webapp'
import { exchangeVkOneTapCode, renderVkOneTap } from '@/shared/lib/vkid'
import { Button } from '@/shared/ui/button'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'

type BackupMethod = 'name' | 'union' | 'sks'

// Обмен кода VK и проверка на бэкенде занимают несколько секунд — без явного
// ожидания страница выглядит так, будто вход не сработал.
type SocialProgressKind = 'vk' | 'telegram' | 'session'

const socialProgressText: Record<SocialProgressKind, string> = {
  vk: 'Входим через VK ID…',
  telegram: 'Входим через Telegram…',
  session: 'Завершаем вход…',
}

function SocialProgress({ kind }: { kind: SocialProgressKind }) {
  return (
    <div role="status" className="flex items-center gap-2.5 py-2 text-[14px] text-muted">
      <Loader2 className="h-4 w-4 animate-spin text-brand" aria-hidden />
      {socialProgressText[kind]}
    </div>
  )
}

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
      case 'PARTICIPANT_NOT_LINKED':
        return 'Ваш аккаунт не привязан ни к одному активному мероприятию. Попросите организатора указать вашу ссылку или войдите резервным способом.'
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
  if (code === 'unlinked')
    return 'Ваш аккаунт не привязан ни к одному активному мероприятию. Попросите организатора указать вашу ссылку или войдите резервным способом.'
  if (code === 'social') return 'Не удалось войти через соцсеть. Проверьте, что вы есть в списке участников.'
  return ''
}

export function ParticipantSignIn({
  miniApp,
  onBack,
}: {
  miniApp: boolean
  onBack: () => void
}) {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const { acceptSession, session, status } = useParticipantAuth()
  const [options, setOptions] = useState<LoginOptions | null>(null)
  const [authError, setAuthError] = useState(socialErrorFromQuery(params.get('error')))
  const [backupOpen, setBackupOpen] = useState(false)
  const [backupMethod, setBackupMethod] = useState<BackupMethod>('name')
  const [busy, setBusy] = useState<SocialProgressKind | null>(() => {
    if (miniApp) return 'telegram'
    return params.get('continue')?.trim() ? 'session' : null
  })
  const [continueToken, setContinueToken] = useState(params.get('continue')?.trim() ?? '')
  const [matchedEvents, setMatchedEvents] = useState<PublicEvent[]>([])
  const miniAppAttempted = useRef(false)
  const continueAttempted = useRef(false)
  const preferredSlug = params.get('event')?.trim() ?? ''

  const backupEvents = options?.events ?? []
  const backupSlug = preferredSlug || (backupEvents.length === 1 ? backupEvents[0].slug : '')
  const backupEventName = useMemo(
    () => backupEvents.find((event) => event.slug === backupSlug)?.name ?? backupSlug,
    [backupEvents, backupSlug],
  )

  useEffect(() => {
    void fetchLoginOptions()
      .then(setOptions)
      .catch(() => setOptions({ telegram: { enabled: false }, vk: { enabled: false }, events: [] }))
  }, [])

  useEffect(() => {
    if (session?.event.slug) {
      navigate(`/event/${encodeURIComponent(session.event.slug)}/me`, { replace: true })
    }
  }, [navigate, session])

  useEffect(() => {
    const token = params.get('continue')?.trim()
    if (!token || status === 'loading' || continueAttempted.current) return
    continueAttempted.current = true
    setBusy('session')
    void completeSocial(() => continueSocialLogin(token))
      .then(() => {
        const next = new URLSearchParams(params)
        next.delete('continue')
        setParams(next, { replace: true })
      })
      .finally(() => setBusy(null))
    // OAuth вернул несколько мероприятий — добираем список.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status])

  useEffect(() => {
    if (status === 'loading' || !miniApp || miniAppAttempted.current) return
    miniAppAttempted.current = true
    const app = telegramWebApp()
    if (!app) {
      setBusy(null)
      return
    }
    app.ready()
    app.expand()
    const startEvent = miniAppEventSlug(app.initDataUnsafe?.start_param) || preferredSlug
    void completeSocial(() =>
      loginParticipantByTelegramWebApp(app.initData, startEvent || undefined),
    ).finally(() => setBusy(null))
    // Mini App: один автологин на запуск.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [miniApp, status])

  function applySocialResult(result: SocialLoginResult) {
    if (isAuthenticatedSession(result)) {
      acceptSession(result)
      navigate(`/event/${encodeURIComponent(result.event.slug)}/me`, { replace: true })
      return
    }
    if (result.status === 'choose_event') {
      setContinueToken(result.continue_token)
      setMatchedEvents(result.events ?? [])
      return
    }
  }

  async function completeSocial(request: () => Promise<SocialLoginResult>) {
    setAuthError('')
    try {
      applySocialResult(await request())
    } catch (error) {
      setAuthError(loginErrorMessage(error))
    }
  }

  async function completeBackup(request: () => Promise<ParticipantSession>) {
    setAuthError('')
    try {
      const next = await request()
      acceptSession(next)
      navigate(`/event/${encodeURIComponent(next.event.slug)}/me`, { replace: true })
    } catch (error) {
      setAuthError(loginErrorMessage(error))
    }
  }

  function chooseMatchedEvent(slug: string) {
    if (!continueToken) return
    void completeSocial(() => continueSocialLogin(continueToken, slug))
  }

  function chooseBackupEvent(slug: string) {
    const next = new URLSearchParams(params)
    next.set('event', slug)
    next.delete('error')
    setParams(next, { replace: true })
    setAuthError('')
  }

  const vkOn = options?.vk.enabled ?? false
  const choosing = matchedEvents.length > 0 && Boolean(continueToken)

  return (
    <div className="flex flex-col gap-4 rounded-card border border-border bg-surface p-6 shadow-subtle">
      {busy && <SocialProgress kind={busy} />}
      {!busy && choosing && (
        <EventPicker
          title="Выберите мероприятие"
          events={matchedEvents}
          loading={false}
          onPick={chooseMatchedEvent}
        />
      )}
      {/* Виджет VK держим смонтированным: размонтирование оборвёт обмен кода. */}
      <div className={busy || choosing ? 'hidden' : 'flex flex-col gap-3'}>
        <VkOneTapButton
          enabled={vkOn}
          eventSlug={preferredSlug}
          appId={options?.vk.app_id}
          redirectUrl={options?.vk.redirect_url}
          onStart={() => setBusy('vk')}
          onToken={(token) =>
            void completeSocial(() =>
              loginParticipantByVKToken(token, preferredSlug || undefined),
            ).finally(() => setBusy(null))
          }
          onError={() => {
            setBusy(null)
            setAuthError('Не удалось войти через VK. Попробуйте ещё раз.')
          }}
        />
        {!vkOn && (
          <p className="text-[13px] text-muted">
            Вход через VK ещё не подключён. Используйте резервный способ ниже.
          </p>
        )}
      </div>

      {authError && (
        <div role="alert" className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[14px] text-danger">
          {authError}
        </div>
      )}

      {!choosing && !busy && (
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
              {!backupSlug ? (
                <EventPicker
                  title="Сначала выберите мероприятие"
                  events={backupEvents}
                  loading={!options}
                  onPick={chooseBackupEvent}
                />
              ) : (
                <>
                  {backupEvents.length > 1 && (
                    <p className="text-[13px] font-medium text-brand">Мероприятие · {backupEventName}</p>
                  )}
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
                      onSubmit={(values) => completeBackup(() => loginParticipantByName(backupSlug, values))}
                    />
                  )}
                  {backupMethod === 'union' && (
                    <IdentifierLoginForm
                      method="union"
                      error=""
                      onSubmit={(values) =>
                        completeBackup(() => loginParticipantByUnionCard(backupSlug, values))
                      }
                    />
                  )}
                  {backupMethod === 'sks' && (
                    <IdentifierLoginForm
                      method="sks"
                      error=""
                      onSubmit={(values) => completeBackup(() => loginParticipantBySKS(backupSlug, values))}
                    />
                  )}
                </>
              )}
            </div>
          )}
        </div>
      )}

      {!miniApp && <BackButton onClick={onBack} />}
    </div>
  )
}

function VkOneTapButton({
  enabled,
  eventSlug,
  appId,
  redirectUrl,
  onStart,
  onToken,
  onError,
}: {
  enabled: boolean
  eventSlug?: string
  appId?: string
  redirectUrl?: string
  onStart: () => void
  onToken: (accessToken: string) => void
  onError: () => void
}) {
  const container = useRef<HTMLDivElement>(null)
  const onStartRef = useRef(onStart)
  const onTokenRef = useRef(onToken)
  const onErrorRef = useRef(onError)
  const [failed, setFailed] = useState(false)
  onStartRef.current = onStart
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
        onStartRef.current()
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
        if (!disposed) setFailed(true)
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
      <div
        ref={container}
        className={failed ? 'hidden' : 'min-h-[44px] w-full overflow-hidden'}
      />
      {failed && (
        <Button
          type="button"
          className="w-full bg-[#0077FF] hover:bg-[#0066dd]"
          onClick={() => {
            window.location.href = socialAuthStartURL('vk', eventSlug)
          }}
        >
          Войти через VK
        </Button>
      )}
    </div>
  )
}

function EventPicker({
  title,
  events,
  loading,
  onPick,
}: {
  title: string
  events: PublicEvent[]
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
      <p className="text-[14px] font-medium text-ink">{title}</p>
      {events.map((event) => (
        <button
          key={event.slug}
          type="button"
          onClick={() => onPick(event.slug)}
          className="rounded-[12px] border border-border px-3.5 py-3 text-left hover:border-brand/40 hover:bg-brand-subtle/40"
        >
          <span className="block font-medium text-ink">{event.name}</span>
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
