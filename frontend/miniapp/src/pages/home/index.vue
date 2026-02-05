<template>
  <view class="page">
    <view class="card">
      <view class="title">开始新的得分簿</view>
      <input class="input" v-model="newName" placeholder="名称（可空，默认 时间 + 位置）" />
      <view class="loc-row">
        <input class="input loc-input" v-model="locationText" placeholder="位置（可空）" />
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
        <!-- <button size="mini" class="loc-btn text" v-if="isMpWeixin" @click="chooseLocationFromMap" hover-class="none">地图</button> -->
        <button size="mini" class="loc-btn text" v-if="locationText" @click="clearLocation" hover-class="none">清空</button>
      </view>
      <view class="loc-history" v-if="locationHistory.length">
        <view class="loc-history-title">历史</view>
        <view class="loc-history-list">
          <view class="loc-chip" v-for="(h, i) in locationHistory" :key="`${h}-${i}`" @click="selectLocationHistory(h)">
            {{ h }}
          </view>
        </view>
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
      <button class="btn" v-if="token" @click="onJoinByCode">邀请码加入</button>
      <button class="btn" v-else @click="goLogin">邀请码加入</button>
    </view>

    <view class="card" v-if="token">
      <view class="title-row">
        <view class="title">记录中的得分簿</view>
        <view class="more" @click="openScorebookList">
          <image class="more-icon" :src="moreIcon" mode="aspectFit" />
        </view>
      </view>
      <view v-if="loadingScorebooks && activeScorebooks.length === 0" class="hint">加载中…</view>
      <view v-else-if="activeScorebooks.length === 0" class="hint">暂无记录中的得分簿</view>
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
const locationText = ref('')
const locationHistory = ref<string[]>([])

// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const newName = ref('')
const inviteCode = ref('')
const myScorebooks = ref<any[]>([])
const loadingScorebooks = ref(false)
const activeScorebooks = computed(() =>
  (myScorebooks.value || []).filter((it) => isScorebook(it) && String(it?.status || '') !== 'ended'),
)

const scanIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7V5a1 1 0 0 1 1-1h2"/><path d="M17 4h2a1 1 0 0 1 1 1v2"/><path d="M20 17v2a1 1 0 0 1-1 1h-2"/><path d="M7 20H5a1 1 0 0 1-1-1v-2"/><path d="M8 12h8"/></svg>'
const moreIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none"><circle cx="6" cy="12" r="2" fill="%23111"/><circle cx="12" cy="12" r="2" fill="%23111"/><circle cx="18" cy="12" r="2" fill="%23111"/></svg>'

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
    locationText.value = String(loc.text)
  }
  const history = (uni.getStorageSync('locationHistory') as any) || []
  if (Array.isArray(history)) {
    locationHistory.value = history.map((v) => String(v || '').trim()).filter((v) => v)
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
    locationText.value = text
  } catch (e: any) {
    // 用户拒绝授权时给出可恢复提示
    uni.showToast({ title: e?.errMsg?.includes('auth') ? '定位权限未开启' : '获取位置失败', icon: 'none' })
  } finally {
    locating.value = false
  }
  // #endif
}

function rememberLocation(text: string, coord?: { latitude?: number; longitude?: number }) {
  const t = String(text || '').trim()
  if (!t) return
  const next = [t, ...locationHistory.value.filter((v) => v !== t)].slice(0, 4)
  locationHistory.value = next
  uni.setStorageSync('locationHistory', next)
  if (coord?.latitude != null && coord?.longitude != null) {
    uni.setStorageSync('lastLocation', { latitude: coord.latitude, longitude: coord.longitude, text: t, ts: Date.now() })
  } else {
    uni.setStorageSync('lastLocation', { text: t, ts: Date.now() })
  }
}

function selectLocationHistory(text: string) {
  locationText.value = String(text || '').trim()
}

function clearLocation() {
  locationText.value = ''
}

async function chooseLocationFromMap() {
  // #ifndef MP-WEIXIN
  uni.showToast({ title: '请在微信小程序内使用', icon: 'none' })
  return
  // #endif

  // #ifdef MP-WEIXIN
  try {
    const res = await new Promise<UniApp.ChooseLocationSuccess>((resolve, reject) => {
      uni.chooseLocation({ success: resolve, fail: reject } as any)
    })
    const text = String(res?.name || res?.address || '').trim()
    if (!text) {
      uni.showToast({ title: '未获取到位置', icon: 'none' })
      return
    }
    locationText.value = text
  } catch (e: any) {
    if (String(e?.errMsg || '').includes('cancel')) return
    uni.showToast({ title: e?.message || '选取位置失败', icon: 'none' })
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
    const loc = locationText.value.trim()
    if (loc) rememberLocation(loc)
    const res = await createScorebook({ name: newName.value.trim(), locationText: loc })
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
    loadingScorebooks.value = false
    return
  }
  const showLoading = myScorebooks.value.length === 0
  loadingScorebooks.value = showLoading
  try {
    const res = await listMyScorebooks()
    myScorebooks.value = res.items || []
  } catch (e: any) {
    // 首页不打扰用户流程，静默失败即可
  } finally {
    if (showLoading) loadingScorebooks.value = false
  }
}

function openScorebook(id: string) {
  if (!id) return
  uni.navigateTo({ url: `/pages/scorebook/detail?id=${id}` })
}

function openScorebookList() {
  uni.navigateTo({ url: '/pages/scorebook/list' })
}

function isScorebook(item: any): boolean {
  const t = String(item?.bookType || 'scorebook').toLowerCase()
  return t === 'scorebook'
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
.title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  margin-bottom: 12rpx;
}
.title-row .title {
  margin-bottom: 0;
}
.more {
  width: 56rpx;
  height: 56rpx;
  border-radius: 999rpx;
  background: #f1f2f4;
  display: flex;
  align-items: center;
  justify-content: center;
}
.more:active {
  opacity: 0.8;
}
.more-icon {
  width: 28rpx;
  height: 28rpx;
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
  gap: 12rpx;
}
.loc-input {
  flex: 1;
  min-width: 0;
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
.loc-btn.text {
  width: auto;
  padding: 0 16rpx;
  font-size: 24rpx;
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
.loc-history {
  margin-top: 10rpx;
}
.loc-history-title {
  color: #666;
  font-size: 24rpx;
  margin-bottom: 6rpx;
}
.loc-history-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
}
.loc-chip {
  background: #f6f7fb;
  color: #333;
  font-size: 24rpx;
  padding: 6rpx 12rpx;
  border-radius: 999rpx;
}
.loc-chip:active {
  opacity: 0.85;
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
