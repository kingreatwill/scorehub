export const THEME_COLOR_KEY = 'scorehub.theme.color'
export const LEGACY_THEME_COLOR_KEY = 'scorehub.my.colorDot'
export const THEME_ALPHA_KEY = 'scorehub.theme.alpha'
export const DEFAULT_THEME_COLOR = '#111111'

const TAB_PAGE_ROUTES = ['pages/home/index', 'pages/ledger/index', 'pages/my/index']

let cachedThemeBaseColor = ''
let cachedThemeAlpha = 1
let lastNavApplyKey = ''
let lastTabSelectedColor = ''
let lastCustomTabSyncKey = ''

export function normalizeHexColor(raw: string): string {
  const v = String(raw || '').trim().toUpperCase()
  if (/^#[0-9A-F]{6}$/.test(v)) return v
  const short = v.match(/^#([0-9A-F])([0-9A-F])([0-9A-F])$/)
  if (short) return `#${short[1]}${short[1]}${short[2]}${short[2]}${short[3]}${short[3]}`
  const longWithAlpha = v.match(/^#([0-9A-F]{8})$/)
  if (longWithAlpha) return `#${longWithAlpha[1].slice(0, 6)}`
  const shortWithAlpha = v.match(/^#([0-9A-F])([0-9A-F])([0-9A-F])([0-9A-F])$/)
  if (shortWithAlpha) return `#${shortWithAlpha[1]}${shortWithAlpha[1]}${shortWithAlpha[2]}${shortWithAlpha[2]}${shortWithAlpha[3]}${shortWithAlpha[3]}`
  return ''
}

function clampAlpha(raw: unknown): number {
  const n = typeof raw === 'number' ? raw : Number.parseFloat(String(raw ?? ''))
  if (!Number.isFinite(n)) return 1
  return Math.max(0, Math.min(1, n))
}

function alphaToHex(alpha: number): string {
  const n = Math.round(clampAlpha(alpha) * 255)
  return n.toString(16).padStart(2, '0').toUpperCase()
}

function extractAlphaFromColor(raw: string): number {
  const v = String(raw || '').trim().toUpperCase()
  const longWithAlpha = v.match(/^#([0-9A-F]{8})$/)
  if (longWithAlpha) return Number.parseInt(longWithAlpha[1].slice(6, 8), 16) / 255
  const shortWithAlpha = v.match(/^#([0-9A-F])([0-9A-F])([0-9A-F])([0-9A-F])$/)
  if (shortWithAlpha) {
    const aa = `${shortWithAlpha[4]}${shortWithAlpha[4]}`
    return Number.parseInt(aa, 16) / 255
  }
  return 1
}

function withAlpha(hex: string, alpha: number): string {
  const base = normalizeHexColor(hex) || DEFAULT_THEME_COLOR
  const a = clampAlpha(alpha)
  if (a >= 0.999) return base
  return `${base}${alphaToHex(a)}`
}

export function getThemeBaseColor(): string {
  if (cachedThemeBaseColor) return cachedThemeBaseColor
  const currentRaw =
    String((uni.getStorageSync(THEME_COLOR_KEY) as any) || '').trim() ||
    String((uni.getStorageSync(LEGACY_THEME_COLOR_KEY) as any) || '').trim()
  cachedThemeBaseColor = normalizeHexColor(currentRaw) || DEFAULT_THEME_COLOR
  return cachedThemeBaseColor
}

function rgbToHex(r: number, g: number, b: number): string {
  const toHex = (n: number) => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0').toUpperCase()
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`
}

function hexToRgb(hex: string): { r: number; g: number; b: number } {
  const normalized = normalizeHexColor(hex) || DEFAULT_THEME_COLOR
  return {
    r: Number.parseInt(normalized.slice(1, 3), 16),
    g: Number.parseInt(normalized.slice(3, 5), 16),
    b: Number.parseInt(normalized.slice(5, 7), 16),
  }
}

function hexToRgbaString(hex: string, alpha: number): string {
  const { r, g, b } = hexToRgb(hex)
  const a = Math.max(0, Math.min(1, alpha))
  return `rgba(${r}, ${g}, ${b}, ${a})`
}

function mixHex(a: string, b: string, ratio: number): string {
  const aa = normalizeHexColor(a) || DEFAULT_THEME_COLOR
  const bb = normalizeHexColor(b) || '#000000'
  const t = Math.max(0, Math.min(1, ratio))
  const ar = Number.parseInt(aa.slice(1, 3), 16)
  const ag = Number.parseInt(aa.slice(3, 5), 16)
  const ab = Number.parseInt(aa.slice(5, 7), 16)
  const br = Number.parseInt(bb.slice(1, 3), 16)
  const bg = Number.parseInt(bb.slice(3, 5), 16)
  const bbv = Number.parseInt(bb.slice(5, 7), 16)
  return rgbToHex(ar + (br - ar) * t, ag + (bg - ag) * t, ab + (bbv - ab) * t)
}

export function darkenHex(hex: string, ratio: number): string {
  return mixHex(hex, '#000000', ratio)
}

export function lightenHex(hex: string, ratio: number): string {
  return mixHex(hex, '#FFFFFF', ratio)
}

export function buildThemeVars(base: string): Record<string, string> {
  const color = normalizeHexColor(base) || DEFAULT_THEME_COLOR
  const isDefaultTheme = color === DEFAULT_THEME_COLOR
  const vars: Record<string, string> = {
    '--brand-1': darkenHex(color, 0.26),
    '--brand-2': lightenHex(color, 0.14),
    '--brand-strong': darkenHex(color, 0.1),
    '--brand-solid': color,
    '--brand-soft': lightenHex(color, 0.82),
  }
  if (!isDefaultTheme) {
    vars['--confirm-btn-bg-rgba'] = hexToRgbaString(color, 0.9)
    vars['--confirm-btn-color'] = '#FFFFFF'
  }
  return vars
}

function navFrontColor(hex: string): '#000000' | '#ffffff' {
  const normalized = normalizeHexColor(hex) || DEFAULT_THEME_COLOR
  const r = Number.parseInt(normalized.slice(1, 3), 16)
  const g = Number.parseInt(normalized.slice(3, 5), 16)
  const b = Number.parseInt(normalized.slice(5, 7), 16)
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255
  return luminance > 0.62 ? '#000000' : '#ffffff'
}

function currentRouteFromPages(): string {
  const pages = (typeof getCurrentPages === 'function' ? (getCurrentPages() as any[]) : []) || []
  return String(pages[pages.length - 1]?.route || '')
}

function tabIndexByRoute(route: string): number {
  return TAB_PAGE_ROUTES.indexOf(String(route || ''))
}

export function applyNavigationBarTheme(base?: string) {
  const bg = normalizeHexColor(base || '') || normalizeHexColor(getThemeBaseColor()) || DEFAULT_THEME_COLOR
  const front = navFrontColor(bg)
  const applyKey = `${currentRouteFromPages()}|${bg}|${front}`
  if (lastNavApplyKey === applyKey) return
  lastNavApplyKey = applyKey
  uni.setNavigationBarColor({
    frontColor: front,
    backgroundColor: bg,
    animation: { duration: 0, timingFunc: 'linear' },
  } as any)
}

export function syncCurrentPageCustomTabBar(base?: string, pageVm?: any) {
  const pages = (typeof getCurrentPages === 'function' ? (getCurrentPages() as any[]) : []) || []
  const currentPage = pageVm || pages[pages.length - 1]
  const route = String(currentPage?.route || '')
  const selected = tabIndexByRoute(route)
  if (selected < 0) return
  const themeColor = normalizeHexColor(base || '') || normalizeHexColor(getThemeBaseColor()) || DEFAULT_THEME_COLOR
  const syncKey = `${route}|${selected}|${themeColor}`
  if (lastCustomTabSyncKey === syncKey) return
  lastCustomTabSyncKey = syncKey
  const tabBar = typeof currentPage?.getTabBar === 'function' ? currentPage.getTabBar() : null
  if (!tabBar) return
  try {
    if (typeof tabBar.syncState === 'function') {
      tabBar.syncState({ selected, themeColor })
    } else if (typeof tabBar.setData === 'function') {
      tabBar.setData({ selected, themeColor })
    }
  } catch (e) {}
}

export function applyTabBarTheme(base?: string) {
  const selectedColor = normalizeHexColor(base || '') || normalizeHexColor(getThemeBaseColor()) || DEFAULT_THEME_COLOR
  lastTabSelectedColor = selectedColor
  syncCurrentPageCustomTabBar(selectedColor)
}

export function saveThemeColor(base: string): string {
  const normalized = normalizeHexColor(base) || DEFAULT_THEME_COLOR
  uni.setStorageSync(THEME_COLOR_KEY, normalized)
  uni.setStorageSync(LEGACY_THEME_COLOR_KEY, normalized)
  uni.removeStorageSync(THEME_ALPHA_KEY)
  cachedThemeBaseColor = normalized
  return normalized
}

export function bootstrapTheme(): string {
  const base = getThemeBaseColor()
  applyNavigationBarTheme(base)
  applyTabBarTheme(base)
  return base
}
