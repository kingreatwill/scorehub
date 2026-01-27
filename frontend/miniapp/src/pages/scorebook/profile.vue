<template>
  <view class="page">
    <view class="card">
      <view class="title">修改我的头像/昵称</view>
      <button class="btn" @click="fillFromWeChat">从微信获取头像/昵称</button>
      <form @submit="onSaveSubmit">
        <input class="input" name="nickname" :type="nicknameInputType" v-model="nickname" placeholder="昵称" />
        <input class="input" name="avatarUrl" v-model="avatarUrl" placeholder="头像 URL（可选）" />
        <button class="btn primary" form-type="submit">保存</button>
      </form>
    </view>
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { updateMyProfile } from '../../utils/api'

const scorebookId = ref('')
const nickname = ref('')
const avatarUrl = ref('')

let nicknameInputType = 'text'
// #ifdef MP-WEIXIN
nicknameInputType = 'nickname'
// #endif

onLoad((q) => {
  const query = q as any
  scorebookId.value = String(query.id || '')
  nickname.value = decodeURIComponent(String(query.nickname || ''))
  avatarUrl.value = decodeURIComponent(String(query.avatarUrl || ''))
})

async function fillFromWeChat() {
  // #ifndef MP-WEIXIN
  uni.showToast({ title: '请在微信小程序内使用', icon: 'none' })
  return
  // #endif

  // #ifdef MP-WEIXIN
  try {
    const profile = await new Promise<any>((resolve, reject) => {
      uni.getUserProfile({
        desc: '用于显示头像和昵称',
        success: resolve,
        fail: reject,
      } as any)
    })
    const userInfo = profile?.userInfo || {}
    if (userInfo.nickName) nickname.value = String(userInfo.nickName)
    if (userInfo.avatarUrl) avatarUrl.value = String(userInfo.avatarUrl)
  } catch (e: any) {
    uni.showToast({ title: e?.message || '获取失败', icon: 'none' })
  }
  // #endif
}

async function save() {
  if (!nickname.value.trim()) return uni.showToast({ title: '昵称不能为空', icon: 'none' })
  try {
    await updateMyProfile(scorebookId.value, { nickname: nickname.value.trim(), avatarUrl: avatarUrl.value.trim() })
    uni.showToast({ title: '已保存', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 300)
  } catch (e: any) {
    uni.showToast({ title: e?.message || '保存失败', icon: 'none' })
  }
}

async function onSaveSubmit(e: any) {
  const submittedNickname = String(e?.detail?.value?.nickname || '').trim()
  const submittedAvatarUrl = String(e?.detail?.value?.avatarUrl || '').trim()
  if (submittedNickname !== nickname.value.trim()) nickname.value = submittedNickname
  if (submittedAvatarUrl !== avatarUrl.value.trim()) avatarUrl.value = submittedAvatarUrl
  await save()
}
</script>

<style scoped>
.page {
  padding: 24rpx;
}
.card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
}
.title {
  font-size: 32rpx;
  font-weight: 600;
  margin-bottom: 12rpx;
}
.input {
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 28rpx;
  margin-top: 12rpx;
}
.btn {
  margin-top: 20rpx;
}
.primary {
  background: #111;
  color: #fff;
}
</style>
