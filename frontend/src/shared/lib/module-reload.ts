const MODULE_RELOAD_KEY = 'student-leader:last-module-reload'
const MODULE_RELOAD_COOLDOWN_MS = 60_000

/**
 * Allows one automatic refresh for a stale or failed lazy chunk. The cooldown
 * prevents a compatibility/network failure from turning into a reload loop.
 */
export function markModuleReloadAttempt(now = Date.now()): boolean {
  try {
    const previous = Number(window.sessionStorage.getItem(MODULE_RELOAD_KEY) ?? 0)
    if (Number.isFinite(previous) && now - previous < MODULE_RELOAD_COOLDOWN_MS) {
      return false
    }
    window.sessionStorage.setItem(MODULE_RELOAD_KEY, String(now))
    return true
  } catch {
    return false
  }
}

export function isModuleLoadError(error: unknown): boolean {
  const message =
    error instanceof Error
      ? `${error.name}: ${error.message}`
      : typeof error === 'string'
        ? error
        : ''

  return [
    'importing a module script failed',
    'failed to fetch dynamically imported module',
    'error loading dynamically imported module',
    'unable to preload css',
  ].some((pattern) => message.toLowerCase().includes(pattern))
}
