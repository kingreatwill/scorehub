<template>
  <view class="page" :style="themeStyle">
    <view class="card" v-if="!token">
      <view class="title">登录</view>
      <view class="form">
        <template v-if="isMpWeixin">
          <form @submit="onWechatLoginSubmit">
            <button class="avatar-wrapper" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
              <image class="avatar" :src="avatarUrl || fallbackAvatar" mode="aspectFill" />
              <view class="avatar-tip">{{ avatarUrl ? '点击更换头像' : '点击选择头像（可选）' }}</view>
            </button>
            <input
              class="input"
              name="nickname"
              type="nickname"
              :value="nickname"
              placeholder="昵称（可选）"
              :controlled="true"
              :cursor="nicknameCursor"
              @input="onNicknameInput"
            />
            <button class="btn confirm-btn" form-type="submit" :disabled="wechatLoggingIn">
              {{ wechatLoggingIn ? '提交中…' : '微信登录' }}
            </button>
          </form>
        </template>
        <template v-else>
          <input class="input" :value="nickname" placeholder="昵称" :controlled="true" :cursor="nicknameCursor" @input="onNicknameInput" />
          <input class="input" v-model="avatarUrl" placeholder="头像 URL（可选）" />
          <button class="btn" @click="onDevLogin">登录</button>
        </template>
      </view>
    </view>

    <view class="card my-card" v-else>
      <view class="title-row">
        <view class="title">我的</view>
        <button class="logout-btn" @click="logout" hover-class="none">
          <image class="logout-icon" :src="logoutIcon" mode="aspectFit" />
        </button>
      </view>
      <view class="user-row" @click="openEdit">
        <image class="user-avatar" :src="user?.avatarUrl || fallbackAvatar" mode="aspectFill" />
        <view class="user-info">
          <view class="user-name">{{ user?.nickname || '未设置昵称' }}</view>
          <view class="user-sub">已登录</view>
        </view>
      </view>
    </view>

    <view class="card">
      <view class="title">我的功能</view>
      <view class="feature-grid">
        <view class="feature" @click="openScorebookList">
          <image class="feature-icon" :src="scorebookIcon" mode="aspectFit" />
          <view class="feature-label">得分簿</view>
        </view>
        <view class="feature" @click="openLedgerList">
          <image class="feature-icon" :src="ledgerIcon" mode="aspectFit" />
          <view class="feature-label">记账簿</view>
        </view>
      </view>
    </view>

    <view class="card">
      <view class="title">邀请码加入</view>
      <view class="invite-row">
        <input class="input invite-input" v-model="inviteCode" placeholder="邀请码（例如 8 位码）" />
        <button size="mini" class="scan-btn" v-if="isMpWeixin" @click="onScanInviteToInput" hover-class="none">
          <image class="scan-icon" :src="scanIcon" mode="aspectFit" />
        </button>
      </view>
      <button class="btn confirm-btn" :disabled="inviteJoining" @click="onJoinByCode">
        {{ inviteJoining ? '提交中…' : '加入' }}
      </button>
    </view>

    <button class="color-dot" :style="colorDotStyle" @tap.stop="onColorDotTap" hover-class="none" />
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { devLogin, getInviteInfo, joinByInviteCode, updateMe, wechatLogin } from '../../utils/api'
import { clampNickname } from '../../utils/nickname'
import {
  applyNavigationBarTheme,
  applyTabBarTheme,
  buildThemeVars,
  getThemeBaseColor,
  normalizeHexColor as normalizeThemeHex,
  saveThemeColor,
} from '../../utils/theme'

const token = ref('')
const user = ref<any>(null)

const isMpWeixin = ref(false)
// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const nickname = ref('')
const avatarUrl = ref('')
const nicknameCursor = ref(0)
const wechatLoggingIn = ref(false)
const inviteCode = ref('')
const inviteJoining = ref(false)
const colorDot = ref('#111111')
const themeStyle = computed(() => buildThemeVars(colorDot.value))
const colorDotStyle = computed(() => ({
  backgroundColor: toFillColor(colorDot.value),
  boxShadow: '0 10rpx 24rpx rgba(0, 0, 0, 0.16)',
}))
const presetDotColors = ['#111111', '#3B82F6', '#F59E0B', '#EF4444', '#8B5CF6']
const scorebookIcon = computed(() => iconDataUrl('scorebook', colorDot.value))
const ledgerIcon = computed(() => iconDataUrl('ledger', colorDot.value))
const scanIcon = computed(() => iconDataUrl('scan', colorDot.value))
const logoutIcon = computed(() => iconDataUrl('logout', colorDot.value))

const fallbackAvatar = 'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'

onShow(() => {
  token.value = (uni.getStorageSync('token') as string) || ''
  user.value = (uni.getStorageSync('user') as any) || null
  loadSavedUserDraft()
  loadColorDot()
  applyNavBarTheme()
  applyTabBarTheme(colorDot.value)
})

function onNicknameInput(e: any) {
  const next = clampNickname(String(e?.detail?.value || ''))
  nickname.value = next
  nicknameCursor.value = next.length
  return next
}

function loadSavedUserDraft() {
  const u = (uni.getStorageSync('user') as any) || null
  if (!u) return
  if (u.nickname && !nickname.value.trim()) nickname.value = clampNickname(String(u.nickname))
  if (u.avatarUrl && !avatarUrl.value.trim()) avatarUrl.value = String(u.avatarUrl)
}

function loadColorDot() {
  colorDot.value = getThemeBaseColor()
}

function normalizeHexColor(raw: string): string {
  return normalizeThemeHex(raw)
}

function toFillColor(hex: string): string {
  const normalized = normalizeHexColor(hex) || '#111111'
  const r = Number.parseInt(normalized.slice(1, 3), 16)
  const g = Number.parseInt(normalized.slice(3, 5), 16)
  const b = Number.parseInt(normalized.slice(5, 7), 16)
  return `rgba(${r}, ${g}, ${b}, 0.32)`
}

type IconKind = 'scorebook' | 'ledger' | 'scan' | 'logout'

function iconDataUrl(kind: IconKind, hex: string): string {
  const stroke = normalizeHexColor(hex) || '#111111'
  const svg = iconSvg(kind, stroke)
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
}

function iconSvg(kind: IconKind, stroke: string): string {
  if (kind === 'scorebook') {
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${stroke}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6a2 2 0 0 1 2-2h12v16H6a2 2 0 0 0-2 2z"/><path d="M8 4v16"/><path d="M11 8h5"/><path d="M11 12h5"/></svg>`
  }
  if (kind === 'ledger') {
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${stroke}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="6" width="18" height="12" rx="2"/><path d="M3 10h18"/><circle cx="16.5" cy="14" r="1"/></svg>`
  }
  if (kind === 'scan') {
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${stroke}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7V5a1 1 0 0 1 1-1h2"/><path d="M17 4h2a1 1 0 0 1 1 1v2"/><path d="M20 17v2a1 1 0 0 1-1 1h-2"/><path d="M7 20H5a1 1 0 0 1-1-1v-2"/><path d="M8 12h8"/></svg>`
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${stroke}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><path d="M10 17l5-5-5-5"/><path d="M15 12H3"/></svg>`
}

function applyColorDot(hex: string) {
  const normalized = normalizeHexColor(hex)
  if (!normalized) return false
  colorDot.value = saveThemeColor(normalized)
  applyNavBarTheme()
  applyTabBarTheme(colorDot.value)
  return true
}

function applyNavBarTheme() {
  applyNavigationBarTheme(colorDot.value)
}

async function onColorDotTap() {
  const labels = [...presetDotColors.map((v) => `颜色 ${v}`), '自定义 HEX']
  try {
    const tap = await new Promise<{ tapIndex: number }>((resolve, reject) => {
      uni.showActionSheet({ itemList: labels, success: resolve, fail: reject })
    })
    if (tap.tapIndex < presetDotColors.length) {
      applyColorDot(presetDotColors[tap.tapIndex])
      return
    }
    // #ifdef MP-WEIXIN
    const custom = await new Promise<{ confirm: boolean; content?: string }>((resolve) => {
      uni.showModal({
        title: '自定义颜色',
        editable: true,
        placeholderText: '#111111',
        content: colorDot.value,
        success: resolve,
        fail: () => resolve({ confirm: false }),
      } as any)
    })
    if (!custom.confirm) return
    if (!applyColorDot(String(custom.content || ''))) {
      uni.showToast({ title: '请输入正确 HEX 颜色', icon: 'none' })
    }
    // #endif
    // #ifndef MP-WEIXIN
    uni.showToast({ title: '请选择预设颜色', icon: 'none' })
    // #endif
  } catch (e: any) {
    if (String(e?.errMsg || '').includes('cancel')) return
    uni.showToast({ title: '打开颜色面板失败', icon: 'none' })
  }
}

function openLedgerList() {
  uni.navigateTo({ url: '/pages/ledger/list' })
}

function openScorebookList() {
  uni.navigateTo({ url: '/pages/scorebook/list' })
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
  try {
    const u = new URL(raw)
    const code = u.searchParams.get('code') || ''
    return decodeURIComponent(code).trim()
  } catch (e) {}
  const m = raw.match(/(?:^|[?&])code=([^&]+)/)
  if (m?.[1]) return decodeURIComponent(m[1]).trim()
  return raw
}

async function onJoinByCode() {
  const code = normalizeCode(inviteCode.value)
  if (!code) {
    uni.showToast({ title: '请输入邀请码', icon: 'none' })
    return
  }
  if (inviteJoining.value) return
  inviteJoining.value = true
  try {
    const res = await getInviteInfo(code)
    const invite = res?.invite
    if (!invite) {
      uni.showToast({ title: '邀请码无效', icon: 'none' })
      return
    }
    const bookType = String(invite.bookType || 'scorebook').toLowerCase()
    const bookId = String(invite.bookId || invite.scorebookId || invite.ledgerId || '').trim()
    if (!bookId) {
      uni.showToast({ title: '邀请码无效', icon: 'none' })
      return
    }
    if (String(invite.status || '') === 'ended') {
      const label = bookType === 'ledger' ? '记账簿' : '得分簿'
      uni.showToast({ title: `${label}已结束`, icon: 'none' })
      return
    }
    if (bookType === 'ledger') {
      uni.navigateTo({ url: `/pages/ledger/detail?id=${encodeURIComponent(bookId)}&bind=1` })
      return
    }
    if (!token.value) {
      uni.showToast({ title: '请先登录', icon: 'none' })
      return
    }
    const joined = await joinByInviteCode(code, {})
    uni.navigateTo({ url: `/pages/scorebook/detail?id=${joined.scorebookId}` })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '邀请码无效', icon: 'none' })
  } finally {
    inviteJoining.value = false
  }
}

function openEdit() {
  if (!token.value) return
  uni.navigateTo({ url: '/pages/profile/edit?mode=me' })
}

function getOrCreateDevOpenid(): string {
  const key = 'devOpenid'
  const existing = String((uni.getStorageSync(key) as any) || '').trim()
  if (existing) return existing
  const created = `dev-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`
  uni.setStorageSync(key, created)
  return created
}

async function onDevLogin() {
  try {
    const res = await devLogin(getOrCreateDevOpenid(), clampNickname(nickname.value.trim()), avatarUrl.value.trim())
    token.value = res.token
    user.value = res.user
    await afterLoginRedirect()
    uni.showToast({ title: '登录成功', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '登录失败', icon: 'none' })
  }
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
      uni.login({ success: resolve, fail: reject })
    })
    if (!loginRes.code) {
      uni.showToast({ title: '获取登录 code 失败', icon: 'none' })
      return
    }

    const res = await wechatLogin(loginRes.code)
    token.value = res.token
    user.value = res.user

    // 同步昵称/头像（可选）
    const nextNickname = clampNickname(nickname.value.trim())
    const nextAvatar = avatarUrl.value.trim()
    if (nextNickname || nextAvatar) {
      try {
        const u = await updateMe({ nickname: nextNickname, avatarUrl: nextAvatar })
        user.value = u.user
        nickname.value = clampNickname(String(u?.user?.nickname || nickname.value))
        avatarUrl.value = String(u?.user?.avatarUrl || avatarUrl.value)
      } catch (e: any) {
        // 登录成功，资料同步失败不阻断
      }
    }

    await afterLoginRedirect()
    uni.showToast({ title: '登录成功', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || e?.errMsg || '微信登录失败', icon: 'none' })
  }
  // #endif
}

async function onWechatLoginSubmit(e: any) {
  if (wechatLoggingIn.value) return
  wechatLoggingIn.value = true
  const submitted = clampNickname(String(e?.detail?.value?.nickname || '').trim())
  if (submitted !== nickname.value.trim()) nickname.value = submitted
  try {
    await onWechatLogin()
  } finally {
    wechatLoggingIn.value = false
  }
}

async function afterLoginRedirect() {
  const key = 'scorehub.afterLogin'
  const v = (uni.getStorageSync(key) as any) || null
  if (!v) return
  uni.removeStorageSync(key)

  if (v?.to === 'home') {
    uni.switchTab({ url: '/pages/home/index' })
    return
  }
  if (typeof v?.url === 'string' && v.url) {
    uni.navigateTo({ url: String(v.url) })
  }
}

function logout() {
  uni.removeStorageSync('token')
  uni.removeStorageSync('user')
  token.value = ''
  user.value = null
  nickname.value = ''
  avatarUrl.value = ''
  uni.showToast({ title: '已退出', icon: 'success' })
}
</script>

<style scoped>
.page {
  padding: 24rpx;
  padding-bottom: calc(150rpx + env(safe-area-inset-bottom));
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
.title-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 12rpx;
  margin-bottom: 12rpx;
  position: relative;
}
.hint {
  color: #666;
  font-size: 26rpx;
}
.form {
  margin-top: 16rpx;
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
.invite-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-top: 12rpx;
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
.btn {
  margin-top: 8rpx;
}
.primary {
  background: #111;
  color: #fff;
}
.avatar-wrapper {
  margin-bottom: 12rpx;
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
.user-row {
  display: flex;
  align-items: center;
}
.user-row:active {
  opacity: 0.9;
}
.my-card {
  position: relative;
}
.logout-btn {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 44rpx;
  height: 44rpx;
  padding: 0;
  border-radius: 999rpx;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
}
.logout-btn::after {
  border: none;
}
.logout-icon {
  width: 28rpx;
  height: 28rpx;
}
.user-avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 44rpx;
  background: #fff;
  flex: none;
}
.user-info {
  margin-left: 16rpx;
  min-width: 0;
  flex: 1;
}
.user-name {
  font-size: 30rpx;
  font-weight: 600;
  line-height: 1.2;
}
.user-sub {
  margin-top: 6rpx;
  color: #666;
  font-size: 24rpx;
}
.feature-grid {
  margin-top: 12rpx;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}
.feature {
  padding: 18rpx 12rpx;
  border-radius: 16rpx;
  background: #f6f7fb;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
}
.feature:active {
  opacity: 0.85;
}
.feature-icon {
  width: 56rpx;
  height: 56rpx;
  flex: none;
}
.feature-label {
  font-size: 28rpx;
  font-weight: 600;
  color: #111;
  text-align: center;
}
.color-dot {
  position: fixed;
  right: 24rpx;
  bottom: calc(138rpx + env(safe-area-inset-bottom));
  width: 70rpx;
  height: 70rpx;
  min-width: 70rpx;
  min-height: 70rpx;
  padding: 0;
  border-radius: 999rpx;
  border: 1rpx solid rgba(255, 255, 255, 0.24);
  z-index: 1201;
}
.color-dot::after {
  border: none;
}
</style>
