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
            <input class="input" name="nickname" type="nickname" v-model="nickname" placeholder="昵称（可选）" />
            <button class="btn primary" form-type="submit">微信登录</button>
          </form>
        </template>
        <template v-else>
          <input class="input" v-model="nickname" placeholder="昵称" />
          <input class="input" v-model="avatarUrl" placeholder="头像 URL（可选）" />
          <button class="btn primary" @click="onDevLogin">登录</button>
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

    <view class="card" v-if="token">
      <view class="title">我的得分簿</view>
      <view v-if="items.length === 0" class="hint">暂无得分簿</view>
      <view v-else class="list">
        <view class="item" v-for="it in items" :key="it.id" @click="open(it.id)">
          <view class="row">
            <text class="name">{{ it.name }}</text>
            <text class="status">{{ it.status === 'ended' ? '已结束' : '记录中' }}</text>
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

    <view class="modal-mask" v-if="editOpen" @click="closeEdit" />
    <view class="modal" v-if="editOpen">
      <view class="modal-head">
        <view class="modal-title">修改资料</view>
        <view class="modal-close" @click="closeEdit">×</view>
      </view>

      <view class="form">
        <template v-if="isMpWeixin">
          <button class="avatar-wrapper" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
            <image class="avatar" :src="avatarUrl || user?.avatarUrl || fallbackAvatar" mode="aspectFill" />
            <view class="avatar-tip">点击更换头像</view>
          </button>
          <input class="input" name="nickname" type="nickname" v-model="nickname" placeholder="昵称（可选）" />
        </template>
        <template v-else>
          <input class="input" v-model="nickname" placeholder="昵称" />
          <input class="input" v-model="avatarUrl" placeholder="头像 URL（可选）" />
        </template>
        <button class="btn primary" :disabled="editSubmitting" @click="saveProfile">保存</button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { devLogin, listMyScorebooks, updateMe, wechatLogin } from '../../utils/api'

const token = ref('')
const items = ref<any[]>([])
const user = ref<any>(null)

const isMpWeixin = ref(false)
// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const nickname = ref('')
const avatarUrl = ref('')
const editOpen = ref(false)
const editSubmitting = ref(false)

const fallbackAvatar = 'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'

onShow(async () => {
  token.value = (uni.getStorageSync('token') as string) || ''
  user.value = (uni.getStorageSync('user') as any) || null
  loadSavedUserDraft()

  if (!token.value) {
    items.value = []
    return
  }
  await loadMyScorebooks()
})

function loadSavedUserDraft() {
  const u = (uni.getStorageSync('user') as any) || null
  if (!u) return
  if (u.nickname && !nickname.value.trim()) nickname.value = String(u.nickname)
  if (u.avatarUrl && !avatarUrl.value.trim()) avatarUrl.value = String(u.avatarUrl)
}

async function loadMyScorebooks() {
  try {
    const res = await listMyScorebooks()
    items.value = res.items || []
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  }
}

function open(id: string) {
  uni.navigateTo({ url: `/pages/scorebook/detail?id=${id}` })
}

function openEdit() {
  if (!token.value) return
  nickname.value = String(user.value?.nickname || '').trim()
  avatarUrl.value = String(user.value?.avatarUrl || '').trim()
  editOpen.value = true
}

function closeEdit() {
  if (editSubmitting.value) return
  editOpen.value = false
}

async function saveProfile() {
  if (!token.value) return
  if (editSubmitting.value) return
  const nextNickname = nickname.value.trim()
  const nextAvatar = avatarUrl.value.trim()
  if (!nextNickname) {
    uni.showToast({ title: '请输入昵称', icon: 'none' })
    return
  }

  editSubmitting.value = true
  try {
    const res = await updateMe({ nickname: nextNickname, avatarUrl: nextAvatar })
    user.value = res.user
    avatarUrl.value = String(res?.user?.avatarUrl || nextAvatar)
    nickname.value = String(res?.user?.nickname || nextNickname)
    uni.showToast({ title: '已保存', icon: 'success' })
    editOpen.value = false
  } catch (e: any) {
    uni.showToast({ title: e?.message || '保存失败', icon: 'none' })
  } finally {
    editSubmitting.value = false
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

async function onDevLogin() {
  try {
    const res = await devLogin(getOrCreateDevOpenid(), nickname.value.trim(), avatarUrl.value.trim())
    token.value = res.token
    user.value = res.user
    await afterLoginRedirect()
    await loadMyScorebooks()
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
    const nextNickname = nickname.value.trim()
    const nextAvatar = avatarUrl.value.trim()
    if (nextNickname || nextAvatar) {
      try {
        const u = await updateMe({ nickname: nextNickname, avatarUrl: nextAvatar })
        user.value = u.user
        nickname.value = String(u?.user?.nickname || nickname.value)
        avatarUrl.value = String(u?.user?.avatarUrl || avatarUrl.value)
      } catch (e: any) {
        // 登录成功，资料同步失败不阻断
      }
    }

    await afterLoginRedirect()
    await loadMyScorebooks()
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
  items.value = []
  nickname.value = ''
  avatarUrl.value = ''
  editOpen.value = false
  uni.showToast({ title: '已退出', icon: 'success' })
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
.list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
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
.modal-mask {
  position: fixed;
  z-index: 1000;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.45);
}
.modal {
  position: fixed;
  z-index: 1001;
  left: 24rpx;
  right: 24rpx;
  top: 18%;
  background: #fff;
  border-radius: 18rpx;
  padding: 24rpx;
  box-shadow: 0 18rpx 48rpx rgba(0, 0, 0, 0.18);
}
.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}
.modal-title {
  font-size: 32rpx;
  font-weight: 600;
}
.modal-close {
  width: 64rpx;
  height: 64rpx;
  border-radius: 999rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
  font-size: 38rpx;
}
.modal-close:active {
  opacity: 0.85;
}
</style>
