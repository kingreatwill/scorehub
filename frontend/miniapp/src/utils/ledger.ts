export type LedgerStatus = 'recording' | 'ended'

export type LedgerMember = {
  id: string
  nickname: string
  avatarUrl?: string
  createdAt: string
}

export type LedgerRecord = {
  id: string
  memberId: string
  type: 'income' | 'expense'
  amount: number
  note?: string
  createdAt: string
}

export type LedgerBook = {
  id: string
  name: string
  status: LedgerStatus
  createdAt: string
  updatedAt: string
  endedAt?: string
  members: LedgerMember[]
  records: LedgerRecord[]
}

const STORAGE_KEY = 'scorehub.ledger.books.v1'

function nowIso(): string {
  return new Date().toISOString()
}

function safeArray<T>(input: any): T[] {
  return Array.isArray(input) ? input : []
}

function normalizeLedger(raw: any): LedgerBook | null {
  if (!raw || !raw.id) return null
  const createdAt = String(raw.createdAt || nowIso())
  const updatedAt = String(raw.updatedAt || createdAt)
  return {
    id: String(raw.id),
    name: String(raw.name || ''),
    status: raw.status === 'ended' ? 'ended' : 'recording',
    createdAt,
    updatedAt,
    endedAt: raw.endedAt ? String(raw.endedAt) : undefined,
    members: safeArray<LedgerMember>(raw.members),
    records: safeArray<LedgerRecord>(raw.records),
  }
}

function readAll(): LedgerBook[] {
  const raw = uni.getStorageSync(STORAGE_KEY)
  if (!raw) return []
  try {
    const list = Array.isArray(raw) ? raw : typeof raw === 'string' ? JSON.parse(raw) : []
    if (!Array.isArray(list)) return []
    return list.map(normalizeLedger).filter(Boolean) as LedgerBook[]
  } catch (e) {
    return []
  }
}

function writeAll(list: LedgerBook[]) {
  uni.setStorageSync(STORAGE_KEY, list)
}

function newId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`
}

function defaultSelfMember(): LedgerMember {
  const raw = (uni.getStorageSync('user') as any) || {}
  const nickname = String(raw?.nickname || '').trim() || '我'
  const avatarUrl = String(raw?.avatarUrl || '').trim()
  return {
    id: newId('member'),
    nickname,
    avatarUrl,
    createdAt: nowIso(),
  }
}

function sortLedgers(items: LedgerBook[]): LedgerBook[] {
  return items.slice().sort((a, b) => String(b.updatedAt).localeCompare(String(a.updatedAt)))
}

export function listLedgers(): LedgerBook[] {
  return sortLedgers(readAll())
}

export function getLedger(id: string): LedgerBook | null {
  const items = readAll()
  return items.find((it) => it.id === id) || null
}

export function upsertLedger(ledger: LedgerBook): LedgerBook {
  const items = readAll()
  const idx = items.findIndex((it) => it.id === ledger.id)
  if (idx >= 0) items[idx] = ledger
  else items.unshift(ledger)
  writeAll(items)
  return ledger
}

export function createLedger(name: string): LedgerBook {
  const now = nowIso()
  const selfMember = defaultSelfMember()
  const ledger: LedgerBook = {
    id: newId('ledger'),
    name: String(name || '').trim() || '记账簿',
    status: 'recording',
    createdAt: now,
    updatedAt: now,
    members: [selfMember],
    records: [],
  }
  const items = readAll()
  items.unshift(ledger)
  writeAll(items)
  return ledger
}

export function endLedger(id: string): LedgerBook | null {
  const ledger = getLedger(id)
  if (!ledger) return null
  if (ledger.status === 'ended') return ledger
  const now = nowIso()
  ledger.status = 'ended'
  ledger.endedAt = now
  ledger.updatedAt = now
  return upsertLedger(ledger)
}

export function addLedgerMember(id: string, payload: { nickname: string; avatarUrl?: string }): LedgerBook | null {
  const ledger = getLedger(id)
  if (!ledger || ledger.status === 'ended') return ledger
  const nickname = String(payload.nickname || '').trim()
  if (!nickname) return ledger
  const now = nowIso()
  const member: LedgerMember = {
    id: newId('member'),
    nickname,
    avatarUrl: payload.avatarUrl ? String(payload.avatarUrl).trim() : '',
    createdAt: now,
  }
  ledger.members = [...ledger.members, member]
  ledger.updatedAt = now
  return upsertLedger(ledger)
}

export function addLedgerRecord(
  id: string,
  payload: { memberId: string; type: 'income' | 'expense'; amount: number; note?: string },
): LedgerBook | null {
  const ledger = getLedger(id)
  if (!ledger || ledger.status === 'ended') return ledger
  const amount = Number(payload.amount)
  if (!payload.memberId || !Number.isFinite(amount) || amount <= 0) return ledger
  const now = nowIso()
  const record: LedgerRecord = {
    id: newId('record'),
    memberId: payload.memberId,
    type: payload.type === 'income' ? 'income' : 'expense',
    amount,
    note: payload.note ? String(payload.note).trim() : '',
    createdAt: now,
  }
  ledger.records = [record, ...ledger.records]
  ledger.updatedAt = now
  return upsertLedger(ledger)
}
