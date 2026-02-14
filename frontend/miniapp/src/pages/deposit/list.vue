<template>
  <view class="page" :style="themeStyle">
    <view class="card hero">
      <view class="title-row hero-row">
        <view class="title">{{ displayName }}</view>
        <view class="badge">
          <image class="bank-icon" :src="bankIcon" mode="aspectFit" />
          <text>银行卡 {{ accounts.length }}</text>
        </view>
      </view>
      <view class="hero-stats">
        <view class="stat">
          <view class="stat-label">存款</view>
          <view class="stat-value" v-for="(line, idx) in depositLines" :key="`deposit-${idx}`">{{ line }}</view>
        </view>
        <view class="stat stat-right">
          <view class="stat-label">本年收益</view>
          <view class="stat-value" v-for="(line, idx) in yieldLines" :key="`yield-${idx}`">{{ line }}</view>
        </view>
      </view>
    </view>

    <view class="card member-card">
      <view class="title-row">
        <view class="title">银行</view>
        <view class="title-actions">
          <button class="icon-btn primary" @click="openAccountCreate">
            <image class="icon-img" :src="addIcon" mode="aspectFit" />
          </button>
        </view>
      </view>
      <view class="tip">点头像记一笔存款</view>
      <view class="hint" v-if="accounts.length === 0">暂无银行账户</view>
      <view class="member-grid" v-else>
        <view class="member" v-for="acc in accounts" :key="acc.id" @click="openDeposit(acc)">
          <button class="member-edit" @click.stop="openAccountEdit(acc)">
            <image class="icon-img small" :src="editIcon" mode="aspectFit" />
          </button>
          <view class="avatar-wrap">
            <image v-if="acc.avatarUrl" class="avatar" :src="acc.avatarUrl" mode="aspectFill" />
            <view v-else class="avatar avatar-fallback" :style="avatarStyle(acc.bank)">{{ initialOf(acc.bank) }}</view>
          </view>
          <view class="member-name">{{ acc.bank }}</view>
          <view class="member-total">{{ accountTotalLabel(acc.id) }}</view>
          <view class="member-sub" v-if="acc.branch">{{ acc.branch }}</view>
          <view class="member-sub" v-else-if="acc.accountNo">尾号 {{ tailOf(acc.accountNo) }}</view>
          <view class="member-sub" v-else-if="acc.holder">户名 {{ acc.holder }}</view>
        </view>
      </view>
    </view>

    <view class="card">
      <view class="title-row">
        <view class="title">记录</view>
        <view class="record-totals" v-if="showActiveTotal || showActiveInterest">
          <view class="record-total" v-if="showActiveTotal">总 {{ filteredActiveTotal }}</view>
          <view class="record-total" v-if="showActiveInterest">总利息 {{ filteredActiveInterest }}</view>
        </view>
      </view>
      <view class="filter-panel">
        <scroll-view class="filter-scroll" scroll-x>
          <view
            v-for="opt in statusOptions"
            :key="opt"
            class="filter-chip"
            :class="{ active: statusFilter === opt }"
            @click="statusFilter = opt"
          >
            {{ opt }}
          </view>
        </scroll-view>
      </view>
      <view class="filter-panel" v-if="bankOptions.length">
        <scroll-view class="filter-scroll" scroll-x>
          <view class="filter-chip" :class="{ active: !bankFilterId }" @click="setBankFilter('')">全部</view>
          <view
            class="filter-chip"
            v-for="opt in bankOptions"
            :key="opt.id"
            :class="{ active: bankFilterId === opt.id }"
            @click="setBankFilter(opt.id)"
          >
            {{ opt.label }}
          </view>
        </scroll-view>
      </view>
      <view class="filter-panel" v-if="tagOptions.length">
        <scroll-view class="filter-scroll" scroll-x>
          <view class="filter-chip" :class="{ active: tagFilters.length === 0 }" @click="clearTagFilter">全部</view>
          <view
            class="filter-chip"
            v-for="tag in tagOptions"
            :key="tag"
            :class="{ active: tagFilters.includes(tag) }"
            @click="toggleTagFilter(tag)"
          >
            {{ tag }}
          </view>
        </scroll-view>
      </view>
      <view class="hint" v-if="filteredRecords.length === 0">暂无存款记录</view>
      <view class="records" v-else>
        <view class="record-wrap" v-for="(rec, idx) in filteredRecords" :key="rec.id">
          <view class="swipe-item" :class="{ open: isSwiped(rec.id) }">
            <view
              class="swipe-actions"
              :class="{ dragging: isDragging && touchItemId === rec.id }"
              :style="swipeActionStyle(rec.id)"
            >
              <button class="swipe-btn" v-if="canWithdraw(rec)" @click.stop="onWithdraw(rec)">支取</button>
              <button class="swipe-btn danger" @click.stop="confirmDelete(rec)">删除</button>
            </view>
            <view
              class="record swipe-main"
              :class="{ dragging: isDragging && touchItemId === rec.id }"
              :style="swipeMainStyle(rec.id)"
              @touchstart="onTouchStart($event, rec.id)"
              @touchmove="onTouchMove($event, rec.id)"
              @touchend="onTouchEnd($event, rec.id)"
              @click="onRecordTap(rec)"
            >
              <view class="record-row">
                <view class="record-user">
                  <image v-if="rec.avatarUrl" class="record-avatar" :src="rec.avatarUrl" mode="aspectFill" />
                  <view v-else class="record-avatar avatar-fallback" :style="avatarStyle(rec.bank)">{{ initialOf(rec.bank) }}</view>
                  <text class="record-name">{{ rec.bank }}</text>
                </view>
                <view class="record-amount">{{ currencySymbol(rec.currency) }}{{ formatAmount(rec.amount) }}</view>
              </view>
              <view class="record-footer">
                <view class="record-info">
                  <view class="record-info-top">
                  <text class="record-item" v-if="rec.status === '已支取'">支取日 {{ rec.withdrawnAt || rec.endDate }}</text>
                    <template v-else-if="rec.status === '已到期'">
                      <text class="record-item">到期日 {{ rec.endDate }}</text>
                    </template>
                    <template v-else>
                      <text class="record-item">到期日 {{ rec.endDate }}</text>
                      <text class="record-item">距到期 {{ rec.daysLeft }} 天</text>
                    </template>
                  </view>
                  <view class="record-info-bottom">
                    <text class="record-item">利率 {{ rec.rate }}%</text>
                    <text class="record-item">存期 {{ rec.termValue }}{{ termLabel(rec.termUnit) }}</text>
                  </view>
                  <view class="record-tags">
                    <template v-if="rec.tags && rec.tags.length">
                      <view class="record-tag" v-for="(tag, idx2) in rec.tags" :key="`${tag}-${idx2}`">{{ tag }}</view>
                    </template>
                    <text class="record-tag empty" v-else>无标签</text>
                  </view>
                </view>
                <view class="record-status" v-if="rec.status !== '未到期'">{{ rec.status }}</view>
              </view>
            </view>
          </view>
          <view class="record-divider" v-if="idx !== filteredRecords.length - 1"></view>
        </view>
      </view>
      <view class="records-footer" v-if="recordsLoaded && records.length > 0">
        <t-loading v-if="recordsPaging" :loading="true" text="加载中…" />
        <text v-else-if="recordsHasMore">滑动加载下一页</text>
        <text v-else>已全部加载完毕</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onReachBottom, onShow } from '@dcloudio/uni-app'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor } from '../../utils/theme'
import { avatarStyle } from '../../utils/avatar-color'
import { getCurrencyMeta } from '../../utils/currency'
import {
  deleteDepositRecord,
  getDepositStats,
  listDepositAccounts,
  listDepositRecords,
  listDepositTags,
  updateDepositRecord,
} from '../../utils/api'

type Account = {
  id: string
  bank: string
  branch: string
  accountNo: string
  holder: string
  avatarUrl: string
  note: string
}

type DepositRecord = {
  id: string
  accountId: string
  currency: string
  amount: number
  amountUpper: string
  termValue: number
  termUnit: 'year' | 'month' | 'day'
  rate: number
  startDate: string
  endDate: string
  withdrawnAt?: string
  interest: number
  tags?: string[]
  note: string
  attachments: AttachmentItem[]
  status: '未到期' | '已到期' | '已支取'
}

type AttachmentItem = {
  type: 'image' | 'file'
  url: string
  name?: string
}

type RecordView = DepositRecord & {
  bank: string
  avatarUrl: string
  daysLeft: number
  statusClass: string
}

const accounts = ref<Account[]>([])
const records = ref<DepositRecord[]>([])
const recordsLoaded = ref(false)
const recordsPaging = ref(false)
const recordsHasMore = ref(false)
const recordsNextOffset = ref(0)
const recordsPageSize = 20
const statusFilter = ref('未到期')
const bankFilterId = ref('')
const tagFilters = ref<string[]>([])
const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))
const user = ref<any>(null)
const tagOptionsRemote = ref<string[]>([])
const stats = ref<any>(null)
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
const SWIPE_ACTION_WIDTH = 220
const addIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14"/><path d="M5 12h14"/></svg>'
const editIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5z"/></svg>'
const bankIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="%23ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 10h18"/><path d="M5 10v8"/><path d="M9 10v8"/><path d="M15 10v8"/><path d="M19 10v8"/><path d="M2 18h20"/><path d="M12 4l9 4H3z"/></svg>'

onShow(async () => {
  syncTheme()
  user.value = (uni.getStorageSync('user') as any) || null
  await load()
  openId.value = ''
  swipeOffsetById.value = {}
})

onReachBottom(() => {
  if (recordsPaging.value || !recordsHasMore.value) return
  loadMoreRecords()
})

function syncTheme() {
  const base = getThemeBaseColor()
  themeStyle.value = buildThemeVars(base)
  applyNavigationBarTheme(base)
  applyTabBarTheme(base)
}

const recordItems = computed<RecordView[]>(() => {
  const accMap = new Map(accounts.value.map((acc) => [acc.id, acc]))
  const list = records.value
    .map((rec) => {
      const acc = accMap.get(rec.accountId)
      const bank = acc?.bank || '未知银行'
      const avatarUrl = acc?.avatarUrl || ''
      const daysLeft = calcDaysLeft(rec.endDate)
      const status = normalizeStatus(rec, daysLeft)
      return {
        ...rec,
        bank,
        avatarUrl,
        daysLeft: Math.max(0, daysLeft),
        status,
        statusClass: status === '已支取' ? 'status-withdrawn' : status === '已到期' ? 'status-matured' : 'status-active',
      }
    })
    .sort((a, b) => compareRecord(a, b))

  return list
})

const statusOptions = ['全部', '未到期', '已到期', '已支取']

const bankOptions = computed(() => {
  const map = new Map<string, string>()
  for (const acc of accounts.value) {
    const id = String(acc.id || '')
    const label = String(acc.bank || '').trim()
    if (id && label && !map.has(id)) map.set(id, label)
  }
  return Array.from(map.entries()).map(([id, label]) => ({ id, label }))
})

const tagOptions = computed(() => {
  if (tagOptionsRemote.value.length) return tagOptionsRemote.value
  const set = new Set<string>()
  for (const rec of recordItems.value) {
    const tags = Array.isArray(rec.tags) ? rec.tags : []
    for (const tag of tags) {
      const v = String(tag || '').trim()
      if (v) set.add(v)
    }
  }
  return Array.from(set.values())
})

const filteredRecords = computed<RecordView[]>(() => {
  const status = statusFilter.value
  return recordItems.value.filter((rec) => {
    if (bankFilterId.value && rec.accountId !== bankFilterId.value) return false
    if (tagFilters.value.length > 0) {
      const tags = Array.isArray(rec.tags) ? rec.tags : []
      return tagFilters.value.some((tag) => tags.includes(tag))
    }
    if (status === '全部') return true
    return rec.status === status
  })
})

const filteredActiveTotals = computed(() => {
  const active = filteredRecords.value.filter((rec) => rec.status === '未到期')
  return sortCurrencyItems(aggregateByCurrency(active, (rec) => rec.amount))
})
const filteredActiveTotal = computed(() => formatCurrencySummary(filteredActiveTotals.value))
const showActiveTotal = computed(() => filteredActiveTotals.value.some((item) => Number(item.amount || 0) > 0))
const filteredActiveInterestTotals = computed(() => {
  const active = filteredRecords.value.filter((rec) => rec.status === '未到期')
  return sortCurrencyItems(aggregateByCurrency(active, (rec) => rec.interest || 0))
})
const filteredActiveInterest = computed(() => formatCurrencySummary(filteredActiveInterestTotals.value))
const showActiveInterest = computed(() => filteredActiveInterestTotals.value.some((item) => Number(item.amount || 0) > 0))

const displayName = computed(() => String(user.value?.nickname || '').trim() || '我')

const accountTotals = computed(() => {
  if (stats.value?.accountTotals?.length) {
    const map = new Map<string, Map<string, number>>()
    for (const item of stats.value.accountTotals) {
      const id = String(item.accountId || '')
      if (!id) continue
      const currency = String(item.currency || 'CNY').toUpperCase()
      if (!map.has(id)) map.set(id, new Map())
      const curMap = map.get(id)!
      curMap.set(currency, (curMap.get(currency) || 0) + Number(item.amount || 0))
    }
    return map
  }
  const map = new Map<string, Map<string, number>>()
  for (const rec of recordItems.value) {
    if (rec.status === '已支取') continue
    const id = String(rec.accountId || '')
    if (!id) continue
    const currency = String(rec.currency || 'CNY').toUpperCase()
    if (!map.has(id)) map.set(id, new Map())
    const curMap = map.get(id)!
    curMap.set(currency, (curMap.get(currency) || 0) + (rec.amount || 0))
  }
  return map
})

const totalDepositItems = computed(() => {
  if (stats.value?.totals?.length) {
    return sortCurrencyItems(
      stats.value.totals.map((item: any) => ({
        currency: String(item.currency || 'CNY'),
        amount: Number(item.amount || 0),
      })),
    )
  }
  const active = recordItems.value.filter((rec) => rec.status === '未到期')
  return sortCurrencyItems(aggregateByCurrency(active, (rec) => rec.amount))
})

const annualYieldItems = computed(() => {
  if (stats.value?.annualYields?.length) {
    return sortCurrencyItems(
      stats.value.annualYields.map((item: any) => ({
        currency: String(item.currency || 'CNY'),
        amount: Number(item.amount || 0),
      })),
    )
  }
  const active = recordItems.value.filter((rec) => rec.status === '未到期' && isSameYear(rec.endDate))
  return sortCurrencyItems(aggregateByCurrency(active, (rec) => rec.interest || 0))
})

const depositLines = computed(() => formatCurrencyLines(totalDepositItems.value, ''))
const yieldLines = computed(() => formatCurrencyLines(annualYieldItems.value, '+'))

async function load() {
  try {
    recordsLoaded.value = false
    recordsPaging.value = false
    recordsHasMore.value = true
    recordsNextOffset.value = 0
    const [accountsRes, recordsRes] = await Promise.all([
      listDepositAccounts(200, 0),
      listDepositRecords({ limit: recordsPageSize, offset: 0 }),
    ])
    accounts.value = (accountsRes?.items || []).map((acc: any) => ({
      id: String(acc.id || ''),
      bank: String(acc.bank || ''),
      branch: String(acc.branch || ''),
      accountNo: String(acc.accountNo || ''),
      holder: String(acc.holder || ''),
      avatarUrl: String(acc.avatarUrl || ''),
      note: String(acc.note || ''),
    }))
    records.value = (recordsRes?.items || []).map((rec: any) => ({
      id: String(rec.id || ''),
      accountId: String(rec.accountId || ''),
      currency: String(rec.currency || 'CNY'),
      amount: Number(rec.amount || 0),
      amountUpper: String(rec.amountUpper || ''),
      termValue: Number(rec.termValue || 0),
      termUnit: (rec.termUnit === 'month' ? 'month' : rec.termUnit === 'day' ? 'day' : 'year') as
        | 'year'
        | 'month'
        | 'day',
      rate: Number(rec.rate || 0),
      startDate: String(rec.startDate || ''),
      endDate: String(rec.endDate || ''),
      withdrawnAt: String(rec.withdrawnAt || ''),
      interest: Number(rec.interest || 0),
      tags: Array.isArray(rec.tags) ? rec.tags : [],
      note: String(rec.note || ''),
      attachments: Array.isArray(rec.attachments) ? rec.attachments : [],
      status: (rec.status === '已到期' || rec.status === '已支取' ? rec.status : '未到期') as
        | '未到期'
        | '已到期'
        | '已支取',
    }))
    recordsLoaded.value = true
    recordsNextOffset.value = records.value.length
    recordsHasMore.value = records.value.length >= recordsPageSize
    await loadStatsAndTags()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  }
}

async function loadMoreRecords() {
  if (recordsPaging.value || !recordsHasMore.value) return
  recordsPaging.value = true
  try {
    const res = await listDepositRecords({ limit: recordsPageSize, offset: recordsNextOffset.value })
    const items = res?.items || []
    const seen = new Set<string>()
    for (const it of records.value) {
      const id = String(it?.id || '')
      if (id) seen.add(id)
    }
    for (const rec of items) {
      const id = String(rec?.id || '')
      if (!id || seen.has(id)) continue
      seen.add(id)
      records.value.push({
        id: String(rec.id || ''),
        accountId: String(rec.accountId || ''),
        currency: String(rec.currency || 'CNY'),
        amount: Number(rec.amount || 0),
        amountUpper: String(rec.amountUpper || ''),
        termValue: Number(rec.termValue || 0),
        termUnit: (rec.termUnit === 'month' ? 'month' : rec.termUnit === 'day' ? 'day' : 'year') as
          | 'year'
          | 'month'
          | 'day',
        rate: Number(rec.rate || 0),
        startDate: String(rec.startDate || ''),
        endDate: String(rec.endDate || ''),
        withdrawnAt: String(rec.withdrawnAt || ''),
        interest: Number(rec.interest || 0),
        tags: Array.isArray(rec.tags) ? rec.tags : [],
        note: String(rec.note || ''),
        attachments: Array.isArray(rec.attachments) ? rec.attachments : [],
        status: (rec.status === '已到期' || rec.status === '已支取' ? rec.status : '未到期') as
          | '未到期'
          | '已到期'
          | '已支取',
      })
    }
    recordsNextOffset.value += items.length
    recordsHasMore.value = items.length >= recordsPageSize
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    recordsPaging.value = false
    recordsLoaded.value = true
  }
}

async function loadStatsAndTags() {
  try {
    const [tagsRes, statsRes] = await Promise.all([
      listDepositTags(),
      getDepositStats({ status: '未到期' }),
    ])
    tagOptionsRemote.value = (tagsRes?.items || []).map((it: any) => String(it?.tag || '')).filter(Boolean)
    stats.value = statsRes?.stats || null
  } catch (e) {}
}

function openAccountCreate() {
  uni.navigateTo({ url: '/pages/deposit/account-create' })
}

function openDeposit(acc: Account) {
  const id = encodeURIComponent(String(acc.id || ''))
  uni.navigateTo({ url: `/pages/deposit/deposit-create?accountId=${id}` })
}

function openAccountEdit(acc: Account) {
  if (!acc?.id) return
  const id = encodeURIComponent(String(acc.id))
  uni.navigateTo({ url: `/pages/deposit/account-create?id=${id}` })
}

function openRecordDetail(rec: RecordView) {
  const id = encodeURIComponent(String(rec.id || ''))
  if (!id) return
  uni.navigateTo({ url: `/pages/deposit/detail?id=${id}` })
}

function onRecordTap(rec: RecordView) {
  if (swipeJustFinished.value) return
  if (openId.value === rec.id) {
    setSwipeOffset(rec.id, 0)
    openId.value = ''
    return
  }
  openRecordDetail(rec)
}

function canWithdraw(rec: RecordView): boolean {
  return rec.status === '未到期'
}

async function onWithdraw(rec: RecordView) {
  if (!canWithdraw(rec)) {
    uni.showToast({ title: '只能支取未到期存款', icon: 'none' })
    return
  }
  const res = await new Promise<UniApp.ShowModalRes>((resolve) => {
    uni.showModal({ title: '确认支取', content: '提前支取没有利息，是否确认？', success: resolve })
  })
  if (!res.confirm) return
  try {
    const nextDate = formatDate(new Date())
    const resp = await updateDepositRecord(rec.id, {
      status: '已支取',
      withdrawnAt: nextDate,
      endDate: nextDate,
    })
    const updated = resp?.record
    if (updated) {
      records.value = records.value.map((item) =>
        item.id === rec.id
          ? {
              ...item,
              status: updated.status || '已支取',
              endDate: updated.endDate || nextDate,
              withdrawnAt: updated.withdrawnAt || nextDate,
            }
          : item,
      )
    } else {
      records.value = records.value.map((item) =>
        item.id === rec.id ? { ...item, status: '已支取', endDate: nextDate, withdrawnAt: nextDate } : item,
      )
    }
    await loadStatsAndTags()
    setSwipeOffset(rec.id, 0)
    openId.value = ''
    uni.showToast({ title: '已支取', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '支取失败', icon: 'none' })
  }
}

async function confirmDelete(rec: RecordView) {
  const res = await new Promise<UniApp.ShowModalRes>((resolve) => {
    uni.showModal({ title: '确认删除', content: '确定删除这笔存款记录？', success: resolve })
  })
  if (!res.confirm) return
  try {
    await deleteDepositRecord(rec.id)
    records.value = records.value.filter((item) => item.id !== rec.id)
    await loadStatsAndTags()
    setSwipeOffset(rec.id, 0)
    openId.value = ''
    uni.showToast({ title: '已删除', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '删除失败', icon: 'none' })
  }
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
    if (Math.abs(dx) < 6 && Math.abs(dy) < 6) return
    if (Math.abs(dx) <= Math.abs(dy)) {
      touchItemId.value = ''
      return
    }
    isDragging.value = true
  }
  if (e?.cancelable && typeof e.preventDefault === 'function') {
    e.preventDefault()
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

function tailOf(accountNo: string): string {
  const v = String(accountNo || '').replace(/\s+/g, '')
  if (!v) return ''
  return v.slice(-4)
}

function termLabel(unit: string): string {
  if (unit === 'year') return '年'
  if (unit === 'month') return '月'
  return '天'
}

function initialOf(name: string): string {
  const v = String(name || '').trim()
  return v ? v.slice(0, 1) : '存'
}

function currencySymbol(code: string): string {
  return getCurrencyMeta(code).symbol
}

function toggleTagFilter(tag: string) {
  const v = String(tag || '')
  if (!v) return
  if (tagFilters.value.includes(v)) {
    tagFilters.value = tagFilters.value.filter((item) => item !== v)
    return
  }
  tagFilters.value = [...tagFilters.value, v]
}

function clearTagFilter() {
  tagFilters.value = []
}

function setBankFilter(id: string) {
  bankFilterId.value = String(id || '')
}

function accountTotalLabel(accountId: string): string {
  const map = accountTotals.value.get(String(accountId || ''))
  if (!map || map.size === 0) return '0.00'
  const items = Array.from(map.entries()).map(([currency, amount]) => ({ currency, amount }))
  return formatCurrencySummary(items)
}

function normalizeStatus(rec: DepositRecord, daysLeft: number): '未到期' | '已到期' | '已支取' {
  if (rec.status === '已支取') return '已支取'
  if (daysLeft < 0) return '已到期'
  return '未到期'
}

function calcDaysLeft(endDate: string): number {
  const end = parseDate(endDate)
  if (!end) return 0
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const diff = end.getTime() - today.getTime()
  return Math.ceil(diff / 86400000)
}

function parseDate(value: string): Date | null {
  if (!value) return null
  const parts = String(value).split('-').map((v) => Number.parseInt(v, 10))
  if (parts.length < 3 || parts.some((p) => !Number.isFinite(p))) return null
  const [y, m, d] = parts
  return new Date(y, m - 1, d)
}

function isSameYear(dateStr: string): boolean {
  const date = parseDate(dateStr)
  if (!date) return false
  const now = new Date()
  return date.getFullYear() === now.getFullYear()
}

function compareRecord(a: RecordView, b: RecordView): number {
  const rank = (status: string) => {
    if (status === '未到期') return 0
    if (status === '已到期') return 1
    return 2
  }
  const r = rank(a.status) - rank(b.status)
  if (r !== 0) return r
  const dateA = a.status === '已支取' ? a.withdrawnAt || a.endDate : a.endDate
  const dateB = b.status === '已支取' ? b.withdrawnAt || b.endDate : b.endDate
  const da = parseDate(dateA)?.getTime() || 0
  const db = parseDate(dateB)?.getTime() || 0
  return da - db
}

function formatDate(date: Date): string {
  const y = date.getFullYear()
  const m = `${date.getMonth() + 1}`.padStart(2, '0')
  const d = `${date.getDate()}`.padStart(2, '0')
  return `${y}-${m}-${d}`
}

function formatAmount(amount: number): string {
  const n = Number(amount)
  if (!Number.isFinite(n)) return '0.00'
  return n.toFixed(2)
}

function aggregateByCurrency(list: RecordView[], picker: (rec: RecordView) => number) {
  const map = new Map<string, number>()
  for (const rec of list) {
    const key = String(rec.currency || 'CNY').toUpperCase()
    const prev = map.get(key) || 0
    map.set(key, prev + (picker(rec) || 0))
  }
  return Array.from(map.entries()).map(([currency, amount]) => ({ currency, amount }))
}

function formatCurrencySummary(items: { currency: string; amount: number }[]): string {
  if (!items.length) return '0.00'
  return items
    .map((item) => `${currencySymbol(item.currency)}${formatAmount(item.amount)}`)
    .join(' / ')
}

function sortCurrencyItems(items: { currency: string; amount: number }[]) {
  const list = [...items]
  list.sort((a, b) => {
    const aIsCny = String(a.currency || '').toUpperCase() === 'CNY'
    const bIsCny = String(b.currency || '').toUpperCase() === 'CNY'
    if (aIsCny && !bIsCny) return -1
    if (!aIsCny && bIsCny) return 1
    return String(a.currency || '').localeCompare(String(b.currency || ''))
  })
  return list
}

function formatCurrencyItem(item: { currency: string; amount: number }, prefix = ''): string {
  return `${prefix}${currencySymbol(item.currency)}${formatAmount(item.amount)}`
}

function formatCurrencyLines(items: { currency: string; amount: number }[], prefix = ''): string[] {
  if (!items.length) return ['0.00']
  return items.map((item) => formatCurrencyItem(item, prefix))
}
</script>

<style scoped>
.page {
  padding: 24rpx;
  padding-bottom: calc(140rpx + env(safe-area-inset-bottom));
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}
.card {
  background: #fff;
  border-radius: 18rpx;
  padding: 24rpx;
  box-shadow: 0 10rpx 26rpx rgba(0, 0, 0, 0.06);
}
.hero {
  background: linear-gradient(135deg, var(--brand-1) 0%, var(--brand-2) 100%);
  color: #fff;
  position: relative;
  overflow: hidden;
  border-radius: 20rpx;
}
.hero::before {
  content: '';
  position: absolute;
  right: -120rpx;
  top: -140rpx;
  width: 360rpx;
  height: 360rpx;
  border-radius: 999rpx;
  background: radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.18), rgba(255, 255, 255, 0));
  transform: rotate(12deg);
  pointer-events: none;
}
.hero::after {
  content: '';
  position: absolute;
  left: -140rpx;
  bottom: -180rpx;
  width: 420rpx;
  height: 420rpx;
  border-radius: 999rpx;
  background: radial-gradient(circle at 60% 40%, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0));
  transform: rotate(-10deg);
  pointer-events: none;
}
.hero-row {
  margin-bottom: 12rpx;
  position: relative;
  z-index: 1;
}
.hero .title {
  font-size: 36rpx;
  font-weight: 700;
  color: #fff;
}
.badge {
  display: inline-flex;
  align-items: center;
  gap: 8rpx;
  padding: 6rpx 12rpx;
  border-radius: 999rpx;
  background: var(--brand-strong);
  border: 1rpx solid rgba(255, 255, 255, 0.18);
  color: #fff;
  font-size: 22rpx;
}
.bank-icon {
  width: 20rpx;
  height: 20rpx;
}
.sub {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10rpx;
  position: relative;
  z-index: 1;
}
.hero-stats {
  display: flex;
  gap: 16rpx;
  position: relative;
  z-index: 1;
}
.stat {
  flex: 1;
  min-width: 0;
}
.stat-right {
  text-align: right;
}
.stat-label {
  font-size: 22rpx;
  color: rgba(255, 255, 255, 0.8);
}
.stat-value {
  margin-top: 6rpx;
  font-size: 34rpx;
  font-weight: 700;
  letter-spacing: 0.5rpx;
}
.pill {
  padding: 0 12rpx;
  height: 40rpx;
  line-height: 40rpx;
  border-radius: 999rpx;
  font-size: 22rpx;
  background: rgba(255, 255, 255, 0.18);
  color: #fff;
  border: 1rpx solid rgba(255, 255, 255, 0.16);
}
.pill.with-icon {
  position: relative;
  padding-left: 38rpx;
}
.pill.with-icon::before {
  content: '';
  position: absolute;
  left: 12rpx;
  top: 50%;
  width: 14rpx;
  height: 14rpx;
  border-radius: 4rpx;
  transform: translateY(-50%);
  background: #fff;
}
.pill.icon-yield::before {
  border-radius: 999rpx;
}
.pill.icon-member::before {
  background: #fff;
}
.title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  margin-bottom: 8rpx;
}
.title {
  font-size: 30rpx;
  font-weight: 600;
}
.record-totals {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4rpx;
}
.record-total {
  font-size: 24rpx;
  color: #666;
}
.title-actions {
  display: flex;
  align-items: center;
  gap: 10rpx;
}
.icon-btn {
  width: 60rpx;
  height: 60rpx;
  border-radius: 999rpx;
  background: #f6f7fb;
  color: #111;
  font-size: 30rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}
.icon-btn::after {
  border: none;
}
.icon-btn.primary {
  background: var(--brand-soft);
  color: var(--brand-strong);
}
.icon-img {
  width: 28rpx;
  height: 28rpx;
}
.tip {
  color: #888;
  font-size: 24rpx;
  margin-top: 6rpx;
}
.hint {
  color: #666;
  font-size: 26rpx;
  margin-top: 8rpx;
}
.member-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16rpx;
  margin-top: 12rpx;
}
.member {
  background: #f6f7fb;
  border-radius: 16rpx;
  padding: 16rpx 12rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
  position: relative;
}
.member-edit {
  position: absolute;
  right: 6rpx;
  top: 6rpx;
  width: 44rpx;
  height: 44rpx;
  border-radius: 999rpx;
  background: var(--brand-soft);
  color: var(--brand-strong);
  border: 1rpx solid #bde3dc;
  font-size: 22rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}
.member-edit::after {
  border: none;
}
.icon-img.small {
  width: 22rpx;
  height: 22rpx;
}
.avatar-wrap {
  width: 96rpx;
  height: 96rpx;
  border-radius: 48rpx;
  overflow: hidden;
  background: #fff;
}
.avatar {
  width: 96rpx;
  height: 96rpx;
  border-radius: 48rpx;
}
.avatar-fallback {
  background: var(--brand-soft);
  color: var(--brand-solid);
  font-size: 28rpx;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}
.member-name {
  font-size: 26rpx;
  font-weight: 600;
  text-align: center;
}
.member-total {
  font-size: 22rpx;
  color: var(--brand-solid);
  text-align: center;
}
.member-sub {
  font-size: 22rpx;
  color: #666;
  text-align: center;
}
.chip-group {
  display: flex;
  align-items: center;
  gap: 8rpx;
}
.chip {
  padding: 0 16rpx;
  height: 46rpx;
  line-height: 46rpx;
  border-radius: 999rpx;
  font-size: 22rpx;
  color: #666;
  background: #f1f2f6;
}
.chip.active {
  background: var(--brand-solid);
  color: #fff;
}
.filter-panel {
  margin-top: 12rpx;
  margin-bottom: 8rpx;
}
.filter-title {
  font-size: 24rpx;
  color: #666;
  margin-bottom: 8rpx;
}
.filter-scroll {
  white-space: nowrap;
}
.filter-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 16rpx;
  height: 44rpx;
  line-height: 44rpx;
  border-radius: 999rpx;
  font-size: 22rpx;
  color: #666;
  background: #f1f2f6;
  margin-right: 10rpx;
}
.filter-chip.active {
  background: var(--brand-solid);
  color: #fff;
}
.records {
  margin-top: 12rpx;
  display: flex;
  flex-direction: column;
  gap: 0;
}
.records-footer {
  margin-top: 16rpx;
  text-align: center;
  color: #999;
  font-size: 22rpx;
}
.record-wrap {
  display: flex;
  flex-direction: column;
}
.record-divider {
  height: 1rpx;
  background: #ececf2;
  margin: 12rpx 24rpx 0;
}
.swipe-item {
  position: relative;
  overflow: hidden;
  border-radius: 16rpx;
}
.swipe-actions {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 220rpx;
  display: flex;
  align-items: stretch;
  justify-content: center;
  background: #f3f4f6;
  opacity: 1;
  pointer-events: auto;
  transform: translateX(220rpx);
  transition: transform 0.28s cubic-bezier(0.22, 0.8, 0.2, 1);
}
.swipe-actions.dragging {
  transition: none;
}
.swipe-btn {
  flex: 1;
  height: 100%;
  padding: 0;
  border-radius: 0;
  background: #f1f2f6;
  color: #111;
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
.swipe-btn.danger {
  background: #ff6b6b;
  color: #fff;
}
.swipe-btn.disabled {
  background: #c7c7c7;
  color: #fff;
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
.record {
  background: #fff;
  border-radius: 16rpx;
  padding: 16rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
}
.record-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}
.record-user {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
}
.record-avatar {
  width: 44rpx;
  height: 44rpx;
  border-radius: 22rpx;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  font-size: 22rpx;
}
.record-avatar.avatar-fallback {
  background: var(--brand-soft);
  color: var(--brand-solid);
}
.record-name {
  font-size: 26rpx;
  font-weight: 600;
}
.record-amount {
  font-weight: 700;
  color: #111;
  font-size: 28rpx;
}
.record-footer {
  margin-top: 10rpx;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12rpx;
}
.record-info {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  font-size: 24rpx;
  color: #666;
}
.record-info-top,
.record-info-bottom {
  display: flex;
  flex-wrap: wrap;
  gap: 10rpx 16rpx;
}
.record-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
  margin-top: 6rpx;
}
.record-tag {
  padding: 0 12rpx;
  height: 34rpx;
  line-height: 34rpx;
  border-radius: 999rpx;
  background: #f1f2f6;
  color: #666;
  font-size: 20rpx;
}
.record-tag.empty {
  background: #f8f8fb;
  color: #999;
}
.record-item {
  color: #666;
}
.record-status {
  padding: 0 14rpx;
  height: 40rpx;
  line-height: 40rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.6);
  color: rgba(17, 17, 17, 0.6);
  font-size: 22rpx;
  font-weight: 600;
  flex: none;
}
</style>
