<template>
  <view class="page" v-if="scorebook">
    <view class="card hero">
      <view class="row hero-row">
        <view class="name">{{ scorebook.name }}</view>
        <view class="badge" :class="{ ended: scorebook.status === 'ended' }">
          {{ scorebook.status === 'ended' ? '已结束' : '记录中' }}
        </view>
      </view>
      <view class="sub hero-sub">
        <view class="pill" v-if="scorebook.locationText">{{ scorebook.locationText }}</view>
        <view class="pill">成员 {{ members.length }}</view>
        <view class="pill code" @click="copyInvite">
          邀请码 <text class="mono">{{ scorebook.inviteCode }}</text>
        </view>
      </view>
      <view class="actions">
        <button size="mini" class="action-btn" @click="copyInvite">复制邀请码</button>
        <button size="mini" class="action-btn" v-if="scorebook.status !== 'ended'" @click="openQRCode">二维码</button>
        <!-- #ifdef MP-WEIXIN -->
        <button size="mini" class="action-btn" open-type="share">分享</button>
        <!-- #endif -->
        <button size="mini" class="action-btn" v-if="me?.isOwner" @click="rename">改名</button>
        <button size="mini" class="action-btn danger" v-if="me?.isOwner && scorebook.status !== 'ended'" @click="end">结束</button>
      </view>
    </view>

    <view class="card" v-if="scorebook.status === 'ended' && (computedWinners.champion || computedWinners.runnerUp)">
      <view class="title">本局排名（得分 &gt; 0）</view>
      <view class="rank-row" v-if="computedWinners.champion">
        <text class="rank-label">冠军</text>
        <view class="rank-user">
          <image class="rank-avatar" :src="computedWinners.champion.avatarUrl || fallbackAvatar" mode="aspectFill" />
          <text class="rank-name">{{ computedWinners.champion.nickname }}</text>
        </view>
        <text class="rank-score pos">{{ formatScore(computedWinners.champion.score) }}</text>
      </view>
      <view class="rank-row" v-if="computedWinners.runnerUp">
        <text class="rank-label">亚军</text>
        <view class="rank-user">
          <image class="rank-avatar" :src="computedWinners.runnerUp.avatarUrl || fallbackAvatar" mode="aspectFill" />
          <text class="rank-name">{{ computedWinners.runnerUp.nickname }}</text>
        </view>
        <text class="rank-score pos">{{ formatScore(computedWinners.runnerUp.score) }}</text>
      </view>
    </view>

    <view class="card">
      <view class="title">成员（点头像记分）</view>
      <view class="tip">点自己可修改头像/昵称</view>
      <view class="grid">
        <view class="member" :class="{ me: m.isMe }" v-for="m in members" :key="m.id" @click="onClickMember(m)">
          <image class="avatar" :src="m.avatarUrl || fallbackAvatar" mode="aspectFill" />
          <view class="member-body">
            <view class="member-top">
              <view class="nick">
                <text class="tag" v-if="m.isMe">我</text>
                <text class="tag owner" v-if="m.isOwner">掌柜</text>
                <text class="nick-text">{{ m.nickname }}</text>
              </view>
              <view class="score" :class="scoreTone(m.score)">{{ formatScore(m.score) }}</view>
            </view>
          </view>
        </view>
      </view>
    </view>

    <view class="modal-mask" v-if="scoreModalOpen" @click="closeScoreModal" />
    <view class="modal" v-if="scoreModalOpen">
      <view class="modal-title">给「{{ scoreTarget?.nickname }}」记分</view>
      <input class="input" v-model="scoreDelta" placeholder="分数（例如 10 或 -5）" />
      <input class="input" v-model="scoreNote" placeholder="备注（可选）" />
      <view class="modal-actions">
        <button size="mini" @click="closeScoreModal">取消</button>
        <button size="mini" class="primary" @click="submitScore">确认</button>
      </view>
    </view>

    <view class="modal-mask" v-if="qrModalOpen" @click="closeQRCode" />
    <view class="modal" v-if="qrModalOpen">
      <view class="modal-title">扫码加入（进行中）</view>
      <view v-if="qrLoading" class="hint">生成中…</view>
      <image v-else class="qr" :src="qrSrc" mode="widthFix" @click="previewQRCode" />
      <view class="modal-actions">
        <button size="mini" @click="closeQRCode">关闭</button>
      </view>
    </view>

    <view class="modal-mask" v-if="endModalOpen" @click="closeEndModal" />
    <view class="modal" v-if="endModalOpen">
      <view class="modal-title">本局结束</view>
      <view class="hint" v-if="!endWinners?.champion">本局无人得分 &gt; 0</view>
      <view class="end-row" v-if="endWinners?.champion">
        <text class="end-label">冠军</text>
        <text class="end-name">{{ endWinners.champion.nickname }}</text>
        <text class="end-score pos">{{ formatScore(endWinners.champion.score) }}</text>
      </view>
      <view class="end-row" v-if="endWinners?.runnerUp">
        <text class="end-label">亚军</text>
        <text class="end-name">{{ endWinners.runnerUp.nickname }}</text>
        <text class="end-score pos">{{ formatScore(endWinners.runnerUp.score) }}</text>
      </view>
      <view class="modal-actions">
        <button size="mini" @click="closeEndModal">关闭</button>
      </view>
    </view>
  </view>

  <view class="page" v-else>
    <view class="empty">加载中…</view>
  </view>
</template>

<script setup lang="ts">
import { onLoad, onShareAppMessage, onUnload } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { connectScorebookWS, createRecord, endScorebook, getInviteQRCode, getScorebookDetail, updateScorebookName } from '../../utils/api'

const id = ref('')
const scorebook = ref<any>(null)
const me = ref<{ memberId: string; isOwner: boolean } | null>(null)
const members = ref<any[]>([])
const socketTask = ref<UniApp.SocketTask | null>(null)

const fallbackAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect width="64" height="64" fill="%23ddd"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" fill="%23666" font-size="14">avatar</text></svg>'

const scoreModalOpen = ref(false)
const scoreTarget = ref<any>(null)
const scoreDelta = ref('')
const scoreNote = ref('')

const qrModalOpen = ref(false)
const qrLoading = ref(false)
const qrSrc = ref('')
const endModalOpen = ref(false)
const endWinners = ref<any>(null)

function scoreTone(v: any): string {
  const n = Number(v || 0)
  if (n > 0) return 'pos'
  if (n < 0) return 'neg'
  return 'zero'
}

function formatScore(v: any): string {
  const n = Number(v || 0)
  if (!Number.isFinite(n)) return String(v ?? '')
  return n > 0 ? `+${n}` : String(n)
}

const computedWinners = computed(() => {
  const eligible = (members.value || [])
    .map((m) => ({ ...m, score: Number(m.score || 0) }))
    .filter((m) => Number.isFinite(m.score) && m.score > 0)
    .sort((a, b) => (b.score as number) - (a.score as number))

  const champion = eligible[0]
    ? { memberId: eligible[0].id, nickname: eligible[0].nickname, avatarUrl: eligible[0].avatarUrl, score: eligible[0].score }
    : null
  const runnerUp = eligible[1]
    ? { memberId: eligible[1].id, nickname: eligible[1].nickname, avatarUrl: eligible[1].avatarUrl, score: eligible[1].score }
    : null

  return { champion, runnerUp }
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
  id.value = String((q as any).id || '')
  await refresh()
  socketTask.value = connectScorebookWS(id.value, onEvent)
})

onUnload(() => {
  try {
    socketTask.value?.close({})
  } catch (e) {}
})

async function refresh() {
  const res = await getScorebookDetail(id.value)
  scorebook.value = res.scorebook
  me.value = res.me
  members.value = res.members || []
}

function onEvent(evt: any) {
  if (!evt?.type) return
  if (evt.type === 'record.created') {
    const r = evt.data?.record
    if (!r?.toMemberId || !r?.fromMemberId) return
    const delta = Number(r.delta)
    if (!Number.isFinite(delta) || delta === 0) return
    const to = members.value.find((m) => m.id === r.toMemberId)
    if (to) to.score = Number(to.score || 0) + delta
    const from = members.value.find((m) => m.id === r.fromMemberId)
    if (from) from.score = Number(from.score || 0) - delta
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
  refresh()
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
    content: '只有掌柜可以结束，结束后不可再记分。',
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
      url: `/pages/scorebook/profile?id=${encodeURIComponent(id.value)}&nickname=${encodeURIComponent(
        m.nickname || ''
      )}&avatarUrl=${encodeURIComponent(m.avatarUrl || '')}`,
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

async function openQRCode() {
  if (scorebook.value?.status === 'ended') {
    uni.showToast({ title: '已结束，不能加入', icon: 'none' })
    return
  }
  qrModalOpen.value = true
  qrLoading.value = true
  try {
    qrSrc.value = await getInviteQRCode(id.value)
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

function previewQRCode() {
  if (!qrSrc.value) return
  uni.previewImage({ urls: [qrSrc.value] })
}

function closeEndModal() {
  endModalOpen.value = false
}

async function submitScore() {
  const delta = Number(scoreDelta.value)
  if (!scoreTarget.value?.id) return
  if (!delta) return uni.showToast({ title: '请输入分数', icon: 'none' })
  try {
    await createRecord(id.value, { toMemberId: scoreTarget.value.id, delta, note: scoreNote.value.trim() })
    closeScoreModal()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '记分失败', icon: 'none' })
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
}
.hero-row {
  align-items: flex-start;
}
.name {
  font-size: 34rpx;
  font-weight: 700;
}
.badge {
  font-size: 24rpx;
  padding: 6rpx 12rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.16);
  color: #fff;
}
.badge.ended {
  background: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.85);
}
.sub {
  margin-top: 8rpx;
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}
.hero-sub {
  margin-top: 14rpx;
}
.pill {
  font-size: 24rpx;
  padding: 8rpx 12rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.14);
  color: rgba(255, 255, 255, 0.92);
}
.pill.code:active {
  opacity: 0.85;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  letter-spacing: 1rpx;
}
.actions {
  margin-top: 16rpx;
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}
.action-btn {
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
  border-radius: 12rpx;
}
.action-btn::after {
  border: none;
}
.action-btn:active {
  opacity: 0.85;
}
.action-btn.danger {
  background: rgba(255, 77, 79, 0.2);
  color: #ffd1d1;
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
.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16rpx;
}
.member {
  background: #f6f7fb;
  border-radius: 16rpx;
  padding: 16rpx;
  display: flex;
  gap: 12rpx;
  align-items: center;
  transition: transform 0.08s ease;
}
.member:active {
  transform: scale(0.98);
}
.member.me {
  background: #fff;
  border: 1rpx solid rgba(17, 17, 17, 0.16);
}
.avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 16rpx;
  background: #ddd;
  flex: none;
}
.member-body {
  flex: 1;
  min-width: 0;
}
.member-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12rpx;
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
  text-align: right;
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
.empty {
  margin-top: 120rpx;
  text-align: center;
  color: #666;
}
.modal-mask {
  position: fixed;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.4);
}
.modal {
  position: fixed;
  left: 24rpx;
  right: 24rpx;
  bottom: 24rpx;
  background: #fff;
  border-radius: 20rpx;
  padding: 20rpx;
  box-shadow: 0 18rpx 50rpx rgba(0, 0, 0, 0.18);
}
.modal-title {
  font-size: 30rpx;
  font-weight: 600;
  margin-bottom: 12rpx;
}
.input {
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 28rpx;
  margin-top: 12rpx;
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
