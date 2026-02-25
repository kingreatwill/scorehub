function normalizeBase(v: string): string {
  return String(v || '').trim().replace(/\/+$/, '')
}

function parseBaseFromAPI(v: string): string {
  const raw = normalizeBase(v)
  if (!raw) return ''
  const m = raw.match(/^(https?:\/\/[^/]+)(\/.*)?$/i)
  if (!m) return ''
  const origin = normalizeBase(m[1] || '')
  const pathname = String(m[2] || '')
    .replace(/\/api\/v\d+$/i, '')
    .replace(/\/api$/i, '')
    .replace(/\/+$/, '')
  return normalizeBase(`${origin}${pathname}`)
}

function h5RuntimeBase(): string {
  // #ifndef H5
  return ''
  // #endif

  // #ifdef H5
  if (typeof window === 'undefined') return ''
  const origin = normalizeBase(window.location?.origin || '')
  if (!origin) return ''
  return origin
  // #endif
}

export function resolveInviteH5BaseURL(): string {
  const envBase = normalizeBase(String(import.meta.env.VITE_SCOREHUB_H5_BASE || ''))
  if (envBase) return envBase

  const runtimeBase = h5RuntimeBase()
  if (runtimeBase) return runtimeBase

  const fromAPI = parseBaseFromAPI(String(import.meta.env.VITE_SCOREHUB_API_BASE || ''))
  if (fromAPI) return fromAPI

  return 'https://wxapi.wcoder.com'
}

export function buildInviteBrowserJoinURL(inviteCode: string): string {
  const code = String(inviteCode || '').trim()
  if (!code) return ''
  const base = resolveInviteH5BaseURL()
  return `${base}/#/pages/join/index?code=${encodeURIComponent(code)}`
}
