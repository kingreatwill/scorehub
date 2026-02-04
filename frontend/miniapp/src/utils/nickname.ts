export const DEFAULT_NICKNAME_MAX_UNITS = 12

function unitSizeOfChar(ch: string): number {
  const codePoint = ch.codePointAt(0) ?? 0
  return codePoint <= 0x7f ? 1 : 2
}

export function nicknameUnits(input: string): number {
  let units = 0
  for (const ch of String(input || '')) units += unitSizeOfChar(ch)
  return units
}

export function clampNickname(input: string, maxUnits: number = DEFAULT_NICKNAME_MAX_UNITS): string {
  const raw = String(input || '')
  if (maxUnits <= 0 || !raw) return ''

  let used = 0
  let out = ''
  for (const ch of raw) {
    const next = used + unitSizeOfChar(ch)
    if (next > maxUnits) break
    used = next
    out += ch
  }
  return out
}

