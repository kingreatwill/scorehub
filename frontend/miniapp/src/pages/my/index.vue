<template>
  <view class="page">
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
            <button class="btn" form-type="submit">微信登录</button>
          </form>
        </template>
        <template v-else>
          <input class="input" :value="nickname" placeholder="昵称" :controlled="true" :cursor="nicknameCursor" @input="onNicknameInput" />
          <input class="input" v-model="avatarUrl" placeholder="头像 URL（可选）" />
          <button class="btn" @click="onDevLogin">登录</button>
        </template>
      </view>
    </view>

    <view class="card" v-else>
      <view class="title">我的</view>
      <view class="user-row" @click="openEdit">
        <image class="user-avatar" :src="user?.avatarUrl || fallbackAvatar" mode="aspectFill" />
        <view class="user-info">
          <view class="user-name">{{ user?.nickname || '未设置昵称' }}</view>
          <view class="user-sub">已登录</view>
        </view>
      </view>
      <button class="btn" @click="logout">退出登录</button>
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
      <button class="btn" :disabled="inviteJoining" @click="onJoinByCode">
        {{ inviteJoining ? '处理中…' : '邀请码加入' }}
      </button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { devLogin, getInviteInfo, joinByInviteCode, updateMe, wechatLogin } from '../../utils/api'
import { clampNickname } from '../../utils/nickname'

const token = ref('')
const user = ref<any>(null)

const isMpWeixin = ref(false)
// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const nickname = ref('')
const avatarUrl = ref('')
const nicknameCursor = ref(0)
const inviteCode = ref('')
const inviteJoining = ref(false)
const ledgerIcon = '/static/tabbar/ledger.png'
const scorebookIcon = '/static/tabbar/scorebook.png'
const scanIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7V5a1 1 0 0 1 1-1h2"/><path d="M17 4h2a1 1 0 0 1 1 1v2"/><path d="M20 17v2a1 1 0 0 1-1 1h-2"/><path d="M7 20H5a1 1 0 0 1-1-1v-2"/><path d="M8 12h8"/></svg>'

const fallbackAvatar = 'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'

onShow(() => {
  token.value = (uni.getStorageSync('token') as string) || ''
  user.value = (uni.getStorageSync('user') as any) || null
  loadSavedUserDraft()
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
  const submitted = clampNickname(String(e?.detail?.value?.nickname || '').trim())
  if (submitted !== nickname.value.trim()) nickname.value = submitted
  await onWechatLogin()
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
  gap: 16rpx;
}
.user-row:active {
  opacity: 0.9;
}
.user-avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 44rpx;
  background: #fff;
  flex: none;
}
.user-info {
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
</style>
