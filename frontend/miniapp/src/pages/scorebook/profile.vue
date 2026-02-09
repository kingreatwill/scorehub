<template>
  <view class="page" />
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'

onLoad((q) => {
  const query = (q || {}) as any
  const id = String(query.id || '')
  if (!id) {
    uni.showToast({ title: '缺少得分簿参数', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 300)
    return
  }
  uni.redirectTo({ url: `/pages/profile/edit?mode=scorebook&id=${encodeURIComponent(id)}` })
})
</script>

<style scoped>
.page {
  padding: 24rpx;
}
</style>
