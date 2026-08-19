const VKID_SDK_SRC = 'https://unpkg.com/@vkid/sdk@2.6.1/dist-sdk/umd/index.js'

interface VKIDTokenResult {
  access_token: string
  user_id?: number | string
}

interface VKIDChain {
  on: (event: unknown, handler: (payload: { code: string; device_id: string }) => void) => VKIDChain
}

interface VKIDOneTap {
  render: (options: { container: HTMLElement; showAlternativeLogin?: boolean }) => VKIDChain
}

interface VKIDSDK {
  Config: {
    init: (config: Record<string, unknown>) => void
  }
  ConfigResponseMode: { Callback: unknown }
  ConfigSource: { LOWCODE: unknown }
  OneTap: new () => VKIDOneTap
  WidgetEvents: { ERROR: unknown }
  OneTapInternalEvents: { LOGIN_SUCCESS: unknown }
  Auth: {
    exchangeCode: (code: string, deviceId: string) => Promise<VKIDTokenResult>
  }
}

declare global {
  interface Window {
    VKIDSDK?: VKIDSDK
  }
}

let sdkPromise: Promise<VKIDSDK> | null = null

function loadVkIdSdk(): Promise<VKIDSDK> {
  if (window.VKIDSDK) return Promise.resolve(window.VKIDSDK)
  if (sdkPromise) return sdkPromise
  sdkPromise = new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(`script[src="${VKID_SDK_SRC}"]`)
    if (existing) {
      existing.addEventListener('load', () => {
        if (window.VKIDSDK) resolve(window.VKIDSDK)
        else reject(new Error('VK ID SDK не загрузился'))
      })
      existing.addEventListener('error', () => reject(new Error('VK ID SDK не загрузился')))
      return
    }
    const script = document.createElement('script')
    script.src = VKID_SDK_SRC
    script.async = true
    script.onload = () => {
      if (window.VKIDSDK) resolve(window.VKIDSDK)
      else reject(new Error('VK ID SDK не загрузился'))
    }
    script.onerror = () => reject(new Error('VK ID SDK не загрузился'))
    document.head.appendChild(script)
  })
  return sdkPromise
}

export async function exchangeVkOneTapCode(code: string, deviceId: string): Promise<string> {
  const VKID = await loadVkIdSdk()
  const data = await VKID.Auth.exchangeCode(code, deviceId)
  const token = data.access_token?.trim()
  if (!token) throw new Error('VK ID не вернул access_token')
  return token
}

export async function renderVkOneTap(options: {
  container: HTMLElement
  appId: string
  redirectUrl: string
  onCode: (code: string, deviceId: string) => void
  onError: (error: unknown) => void
}): Promise<() => void> {
  const VKID = await loadVkIdSdk()
  VKID.Config.init({
    app: Number(options.appId),
    redirectUrl: options.redirectUrl,
    responseMode: VKID.ConfigResponseMode.Callback,
    source: VKID.ConfigSource.LOWCODE,
    scope: '',
  })
  const oneTap = new VKID.OneTap()
  oneTap
    .render({ container: options.container, showAlternativeLogin: true })
    .on(VKID.WidgetEvents.ERROR, (error) => options.onError(error))
    .on(VKID.OneTapInternalEvents.LOGIN_SUCCESS, (payload) => {
      options.onCode(payload.code, payload.device_id)
    })
  return () => {
    options.container.replaceChildren()
  }
}
