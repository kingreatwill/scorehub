<template>
  <view class="page" :style="themeStyle">
    <view class="card" v-if="loading">
      <t-loading :loading="true" text="加载中…" />
    </view>
    <template v-else>
      <view class="card account-card" v-if="account">
        <view class="account-avatar">
          <image v-if="account.avatarUrl" class="avatar" :src="account.avatarUrl" mode="aspectFill" />
          <view v-else class="avatar avatar-fallback" :style="avatarStyle(account.bank)">{{ initialOf(account.bank) }}</view>
        </view>
        <view class="account-info">
          <view class="account-name">{{ account.bank }}</view>
          <view class="account-sub" v-if="account.branch">{{ account.branch }}</view>
          <view class="account-sub" v-if="account.accountNo">尾号 {{ tailOf(account.accountNo) }}</view>
        </view>
      </view>

      <view class="card" v-if="record">
        <view class="section-title">存款信息</view>
        <view class="info-row">
          <text class="label">币种</text>
          <text class="value">{{ currencyMeta.label }}</text>
        </view>
        <view class="info-row">
          <text class="label">存款金额</text>
          <text class="value">{{ currencyMeta.symbol }}{{ formatAmount(record.amount) }}</text>
        </view>
        <view class="info-row" v-if="amountUpperDisplay">
          <text class="label">金额大写</text>
          <text class="value">{{ amountUpperDisplay }}</text>
        </view>
        <view class="info-row">
          <text class="label">存期</text>
          <text class="value">{{ record.termValue }}{{ termLabel(record.termUnit) }}</text>
        </view>
        <view class="info-row">
          <text class="label">年化利率</text>
          <text class="value">{{ record.rate }}%</text>
        </view>
        <view class="info-row" v-if="record.receiptNo">
          <text class="label">单据号</text>
          <text class="value">{{ record.receiptNo }}</text>
        </view>
        <view class="info-row">
          <text class="label">起息日</text>
          <text class="value">{{ record.startDate }}</text>
        </view>
        <view class="info-row">
          <text class="label">到期日</text>
          <text class="value">{{ record.endDate }}</text>
        </view>
        <view class="info-row">
          <text class="label">到期利息</text>
          <text class="value">{{ currencyMeta.symbol }}{{ formatAmount(record.interest) }}</text>
        </view>
        <view class="info-row">
          <text class="label">状态</text>
          <text class="value">{{ record.status }}</text>
        </view>
      </view>

      <view class="card" v-if="record && record.tags && record.tags.length">
        <view class="section-title">标签</view>
        <view class="tag-list">
          <view class="tag-chip" v-for="(tag, idx) in record.tags" :key="`${tag}-${idx}`">{{ tag }}</view>
        </view>
      </view>

      <view class="card" v-if="record && attachments.length">
        <view class="section-title">存单照片/附件</view>
        <view class="attachment-grid">
          <view class="attachment" v-for="(item, idx) in attachments" :key="attachmentKey(item, idx)">
            <template v-if="item.type === 'image'">
              <image class="attachment-img" :src="item.url" mode="aspectFill" />
            </template>
            <template v-else>
              <view class="file-card">
                <view class="file-icon">FILE</view>
                <view class="file-name">{{ item.name || '附件' }}</view>
              </view>
            </template>
          </view>
        </view>
      </view>

      <view class="card" v-if="record && record.note">
        <view class="section-title">备注</view>
        <view class="note">{{ record.note }}</view>
      </view>

      <view class="card" v-if="!record">
        <view class="hint">未找到记录</view>
      </view>
    </template>

    <view class="fab-mask" v-if="actionMenuOpen && hasActions" @tap="closeActionMenu" />
    <view class="fab" v-if="hasActions" @tap.stop>
      <view class="fab-panel" :class="{ open: actionMenuOpen }" @tap.stop>
        <button size="mini" class="action-btn" :class="{ disabled: !canWithdraw }" @click="withdrawRecord">
          支取
        </button>
      </view>
      <button class="fab-toggle" :class="{ active: actionMenuOpen }" :style="fabToggleStyle" @tap.stop="toggleActionMenu">
        <image class="fab-icon" :src="actionMenuOpen ? closeIcon : moreIcon" mode="aspectFit" />
      </button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor, normalizeHexColor } from '../../utils/theme'
import { avatarStyle } from '../../utils/avatar-color'
import { getCurrencyMeta } from '../../utils/currency'
import { getDepositAccount, getDepositRecord, updateDepositRecord } from '../../utils/api'

type Account = {
  id: string
  bank: string
  branch: string
  accountNo: string
  holder: string
  avatarUrl: string
  note: string
}

type AttachmentItem = {
  type: 'image' | 'file'
  url: string
  name?: string
}

type DepositRecord = {
  id: string
  accountId: string
  currency: string
  amount: number
  amountUpper?: string
  termValue: number
  termUnit: 'year' | 'month' | 'day'
  rate: number
  receiptNo?: string
  startDate: string
  endDate: string
  withdrawnAt?: string
  interest: number
  tags?: string[]
  note: string
  attachments: AttachmentItem[] | string[]
  status: '未到期' | '已到期' | '已支取'
}

const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))
const recordId = ref('')
const record = ref<DepositRecord | null>(null)
const account = ref<Account | null>(null)
const loading = ref(true)
const actionMenuOpen = ref(false)

const moreIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="%23ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="5" cy="12" r="1.8"/><circle cx="12" cy="12" r="1.8"/><circle cx="19" cy="12" r="1.8"/></svg>'
const closeIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="%23ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18"/><path d="M6 6l12 12"/></svg>'

onLoad((q) => {
  const query = (q || {}) as any
  recordId.value = String(query.id || '')
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

function load() {
  loading.value = true
  fetchDetail()
}

async function fetchDetail() {
  try {
    const res = await getDepositRecord(recordId.value)
    const found = res?.record || null
    record.value = found
    if (found?.accountId) {
      const accRes = await getDepositAccount(String(found.accountId))
      const acc = accRes?.account
      account.value = acc
        ? {
            id: String(acc.id || ''),
            bank: String(acc.bank || ''),
            branch: String(acc.branch || ''),
            accountNo: String(acc.accountNo || ''),
            holder: String(acc.holder || ''),
            avatarUrl: String(acc.avatarUrl || ''),
            note: String(acc.note || ''),
          }
        : null
    } else {
      account.value = null
    }
  } catch (e: any) {
    record.value = null
    account.value = null
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

const currencyMeta = computed(() => getCurrencyMeta(record.value?.currency || 'CNY'))
const canWithdraw = computed(() => record.value?.status === '未到期')
const hasActions = computed(() => !!record.value)
const fabToggleStyle = computed(() => {
  const base = normalizeHexColor(getThemeBaseColor()) || '#111111'
  const { r, g, b } = hexToRgb(base)
  const alpha = actionMenuOpen.value ? 0.58 : 0.32
  return { backgroundColor: `rgba(${r}, ${g}, ${b}, ${alpha})` }
})

const attachments = computed<AttachmentItem[]>(() => {
  if (!record.value) return []
  const raw = record.value.attachments || []
  if (Array.isArray(raw) && raw.length && typeof raw[0] === 'string') {
    return (raw as string[]).map((url) => ({ type: 'image', url }))
  }
  return (raw as AttachmentItem[]) || []
})

const amountUpperDisplay = computed(() => {
  if (!record.value) return ''
  if (record.value.amountUpper) return record.value.amountUpper
  const amount = Number(record.value.amount)
  if (!Number.isFinite(amount) || amount <= 0) return ''
  const meta = getCurrencyMeta(record.value.currency)
  if (meta.code !== 'CNY') return `${meta.symbol}${formatAmount(amount)}`
  return formatCurrencyUpper(amount)
})

function tailOf(accountNo: string): string {
  const v = String(accountNo || '').replace(/\s+/g, '')
  if (!v) return ''
  return v.slice(-4)
}

function initialOf(name: string): string {
  const v = String(name || '').trim()
  return v ? v.slice(0, 1) : '存'
}

function termLabel(unit: string): string {
  if (unit === 'year') return '年'
  if (unit === 'month') return '月'
  return '天'
}

function formatAmount(amount: number): string {
  const n = Number(amount)
  if (!Number.isFinite(n)) return '0.00'
  return n.toFixed(2)
}

function formatDate(date: Date): string {
  const y = date.getFullYear()
  const m = `${date.getMonth() + 1}`.padStart(2, '0')
  const d = `${date.getDate()}`.padStart(2, '0')
  return `${y}-${m}-${d}`
}

function hexToRgb(hex: string): { r: number; g: number; b: number } {
  const normalized = normalizeHexColor(hex) || '#111111'
  return {
    r: Number.parseInt(normalized.slice(1, 3), 16),
    g: Number.parseInt(normalized.slice(3, 5), 16),
    b: Number.parseInt(normalized.slice(5, 7), 16),
  }
}

function toggleActionMenu() {
  actionMenuOpen.value = !actionMenuOpen.value
}

function closeActionMenu() {
  actionMenuOpen.value = false
}

async function withdrawRecord() {
  if (!record.value) return
  if (!canWithdraw.value) {
    uni.showToast({ title: '只能支取未到期存款', icon: 'none' })
    return
  }
  const res = await new Promise<UniApp.ShowModalRes>((resolve) => {
    uni.showModal({ title: '确认支取', content: '确认支取这笔存款？', success: resolve })
  })
  if (!res.confirm) return
  try {
    const nextDate = formatDate(new Date())
    const res = await updateDepositRecord(String(record.value.id), {
      status: '已支取',
      withdrawnAt: nextDate,
      endDate: nextDate,
    })
    if (res?.record) {
      record.value = { ...record.value, ...res.record }
    } else {
      record.value = { ...record.value, status: '已支取', endDate: nextDate, withdrawnAt: nextDate }
    }
    uni.showToast({ title: '已支取', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '支取失败', icon: 'none' })
  }
  closeActionMenu()
}

function attachmentKey(item: AttachmentItem, idx: number): string {
  return item.url ? `${item.type}-${item.url}` : `${item.type}-${idx}`
}

function formatCurrencyUpper(amount: number): string {
  const CN_NUM = ['零', '壹', '贰', '叁', '肆', '伍', '陆', '柒', '捌', '玖']
  const CN_UNIT = ['', '拾', '佰', '仟']
  const CN_SECTION = ['', '万', '亿', '兆']

  let integer = Math.floor(amount)
  let decimal = Math.round((amount - integer) * 100)
  let chinese = ''
  let sectionPos = 0
  let needZero = false

  if (integer === 0) {
    chinese = '零'
  } else {
    while (integer > 0) {
      const section = integer % 10000
      if (section !== 0) {
        let sectionStr = ''
        let unitPos = 0
        let zero = true
        let sectionCopy = section
        while (sectionCopy > 0) {
          const digit = sectionCopy % 10
          if (digit === 0) {
            if (!zero) {
              zero = true
              sectionStr = CN_NUM[0] + sectionStr
            }
          } else {
            zero = false
            sectionStr = CN_NUM[digit] + CN_UNIT[unitPos] + sectionStr
          }
          unitPos += 1
          sectionCopy = Math.floor(sectionCopy / 10)
        }
        chinese = sectionStr + CN_SECTION[sectionPos] + chinese
        needZero = true
      } else if (needZero && !chinese.startsWith(CN_NUM[0])) {
        chinese = CN_NUM[0] + chinese
      }
      sectionPos += 1
      integer = Math.floor(integer / 10000)
    }
  }

  chinese += '元'

  if (decimal === 0) {
    chinese += '整'
  } else {
    const jiao = Math.floor(decimal / 10)
    const fen = decimal % 10
    if (jiao > 0) chinese += CN_NUM[jiao] + '角'
    if (jiao === 0 && fen > 0) chinese += CN_NUM[0]
    if (fen > 0) chinese += CN_NUM[fen] + '分'
  }

  return chinese
}
</script>

<style scoped>
.page {
  padding: 24rpx;
  padding-bottom: calc(150rpx + env(safe-area-inset-bottom));
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}
.card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 8rpx 22rpx rgba(0, 0, 0, 0.06);
}
.section-title {
  font-size: 28rpx;
  font-weight: 600;
  margin-bottom: 12rpx;
}
.account-card {
  display: flex;
  align-items: center;
  gap: 16rpx;
}
.account-avatar {
  width: 96rpx;
  height: 96rpx;
  border-radius: 48rpx;
  overflow: hidden;
  background: #fff;
  flex: none;
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
.account-info {
  flex: 1;
  min-width: 0;
}
.account-name {
  font-size: 30rpx;
  font-weight: 600;
}
.account-sub {
  margin-top: 6rpx;
  color: #666;
  font-size: 24rpx;
}
.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  padding: 10rpx 0;
  border-bottom: 1rpx solid #f0f0f5;
}
.info-row:last-child {
  border-bottom: none;
}
.label {
  color: #666;
  font-size: 24rpx;
}
.value {
  font-size: 26rpx;
  font-weight: 600;
  color: #111;
}
.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10rpx;
}
.tag-chip {
  padding: 0 12rpx;
  height: 42rpx;
  line-height: 42rpx;
  border-radius: 999rpx;
  background: #f1f2f6;
  color: #555;
  font-size: 22rpx;
}
.attachment-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
}
.attachment {
  border-radius: 12rpx;
  overflow: hidden;
  background: #f6f7fb;
  height: 180rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}
.attachment-img {
  width: 100%;
  height: 100%;
}
.file-card {
  width: 100%;
  height: 100%;
  padding: 12rpx;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 10rpx;
  color: #666;
  font-size: 22rpx;
  text-align: center;
  box-sizing: border-box;
}
.file-icon {
  width: 56rpx;
  height: 56rpx;
  border-radius: 12rpx;
  background: #e9ecf5;
  color: #666;
  font-size: 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}
.file-name {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.note {
  color: #666;
  font-size: 26rpx;
  line-height: 1.5;
}
.hint {
  color: #666;
  font-size: 26rpx;
}
.fab-mask {
  position: fixed;
  left: 0;
  right: 0;
  top: 0;
  bottom: 0;
  z-index: 1200;
}
.fab {
  position: fixed;
  right: 24rpx;
  bottom: calc(28rpx + env(safe-area-inset-bottom));
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12rpx;
  z-index: 1201;
}
.fab-panel {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0;
  padding: 6rpx 0;
  border-radius: 14rpx;
  background: #fff;
  border: 1rpx solid rgba(0, 0, 0, 0.06);
  box-shadow: 0 10rpx 26rpx rgba(0, 0, 0, 0.12);
  transform: translateY(10rpx);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.fab-panel.open {
  transform: translateY(0);
  opacity: 1;
  pointer-events: auto;
}
.fab-panel .action-btn {
  width: 200rpx;
  text-align: center;
}
.fab-panel .action-btn.disabled {
  color: #999;
}
.fab-toggle {
  width: 70rpx;
  height: 70rpx;
  border-radius: 999rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid rgba(255, 255, 255, 0.24);
  box-shadow: 0 10rpx 24rpx rgba(0, 0, 0, 0.16);
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}
.fab-toggle::after {
  border: none;
}
.fab-toggle.active {
  border-color: rgba(255, 255, 255, 0.22);
  box-shadow: 0 12rpx 26rpx rgba(0, 0, 0, 0.2);
}
.fab-toggle:active {
  transform: scale(0.98);
}
.fab-icon {
  width: 28rpx;
  height: 28rpx;
}
.action-btn {
  width: 100%;
  height: 68rpx;
  line-height: 68rpx;
  padding: 0;
  font-size: 26rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  color: #111;
  border-radius: 0;
}
.action-btn::after {
  border: none;
}
</style>
