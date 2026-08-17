import { useState, type ComponentType } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { IdCard, ScanLine, ShieldCheck, UserRound } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  loginParticipantByName,
  loginParticipantBySKS,
  loginParticipantByUnionCard,
} from '@/entities/event-participant/api'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import type { ParticipantSession } from '@/entities/event-participant/types'
import {
  participantIdentifierLoginSchema,
  participantNameLoginSchema,
  type ParticipantIdentifierLoginValues,
  type ParticipantNameLoginValues,
} from '@/features/participant-auth/login-schema'
import { ApiRequestError } from '@/shared/api/client'
import { cn } from '@/shared/lib/cn'
import { Button } from '@/shared/ui/button'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'

type LoginMethod = 'name' | 'union' | 'sks'

const methods: Array<{
  id: LoginMethod
  label: string
  shortLabel: string
  icon: ComponentType<{ className?: string }>
}> = [
  { id: 'name', label: 'ФИО и дата рождения', shortLabel: 'По ФИО', icon: UserRound },
  { id: 'union', label: 'Номер профсоюзного билета', shortLabel: 'Профбилет', icon: IdCard },
  { id: 'sks', label: 'Barcode СКС РФ', shortLabel: 'Barcode СКС', icon: ScanLine },
]

function loginErrorMessage(error: unknown): string {
  if (error instanceof ApiRequestError) {
    switch (error.code) {
      case 'PARTICIPANT_IDENTITY_AMBIGUOUS':
        return 'Найдено несколько совпадений. Войдите по профбилету или barcode СКС.'
      case 'RATE_LIMIT_EXCEEDED':
        return 'Слишком много попыток. Подождите несколько минут и попробуйте снова.'
      case 'PARTICIPANT_AUTH_FAILED':
        return 'Участник с такими данными не найден. Проверьте введённые значения.'
      case 'NETWORK_ERROR':
        return 'Нет связи с сервером. Проверьте подключение и попробуйте снова.'
    }
  }
  return 'Не удалось выполнить вход. Попробуйте ещё раз.'
}

function AuthError({ message }: { message: string }) {
  if (!message) return null
  return (
    <div role="alert" className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[14px] text-danger">
      {message}
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
          <Input
            {...props}
            autoFocus
            autoComplete="name"
            placeholder="Иванов Иван Иванович"
            {...register('full_name')}
          />
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
      <AuthError message={error} />
      <Button type="submit" loading={isSubmitting} className="w-full">
        Войти в кабинет
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
        description={
          isScanner
            ? 'Введите код вручную или отсканируйте карту USB-сканером — форма отправится по Enter.'
            : 'Введите номер так, как он указан в профсоюзном билете.'
        }
        required
        error={errors.value?.message}
      >
        {(props) => (
          <Input
            {...props}
            autoFocus
            autoComplete="off"
            placeholder={isScanner ? 'Отсканируйте или введите barcode' : 'Номер профбилета'}
            {...register('value')}
          />
        )}
      </Field>
      <AuthError message={error} />
      <Button type="submit" loading={isSubmitting} className="w-full">
        {isScanner ? 'Войти по barcode' : 'Войти по профбилету'}
      </Button>
    </form>
  )
}

export function ParticipantLoginPage() {
  const { eventSlug = '' } = useParams()
  const navigate = useNavigate()
  const { acceptSession } = useParticipantAuth()
  const [method, setMethod] = useState<LoginMethod>('name')
  const [authError, setAuthError] = useState('')

  async function completeLogin(request: () => Promise<ParticipantSession>) {
    setAuthError('')
    try {
      const session = await request()
      acceptSession(session)
      navigate(`/event/${encodeURIComponent(session.event.slug)}/me`, { replace: true })
    } catch (error) {
      setAuthError(loginErrorMessage(error))
    }
  }

  function selectMethod(next: LoginMethod) {
    setMethod(next)
    setAuthError('')
  }

  return (
    <div className="min-h-screen bg-surface-2 px-4 py-8 sm:py-12">
      <main className="mx-auto w-full max-w-[560px]">
        <div className="mb-7 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-btn bg-brand text-white">
            <ShieldCheck className="h-6 w-6" aria-hidden />
          </div>
          <p className="text-[13px] font-semibold uppercase tracking-[0.12em] text-brand">
            Мероприятие · {eventSlug}
          </p>
          <h1 className="mt-2 text-[28px] font-bold text-ink sm:text-[32px]">Вход участника</h1>
          <p className="mx-auto mt-2 max-w-md text-[15px] text-muted">
            Выберите удобный способ. Сессия участника отделена от кабинета сотрудников.
          </p>
        </div>

        <section className="rounded-card border border-border bg-surface p-5 shadow-subtle sm:p-6">
          <div
            role="tablist"
            aria-label="Способ входа"
            className="mb-6 grid grid-cols-3 gap-1 rounded-[12px] bg-surface-2 p-1"
          >
            {methods.map(({ id, shortLabel, icon: Icon }) => (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={method === id}
                onClick={() => selectMethod(id)}
                className={cn(
                  'flex min-h-12 flex-col items-center justify-center gap-1 rounded-[9px] px-1.5 py-2 text-[12px] font-medium transition-colors sm:flex-row sm:text-[13px]',
                  method === id
                    ? 'bg-surface text-brand shadow-micro'
                    : 'text-muted hover:text-ink',
                )}
              >
                <Icon className="h-4 w-4" aria-hidden />
                {shortLabel}
              </button>
            ))}
          </div>

          <div role="tabpanel" aria-label={methods.find((item) => item.id === method)?.label}>
            {method === 'name' && (
              <NameLoginForm
                error={authError}
                onSubmit={(values) =>
                  completeLogin(() => loginParticipantByName(eventSlug, values))
                }
              />
            )}
            {method === 'union' && (
              <IdentifierLoginForm
                method="union"
                error={authError}
                onSubmit={(values) =>
                  completeLogin(() => loginParticipantByUnionCard(eventSlug, values))
                }
              />
            )}
            {method === 'sks' && (
              <IdentifierLoginForm
                method="sks"
                error={authError}
                onSubmit={(values) => completeLogin(() => loginParticipantBySKS(eventSlug, values))}
              />
            )}
          </div>
        </section>

        <p className="mt-5 text-center text-[13px] text-muted">
          Вы сотрудник?{' '}
          <Link to="/login" className="font-medium text-brand hover:text-brand-dark">
            Войти в административный кабинет
          </Link>
        </p>
      </main>
    </div>
  )
}
