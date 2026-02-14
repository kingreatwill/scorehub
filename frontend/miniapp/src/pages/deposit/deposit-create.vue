<template>
  <view class="page" :style="themeStyle">
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

    <view class="card">
      <view class="section-title">存款信息</view>
      <view class="field">
        <text class="label">币种</text>
        <view class="field-body">
          <view class="chip-group">
            <view
              v-for="item in currencyList"
              :key="item.code"
              class="chip"
              :class="{ active: form.currency === item.code }"
              @click="setCurrency(item.code)"
            >
              {{ item.label }}
            </view>
          </view>
        </view>
      </view>
      <view class="field">
        <text class="label">存款金额</text>
        <view class="field-body">
          <input class="input" :value="form.amount" type="digit" placeholder="" @input="onAmountInput" />
          <text class="assist" v-if="amountUpper">大写：{{ amountUpper }}</text>
          <view class="history-row" v-if="amountHistory.length">
            <text class="history-label">历史</text>
            <view class="history-chips">
              <view
                class="history-chip"
                v-for="item in amountHistory"
                :key="`amount-${item}`"
                @click="applyAmountHistory(item)"
                @longpress="removeAmountHistory(item)"
              >
                {{ item }}
              </view>
            </view>
            <text class="history-clear" @click="confirmClearHistory('amount')">清空</text>
          </view>
        </view>
      </view>
      <view class="field">
        <text class="label">存期</text>
        <view class="field-body">
          <view class="term-row">
            <input class="input" :value="form.termValue" type="number" placeholder="" @input="onTermInput" />
            <view class="chip-group">
              <view class="chip" :class="{ active: form.termUnit === 'year' }" @click="setTermUnit('year')">年</view>
              <view class="chip" :class="{ active: form.termUnit === 'month' }" @click="setTermUnit('month')">月</view>
            </view>
          </view>
          <view class="history-row" v-if="termHistory.length">
            <text class="history-label">历史</text>
            <view class="history-chips">
              <view
                class="history-chip"
                v-for="item in termHistory"
                :key="`term-${item.value}-${item.unit}`"
                @click="applyTermHistory(item)"
                @longpress="removeTermHistory(item)"
              >
                {{ item.value }}{{ termLabel(item.unit) }}
              </view>
            </view>
            <text class="history-clear" @click="confirmClearHistory('term')">清空</text>
          </view>
        </view>
      </view>
      <view class="field">
        <text class="label">年化利率</text>
        <view class="field-body">
          <view class="rate-row">
            <input class="input rate-input" :value="form.rate" type="digit" placeholder="" @input="onRateInput" />
            <text class="rate-unit">%</text>
          </view>
          <view class="history-row" v-if="rateHistory.length">
            <text class="history-label">历史</text>
            <view class="history-chips">
              <view
                class="history-chip"
                v-for="item in rateHistory"
                :key="`rate-${item}`"
                @click="applyRateHistory(item)"
                @longpress="removeRateHistory(item)"
              >
                {{ item }}%
              </view>
            </view>
            <text class="history-clear" @click="confirmClearHistory('rate')">清空</text>
          </view>
        </view>
      </view>
      <view class="field">
        <text class="label">单据号</text>
        <view class="field-body">
          <input class="input" v-model="form.receiptNo" placeholder="（可选）" />
        </view>
      </view>
      <view class="field">
        <text class="label">起息日</text>
        <view class="field-body">
          <picker mode="date" :value="form.startDate" @change="onStartDateChange">
            <view class="picker-value">{{ form.startDate || '选择日期' }}</view>
          </picker>
        </view>
      </view>
      <view class="field">
        <text class="label">到期日</text>
        <view class="field-body">
          <picker mode="date" :value="form.endDate" @change="onEndDateChange">
            <view class="picker-value">{{ form.endDate || '自动计算' }}</view>
          </picker>
          <view class="assist-row">
            <text class="assist">{{ endDateManual ? '已手动修正' : '自动计算' }}</text>
            <text class="assist-link" v-if="endDateManual" @click="resetEndDate">重算</text>
          </view>
        </view>
      </view>
      <view class="field">
        <text class="label">到期利息</text>
        <view class="field-body">
          <input class="input" :value="interestInput" type="digit" placeholder="自动计算" @input="onInterestInput" />
          <text class="assist" v-if="interestHint">{{ interestHint }}</text>
        </view>
      </view>
    </view>

    <view class="card">
      <view class="section-title">标签</view>
      <view class="tag-input">
        <view class="tag-chip" v-for="(tag, idx) in form.tags" :key="`${tag}-${idx}`">
          <text class="tag-text">{{ tag }}</text>
          <text class="tag-remove" @click="removeTag(idx)">×</text>
        </view>
        <input
          class="tag-input-field"
          v-model="tagInput"
          placeholder="回车添加标签"
          confirm-type="done"
          @confirm="onTagConfirm"
        />
      </view>
      <view class="history-row tag-history" v-if="tagHistory.length">
        <text class="history-label">历史</text>
        <view class="history-chips">
          <view
            class="history-chip"
            v-for="tag in tagHistory"
            :key="`tag-${tag}`"
            @click="applyTagHistory(tag)"
            @longpress="removeTagHistory(tag)"
          >
            {{ tag }}
          </view>
        </view>
        <text class="history-clear" @click="confirmClearHistory('tag')">清空</text>
      </view>
    </view>

    <view class="card">
      <view class="section-title">存单照片/附件</view>
      <view class="attachment-grid">
        <view class="attachment" v-for="(item, idx) in form.attachments" :key="attachmentKey(item, idx)">
          <template v-if="item.type === 'image'">
            <image class="attachment-img" :src="item.url" mode="aspectFill" />
          </template>
          <template v-else>
            <view class="file-card">
              <view class="file-icon">FILE</view>
              <view class="file-name">{{ item.name || '附件' }}</view>
            </view>
          </template>
          <view class="attachment-remove" @click="removeAttachment(idx)">×</view>
        </view>
        <view class="attachment add" @click="addAttachment">
          <text class="add-plus">+</text>
        </view>
      </view>
    </view>

    <view class="card">
      <view class="section-title">备注</view>
      <textarea class="textarea" v-model="form.note" placeholder="（可选）" />
    </view>

    <button class="btn confirm-btn" :disabled="saving" @click="onSave">{{ saving ? '保存中…' : '保存' }}</button>
  </view>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor } from '../../utils/theme'
import { avatarStyle } from '../../utils/avatar-color'
import { currencyList, getCurrencyMeta } from '../../utils/currency'
import { createDepositRecord, getDepositAccount } from '../../utils/api'

type Account = {
  id: string
  bank: string
  branch: string
  accountNo: string
  holder: string
  avatarUrl: string
  note: string
}

type DepositForm = {
  currency: string
  amount: string
  termValue: string
  termUnit: 'year' | 'month'
  rate: string
  receiptNo: string
  startDate: string
  endDate: string
  tags: string[]
  note: string
  attachments: AttachmentItem[]
}

type AttachmentItem = {
  type: 'image' | 'file'
  url: string
  name?: string
}

type TermHistoryItem = {
  value: string
  unit: 'year' | 'month'
}

const account = ref<Account | null>(null)
const accountId = ref('')
const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))
const saving = ref(false)
const endDateManual = ref(false)
const tagInput = ref('')
const amountHistory = ref<string[]>([])
const termHistory = ref<TermHistoryItem[]>([])
const rateHistory = ref<string[]>([])
const tagHistory = ref<string[]>([])

const form = ref<DepositForm>({
  currency: 'CNY',
  amount: '',
  termValue: '',
  termUnit: 'year',
  rate: '',
  receiptNo: '',
  startDate: todayString(),
  endDate: '',
  tags: [],
  note: '',
  attachments: [],
})

const amountUpper = computed(() => amountToChineseUpper(parseAmount(form.value.amount), form.value.currency))
const interestInput = ref('')
const interestManual = ref(false)
const calculatedInterest = computed(() =>
  calcInterest(parseAmount(form.value.amount), parseRate(form.value.rate), parseTerm(form.value.termValue), form.value.termUnit),
)
const interestValue = computed(() => {
  if (!interestManual.value && !interestInput.value) return calculatedInterest.value
  return parseAmount(interestInput.value)
})
const interestHint = computed(() => {
  const term = parseTerm(form.value.termValue)
  if (!term) return ''
  const unit = form.value.termUnit === 'year' ? '年' : '个月'
  return `按 ${term} ${unit}计息（年化，可修改）`
})

onLoad((q) => {
  const query = (q || {}) as any
  accountId.value = String(query.accountId || '')
})

onShow(async () => {
  syncTheme()
  loadHistory()
  await loadAccount()
  if (!form.value.endDate) {
    form.value.endDate = calcEndDate(form.value.startDate, form.value.termValue, form.value.termUnit)
  }
})

watch(
  [() => form.value.startDate, () => form.value.termValue, () => form.value.termUnit],
  () => {
    if (endDateManual.value) return
    form.value.endDate = calcEndDate(form.value.startDate, form.value.termValue, form.value.termUnit)
  },
)

watch(
  calculatedInterest,
  (next) => {
    if (interestManual.value) return
    interestInput.value = formatAmount(next)
  },
  { immediate: true },
)

function syncTheme() {
  const base = getThemeBaseColor()
  themeStyle.value = buildThemeVars(base)
  applyNavigationBarTheme(base)
  applyTabBarTheme(base)
}

async function loadAccount() {
  if (!accountId.value) return
  try {
    const res = await getDepositAccount(accountId.value)
    const found = res?.account
    if (!found) {
      throw new Error('未找到账户')
    }
    account.value = {
      id: String(found.id || ''),
      bank: String(found.bank || ''),
      branch: String(found.branch || ''),
      accountNo: String(found.accountNo || ''),
      holder: String(found.holder || ''),
      avatarUrl: String(found.avatarUrl || ''),
      note: String(found.note || ''),
    }
  } catch (e: any) {
    uni.showToast({ title: e?.message || '未找到账户', icon: 'none' })
    setTimeout(() => {
      uni.navigateBack({ delta: 1 })
    }, 400)
  }
}

function setCurrency(code: string) {
  form.value.currency = code
}

function setTermUnit(unit: 'year' | 'month') {
  form.value.termUnit = unit
}

function onStartDateChange(e: any) {
  form.value.startDate = String(e?.detail?.value || '')
}

function onEndDateChange(e: any) {
  form.value.endDate = String(e?.detail?.value || '')
  endDateManual.value = true
}

function resetEndDate() {
  endDateManual.value = false
  form.value.endDate = calcEndDate(form.value.startDate, form.value.termValue, form.value.termUnit)
}

function onAmountInput(e: any) {
  const next = sanitizeDecimal(String(e?.detail?.value || ''))
  form.value.amount = next
  return next
}

function onRateInput(e: any) {
  const next = sanitizeDecimal(String(e?.detail?.value || ''))
  form.value.rate = next
  return next
}

function onInterestInput(e: any) {
  const next = sanitizeDecimal(String(e?.detail?.value || ''))
  if (!next) {
    interestManual.value = false
    interestInput.value = formatAmount(calculatedInterest.value)
    return interestInput.value
  }
  interestManual.value = true
  interestInput.value = next
  return next
}

function onTermInput(e: any) {
  const next = String(e?.detail?.value || '').replace(/[^0-9]/g, '')
  form.value.termValue = next
  return next
}

function applyAmountHistory(value: string) {
  form.value.amount = String(value || '')
}

function removeAmountHistory(value: string) {
  amountHistory.value = amountHistory.value.filter((item) => item !== value)
  uni.setStorageSync('depositAmountHistory', amountHistory.value)
}

function applyTermHistory(item: TermHistoryItem) {
  if (!item?.value) return
  form.value.termValue = String(item.value)
  form.value.termUnit = item.unit === 'month' ? 'month' : 'year'
}

function removeTermHistory(item: TermHistoryItem) {
  termHistory.value = termHistory.value.filter((it) => it.value !== item.value || it.unit !== item.unit)
  uni.setStorageSync('depositTermHistory', termHistory.value)
}

function applyRateHistory(value: string) {
  form.value.rate = String(value || '')
}

function removeRateHistory(value: string) {
  rateHistory.value = rateHistory.value.filter((item) => item !== value)
  uni.setStorageSync('depositRateHistory', rateHistory.value)
}

function applyTagHistory(tag: string) {
  addTag(tag)
}

function removeTagHistory(tag: string) {
  tagHistory.value = tagHistory.value.filter((item) => item !== tag)
  uni.setStorageSync('depositTagHistory', tagHistory.value)
}

async function addAttachment() {
  const actions = ['拍照', '选择图片', '选择附件']
  try {
    const res = await new Promise<any>((resolve, reject) => {
      uni.showActionSheet({ itemList: actions, success: resolve, fail: reject })
    })
    const index = Number(res?.tapIndex ?? -1)
    if (index === 0) {
      await chooseImages(['camera'])
    } else if (index === 1) {
      await chooseImages(['album'])
    } else if (index === 2) {
      await chooseFiles()
    }
  } catch (e) {}
}

function removeAttachment(idx: number) {
  form.value.attachments = form.value.attachments.filter((_, i) => i !== idx)
}

function onTagConfirm() {
  addTag(tagInput.value)
  tagInput.value = ''
}

function addTag(raw: string) {
  const tag = String(raw || '').trim()
  if (!tag) return
  if (form.value.tags.includes(tag)) return
  form.value.tags = [...form.value.tags, tag]
}

function removeTag(idx: number) {
  form.value.tags = form.value.tags.filter((_, i) => i !== idx)
}

function attachmentKey(item: AttachmentItem, idx: number): string {
  return item.url ? `${item.type}-${item.url}` : `${item.type}-${idx}`
}

async function chooseImages(sourceType: Array<'camera' | 'album'>) {
  const res = await new Promise<any>((resolve, reject) => {
    uni.chooseImage({ count: 6, sourceType, success: resolve, fail: reject })
  })
  const files = (res?.tempFilePaths || res?.tempFiles || []).map((item: any) => (typeof item === 'string' ? item : item?.path))
  const next = files
    .filter(Boolean)
    .map((url: string) => ({ type: 'image' as const, url }))
  form.value.attachments = [...form.value.attachments, ...next].slice(0, 9)
}

async function chooseFiles() {
  const res = await new Promise<any>((resolve, reject) => {
    ;(uni as any).chooseMessageFile({ count: 6, type: 'file', success: resolve, fail: reject })
  })
  const files = (res?.tempFiles || []).map((file: any) => ({
    type: 'file' as const,
    url: String(file?.path || ''),
    name: String(file?.name || '附件'),
  }))
  const next = files.filter((file: AttachmentItem) => file.url)
  form.value.attachments = [...form.value.attachments, ...next].slice(0, 9)
}

async function onSave() {
  if (!account.value) return
  if (saving.value) return
  if (tagInput.value.trim()) {
    addTag(tagInput.value)
    tagInput.value = ''
  }
  if (!parseAmount(form.value.amount)) {
    uni.showToast({ title: '请输入存款金额', icon: 'none' })
    return
  }
  if (!parseTerm(form.value.termValue)) {
    uni.showToast({ title: '请输入存期', icon: 'none' })
    return
  }
  if (String(form.value.rate || '').trim() === '') {
    uni.showToast({ title: '请输入年化利率', icon: 'none' })
    return
  }
  if (!form.value.startDate) {
    uni.showToast({ title: '请选择起息日', icon: 'none' })
    return
  }
  if (!form.value.endDate) {
    uni.showToast({ title: '请选择到期日', icon: 'none' })
    return
  }

  saving.value = true
  try {
    await createDepositRecord(account.value.id, {
      currency: form.value.currency,
      amount: parseAmount(form.value.amount),
      amountUpper: amountUpper.value,
      termValue: parseTerm(form.value.termValue),
      termUnit: form.value.termUnit,
      rate: parseRate(form.value.rate),
      receiptNo: form.value.receiptNo.trim(),
      startDate: form.value.startDate,
      endDate: form.value.endDate,
      interest: interestValue.value,
      tags: [...form.value.tags],
      note: form.value.note.trim(),
      attachments: form.value.attachments,
      status: '未到期',
    })
    recordHistory()
    uni.showToast({ title: '已保存', icon: 'success' })
    setTimeout(() => {
      uni.navigateBack({ delta: 1 })
    }, 400)
  } finally {
    saving.value = false
  }
}

function termLabel(unit: 'year' | 'month'): string {
  return unit === 'month' ? '月' : '年'
}

function loadHistory() {
  amountHistory.value = normalizeHistoryList(uni.getStorageSync('depositAmountHistory'))
  rateHistory.value = normalizeHistoryList(uni.getStorageSync('depositRateHistory'))
  tagHistory.value = normalizeHistoryList(uni.getStorageSync('depositTagHistory'))
  const raw = uni.getStorageSync('depositTermHistory')
  if (Array.isArray(raw)) {
    termHistory.value = raw
      .map((item) => ({
        value: String(item?.value || '').replace(/[^0-9]/g, ''),
        unit: item?.unit === 'month' ? 'month' : 'year',
      }))
      .filter((item) => item.value)
  } else {
    termHistory.value = []
  }
}

function recordHistory() {
  const amount = formatAmount(parseAmount(form.value.amount))
  amountHistory.value = pushHistory(amountHistory.value, amount)
  rateHistory.value = pushHistory(rateHistory.value, String(parseRate(form.value.rate)))
  if (form.value.termValue) {
    termHistory.value = pushTermHistory(termHistory.value, {
      value: String(parseTerm(form.value.termValue)),
      unit: form.value.termUnit,
    })
  }
  const tags = form.value.tags.map((tag) => String(tag || '').trim()).filter(Boolean)
  if (tags.length) {
    for (const tag of tags) {
      tagHistory.value = pushHistory(tagHistory.value, tag)
    }
  }
  uni.setStorageSync('depositAmountHistory', amountHistory.value)
  uni.setStorageSync('depositRateHistory', rateHistory.value)
  uni.setStorageSync('depositTermHistory', termHistory.value)
  uni.setStorageSync('depositTagHistory', tagHistory.value)
}

function normalizeHistoryList(value: any): string[] {
  if (!Array.isArray(value)) return []
  return value.map((item) => String(item || '').trim()).filter(Boolean)
}

function pushHistory(list: string[], value: string, limit = 6): string[] {
  const v = String(value || '').trim()
  if (!v) return list
  const next = list.filter((item) => item !== v)
  next.unshift(v)
  return next.slice(0, limit)
}

function pushTermHistory(list: TermHistoryItem[], item: TermHistoryItem, limit = 6): TermHistoryItem[] {
  const v = String(item?.value || '').trim()
  if (!v) return list
  const unit = item.unit === 'month' ? 'month' : 'year'
  const next = list.filter((it) => it.value !== v || it.unit !== unit)
  next.unshift({ value: v, unit })
  return next.slice(0, limit)
}

async function confirmClearHistory(type: 'amount' | 'term' | 'rate' | 'tag') {
  const res = await new Promise<UniApp.ShowModalRes>((resolve) => {
    uni.showModal({ title: '清空历史', content: '确认清空历史记录？', success: resolve })
  })
  if (!res.confirm) return
  if (type === 'amount') {
    amountHistory.value = []
    uni.setStorageSync('depositAmountHistory', amountHistory.value)
    return
  }
  if (type === 'term') {
    termHistory.value = []
    uni.setStorageSync('depositTermHistory', termHistory.value)
    return
  }
  if (type === 'rate') {
    rateHistory.value = []
    uni.setStorageSync('depositRateHistory', rateHistory.value)
    return
  }
  tagHistory.value = []
  uni.setStorageSync('depositTagHistory', tagHistory.value)
}

function initialOf(name: string): string {
  const v = String(name || '').trim()
  return v ? v.slice(0, 1) : '存'
}

function tailOf(accountNo: string): string {
  const v = String(accountNo || '').replace(/\s+/g, '')
  if (!v) return ''
  return v.slice(-4)
}

function todayString(): string {
  return formatDate(new Date())
}

function formatDate(date: Date): string {
  const y = date.getFullYear()
  const m = `${date.getMonth() + 1}`.padStart(2, '0')
  const d = `${date.getDate()}`.padStart(2, '0')
  return `${y}-${m}-${d}`
}

function parseDate(value: string): Date | null {
  if (!value) return null
  const parts = String(value).split('-').map((v) => Number.parseInt(v, 10))
  if (parts.length < 3 || parts.some((p) => !Number.isFinite(p))) return null
  const [y, m, d] = parts
  return new Date(y, m - 1, d)
}

function addMonths(date: Date, months: number): Date {
  const d = new Date(date)
  const targetMonth = d.getMonth() + months
  const targetYear = d.getFullYear() + Math.floor(targetMonth / 12)
  const monthIndex = ((targetMonth % 12) + 12) % 12
  const day = d.getDate()
  const first = new Date(targetYear, monthIndex, 1)
  const lastDay = new Date(targetYear, monthIndex + 1, 0).getDate()
  return new Date(targetYear, monthIndex, Math.min(day, lastDay))
}

function calcEndDate(startDate: string, termValue: string, termUnit: 'year' | 'month'): string {
  const start = parseDate(startDate)
  const term = parseTerm(termValue)
  if (!start || !term) return ''
  if (termUnit === 'month') return formatDate(addMonths(start, term))
  return formatDate(addMonths(start, term * 12))
}

function calcInterest(amount: number, rate: number, termValue: number, termUnit: 'year' | 'month'): number {
  if (!amount || !rate || !termValue) return 0
  const years = termUnit === 'month' ? termValue / 12 : termValue
  return (amount * rate * years) / 100
}

function parseAmount(raw: string): number {
  const n = Number.parseFloat(String(raw || '').replace(/,/g, ''))
  return Number.isFinite(n) ? n : 0
}

function parseRate(raw: string): number {
  const n = Number.parseFloat(String(raw || '').replace(/,/g, ''))
  return Number.isFinite(n) ? n : 0
}

function parseTerm(raw: string): number {
  const n = Number.parseInt(String(raw || '').replace(/[^0-9]/g, ''), 10)
  return Number.isFinite(n) ? n : 0
}

function sanitizeDecimal(raw: string): string {
  const cleaned = raw.replace(/[^\d.]/g, '')
  const firstDot = cleaned.indexOf('.')
  if (firstDot === -1) return cleaned
  const head = cleaned.slice(0, firstDot + 1)
  const tail = cleaned.slice(firstDot + 1).replace(/\./g, '')
  return head + tail
}

function formatAmount(amount: number): string {
  const n = Number(amount)
  if (!Number.isFinite(n)) return '0.00'
  return n.toFixed(2)
}

function amountToChineseUpper(amount: number, currency: string): string {
  if (!amount || !Number.isFinite(amount)) return ''
  const meta = getCurrencyMeta(currency)
  if (meta.code !== 'CNY') {
    return `${meta.symbol}${formatAmount(amount)}`
  }
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
.account-card {
  display: flex;
  align-items: center;
  gap: 16rpx;
}
.account-avatar {
  width: 92rpx;
  height: 92rpx;
  border-radius: 46rpx;
  overflow: hidden;
  background: #fff;
  flex: none;
}
.avatar {
  width: 92rpx;
  height: 92rpx;
  border-radius: 46rpx;
}
.avatar-fallback {
  background: var(--brand-soft);
  color: var(--brand-solid);
  font-size: 30rpx;
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
.section-title {
  font-size: 28rpx;
  font-weight: 600;
  margin-bottom: 12rpx;
}
.field {
  display: flex;
  align-items: flex-start;
  gap: 16rpx;
  margin-bottom: 16rpx;
  flex-wrap: wrap;
}
.field:last-child {
  margin-bottom: 0;
}
.label {
  width: 140rpx;
  color: #666;
  font-size: 26rpx;
  padding-top: 8rpx;
  flex: none;
}
.field-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}
.input {
  width: 100%;
  background: #f6f7fb;
  border-radius: 12rpx;
  height: 72rpx;
  line-height: 72rpx;
  padding: 0 16rpx;
  font-size: 28rpx;
  box-sizing: border-box;
}
.rate-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 10rpx;
}
.rate-input {
  flex: 1;
  min-width: 0;
}
.rate-unit {
  height: 72rpx;
  line-height: 72rpx;
  font-size: 26rpx;
  color: #666;
}
.term-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 12rpx;
}
.term-input {
  flex: 1;
  min-width: 0;
}
.chip-group {
  display: flex;
  align-items: center;
  gap: 8rpx;
  flex-wrap: wrap;
}
.chip {
  padding: 0 18rpx;
  height: 48rpx;
  line-height: 48rpx;
  border-radius: 999rpx;
  background: #f1f2f6;
  color: #666;
  font-size: 24rpx;
}
.chip.active {
  background: var(--brand-solid);
  color: #fff;
}
.picker-value {
  width: 100%;
  background: #f6f7fb;
  border-radius: 12rpx;
  height: 72rpx;
  line-height: 72rpx;
  padding: 0 16rpx;
  font-size: 28rpx;
  color: #111;
  box-sizing: border-box;
}
.value {
  font-size: 28rpx;
  font-weight: 600;
}
.assist {
  color: #888;
  font-size: 22rpx;
}
.history-row {
  display: flex;
  align-items: center;
  gap: 10rpx;
  flex-wrap: wrap;
}
.history-label {
  color: #999;
  font-size: 22rpx;
}
.history-chips {
  display: flex;
  align-items: center;
  gap: 8rpx;
  flex-wrap: wrap;
}
.history-chip {
  padding: 0 14rpx;
  height: 36rpx;
  line-height: 36rpx;
  border-radius: 999rpx;
  background: #f1f2f6;
  color: #666;
  font-size: 22rpx;
}
.tag-history {
  margin-top: 8rpx;
}
.history-clear {
  margin-left: auto;
  color: var(--brand-solid);
  font-size: 22rpx;
}
.assist-row {
  display: flex;
  align-items: center;
  gap: 10rpx;
}
.assist-link {
  color: var(--brand-solid);
  font-size: 22rpx;
}
.attachment-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
}
.attachment {
  position: relative;
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
.tag-input {
  display: flex;
  flex-wrap: wrap;
  gap: 10rpx;
  padding: 12rpx;
  border-radius: 12rpx;
  background: #f6f7fb;
}
.tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 6rpx;
  padding: 0 12rpx;
  height: 42rpx;
  line-height: 42rpx;
  border-radius: 999rpx;
  background: #fff;
  color: #555;
  font-size: 22rpx;
}
.tag-text {
  line-height: 1;
}
.tag-remove {
  font-size: 22rpx;
  color: #999;
}
.tag-input-field {
  flex: 1;
  min-width: 160rpx;
  height: 42rpx;
  line-height: 42rpx;
  font-size: 24rpx;
  color: #111;
  background: transparent;
  padding: 0 6rpx;
  box-sizing: border-box;
}
.attachment.add {
  border: 2rpx dashed #d7d7e0;
  color: #888;
}
.add-plus {
  font-size: 44rpx;
  font-weight: 600;
}
.attachment-remove {
  position: absolute;
  right: 8rpx;
  top: 8rpx;
  width: 36rpx;
  height: 36rpx;
  border-radius: 18rpx;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24rpx;
}
.textarea {
  width: 100%;
  min-height: 160rpx;
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 16rpx;
  font-size: 28rpx;
  box-sizing: border-box;
}
.btn {
  width: 100%;
  margin-top: 8rpx;
}
</style>
