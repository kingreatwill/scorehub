<template>
  <view class="page" v-if="loading" :style="themeStyle">
    <view class="card">
      <t-loading :loading="true" text="加载中…" />
    </view>
  </view>
  <view class="page" v-else :style="themeStyle">
    <view class="card profile-card">
      <template v-if="isMpWeixin">
        <button class="avatar-wrap avatar-button" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
          <image v-if="form.avatarUrl" class="avatar" :src="form.avatarUrl" mode="aspectFill" />
          <view v-else class="avatar avatar-fallback" :style="avatarStyle(form.name)">{{ initial }}</view>
        </button>
      </template>
      <template v-else>
        <view class="avatar-wrap">
          <image v-if="form.avatarUrl" class="avatar" :src="form.avatarUrl" mode="aspectFill" />
          <view v-else class="avatar avatar-fallback" :style="avatarStyle(form.name)">{{ initial }}</view>
        </view>
      </template>
      <view class="profile-info">
        <view class="name">{{ form.name || '联系人' }}</view>
        <view class="sub" v-if="form.relation">{{ form.relation }}</view>
      </view>
    </view>

    <view class="card">
      <view class="section-title">基础信息</view>
      <view class="field">
        <text class="label">姓名</text>
        <input class="input" v-model="form.name" placeholder="例如 小北" />
      </view>
      <view class="field">
        <text class="label">性别</text>
        <view class="gender-select">
          <view class="gender-option" :class="{ active: form.gender === '男' }" @click="form.gender = '男'">
            <view class="gender-radio" />
            <view class="gender-label">男</view>
          </view>
          <view class="gender-option" :class="{ active: form.gender === '女' }" @click="form.gender = '女'">
            <view class="gender-radio" />
            <view class="gender-label">女</view>
          </view>
        </view>
      </view>
      <view class="field">
        <text class="label">手机</text>
        <input class="input" v-model="form.phone" placeholder="手机号（可选）" />
      </view>
      <view class="field">
        <text class="label">关系</text>
        <input class="input" v-model="form.relation" placeholder="家人 / 朋友 / 同事" />
        <view class="chips">
          <view
            v-for="opt in relationOptions"
            :key="opt"
            class="chip"
            :class="{ active: form.relation === opt }"
            @click="form.relation = opt"
          >
            {{ opt }}
          </view>
        </view>
      </view>
    </view>

    <view class="card">
      <view class="section-title">生日信息</view>
      <view class="field">
        <view class="picker-value" @click="openPicker">{{ primaryDisplay || '选择日期' }}</view>
        <text class="assist" v-if="secondaryDisplay">对应{{ secondaryLabel }}：{{ secondaryDisplay }}</text>
      </view>
      <view class="field">
        <text class="hint">当前选择：{{ primaryLabel }} · {{ primaryBadge }}</text>
        <text class="assist" v-if="ageLabel">距{{ ageLabel }}岁生日</text>
      </view>
    </view>

    <view class="card">
      <view class="section-title">备注</view>
      <textarea class="textarea" v-model="form.note" placeholder="备注 / 喜好" />
    </view>

    <button class="btn confirm-btn" :disabled="saving" @click="onSave">{{ saving ? '保存中…' : '保存' }}</button>

    <BirthdayCalendar
      :visible="pickerVisible"
      :type="pickerType"
      :year="pickerYear"
      :month="pickerMonth"
      :day="pickerDay"
      @update:visible="pickerVisible = $event"
      @confirm="onCalendarConfirm"
    />
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { getBirthday, updateBirthday } from '../../utils/api'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor } from '../../utils/theme'
import lunarCalendar from '../../utils/lunar-calendar.mjs'
import BirthdayCalendar from './calendar.vue'
import { avatarStyle } from '../../utils/avatar-color'

type BirthdayForm = {
  id: string
  name: string
  gender: string
  phone: string
  relation: string
  note: string
  avatarUrl: string
  solarBirthday: string
  lunarBirthday: string
  primaryType: 'solar' | 'lunar'
}

const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))
const form = ref<BirthdayForm>({
  id: '',
  name: '',
  gender: '',
  phone: '',
  relation: '',
  note: '',
  avatarUrl: '',
  solarBirthday: '',
  lunarBirthday: '',
  primaryType: 'solar',
})

const relationOptions = ['家人', '亲属', '同事', '朋友', '恋人', '同学']

const initial = computed(() => String(form.value.name || '').trim().slice(0, 1) || '友')
const isMpWeixin = ref(false)
// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const primaryLabel = computed(() => (form.value.primaryType === 'lunar' ? '农历' : '公历'))

const primaryBadge = computed(() => {
  const base = form.value.primaryType === 'lunar' ? form.value.lunarBirthday : form.value.solarBirthday
  if (!base) return '未设置'
  const mmdd = toMonthDay(base)
  const daysLeft = calcDaysLeft(mmdd)
  return daysLeft === 0 ? '今天' : daysLeft === 1 ? '明天' : daysLeft === 2 ? '后天' : `还有${daysLeft}天`
})

const ageLabel = computed(() => {
  const base = form.value.primaryType === 'lunar' ? form.value.lunarBirthday : form.value.solarBirthday
  const birthYear = extractYear(base)
  if (!birthYear) return ''
  const mmdd = toMonthDay(base)
  const nextYear = nextBirthdayYear(mmdd)
  if (!nextYear) return ''
  const age = nextYear - birthYear
  if (age <= 0 || !Number.isFinite(age)) return ''
  return String(age)
})

const primaryDisplay = computed(() => {
  if (form.value.primaryType === 'lunar') {
    return form.value.lunarBirthday ? `农历 ${formatDisplayLunar(form.value.lunarBirthday)}` : ''
  }
  return form.value.solarBirthday ? `公历 ${formatDisplaySolar(form.value.solarBirthday)}` : ''
})

const secondaryLabel = computed(() => (form.value.primaryType === 'lunar' ? '公历' : '农历'))

const secondaryDisplay = computed(() => {
  if (form.value.primaryType === 'lunar') {
    if (!form.value.solarBirthday) return ''
    return formatDisplaySolar(form.value.solarBirthday)
  }
  if (!form.value.lunarBirthday) return ''
  return formatDisplayLunar(form.value.lunarBirthday)
})

const pickerVisible = ref(false)
const pickerType = ref<'solar' | 'lunar'>('solar')
const pickerYear = ref(new Date().getFullYear())
const pickerMonth = ref(new Date().getMonth() + 1)
const pickerDay = ref(new Date().getDate())
const saving = ref(false)
const loading = ref(false)

onShow(() => {
  syncTheme()
})

onLoad((options) => {
  const id = String((options as any)?.id || '').trim()
  loadDetail(id)
})

function syncTheme() {
  const base = getThemeBaseColor()
  themeStyle.value = buildThemeVars(base)
  applyNavigationBarTheme(base)
  applyTabBarTheme(base)
}

async function loadDetail(id: string) {
  if (!id) {
    uni.showToast({ title: '缺少参数', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 300)
    return
  }
  loading.value = true
  try {
    const res = await getBirthday(id)
    const data = res?.birthday || {}
    form.value = {
      id: String(data.id || id),
      name: String(data.name || ''),
      gender: String(data.gender || ''),
      phone: String(data.phone || ''),
      relation: String(data.relation || ''),
      note: String(data.note || ''),
      avatarUrl: String(data.avatarUrl || ''),
      solarBirthday: String(data.solarBirthday || ''),
      lunarBirthday: String(data.lunarBirthday || ''),
      primaryType: data.primaryType === 'lunar' ? 'lunar' : 'solar',
    }
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 300)
  } finally {
    loading.value = false
  }
}

function openPicker() {
  pickerType.value = form.value.primaryType || 'solar'
  hydratePickerFromForm()
  pickerVisible.value = true
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
    form.value.avatarUrl = dataUrl
  } catch (err) {
    uni.showToast({ title: '头像处理失败', icon: 'none' })
  }
  // #endif
}

function hydratePickerFromForm() {
  const base = pickerType.value === 'lunar' ? form.value.lunarBirthday : form.value.solarBirthday
  if (!base) return
  const parts = base.split('-')
  if (parts.length === 3) {
    pickerYear.value = Number(parts[0]) || pickerYear.value
    pickerMonth.value = Number(parts[1]) || pickerMonth.value
    pickerDay.value = Number(parts[2]) || pickerDay.value
    return
  }
  if (parts.length === 2) {
    pickerMonth.value = Number(parts[0]) || pickerMonth.value
    pickerDay.value = Number(parts[1]) || pickerDay.value
  }
}

function onCalendarConfirm(payload: { time: string; timeObject: any; isLunar: boolean }) {
  const solar = parseYMD(payload.time)
  const lunarYear = Number(payload.timeObject?.lYear || solar.year)
  const lunarMonth = Number(payload.timeObject?.lMonth || solar.month)
  const lunarDay = Number(payload.timeObject?.lDay || solar.day)
  const solarFull = formatYMD(solar.year, solar.month, solar.day)
  const lunarFull = formatYMD(lunarYear, lunarMonth, lunarDay)

  pickerType.value = payload.isLunar ? 'lunar' : 'solar'
  if (payload.isLunar) {
    pickerYear.value = lunarYear
    pickerMonth.value = lunarMonth
    pickerDay.value = lunarDay
  } else {
    pickerYear.value = solar.year
    pickerMonth.value = solar.month
    pickerDay.value = solar.day
  }

  form.value = {
    ...form.value,
    solarBirthday: solarFull,
    lunarBirthday: lunarFull,
    primaryType: payload.isLunar ? 'lunar' : 'solar',
  }
}

async function onSave() {
  if (saving.value) return
  const name = String(form.value.name || '').trim()
  if (!name) {
    uni.showToast({ title: '请填写姓名', icon: 'none' })
    return
  }
  const primaryType = form.value.primaryType === 'lunar' ? 'lunar' : 'solar'
  if (primaryType === 'solar' && !String(form.value.solarBirthday || '').trim()) {
    uni.showToast({ title: '请选择公历生日', icon: 'none' })
    return
  }
  if (primaryType === 'lunar' && !String(form.value.lunarBirthday || '').trim()) {
    uni.showToast({ title: '请选择农历生日', icon: 'none' })
    return
  }
  if (!form.value.id) {
    uni.showToast({ title: '缺少记录', icon: 'none' })
    return
  }

  const { primaryMonth, primaryDay, primaryYear } = resolvePrimaryPayload(primaryType)
  saving.value = true
  try {
    await updateBirthday(form.value.id, {
      name,
      gender: String(form.value.gender || '').trim(),
      phone: String(form.value.phone || '').trim(),
      relation: String(form.value.relation || '').trim(),
      note: String(form.value.note || '').trim(),
      avatarUrl: String(form.value.avatarUrl || '').trim(),
      solarBirthday: String(form.value.solarBirthday || '').trim(),
      lunarBirthday: String(form.value.lunarBirthday || '').trim(),
      primaryType,
      primaryMonth,
      primaryDay,
      primaryYear,
    })
    uni.showToast({ title: '已保存', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '保存失败', icon: 'none' })
  } finally {
    saving.value = false
  }
}

function resolvePrimaryPayload(primaryType: 'solar' | 'lunar') {
  if (primaryType === 'solar') {
    const solar = parseYMD(form.value.solarBirthday)
    return { primaryMonth: solar.month, primaryDay: solar.day, primaryYear: 0 }
  }
  const lunar = parseMonthDay(form.value.lunarBirthday)
  const month = lunar?.month || 1
  const day = lunar?.day || 1
  const currentYear = new Date().getFullYear()
  const solarObj = lunarCalendar.lunar2solar(currentYear, month, day, false)
  const primaryMonth = Number(solarObj?.cMonth) || month
  const primaryDay = Number(solarObj?.cDay) || day
  return { primaryMonth, primaryDay, primaryYear: currentYear }
}

function parseMonthDay(raw: string): { month: number; day: number } | null {
  const v = String(raw || '').trim()
  const m1 = v.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (m1) return { month: Number(m1[2]), day: Number(m1[3]) }
  const m2 = v.match(/^(\d{1,2})-(\d{1,2})$/)
  if (m2) return { month: Number(m2[1]), day: Number(m2[2]) }
  return null
}

function formatDisplay(raw: string): string {
  const parts = String(raw || '').split('-')
  if (parts.length === 3) return `${parts[0]}-${parts[1]}-${parts[2]}`
  if (parts.length === 2) return `${parts[0]}-${parts[1]}`
  return raw
}

function formatDisplaySolar(raw: string): string {
  const parts = String(raw || '').split('-')
  if (parts.length === 3) {
    const y = Number(parts[0])
    const m = Number(parts[1])
    const d = Number(parts[2])
    if (Number.isFinite(y) && Number.isFinite(m) && Number.isFinite(d)) {
      return `${y}年${m}月${d}日`
    }
  }
  if (parts.length === 2) {
    const m = Number(parts[0])
    const d = Number(parts[1])
    if (Number.isFinite(m) && Number.isFinite(d)) {
      return `${m}月${d}日`
    }
  }
  return raw
}

function formatDisplayLunar(raw: string): string {
  const parts = String(raw || '').split('-')
  if (parts.length === 3) {
    const y = Number(parts[0])
    const m = Number(parts[1])
    const d = Number(parts[2])
    if (Number.isFinite(y) && Number.isFinite(m) && Number.isFinite(d)) {
      const gz = lunarCalendar?.toGanZhiYear ? lunarCalendar.toGanZhiYear(y) : ''
      const yearLabel = gz ? `${y}(${gz}年)` : `${y}年`
      return `${yearLabel}${toChinaMonth(m)}${toChinaDay(d)}`
    }
  }
  if (parts.length === 2) {
    const m = Number(parts[0])
    const d = Number(parts[1])
    if (Number.isFinite(m) && Number.isFinite(d)) {
      return `${toChinaMonth(m)}${toChinaDay(d)}`
    }
  }
  return raw
}

function toChinaMonth(m: number): string {
  return lunarCalendar?.toChinaMonth ? lunarCalendar.toChinaMonth(m) : `${m}月`
}

function toChinaDay(d: number): string {
  return lunarCalendar?.toChinaDay ? lunarCalendar.toChinaDay(d) : `${d}日`
}

function parseYMD(raw: string): { year: number; month: number; day: number } {
  const m = String(raw || '').trim().match(/(\d{4})\D+(\d{1,2})\D+(\d{1,2})/)
  if (m) {
    return { year: Number(m[1]), month: Number(m[2]), day: Number(m[3]) }
  }
  const now = new Date()
  return { year: now.getFullYear(), month: now.getMonth() + 1, day: now.getDate() }
}

function formatYMD(year: number, month: number, day: number): string {
  return `${year}-${pad2(month)}-${pad2(day)}`
}

function pad2(v: number): string {
  return String(v).padStart(2, '0')
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

function toMonthDay(raw: string): string {
  const v = String(raw || '').trim()
  const m1 = v.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (m1) return formatMonthDay(Number(m1[2]), Number(m1[3]))
  const m2 = v.match(/^(\d{1,2})-(\d{1,2})$/)
  if (m2) return formatMonthDay(Number(m2[1]), Number(m2[2]))
  return '01-01'
}

function formatMonthDay(month: number, day: number): string {
  const mm = String(Math.max(1, Math.min(12, month))).padStart(2, '0')
  const dd = String(Math.max(1, Math.min(31, day))).padStart(2, '0')
  return `${mm}-${dd}`
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
</script>

<style scoped>
.page {
  padding: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}
.card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
  border: 1rpx solid rgba(0, 0, 0, 0.04);
}
.profile-card {
  display: flex;
  align-items: center;
  gap: 16rpx;
}
.avatar-wrap {
  width: 88rpx;
  height: 88rpx;
}
.avatar-button {
  padding: 0;
  background: transparent;
  border: none;
  line-height: 1;
}
.avatar-button::after {
  border: none;
}
.avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 44rpx;
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
.profile-info {
  flex: 1;
  min-width: 0;
}
.name {
  font-size: 32rpx;
  font-weight: 600;
}
.sub {
  margin-top: 6rpx;
  color: #666;
  font-size: 24rpx;
}
.section-title {
  font-size: 30rpx;
  font-weight: 600;
  margin-bottom: 12rpx;
  color: var(--brand-solid);
}
.field {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
  margin-top: 16rpx;
}
.label {
  font-size: 24rpx;
  color: #666;
}
.input,
.picker-value,
.textarea {
  background: #f9fafb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 28rpx;
  border: 1rpx solid #e6e8ef;
  box-shadow: 0 6rpx 16rpx rgba(0, 0, 0, 0.04);
  box-sizing: border-box;
  width: 100%;
}
.gender-select {
  margin-top: 4rpx;
  display: flex;
  align-items: center;
  gap: 12rpx;
  background: #f1f2f5;
  border-radius: 999rpx;
  padding: 6rpx;
}
.gender-option {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
  flex: 1;
  padding: 10rpx 0;
  border-radius: 999rpx;
  border: 1rpx solid transparent;
  background: transparent;
  color: #444;
}
.gender-option.active {
  border-color: #d6d9de;
  background: #f6f7fb;
  color: #333;
  box-shadow: 0 8rpx 18rpx rgba(0, 0, 0, 0.08);
}
.gender-radio {
  width: 18rpx;
  height: 18rpx;
  border-radius: 999rpx;
  border: 2rpx solid #999;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
}
.gender-radio::after {
  content: '';
  width: 8rpx;
  height: 8rpx;
  border-radius: 999rpx;
  background: #fff;
  opacity: 0;
}
.gender-option.active .gender-radio {
  border-color: #666;
}
.gender-option.active .gender-radio::after {
  background: #666;
  opacity: 1;
}
.gender-label {
  font-size: 28rpx;
  font-weight: 600;
}
.chips {
  margin-top: 8rpx;
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
}
.chip {
  padding: 8rpx 14rpx;
  border-radius: 999rpx;
  background: #f2f2f2;
  color: #666;
  font-size: 24rpx;
}
.chip.active {
  background: var(--brand-soft);
  color: var(--brand-solid);
}
.picker-value {
  color: #333;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 72rpx;
}
.input {
  min-height: 72rpx;
  line-height: 72rpx;
}
.picker-value::after {
  content: '>';
  color: #999;
  font-size: 26rpx;
  margin-left: 12rpx;
}
.hint {
  margin-top: 8rpx;
  font-size: 24rpx;
  color: #888;
}
.assist {
  margin-top: 8rpx;
  font-size: 24rpx;
  color: #777;
}
.textarea {
  min-height: 200rpx;
  line-height: 1.6;
  width: 100%;
}
.btn {
  margin-top: 6rpx;
  width: 100%;
}
</style>
