<template>
  <view class="page" v-if="scorebook">
    <view class="card hero">
      <view class="row hero-row">
        <view class="name">{{ scorebook.name }}</view>
        <view class="badge" :class="{ ended: scorebook.status === 'ended' }">
          <view class="badge-dot" :class="{ ended: scorebook.status === 'ended' }" />
          <text>{{ scorebook.status === 'ended' ? '已结束' : '记录中' }}</text>
        </view>
      </view>
      <view class="sub hero-sub">
        <view class="pill" v-if="scorebook.locationText">{{ scorebook.locationText }}</view>
        <view class="pill" v-if="scorebook.startTime">{{ formatTime(scorebook.startTime) }}</view>
        <view class="pill">成员 {{ members.length }}</view>
        <view class="pill code" @click="copyInvite">
          <view class="qr-icon" @click.stop="openInviteCodeQR">
            <view class="qr-finder tl"><view class="qr-finder-inner" /></view>
            <view class="qr-finder tr"><view class="qr-finder-inner" /></view>
            <view class="qr-finder bl"><view class="qr-finder-inner" /></view>
            <view class="qr-dot d1" />
            <view class="qr-dot d2" />
            <view class="qr-dot d3" />
            <view class="qr-dot d4" />
            <view class="qr-dot d5" />
            <view class="qr-dot d6" />
            <view class="qr-dot d7" />
          </view>
          <text class="mono"> {{ scorebook.inviteCode }}</text> 
        </view>
      </view>
    </view>

    <view class="card" v-if="scorebook.status === 'ended' && (computedWinners.champion || computedWinners.runnerUp || computedWinners.third)">
      <view class="title">本局排名（得分大于0）</view>
      <view class="rank-row" v-if="computedWinners.champion">
        <text class="rank-label">冠军</text>
        <view class="rank-user">
          <image class="rank-avatar" :src="computedWinners.champion.avatarUrl || fallbackAvatar" mode="aspectFill" />
          <text class="rank-name">{{ displayNickname(computedWinners.champion.nickname) }}</text>
        </view>
        <text class="rank-score pos">{{ formatScore(computedWinners.champion.score) }}</text>
      </view>
      <view class="rank-row" v-if="computedWinners.runnerUp">
        <text class="rank-label">亚军</text>
        <view class="rank-user">
          <image class="rank-avatar" :src="computedWinners.runnerUp.avatarUrl || fallbackAvatar" mode="aspectFill" />
          <text class="rank-name">{{ displayNickname(computedWinners.runnerUp.nickname) }}</text>
        </view>
        <text class="rank-score pos">{{ formatScore(computedWinners.runnerUp.score) }}</text>
      </view>
      <view class="rank-row" v-if="computedWinners.third">
        <text class="rank-label">季军</text>
        <view class="rank-user">
          <image class="rank-avatar" :src="computedWinners.third.avatarUrl || fallbackAvatar" mode="aspectFill" />
          <text class="rank-name">{{ displayNickname(computedWinners.third.nickname) }}</text>
        </view>
        <text class="rank-score pos">{{ formatScore(computedWinners.third.score) }}</text>
      </view>
    </view>

    <view class="card">
      <view class="title">成员</view>
      <view class="tip" v-if="me">点成员记分，点自己可编辑</view>
      <view class="tip" v-else>登录并加入后可记分</view>
      <view class="member-grid">
        <view class="member" v-for="m in members" :key="m.id" @click="onClickMember(m)">
          <view class="avatar-wrap">
            <image class="avatar" :src="m.avatarUrl || fallbackAvatar" mode="aspectFill" />
            <view class="tag avatar-tag me" v-if="m.isMe">我</view>
            <view class="tag avatar-tag owner" v-if="m.isOwner">掌柜</view>
          </view>
          <view class="member-body">
            <view class="member-top">
              <view class="nick">
                <text class="nick-text">{{ displayNickname(m.nickname) }}</text>
              </view>
              <view class="score" :class="scoreTone(m.score)">{{ formatScore(m.score) }}</view>
            </view>
          </view>
        </view>
      </view>
    </view>

    <view class="card">
      <view class="row">
        <view class="title">记录</view>
      </view>
      <view class="empty-records" v-if="recordsLoaded && records.length === 0">暂无记录</view>
      <view class="records" v-else-if="records.length > 0">
        <view class="record" v-for="r in records" :key="r.id">
          <view class="record-row">
            <view class="record-users">
              <image class="record-avatar" :src="avatarOf(r.fromMemberId)" mode="aspectFill" />
              <text class="record-name">{{ nicknameOf(r.fromMemberId) }}</text>
              <text class="record-arrow">→</text>
              <image class="record-avatar" :src="avatarOf(r.toMemberId)" mode="aspectFill" />
              <text class="record-name">{{ nicknameOf(r.toMemberId) }}</text>
            </view>
            <view class="record-meta">
              <text class="record-delta pos">{{ formatScore(r.delta) }}</text>
              <text class="record-time">{{ formatTime(r.createdAt) }}</text>
            </view>
          </view>
          <view class="record-note" v-if="r.note">{{ r.note }}</view>
        </view>
      </view>
      <view class="records-more" v-if="recordsHasMore || recordsPaging">
        <button size="mini" class="more-btn" v-if="recordsHasMore && !recordsPaging" @click="loadMoreRecords">加载更多</button>
        <view class="hint" v-else>加载中…</view>
      </view>
    </view>

	    <view class="modal-mask score-mask" v-if="scoreModalOpen" @click="closeScoreModal" />
	    <view class="modal score-modal" v-if="scoreModalOpen">
	      <view class="modal-head">
	        <view class="modal-title">记分</view>
	        <view class="modal-close" @click="closeScoreModal">×</view>
	      </view>

	      <view class="score-target">
	        <image class="score-target-avatar" :src="scoreTarget?.avatarUrl || fallbackAvatar" mode="aspectFill" />
	        <view class="score-target-body">
	          <view class="score-target-name">{{ displayNickname(scoreTarget?.nickname) }}</view>
	          <view class="score-target-sub">
	            <text class="score-target-sub-label">当前</text>
	            <text class="score-target-sub-score" :class="scoreTone(scoreTarget?.score)">{{ formatScore(scoreTarget?.score) }}</text>
	          </view>
	        </view>
	        <view class="delta-badge" :class="deltaTone(deltaValue)">
	          {{ formatScore(deltaValue) }}
	        </view>
	      </view>

	      <view class="delta-input-row">
	        <input class="delta-input" type="digit" v-model="scoreDelta" placeholder="输入分数" />
	        <button size="mini" class="chip" @click="clearDelta">清零</button>
	      </view>

	      <view class="preset-title">快捷</view>
	      <view class="preset-grid">
	        <button size="mini" class="preset pos" @click="addDelta(5)">+5</button>
	        <button size="mini" class="preset pos" @click="addDelta(10)">+10</button>
	        <button size="mini" class="preset pos" @click="addDelta(20)">+20</button>
	        <button size="mini" class="preset pos" @click="addDelta(30)">+30</button>
	        <button size="mini" class="preset pos" @click="addDelta(50)">+50</button>
	      </view>

	      <view class="preview">
	        <view class="preview-row">
	          <text class="preview-label">对方</text>
	          <text class="preview-before" :class="scoreTone(scoreTarget?.score)">{{ formatScore(scoreTarget?.score) }}</text>
	          <text class="preview-arrow">→</text>
	          <text class="preview-after" :class="scoreTone(targetAfter)">{{ formatScore(targetAfter) }}</text>
	        </view>
	        <view class="preview-row">
	          <text class="preview-label">我</text>
	          <text class="preview-before" :class="scoreTone(myScore)">{{ formatScore(myScore) }}</text>
	          <text class="preview-arrow">→</text>
	          <text class="preview-after" :class="scoreTone(myAfter)">{{ formatScore(myAfter) }}</text>
	        </view>
	      </view>

	      <input class="input" v-model="scoreNote" placeholder="备注（可选）" />

	      <view class="modal-actions">
	        <button size="mini" @click="closeScoreModal">取消</button>
	        <button size="mini" :disabled="scoreSubmitting" @click="submitScore">
	          {{ scoreSubmitting ? '提交中…' : '确认记分' }}
	        </button>
	      </view>
	    </view>

    <view class="modal-mask" v-if="qrModalOpen" @click="closeQRCode" />
    <view class="modal" v-if="qrModalOpen">
      <view class="modal-title">使用微信扫码加入</view>
      <view v-if="qrLoading" class="hint">生成中…</view>
      <image v-else class="qr" :src="qrSrc" mode="widthFix" @click="previewQRCode" />
    </view>

    <view class="modal-mask" v-if="inviteQRModalOpen" @click="closeInviteCodeQR" />
    <view class="modal" v-if="inviteQRModalOpen">
      <view class="modal-title">邀请码二维码</view>
      <view v-if="inviteQRLoading" class="hint">生成中…</view>
      <canvas
        class="invite-qr-canvas"
        canvas-id="inviteQrCanvas"
        id="inviteQrCanvas"
        :style="{ width: `${inviteQRSize}px`, height: `${inviteQRSize}px` }"
        :width="inviteQRSize"
        :height="inviteQRSize"
      />
      <view class="hint">在我的页面点「扫码」即可识别</view>
    </view>

    <view class="modal-mask" v-if="endModalOpen" @click="closeEndModal" />
    <view class="modal" v-if="endModalOpen">
      <view class="modal-title">本局结束</view>
      <view class="hint" v-if="!endWinners?.champion">本局无人得分大于0</view>
      <view class="end-row" v-if="endWinners?.champion">
        <text class="end-label">冠军</text>
        <text class="end-name">{{ displayNickname(endWinners.champion.nickname) }}</text>
        <text class="end-score pos">{{ formatScore(endWinners.champion.score) }}</text>
      </view>
      <view class="end-row" v-if="endWinners?.runnerUp">
        <text class="end-label">亚军</text>
        <text class="end-name">{{ displayNickname(endWinners.runnerUp.nickname) }}</text>
        <text class="end-score pos">{{ formatScore(endWinners.runnerUp.score) }}</text>
      </view>
      <view class="end-row" v-if="endWinners?.third">
        <text class="end-label">季军</text>
        <text class="end-name">{{ displayNickname(endWinners.third.nickname) }}</text>
        <text class="end-score pos">{{ formatScore(endWinners.third.score) }}</text>
      </view>
    </view>

    <view class="fab-mask" v-if="actionMenuOpen && hasActions" @click="closeActionMenu" />
    <view class="fab" v-if="hasActions">
      <view class="fab-panel" :class="{ open: actionMenuOpen }">
        <button size="mini" class="action-btn" v-if="canOpenQRCode" @click="closeActionMenu(); openQRCode()">
          小程序码
        </button>
        <!-- #ifdef MP-WEIXIN -->
        <button size="mini" class="action-btn" v-if="canShare" open-type="share" @click="closeActionMenu">分享</button>
        <!-- #endif -->
        <button size="mini" class="action-btn" v-if="canRename" @click="closeActionMenu(); rename()">改名</button>
        <button size="mini" class="action-btn danger" v-if="canEnd" @click="closeActionMenu(); end()">结束</button>
      </view>
      <button class="fab-toggle" :class="{ active: actionMenuOpen }" @click="toggleActionMenu">
        <image class="fab-icon" :src="actionMenuOpen ? closeIcon : moreIcon" mode="aspectFit" />
      </button>
    </view>
  </view>

  <view class="page" v-else>
    <view class="empty" v-if="loadError">
      <view>{{ loadError }}</view>
      <button class="btn confirm-btn" v-if="!token" @click="goLogin">去「我的」登录</button>
    </view>
    <view class="empty" v-else>加载中…</view>
  </view>
</template>

<script setup lang="ts">
import { onHide, onLoad, onShareAppMessage, onShow, onUnload } from '@dcloudio/uni-app'
import { computed, getCurrentInstance, nextTick, ref } from 'vue'
import {
  connectScorebookWS,
  createRecord,
  endScorebook,
  getInviteQRCode,
  getScorebookDetail,
  listScorebookRecords,
  updateScorebookName,
} from '../../utils/api'
import { makeInviteCodeQRMatrix } from '../../utils/qrcode'
import { clampNickname } from '../../utils/nickname'

const token = ref('')
const id = ref('')
const scorebook = ref<any>(null)
const me = ref<{ memberId: string; isOwner: boolean } | null>(null)
const members = ref<any[]>([])
const socketTask = ref<UniApp.SocketTask | null>(null)
const loadError = ref('')
const isMpWeixin = ref(false)
// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const fallbackAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect width="64" height="64" fill="%23ddd"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" fill="%23666" font-size="14">avatar</text></svg>'
const moreIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="%23ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="5" cy="12" r="1.8"/><circle cx="12" cy="12" r="1.8"/><circle cx="19" cy="12" r="1.8"/></svg>'
const closeIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="%23ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18"/><path d="M6 6l12 12"/></svg>'

const scoreModalOpen = ref(false)
const scoreTarget = ref<any>(null)
const scoreDelta = ref('')
const scoreNote = ref('')
const scoreSubmitting = ref(false)

const records = ref<any[]>([])
const recordsLoaded = ref(false)
const recordsPaging = ref(false)
const recordsHasMore = ref(false)
const recordsNextOffset = ref(0)
const recordsPageSize = 50

const qrModalOpen = ref(false)
const qrLoading = ref(false)
const qrSrc = ref('')
const inviteQRModalOpen = ref(false)
const inviteQRLoading = ref(false)
const inviteQRSize = 232
const endModalOpen = ref(false)
const endWinners = ref<any>(null)
const actionMenuOpen = ref(false)

const refreshing = ref(false)
const refreshQueued = ref(false)
const pollTimer = ref<any>(null)
const localRecordIDs = new Map<string, number>()

function scoreTone(v: any): string {
  const n = Number(v || 0)
  if (n > 0) return 'pos'
  if (n < 0) return 'neg'
  return 'zero'
}

function formatScore(v: any): string {
  const n = Number(v || 0)
  if (!Number.isFinite(n)) return String(v ?? '')
  const abs = Math.abs(n)
  const text = abs.toFixed(2).replace(/\.00$/, '')
  if (n > 0) return `+${text}`
  if (n < 0) return `-${text}`
  return '0'
}

function formatTime(v: any): string {
  const d = new Date(String(v || ''))
  if (Number.isNaN(d.getTime())) return ''
  const now = new Date()
  const yyyy = String(d.getFullYear())
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  if (d.getFullYear() === now.getFullYear()) return `${mm}-${dd} ${hh}:${mi}`
  return `${yyyy}-${mm}-${dd} ${hh}:${mi}`
}

function memberByID(memberID: any) {
  const id = String(memberID || '')
  return members.value.find((m) => String(m.id) === id)
}

function nicknameOf(memberID: any): string {
  const m = memberByID(memberID)
  return displayNickname(m?.nickname || '未知')
}

function avatarOf(memberID: any): string {
  const m = memberByID(memberID)
  return String(m?.avatarUrl || fallbackAvatar)
}

function displayNickname(v: any): string {
  const s = String(v ?? '').trim()
  if (!s) return '未知'
  return clampNickname(s)
}

function deltaTone(v: any): string {
  const n = Number(v || 0)
  if (n > 0) return 'pos'
  if (n < 0) return 'neg'
  return 'zero'
}

function parseAmountSafe(v: any): number {
  const n = Number(v)
  if (!Number.isFinite(n)) return 0
  return Math.round(n * 100) / 100
}

function isTwoDecimals(v: number): boolean {
  if (!Number.isFinite(v)) return false
  return Math.abs(v * 100 - Math.round(v * 100)) < 1e-6
}

const deltaValue = computed(() => parseAmountSafe(scoreDelta.value))
const myMember = computed(() => members.value.find((m) => m.id === me.value?.memberId) || members.value.find((m) => m.isMe))
const myScore = computed(() => Number(myMember.value?.score || 0))
const targetAfter = computed(() => Number(scoreTarget.value?.score || 0) + deltaValue.value)
const myAfter = computed(() => myScore.value - deltaValue.value)
const canOpenQRCode = computed(() => !!me.value?.memberId && scorebook.value?.status !== 'ended')
const canShare = computed(() => isMpWeixin.value)
const canRename = computed(() => !!me.value?.isOwner)
const canEnd = computed(() => !!me.value?.isOwner && scorebook.value?.status !== 'ended')
const hasActions = computed(
  () => canOpenQRCode.value || canShare.value || canRename.value || canEnd.value,
)

function addDelta(v: number) {
  const next = parseAmountSafe(scoreDelta.value) + v
  scoreDelta.value = next === 0 ? '' : String(parseAmountSafe(next))
}

function clearDelta() {
  scoreDelta.value = ''
}

const computedWinners = computed(() => {
  const eligible = (members.value || [])
    .map((m) => ({ ...m, score: Number(m.score || 0) }))
    .filter((m) => Number.isFinite(m.score) && m.score > 0)
    .sort((a, b) => {
      const ds = (b.score as number) - (a.score as number)
      if (ds !== 0) return ds
      const ta = Date.parse(String(a.joinedAt || ''))
      const tb = Date.parse(String(b.joinedAt || ''))
      const na = Number.isFinite(ta) ? ta : 0
      const nb = Number.isFinite(tb) ? tb : 0
      return na - nb
    })

  const champion = eligible[0]
    ? { memberId: eligible[0].id, nickname: eligible[0].nickname, avatarUrl: eligible[0].avatarUrl, score: eligible[0].score }
    : null
  const runnerUp = eligible[1]
    ? { memberId: eligible[1].id, nickname: eligible[1].nickname, avatarUrl: eligible[1].avatarUrl, score: eligible[1].score }
    : null
  const third = eligible[2]
    ? { memberId: eligible[2].id, nickname: eligible[2].nickname, avatarUrl: eligible[2].avatarUrl, score: eligible[2].score }
    : null

  return { champion, runnerUp, third }
})

onShareAppMessage(() => {
  const sb = scorebook.value
  const code = sb?.inviteCode || ''
  return {
    title: sb?.name ? `加入得分簿：${sb.name}` : '得分簿',
    path: code ? `/pages/join/index?code=${encodeURIComponent(code)}` : '/pages/home/index',
  }
})

onLoad(async (q) => {
  token.value = (uni.getStorageSync('token') as string) || ''

  const query = (q || {}) as any
  const code = String(query.code || query.scene || '').trim()
  if (!String(query.id || '').trim() && code) {
    uni.redirectTo({ url: `/pages/join/index?code=${encodeURIComponent(code)}` })
    return
  }

  id.value = String(query.id || '').trim()
  if (!token.value) {
    loadError.value = '未登录，无法查看得分簿。'
    return
  }

  try {
    await refresh()
  } catch (e: any) {
    loadError.value = String(e?.message || '加载失败')
    return
  }

  try {
    socketTask.value = await connectScorebookWS(id.value, onEvent)
  } catch (e) {
    socketTask.value = null
  }
  startPolling()
})

onShow(() => {
  startPolling()
})

onHide(() => {
  stopPolling()
})

onUnload(() => {
  stopPolling()
  try {
    socketTask.value?.close({})
  } catch (e) {}
})

async function refresh() {
  loadError.value = ''
  if (!id.value) {
    throw new Error('缺少参数')
  }

  const res = await getScorebookDetail(id.value)
  scorebook.value = res.scorebook
  me.value = res.me
  members.value = decorateMembers(res.members || [])

  await refreshRecordsFirstPage()
}

function decorateMembers(list: any[]): any[] {
  const myId = String(me.value?.memberId || '')
  return (list || []).map((m) => ({
    ...m,
    score: Number(m?.score || 0),
    isMe: myId && String(m?.id || '') === myId,
    isOwner: String(m?.role || '') === 'owner',
  }))
}

function mergeRecordsTop(incoming: any[]) {
  const map = new Map<string, any>()
  for (const r of records.value || []) {
    const rid = String(r?.id || '')
    if (rid) map.set(rid, r)
  }
  for (const r of incoming || []) {
    const rid = String(r?.id || '')
    if (rid) map.set(rid, r)
  }
  const out = Array.from(map.values())
  out.sort((a, b) => {
    const ta = Date.parse(String(a?.createdAt || ''))
    const tb = Date.parse(String(b?.createdAt || ''))
    const na = Number.isFinite(ta) ? ta : 0
    const nb = Number.isFinite(tb) ? tb : 0
    return nb - na
  })
  records.value = out
}

async function refreshRecordsFirstPage() {
  try {
    const r = await listScorebookRecords(id.value, recordsPageSize, 0)
    const items = r.items || []
    if (!recordsLoaded.value) {
      records.value = items
    } else {
      mergeRecordsTop(items)
    }

    // 仅在“尚未翻页”的情况下，才根据第 1 页更新翻页状态，避免翻到末尾后又被后台刷新把按钮刷出来
    if (recordsNextOffset.value <= recordsPageSize) {
      if (items.length >= recordsPageSize) {
        recordsHasMore.value = true
        recordsNextOffset.value = Math.max(recordsNextOffset.value, recordsPageSize)
      } else {
        recordsHasMore.value = false
        recordsNextOffset.value = Math.max(recordsNextOffset.value, items.length)
      }
    }
  } catch (e) {
    // 后台刷新失败不弹 toast
  } finally {
    recordsLoaded.value = true
  }
}

async function loadMoreRecords() {
  if (recordsPaging.value || !recordsHasMore.value) return
  recordsPaging.value = true
  try {
    const r = await listScorebookRecords(id.value, recordsPageSize, recordsNextOffset.value)
    const items = r.items || []
    const seen = new Set<string>()
    for (const x of records.value || []) {
      const rid = String(x?.id || '')
      if (rid) seen.add(rid)
    }
    for (const it of items) {
      const rid = String(it?.id || '')
      if (!rid || seen.has(rid)) continue
      seen.add(rid)
      records.value.push(it)
    }
    recordsNextOffset.value += items.length
    recordsHasMore.value = items.length >= recordsPageSize
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    recordsPaging.value = false
    recordsLoaded.value = true
  }
}

async function safeRefresh() {
  if (refreshing.value) {
    refreshQueued.value = true
    return
  }
  refreshing.value = true
  try {
    await refresh()
  } catch (e) {
    // 后台刷新失败不弹 toast
  } finally {
    refreshing.value = false
    if (refreshQueued.value) {
      refreshQueued.value = false
      safeRefresh()
    }
  }
}

function startPolling() {
  stopPolling()
  pollTimer.value = setInterval(() => {
    if (scoreModalOpen.value || qrModalOpen.value || inviteQRModalOpen.value || endModalOpen.value || recordsPaging.value) return
    safeRefresh()
  }, 5000)
}

function stopPolling() {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
    pollTimer.value = null
  }
}

function applyRecordToMembers(r: any) {
  if (!r?.toMemberId || !r?.fromMemberId) return
  const delta = Number(r.delta)
  if (!Number.isFinite(delta) || delta === 0) return
  const to = members.value.find((m) => m.id === r.toMemberId)
  if (to) to.score = Number(to.score || 0) + delta
  const from = members.value.find((m) => m.id === r.fromMemberId)
  if (from) from.score = Number(from.score || 0) - delta
}

function prependRecord(r: any) {
  if (!r?.id) return
  const rid = String(r.id)
  if (!rid) return
  if (records.value.some((x) => String(x?.id) === rid)) return
  records.value.unshift(r)
}

function rememberLocalRecordID(id: string) {
  localRecordIDs.set(id, Date.now())
  const now = Date.now()
  for (const [k, ts] of localRecordIDs.entries()) {
    if (now - ts > 60_000) localRecordIDs.delete(k)
  }
}

function onEvent(evt: any) {
  if (!evt?.type) return
  if (evt.type === 'record.created') {
    const r = evt.data?.record
    const rid = String(r?.id || '')
    if (rid && localRecordIDs.has(rid)) {
      localRecordIDs.delete(rid)
      return
    }
    applyRecordToMembers(r)
    prependRecord(r)
    return
  }
	  if (evt.type === 'member.joined') {
	    const m = evt.data?.member
	    if (!m?.id) return
	    if (!members.value.some((x) => x.id === m.id)) {
	      members.value.push({
	        ...m,
	        score: 0,
	        isMe: m.id === me.value?.memberId,
	        isOwner: m.role === 'owner',
	      })
	    }
	    return
	  }
  if (evt.type === 'member.updated') {
    const m = evt.data?.member
    if (!m?.id) return
    const t = members.value.find((x) => x.id === m.id)
    if (!t) return
    if (m.nickname !== undefined) t.nickname = m.nickname
    if (m.avatarUrl !== undefined) t.avatarUrl = m.avatarUrl
    return
  }
  if (evt.type === 'scorebook.updated') {
    if (scorebook.value && evt.data?.name) scorebook.value.name = evt.data.name
    return
  }
  if (evt.type === 'scorebook.ended') {
    const wasEnded = scorebook.value?.status === 'ended'
    if (scorebook.value) scorebook.value.status = 'ended'
    if (!wasEnded) {
      // 结束时弹出冠亚军（得分 > 0）
      scoreModalOpen.value = false
      qrModalOpen.value = false
      endWinners.value = evt?.data?.winners || computedWinners.value
      endModalOpen.value = true
    }
    return
  }
  safeRefresh()
}

function copyInvite() {
  if (!scorebook.value?.inviteCode) return
  uni.setClipboardData({ data: scorebook.value.inviteCode })
}

async function rename() {
  const current = scorebook.value?.name || ''
  uni.showModal({
    title: '修改名称',
    editable: true,
    placeholderText: current,
    success: async (res) => {
      if (!res.confirm) return
      const name = String((res as any).content || '').trim()
      if (!name) return
      try {
        await updateScorebookName(id.value, name)
      } catch (e: any) {
        uni.showToast({ title: e?.message || '修改失败', icon: 'none' })
      }
    },
  } as any)
}

async function end() {
  uni.showModal({
    title: '结束得分簿',
    content: '结束后不可再记分，且无法恢复。',
    showCancel: true,
    confirmText: '确认结束',
    cancelText: '取消',
    confirmColor: '#ff4d4f',
    success: async (res) => {
      if (!res.confirm) return
      try {
        const wasEnded = scorebook.value?.status === 'ended'
        const body = await endScorebook(id.value)
        if (body?.scorebook) scorebook.value = body.scorebook
        scoreModalOpen.value = false
        qrModalOpen.value = false
        if (!wasEnded) {
          endWinners.value = body?.winners || computedWinners.value
          endModalOpen.value = true
        }
      } catch (e: any) {
        uni.showToast({ title: e?.message || '结束失败', icon: 'none' })
      }
    },
  })
}

function onClickMember(m: any) {
  if (m.isMe) {
    uni.navigateTo({
      url: `/pages/profile/edit?mode=scorebook&id=${encodeURIComponent(id.value)}`,
    })
    return
  }
  if (scorebook.value?.status === 'ended') {
    uni.showToast({ title: '已结束，不能记分', icon: 'none' })
    return
  }
  scoreTarget.value = m
  scoreDelta.value = ''
  scoreNote.value = ''
  scoreModalOpen.value = true
}

function closeScoreModal() {
  scoreModalOpen.value = false
}

function inviteQRCodeCacheKey(scorebookId: string): string {
  return `scorehub.inviteQRCode.${encodeURIComponent(String(scorebookId || '').trim())}`
}

function getCachedInviteQRCode(scorebookId: string): string {
  const key = inviteQRCodeCacheKey(scorebookId)
  if (!key) return ''
  try {
    const v: any = uni.getStorageSync(key)
    if (!v) return ''
    if (typeof v === 'string') return v
    if (typeof v === 'object' && v) {
      const src = String(v.src || '').trim()
      return src
    }
    return ''
  } catch (e) {
    return ''
  }
}

function setCachedInviteQRCode(scorebookId: string, src: string) {
  const key = inviteQRCodeCacheKey(scorebookId)
  if (!key) return
  const next = String(src || '').trim()
  if (!next) return
  try {
    uni.setStorageSync(key, { src: next, ts: Date.now() })
  } catch (e) {
    // ignore storage errors (quota etc.)
  }
}

async function openQRCode() {
  if (scorebook.value?.status === 'ended') {
    uni.showToast({ title: '已结束，不能加入', icon: 'none' })
    return
  }
  qrModalOpen.value = true
  qrLoading.value = true
  try {
    const cached = getCachedInviteQRCode(id.value)
    if (cached) {
      qrSrc.value = cached
      return
    }
    const fresh = await getInviteQRCode(id.value)
    qrSrc.value = fresh
    setCachedInviteQRCode(id.value, fresh)
  } catch (e: any) {
    uni.showToast({ title: e?.message || '生成二维码失败', icon: 'none' })
    qrModalOpen.value = false
  } finally {
    qrLoading.value = false
  }
}

function closeQRCode() {
  qrModalOpen.value = false
}

function toggleActionMenu() {
  if (!hasActions.value) return
  actionMenuOpen.value = !actionMenuOpen.value
}

function closeActionMenu() {
  actionMenuOpen.value = false
}

function previewQRCode() {
  if (!qrSrc.value) return
  uni.previewImage({ urls: [qrSrc.value] })
}

function goLogin() {
  uni.setStorageSync('scorehub.afterLogin', { to: 'home', ts: Date.now() })
  uni.switchTab({ url: '/pages/my/index' })
}

async function openInviteCodeQR() {
  const code = String(scorebook.value?.inviteCode || '').trim()
  if (!code) return

  // #ifndef MP-WEIXIN
  uni.showToast({ title: '请在微信小程序内使用', icon: 'none' })
  return
  // #endif

  // #ifdef MP-WEIXIN
  inviteQRModalOpen.value = true
  inviteQRLoading.value = true
  scoreModalOpen.value = false
  qrModalOpen.value = false
  endModalOpen.value = false

  try {
    await nextTick()
    await drawInviteCodeQR(code)
  } catch (e: any) {
    inviteQRModalOpen.value = false
    const raw = String(e?.message || '')
    const msg = raw || '生成二维码失败'
    uni.showToast({ title: msg, icon: 'none' })
  } finally {
    inviteQRLoading.value = false
  }
  // #endif
}

function closeInviteCodeQR() {
  inviteQRModalOpen.value = false
}

function drawInviteCodeQR(code: string): Promise<void> {
  const instance = getCurrentInstance()
  const proxy = (instance?.proxy as any) || undefined
  const matrix = makeInviteCodeQRMatrix(code)

  const n = matrix.length
  const margin = 4
  const moduleSize = Math.max(1, Math.floor(inviteQRSize / (n + margin * 2)))
  const drawSize = moduleSize * (n + margin * 2)

  const ctx = uni.createCanvasContext('inviteQrCanvas', proxy)
  ctx.setFillStyle('#ffffff')
  ctx.fillRect(0, 0, drawSize, drawSize)
  ctx.setFillStyle('#000000')
  for (let r = 0; r < n; r++) {
    for (let c = 0; c < n; c++) {
      if (!matrix[r][c]) continue
      const x = (c + margin) * moduleSize
      const y = (r + margin) * moduleSize
      ctx.fillRect(x, y, moduleSize, moduleSize)
    }
  }

  return new Promise((resolve) => {
    ctx.draw(false, resolve)
  })
}

function closeEndModal() {
  endModalOpen.value = false
}

async function submitScore() {
  const rawDelta = Number(scoreDelta.value)
  const delta = deltaValue.value
  if (!scoreTarget.value?.id) return
  if (!Number.isFinite(rawDelta) || rawDelta <= 0) return uni.showToast({ title: '请输入正分数', icon: 'none' })
  if (!isTwoDecimals(rawDelta)) return uni.showToast({ title: '最多两位小数', icon: 'none' })
  if (scoreSubmitting.value) return
  try {
    scoreSubmitting.value = true
    const res = await createRecord(id.value, { toMemberId: scoreTarget.value.id, delta, note: scoreNote.value.trim() })
    if (res?.record?.id) {
      rememberLocalRecordID(String(res.record.id))
      applyRecordToMembers(res.record)
      prependRecord(res.record)
    }
    closeScoreModal()
    scoreDelta.value = ''
    scoreNote.value = ''
    safeRefresh()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '记分失败', icon: 'none' })
  } finally {
    scoreSubmitting.value = false
  }
}
</script>

<style scoped>
.page {
  padding: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 24rpx;
  background: #f6f7fb;
  min-height: 100vh;
}
.card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(0, 0, 0, 0.06);
}
.row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.hero {
  background: linear-gradient(135deg, #111 0%, #2b2b2b 100%);
  color: #fff;
  position: relative;
  overflow: hidden;
  border-radius: 20rpx;
}
.hero::before {
  content: '';
  position: absolute;
  right: -120rpx;
  top: -140rpx;
  width: 360rpx;
  height: 360rpx;
  border-radius: 999rpx;
  background: radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.18), rgba(255, 255, 255, 0));
  transform: rotate(12deg);
  pointer-events: none;
}
.hero::after {
  content: '';
  position: absolute;
  left: -140rpx;
  bottom: -180rpx;
  width: 420rpx;
  height: 420rpx;
  border-radius: 999rpx;
  background: radial-gradient(circle at 60% 40%, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0));
  transform: rotate(-10deg);
  pointer-events: none;
}
.hero-row {
  align-items: flex-start;
  gap: 14rpx;
  position: relative;
  z-index: 1;
}
.name {
  font-size: 36rpx;
  font-weight: 700;
  flex: 1;
  min-width: 0;
  line-height: 1.25;
  word-break: break-all;
  white-space: normal;
}
.badge {
  font-size: 24rpx;
  padding: 8rpx 12rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.16);
  color: #fff;
  white-space: nowrap;
  flex: none;
  display: flex;
  align-items: center;
  gap: 8rpx;
  position: relative;
  z-index: 1;
}
.badge.ended {
  background: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.85);
}
.badge-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 999rpx;
  background: rgba(0, 200, 83, 0.95);
  box-shadow: 0 0 0 6rpx rgba(0, 200, 83, 0.18);
  flex: none;
}
.badge-dot.ended {
  background: rgba(255, 255, 255, 0.75);
  box-shadow: 0 0 0 6rpx rgba(255, 255, 255, 0.12);
}
.sub {
  margin-top: 8rpx;
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
  position: relative;
  z-index: 1;
}
.hero-sub {
  margin-top: 16rpx;
}
.pill {
  font-size: 24rpx;
  padding: 8rpx 12rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.14);
  color: rgba(255, 255, 255, 0.92);
  border: 1rpx solid rgba(255, 255, 255, 0.12);
}
.pill.code:active {
  opacity: 0.85;
}
.pill.code {
  display: flex;
  align-items: center;
  gap: 8rpx;
}
.qr-icon {
  width: 34rpx;
  height: 34rpx;
  border-radius: 8rpx;
  border: 1rpx solid rgba(255, 255, 255, 0.28);
  background: rgba(255, 255, 255, 0.12);
  position: relative;
  overflow: hidden;
}
.qr-icon:active {
  opacity: 0.85;
}
.qr-finder {
  position: absolute;
  width: 10rpx;
  height: 10rpx;
  border: 2rpx solid rgba(255, 255, 255, 0.92);
  border-radius: 3rpx;
  box-sizing: border-box;
}
.qr-finder-inner {
  position: absolute;
  left: 2rpx;
  top: 2rpx;
  width: 4rpx;
  height: 4rpx;
  border-radius: 2rpx;
  background: rgba(255, 255, 255, 0.92);
}
.qr-finder.tl {
  left: 4rpx;
  top: 4rpx;
}
.qr-finder.tr {
  right: 4rpx;
  top: 4rpx;
}
.qr-finder.bl {
  left: 4rpx;
  bottom: 4rpx;
}
.qr-dot {
  position: absolute;
  width: 4rpx;
  height: 4rpx;
  border-radius: 2rpx;
  background: rgba(255, 255, 255, 0.92);
}
.qr-dot.d1 {
  left: 16rpx;
  top: 8rpx;
}
.qr-dot.d2 {
  left: 22rpx;
  top: 14rpx;
}
.qr-dot.d3 {
  left: 16rpx;
  top: 18rpx;
}
.qr-dot.d4 {
  left: 22rpx;
  top: 22rpx;
}
.qr-dot.d5 {
  left: 12rpx;
  top: 14rpx;
}
.qr-dot.d6 {
  left: 20rpx;
  top: 10rpx;
}
.qr-dot.d7 {
  left: 14rpx;
  top: 24rpx;
}
.invite-qr-canvas {
  margin: 16rpx auto 10rpx;
  background: #fff;
  border-radius: 16rpx;
  box-shadow: 0 10rpx 30rpx rgba(0, 0, 0, 0.08);
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  letter-spacing: 1rpx;
}
.fab-mask {
  position: fixed;
  left: 0;
  right: 0;
  top: 0;
  bottom: 0;
  z-index: 40;
}
.fab {
  position: fixed;
  right: 24rpx;
  bottom: calc(28rpx + env(safe-area-inset-bottom));
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12rpx;
  z-index: 41;
}
.fab-panel {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0;
  padding: 6rpx 0;
  border-radius: 14rpx;
  background: #fff;
  border: 1rpx solid rgba(0, 0, 0, 0.06);
  box-shadow: 0 10rpx 26rpx rgba(0, 0, 0, 0.12);
  transform: translateY(10rpx);
  opacity: 0;
  pointer-events: none;
  transition: all 0.2s ease;
}
.fab-panel.open {
  transform: translateY(0);
  opacity: 1;
  pointer-events: auto;
}
.fab-panel .action-btn {
  width: 200rpx;
  text-align: left;
}
.fab-toggle {
  width: 70rpx;
  height: 70rpx;
  border-radius: 999rpx;
  background: rgba(120, 120, 120, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid rgba(255, 255, 255, 0.18);
  box-shadow: 0 10rpx 24rpx rgba(0, 0, 0, 0.16);
  transition: all 0.2s ease;
}
.fab-toggle::after {
  border: none;
}
.fab-toggle.active {
  background: rgba(120, 120, 120, 0.6);
  border-color: rgba(255, 255, 255, 0.22);
  box-shadow: 0 12rpx 26rpx rgba(0, 0, 0, 0.2);
}
.fab-toggle:active {
  transform: scale(0.98);
}
.fab-icon {
  width: 28rpx;
  height: 28rpx;
}
.action-btn {
  position: relative;
  background: transparent;
  color: #444;
  border-radius: 0;
  height: 64rpx;
  line-height: 64rpx;
  padding: 0 12rpx;
  font-size: 26rpx;
  font-weight: 500;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
}
.fab-panel .action-btn {
  background-image: linear-gradient(#eee, #eee);
  background-repeat: no-repeat;
  background-position: center bottom;
  background-size: 60% 1rpx;
}
.fab-panel .action-btn:last-child {
  background-image: none;
}
.action-btn::after {
  border: none;
}
.action-btn:active {
  opacity: 0.85;
}
.action-btn.danger {
  background: transparent;
  color: #d92d20;
}
.title {
  font-size: 30rpx;
  font-weight: 600;
  margin-bottom: 12rpx;
}
.tip {
  color: #666;
  font-size: 24rpx;
  margin-bottom: 12rpx;
}
.rank-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  padding: 14rpx 0;
  border-top: 1rpx solid rgba(0, 0, 0, 0.06);
}
.rank-row:first-of-type {
  border-top: none;
  padding-top: 0;
}
.rank-label {
  width: 80rpx;
  color: #666;
  font-size: 24rpx;
}
.rank-user {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10rpx;
}
.rank-avatar {
  width: 44rpx;
  height: 44rpx;
  border-radius: 12rpx;
  background: #f6f7fb;
}
.rank-name {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 26rpx;
  color: #111;
}
.rank-score {
  font-size: 30rpx;
  font-weight: 700;
}
.member-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16rpx;
}
.member {
  background: #f6f7fb;
  border-radius: 16rpx;
  padding: 16rpx;
  height: 120rpx;
  box-sizing: border-box;
  display: flex;
  gap: 12rpx;
  align-items: center;
  transition: opacity 0.08s ease;
}
.member:active {
  opacity: 0.92;
}
.avatar-wrap {
  width: 88rpx;
  height: 88rpx;
  flex: none;
  position: relative;
}
.avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 16rpx;
  background: #ddd;
  flex: none;
}
.tag.avatar-tag {
  position: absolute;
  top: 6rpx;
  z-index: 1;
  font-size: 18rpx;
  padding: 0 6rpx;
  line-height: 28rpx;
}
.tag.avatar-tag.owner {
  left: 6rpx;
}
.tag.avatar-tag.me {
  right: 6rpx;
}
.member-body {
  flex: 1;
  min-width: 0;
}
.member-top {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6rpx;
}
.nick {
  display: flex;
  align-items: center;
  gap: 8rpx;
  flex: 1;
  min-width: 0;
}
.nick-text {
  font-size: 26rpx;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.tag {
  font-size: 20rpx;
  padding: 2rpx 8rpx;
  border-radius: 999rpx;
  background: #111;
  color: #fff;
}
.tag.owner {
  background: #ff9800;
}
.score {
  font-size: 36rpx;
  font-weight: 700;
  text-align: left;
  flex: none;
}
.score.pos {
  color: #00c853;
}
.score.neg {
  color: #ff5252;
}
.score.zero {
  color: #333;
}
.end-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  margin-top: 10rpx;
}
.end-label {
  color: #666;
  font-size: 26rpx;
  width: 80rpx;
}
.end-name {
  flex: 1;
  min-width: 0;
  font-size: 28rpx;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.end-score {
  font-size: 30rpx;
  font-weight: 700;
}
.pos {
  color: #00c853;
}
.neg {
  color: #ff5252;
}
.zero {
  color: #333;
}
.empty {
  margin-top: 120rpx;
  text-align: center;
  color: #666;
}
.modal-mask {
  position: fixed;
  z-index: 1000;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.4);
}
.modal-mask.score-mask {
  z-index: 1200;
}
.modal {
  position: fixed;
  z-index: 1001;
  left: 24rpx;
  right: 24rpx;
  bottom: 24rpx;
  background: #fff;
  border-radius: 20rpx;
  padding: 20rpx;
  box-shadow: 0 18rpx 50rpx rgba(0, 0, 0, 0.18);
}
.score-modal {
  top: 50%;
  bottom: auto;
  transform: translateY(-50%);
  max-height: 80vh;
  overflow: auto;
  z-index: 1201;
}
.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  margin-bottom: 10rpx;
}
.modal-title {
  font-size: 30rpx;
  font-weight: 600;
  margin-bottom: 0;
}
.modal-close {
  width: 56rpx;
  height: 56rpx;
  line-height: 56rpx;
  text-align: center;
  border-radius: 999rpx;
  background: #f6f7fb;
  color: #333;
  font-size: 36rpx;
}
.modal-close:active {
  opacity: 0.85;
}
.input {
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 28rpx;
  margin-top: 12rpx;
}
.score-target {
  margin-top: 4rpx;
  display: flex;
  align-items: center;
  gap: 12rpx;
  background: #f6f7fb;
  border-radius: 16rpx;
  padding: 14rpx;
}
.score-target-avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 18rpx;
  background: #fff;
}
.score-target-body {
  flex: 1;
  min-width: 0;
}
.score-target-name {
  font-size: 28rpx;
  font-weight: 600;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.score-target-sub {
  margin-top: 4rpx;
  display: flex;
  align-items: center;
  gap: 8rpx;
  color: #666;
  font-size: 24rpx;
}
.score-target-sub-score {
  font-weight: 700;
}
.delta-badge {
  min-width: 120rpx;
  padding: 10rpx 12rpx;
  border-radius: 999rpx;
  text-align: center;
  font-size: 30rpx;
  font-weight: 800;
  background: #fff;
  color: #333;
}
.delta-badge.pos {
  background: rgba(0, 200, 83, 0.12);
  color: #00c853;
}
.delta-badge.neg {
  background: rgba(255, 82, 82, 0.12);
  color: #ff5252;
}
.delta-input-row {
  margin-top: 12rpx;
  display: flex;
  align-items: center;
  gap: 12rpx;
}
.delta-input {
  flex: 1;
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 34rpx;
  font-weight: 700;
}
.chip {
  border-radius: 12rpx;
}
.preset-title {
  margin-top: 14rpx;
  color: #666;
  font-size: 24rpx;
}
.preset-grid {
  margin-top: 10rpx;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10rpx;
}
.preset {
  border-radius: 14rpx;
  background: #f6f7fb;
}
.preset::after {
  border: none;
}
.preset.pos {
  color: #00c853;
}
.preset.neg {
  color: #ff5252;
}
.preview {
  margin-top: 12rpx;
  padding: 12rpx 14rpx;
  border-radius: 16rpx;
  background: #fff;
  border: 1rpx solid rgba(0, 0, 0, 0.06);
}
.preview-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10rpx;
  margin-top: 8rpx;
}
.preview-row:first-child {
  margin-top: 0;
}
.preview-label {
  width: 64rpx;
  color: #666;
  font-size: 24rpx;
}
.preview-before,
.preview-after {
  font-size: 28rpx;
  font-weight: 700;
}
.preview-arrow {
  color: #999;
  font-size: 22rpx;
}

.records {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}
.record {
  background: #f6f7fb;
  border-radius: 16rpx;
  padding: 14rpx 14rpx;
}
.record-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}
.record-users {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
  flex: 1;
}
.record-avatar {
  width: 44rpx;
  height: 44rpx;
  border-radius: 12rpx;
  background: #ddd;
  flex: none;
}
.record-name {
  max-width: 180rpx;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 26rpx;
  color: #111;
}
.record-arrow {
  color: #999;
  font-size: 24rpx;
  flex: none;
}
.record-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4rpx;
  flex: none;
}
.record-delta {
  font-size: 28rpx;
  font-weight: 800;
}
.record-time {
  font-size: 22rpx;
  color: #888;
}
.record-note {
  margin-top: 10rpx;
  font-size: 24rpx;
  color: #666;
  line-height: 1.5;
  word-break: break-all;
}
.records-more {
  margin-top: 12rpx;
  display: flex;
  justify-content: center;
}
.more-btn {
  border-radius: 999rpx;
  padding: 0 26rpx;
}
.more-btn::after {
  border: none;
}
.empty-records {
  color: #888;
  font-size: 24rpx;
  padding: 10rpx 0;
}
.score-modal .chip,
.score-modal .preset,
.score-modal .modal-actions button {
  height: 72rpx;
  line-height: 72rpx;
  font-size: 28rpx;
  font-weight: 600;
}
.score-modal .chip::after,
.score-modal .modal-actions button::after {
  border: none;
}
.score-modal .modal-actions {
  justify-content: space-between;
}
.score-modal .modal-actions button {
  flex: 1;
  border-radius: 14rpx;
}
.modal-actions {
  margin-top: 16rpx;
  display: flex;
  justify-content: flex-end;
  gap: 12rpx;
}
.qr {
  width: 100%;
  border-radius: 12rpx;
  background: #f6f7fb;
}
.hint {
  color: #666;
  font-size: 26rpx;
}
.primary {
  background: #111;
  color: #fff;
}
</style>
