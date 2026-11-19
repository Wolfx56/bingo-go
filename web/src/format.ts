const dateTimeFmt = new Intl.DateTimeFormat('fr-FR', { dateStyle: 'long', timeStyle: 'short' })
const timeFmt = new Intl.DateTimeFormat('fr-FR', { hour: '2-digit', minute: '2-digit' })
const dayFmt = new Intl.DateTimeFormat('fr-FR', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})
const relativeFmt = new Intl.RelativeTimeFormat('fr', { numeric: 'auto' })

export const formatDateTime = (iso: string) => dateTimeFmt.format(new Date(iso))
export const formatTime = (iso: string) => timeFmt.format(new Date(iso))

/** Per-day grouping key, in local time. */
export function dayKey(iso: string): string {
  const d = new Date(iso)
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
}

export function dayLabel(iso: string): string {
  const today = new Date()
  const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1)
  const key = dayKey(iso)
  if (key === dayKey(today.toISOString())) return 'Aujourd’hui'
  if (key === dayKey(yesterday.toISOString())) return 'Hier'
  return dayFmt.format(new Date(iso))
}

export function formatRelative(iso: string): string {
  const seconds = Math.round((new Date(iso).getTime() - Date.now()) / 1000)
  const abs = Math.abs(seconds)
  if (abs < 45) return 'à l’instant'
  if (abs < 3600) return relativeFmt.format(Math.round(seconds / 60), 'minute')
  if (abs < 86400) return relativeFmt.format(Math.round(seconds / 3600), 'hour')
  if (abs < 86400 * 30) return relativeFmt.format(Math.round(seconds / 86400), 'day')
  if (abs < 86400 * 365) return relativeFmt.format(Math.round(seconds / (86400 * 30)), 'month')
  return relativeFmt.format(Math.round(seconds / (86400 * 365)), 'year')
}

export const plural = (n: number, one: string, many: string) => `${n} ${n > 1 ? many : one}`
