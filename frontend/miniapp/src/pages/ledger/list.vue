<template>
  <view class="page">
    <view class="card">
      <!-- <view class="title">我的记账簿</view> -->
      <view class="hint" v-if="!token">登录后可查看记账簿</view>
      <button class="btn" v-if="!token" @click="goLogin">去登录</button>
      <template v-else>
        <view class="searchbar">
          <view class="search-input-wrap" :class="{ focused: searchFocused }">
            <image class="search-icon" :src="searchIcon" mode="aspectFit" />
            <input
              class="search-input"
              v-model="keyword"
              placeholder="搜索记账簿"
              confirm-type="search"
              @focus="onSearchFocus"
              @blur="onSearchBlur"
            />
            <view class="search-clear" v-if="keyword" @click="clearKeyword">×</view>
          </view>
        </view>
        <view class="hint" v-if="loading">加载中…</view>
        <view class="hint" v-else-if="items.length === 0">暂无记账簿</view>
        <view class="hint" v-else-if="filteredItems.length === 0">没有匹配结果</view>
        <view v-else class="list">
          <view class="swipe-item" :class="{ open: openId === it.id }" v-for="it in filteredItems" :key="it.id">
            <view class="swipe-actions">
              <button class="swipe-btn" :class="{ disabled: !canDelete(it) }" @click.stop="confirmDelete(it)">删除</button>
            </view>
            <view
              class="item swipe-main"
              :class="{ open: openId === it.id }"
              @touchstart="onTouchStart($event, it.id)"
              @touchend="onTouchEnd($event, it.id)"
              @click="onItemTap(it)"
            >
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
      </template>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { deleteLedger, listLedgers } from '../../utils/api'

const items = ref<any[]>([])
const token = ref('')
const loading = ref(false)
const keyword = ref('')
const searchFocused = ref(false)
const openId = ref('')
const touchStartX = ref(0)
const touchStartY = ref(0)
const touchItemId = ref('')
const swipeJustFinished = ref(false)
const searchIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="%23999" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>'

const filteredItems = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return items.value
  return items.value.filter((it) => String(it?.name || '').toLowerCase().includes(q))
})

onShow(() => {
  load()
})

async function load() {
  token.value = (uni.getStorageSync('token') as string) || ''
  if (!token.value) {
    items.value = []
    loading.value = false
    return
  }
  const showLoading = items.value.length === 0
  loading.value = showLoading
  try {
    const res = await listLedgers()
    items.value = res.items || []
    openId.value = ''
  } catch (e: any) {
    if (items.value.length === 0) {
      uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
    }
  } finally {
    if (showLoading) loading.value = false
  }
}

function openLedger(id: string) {
  uni.navigateTo({ url: `/pages/ledger/detail?id=${id}` })
}

function onItemTap(it: any) {
  if (swipeJustFinished.value) return
  if (openId.value === it.id) {
    openId.value = ''
    return
  }
  openLedger(it.id)
}

function onTouchStart(e: any, id: string) {
  const t = e?.touches?.[0]
  if (!t) return
  touchStartX.value = t.clientX
  touchStartY.value = t.clientY
  touchItemId.value = id
}

function onTouchEnd(e: any, id: string) {
  if (touchItemId.value !== id) return
  const t = e?.changedTouches?.[0]
  if (!t) {
    touchItemId.value = ''
    return
  }
  const dx = t.clientX - touchStartX.value
  const dy = t.clientY - touchStartY.value
  if (Math.abs(dx) < 30 || Math.abs(dx) < Math.abs(dy)) {
    touchItemId.value = ''
    return
  }
  if (dx < 0) {
    openId.value = id
  } else {
    openId.value = ''
  }
  swipeJustFinished.value = true
  setTimeout(() => {
    swipeJustFinished.value = false
  }, 200)
  touchItemId.value = ''
}

function canDelete(it: any): boolean {
  return String(it?.status || '') === 'ended'
}

async function confirmDelete(it: any) {
  if (!canDelete(it)) {
    uni.showToast({ title: '请先结束', icon: 'none' })
    return
  }
  const res = await new Promise<UniApp.ShowModalRes>((resolve) => {
    uni.showModal({ title: '确认删除', content: `确定删除「${it.name}」？`, success: resolve })
  })
  if (!res.confirm) return
  try {
    await deleteLedger(String(it.id))
    items.value = items.value.filter((x) => x.id !== it.id)
    openId.value = ''
  } catch (e: any) {
    uni.showToast({ title: e?.message || '删除失败', icon: 'none' })
  }
}

function goLogin() {
  uni.setStorageSync('scorehub.afterLogin', { url: '/pages/ledger/list', ts: Date.now() })
  uni.switchTab({ url: '/pages/my/index' })
}

function clearKeyword() {
  keyword.value = ''
}

function onSearchFocus() {
  searchFocused.value = true
}

function onSearchBlur() {
  searchFocused.value = false
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
.btn {
  margin-top: 12rpx;
}
.searchbar {
  margin-top: 8rpx;
}
.search-input-wrap {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 0 20rpx;
  height: 76rpx;
  border-radius: 999rpx;
  border: 2rpx solid #e2e2e2;
  background: #f7f8fa;
  box-shadow: inset 0 2rpx 6rpx rgba(0, 0, 0, 0.04);
  transition: all 0.2s ease;
}
.search-input-wrap.focused {
  border-color: #111;
  background: #fff;
  box-shadow: 0 0 0 4rpx rgba(0, 0, 0, 0.08);
}
.search-icon {
  width: 32rpx;
  height: 32rpx;
  flex: none;
  opacity: 0.7;
}
.search-input {
  flex: 1;
  min-width: 0;
  font-size: 28rpx;
}
.search-clear {
  width: 40rpx;
  height: 40rpx;
  border-radius: 999rpx;
  background: #fff;
  border: 1rpx solid #e1e1e1;
  color: #666;
  font-size: 26rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}
.search-clear:active {
  opacity: 0.8;
}
.list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
  margin-top: 12rpx;
}
.swipe-item {
  position: relative;
  overflow: hidden;
  border-radius: 16rpx;
  background: #fff;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
}
.swipe-actions {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 140rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f3f4f6;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease;
}
.swipe-item.open .swipe-actions {
  opacity: 1;
  pointer-events: auto;
}
.swipe-btn {
  width: 100%;
  height: 100%;
  padding: 0;
  border-radius: 0;
  background: #ef4444;
  color: #fff;
  font-size: 26rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  border: 0;
}
.swipe-btn.disabled {
  background: #c7c7c7;
}
.swipe-main {
  position: relative;
  z-index: 1;
  transform: translateX(0);
  transition: transform 0.2s ease;
}
.swipe-main.open {
  transform: translateX(-140rpx);
}
.item {
  background: transparent;
  border-radius: 0;
  padding: 20rpx;
  box-shadow: none;
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
