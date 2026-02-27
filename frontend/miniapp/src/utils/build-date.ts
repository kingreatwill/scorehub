function normalizeYmd(raw: string): string {
  const v = String(raw || '').trim()
  const m = v.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (!m) return ''
  const y = Number(m[1])
  const mm = Number(m[2])
  const dd = Number(m[3])
  if (!Number.isFinite(y) || !Number.isFinite(mm) || !Number.isFinite(dd)) return ''
  if (mm < 1 || mm > 12 || dd < 1 || dd > 31) return ''
  return `${String(y).padStart(4, '0')}-${String(mm).padStart(2, '0')}-${String(dd).padStart(2, '0')}`
}

function todayYmdLocal(): string {
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

export function isBuildDateToday(): boolean {
  const buildDate = normalizeYmd(String(import.meta.env.VITE_SCOREHUB_BUILD_DATE || ''))
  if (!buildDate) return false
  return buildDate === todayYmdLocal()
}

