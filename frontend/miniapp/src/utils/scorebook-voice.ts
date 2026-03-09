import { getScoreReceivedSpeechAudio } from './api'

export type ScorebookVoiceOption = {
  id: string
  label: string
  language?: string
  description?: string
  isDefault?: boolean
}

export type ScorebookVoiceSettings = {
  enabled: boolean
  voice: string
  template: string
}

const isMpWeixinRuntime = (() => {
  let flag = false
  // #ifdef MP-WEIXIN
  flag = true
  // #endif
  return flag
})()

const isH5Runtime = (() => {
  let flag = false
  // #ifdef H5
  flag = true
  // #endif
  return flag
})()

type H5VoiceProfile = {
  h5Candidates: string[]
  h5Rate: number
  h5Pitch: number
  volume: number
}

export type ScorebookVoiceTemplatePreset = {
  key: 'received' | 'sender'
  label: string
  template: string
}

type RecordLike = {
  id: string
  fromMemberId: string
  toMemberId: string
  delta: number
  createdAtMs: number
}

type SyncOptions = {
  scorebookId: string
  meMemberId: string
  records: any[]
  pageSize: number
  fetchPage?: (offset: number) => Promise<any[]>
  resolveFromMemberName?: (memberId: string) => string
}

type RealtimeOptions = {
  scorebookId: string
  meMemberId: string
  record: any
  resolveFromMemberName?: (memberId: string) => string
}

type VoiceJob = {
  dedupeKey: string
  actorName: string
  delta: number
  createdAtMs: number
}

type AudioAsset = {
  src: string
  dispose?: () => void
  createdAtMs: number
  lastUsedAtMs: number
}

const SCOREBOOK_VOICE_SETTINGS_KEY = 'scorehub.scorebook.voice.settings.v2'
const SCOREBOOK_VOICE_CATALOG_KEY = 'scorehub.scorebook.voice.catalog.v1'
const DEFAULT_SCOREBOOK_VOICE_TEMPLATE = '收到{N}分'
const MAP_TTL_MS = 24 * 60 * 60 * 1000
const MAP_MAX_SIZE = 2400
const CATCHUP_MAX_PAGES = 5
const AUDIO_CACHE_TTL_MS = 6 * 60 * 60 * 1000
const AUDIO_CACHE_MAX_SIZE = 180

const backendAudioCache = new Map<string, AudioAsset>()

const VOICE_TEMPLATE_PRESETS: ScorebookVoiceTemplatePreset[] = [
  { key: 'received', label: '收到{N}分', template: '收到{N}分' },
  { key: 'sender', label: '{X} 给你记了 {N} 分', template: '{X} 给你记了 {N} 分' },
]

function normalizeVoiceOptions(list: any[]): ScorebookVoiceOption[] {
  const out: ScorebookVoiceOption[] = []
  const seen = new Set<string>()
  for (const item of list || []) {
    const id = String(item?.id || item?.voice || item?.name || '').trim()
    if (!id || seen.has(id)) continue
    seen.add(id)
    const label = String(item?.label || item?.displayName || item?.name || id).trim() || id
    out.push({
      id,
      label,
      language: String(item?.language || '').trim(),
      description: String(item?.description || '').trim(),
      isDefault: !!item?.isDefault,
    })
  }
  if (!out.length) return []
  if (!out.some((item) => item.isDefault)) out[0].isDefault = true
  return out
}

function defaultVoiceID(catalog: ScorebookVoiceOption[]): string {
  return catalog.find((item) => item.isDefault)?.id || catalog[0]?.id || ''
}

function normalizeVoiceID(raw: any, catalog: ScorebookVoiceOption[] = loadScorebookVoiceCatalog()): string {
  const id = String(raw || '').trim()
  if (id && catalog.some((item) => item.id === id)) return id
  return defaultVoiceID(catalog)
}

export function saveScorebookVoiceCatalog(items: any[]): ScorebookVoiceOption[] {
  const normalized = normalizeVoiceOptions(items)
  uni.setStorageSync(SCOREBOOK_VOICE_CATALOG_KEY, normalized)
  const current = loadScorebookVoiceSettings()
  if (current.voice !== normalizeVoiceID(current.voice, normalized)) {
    saveScorebookVoiceSettings({ voice: normalizeVoiceID(current.voice, normalized) })
  }
  return normalized
}

export function loadScorebookVoiceCatalog(): ScorebookVoiceOption[] {
  const raw = (uni.getStorageSync(SCOREBOOK_VOICE_CATALOG_KEY) as any) || []
  return normalizeVoiceOptions(raw)
}

export function normalizeScorebookVoiceTemplate(raw: any): string {
  const text = String(raw ?? '')
    .replace(/\s+/g, ' ')
    .trim()
  if (!text) return DEFAULT_SCOREBOOK_VOICE_TEMPLATE
  const clipped = Array.from(text)
    .slice(0, 48)
    .join('')
  if (!clipped.includes('{N}')) return DEFAULT_SCOREBOOK_VOICE_TEMPLATE
  return clipped
}

function parseBoolean(raw: any, fallback: boolean): boolean {
  if (typeof raw === 'boolean') return raw
  if (raw === 1 || raw === '1' || raw === 'true') return true
  if (raw === 0 || raw === '0' || raw === 'false') return false
  return fallback
}

function nowMs(): number {
  return Date.now()
}

function normalizeDelta(raw: any): number {
  const value = Number(raw || 0)
  if (!Number.isFinite(value) || value <= 0) return 0
  return Math.round(value * 100) / 100
}

function parseTimeMs(raw: any): number {
  const ms = Date.parse(String(raw || ''))
  return Number.isFinite(ms) ? ms : 0
}

function normalizeRecordLike(record: any): RecordLike | null {
  const id = String(record?.id || '').trim()
  if (!id) return null
  return {
    id,
    fromMemberId: String(record?.fromMemberId || '').trim(),
    toMemberId: String(record?.toMemberId || '').trim(),
    delta: normalizeDelta(record?.delta),
    createdAtMs: parseTimeMs(record?.createdAt),
  }
}

function scopedRecordKey(scorebookId: string, recordId: string): string {
  return `${String(scorebookId || '').trim()}:${String(recordId || '').trim()}`
}

function pruneMap(map: Map<string, number>) {
  const now = nowMs()
  for (const [key, ts] of map.entries()) {
    if (now - ts > MAP_TTL_MS) map.delete(key)
  }
  if (map.size <= MAP_MAX_SIZE) return
  const entries = Array.from(map.entries()).sort((a, b) => a[1] - b[1])
  for (const [key] of entries) {
    map.delete(key)
    if (map.size <= MAP_MAX_SIZE) break
  }
}

function cleanupBackendAudioCache() {
  const now = nowMs()
  for (const [key, item] of backendAudioCache.entries()) {
    if (now - item.lastUsedAtMs <= AUDIO_CACHE_TTL_MS) continue
    try {
      item.dispose?.()
    } catch (e) {}
    backendAudioCache.delete(key)
  }
  if (backendAudioCache.size <= AUDIO_CACHE_MAX_SIZE) return
  const entries = Array.from(backendAudioCache.entries()).sort((a, b) => a[1].lastUsedAtMs - b[1].lastUsedAtMs)
  for (const [key, item] of entries) {
    try {
      item.dispose?.()
    } catch (e) {}
    backendAudioCache.delete(key)
    if (backendAudioCache.size <= AUDIO_CACHE_MAX_SIZE) break
  }
}

function normalizeSpeechText(raw: any): string {
  return String(raw ?? '')
    .replace(/\s+/g, ' ')
    .trim()
}

function audioCacheKey(text: string, voice: string): string {
  const resolvedVoice = String(voice || '').trim() || '__default__'
  return `${resolvedVoice}:${normalizeSpeechText(text)}`
}

function contentTypeToExtension(contentType: string): string {
  const normalized = String(contentType || '').toLowerCase()
  if (normalized.includes('wav')) return 'wav'
  if (normalized.includes('mpeg') || normalized.includes('mp3')) return 'mp3'
  return 'mp3'
}

function hashString(input: string): string {
  let hash = 2166136261
  for (let i = 0; i < input.length; i += 1) {
    hash ^= input.charCodeAt(i)
    hash = Math.imul(hash, 16777619)
  }
  return (hash >>> 0).toString(16)
}

function createH5AudioAsset(data: ArrayBuffer, contentType: string): AudioAsset {
  const blob = new Blob([data], { type: contentType || 'audio/mpeg' })
  const src = URL.createObjectURL(blob)
  return {
    src,
    createdAtMs: nowMs(),
    lastUsedAtMs: nowMs(),
    dispose: () => {
      try {
        URL.revokeObjectURL(src)
      } catch (e) {}
    },
  }
}

function fileExists(fs: any, filePath: string): Promise<boolean> {
  return new Promise((resolve) => {
    if (!fs?.access) {
      resolve(false)
      return
    }
    fs.access({
      path: filePath,
      success: () => resolve(true),
      fail: () => resolve(false),
    })
  })
}

async function createMpAudioAsset(data: ArrayBuffer, contentType: string, cacheKey: string): Promise<AudioAsset> {
  const fs = (uni as any).getFileSystemManager?.()
  const root =
    String((globalThis as any)?.wx?.env?.USER_DATA_PATH || '') ||
    String((globalThis as any)?.my?.env?.USER_DATA_PATH || '')
  if (!fs?.writeFile || !root) throw new Error('local fs unavailable')

  const ext = contentTypeToExtension(contentType)
  const filePath = `${root}/scorehub_tts_${hashString(cacheKey)}.${ext}`
  if (!(await fileExists(fs, filePath))) {
    await new Promise<void>((resolve, reject) => {
      fs.writeFile({
        filePath,
        data,
        success: () => resolve(),
        fail: reject,
      })
    })
  }

  return {
    src: filePath,
    createdAtMs: nowMs(),
    lastUsedAtMs: nowMs(),
    dispose: () => {
      try {
        fs.unlink?.({ filePath })
      } catch (e) {}
    },
  }
}

async function getBackendAudioAsset(text: string, voice: string): Promise<AudioAsset> {
  const normalizedText = normalizeSpeechText(text)
  if (!normalizedText) throw new Error('invalid text')
  const cacheKey = audioCacheKey(normalizedText, voice)
  const cached = backendAudioCache.get(cacheKey)
  if (cached) {
    cached.lastUsedAtMs = nowMs()
    return cached
  }

  const res = await getScoreReceivedSpeechAudio(normalizedText, voice)
  let asset: AudioAsset
  if (isMpWeixinRuntime) {
    asset = await createMpAudioAsset(res.data, res.contentType, cacheKey)
  } else if (isH5Runtime && typeof Blob !== 'undefined' && typeof URL !== 'undefined' && typeof URL.createObjectURL === 'function') {
    asset = createH5AudioAsset(res.data, res.contentType)
  } else {
    const base64 = uni.arrayBufferToBase64(res.data)
    asset = {
      src: `data:${res.contentType || 'audio/mpeg'};base64,${base64}`,
      createdAtMs: nowMs(),
      lastUsedAtMs: nowMs(),
    }
  }

  cleanupBackendAudioCache()
  backendAudioCache.set(cacheKey, asset)
  return asset
}

function isRecordForCurrentMember(record: RecordLike | null, meMemberId: string): boolean {
  if (!record) return false
  if (!meMemberId) return false
  if (record.toMemberId !== meMemberId) return false
  return Number.isFinite(record.delta) && record.delta > 0
}

function speechSynthesisRef(): SpeechSynthesis | null {
  const root = globalThis as any
  const synth = root?.speechSynthesis
  return synth && typeof synth.speak === 'function' ? synth : null
}

function resolveH5FallbackProfile(option?: ScorebookVoiceOption | null): H5VoiceProfile {
  const raw = `${String(option?.id || '')} ${String(option?.label || '')} ${String(option?.description || '')}`.toLowerCase()
  if (raw.includes('男') || raw.includes('稳') || raw.includes('steady') || raw.includes('deep')) {
    return {
      h5Candidates: ['yunxi', 'yunyang', 'male', 'zh-cn'],
      h5Rate: 0.95,
      h5Pitch: 0.88,
      volume: 1,
    }
  }
  if (raw.includes('亮') || raw.includes('bright') || raw.includes('快')) {
    return {
      h5Candidates: ['xiaomo', 'hanhan', 'tingting', 'zh-cn'],
      h5Rate: 1.08,
      h5Pitch: 1.2,
      volume: 1,
    }
  }
  return {
    h5Candidates: ['xiaoxiao', 'xiaoyi', 'tingting', 'female', 'zh-cn'],
    h5Rate: 1,
    h5Pitch: 1.08,
    volume: 1,
  }
}

function pickH5Voice(profile: H5VoiceProfile): SpeechSynthesisVoice | null {
  const synth = speechSynthesisRef()
  if (!synth || typeof synth.getVoices !== 'function') return null
  const voices = synth.getVoices() || []
  if (!voices.length) return null
  const lowerCandidates = profile.h5Candidates.map((item) => item.toLowerCase())
  const normalized = voices.map((voice) => ({
    raw: voice,
    name: String(voice?.name || '').toLowerCase(),
    lang: String(voice?.lang || '').toLowerCase(),
  }))
  for (const candidate of lowerCandidates) {
    const hit = normalized.find((item) => item.name.includes(candidate))
    if (hit) return hit.raw
  }
  const zh = normalized.find((item) => item.lang.includes('zh') || item.name.includes('chinese'))
  return zh?.raw || normalized[0]?.raw || null
}

async function speakWithH5(text: string, option: ScorebookVoiceOption | null, registerStop: (stopper: (() => void) | null) => void): Promise<void> {
  const synth = speechSynthesisRef()
  const UtteranceCtor = (globalThis as any)?.SpeechSynthesisUtterance
  if (!synth || typeof UtteranceCtor !== 'function') return

  const profile = resolveH5FallbackProfile(option)
  await new Promise<void>((resolve) => {
    const utterance = new UtteranceCtor(text)
    utterance.lang = String(option?.language || 'zh-CN') || 'zh-CN'
    utterance.rate = profile.h5Rate
    utterance.pitch = profile.h5Pitch
    utterance.volume = profile.volume
    const matchedVoice = pickH5Voice(profile)
    if (matchedVoice) utterance.voice = matchedVoice

    let finished = false
    const done = () => {
      if (finished) return
      finished = true
      registerStop(null)
      resolve()
    }

    utterance.onend = done
    utterance.onerror = done
    registerStop(() => {
      try {
        synth.cancel()
      } catch (e) {}
      done()
    })
    synth.speak(utterance)
  })
}

async function playAudioSource(src: string, registerStop: (stopper: (() => void) | null) => void): Promise<void> {
  if (!src) throw new Error('empty audio src')

  await new Promise<void>((resolve, reject) => {
    const audio = uni.createInnerAudioContext()
    let settled = false

    const finish = (err?: any) => {
      if (settled) return
      settled = true
      registerStop(null)
      try {
        audio.destroy()
      } catch (e) {}
      if (err) reject(err)
      else resolve()
    }

    audio.onEnded(() => finish())
    audio.onStop(() => finish())
    audio.onError((err: any) => finish(err || new Error('audio play failed')))

    registerStop(() => {
      try {
        audio.stop()
      } catch (e) {}
      finish()
    })

    try {
      ;(audio as any).obeyMuteSwitch = false
    } catch (e) {}

    audio.src = src
    try {
      audio.play()
    } catch (e) {
      finish(e)
    }
  })
}

async function speakScoreText(text: string, voice: string, registerStop: (stopper: (() => void) | null) => void): Promise<void> {
  const normalizedText = normalizeSpeechText(text)
  if (!normalizedText) return
  const catalog = loadScorebookVoiceCatalog()
  const option = catalog.find((item) => item.id === voice) || catalog.find((item) => item.isDefault) || catalog[0] || null

  let backendErr: any = null
  try {
    const asset = await getBackendAudioAsset(normalizedText, voice)
    await playAudioSource(asset.src, registerStop)
    return
  } catch (e: any) {
    backendErr = e
  }

  if (isH5Runtime) {
    await speakWithH5(normalizedText, option, registerStop)
    return
  }

  throw backendErr || new Error('voice playback failed')
}

export function listScorebookVoiceProfiles(catalog?: ScorebookVoiceOption[]) {
  return normalizeVoiceOptions(catalog || loadScorebookVoiceCatalog())
}

export function listScorebookVoiceTemplatePresets(): ScorebookVoiceTemplatePreset[] {
  return VOICE_TEMPLATE_PRESETS.map((item) => ({ ...item }))
}

export function resolveScorebookVoiceLabel(id?: string, catalog?: ScorebookVoiceOption[]): string {
  const items = normalizeVoiceOptions(catalog || loadScorebookVoiceCatalog())
  const resolved = items.find((item) => item.id === String(id || '').trim())
  return resolved?.label || items.find((item) => item.isDefault)?.label || items[0]?.label || '默认'
}

export function loadScorebookVoiceSettings(): ScorebookVoiceSettings {
  const raw = (uni.getStorageSync(SCOREBOOK_VOICE_SETTINGS_KEY) as any) || {}
  const catalog = loadScorebookVoiceCatalog()
  return {
    enabled: parseBoolean(raw?.enabled, true),
    voice: normalizeVoiceID(raw?.voice ?? raw?.voiceKey, catalog),
    template: normalizeScorebookVoiceTemplate(raw?.template),
  }
}

export function saveScorebookVoiceSettings(
  patch: Partial<ScorebookVoiceSettings> & { voiceKey?: string },
): ScorebookVoiceSettings {
  const current = loadScorebookVoiceSettings()
  const catalog = loadScorebookVoiceCatalog()
  const next: ScorebookVoiceSettings = {
    enabled: parseBoolean(patch.enabled, current.enabled),
    voice: normalizeVoiceID(patch.voice ?? patch.voiceKey ?? current.voice, catalog),
    template: normalizeScorebookVoiceTemplate(patch.template ?? current.template),
  }
  uni.setStorageSync(SCOREBOOK_VOICE_SETTINGS_KEY, next)
  return next
}

export function scorebookVoicePlatformHint(): string {
  if (isMpWeixinRuntime) return '后端语音播报'
  if (isH5Runtime) return '后端语音，失败时浏览器播报'
  return '当前平台暂不支持'
}

export function buildReceivedScoreSpeech(delta: any): string {
  return renderScorebookVoiceText(DEFAULT_SCOREBOOK_VOICE_TEMPLATE, { delta })
}

function formatSpeechDelta(delta: any): string {
  const n = normalizeDelta(delta)
  if (!n) return ''
  return n.toFixed(2).replace(/\.00$/, '').replace(/(\.\d)0$/, '$1')
}

function normalizeActorName(raw: any): string {
  const text = normalizeSpeechText(raw)
  if (!text) return '有人'
  return Array.from(text)
    .slice(0, 12)
    .join('')
}

export function renderScorebookVoiceText(template: any, payload: { delta: any; fromName?: any }): string {
  const resolvedTemplate = normalizeScorebookVoiceTemplate(template)
  const deltaText = formatSpeechDelta(payload?.delta)
  if (!deltaText) return '收到分数'
  const actorName = normalizeActorName(payload?.fromName)
  const rendered = resolvedTemplate.replace(/\{X\}/g, actorName).replace(/\{N\}/g, deltaText)
  return normalizeSpeechText(rendered) || `收到${deltaText}分`
}

export async function previewScorebookVoice(
  deltaOrOptions:
    | number
    | {
        delta?: number
        fromName?: string
        override?: Partial<ScorebookVoiceSettings> & { voiceKey?: string }
      } = 8,
  override?: Partial<ScorebookVoiceSettings> & { voiceKey?: string },
) {
  let delta = 8
  let fromName = '小王'
  let nextOverride = override
  if (typeof deltaOrOptions === 'object' && deltaOrOptions) {
    delta = normalizeDelta(deltaOrOptions.delta || 8) || 8
    fromName = normalizeActorName(deltaOrOptions.fromName || '小王')
    nextOverride = deltaOrOptions.override
  } else {
    delta = normalizeDelta(deltaOrOptions) || 8
  }

  const merged: ScorebookVoiceSettings = {
    ...loadScorebookVoiceSettings(),
    ...nextOverride,
    voice: normalizeVoiceID(nextOverride?.voice ?? nextOverride?.voiceKey ?? loadScorebookVoiceSettings().voice),
    enabled: parseBoolean(nextOverride?.enabled, true),
    template: normalizeScorebookVoiceTemplate(nextOverride?.template ?? loadScorebookVoiceSettings().template),
  }
  if (!merged.enabled) return

  let currentStopper: (() => void) | null = null
  const text = renderScorebookVoiceText(merged.template, { delta, fromName })
  await speakScoreText(text, merged.voice, (stopper) => {
    currentStopper = stopper
  }).finally(() => {
    currentStopper = null
  })
}

export class ScorebookVoiceAnnouncer {
  private baselineReady = false
  private knownRecordIDs = new Map<string, number>()
  private announcedRecordIDs = new Map<string, number>()
  private queue: VoiceJob[] = []
  private playing = false
  private destroyed = false
  private playSession = 0
  private currentStopper: (() => void) | null = null
  private settings: ScorebookVoiceSettings = loadScorebookVoiceSettings()

  refreshSettings() {
    this.settings = loadScorebookVoiceSettings()
  }

  stop() {
    this.playSession += 1
    this.queue = []
    this.playing = false
    const stopper = this.currentStopper
    this.currentStopper = null
    try {
      stopper?.()
    } catch (e) {}
  }

  dispose() {
    this.destroyed = true
    this.stop()
    this.knownRecordIDs.clear()
    this.announcedRecordIDs.clear()
  }

  async syncRecords(options: SyncOptions) {
    const scorebookId = String(options.scorebookId || '').trim()
    const meMemberId = String(options.meMemberId || '').trim()
    const pageSize = Math.max(1, Number(options.pageSize || 20))
    if (!scorebookId) return

    const firstPage = this.normalizeRecords(options.records)
    if (!this.baselineReady) {
      for (const record of firstPage) this.markKnown(scorebookId, record.id)
      this.baselineReady = true
      return
    }

    const batches: RecordLike[][] = [firstPage]
    const firstPageAllUnknown = firstPage.length >= pageSize && firstPage.every((record) => !this.isKnown(scorebookId, record.id))

    if (options.fetchPage && firstPageAllUnknown) {
      let offset = pageSize
      for (let i = 1; i < CATCHUP_MAX_PAGES; i += 1) {
        const more = this.normalizeRecords(await options.fetchPage(offset))
        if (!more.length) break
        batches.push(more)
        offset += more.length
        if (more.length < pageSize || more.some((record) => this.isKnown(scorebookId, record.id))) break
      }
    }

    const unseenRelevant: RecordLike[] = []
    for (const batch of batches) {
      for (const record of batch) {
        const known = this.isKnown(scorebookId, record.id)
        this.markKnown(scorebookId, record.id)
        if (!known && isRecordForCurrentMember(record, meMemberId)) unseenRelevant.push(record)
      }
    }

    unseenRelevant.sort((a, b) => a.createdAtMs - b.createdAtMs)
    for (const record of unseenRelevant) {
      const actorName = options.resolveFromMemberName?.(record.fromMemberId) || ''
      this.enqueueRecord(scorebookId, record, actorName)
    }
  }

  handleRealtimeRecord(options: RealtimeOptions) {
    const scorebookId = String(options.scorebookId || '').trim()
    const meMemberId = String(options.meMemberId || '').trim()
    if (!scorebookId || !meMemberId) return

    const record = normalizeRecordLike(options.record)
    if (!record) return
    this.markKnown(scorebookId, record.id)
    if (!isRecordForCurrentMember(record, meMemberId)) return
    const actorName = options.resolveFromMemberName?.(record.fromMemberId) || ''
    this.enqueueRecord(scorebookId, record, actorName)
  }

  private normalizeRecords(list: any[]): RecordLike[] {
    const out: RecordLike[] = []
    for (const item of list || []) {
      const normalized = normalizeRecordLike(item)
      if (normalized) out.push(normalized)
    }
    return out
  }

  private isKnown(scorebookId: string, recordId: string): boolean {
    return this.knownRecordIDs.has(scopedRecordKey(scorebookId, recordId))
  }

  private markKnown(scorebookId: string, recordId: string) {
    this.knownRecordIDs.set(scopedRecordKey(scorebookId, recordId), nowMs())
    pruneMap(this.knownRecordIDs)
  }

  private enqueueRecord(scorebookId: string, record: RecordLike, actorName: string) {
    const dedupeKey = scopedRecordKey(scorebookId, record.id)
    if (this.announcedRecordIDs.has(dedupeKey)) return
    this.announcedRecordIDs.set(dedupeKey, nowMs())
    pruneMap(this.announcedRecordIDs)

    this.queue.push({
      dedupeKey,
      actorName: normalizeActorName(actorName),
      delta: record.delta,
      createdAtMs: record.createdAtMs,
    })
    this.queue.sort((a, b) => a.createdAtMs - b.createdAtMs)
    void this.playNext()
  }

  private async playNext() {
    if (this.playing || this.destroyed) return
    if (!this.queue.length) return

    this.playing = true
    const session = this.playSession
    try {
      while (!this.destroyed && session === this.playSession && this.queue.length) {
        const job = this.queue.shift()
        if (!job) break
        await this.playJob(job)
      }
    } finally {
      if (session === this.playSession) {
        this.playing = false
        this.currentStopper = null
      }
    }
  }

  private async playJob(job: VoiceJob) {
    this.refreshSettings()
    if (!this.settings.enabled) return

    const text = renderScorebookVoiceText(this.settings.template, { delta: job.delta, fromName: job.actorName })
    await speakScoreText(text, this.settings.voice, (stopper) => {
      this.currentStopper = stopper
    }).catch(() => Promise.resolve())
    this.currentStopper = null
  }
}
