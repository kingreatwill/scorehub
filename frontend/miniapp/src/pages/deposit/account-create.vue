<template>
  <view class="page" :style="themeStyle">
    <view class="card profile-card">
      <template v-if="isMpWeixin">
        <button class="avatar-wrap avatar-button" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
          <image v-if="form.avatarUrl" class="avatar" :src="form.avatarUrl" mode="aspectFill" />
          <view v-else class="avatar avatar-fallback" :style="avatarStyle(form.bank)">{{ initial }}</view>
        </button>
      </template>
      <template v-else>
        <view class="avatar-wrap">
          <image v-if="form.avatarUrl" class="avatar" :src="form.avatarUrl" mode="aspectFill" />
          <view v-else class="avatar avatar-fallback" :style="avatarStyle(form.bank)">{{ initial }}</view>
        </view>
      </template>
      <view class="profile-info">
        <view class="name">{{ form.bank || '银行账户' }}</view>
        <view class="sub" v-if="form.branch">{{ form.branch }}</view>
      </view>
    </view>

    <view class="card base-card">
      <view class="section-title">账户信息</view>
      <view class="field">
        <text class="label">银行</text>
        <view class="field-body">
          <view class="bank-input-row">
            <input class="input bank-input" v-model="form.bank" placeholder="" :focus="bankInputFocus" />
            <button class="select-btn" @click="openBankPicker">
              <!-- #ifdef H5 -->
              <img class="select-icon select-icon-h5" :src="selectIcon" alt="" />
              <!-- #endif -->
              <!-- #ifndef H5 -->
              <image class="select-icon" :src="selectIcon" mode="aspectFit" />
              <!-- #endif -->
            </button>
          </view>
        </view>
      </view>
      <view class="field">
        <text class="label">支行</text>
        <view class="field-body">
          <input class="input" v-model="form.branch" placeholder="（可选）" />
        </view>
      </view>
      <view class="field">
        <text class="label">账号</text>
        <view class="field-body">
          <input class="input" v-model="form.accountNo" placeholder="（可选）" />
        </view>
      </view>
      <view class="field">
        <text class="label">户名</text>
        <view class="field-body">
          <input class="input" v-model="form.holder" placeholder="（可选）" />
        </view>
      </view>
      <view class="field" v-if="!isMpWeixin">
        <text class="label">头像</text>
        <view class="field-body">
          <input class="input" v-model="form.avatarUrl" placeholder="（可选）" />
        </view>
      </view>
    </view>

    <view class="card">
      <view class="section-title">备注</view>
      <textarea class="textarea" v-model="form.note" placeholder="（可选）" />
    </view>

    <button class="btn confirm-btn" :disabled="saving" @click="onSave">
      {{ saving ? (isEditing ? '保存中…' : '创建中…') : isEditing ? '保存' : '创建账户' }}
    </button>

    <view class="modal-mask" v-if="bankPickerVisible" @click="closeBankPicker" />
    <view class="modal bank-modal" v-if="bankPickerVisible">
      <view class="modal-head">
        <view class="modal-title">选择银行</view>
        <view class="modal-close" @click="closeBankPicker">×</view>
      </view>
      <view class="bank-search">
        <input class="input" v-model="bankKeyword" placeholder="搜索银行名称或简称" />
      </view>
      <scroll-view class="bank-scroll" scroll-y>
        <view class="bank-item" v-for="bank in filteredBanks" :key="bank.name" @click="selectBank(bank)">
          <!-- #ifdef H5 -->
          <img
            v-if="bankLogoSrc(bank)"
            class="bank-logo bank-logo-h5"
            :src="bankLogoSrc(bank)"
            alt=""
            loading="lazy"
            @error="onBankLogoError(bank)"
          />
          <view v-else class="bank-logo bank-logo-fallback">{{ bankLogoFallbackText(bank) }}</view>
          <!-- #endif -->
          <!-- #ifndef H5 -->
          <image v-if="bank.logo" class="bank-logo" :src="bank.logo" mode="aspectFit" />
          <!-- #endif -->
          <view class="bank-info">
            <view class="bank-name">{{ bank.name }}</view>
            <view class="bank-abbr" v-if="bank.abbr">{{ bank.abbr }}</view>
          </view>
        </view>
      </scroll-view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor } from '../../utils/theme'
import { avatarStyle } from '../../utils/avatar-color'
import { bankList, type BankMeta } from '../../utils/banks'
import { createDepositAccount, getDepositAccount, updateDepositAccount } from '../../utils/api'

type Account = {
  id: string
  bank: string
  branch: string
  accountNo: string
  holder: string
  avatarUrl: string
  note: string
}

const form = ref<Account>({
  id: '',
  bank: '',
  branch: '',
  accountNo: '',
  holder: '',
  avatarUrl: '',
  note: '',
})
const saving = ref(false)
const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))
const isEditing = ref(false)
const bankPickerVisible = ref(false)
const bankKeyword = ref('')
const bankInputFocus = ref(false)
const bankLogoCandidateIndex = ref<Record<string, number>>({})
const selectIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="44" height="44" viewBox="0 0 24 24" fill="none" stroke="%23888" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 10h18"/><path d="M5 10v8"/><path d="M9 10v8"/><path d="M15 10v8"/><path d="M19 10v8"/><path d="M2 18h20"/><path d="M12 4l9 4H3z"/></svg>'

const initial = computed(() => String(form.value.bank || '').trim().slice(0, 1) || '存')
const isMpWeixin = ref(false)
// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

onLoad((q) => {
  const query = (q || {}) as any
  const id = String(query.id || '').trim()
  if (id) {
    loadAccount(id)
  }
})

onShow(() => {
  syncTheme()
})

function syncTheme() {
  const base = getThemeBaseColor()
  themeStyle.value = buildThemeVars(base)
  applyNavigationBarTheme(base)
  applyTabBarTheme(base)
}

async function onSave() {
  if (saving.value) return
  if (!form.value.bank.trim()) {
    uni.showToast({ title: '请输入银行名称', icon: 'none' })
    return
  }
  saving.value = true
  try {
    const base: Account = {
      ...form.value,
      bank: form.value.bank.trim(),
      branch: form.value.branch.trim(),
      accountNo: form.value.accountNo.trim(),
      holder: form.value.holder.trim(),
      avatarUrl: form.value.avatarUrl.trim(),
      note: form.value.note.trim(),
    }
    if (isEditing.value && form.value.id) {
      const res = await updateDepositAccount(form.value.id, {
        bank: base.bank,
        branch: base.branch,
        accountNo: base.accountNo,
        holder: base.holder,
        avatarUrl: base.avatarUrl,
        note: base.note,
      })
      if (res?.account) {
        form.value = {
          ...base,
          id: String(res.account.id || form.value.id),
        }
      }
      uni.showToast({ title: '已保存', icon: 'success' })
    } else {
      const res = await createDepositAccount({
        bank: base.bank,
        branch: base.branch,
        accountNo: base.accountNo,
        holder: base.holder,
        avatarUrl: base.avatarUrl,
        note: base.note,
      })
      if (res?.account) {
        form.value = {
          ...base,
          id: String(res.account.id || ''),
        }
      }
      uni.showToast({ title: '已创建', icon: 'success' })
    }
    setTimeout(() => {
      uni.navigateBack({ delta: 1 })
    }, 400)
  } finally {
    saving.value = false
  }
}

async function loadAccount(id: string) {
  try {
    const res = await getDepositAccount(id)
    const acc = res?.account
    if (acc) {
      form.value = {
        id: String(acc.id || id),
        bank: String(acc.bank || ''),
        branch: String(acc.branch || ''),
        accountNo: String(acc.accountNo || ''),
        holder: String(acc.holder || ''),
        avatarUrl: String(acc.avatarUrl || ''),
        note: String(acc.note || ''),
      }
      isEditing.value = true
      uni.setNavigationBarTitle({ title: '编辑账户' })
      return
    }
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  }
}

const filteredBanks = computed(() => {
  const keyword = bankKeyword.value.trim().toLowerCase()
  const list = bankList.filter((bank) => bank.name !== '自定义')
  if (!keyword) return list
  return list.filter((bank) => {
    const name = String(bank.name || '').toLowerCase()
    const abbr = String(bank.abbr || '').toLowerCase()
    return name.includes(keyword) || abbr.includes(keyword)
  })
})

function openBankPicker() {
  bankKeyword.value = ''
  bankPickerVisible.value = true
}

function closeBankPicker() {
  bankPickerVisible.value = false
}

function toAssetPath(raw: string): string {
  const v = String(raw || '').trim()
  if (!v) return ''
  if (v.startsWith('/')) return v
  if (/^https?:\/\//i.test(v)) {
    try {
      return String(new URL(v).pathname || '').trim()
    } catch (e) {}
  }
  return ''
}

function currentOrigin(): string {
  try {
    // #ifdef H5
    return String(window?.location?.origin || '').trim()
    // #endif
  } catch (e) {}
  return ''
}

function bankLogoCandidates(bank: BankMeta): string[] {
  const list: string[] = []
  const push = (v: string) => {
    const s = String(v || '').trim()
    if (!s || list.includes(s)) return
    list.push(s)
  }
  const origin = currentOrigin()
  const addFrom = (raw: string) => {
    const source = String(raw || '').trim()
    if (!source) return
    push(source)
    const pathname = toAssetPath(source)
    if (!pathname) return
    push(pathname)
    if (origin) push(`${origin}${pathname}`)
    push(`https://wxapi.wcoder.com${pathname}`)
  }
  addFrom(bank.logo)
  addFrom(bank.wordmark)
  return list
}

function bankLogoSrc(bank: BankMeta): string {
  const key = String(bank?.name || '')
  const idx = bankLogoCandidateIndex.value[key] ?? 0
  const candidates = bankLogoCandidates(bank)
  if (idx < 0 || idx >= candidates.length) return ''
  return candidates[idx]
}

function onBankLogoError(bank: BankMeta) {
  const key = String(bank?.name || '')
  if (!key) return
  const current = bankLogoCandidateIndex.value[key] ?? 0
  const total = bankLogoCandidates(bank).length
  if (current + 1 < total) {
    bankLogoCandidateIndex.value[key] = current + 1
    return
  }
  bankLogoCandidateIndex.value[key] = -1
}

function bankLogoFallbackText(bank: BankMeta): string {
  const abbr = String(bank?.abbr || '').trim()
  if (abbr) return abbr.slice(0, 2).toUpperCase()
  return String(bank?.name || '').trim().slice(0, 1) || '银'
}

function selectBank(bank: BankMeta) {
  form.value.bank = bank.name
  form.value.avatarUrl = bankLogoSrc(bank) || bank.logo || bank.wordmark || ''
  closeBankPicker()
}

function focusBankInput() {
  bankInputFocus.value = true
  nextTick(() => {
    setTimeout(() => {
      bankInputFocus.value = false
    }, 200)
  })
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
  } catch (err: any) {
    uni.showToast({ title: '头像处理失败', icon: 'none' })
  }
  // #endif
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
.profile-card {
  display: flex;
  align-items: center;
  gap: 16rpx;
}
.avatar-wrap {
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
  font-size: 32rpx;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}
.avatar-button {
  padding: 0;
  border: none;
  background: transparent;
}
.avatar-button::after {
  border: none;
}
.profile-info {
  flex: 1;
  min-width: 0;
}
.name {
  font-size: 30rpx;
  font-weight: 600;
}
.sub {
  margin-top: 6rpx;
  color: #666;
  font-size: 24rpx;
}
.section-title {
  font-size: 28rpx;
  font-weight: 600;
  margin-bottom: 12rpx;
}
.base-card {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.field {
  display: flex;
  align-items: center;
  gap: 16rpx;
}
.label {
  width: 120rpx;
  color: #666;
  font-size: 26rpx;
  flex: none;
}
.field-body {
  flex: 1;
  min-width: 0;
}
.bank-input-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}
.bank-input {
  flex: 1;
  min-width: 0;
}
.select-btn {
  height: 72rpx;
  line-height: 72rpx;
  width: 104rpx;
  padding: 0;
  border-radius: 12rpx;
  background: #f6f7fb;
  font-size: 24rpx;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
}
.select-btn::after {
  border: none;
}
.select-icon {
  width: 88rpx;
  height: 44rpx;
}
.select-icon-h5 {
  display: block;
  object-fit: contain;
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
.modal-mask {
  position: fixed;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.35);
  z-index: 2000;
}
.modal {
  position: fixed;
  left: 24rpx;
  right: 24rpx;
  top: 10%;
  bottom: 10%;
  background: #fff;
  border-radius: 20rpx;
  padding: 24rpx;
  z-index: 2001;
  display: flex;
  flex-direction: column;
  gap: 12rpx;
  box-shadow: 0 16rpx 40rpx rgba(0, 0, 0, 0.2);
}
.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.modal-title {
  font-size: 30rpx;
  font-weight: 600;
}
.modal-close {
  font-size: 32rpx;
  color: #888;
  padding: 6rpx 12rpx;
}
.bank-search {
  margin-top: 4rpx;
}
.bank-scroll {
  flex: 1;
  height: 100%;
  overflow: hidden;
}
.bank-item {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 12rpx 8rpx;
  border-radius: 12rpx;
  width: 100%;
  box-sizing: border-box;
}
.bank-item:active {
  background: #f6f7fb;
}
.bank-item.custom {
  background: #f6f7fb;
}
.bank-logo {
  width: 52rpx;
  height: 52rpx;
  flex: none;
}
.bank-logo-h5 {
  display: block;
  object-fit: contain;
}
.bank-logo-fallback {
  border-radius: 10rpx;
  background: #f1f2f6;
  color: #666;
  font-size: 20rpx;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}
.bank-info {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}
.bank-name {
  font-size: 28rpx;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.bank-abbr {
  font-size: 22rpx;
  color: #666;
}
</style>
