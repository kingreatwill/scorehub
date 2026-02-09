<template>
  <view class="page">
    <view class="card">
      <view class="title">开始新的记账簿</view>
      <input class="input" v-model="newName" placeholder="名称（可空，默认 时间）" />
      <button class="btn" @click="onCreate">开始</button>
    </view>

    <view class="card">
      <view class="title-row">
        <view class="title">记录中的记账簿</view>
        <view class="more" @click="openList">
          <image class="more-icon" :src="moreIcon" mode="aspectFit" />
        </view>
      </view>
      <view v-if="!token" class="hint">登录后可查看记账簿</view>
      <button class="btn" v-if="!token" @click="goLogin">去登录</button>
      <view v-else-if="loading" class="hint">加载中…</view>
      <view v-else-if="activeLedgers.length === 0" class="hint">暂无记录中的记账簿</view>
      <view v-else class="list">
        <view class="item" v-for="it in activeLedgers" :key="it.id" @click="openLedger(it.id)">
          <view class="row">
            <text class="name">{{ it.name }}</text>
            <text class="status">{{ it.status === 'ended' ? '已结束' : '记录中' }}</text>
          </view>
          <view class="sub">
            <view class="sub-left">
              <text v-if="it.createdAt">{{ formatTime(it.createdAt) }}</text>
              <text>记录 {{ it.recordCount || 0 }}</text>
            </view>
            <text class="sub-right">成员 {{ it.memberCount || 0 }}</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { createLedger, listLedgers } from '../../utils/api'

const newName = ref('')
const ledgers = ref<any[]>([])
const token = ref('')
const loading = ref(false)
const moreIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none"><circle cx="6" cy="12" r="2" fill="%23111"/><circle cx="12" cy="12" r="2" fill="%23111"/><circle cx="18" cy="12" r="2" fill="%23111"/></svg>'

const activeLedgers = computed(() => (ledgers.value || []).filter((it) => it.status !== 'ended'))

onShow(() => {
  loadLedgers()
})

async function loadLedgers() {
  token.value = (uni.getStorageSync('token') as string) || ''
  if (!token.value) {
    ledgers.value = []
    loading.value = false
    return
  }
  const showLoading = ledgers.value.length === 0
  loading.value = showLoading
  try {
    const res = await listLedgers()
    ledgers.value = res.items || []
  } catch (e: any) {
    if (ledgers.value.length === 0) {
      uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
    }
  } finally {
    if (showLoading) loading.value = false
  }
}

async function onCreate() {
  if (!token.value) {
    goLogin()
    return
  }
  const name = newName.value.trim() || defaultLedgerName()
  try {
    const res = await createLedger({ name })
    newName.value = ''
    uni.navigateTo({ url: `/pages/ledger/detail?id=${res.ledger.id}` })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '创建失败', icon: 'none' })
  }
}

function openLedger(id: string) {
  uni.navigateTo({ url: `/pages/ledger/detail?id=${id}` })
}

function openList() {
  uni.navigateTo({ url: '/pages/ledger/list' })
}

function goLogin() {
  uni.setStorageSync('scorehub.afterLogin', { url: '/pages/ledger/index', ts: Date.now() })
  uni.switchTab({ url: '/pages/my/index' })
}

function defaultLedgerName(): string {
  const d = new Date()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${mi} 记账`
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
.title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  margin-bottom: 12rpx;
}
.title {
  font-size: 32rpx;
  font-weight: 600;
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
