<template>
  <view class="page" :style="themeStyle">
    <view class="card">
      <view class="hint" v-if="!token">登录后可查看生日薄</view>
      <button class="btn confirm-btn" v-if="!token" @click="goLogin">去登录</button>

      <template v-else>
        <view class="searchbar">
          <view class="search-input-wrap" :class="{ focused: searchFocused }">
            <image class="search-icon" :src="searchIcon" mode="aspectFit" />
            <input
              class="search-input"
              v-model="keyword"
              placeholder="搜索联系人、备注或日期"
              placeholder-class="search-placeholder"
              confirm-type="search"
              @focus="onSearchFocus"
              @blur="onSearchBlur"
            />
            <view class="search-clear" v-if="keyword" @click="clearKeyword">×</view>
          </view>
          <button class="add-btn" @click="onAdd">
            <text class="add-plus">+</text>
          </button>
        </view>

        <view class="list-loading" v-if="loading">
          <t-loading :loading="true" text="加载中…" />
        </view>
        <view class="hint" v-else-if="loadError">加载失败</view>
        <view class="hint" v-else-if="items.length === 0">暂无生日记录</view>
        <view class="hint" v-else-if="filteredItems.length === 0">没有匹配结果</view>
        <view class="list" v-else>
        <view class="swipe-item" :class="{ open: isSwiped(it.id) }" v-for="it in filteredItems" :key="it.id">
          <view class="swipe-actions" :class="{ dragging: isDragging && touchItemId === it.id }" :style="swipeActionStyle(it.id)">
            <button class="swipe-btn" @click.stop="confirmDelete(it)">删除</button>
          </view>
          <view
            class="item swipe-main"
              :class="{ dragging: isDragging && touchItemId === it.id }"
              :style="swipeMainStyle(it.id)"
              @touchstart="onTouchStart($event, it.id)"
              @touchmove.stop="onTouchMove($event, it.id)"
              @touchend="onTouchEnd($event, it.id)"
              @click="onItemTap(it)"
            >
            <view class="avatar-wrap">
              <image v-if="it.avatarUrl" class="avatar" :src="it.avatarUrl" mode="aspectFill" />
              <view v-else class="avatar avatar-fallback" :style="avatarStyle(it.name || it.initial)">{{ it.initial }}</view>
            </view>
            <view class="item-body">
              <view class="row">
                <text class="name">{{ it.name }}</text>
                <text class="tag" v-if="it.relation">{{ it.relation }}</text>
                <view class="badge" :class="{ today: it.daysLeft === 0 }">{{ it.badgeText }}</view>
              </view>
              <view class="meta">
                <view class="date-row">
                <view class="date-pill" :class="{ primary: it.primaryType === 'solar' }">公历 {{ it.solarText }}</view>
                <view class="date-pill" :class="{ primary: it.primaryType === 'lunar' }">农历 {{ it.lunarText }}</view>
                </view>
                <text class="note" v-if="it.note">{{ it.note }}</text>
                <text class="age-row" v-if="it.ageLabel">距{{ it.ageLabel }}岁生日</text>
              </view>
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
import { deleteBirthday, listBirthdays, updateBirthday } from '../../utils/api'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor } from '../../utils/theme'
import { avatarStyle } from '../../utils/avatar-color'
import lunarCalendar from '../../utils/lunar-calendar.mjs'

type BirthdayItem = {
  id: string
  name: string
  solarBirthday: string
  lunarBirthday: string
  primaryType: 'solar' | 'lunar'
  relation?: string
  note?: string
  avatarUrl?: string
  daysLeft?: number | null
  primaryYear?: number
}

type BirthdayView = BirthdayItem & {
  solarText: string
  lunarText: string
  daysLeft: number
  badgeText: string
  initial: string
  ageLabel: string
}

const items = ref<BirthdayView[]>([])
const token = ref('')
const loading = ref(false)
const loadError = ref('')
const keyword = ref('')
const searchFocused = ref(false)
const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))
const openId = ref('')
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
    const relation = String(it?.relation || '').toLowerCase()
    const note = String(it?.note || '').toLowerCase()
    const solar = String(it?.solarText || '').toLowerCase()
    const lunar = String(it?.lunarText || '').toLowerCase()
    return name.includes(q) || relation.includes(q) || note.includes(q) || solar.includes(q) || lunar.includes(q)
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
    const res = await listBirthdays()
    const rawItems = res.items || []
    const updated = await refreshPrimaryDates(rawItems)
    const finalItems = updated ? (await listBirthdays()).items || [] : rawItems
    const mapped = finalItems.map((it: any) => decorateItem({
      id: String(it.id || ''),
      name: String(it.name || ''),
      solarBirthday: String(it.solarBirthday || ''),
      lunarBirthday: String(it.lunarBirthday || ''),
      primaryType: (it.primaryType === 'lunar' ? 'lunar' : 'solar') as 'solar' | 'lunar',
      relation: String(it.relation || ''),
      note: String(it.note || ''),
      avatarUrl: String(it.avatarUrl || ''),
      daysLeft: Number.isFinite(Number(it.daysLeft)) ? Number(it.daysLeft) : null,
      primaryYear: Number(it.primaryYear || 0),
    } as any))
    items.value = sortItems(mapped)
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

function decorateItem(it: BirthdayItem & { daysLeft?: number | null }): BirthdayView {
  const solarMmdd = toMonthDay(it.solarBirthday)
  const lunarMmdd = toMonthDay(it.lunarBirthday)
  const solarText = solarMmdd
  const lunarText = toLunarMonthDay(it.lunarBirthday)
  const primaryMmdd = it.primaryType === 'lunar' ? lunarMmdd : solarMmdd
  const daysLeft = Number.isFinite(it.daysLeft as number) ? Number(it.daysLeft) : calcDaysLeft(primaryMmdd)
  const badgeText =
    daysLeft === 0 ? '今天' : daysLeft === 1 ? '明天' : daysLeft === 2 ? '后天' : `还有${daysLeft}天`
  const ageLabel = calcAgeLabel(it.primaryType === 'lunar' ? it.lunarBirthday : it.solarBirthday, primaryMmdd)
  return {
    ...it,
    solarText,
    lunarText,
    daysLeft,
    badgeText,
    initial: String(it.name || '').trim().slice(0, 1) || '友',
    ageLabel,
  }
}

function toMonthDay(raw: string): string {
  const v = String(raw || '').trim()
  const m1 = v.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (m1) return formatMonthDay(Number(m1[2]), Number(m1[3]))
  const m2 = v.match(/^(\d{1,2})-(\d{1,2})$/)
  if (m2) return formatMonthDay(Number(m2[1]), Number(m2[2]))
  return '01-01'
}

function formatMonthDay(month: number, day: number): string {
  const mm = String(Math.max(1, Math.min(12, month)))
  const dd = String(Math.max(1, Math.min(31, day)))
  return `${mm}-${dd}`
}

function toLunarMonthDay(raw: string): string {
  const v = String(raw || '').trim()
  const m1 = v.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (m1) return formatLunarMonthDay(Number(m1[2]), Number(m1[3]))
  const m2 = v.match(/^(\d{1,2})-(\d{1,2})$/)
  if (m2) return formatLunarMonthDay(Number(m2[1]), Number(m2[2]))
  return '一月一日'
}

function formatLunarMonthDay(month: number, day: number): string {
  const m = Math.max(1, Math.min(12, month))
  const d = Math.max(1, Math.min(30, day))
  return `${toLunarMonth(m)}${toLunarDay(d)}`
}

function toLunarMonth(month: number): string {
  const names = ['一月', '二月', '三月', '四月', '五月', '六月', '七月', '八月', '九月', '十月', '冬月', '腊月']
  return names[month - 1] || `${month}月`
}

function toLunarDay(day: number): string {
  const names = [
    '初一',
    '初二',
    '初三',
    '初四',
    '初五',
    '初六',
    '初七',
    '初八',
    '初九',
    '初十',
    '十一',
    '十二',
    '十三',
    '十四',
    '十五',
    '十六',
    '十七',
    '十八',
    '十九',
    '二十',
    '廿一',
    '廿二',
    '廿三',
    '廿四',
    '廿五',
    '廿六',
    '廿七',
    '廿八',
    '廿九',
    '三十',
  ]
  return names[day - 1] || `${day}日`
}

function calcAgeLabel(raw: string, mmdd: string): string {
  const birthYear = extractYear(raw)
  if (!birthYear) return ''
  const nextYear = nextBirthdayYear(mmdd)
  if (!nextYear) return ''
  const age = nextYear - birthYear
  if (age <= 0 || !Number.isFinite(age)) return ''
  return String(age)
}

function extractYear(raw: string): number | null {
  const parts = String(raw || '').trim().split('-')
  if (parts.length !== 3) return null
  const y = Number(parts[0])
  return Number.isFinite(y) ? y : null
}

function nextBirthdayYear(mmdd: string): number | null {
  const parts = mmdd.split('-')
  if (parts.length !== 2) return null
  const month = Number(parts[0])
  const day = Number(parts[1])
  if (!Number.isFinite(month) || !Number.isFinite(day)) return null
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  let target = new Date(now.getFullYear(), month - 1, day)
  if (target < today) target = new Date(now.getFullYear() + 1, month - 1, day)
  return target.getFullYear()
}

function calcDaysLeft(mmdd: string): number {
  const parts = mmdd.split('-')
  const month = Number(parts[0])
  const day = Number(parts[1])
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  let target = new Date(now.getFullYear(), month - 1, day)
  if (target < today) {
    target = new Date(now.getFullYear() + 1, month - 1, day)
  }
  const diff = target.getTime() - today.getTime()
  return Math.round(diff / 86400000)
}

function onItemTap(it: BirthdayView) {
  if (swipeJustFinished.value) return
  if (openId.value === it.id) {
    openId.value = ''
    return
  }
  uni.navigateTo({ url: `/pages/birthday/detail?id=${encodeURIComponent(it.id)}` })
}

function onAdd() {
  uni.navigateTo({ url: '/pages/birthday/create' })
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
  if (e?.cancelable && typeof e.preventDefault === 'function') {
    e.preventDefault()
  }
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

async function confirmDelete(it: BirthdayView) {
  const res = await new Promise<UniApp.ShowModalRes>((resolve) => {
    uni.showModal({ title: '确认删除', content: `确定删除「${it.name}」？`, success: resolve })
  })
  if (!res.confirm) return
  try {
    await deleteBirthday(String(it.id))
    items.value = items.value.filter((x) => x.id !== it.id)
    openId.value = ''
  } catch (e: any) {
    uni.showToast({ title: e?.message || '删除失败', icon: 'none' })
  }
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

function goLogin() {
  uni.setStorageSync('scorehub.afterLogin', { url: '/pages/birthday/list', ts: Date.now() })
  uni.switchTab({ url: '/pages/my/index' })
}

function sortItems(list: BirthdayView[]): BirthdayView[] {
  return [...list].sort((a, b) => {
    if (a.daysLeft !== b.daysLeft) return a.daysLeft - b.daysLeft
    return a.name.localeCompare(b.name)
  })
}

async function refreshPrimaryDates(list: any[]): Promise<boolean> {
  const currentYear = new Date().getFullYear()
  const targets = list.filter(
    (it) =>
      String(it?.primaryType || '') === 'lunar' &&
      Number(it?.primaryYear || 0) !== currentYear &&
      String(it?.lunarBirthday || '').trim(),
  )
  if (!targets.length) return false
  await Promise.all(
    targets.map(async (it) => {
      const md = parseMonthDay(String(it.lunarBirthday || ''))
      if (!md) return
      const solarObj = lunarCalendar.lunar2solar(currentYear, md.month, md.day, false)
      const primaryMonth = Number(solarObj?.cMonth) || md.month
      const primaryDay = Number(solarObj?.cDay) || md.day
      try {
        await updateBirthday(String(it.id || ''), { primaryMonth, primaryDay, primaryYear: currentYear })
      } catch (e: any) {
        // ignore single update failures
      }
    }),
  )
  return true
}

function parseMonthDay(raw: string): { month: number; day: number } | null {
  const v = String(raw || '').trim()
  const m1 = v.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (m1) return { month: Number(m1[2]), day: Number(m1[3]) }
  const m2 = v.match(/^(\d{1,2})-(\d{1,2})$/)
  if (m2) return { month: Number(m2[1]), day: Number(m2[2]) }
  return null
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
.hint {
  color: #666;
  font-size: 26rpx;
  margin-top: 20rpx;
  text-align: center;
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
  display: flex;
  align-items: center;
  gap: 12rpx;
}
.search-input-wrap {
  flex: 1;
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
.add-btn {
  width: 72rpx;
  height: 72rpx;
  padding: 0;
  border-radius: 18rpx;
  background: var(--brand-soft);
  color: var(--brand-solid);
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
}
.add-btn::after {
  border: none;
}
.add-plus {
  font-size: 40rpx;
  font-weight: 600;
  line-height: 1;
}
.list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
  margin-top: 16rpx;
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
  padding: 18rpx 20rpx;
  display: flex;
  gap: 16rpx;
  align-items: center;
}
.avatar-wrap {
  width: 72rpx;
  height: 72rpx;
  flex: none;
}
.avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 36rpx;
  background: #f6f7fb;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 30rpx;
  font-weight: 600;
}
.avatar-fallback {
  background: var(--brand-soft);
  color: var(--brand-solid);
}
.item-body {
  flex: 1;
  min-width: 0;
}
.row {
  display: flex;
  align-items: center;
  gap: 10rpx;
}
.name {
  font-size: 30rpx;
  font-weight: 600;
  color: #111;
  max-width: 220rpx;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tag {
  font-size: 22rpx;
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  background: #f2f2f2;
  color: #666;
  flex: none;
}
.badge {
  margin-left: auto;
  font-size: 22rpx;
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  background: var(--brand-soft);
  color: var(--brand-solid);
  flex: none;
}
.badge.today {
  background: var(--brand-solid);
  color: #fff;
}
.meta {
  margin-top: 6rpx;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
  align-items: flex-start;
  font-size: 24rpx;
  color: #666;
}
.date-row {
  display: flex;
  gap: 10rpx;
  flex-wrap: wrap;
}
.date-pill {
  font-size: 22rpx;
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  background: #f2f2f2;
  color: #666;
}
.date-pill.primary {
  background: var(--brand-soft);
  color: var(--brand-solid);
}
.note {
  color: #999;
  max-width: 280rpx;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.age-row {
  margin-top: 2rpx;
  align-self: flex-end;
  font-size: 22rpx;
  color: #888;
}
</style>
