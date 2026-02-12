<template>
  <view v-if="visible" class="picker-mask" @click="onCancel">
    <view class="bgView" @click.stop>
      <view class="pickerHeader">
        <view class="pickerCancel" @click="onCancel">{{ cancelText }}</view>
        <view class="pickerTitle">
          <view class="tabWrap">
            <view class="tabItem left" :class="{ active: titleActive === 0 }" :style="tabStyle(0)" @click="onChangeTab(0)">公历</view>
            <view class="tabItem right" :class="{ active: titleActive === 1 }" :style="tabStyle(1)" @click="onChangeTab(1)">农历</view>
          </view>
        </view>
        <button class="pickerConfirm confirm-btn" @click="onConfirm">{{ confirmText }}</button>
      </view>

      <picker-view
        class="pickerView"
        indicator-style="height: 40px;border-top:1px solid #eee;border-bottom:1px solid #eee;"
        :value="pickerSelectIndexArr"
        @change="onChange"
        @pickstart="onChangeStart"
        @pickend="onChangeEnd"
      >
        <picker-view-column class="pickerColumn0">
          <view v-for="item in pickerYearArr" :key="item.value" class="pickerColumn">{{ item.label }}</view>
        </picker-view-column>
        <picker-view-column class="pickerColumn1">
          <view v-for="item in pickerMonthArr" :key="item.label" class="pickerColumn">{{ item.label }}</view>
        </picker-view-column>
        <picker-view-column class="pickerColumn2">
          <view v-for="item in pickerDayArr" :key="item.label" class="pickerColumn">{{ item.label }}</view>
        </picker-view-column>
      </picker-view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import lunarCalendar from '../../utils/lunar-calendar.mjs'

type PickerType = 'solar' | 'lunar'

const props = withDefaults(
  defineProps<{
    visible: boolean
    type: PickerType
    year: number
    month: number
    day: number
    minTime?: string
    maxTime?: string
    cancelText?: string
    confirmText?: string
    titleColor?: string
  }>(),
  {
    visible: false,
    type: 'solar',
    year: () => new Date().getFullYear(),
    month: () => new Date().getMonth() + 1,
    day: () => new Date().getDate(),
    minTime: '1901/01/01',
    maxTime: '2100/12/31',
    cancelText: '取消',
    confirmText: '确定',
    titleColor: 'var(--brand-solid)',
  },
)

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'confirm', payload: {
    time: string
    timeStamp: number
    timeObject: any
    isLunar: boolean
  }): void
}>()

const pickerYearArr = ref<any[]>([])
const pickerMonthArr = ref<any[]>([])
const pickerDayArr = ref<any[]>([])
const pickerSelectIndexArr = ref<number[]>([0, 0, 0])
const pickerSelectTime = ref('')
const pickerIsLunar = ref(false)
const titleActive = ref(0)
const isUpdateIndex = ref(false)
const isAutoUpdateIndex = ref(false)

const titleColor = computed(() => props.titleColor || 'var(--brand-solid)')

const tabStyle = (index: number) => {
  if (titleActive.value === index) {
    return { background: titleColor.value, color: '#fff' }
  }
  return { color: titleColor.value }
}

watch(
  () => props.visible,
  (v) => {
    if (v) initData(false)
  },
)

watch(
  () => props.type,
  (v) => {
    if (!props.visible) {
      pickerIsLunar.value = v === 'lunar'
      titleActive.value = pickerIsLunar.value ? 1 : 0
    } else {
      pickerIsLunar.value = v === 'lunar'
      titleActive.value = pickerIsLunar.value ? 1 : 0
      initData(true)
    }
  },
)


function initData(isUpdate: boolean) {
  if (!isUpdate) {
    pickerIsLunar.value = props.type === 'lunar'
    titleActive.value = pickerIsLunar.value ? 1 : 0
  }
  const selectTime = resolveSelectTime()
  if (!isUpdate || !pickerSelectTime.value) {
    pickerSelectTime.value = selectTime
  }
  const timeObj = getYearMonthDayObj(pickerSelectTime.value || selectTime)
  if (pickerIsLunar.value) {
    const lunarTimeObj = lunarCalendar.solar2lunar(timeObj.year, timeObj.month, timeObj.day)
    const newTimeObj = {
      year: lunarTimeObj.lYear,
      month: lunarTimeObj.lMonth,
      day: lunarTimeObj.lDay,
      isLeapMonth: String(lunarTimeObj.IMonthCn || '').includes('闰'),
    }
    setDateLunarTime(newTimeObj)
  } else {
    setDateTime(timeObj)
  }
}

function resolveSelectTime(): string {
  const year = Number.isFinite(props.year) ? props.year : new Date().getFullYear()
  const month = Number.isFinite(props.month) ? props.month : new Date().getMonth() + 1
  const day = Number.isFinite(props.day) ? props.day : new Date().getDate()
  if (pickerIsLunar.value) {
    const solarObj = lunarCalendar.lunar2solar(year, month, day, false)
    if (solarObj && solarObj.date) {
      return normalizeTime(String(solarObj.date))
    }
  }
  return formatYMD(year, month, day, '/')
}

function setDateTime(timeObj: { year: number; month: number; day: number }) {
  const yearArr = getYears()
  const monthArr = getMonths(timeObj.year)
  const dayArr = getDays(timeObj.year, timeObj.month)
  pickerYearArr.value = yearArr
  pickerMonthArr.value = monthArr
  pickerDayArr.value = dayArr

  let yearIndex = yearArr.findIndex((item) => item.value === timeObj.year)
  let monthIndex = monthArr.findIndex((item) => item.value === timeObj.month)
  let dayIndex = dayArr.findIndex((item) => item.value === timeObj.day)

  const minTime = props.minTime || '1901/01/01'
  if (timeObj.year <= getYear(minTime)) {
    yearIndex = yearArr.findIndex((item) => item.value === getYear(minTime))
  }
  if (timeObj.year <= getYear(minTime) && timeObj.month <= getMonth(minTime)) {
    monthIndex = monthArr.findIndex((item) => item.value === getMonth(minTime))
  }
  if (timeObj.year <= getYear(minTime) && timeObj.month <= getMonth(minTime) && timeObj.day <= getDay(minTime)) {
    dayIndex = dayArr.findIndex((item) => item.value === getDay(minTime))
  }

  const maxTime = props.maxTime || '2100/12/31'
  if (timeObj.year >= getYear(maxTime)) {
    yearIndex = yearArr.findIndex((item) => item.value === getYear(maxTime))
  }
  if (timeObj.year >= getYear(maxTime) && timeObj.month >= getMonth(maxTime)) {
    monthIndex = monthArr.findIndex((item) => item.value === getMonth(maxTime))
  }
  if (timeObj.year >= getYear(maxTime) && timeObj.month >= getMonth(maxTime) && timeObj.day >= getDay(maxTime)) {
    dayIndex = dayArr.findIndex((item) => item.value === getDay(maxTime))
  }

  yearIndex = clampIndex(yearIndex, yearArr.length)
  monthIndex = clampIndex(monthIndex, monthArr.length)
  dayIndex = clampIndex(dayIndex, dayArr.length)

  const selectTime = formatYMD(
    yearArr[yearIndex]?.value || timeObj.year,
    monthArr[monthIndex]?.value || timeObj.month,
    dayArr[dayIndex]?.value || timeObj.day,
    '/',
  )
  pickerSelectTime.value = selectTime

  isAutoUpdateIndex.value = true
  isUpdateIndex.value = false
  pickerSelectIndexArr.value = [yearIndex, monthIndex, dayIndex]
  setTimeout(() => {
    isAutoUpdateIndex.value = false
  }, 300)
}

function setDateLunarTime(timeObj: { year: number; month: number; day: number; isLeapMonth: boolean }) {
  const isLeapMonth = timeObj.isLeapMonth && lunarCalendar.leapMonth(timeObj.year) === timeObj.month
  const lunarTimeObj = lunarCalendar.lunar2solar(timeObj.year, timeObj.month, timeObj.day, isLeapMonth)
  if (!lunarTimeObj || lunarTimeObj === -1) return

  const yearArr = getLunarYears()
  const monthArr = getLunarMonths(lunarTimeObj.lYear)
  const dayArr = getLunarDays(lunarTimeObj.lYear, lunarTimeObj.lMonth, isLeapMonth)
  pickerYearArr.value = yearArr
  pickerMonthArr.value = monthArr
  pickerDayArr.value = dayArr

  let yearIndex = yearArr.findIndex((item) => item.value === timeObj.year)
  let monthIndex = monthArr.findIndex((item) => item.label === lunarTimeObj.IMonthCn)
  let dayIndex = dayArr.findIndex((item) => item.label === lunarTimeObj.IDayCn)

  const minLunarTimeObj = convertLunarTime(props.minTime || '1901/01/01')
  const maxLunarTimeObj = convertLunarTime(props.maxTime || '2100/12/31')
  const minYear = minLunarTimeObj.lYear
  const minMonth = minLunarTimeObj.lMonth
  const minDay = minLunarTimeObj.lDay
  if (timeObj.year <= minYear) {
    yearIndex = yearArr.findIndex((item) => item.value === minYear)
  }
  if (timeObj.year <= minYear && timeObj.month <= minMonth) {
    monthIndex = monthArr.findIndex((item) => item.value === minMonth)
  }
  if (timeObj.year <= minYear && timeObj.month <= minMonth && timeObj.day <= minDay) {
    dayIndex = dayArr.findIndex((item) => item.value === minDay)
  }

  const maxYear = maxLunarTimeObj.lYear
  const maxMonth = maxLunarTimeObj.lMonth
  const maxDay = maxLunarTimeObj.lDay
  if (timeObj.year >= maxYear) {
    yearIndex = yearArr.findIndex((item) => item.value === maxYear)
  }
  if (timeObj.year >= maxYear && timeObj.month >= maxMonth) {
    monthIndex = monthArr.findIndex((item) => item.value === maxMonth)
  }
  if (timeObj.year >= maxYear && timeObj.month >= maxMonth && timeObj.day >= maxDay) {
    dayIndex = dayArr.findIndex((item) => item.value === maxDay)
  }

  yearIndex = clampIndex(yearIndex, yearArr.length)
  monthIndex = clampIndex(monthIndex, monthArr.length)
  dayIndex = clampIndex(dayIndex, dayArr.length)

  const selectLunarTimeObj = lunarCalendar.lunar2solar(
    yearArr[yearIndex]?.value || timeObj.year,
    monthArr[monthIndex]?.value || timeObj.month,
    dayArr[dayIndex]?.value || timeObj.day,
    isLeapMonth,
  )
  if (selectLunarTimeObj && selectLunarTimeObj.date) {
    pickerSelectTime.value = normalizeTime(String(selectLunarTimeObj.date))
  }

  isAutoUpdateIndex.value = true
  isUpdateIndex.value = false
  pickerSelectIndexArr.value = [yearIndex, monthIndex, dayIndex]
  setTimeout(() => {
    isAutoUpdateIndex.value = false
  }, 300)
}

function getYears() {
  const minYear = getYear(props.minTime || '1901/01/01')
  const maxYear = getYear(props.maxTime || '2100/12/31')
  const tempArr = []
  for (let i = minYear; i <= maxYear; i += 1) {
    tempArr.push({ label: `${i}年`, value: i })
  }
  return tempArr
}

function getMonths(year: number) {
  let minMonth = 1
  let maxMonth = 12
  if (year <= getYear(props.minTime || '1901/01/01')) {
    minMonth = getMonth(props.minTime || '1901/01/01')
  }
  if (year >= getYear(props.maxTime || '2100/12/31')) {
    maxMonth = getMonth(props.maxTime || '2100/12/31')
  }
  const tempArr = []
  for (let i = minMonth; i <= maxMonth; i += 1) {
    tempArr.push({ label: `${i}月`, value: i })
  }
  return tempArr
}

function getDays(year: number, month: number) {
  const dayInMonth = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]
  if ((year % 4 === 0 && year % 100 !== 0) || year % 400 === 0) dayInMonth[1] = 29
  let minDay = 1
  let maxDay = dayInMonth[month - 1]
  if (year <= getYear(props.minTime || '1901/01/01') && month <= getMonth(props.minTime || '1901/01/01')) {
    minDay = getDay(props.minTime || '1901/01/01')
  }
  if (year >= getYear(props.maxTime || '2100/12/31') && month >= getMonth(props.maxTime || '2100/12/31')) {
    maxDay = getDay(props.maxTime || '2100/12/31')
  }
  const tempArr = []
  for (let i = minDay; i <= maxDay; i += 1) {
    tempArr.push({ label: `${i}日`, value: i })
  }
  return tempArr
}

function getLunarYears() {
  const minLunar = convertLunarTime(props.minTime || '1901/01/01')
  const maxLunar = convertLunarTime(props.maxTime || '2100/12/31')
  const minYear = minLunar.lYear
  const maxYear = maxLunar.lYear
  const tempArr = []
  for (let i = minYear; i <= maxYear; i += 1) {
    tempArr.push({ label: `${i}(${lunarCalendar.toGanZhiYear(i)}年)`, value: i })
  }
  return tempArr
}

function getLunarMonths(year: number) {
  let minMonth = 1
  let maxMonth = 12
  const minLunar = convertLunarTime(props.minTime || '1901/01/01')
  const maxLunar = convertLunarTime(props.maxTime || '2100/12/31')
  if (year <= minLunar.lYear) {
    minMonth = minLunar.lMonth
  }
  if (year >= maxLunar.lYear) {
    maxMonth = maxLunar.lMonth
  }
  const leapMonth = lunarCalendar.leapMonth(year)
  const tempArr: Array<{ label: string; value: number; isLeapMonth: boolean }> = []
  for (let i = minMonth; i <= maxMonth; i += 1) {
    tempArr.push({ label: lunarCalendar.toChinaMonth(i), value: i, isLeapMonth: false })
  }
  if (leapMonth > 0 && leapMonth >= minMonth && leapMonth <= maxMonth) {
    tempArr.splice(leapMonth, 0, {
      label: `闰${lunarCalendar.toChinaMonth(leapMonth)}`,
      value: leapMonth,
      isLeapMonth: true,
    })
  }
  return tempArr
}

function getLunarDays(year: number, month: number, isLeapMonth = false) {
  const days = isLeapMonth ? lunarCalendar.leapDays(year) : lunarCalendar.monthDays(year, month)
  let minDay = 1
  let maxDay = days
  const minLunar = convertLunarTime(props.minTime || '1901/01/01')
  const maxLunar = convertLunarTime(props.maxTime || '2100/12/31')
  if (year <= minLunar.lYear && month <= minLunar.lMonth) {
    minDay = minLunar.lDay
  }
  if (year >= maxLunar.lYear && month >= maxLunar.lMonth) {
    maxDay = maxLunar.lDay
  }
  const tempArr = []
  for (let i = minDay; i <= maxDay; i += 1) {
    tempArr.push({ label: lunarCalendar.toChinaDay(i), value: i })
  }
  return tempArr
}

function onCancel() {
  emit('update:visible', false)
}

function onChangeStart() {
  isUpdateIndex.value = true
}

function onChangeEnd() {
  if (isAutoUpdateIndex.value) {
    isAutoUpdateIndex.value = false
  } else {
    setTimeout(() => {
      isUpdateIndex.value = false
    }, 300)
  }
}

function onChange(event: any) {
  const value = event?.detail?.value || []
  const yearIndex = value[0] >= 0 ? value[0] : 0
  const monthIndex = value[1] >= 0 ? value[1] : 0
  const dayIndex = value[2] >= 0 ? value[2] : 0
  const yearObj = pickerYearArr.value[yearIndex]
  const monthObj = pickerMonthArr.value[monthIndex]
  const dayObj = pickerDayArr.value[dayIndex]
  if (!yearObj || !monthObj || !dayObj) return
  if (isUpdateIndex.value) {
    if (pickerIsLunar.value) {
      setDateLunarTime({
        year: yearObj?.value,
        month: monthObj?.value,
        day: dayObj?.value,
        isLeapMonth: !!monthObj?.isLeapMonth,
      })
    } else {
      setDateTime({
        year: yearObj?.value,
        month: monthObj?.value,
        day: dayObj?.value,
      })
    }
  }
}

function onConfirm() {
  const selectTime = pickerSelectTime.value
  const timeStamp = convertTimeStamp(selectTime)
  const time = normalizeTime(selectTime).replace(/-/g, '/')
  const timeObject = convertLunarTime(selectTime)
  emit('confirm', {
    time,
    timeStamp,
    timeObject,
    isLunar: pickerIsLunar.value,
  })
  emit('update:visible', false)
}

function onChangeTab(index: number) {
  titleActive.value = index
  pickerIsLunar.value = index === 1
  initData(true)
}

function normalizeTime(time: string): string {
  const obj = getYearMonthDayObj(time)
  return formatYMD(obj.year, obj.month, obj.day, '-')
}

function convertLunarTime(time: string) {
  const { year, month, day } = getYearMonthDayObj(time)
  return lunarCalendar.solar2lunar(year, month, day)
}

function getYear(time: string) {
  const obj = getYearMonthDayObj(time)
  return Number(obj.year)
}

function getMonth(time: string) {
  const obj = getYearMonthDayObj(time)
  return Number(obj.month)
}

function getDay(time: string) {
  const obj = getYearMonthDayObj(time)
  return Number(obj.day)
}

function getYearMonthDayObj(time: string) {
  const raw = String(time || '').trim()
  const m = raw.match(/(\d{4})\D+(\d{1,2})\D+(\d{1,2})/)
  if (m) {
    return {
      year: Number(m[1]),
      month: Number(m[2]),
      day: Number(m[3]),
    }
  }
  const now = new Date()
  return {
    year: now.getFullYear(),
    month: now.getMonth() + 1,
    day: now.getDate(),
  }
}

function clampIndex(index: number, length: number): number {
  if (!Number.isFinite(index) || index < 0) return 0
  if (index >= length) return Math.max(0, length - 1)
  return index
}

function formatYMD(year: number, month: number, day: number, sep: '/' | '-') {
  const y = String(year)
  const m = String(month).padStart(2, '0')
  const d = String(day).padStart(2, '0')
  return `${y}${sep}${m}${sep}${d}`
}

function convertTimeStamp(time: string): number {
  const obj = getYearMonthDayObj(time)
  return new Date(obj.year, obj.month - 1, obj.day).getTime()
}
</script>

<style scoped>
.picker-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: flex-end;
  z-index: 9999;
}
.bgView {
  width: 100%;
  height: 600rpx;
  background: #fff;
  border-radius: 24rpx 24rpx 0 0;
  overflow: hidden;
}
.pickerHeader {
  display: flex;
  height: 72rpx;
  line-height: 72rpx;
  background: #f8f8f8;
  border-bottom: 4rpx solid #eee;
}
.pickerCancel {
  flex: 15;
  padding: 0 10rpx;
  text-align: center;
  color: #666;
}
.pickerTitle {
  flex: 70;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14rpx;
}
.tabWrap {
  display: flex;
  border-radius: 14rpx;
  overflow: hidden;
  border: none;
  background: #f2f2f2;
  height: 72rpx;
}
.tabItem {
  padding: 0 18rpx;
  font-size: 26rpx;
  color: inherit;
  background: transparent;
  height: 72rpx;
  line-height: 72rpx;
  border-radius: 0;
}
.tabItem.left {
  border-top-left-radius: 14rpx;
  border-bottom-left-radius: 14rpx;
}
.tabItem.right {
  border-top-right-radius: 14rpx;
  border-bottom-right-radius: 14rpx;
}
.tabItem.active {
  color: inherit;
  background: inherit;
}
.pickerConfirm {
  flex: 15;
  padding: 0 6rpx;
  text-align: center;
  font-weight: 600;
  height: 72rpx;
  line-height: 72rpx;
  border-radius: 14rpx;
}
.pickerConfirm.confirm-btn {
  width: auto;
  min-width: 110rpx;
  padding: 0 18rpx;
  height: 72rpx;
  line-height: 72rpx;
  font-size: 26rpx;
}
.pickerConfirm::after {
  border: none;
}
.pickerView {
  height: 480rpx;
  font-size: 34rpx;
}
.pickerColumn {
  height: 40px;
  line-height: 40px;
  text-align: center;
}
.pickerColumn0 {
  flex: 0 0 40%;
}
.pickerColumn1 {
  flex: 0 0 30%;
}
.pickerColumn2 {
  flex: 0 0 30%;
}
</style>
