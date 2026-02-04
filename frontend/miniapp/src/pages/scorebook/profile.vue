<template>
  <view class="page">
    <view class="card">
      <view class="title">修改我的头像/昵称</view>
      <form @submit="onSaveSubmit">
        <!-- #ifdef MP-WEIXIN -->
        <button class="avatar-wrapper" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
          <image class="avatar" :src="avatarUrl || fallbackAvatar" mode="aspectFill" />
          <view class="avatar-tip">{{ avatarUrl ? '点击更换头像' : '点击选择头像' }}</view>
        </button>
        <!-- #endif -->
        <input
          class="input"
          name="nickname"
          :type="nicknameInputType"
          :value="nickname"
          placeholder="昵称"
          :controlled="true"
          :cursor="nicknameCursor"
          @input="onNicknameInput"
        />
        <button class="btn primary" form-type="submit">保存</button>
      </form>
    </view>
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { getScorebookDetail, updateMyProfile } from '../../utils/api'
import { clampNickname } from '../../utils/nickname'

const scorebookId = ref('')
const nickname = ref('')
const avatarUrl = ref('')
const nicknameCursor = ref(0)

const fallbackAvatar =
  'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'

let nicknameInputType = 'text'
// #ifdef MP-WEIXIN
nicknameInputType = 'nickname'
// #endif

onLoad(async (q) => {
  const query = q as any
  scorebookId.value = String(query.id || '')
  nickname.value = clampNickname(decodeURIComponent(String(query.nickname || '')))
  avatarUrl.value = decodeURIComponent(String(query.avatarUrl || ''))

  if (!scorebookId.value) return
  try {
    const res = await getScorebookDetail(scorebookId.value)
    const myID = res?.me?.memberId
    const my = (res?.members || []).find((m: any) => m?.id === myID)
    if (my?.nickname) nickname.value = clampNickname(String(my.nickname))
    if (my?.avatarUrl) avatarUrl.value = String(my.avatarUrl)
  } catch (e) {}
})

function onNicknameInput(e: any) {
  const next = clampNickname(String(e?.detail?.value || ''))
  nickname.value = next
  nicknameCursor.value = next.length
  return next
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
    avatarUrl.value = dataUrl
  } catch (e: any) {
    uni.showToast({ title: '头像处理失败', icon: 'none' })
  }
  // #endif
}

async function save() {
  const nextNickname = clampNickname(nickname.value.trim())
  if (!nextNickname) return uni.showToast({ title: '昵称不能为空', icon: 'none' })
  try {
    await updateMyProfile(scorebookId.value, { nickname: nextNickname, avatarUrl: avatarUrl.value.trim() })
    uni.showToast({ title: '已保存', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 300)
  } catch (e: any) {
    uni.showToast({ title: e?.message || '保存失败', icon: 'none' })
  }
}

async function onSaveSubmit(e: any) {
  const submittedNickname = clampNickname(String(e?.detail?.value?.nickname || '').trim())
  if (submittedNickname !== nickname.value.trim()) nickname.value = submittedNickname
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
.avatar-wrapper {
  margin-top: 12rpx;
  padding: 18rpx 16rpx;
  border-radius: 12rpx;
  background: #f6f7fb;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
}
.avatar-wrapper::after {
  border: none;
}
.avatar {
  width: 120rpx;
  height: 120rpx;
  border-radius: 60rpx;
  background: #fff;
}
.avatar-tip {
  color: #666;
  font-size: 24rpx;
}
.btn {
  margin-top: 20rpx;
}
.primary {
  background: #111;
  color: #fff;
}
</style>
