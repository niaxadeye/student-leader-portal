import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Camera,
  CameraOff,
  CheckCircle2,
  ImageUp,
  Keyboard,
  Users,
  XCircle,
} from 'lucide-react'
import {
  prepareZXingModule,
  Scanner,
  useDevices,
  type IDetectedBarcode,
  type IScannerError,
} from '@yudiel/react-qr-scanner'
import { BarcodeDetector } from 'barcode-detector/ponyfill'
import { toast } from 'sonner'
import zxingReaderWasmUrl from 'zxing-wasm/reader/zxing_reader.wasm?url'
import { useAdminLecture, useLectureAttendance, useScanLecture } from '@/entities/lecture/queries'
import type { ScanResult, ScannerType } from '@/entities/lecture/types'
import { lecturePeopleLine } from '@/entities/lecture/types'
import { ApiRequestError } from '@/shared/api/client'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { ErrorState, Skeleton } from '@/shared/ui/states'

prepareZXingModule({
  overrides: {
    locateFile: (path: string) => (path.endsWith('.wasm') ? zxingReaderWasmUrl : path),
  },
})

const MOBILE_CAMERA_QUERY = '(max-width: 1023px) and (hover: none) and (pointer: coarse)'

function useMobileCamera() {
  const [isMobileCamera, setIsMobileCamera] = useState(
    () => typeof window !== 'undefined' && window.matchMedia(MOBILE_CAMERA_QUERY).matches,
  )

  useEffect(() => {
    const mediaQuery = window.matchMedia(MOBILE_CAMERA_QUERY)
    const update = () => setIsMobileCamera(mediaQuery.matches)
    update()
    mediaQuery.addEventListener('change', update)
    return () => mediaQuery.removeEventListener('change', update)
  }, [])

  return isMobileCamera
}

function cameraErrorMessage(error: IScannerError) {
  switch (error.kind) {
    case 'permission-denied':
      return 'Доступ к камере запрещён. Разрешите камеру для eazytech.ru в настройках браузера.'
    case 'no-camera':
      return 'Камера не найдена. Подключите веб-камеру или загрузите изображение QR-кода.'
    case 'in-use':
      return 'Камера занята другим приложением или вкладкой. Закройте их и запустите сканер снова.'
    case 'insecure-context':
      return 'Камера доступна только по HTTPS. Откройте защищённую версию сайта.'
    case 'overconstrained':
      return 'Выбранная камера недоступна. Вернитесь к автоматическому выбору камеры.'
    case 'unsupported':
      return 'Этот браузер не предоставляет доступ к камере. Можно загрузить изображение QR-кода.'
    default:
      return `Не удалось запустить QR-сканер: ${error.message}`
  }
}

export function LectureScannerPage() {
  const { contestId, lectureId } = useParams()
  const lecture = useAdminLecture(contestId, lectureId)
  const attendance = useLectureAttendance(contestId, lectureId)
  const scan = useScanLecture(contestId ?? '', lectureId ?? '')
  const [token, setToken] = useState('')
  const [inputType, setInputType] = useState<Extract<ScannerType, 'USB' | 'MANUAL'>>('USB')
  const [lastResult, setLastResult] = useState<ScanResult | null>(null)
  const [lastError, setLastError] = useState<string | null>(null)
  const [cameraActive, setCameraActive] = useState(false)
  const [cameraError, setCameraError] = useState<string | null>(null)
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [imagePending, setImagePending] = useState(false)
  const isMobileCamera = useMobileCamera()
  const devices = useDevices()
  const inputRef = useRef<HTMLInputElement>(null)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const inFlight = useRef(false)
  const lastCameraToken = useRef<{ value: string; at: number } | null>(null)

  const cameraConstraints = useMemo<MediaTrackConstraints>(
    () =>
      selectedDeviceId
        ? { deviceId: { exact: selectedDeviceId } }
        : {
            facingMode: { ideal: 'environment' },
            width: { ideal: 1280 },
            height: { ideal: 720 },
          },
    [selectedDeviceId],
  )

  useEffect(() => {
    if (isMobileCamera) {
      inputRef.current?.blur()
      return
    }
    inputRef.current?.focus()
  }, [isMobileCamera])

  const submitToken = useCallback(
    async (raw: string, scannerType: ScannerType) => {
      const value = raw.trim()
      if (!value || inFlight.current) return
      inFlight.current = true
      setLastError(null)
      try {
        const result = await scan.mutateAsync({ token: value, scannerType })
        setLastResult(result)
        if (result.already_checked) {
          toast.warning(`${result.attendance.participant_name} уже отмечен`)
        } else {
          toast.success(
            `${result.attendance.participant_name}: +${result.attendance.points_awarded} баллов`,
          )
          navigator.vibrate?.(80)
        }
      } catch (error) {
        const message =
          error instanceof ApiRequestError ? error.message : 'Не удалось обработать QR-код'
        setLastError(message)
        setLastResult(null)
        toast.error(message)
        navigator.vibrate?.([100, 80, 100])
      } finally {
        setToken('')
        window.setTimeout(() => {
          inFlight.current = false
          if (!isMobileCamera) inputRef.current?.focus()
        }, 700)
      }
    },
    [isMobileCamera, scan],
  )

  const handleCameraScan = useCallback(
    (codes: IDetectedBarcode[]) => {
      const value = codes.find((code) => code.format === 'qr_code')?.rawValue.trim()
      if (!value) return
      const previous = lastCameraToken.current
      if (previous && previous.value === value && Date.now() - previous.at <= 2500) return
      lastCameraToken.current = { value, at: Date.now() }
      setCameraError(null)
      void submitToken(value, 'CAMERA')
    },
    [submitToken],
  )

  const handleCameraError = useCallback((error: IScannerError) => {
    setCameraError(cameraErrorMessage(error))
  }, [])

  async function scanImage(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setImagePending(true)
    setCameraError(null)
    try {
      const detector = new BarcodeDetector({ formats: ['qr_code'] })
      const codes = await detector.detect(file)
      const value = codes[0]?.rawValue.trim()
      if (!value) {
        setCameraError('На изображении не найден QR-код. Выберите более чёткое изображение.')
        return
      }
      await submitToken(value, 'MANUAL')
    } catch {
      setCameraError('Не удалось прочитать изображение. Используйте PNG/JPEG с крупным QR-кодом.')
    } finally {
      setImagePending(false)
    }
  }

  function submitInput(event: React.FormEvent) {
    event.preventDefault()
    void submitToken(token, inputType)
  }

  if (lecture.isLoading) return <Skeleton className="h-72 w-full" />
  if (lecture.isError || !lecture.data) return <ErrorState onRetry={() => lecture.refetch()} />

  const canScan = lecture.data.status === 'ACTIVE'
  const people = lecturePeopleLine(lecture.data)
  return (
    <div className="flex flex-col gap-6">
      <header>
        <Link
          to={`/admin/contests/${contestId}`}
          className="mb-3 inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink"
        >
          <ArrowLeft className="h-4 w-4" /> К мероприятию
        </Link>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-[28px] font-bold text-ink">Сканер посещений</h1>
              <Badge tone={canScan ? 'success' : 'neutral'}>
                {canScan ? 'Регистрация открыта' : 'Только история'}
              </Badge>
            </div>
            <p className="mt-1 text-muted">
              {lecture.data.title} · +{lecture.data.points} баллов
              {(lecture.data.directions?.length ?? 0) === 0
                ? ' · все направления'
                : ` · ${lecture.data.directions.map((item) => item.name).join(', ')}`}
              {people ? ` · ${people}` : ''}
            </p>
          </div>
          {canScan && isMobileCamera && (
            <Button
              variant={cameraActive ? 'secondary' : 'primary'}
              onClick={() => {
                setCameraError(null)
                inputRef.current?.blur()
                setCameraActive((value) => !value)
              }}
            >
              {cameraActive ? <CameraOff className="h-4 w-4" /> : <Camera className="h-4 w-4" />}
              {cameraActive ? 'Остановить камеру' : 'Открыть камеру'}
            </Button>
          )}
        </div>
      </header>

      {canScan && (
        <div className="grid gap-5">
          {isMobileCamera && (
            <Card className="overflow-hidden">
              <CardHeader>
                <CardTitle className="text-[19px]">Камера телефона</CardTitle>
              </CardHeader>
              <CardBody className="p-3 sm:p-5">
                <div className="relative h-[72svh] min-h-[480px] max-h-[820px] w-full overflow-hidden rounded-[18px] bg-ink">
                  {cameraActive ? (
                    <Scanner
                      onScan={handleCameraScan}
                      onError={handleCameraError}
                      constraints={cameraConstraints}
                      formats={['qr_code']}
                      paused={scan.isPending}
                      allowMultiple
                      scanDelay={1500}
                      retryDelay={120}
                      startTimeoutMs={10000}
                      sound
                      components={{ finder: false, torch: true, zoom: true, onOff: false }}
                      styles={{
                        container: { width: '100%', height: '100%', background: '#111827' },
                        video: { width: '100%', height: '100%', objectFit: 'cover' },
                      }}
                    />
                  ) : (
                    <div className="absolute inset-0 flex items-center justify-center text-center text-[13px] text-white/65">
                      Нажмите «Открыть камеру»
                    </div>
                  )}
                </div>
                {cameraActive && devices.length > 1 && (
                  <label className="mt-3 block text-[12px] font-medium text-muted">
                    Камера
                    <select
                      value={selectedDeviceId}
                      onChange={(event) => setSelectedDeviceId(event.target.value)}
                      className="mt-1 h-10 w-full rounded-[10px] border border-border bg-surface px-3 text-[13px] text-ink outline-none focus:border-brand"
                    >
                      <option value="">Автовыбор задней камеры</option>
                      {devices.map((device, index) => (
                        <option key={device.deviceId} value={device.deviceId}>
                          {device.label || `Камера ${index + 1}`}
                        </option>
                      ))}
                    </select>
                  </label>
                )}
                <input
                  ref={imageInputRef}
                  type="file"
                  accept="image/png,image/jpeg,image/webp"
                  className="hidden"
                  onChange={(event) => void scanImage(event)}
                />
                <Button
                  className="mt-3"
                  size="sm"
                  variant="ghost"
                  loading={imagePending}
                  onClick={() => imageInputRef.current?.click()}
                >
                  <ImageUp className="h-4 w-4" /> Загрузить QR из изображения
                </Button>
                {cameraError && (
                  <p className="mt-3 rounded-[10px] bg-danger/10 p-3 text-[13px] text-danger">
                    {cameraError}
                  </p>
                )}
              </CardBody>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle className="text-[19px]">HID-сканер или ручной ввод</CardTitle>
            </CardHeader>
            <CardBody>
              <div className="mb-3 flex gap-2">
                <Button
                  size="sm"
                  variant={inputType === 'USB' ? 'subtle' : 'ghost'}
                  onClick={() => setInputType('USB')}
                >
                  <Keyboard className="h-4 w-4" /> USB HID
                </Button>
                <Button
                  size="sm"
                  variant={inputType === 'MANUAL' ? 'subtle' : 'ghost'}
                  onClick={() => setInputType('MANUAL')}
                >
                  Вручную
                </Button>
              </div>
              <form onSubmit={submitInput} className="flex gap-2">
                <Input
                  ref={inputRef}
                  value={token}
                  onChange={(event) => setToken(event.target.value)}
                  placeholder="Сканируйте QR — Enter отправит код"
                  autoComplete="off"
                  disabled={cameraActive && isMobileCamera}
                  onBlur={() =>
                    window.setTimeout(() => !isMobileCamera && inputRef.current?.focus(), 0)
                  }
                />
                <Button type="submit" loading={scan.isPending}>
                  Отправить
                </Button>
              </form>
              <p className="mt-3 text-[12px] text-muted">
                {isMobileCamera
                  ? 'На телефоне поле активируется только по нажатию и отключается во время работы камеры.'
                  : 'Поле очищается и возвращает фокус после каждого результата.'}
              </p>

              {(lastResult || lastError) && (
                <div
                  className={`mt-5 rounded-[14px] p-4 ${lastError ? 'bg-danger/10' : lastResult?.already_checked ? 'bg-amber-50' : 'bg-success/10'}`}
                >
                  <div className="flex items-start gap-3">
                    {lastError ? (
                      <XCircle className="h-6 w-6 shrink-0 text-danger" />
                    ) : (
                      <CheckCircle2 className="h-6 w-6 shrink-0 text-success" />
                    )}
                    <div>
                      <p className="font-semibold text-ink">
                        {lastError ?? lastResult?.attendance.participant_name}
                      </p>
                      {lastResult && (
                        <p className="mt-1 text-[13px] text-muted">
                          {lastResult.already_checked
                            ? 'Посещение зарегистрировано ранее'
                            : `Начислено ${lastResult.attendance.points_awarded} баллов`}
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </CardBody>
          </Card>
        </div>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Users className="h-5 w-5 text-brand" />
            <CardTitle className="text-[19px]">Посещения</CardTitle>
          </div>
          <Badge>{attendance.data?.length ?? 0}</Badge>
        </CardHeader>
        <CardBody>
          {attendance.isLoading && <Skeleton className="h-24 w-full" />}
          {attendance.isError && <ErrorState onRetry={() => attendance.refetch()} />}
          {attendance.data?.length === 0 && (
            <p className="rounded-[12px] bg-surface-2 p-5 text-center text-[13px] text-muted">
              Пока никто не отмечен.
            </p>
          )}
          {!!attendance.data?.length && (
            <div className="divide-y divide-border">
              {attendance.data.map((item) => (
                <div
                  key={item.id}
                  className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0"
                >
                  <div>
                    <p className="text-[14px] font-medium text-ink">{item.participant_name}</p>
                    <p className="mt-0.5 text-[12px] text-muted">
                      {formatDateTime(item.created_at)} · {item.scanner_type}
                    </p>
                  </div>
                  <span className="font-semibold text-success">+{item.points_awarded}</span>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  )
}
