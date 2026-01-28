<template>
  <view class="page">
    <view v-if="!token" class="empty">
      <text>未登录，请先在「得分簿」页登录。</text>
    </view>

    <view v-else class="list">
      <view class="item" v-for="it in items" :key="it.id" @click="open(it.id)">
        <view class="row">
          <text class="name">{{ it.name }}</text>
          <text class="status">{{ it.status === 'ended' ? '已结束' : '记录中' }}</text>
        </view>
        <view class="sub">
          <text v-if="it.locationText">{{ it.locationText }}</text>
          <text>成员 {{ it.memberCount }}</text>
        </view>
      </view>

      <view v-if="items.length === 0" class="empty">
        <text>暂无得分簿</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { listMyScorebooks } from '../../utils/api'

const token = ref('')
const items = ref<any[]>([])

onShow(async () => {
  token.value = (uni.getStorageSync('token') as string) || ''
  if (!token.value) return
  try {
    const res = await listMyScorebooks()
    items.value = res.items || []
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  }
})

function open(id: string) {
  uni.navigateTo({ url: `/pages/scorebook/detail?id=${id}` })
}
</script>

<style scoped>
.page {
  padding: 24rpx;
}
.list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.item {
  background: #fff;
  border-radius: 16rpx;
  padding: 20rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}
.name {
  font-size: 30rpx;
  font-weight: 600;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.status {
  color: #666;
  font-size: 24rpx;
  flex: none;
  white-space: nowrap;
}
.sub {
  margin-top: 8rpx;
  color: #666;
  font-size: 24rpx;
  display: flex;
  justify-content: space-between;
}
.empty {
  margin-top: 80rpx;
  color: #666;
  text-align: center;
}
</style>
