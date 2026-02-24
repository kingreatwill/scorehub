<template>
  <view class="page" :style="themeStyle">
    <view class="card" v-if="!token">
      <view class="title">登录（Web）</view>
      <view class="hint">请输入在小程序内绑定的用户名与密码</view>
      <view class="form">
        <input class="input" v-model="username" placeholder="用户名（a-z/0-9/_，3-32）" />
        <input class="input" v-model="password" placeholder="密码（8-72）" password />
        <button class="btn confirm-btn" :disabled="loggingIn" @click="onPasswordLogin">
          {{ loggingIn ? '提交中…' : '登录' }}
        </button>
      </view>
    </view>

    <view class="card" v-else>
      <view class="title">账号与安全</view>
      <view class="hint">当前用户名：{{ user?.username || '未绑定' }}</view>

      <view class="divider" />

      <template v-if="!user?.hasPassword">
        <view class="subTitle">绑定用户名密码</view>
        <view class="hint">仅首次可设置；用于 Web 登录</view>
        <view class="form">
          <input class="input" v-model="bindUsername" placeholder="用户名（a-z/0-9/_，3-32）" />
          <input class="input" v-model="bindPassword" placeholder="密码（8-72）" password />
          <button class="btn confirm-btn" :disabled="saving" @click="onSetCredentials">
            {{ saving ? '提交中…' : '绑定' }}
          </button>
        </view>
      </template>
      <template v-else>
        <view class="subTitle">修改密码</view>
        <view class="form">
          <input class="input" v-model="oldPassword" placeholder="旧密码" password />
          <input class="input" v-model="newPassword" placeholder="新密码（8-72）" password />
          <button class="btn confirm-btn" :disabled="saving" @click="onChangePassword">
            {{ saving ? '提交中…' : '修改' }}
          </button>
        </view>
      </template>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { changePassword, passwordLogin, setCredentials } from '../../utils/api'
import { buildThemeVars, getThemeBaseColor } from '../../utils/theme'

const token = ref('')
const user = ref<any>(null)

const username = ref('')
const password = ref('')
const loggingIn = ref(false)

const bindUsername = ref('')
const bindPassword = ref('')
const oldPassword = ref('')
const newPassword = ref('')
const saving = ref(false)

const themeBase = ref('#111111')
const themeStyle = computed(() => buildThemeVars(themeBase.value))

onShow(() => {
  themeBase.value = getThemeBaseColor()
  token.value = (uni.getStorageSync('token') as string) || ''
  user.value = (uni.getStorageSync('user') as any) || null
  bindUsername.value = String(user.value?.username || '')
})

async function onPasswordLogin() {
  const u = username.value.trim()
  const p = password.value
  if (!u) {
    uni.showToast({ title: '请输入用户名', icon: 'none' })
    return
  }
  if (!p) {
    uni.showToast({ title: '请输入密码', icon: 'none' })
    return
  }
  if (loggingIn.value) return
  loggingIn.value = true
  try {
    const res = await passwordLogin(u, p)
    token.value = res.token
    user.value = res.user
    uni.showToast({ title: '登录成功', icon: 'success' })
  } catch (e: any) {
    try {
      console.error('password login failed', e)
    } catch {}
    const msg = String(e?.message || e || '登录失败')
    uni.showToast({ title: msg.slice(0, 120), icon: 'none' })
  } finally {
    loggingIn.value = false
  }
}

async function onSetCredentials() {
  const u = bindUsername.value.trim()
  const p = bindPassword.value
  if (!u) {
    uni.showToast({ title: '请输入用户名', icon: 'none' })
    return
  }
  if (!p) {
    uni.showToast({ title: '请输入密码', icon: 'none' })
    return
  }
  if (saving.value) return
  saving.value = true
  try {
    const res = await setCredentials({ username: u, password: p })
    user.value = res.user
    uni.showToast({ title: '绑定成功', icon: 'success' })
  } catch (e: any) {
    try {
      console.error('set credentials failed', e)
    } catch {}
    const msg = String(e?.message || e || '绑定失败')
    uni.showToast({ title: msg.slice(0, 120), icon: 'none' })
  } finally {
    saving.value = false
  }
}

async function onChangePassword() {
  const oldP = oldPassword.value
  const newP = newPassword.value
  if (!oldP) {
    uni.showToast({ title: '请输入旧密码', icon: 'none' })
    return
  }
  if (!newP) {
    uni.showToast({ title: '请输入新密码', icon: 'none' })
    return
  }
  if (saving.value) return
  saving.value = true
  try {
    const res = await changePassword({ oldPassword: oldP, newPassword: newP })
    user.value = res.user
    oldPassword.value = ''
    newPassword.value = ''
    uni.showToast({ title: '修改成功', icon: 'success' })
  } catch (e: any) {
    try {
      console.error('change password failed', e)
    } catch {}
    const msg = String(e?.message || e || '修改失败')
    uni.showToast({ title: msg.slice(0, 120), icon: 'none' })
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.page {
  padding: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}
.card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
}
.title {
  font-size: 32rpx;
  font-weight: 700;
  margin-bottom: 10rpx;
}
.subTitle {
  font-size: 30rpx;
  font-weight: 700;
  margin-top: 8rpx;
}
.hint {
  color: #666;
  font-size: 26rpx;
}
.form {
  margin-top: 16rpx;
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.btn {
  margin-top: 8rpx;
  width: 100%;
  display: block;
}
.input {
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 28rpx;
}
.divider {
  height: 1rpx;
  background: rgba(0, 0, 0, 0.08);
  margin: 18rpx 0;
}
</style>

