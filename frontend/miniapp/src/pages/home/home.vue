<template>
  <view class="page">跳转中…</view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'

onLoad((q) => {
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
</script>

<style scoped>
.page {
  padding: 24rpx;
  color: #666;
}
</style>

