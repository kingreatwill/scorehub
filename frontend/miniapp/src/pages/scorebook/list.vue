<template>
  <view class="page" :style="themeStyle">
    <view class="card">
      <!-- <view class="title">我的得分簿</view> -->
      <view class="hint" v-if="!token">登录后可查看得分簿</view>
      <button class="btn confirm-btn" v-if="!token" @click="goLogin">去登录</button>

      <template v-else>
        <view class="searchbar">
          <view class="search-input-wrap" :class="{ focused: searchFocused }">
            <image class="search-icon" :src="searchIcon" mode="aspectFit" />
            <input
              class="search-input"
              v-model="keyword"
              placeholder="搜索得分簿"
              placeholder-class="search-placeholder"
              confirm-type="search"
              @focus="onSearchFocus"
              @blur="onSearchBlur"
            />
            <view class="search-clear" v-if="keyword" @click="clearKeyword">×</view>
          </view>
        </view>
        <view class="list-loading" v-if="loading">
          <t-loading :loading="true" text="加载中…" />
        </view>
        <view class="hint" v-else-if="loadError">加载失败</view>
        <view class="hint" v-else-if="items.length === 0">暂无得分簿</view>
        <view class="hint" v-else-if="filteredItems.length === 0">没有匹配结果</view>
        <view v-else class="list">
          <view class="swipe-item" :class="{ open: isSwiped(it.id) }" v-for="it in filteredItems" :key="it.id">
            <view class="swipe-actions" :class="{ dragging: isDragging && touchItemId === it.id }" :style="swipeActionStyle(it.id)">
              <button class="swipe-btn" :class="{ disabled: !canDelete(it) }" @click.stop="confirmDelete(it)">删除</button>
            </view>
            <view
              class="item swipe-main"
              :class="{ dragging: isDragging && touchItemId === it.id }"
              :style="swipeMainStyle(it.id)"
              @touchstart="onTouchStart($event, it.id)"
              @touchmove.stop.prevent="onTouchMove($event, it.id)"
              @touchend="onTouchEnd($event, it.id)"
              @click="onItemTap(it)"
            >
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
      </template>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { deleteScorebook, listMyScorebooks } from '../../utils/api'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor } from '../../utils/theme'

const token = ref('')
const items = ref<any[]>([])
const loading = ref(false)
const loadError = ref('')
const keyword = ref('')
const searchFocused = ref(false)
const openId = ref('')
const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))
const touchStartX = ref(0)
const touchStartY = ref(0)
const touchItemId = ref('')
const touchLastX = ref(0)
const touchDx = ref(0)
const dragStartOffset = ref(0)
const isDragging = ref(false)
const swipeJustFinished = ref(false)
const swipeOffsetById = ref<Record<string, number>>({})
const SWIPE_ACTION_WIDTH = 140
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
  syncTheme()
  load()
})

function syncTheme() {
  const base = getThemeBaseColor()
  themeStyle.value = buildThemeVars(base)
  applyNavigationBarTheme(base)
  applyTabBarTheme(base)
}

async function load() {
  token.value = (uni.getStorageSync('token') as string) || ''
  loadError.value = ''
  if (!token.value) {
    items.value = []
    loading.value = false
    return
  }
  const showLoading = items.value.length === 0
  loading.value = showLoading
  try {
    const res = await listMyScorebooks()
    items.value = sortScorebooks((res.items || []).filter(isScorebook))
    openId.value = ''
    swipeOffsetById.value = {}
  } catch (e: any) {
    if (items.value.length === 0) {
      loadError.value = '加载失败'
    }
  } finally {
    if (showLoading) loading.value = false
  }
}

function open(id: string) {
  uni.navigateTo({ url: `/pages/scorebook/detail?id=${id}` })
}

function onItemTap(it: any) {
  if (swipeJustFinished.value) return
  if (openId.value === it.id) {
    openId.value = ''
    return
  }
  open(it.id)
}

function onTouchStart(e: any, id: string) {
  const t = e?.touches?.[0]
  if (!t) return
  if (openId.value && openId.value !== id) setSwipeOffset(openId.value, 0)
  touchStartX.value = t.clientX
  touchLastX.value = t.clientX
  touchDx.value = 0
  touchStartY.value = t.clientY
  touchItemId.value = id
  dragStartOffset.value = getSwipeOffset(id)
  isDragging.value = false
}

function onTouchMove(e: any, id: string) {
  if (touchItemId.value !== id) return
  const t = e?.touches?.[0]
  if (!t) return
  const dx = t.clientX - touchStartX.value
  const dy = t.clientY - touchStartY.value
  touchLastX.value = t.clientX
  touchDx.value = dx
  if (!isDragging.value) {
    if (Math.abs(dx) < 6) return
    if (Math.abs(dx) <= Math.abs(dy)) {
      touchItemId.value = ''
      return
    }
    isDragging.value = true
  }
  const next = clampSwipeOffset(dragStartOffset.value + dx)
  setSwipeOffset(id, next)
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
  const moved = getSwipeOffset(id)
  if (!isDragging.value || (Math.abs(dx) < 16 && Math.abs(dy) < 16)) {
    touchItemId.value = ''
    isDragging.value = false
    return
  }
  const shouldOpen = moved <= -SWIPE_ACTION_WIDTH * 0.45 || dx < -30
  if (shouldOpen) {
    openId.value = id
    setSwipeOffset(id, -SWIPE_ACTION_WIDTH)
  } else {
    setSwipeOffset(id, 0)
    openId.value = ''
  }
  swipeJustFinished.value = true
  setTimeout(() => {
    swipeJustFinished.value = false
  }, 240)
  touchItemId.value = ''
  isDragging.value = false
}

function canDelete(it: any): boolean {
  return String(it?.status || '') === 'ended'
}

function swipeMainStyle(id: string) {
  return { transform: `translateX(${getSwipeOffset(id)}rpx)` }
}

function swipeActionStyle(id: string) {
  const offset = getSwipeOffset(id)
  const tx = Math.max(0, SWIPE_ACTION_WIDTH + offset)
  return { transform: `translateX(${tx}rpx)` }
}

function isSwiped(id: string): boolean {
  return getSwipeOffset(id) < 0
}

function getSwipeOffset(id: string): number {
  return swipeOffsetById.value[id] || 0
}

function setSwipeOffset(id: string, offset: number) {
  swipeOffsetById.value = { ...swipeOffsetById.value, [id]: clampSwipeOffset(offset) }
}

function clampSwipeOffset(v: number): number {
  return Math.max(-SWIPE_ACTION_WIDTH, Math.min(0, Math.round(v)))
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
    await deleteScorebook(String(it.id))
    items.value = items.value.filter((x) => x.id !== it.id)
    openId.value = ''
  } catch (e: any) {
    uni.showToast({ title: e?.message || '删除失败', icon: 'none' })
  }
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

function sortScorebooks(list: any[]): any[] {
  const rank = (it: any) => (String(it?.status || '') === 'ended' ? 1 : 0)
  const ts = (it: any) => {
    const raw = String(it?.startTime || it?.createdAt || '')
    const time = raw ? new Date(raw).getTime() : 0
    return Number.isFinite(time) ? time : 0
  }
  return [...list].sort((a, b) => {
    const ra = rank(a)
    const rb = rank(b)
    if (ra !== rb) return ra - rb
    return ts(b) - ts(a)
  })
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
.list-loading {
  margin-top: 12rpx;
  display: flex;
  justify-content: center;
  padding: 20rpx 0;
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
  padding: 0 24rpx;
  height: 72rpx;
  border-radius: 16rpx;
  border: 1rpx solid transparent;
  background: #f2f2f2;
  box-shadow: 0 6rpx 16rpx rgba(0, 0, 0, 0.06);
  transition: all 0.2s ease;
}
.search-input-wrap.focused {
  border-color: #e3e5e8;
  background: #fff;
  box-shadow: 0 10rpx 22rpx rgba(0, 0, 0, 0.08);
}
.search-icon {
  width: 30rpx;
  height: 30rpx;
  flex: none;
  opacity: 0.55;
}
.search-input {
  flex: 1;
  min-width: 0;
  font-size: 28rpx;
  color: #333;
}
.search-placeholder {
  color: #999;
}
.search-clear {
  width: 40rpx;
  height: 40rpx;
  border-radius: 999rpx;
  background: #e7e7e7;
  border: none;
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
  opacity: 1;
  pointer-events: auto;
  transform: translateX(140rpx);
  transition: transform 0.28s cubic-bezier(0.22, 0.8, 0.2, 1);
}
.swipe-actions.dragging {
  transition: none;
}
.swipe-btn {
  width: 100%;
  height: 100%;
  padding: 0;
  border-radius: 0;
  background: var(--confirm-btn-bg-rgba, rgba(241, 241, 244, 0.9));
  color: var(--confirm-btn-color, #111111);
  font-size: 26rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  border: 0;
}
.swipe-btn::after {
  border: none;
}
.swipe-btn.disabled {
  background: #c7c7c7;
}
.swipe-main {
  position: relative;
  z-index: 1;
  transform: translateX(0);
  transition: transform 0.28s cubic-bezier(0.22, 0.8, 0.2, 1);
}
.swipe-main.dragging {
  transition: none;
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
