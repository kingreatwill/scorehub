<template>
  <view class="page">
    <view class="card" v-if="!token">
      <view class="title">登录</view>
      <view class="form">
	        <template v-if="isMpWeixin">
	          <form @submit="onWechatLoginSubmit">
	            <button class="avatar-wrapper" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
	              <image class="avatar" :src="avatarUrl || fallbackAvatar" mode="aspectFill" />
	              <view class="avatar-tip">{{ avatarUrl ? '点击更换头像' : '点击选择头像' }}</view>
	            </button>
	            <input class="input" name="nickname" type="nickname" v-model="nickname" placeholder="昵称（可修改）" />
	            <button class="btn primary" form-type="submit">微信登录</button>
	          </form>
	        </template>
        <template v-else>
          <input class="input" v-model="nickname" placeholder="昵称" />
          <input class="input" v-model="avatarUrl" placeholder="头像 URL（可选）" />
          <button class="btn" @click="onDevLogin">登录</button>
        </template>
      </view>
    </view>

    <view class="card" v-if="token">
      <view class="title">开始新的得分簿</view>
      <input class="input" v-model="newName" placeholder="名称（可空，默认 时间 + 位置）" />
      <view class="loc-row">
        <text class="hint">位置：{{ currentLocationText || '未获取' }}</text>
        <button size="mini" class="loc-btn" v-if="isMpWeixin" @click="refreshLocation" :disabled="locating" hover-class="none">
          <view class="loc-logo" :class="{ loading: locating }">
            <view class="loc-logo-dot" />
          </view>
        </button>
      </view>
      <button class="btn" @click="onCreate">开始</button>
    </view>


    <view class="card" v-if="token">
      <input class="input" v-model="inviteCode" placeholder="邀请码（例如 8 位码）" />
      <button class="btn" @click="onJoinByCode">邀请码加入</button>
    </view>


    <view class="card" v-if="token">
      <view v-if="isMpWeixin">
        <button class="btn" @click="onScanJoin">扫码加入</button>
      </view>
      <view v-else class="hint">仅微信小程序支持扫码</view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { createScorebook, devLogin, joinByInviteCode, reverseGeocode, updateMe, wechatLogin } from '../../utils/api'

const token = ref('')
const isMpWeixin = ref(false)
const locating = ref(false)
const currentLocationText = ref('')

// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const nickname = ref('')
const avatarUrl = ref('')

const fallbackAvatar =
  'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'

const newName = ref('')
const inviteCode = ref('')

onShow(() => {
  token.value = (uni.getStorageSync('token') as string) || ''
  loadSavedUser()
  loadSavedLocation()
})

onLoad((q) => {
  const query = (q || {}) as any
  const code = normalizeCode(String(query.scene || query.code || ''))
  if (code) {
    uni.navigateTo({ url: `/pages/join/index?scene=${encodeURIComponent(code)}` })
  }
})

async function onDevLogin() {
  try {
    const res = await devLogin(getOrCreateDevOpenid(), nickname.value.trim(), avatarUrl.value.trim())
    token.value = res.token
    refreshLocation()
    uni.showToast({ title: '登录成功', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '登录失败', icon: 'none' })
  }
}

function getOrCreateDevOpenid(): string {
  const key = 'devOpenid'
  const existing = String((uni.getStorageSync(key) as any) || '').trim()
  if (existing) return existing
  const created = `dev-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`
  uni.setStorageSync(key, created)
  return created
}

async function onChooseAvatar(e: any) {
  // #ifndef MP-WEIXIN
  return
  // #endif

  // #ifdef MP-WEIXIN
  const filePath = String(e?.detail?.avatarUrl || '').trim()
  if (!filePath) return

  try {
    const info = await new Promise<any>((resolve, reject) => {
      uni.getImageInfo({ src: filePath, success: resolve, fail: reject } as any)
    })
	    const t = String(info?.type || '').toLowerCase()
	    const mime = t ? `image/${t === 'jpg' ? 'jpeg' : t}` : 'image/jpeg'

	    const fs = (uni as any).getFileSystemManager?.()
	    if (!fs?.readFile) {
	      uni.showToast({ title: '头像处理失败', icon: 'none' })
	      return
	    }
    const base64 = await new Promise<string>((resolve, reject) => {
      fs.readFile({
        filePath,
        encoding: 'base64',
        success: (r: any) => resolve(String(r?.data || '')),
        fail: reject,
      })
    })
    if (!base64) return
    const dataUrl = `data:${mime};base64,${base64}`
    if (dataUrl.length > 800_000) {
      uni.showToast({ title: '图片太大，请换一张', icon: 'none' })
      return
	    }
	    avatarUrl.value = dataUrl
	  } catch (err: any) {
	    uni.showToast({ title: '头像处理失败', icon: 'none' })
	  }
  // #endif
}

async function onWechatLogin() {
  // #ifndef MP-WEIXIN
  uni.showToast({ title: '请在微信小程序内使用', icon: 'none' })
  return
  // #endif

  // #ifdef MP-WEIXIN
  try {
    const loginRes = await new Promise<UniApp.LoginRes>((resolve, reject) => {
      uni.login({
        success: resolve,
        fail: reject,
      })
    })
    if (!loginRes.code) {
      uni.showToast({ title: '获取登录 code 失败', icon: 'none' })
      return
    }

    const res = await wechatLogin(loginRes.code)
    token.value = res.token
    if (res?.user?.nickname && !nickname.value.trim()) nickname.value = String(res.user.nickname || '')
    if (res?.user?.avatarUrl && !avatarUrl.value.trim()) avatarUrl.value = String(res.user.avatarUrl || '')
    if (nickname.value.trim() || avatarUrl.value.trim()) {
      try {
        const u = await updateMe({ nickname: nickname.value.trim(), avatarUrl: avatarUrl.value.trim() })
        if (u?.user?.nickname) nickname.value = String(u.user.nickname)
        if (u?.user?.avatarUrl) avatarUrl.value = String(u.user.avatarUrl)
      } catch (e: any) {
        // 登录成功，资料同步失败不阻断
      }
    }
    refreshLocation()
    uni.showToast({ title: '登录成功', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || e?.errMsg || '微信登录失败', icon: 'none' })
  }
  // #endif
}

async function onWechatLoginSubmit(e: any) {
  const submitted = String(e?.detail?.value?.nickname || '').trim()
  if (submitted !== nickname.value.trim()) nickname.value = submitted
  await onWechatLogin()
}

function loadSavedUser() {
  const u = (uni.getStorageSync('user') as any) || null
  if (!u) return
  if (u.nickname && !nickname.value.trim()) nickname.value = String(u.nickname)
  if (u.avatarUrl && !avatarUrl.value.trim()) avatarUrl.value = String(u.avatarUrl)
}

function loadSavedLocation() {
  const loc = (uni.getStorageSync('lastLocation') as any) || null
  if (loc?.text) {
    currentLocationText.value = String(loc.text)
  }
}

async function refreshLocation() {
  // #ifndef MP-WEIXIN
  return
  // #endif

  // #ifdef MP-WEIXIN
  if (!token.value) return
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

async function onScanJoin() {
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
    uni.navigateTo({ url: `/pages/join/index?code=${encodeURIComponent(code)}` })
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
  try {
    const res = await joinByInviteCode(code, { nickname: nickname.value.trim(), avatarUrl: avatarUrl.value.trim() })
    uni.navigateTo({ url: `/pages/scorebook/detail?id=${res.scorebookId}` })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加入失败', icon: 'none' })
  }
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
.form {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
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
.avatar-wrapper {
  padding: 18rpx 16rpx;
  border-radius: 12rpx;
  background: #f6f7fb;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
}
.avatar-wrapper::after {
  border: none;
}
.avatar {
  width: 120rpx;
  height: 120rpx;
  border-radius: 60rpx;
  background: #fff;
}
.avatar-tip {
  color: #666;
  font-size: 24rpx;
}
.primary {
  background: #111;
  color: #fff;
}
</style>
