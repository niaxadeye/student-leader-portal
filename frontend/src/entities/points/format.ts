export function formatPoints(value: number): string {
  return new Intl.NumberFormat('ru-RU').format(value)
}

export function signedPoints(value: number): string {
  return `${value > 0 ? '+' : ''}${formatPoints(value)}`
}
