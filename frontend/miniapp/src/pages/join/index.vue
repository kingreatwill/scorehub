<template>
  <view class="page">
    <view class="card">
      <view class="title">加入{{ bookLabel }}</view>
      <view class="hint" v-if="loading">加载中…</view>
      <template v-else>
        <view class="sub" v-if="inviteCode">邀请码：{{ inviteCode }}</view>
        <view class="sub" v-if="invite?.name">名称：{{ invite.name }}</view>
        <view class="sub" v-if="invite?.status">状态：{{ invite.status === 'ended' ? '已结束' : '记录中' }}</view>
        <view class="hint" v-if="inviteCode && !invite">邀请码无效或无权限。</view>
      </template>
    </view>

    <view class="card">
      <view class="title">操作</view>
      <template v-if="!needsLogin">
        <view class="hint" v-if="inviteCode && !invite">邀请码无效或无权限。</view>
        <view class="hint" v-else-if="invite?.status === 'ended'">该{{ bookLabel }}已结束，无法加入。</view>
        <view class="hint" v-else-if="joining">正在加入…</view>
        <view class="hint" v-else>准备就绪</view>
        <button class="btn confirm-btn" :disabled="!canJoin" @click="join">{{ actionText }}</button>
      </template>
      <template v-else>
        <view class="hint">未登录：登录后可加入并记分。</view>
        <button class="btn" @click="goLogin">去「我的」登录</button>
      </template>
    </view>
  </view>
</template>

<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { getInviteInfo, joinByInviteCode } from '../../utils/api'

const inviteCode = ref('')
const token = ref('')
const invite = ref<any>(null)
const loading = ref(false)
const joining = ref(false)

const bookType = computed(() => String(invite.value?.bookType || 'scorebook').toLowerCase())
const isLedger = computed(() => bookType.value === 'ledger')
const bookLabel = computed(() => (isLedger.value ? '记账簿' : '得分簿'))
const needsLogin = computed(() => !isLedger.value && !token.value)
const actionText = computed(() => (isLedger.value ? '进入记账簿' : '加入并进入'))

const canJoin = computed(() => {
  if (!inviteCode.value) return false
  if (!invite.value) return false
  if (loading.value || joining.value) return false
  if (invite.value?.status !== 'recording') return false
  if (!isLedger.value && !token.value) return false
  return true
})

onLoad((q) => {
  const query = q as any
  inviteCode.value = normalizeCode(query.code || query.scene || '')
})

onShow(async () => {
  token.value = (uni.getStorageSync('token') as string) || ''
  if (!inviteCode.value) return
  await loadInvite()
})

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
    if (isLedger.value) {
      const ledgerId = String(invite.value?.bookId || invite.value?.ledgerId || '').trim()
      if (!ledgerId) {
        uni.showToast({ title: '邀请码无效', icon: 'none' })
        return
      }
      uni.redirectTo({ url: `/pages/ledger/detail?id=${encodeURIComponent(ledgerId)}&bind=1` })
      return
    }
    if (!token.value) {
      uni.showToast({ title: '请先登录', icon: 'none' })
      return
    }
    const res = await joinByInviteCode(inviteCode.value, {})
    uni.redirectTo({ url: `/pages/scorebook/detail?id=${res.scorebookId}` })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加入失败', icon: 'none' })
  } finally {
    joining.value = false
  }
}

function goLogin() {
  if (inviteCode.value) {
    uni.setStorageSync('scorehub.afterLogin', {
      to: 'join',
      url: `/pages/join/index?code=${encodeURIComponent(inviteCode.value)}`,
      ts: Date.now(),
    })
  }
  uni.switchTab({ url: '/pages/my/index' })
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
.btn {
  margin-top: 16rpx;
}
.primary {
  background: #111;
  color: #fff;
}
</style>
