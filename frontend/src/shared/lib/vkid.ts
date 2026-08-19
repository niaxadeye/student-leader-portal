import {
  Auth,
  Config,
  ConfigResponseMode,
  ConfigSource,
  OneTap,
  OneTapInternalEvents,
  WidgetEvents,
} from '@vkid/sdk'

export async function exchangeVkOneTapCode(code: string, deviceId: string): Promise<string> {
  const data = await Auth.exchangeCode(code, deviceId)
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
  Config.init({
    app: Number(options.appId),
    redirectUrl: options.redirectUrl,
    responseMode: ConfigResponseMode.Callback,
    source: ConfigSource.LOWCODE,
    scope: '',
  })
  const oneTap = new OneTap()
  oneTap
    .render({ container: options.container, showAlternativeLogin: true })
    .on(WidgetEvents.ERROR, (error: unknown) => options.onError(error))
    .on(OneTapInternalEvents.LOGIN_SUCCESS, (payload: { code: string; device_id: string }) => {
      options.onCode(payload.code, payload.device_id)
    })
  return () => {
    oneTap.close()
    options.container.replaceChildren()
  }
}
