type ApiError = { code: string; message: string }

function normalizeBase(v: string): string {
  const s = String(v || '').trim()
  return s.replace(/\/+$/, '')
}

// `dev` uses local API via env; `build` falls back to prod.
const API_BASE = normalizeBase(import.meta.env.VITE_SCOREHUB_API_BASE || 'https://wxapi.wcoder.com/api/v1')
const WS_BASE = normalizeBase(import.meta.env.VITE_SCOREHUB_WS_BASE || 'wss://wxapi.wcoder.com')
const REQUEST_TIMEOUT_MS = 10_000

function getToken(): string {
  return (uni.getStorageSync('token') as string) || ''
}

async function request<T>(method: UniApp.RequestOptions['method'], path: string, data?: any): Promise<T> {
  const token = getToken()
  const res = await new Promise<UniApp.RequestSuccessCallbackResult>((resolve, reject) => {
    uni.request({
      url: `${API_BASE}${path}`,
      method,
      data,
      timeout: REQUEST_TIMEOUT_MS,
      header: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      success: resolve,
      fail: reject,
    })
  })

  const body = res.data as any
  if (body?.error) throw body.error as ApiError
  return body as T
}

export async function devLogin(openid: string, nickname: string, avatarUrl: string) {
  const body = await request<{ token: string; user: any }>('POST', '/auth/dev_login', { openid, nickname, avatarUrl })
  uni.setStorageSync('token', body.token)
  uni.setStorageSync('user', body.user)
  return body
}

export async function wechatLogin(code: string) {
  const body = await request<{ token: string; user: any }>('POST', '/auth/wechat_login', { code })
  uni.setStorageSync('token', body.token)
  uni.setStorageSync('user', body.user)
  return body
}

export async function getMe() {
  return request<{ user: any }>('GET', '/me')
}

export async function updateMe(payload: { nickname?: string; avatarUrl?: string }) {
  const body = await request<{ user: any }>('PATCH', '/me', payload)
  if (body?.user) uni.setStorageSync('user', body.user)
  return body
}

export async function createScorebook(payload: { name?: string; locationText?: string; bookType?: string }) {
  return request<{ scorebook: any; me: any }>('POST', '/scorebooks', payload)
}

export async function listMyScorebooks() {
  return request<{ items: any[] }>('GET', '/scorebooks')
}

export async function getScorebookDetail(id: string) {
  return request<{ scorebook: any; me: { memberId: string; isOwner: boolean }; members: any[] }>('GET', `/scorebooks/${id}`)
}

export async function updateScorebookName(id: string, name: string) {
  return request<{ scorebook: any }>('PATCH', `/scorebooks/${id}`, { name })
}

export async function endScorebook(id: string) {
  return request<{ scorebook: any; winners?: any }>('POST', `/scorebooks/${id}/end`)
}

export async function deleteScorebook(id: string) {
  return request<{ scorebook: any }>('DELETE', `/scorebooks/${id}`)
}

export async function joinScorebook(id: string, payload: { nickname?: string; avatarUrl?: string }) {
  return request<{ member: any }>('POST', `/scorebooks/${id}/join`, payload)
}

export async function updateMyProfile(id: string, payload: { nickname: string; avatarUrl?: string }) {
  return request<{ member: any }>('PATCH', `/scorebooks/${id}/members/me`, payload)
}

export async function createRecord(id: string, payload: { toMemberId: string; delta: number; note?: string }) {
  return request<{ record: any }>('POST', `/scorebooks/${id}/records`, payload)
}

export async function listScorebookRecords(id: string, limit = 50, offset = 0) {
  const q = `?limit=${encodeURIComponent(String(limit))}&offset=${encodeURIComponent(String(offset))}`
  return request<{ items: any[]; limit: number; offset: number }>('GET', `/scorebooks/${id}/records${q}`)
}

export async function getInviteInfo(code: string) {
  return request<{ invite: any }>('GET', `/invites/${code}`)
}

export async function joinByInviteCode(code: string, payload: { nickname?: string; avatarUrl?: string }) {
  return request<{ scorebookId: string; member: any }>('POST', `/invites/${code}/join`, payload)
}

export async function getInviteQRCode(scorebookId: string) {
  const token = getToken()
  const res = await new Promise<UniApp.RequestSuccessCallbackResult>((resolve, reject) => {
    uni.request({
      url: `${API_BASE}/scorebooks/${encodeURIComponent(scorebookId)}/invite_qrcode`,
      method: 'GET',
      timeout: REQUEST_TIMEOUT_MS,
      header: token ? { Authorization: `Bearer ${token}` } : {},
      responseType: 'arraybuffer',
      success: resolve,
      fail: reject,
    })
  })

  if (res.statusCode !== 200) {
    try {
      const buf = new Uint8Array(res.data as ArrayBuffer)
      const text = String.fromCharCode(...buf)
      const body = JSON.parse(text) as any
      throw body?.error || { message: '获取二维码失败' }
    } catch (e: any) {
      throw e?.message ? e : { message: '获取二维码失败' }
    }
  }

  const base64 = uni.arrayBufferToBase64(res.data as ArrayBuffer)
  return `data:image/png;base64,${base64}`
}

export async function getLedgerInviteQRCode(ledgerId: string) {
  const token = getToken()
  const res = await new Promise<UniApp.RequestSuccessCallbackResult>((resolve, reject) => {
    uni.request({
      url: `${API_BASE}/ledgers/${encodeURIComponent(ledgerId)}/invite_qrcode`,
      method: 'GET',
      timeout: REQUEST_TIMEOUT_MS,
      header: token ? { Authorization: `Bearer ${token}` } : {},
      responseType: 'arraybuffer',
      success: resolve,
      fail: reject,
    })
  })

  if (res.statusCode !== 200) {
    try {
      const buf = new Uint8Array(res.data as ArrayBuffer)
      const text = String.fromCharCode(...buf)
      const body = JSON.parse(text) as any
      throw body?.error || { message: '获取二维码失败' }
    } catch (e: any) {
      throw e?.message ? e : { message: '获取二维码失败' }
    }
  }

  const base64 = uni.arrayBufferToBase64(res.data as ArrayBuffer)
  return `data:image/png;base64,${base64}`
}

export async function reverseGeocode(lat: number, lng: number) {
  const q = `?lat=${encodeURIComponent(String(lat))}&lng=${encodeURIComponent(String(lng))}`
  return request<{ locationText: string; source?: string }>('GET', `/location/reverse_geocode${q}`)
}

export async function createLedger(payload: { name?: string }) {
  return request<{ ledger: any; member?: any }>('POST', '/ledgers', payload)
}

export async function listLedgers() {
  return request<{ items: any[] }>('GET', '/ledgers')
}

export async function getLedgerDetail(id: string) {
  return request<{ ledger: any; members: any[]; records: any[] }>('GET', `/ledgers/${id}`)
}

export async function updateLedgerName(id: string, name: string) {
  return request<{ ledger: any }>('PATCH', `/ledgers/${id}`, { name })
}

export async function updateLedger(id: string, payload: { name?: string; shareDisabled?: boolean }) {
  return request<{ ledger: any }>('PATCH', `/ledgers/${id}`, payload)
}

export async function addLedgerMember(id: string, payload: { nickname: string; avatarUrl?: string; remark?: string }) {
  return request<{ member: any }>('POST', `/ledgers/${id}/members`, payload)
}

export async function bindLedgerMember(
  id: string,
  payload: { memberId: string; nickname?: string; avatarUrl?: string },
) {
  return request<{ member: any }>('POST', `/ledgers/${id}/bind`, payload)
}

export async function updateLedgerMember(
  id: string,
  memberId: string,
  payload: { nickname: string; avatarUrl?: string; remark?: string },
) {
  return request<{ member: any }>('PATCH', `/ledgers/${id}/members/${memberId}`, payload)
}

export async function addLedgerRecord(id: string, payload: { memberId: string; type: 'income' | 'expense'; amount: number; note?: string }) {
  return request<{ record: any }>('POST', `/ledgers/${id}/records`, payload)
}

export async function endLedger(id: string) {
  return request<{ ledger: any }>('POST', `/ledgers/${id}/end`)
}

export async function listBirthdays() {
  return request<{ items: any[] }>('GET', '/birthdays')
}

export async function getBirthday(id: string) {
  return request<{ birthday: any }>('GET', `/birthdays/${id}`)
}

export async function createBirthday(payload: {
  name: string
  gender?: string
  phone?: string
  relation?: string
  note?: string
  avatarUrl?: string
  solarBirthday?: string
  lunarBirthday?: string
  primaryType?: 'solar' | 'lunar'
  primaryMonth?: number
  primaryDay?: number
  primaryYear?: number
}) {
  return request<{ birthday: any }>('POST', '/birthdays', payload)
}

export async function updateBirthday(
  id: string,
  payload: {
    name?: string
    gender?: string
    phone?: string
    relation?: string
    note?: string
    avatarUrl?: string
    solarBirthday?: string
    lunarBirthday?: string
    primaryType?: 'solar' | 'lunar'
    primaryMonth?: number
    primaryDay?: number
    primaryYear?: number
  },
) {
  return request<{ birthday: any }>('PATCH', `/birthdays/${id}`, payload)
}

export async function deleteBirthday(id: string) {
  return request<{ ok: boolean }>('DELETE', `/birthdays/${id}`)
}

export async function listDepositAccounts() {
  return request<{ items: any[] }>('GET', '/deposits/accounts')
}

export async function getDepositAccount(id: string) {
  return request<{ account: any }>('GET', `/deposits/accounts/${id}`)
}

export async function createDepositAccount(payload: {
  bank: string
  branch?: string
  accountNo?: string
  holder?: string
  avatarUrl?: string
  note?: string
}) {
  return request<{ account: any }>('POST', '/deposits/accounts', payload)
}

export async function updateDepositAccount(
  id: string,
  payload: {
    bank?: string
    branch?: string
    accountNo?: string
    holder?: string
    avatarUrl?: string
    note?: string
  },
) {
  return request<{ account: any }>('PATCH', `/deposits/accounts/${id}`, payload)
}

export async function deleteDepositAccount(id: string) {
  return request<{ ok: boolean }>('DELETE', `/deposits/accounts/${id}`)
}

export async function listDepositRecords(params: { accountId?: string; status?: string; tags?: string[] } = {}) {
  const query: string[] = []
  if (params.accountId) query.push(`accountId=${encodeURIComponent(params.accountId)}`)
  if (params.status) query.push(`status=${encodeURIComponent(params.status)}`)
  if (params.tags && params.tags.length > 0) query.push(`tags=${encodeURIComponent(params.tags.join(','))}`)
  const q = query.join('&')
  return request<{ items: any[] }>('GET', `/deposits/records${q ? `?${q}` : ''}`)
}

export async function listDepositAccountRecords(accountId: string, params: { status?: string; tags?: string[] } = {}) {
  const query: string[] = []
  if (params.status) query.push(`status=${encodeURIComponent(params.status)}`)
  if (params.tags && params.tags.length > 0) query.push(`tags=${encodeURIComponent(params.tags.join(','))}`)
  const q = query.join('&')
  return request<{ items: any[] }>('GET', `/deposits/accounts/${accountId}/records${q ? `?${q}` : ''}`)
}

export async function getDepositRecord(id: string) {
  return request<{ record: any }>('GET', `/deposits/records/${id}`)
}

export async function createDepositRecord(
  accountId: string,
  payload: {
    currency: string
    amount: number
    amountUpper?: string
    termValue: number
    termUnit: 'year' | 'month'
    rate: number
    startDate: string
    endDate: string
    interest: number
    receiptNo?: string
    status?: string
    withdrawnAt?: string
    tags?: string[]
    attachments?: any[]
    note?: string
  },
) {
  return request<{ record: any }>('POST', `/deposits/accounts/${accountId}/records`, payload)
}

export async function updateDepositRecord(
  id: string,
  payload: {
    currency?: string
    amount?: number
    amountUpper?: string
    termValue?: number
    termUnit?: 'year' | 'month'
    rate?: number
    startDate?: string
    endDate?: string
    interest?: number
    receiptNo?: string
    status?: string
    withdrawnAt?: string
    tags?: string[]
    attachments?: any[]
    note?: string
  },
) {
  return request<{ record: any }>('PATCH', `/deposits/records/${id}`, payload)
}

export async function deleteDepositRecord(id: string) {
  return request<{ ok: boolean }>('DELETE', `/deposits/records/${id}`)
}

export async function listDepositTags(params: { accountId?: string; status?: string } = {}) {
  const query: string[] = []
  if (params.accountId) query.push(`accountId=${encodeURIComponent(params.accountId)}`)
  if (params.status) query.push(`status=${encodeURIComponent(params.status)}`)
  const q = query.join('&')
  return request<{ items: { tag: string; count: number }[] }>('GET', `/deposits/tags${q ? `?${q}` : ''}`)
}

export async function getDepositStats(params: { accountId?: string; status?: string; tags?: string[] } = {}) {
  const query: string[] = []
  if (params.accountId) query.push(`accountId=${encodeURIComponent(params.accountId)}`)
  if (params.status) query.push(`status=${encodeURIComponent(params.status)}`)
  if (params.tags && params.tags.length > 0) query.push(`tags=${encodeURIComponent(params.tags.join(','))}`)
  const q = query.join('&')
  return request<{ stats: any }>('GET', `/deposits/stats${q ? `?${q}` : ''}`)
}

export async function deleteLedger(id: string) {
  return request<{ ledger: any }>('DELETE', `/ledgers/${id}`)
}

export async function connectScorebookWS(scorebookId: string, onEvent: (evt: any) => void) {
  const token = getToken()
  const url = `${WS_BASE}/ws/scorebooks/${encodeURIComponent(scorebookId)}?token=${encodeURIComponent(token)}`
  // In some uni-app runtimes, WebSocket APIs switch to Promise style when no callback is provided.
  // Adding a no-op `complete` callback + awaiting makes it compatible with both styles.
  const task = await uni.connectSocket({ url, complete: () => {} } as any)

  task.onMessage((msg) => {
    try {
      const evt = JSON.parse(msg.data as string)
      onEvent(evt)
    } catch (e) {}
  })

  return task
}
