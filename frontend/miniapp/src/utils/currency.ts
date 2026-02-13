export type CurrencyMeta = {
  code: string
  label: string
  symbol: string
  icon: string
}

const CNY_ICON =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M7 9h10"/><path d="M9 12h6"/><path d="M10.5 6v12"/></svg>'
const USD_ICON =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M8 12h7a2.5 2.5 0 0 0 0-5h-1.5"/><path d="M9 7.5h4.5"/><path d="M9 16h4.5"/><path d="M11.5 7.5v9"/></svg>'

export const currencyList: CurrencyMeta[] = [
  { code: 'CNY', label: '人民币', symbol: '¥', icon: CNY_ICON },
  { code: 'USD', label: '美元', symbol: '$', icon: USD_ICON },
]

export function normalizeCurrency(code: string): string {
  const raw = String(code || '').toUpperCase()
  if (currencyList.some((item) => item.code === raw)) return raw
  return 'CNY'
}

export function getCurrencyMeta(code: string): CurrencyMeta {
  const normalized = normalizeCurrency(code)
  return currencyList.find((item) => item.code === normalized) || currencyList[0]
}
