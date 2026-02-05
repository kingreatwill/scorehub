<template>
  <view class="page">
    <view class="card">
      <!-- <view class="title">我的得分簿</view> -->
      <view class="hint" v-if="!token">登录后可查看你的得分簿</view>
      <button class="btn" v-if="!token" @click="goLogin">去登录</button>

      <template v-else>
        <view class="searchbar">
          <view class="search-input-wrap" :class="{ focused: searchFocused }">
            <image class="search-icon" :src="searchIcon" mode="aspectFit" />
            <input
              class="search-input"
              v-model="keyword"
              placeholder="搜索得分簿"
              confirm-type="search"
              @focus="onSearchFocus"
              @blur="onSearchBlur"
            />
            <view class="search-clear" v-if="keyword" @click="clearKeyword">×</view>
          </view>
        </view>
        <view class="hint" v-if="loading">加载中…</view>
        <view class="hint" v-else-if="items.length === 0">暂无得分簿</view>
        <view class="hint" v-else-if="filteredItems.length === 0">没有匹配结果</view>
        <view v-else class="list">
          <view class="item" v-for="it in filteredItems" :key="it.id" @click="open(it.id)">
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
      </template>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { listMyScorebooks } from '../../utils/api'

const token = ref('')
const items = ref<any[]>([])
const loading = ref(false)
const keyword = ref('')
const searchFocused = ref(false)
const searchIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="%23999" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>'

const filteredItems = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return items.value
  return items.value.filter((it) => {
    const name = String(it?.name || '').toLowerCase()
    const loc = String(it?.locationText || '').toLowerCase()
    return name.includes(q) || loc.includes(q)
  })
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
    const res = await listMyScorebooks()
    items.value = (res.items || []).filter(isScorebook)
  } catch (e: any) {
    if (items.value.length === 0) {
      uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
    }
  } finally {
    if (showLoading) loading.value = false
  }
}

function open(id: string) {
  uni.navigateTo({ url: `/pages/scorebook/detail?id=${id}` })
}

function goLogin() {
  uni.setStorageSync('scorehub.afterLogin', { url: '/pages/scorebook/list', ts: Date.now() })
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

function isScorebook(item: any): boolean {
  const t = String(item?.bookType || 'scorebook').toLowerCase()
  return t === 'scorebook'
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
