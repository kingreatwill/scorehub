<template>
  <view class="page">
    <view class="card">
      <view class="title">加入得分簿</view>
      <view class="sub" v-if="inviteCode">邀请码：{{ inviteCode }}</view>
      <view class="sub" v-if="invite?.name">名称：{{ invite.name }}</view>
      <view class="sub" v-if="invite?.status">
        状态：{{ invite.status === 'ended' ? '已结束' : '记录中' }}
      </view>
    </view>

    <view class="card" v-if="!token">
      <view class="title">需要登录</view>
      <view class="hint">登录后可直接加入。</view>
      <!-- #ifdef MP-WEIXIN -->
      <form @submit="onWechatLoginSubmit">
        <button class="avatar-wrapper" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
          <image class="avatar" :src="avatarUrl || fallbackAvatar" mode="aspectFill" />
          <view class="avatar-tip">{{ avatarUrl ? '点击更换头像' : '点击选择头像' }}</view>
        </button>
        <input class="input" name="nickname" type="nickname" v-model="nickname" placeholder="昵称（可修改）" />
        <button class="btn primary" form-type="submit">微信登录</button>
      </form>
      <!-- #endif -->
      <button class="btn" @click="goHome">去首页登录</button>
    </view>

    <view class="card" v-else>
      <view class="hint" v-if="loading">加载中…</view>
      <view class="hint" v-else-if="!invite">邀请码无效或无权限。</view>
      <view class="hint" v-else-if="invite?.status === 'ended'">该得分簿已结束，无法加入。</view>
      <view class="hint" v-else-if="joining">正在加入…</view>
      <view class="hint" v-else>准备就绪</view>

      <button class="btn primary" :disabled="!canJoin" @click="join">加入并进入</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { getInviteInfo, joinByInviteCode, updateMe, wechatLogin } from '../../utils/api'

const inviteCode = ref('')
const token = ref('')
const invite = ref<any>(null)
const loading = ref(false)
const joining = ref(false)
const nickname = ref('')
const avatarUrl = ref('')

const fallbackAvatar =
  'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'

const canJoin = computed(() => {
  if (!token.value) return false
  if (!inviteCode.value) return false
  if (!invite.value) return false
  if (loading.value || joining.value) return false
  if (invite.value?.status !== 'recording') return false
  return true
})

onLoad((q) => {
  const query = q as any
  inviteCode.value = normalizeCode(query.code || query.scene || '')
})

onShow(async () => {
  token.value = (uni.getStorageSync('token') as string) || ''
  loadSavedUser()
  if (!token.value || !inviteCode.value) return
  await loadInvite()
  // 点击分享后打开：有 token 时自动尝试加入
  if (invite.value?.status === 'recording') {
    join()
  }
})

function loadSavedUser() {
  const u = (uni.getStorageSync('user') as any) || null
  if (!u) return
  if (u.nickname && !nickname.value.trim()) nickname.value = String(u.nickname)
  if (u.avatarUrl && !avatarUrl.value.trim()) avatarUrl.value = String(u.avatarUrl)
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

function normalizeCode(v: any): string {
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

async function loadInvite() {
  loading.value = true
  try {
    const res = await getInviteInfo(inviteCode.value)
    invite.value = res.invite
  } catch (e: any) {
    invite.value = null
    uni.showToast({ title: e?.message || '邀请码无效', icon: 'none' })
  } finally {
    loading.value = false
  }
}

async function join() {
  if (!canJoin.value) return
  joining.value = true
  try {
    const res = await joinByInviteCode(inviteCode.value, {
      nickname: nickname.value.trim(),
      avatarUrl: avatarUrl.value.trim(),
    })
    uni.redirectTo({ url: `/pages/scorebook/detail?id=${res.scorebookId}` })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加入失败', icon: 'none' })
  } finally {
    joining.value = false
  }
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
    const r = await wechatLogin(loginRes.code)
    token.value = r.token
    if (r?.user?.nickname && !nickname.value.trim()) nickname.value = String(r.user.nickname || '')
    if (r?.user?.avatarUrl && !avatarUrl.value.trim()) avatarUrl.value = String(r.user.avatarUrl || '')
    if (nickname.value.trim() || avatarUrl.value.trim()) {
      try {
        const u = await updateMe({ nickname: nickname.value.trim(), avatarUrl: avatarUrl.value.trim() })
        if (u?.user?.nickname) nickname.value = String(u.user.nickname)
        if (u?.user?.avatarUrl) avatarUrl.value = String(u.user.avatarUrl)
      } catch (e: any) {
        // 登录成功，资料同步失败不阻断
      }
    }
    await loadInvite()
    if (invite.value?.status === 'recording') join()
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

function goHome() {
  uni.switchTab({ url: '/pages/home/index' })
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
.sub {
  color: #333;
  font-size: 26rpx;
  margin-top: 6rpx;
}
.hint {
  color: #666;
  font-size: 26rpx;
}
.input {
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 28rpx;
  margin-top: 16rpx;
}
.avatar-wrapper {
  margin-top: 16rpx;
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
.btn {
  margin-top: 16rpx;
}
.primary {
  background: #111;
  color: #fff;
}
</style>
