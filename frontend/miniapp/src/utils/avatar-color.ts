type AvatarColor = {
  bg: string
  fg: string
}

const palette: AvatarColor[] = [
  { bg: '#FFE4E6', fg: '#BE123C' },
  { bg: '#FCE7F3', fg: '#DB2777' },
  { bg: '#F3E8FF', fg: '#7C3AED' },
  { bg: '#EDE9FE', fg: '#6D28D9' },
  { bg: '#E0F2FE', fg: '#0369A1' },
  { bg: '#E0F7FA', fg: '#0E7490' },
  { bg: '#DCFCE7', fg: '#15803D' },
  { bg: '#ECFCCB', fg: '#3F6212' },
  { bg: '#FEF3C7', fg: '#B45309' },
  { bg: '#FFE4D5', fg: '#C2410C' },
  { bg: '#E2E8F0', fg: '#334155' },
  { bg: '#FEE2E2', fg: '#B91C1C' },
]

function hashSeed(seed: string): number {
  let hash = 0
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash << 5) - hash + seed.charCodeAt(i)
    hash |= 0
  }
  return hash
}

function normalizeSeed(input: string): string {
  const s = String(input || '').trim()
  return s || '友'
}

export function pickAvatarColor(seed: string): AvatarColor {
  const normalized = normalizeSeed(seed)
  const hash = Math.abs(hashSeed(normalized))
  const index = hash % palette.length
  return palette[index]
}

export function avatarStyle(seed: string): Record<string, string> {
  const { bg, fg } = pickAvatarColor(seed)
  return {
    backgroundColor: bg,
    color: fg,
  }
}
