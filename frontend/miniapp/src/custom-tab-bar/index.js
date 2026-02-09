const DEFAULT_THEME_COLOR = '#111111'
const DEFAULT_INACTIVE_COLOR = '#8A8A8A'
const iconCache = Object.create(null)

function normalizeHexColor(raw) {
  const v = String(raw || '').trim().toUpperCase()
  if (/^#[0-9A-F]{6}$/.test(v)) return v
  const short = v.match(/^#([0-9A-F])([0-9A-F])([0-9A-F])$/)
  if (short) return `#${short[1]}${short[1]}${short[2]}${short[2]}${short[3]}${short[3]}`
  return ''
}

function iconSvg(icon, color) {
  if (icon === 'scorebook') {
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6a2 2 0 0 1 2-2h12v16H6a2 2 0 0 0-2 2z"/><path d="M8 4v16"/><path d="M11 8h5"/><path d="M11 12h5"/></svg>`
  }
  if (icon === 'ledger') {
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="6" width="18" height="12" rx="2"/><path d="M3 10h18"/><circle cx="16.5" cy="14" r="1"/></svg>`
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="8" r="3.5"/><path d="M5 19c1.7-3 4-4.5 7-4.5s5.3 1.5 7 4.5"/></svg>`
}

function iconDataUrl(icon, color) {
  const c = normalizeHexColor(color) || DEFAULT_THEME_COLOR
  const key = `${icon}|${c}`
  if (iconCache[key]) return iconCache[key]
  const svg = iconSvg(icon, c)
  const url = `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
  iconCache[key] = url
  return url
}

Component({
  data: {
    selected: 0,
    themeColor: DEFAULT_THEME_COLOR,
    inactiveColor: DEFAULT_INACTIVE_COLOR,
    list: [
      { text: '得分簿', pagePath: '/pages/home/index', route: 'pages/home/index', icon: 'scorebook', activeIcon: '', inactiveIcon: '' },
      { text: '记账簿', pagePath: '/pages/ledger/index', route: 'pages/ledger/index', icon: 'ledger', activeIcon: '', inactiveIcon: '' },
      { text: '我的', pagePath: '/pages/my/index', route: 'pages/my/index', icon: 'my', activeIcon: '', inactiveIcon: '' },
    ],
  },
  lifetimes: {
    attached() {
      this._refreshIcons(this.data.themeColor, this.data.inactiveColor)
      this._syncSelectedByRoute()
    },
  },
  methods: {
    onTap(e) {
      const idx = Number(e?.currentTarget?.dataset?.index || 0)
      if (Number.isNaN(idx) || idx < 0 || idx >= this.data.list.length) return
      if (idx === this.data.selected) return
      const target = this.data.list[idx]
      if (!target?.pagePath) return
      wx.switchTab({ url: target.pagePath })
    },
    _syncSelectedByRoute() {
      const pages = typeof getCurrentPages === 'function' ? getCurrentPages() : []
      const route = String(pages?.[pages.length - 1]?.route || '')
      const idx = this.data.list.findIndex((it) => it.route === route)
      if (idx >= 0 && idx !== this.data.selected) {
        this.setData({ selected: idx })
      }
    },
    syncState(payload) {
      const nextTheme = normalizeHexColor(payload?.themeColor) || this.data.themeColor || DEFAULT_THEME_COLOR
      const nextSelected = Number(payload?.selected)
      const selected = Number.isFinite(nextSelected) && nextSelected >= 0 && nextSelected < this.data.list.length ? nextSelected : this.data.selected
      const changedTheme = nextTheme !== this.data.themeColor
      const changedSelected = selected !== this.data.selected
      if (!changedTheme && !changedSelected) return
      const patch = {}
      if (changedTheme) patch.themeColor = nextTheme
      if (changedSelected) patch.selected = selected
      this.setData(patch, () => {
        if (changedTheme) this._refreshIcons(this.data.themeColor, this.data.inactiveColor)
      })
    },
    _refreshIcons(themeRaw, inactiveRaw) {
      const theme = normalizeHexColor(themeRaw) || DEFAULT_THEME_COLOR
      const inactive = normalizeHexColor(inactiveRaw) || DEFAULT_INACTIVE_COLOR
      const next = this.data.list.map((it) => ({
        ...it,
        activeIcon: iconDataUrl(it.icon, theme),
        inactiveIcon: iconDataUrl(it.icon, inactive),
      }))
      this.setData({ list: next })
    },
  },
})
