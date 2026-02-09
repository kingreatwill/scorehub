<template>
  <view class="page" :style="themeStyle" />
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { applyNavigationBarTheme, applyTabBarTheme, buildThemeVars, getThemeBaseColor } from '../../utils/theme'

const themeStyle = ref<Record<string, string>>(buildThemeVars(getThemeBaseColor()))

onLoad((q) => {
  syncTheme()
  const query = (q || {}) as any
  const id = String(query.id || '')
  if (!id) {
    uni.showToast({ title: '缺少得分簿参数', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 300)
    return
  }
  uni.redirectTo({ url: `/pages/profile/edit?mode=scorebook&id=${encodeURIComponent(id)}` })
})

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
}
</style>
