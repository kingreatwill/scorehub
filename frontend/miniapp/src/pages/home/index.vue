<template>
  <view class="page">
    <view class="card">
      <view class="title">开始新的得分簿</view>
      <input class="input" v-model="newName" placeholder="名称（可空，默认 时间 + 位置）" />
      <view class="loc-row">
        <text class="hint">位置：{{ currentLocationText || '未获取' }}</text>
        <button
          size="mini"
          class="loc-btn"
          v-if="isMpWeixin"
          @click="refreshLocation"
          :disabled="locating"
          hover-class="none"
        >
          <view class="loc-logo" :class="{ loading: locating }">
            <view class="loc-logo-dot" />
          </view>
        </button>
      </view>
      <button class="btn" @click="onCreate">开始</button>
    </view>

    <view class="card">
      <view class="invite-row">
        <input class="input invite-input" v-model="inviteCode" placeholder="邀请码（例如 8 位码）" />
        <button
          size="mini"
          class="scan-btn"
          v-if="isMpWeixin"
          @click="onScanInviteToInput"
          hover-class="none"
        >
          <image class="scan-icon" :src="scanIcon" mode="aspectFit" />
        </button>
      </view>
      <button class="btn primary" v-if="token" @click="onJoinByCode">邀请码加入</button>
      <button class="btn primary" v-else @click="goLogin">邀请码加入</button>
    </view>

    <view class="card" v-if="token">
      <view class="title">记录中的得分簿</view>
      <view v-if="activeScorebooks.length === 0" class="hint">暂无记录中的得分簿</view>
      <view v-else class="list">
        <view class="item" v-for="it in activeScorebooks" :key="it.id" @click="openScorebook(it.id)">
          <view class="row">
            <text class="name">{{ it.name }}</text>
            <text class="status">记录中</text>
          </view>
          <view class="sub">
            <view class="sub-left">
              <text v-if="it.locationText">{{ it.locationText }}</text>
              <text v-if="it.startTime">{{ formatTime(it.startTime) }}</text>
            </view>
            <text class="sub-right">成员 {{ it.memberCount }}</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { createScorebook, joinByInviteCode, listMyScorebooks, reverseGeocode } from '../../utils/api'

const token = ref('')
const isMpWeixin = ref(false)
const locating = ref(false)
const currentLocationText = ref('')

// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const newName = ref('')
const inviteCode = ref('')
const myScorebooks = ref<any[]>([])
const activeScorebooks = computed(() => (myScorebooks.value || []).filter((it) => String(it?.status || '') !== 'ended'))

const scanIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7V5a1 1 0 0 1 1-1h2"/><path d="M17 4h2a1 1 0 0 1 1 1v2"/><path d="M20 17v2a1 1 0 0 1-1 1h-2"/><path d="M7 20H5a1 1 0 0 1-1-1v-2"/><path d="M8 12h8"/></svg>'

onShow(() => {
  token.value = (uni.getStorageSync('token') as string) || ''
  loadSavedLocation()
  loadMyScorebooks()
})

onLoad((q) => {
  const query = (q || {}) as any
  const code = normalizeCode(String(query.scene || query.code || ''))
  if (code) {
    uni.navigateTo({ url: `/pages/join/index?scene=${encodeURIComponent(code)}` })
  }
})

function goLogin() {
  uni.setStorageSync('scorehub.afterLogin', { to: 'home', ts: Date.now() })
  uni.switchTab({ url: '/pages/my/index' })
}

function loadSavedLocation() {
  const loc = (uni.getStorageSync('lastLocation') as any) || null
  if (loc?.text) {
    currentLocationText.value = String(loc.text)
  }
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

async function refreshLocation() {
  // #ifndef MP-WEIXIN
  return
  // #endif

  // #ifdef MP-WEIXIN
  locating.value = true
  try {
    const res = await new Promise<UniApp.GetLocationSuccess>((resolve, reject) => {
      uni.getLocation({
        type: 'gcj02',
        isHighAccuracy: true,
        success: resolve,
        fail: reject,
      } as any)
    })
    const fallback = `${Number(res.latitude).toFixed(4)},${Number(res.longitude).toFixed(4)}`
    let text = fallback
    try {
      const geo = await reverseGeocode(res.latitude, res.longitude)
      if (geo?.locationText) text = String(geo.locationText)
    } catch (e) {}
    currentLocationText.value = text
    uni.setStorageSync('lastLocation', { latitude: res.latitude, longitude: res.longitude, text, ts: Date.now() })
  } catch (e: any) {
    // 用户拒绝授权时给出可恢复提示
    uni.showToast({ title: e?.errMsg?.includes('auth') ? '定位权限未开启' : '获取位置失败', icon: 'none' })
  } finally {
    locating.value = false
  }
  // #endif
}

async function onScanInviteToInput() {
  // #ifndef MP-WEIXIN
  uni.showToast({ title: '请在微信小程序内使用', icon: 'none' })
  return
  // #endif

  // #ifdef MP-WEIXIN
  try {
    const res = await new Promise<any>((resolve, reject) => {
      uni.scanCode({ success: resolve, fail: reject })
    })
    const raw = String(res?.path || res?.result || '').trim()
    const code = normalizeCode(raw)
    if (!code) {
      uni.showToast({ title: '未识别到邀请码', icon: 'none' })
      return
    }
    inviteCode.value = code
  } catch (e: any) {
    uni.showToast({ title: e?.message || '扫码失败', icon: 'none' })
  }
  // #endif
}

function normalizeCode(v: string): string {
  const raw = decodeURIComponent(String(v || '')).trim()
  if (!raw) return ''
  if (/^[0-9A-Z]{6,12}$/.test(raw)) return raw
  const m = raw.match(/(?:^|[?&])code=([^&]+)/)
  if (m?.[1]) return decodeURIComponent(m[1]).trim()
  return raw
}

async function onCreate() {
  if (!token.value) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    goLogin()
    return
  }
  try {
    const res = await createScorebook({ name: newName.value.trim(), locationText: currentLocationText.value.trim() })
    uni.navigateTo({ url: `/pages/scorebook/detail?id=${res.scorebook.id}` })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '创建失败', icon: 'none' })
  }
}

async function onJoinByCode() {
  const code = inviteCode.value.trim()
  if (!code) return uni.showToast({ title: '请输入邀请码', icon: 'none' })
  if (!token.value) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    goLogin()
    return
  }
  try {
    const res = await joinByInviteCode(code, {})
    uni.navigateTo({ url: `/pages/scorebook/detail?id=${res.scorebookId}` })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加入失败', icon: 'none' })
  }
}

async function loadMyScorebooks() {
  if (!token.value) {
    myScorebooks.value = []
    return
  }
  try {
    const res = await listMyScorebooks()
    myScorebooks.value = res.items || []
  } catch (e: any) {
    // 首页不打扰用户流程，静默失败即可
  }
}

function openScorebook(id: string) {
  if (!id) return
  uni.navigateTo({ url: `/pages/scorebook/detail?id=${id}` })
}
</script>

<style scoped>
.page {
  padding: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}
.card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
}
.title {
  font-size: 32rpx;
  font-weight: 600;
  margin-bottom: 12rpx;
}
.hint {
  color: #666;
}
.input {
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 28rpx;
}
.btn {
  margin-top: 8rpx;
}
.loc-row {
  margin-top: 8rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}
.loc-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  padding: 0;
  border-radius: 18rpx;
  background: #f6f7fb;
  color: #111;
}
.loc-btn::after {
  border: none;
}
.loc-btn:active {
  opacity: 0.9;
}
.loc-logo {
  width: 28rpx;
  height: 28rpx;
  background: #111;
  border-radius: 14rpx 14rpx 14rpx 0;
  transform: rotate(-45deg);
  position: relative;
  flex: none;
}
.loc-logo-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 999rpx;
  background: #fff;
  position: absolute;
  left: 8rpx;
  top: 8rpx;
}
.loc-logo.loading .loc-logo-dot {
  animation: locDotPulse 0.7s ease-in-out infinite;
}
@keyframes locDotPulse {
  0%,
  100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(0.6);
    opacity: 0.6;
  }
}
.primary {
  background: #111;
  color: #fff;
}
.invite-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}
.invite-input {
  flex: 1;
  min-width: 0;
}
.scan-btn {
  width: 72rpx;
  height: 72rpx;
  padding: 0;
  border-radius: 18rpx;
  background: #f6f7fb;
  color: #111;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
}
.scan-btn::after {
  border: none;
}
.scan-icon {
  width: 36rpx;
  height: 36rpx;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
  margin-top: 12rpx;
}
.item {
  background: #fff;
  border-radius: 16rpx;
  padding: 20rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
}
.row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12rpx;
}
.name {
  font-size: 30rpx;
  font-weight: 600;
  flex: 1;
  min-width: 0;
  white-space: normal;
  word-break: break-all;
  line-height: 1.35;
}
.status {
  color: #666;
  font-size: 24rpx;
  flex: none;
  white-space: nowrap;
  margin-top: 4rpx;
}
.sub {
  margin-top: 8rpx;
  color: #666;
  font-size: 24rpx;
  display: flex;
  justify-content: space-between;
  gap: 12rpx;
}
.sub-left {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}
.sub-right {
  flex: none;
  white-space: nowrap;
}
</style>
