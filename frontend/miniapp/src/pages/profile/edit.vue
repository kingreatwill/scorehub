<template>
  <view class="page">
    <view class="card">
      <view class="title">{{ titleText }}</view>
      <form @submit="onSaveSubmit">
        <template v-if="isMpWeixin">
          <button class="avatar-wrapper" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
            <image class="avatar" :src="avatarUrl || fallbackAvatar" mode="aspectFill" />
            <view class="avatar-tip">{{ avatarUrl ? '点击更换头像' : '点击选择头像' }}</view>
          </button>
        </template>
        <template v-else>
          <view class="avatar-preview">
            <image class="avatar" :src="avatarUrl || fallbackAvatar" mode="aspectFill" />
            <view class="avatar-tip">头像预览</view>
          </view>
          <input class="input" v-model="avatarUrl" placeholder="头像 URL（可选）" />
        </template>
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
        <input class="input" v-if="showRemark" v-model="remark" placeholder="备注（可选）" />
        <button class="btn" form-type="submit" :disabled="submitting">
          {{ submitting ? '保存中…' : '保存' }}
        </button>
      </form>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { addLedgerMember, getLedgerDetail, getScorebookDetail, updateLedgerMember, updateMe, updateMyProfile } from '../../utils/api'
import { clampNickname } from '../../utils/nickname'

type EditMode = 'me' | 'ledger' | 'scorebook'
type EditAction = 'edit' | 'add'

const mode = ref<EditMode>('me')
const action = ref<EditAction>('edit')
const ledgerId = ref('')
const scorebookId = ref('')
const targetMemberId = ref('')
const memberId = ref('')
const nickname = ref('')
const avatarUrl = ref('')
const remark = ref('')
const nicknameCursor = ref(0)
const submitting = ref(false)
const allowRemark = ref(false)

const isMpWeixin = ref(false)
// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

let nicknameInputType = 'text'
// #ifdef MP-WEIXIN
nicknameInputType = 'nickname'
// #endif

const fallbackAvatar =
  'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'

const showRemark = computed(() => mode.value === 'ledger' && allowRemark.value)
const titleText = computed(() => {
  if (mode.value === 'ledger' && action.value === 'add') return '新增成员'
  if (mode.value === 'ledger') return '修改记账资料'
  if (mode.value === 'scorebook') return '修改得分资料'
  return '修改资料'
})

onLoad(async (q) => {
  const query = (q || {}) as any
  const nextMode = String(query.mode || 'me')
  if (nextMode === 'ledger' || nextMode === 'scorebook' || nextMode === 'me') {
    mode.value = nextMode as EditMode
  }
  const nextAction = String(query.action || 'edit')
  if (nextAction === 'add' || nextAction === 'edit') {
    action.value = nextAction as EditAction
  }
  ledgerId.value = String(query.ledgerId || query.id || '')
  scorebookId.value = String(query.scorebookId || query.id || '')
  targetMemberId.value = String(query.memberId || '').trim()
  nickname.value = clampNickname(decodeURIComponent(String(query.nickname || '')))
  avatarUrl.value = decodeURIComponent(String(query.avatarUrl || ''))
  remark.value = decodeURIComponent(String(query.remark || ''))

  uni.setNavigationBarTitle({ title: titleText.value })
  await loadInitial()
})

function onNicknameInput(e: any) {
  const next = clampNickname(String(e?.detail?.value || ''))
  nickname.value = next
  nicknameCursor.value = next.length
  return next
}

function currentUserId(): string {
  const u = (uni.getStorageSync('user') as any) || null
  return String(u?.id || '').trim()
}

function resolveOwnerUserId(list: any[], createdBy: any): string {
  if (createdBy !== undefined && createdBy !== null && String(createdBy) !== '') {
    return String(createdBy)
  }
  const owner = list.find((m: any) => String(m?.role || '') === 'owner')
  return String(owner?.userId || '')
}

async function loadInitial() {
  if (mode.value === 'me') {
    const u = (uni.getStorageSync('user') as any) || null
    if (!u) {
      uni.showToast({ title: '请先登录', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 300)
      return
    }
    if (u.nickname) nickname.value = clampNickname(String(u.nickname))
    if (u.avatarUrl) avatarUrl.value = String(u.avatarUrl)
    return
  }

  if (mode.value === 'scorebook') {
    if (!scorebookId.value) {
      uni.showToast({ title: '缺少得分簿参数', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 300)
      return
    }
    try {
      const res = await getScorebookDetail(scorebookId.value)
      const myID = res?.me?.memberId
      const my = (res?.members || []).find((m: any) => m?.id === myID)
      if (my?.nickname) nickname.value = clampNickname(String(my.nickname))
      if (my?.avatarUrl) avatarUrl.value = String(my.avatarUrl)
    } catch (e: any) {
      uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
    }
    return
  }

  if (!ledgerId.value) {
    uni.showToast({ title: '缺少记账簿参数', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 300)
    return
  }

  const uid = currentUserId()
  if (!uid) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 300)
    return
  }

  try {
    const res = await getLedgerDetail(ledgerId.value)
    const list = res?.members || []
    const me = list.find((m: any) => String(m?.userId || '') === uid)
    const ownerUserId = resolveOwnerUserId(list, res?.ledger?.createdByUserId)
    const isOwner = !!ownerUserId && ownerUserId === uid
    const meId = String(me?.id || '')
    const requestedId = targetMemberId.value

    if (action.value === 'add' && !isOwner) {
      uni.showToast({ title: '无权限新增成员', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 300)
      return
    }

    if (requestedId && !isOwner && requestedId !== meId) {
      uni.showToast({ title: '无权限编辑成员', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 300)
      return
    }

    if (action.value !== 'add') {
      const target = requestedId ? list.find((m: any) => String(m?.id || '') === requestedId) : me
      if (!target?.id) {
        uni.showToast({ title: '未找到成员信息', icon: 'none' })
        setTimeout(() => uni.navigateBack(), 300)
        return
      }
      memberId.value = String(target.id)
      if (target?.nickname) nickname.value = clampNickname(String(target.nickname))
      if (target?.avatarUrl) avatarUrl.value = String(target.avatarUrl)
      remark.value = String(target?.remark || '')
    } else {
      memberId.value = ''
      nickname.value = ''
      avatarUrl.value = ''
      remark.value = ''
    }
    allowRemark.value = isOwner
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  }
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
  if (submitting.value) return
  const nextNickname = clampNickname(nickname.value.trim())
  const nextAvatar = avatarUrl.value.trim()
  if (!nextNickname) {
    uni.showToast({ title: '昵称不能为空', icon: 'none' })
    return
  }

  submitting.value = true
  try {
    if (mode.value === 'me') {
      const res = await updateMe({ nickname: nextNickname, avatarUrl: nextAvatar })
      if (res?.user?.nickname) nickname.value = clampNickname(String(res.user.nickname))
      if (res?.user?.avatarUrl) avatarUrl.value = String(res.user.avatarUrl)
    } else if (mode.value === 'scorebook') {
      await updateMyProfile(scorebookId.value, { nickname: nextNickname, avatarUrl: nextAvatar })
    } else {
      if (!ledgerId.value) {
        uni.showToast({ title: '缺少成员参数', icon: 'none' })
        return
      }
      if (action.value === 'add') {
        await addLedgerMember(ledgerId.value, {
          nickname: nextNickname,
          avatarUrl: nextAvatar,
          ...(showRemark.value ? { remark: remark.value.trim() } : {}),
        })
      } else {
        if (!memberId.value) {
          uni.showToast({ title: '缺少成员参数', icon: 'none' })
          return
        }
        await updateLedgerMember(ledgerId.value, memberId.value, {
          nickname: nextNickname,
          avatarUrl: nextAvatar,
          ...(showRemark.value ? { remark: remark.value.trim() } : {}),
        })
      }
    }
    uni.showToast({ title: '已保存', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 300)
  } catch (e: any) {
    uni.showToast({ title: e?.message || '保存失败', icon: 'none' })
  } finally {
    submitting.value = false
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
.avatar-preview {
  margin-top: 12rpx;
  padding: 18rpx 16rpx;
  border-radius: 12rpx;
  background: #f6f7fb;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
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
