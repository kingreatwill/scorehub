type ApiError = { code: string; message: string }

const API_BASE = 'https://wxapi.wcoder.com/api/v1'
const WS_BASE = 'wss://wxapi.wcoder.com'

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

export async function createScorebook(payload: { name?: string; locationText?: string }) {
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
