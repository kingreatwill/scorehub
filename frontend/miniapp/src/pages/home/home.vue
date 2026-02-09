<template>
  <view class="page" :style="themeStyle">跳转中…</view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor } from '../../utils/theme'

const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))

onLoad((q) => {
  syncTheme()
  const query = (q || {}) as any
  const code = normalizeCode(String(query.scene || query.code || ''))
  if (code) {
    uni.redirectTo({ url: `/pages/join/index?scene=${encodeURIComponent(code)}` })
    return
  }
  uni.switchTab({ url: '/pages/home/index' })
})

function normalizeCode(v: string): string {
  const raw = decodeURIComponent(String(v || '')).trim()
  if (!raw) return ''
  if (/^[0-9A-Z]{6,12}$/.test(raw)) return raw
  const m = raw.match(/(?:^|[?&])code=([^&]+)/)
  if (m?.[1]) return decodeURIComponent(m[1]).trim()
  return raw
}

function syncTheme() {
  const base = getThemeBaseColor()
  themeStyle.value = buildThemeVars(base)
  applyNavigationBarTheme(base)
  applyTabBarTheme(base)
}
</script>

<style scoped>
.page {
  padding: 24rpx;
  color: #666;
}
</style>
