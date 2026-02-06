<template>
  <view class="page" v-if="ledger">
    <view class="card hero">
      <view class="row hero-row">
        <view class="name">{{ ledger.name }}</view>
        <view class="badge" :class="{ ended: ledger.status === 'ended' }">
          <view class="badge-dot" :class="{ ended: ledger.status === 'ended' }" />
          <text>{{ ledger.status === 'ended' ? '已结束' : '记录中' }}</text>
        </view>
      </view>
      <view class="sub hero-sub">
        <view class="pill" v-if="ledger.createdAt">{{ formatTime(ledger.createdAt) }}</view>
        <view class="pill">成员 {{ members.length }}</view>
        <view class="pill">记录 {{ records.length }}</view>
        <view class="pill code" v-if="canShowInvite" @click="copyInvite">
          邀请码: <text class="mono">{{ ledger.inviteCode }}</text>
          <view class="qr-icon" v-if="canShowInvite" @click.stop="openInviteCodeQR">
            <view class="qr-finder tl"><view class="qr-finder-inner" /></view>
            <view class="qr-finder tr"><view class="qr-finder-inner" /></view>
            <view class="qr-finder bl"><view class="qr-finder-inner" /></view>
            <view class="qr-dot d1" />
            <view class="qr-dot d2" />
            <view class="qr-dot d3" />
            <view class="qr-dot d4" />
            <view class="qr-dot d5" />
            <view class="qr-dot d6" />
            <view class="qr-dot d7" />
          </view>
        </view>
        <view class="pill warn" v-if="ledger.shareDisabled">已禁止分享</view>
        <view class="pill readonly" v-if="shareMode">分享只读</view>
      </view>
    </view>

    <view class="card member-card">
      <view class="title-row">
        <view class="title">人员</view>
        <view class="title-actions" v-if="!isReadonly">
          <button class="icon-btn primary" @click="openMemberModal">
            <image class="icon-img" :src="addIcon" mode="aspectFit" />
          </button>
        </view>
      </view>
      <view class="tip" v-if="isOwner">点成员记账，右上角可编辑</view>
      <view class="tip" v-else-if="boundMember">点自己可修改头像/昵称</view>
      <view class="tip" v-else>仅可查看已加入的账本</view>
      <view class="hint" v-if="members.length === 0">暂无人员</view>
      <view class="member-grid" v-else>
        <view class="member" :class="{ disabled: isReadonly }" v-for="m in members" :key="m.id" @click="onClickMember(m)">
          <button class="member-edit" v-if="canEditMember(m)" @click.stop="openEditMember(m)">
            <image class="icon-img small" :src="editIcon" mode="aspectFit" />
          </button>
          <view class="member-tags" v-if="isOwnerMember(m) || isMeMember(m)">
            <view class="tag owner-tag" v-if="isOwnerMember(m)">账主</view>
            <view class="tag me-tag" v-if="isMeMember(m)">我</view>
          </view>
          <view class="avatar-wrap">
            <image class="avatar" :src="m.avatarUrl || fallbackAvatar" mode="aspectFill" />
          </view>
          <view class="member-name">{{ displayNickname(m.nickname) }}</view>
          <view class="member-total" :class="balanceTone(memberTotals[m.id])">{{ formatAmount(memberTotals[m.id]) }}</view>
        </view>
      </view>
    </view>

    <view class="card">
      <view class="title-row">
        <view class="title">记录</view>
      </view>
      <view class="filter-panel" v-if="filterOptions.length">
        <view class="filter-title">筛选人员</view>
        <scroll-view class="filter-scroll" scroll-x>
          <view class="filter-chip" :class="{ active: !filterMember }" @click="clearFilter">全部</view>
          <view
            class="filter-chip"
            v-for="m in filterOptions"
            :key="m.id"
            :class="{ active: filterMemberId === m.id }"
            @click="setFilter(m.id)"
          >
            {{ displayNickname(m.nickname) }}
          </view>
          <view class="filter-chip" v-if="!isReadonly && filterMember" @click="openRecordModal(filterMember)">
            给他记账
          </view>
        </scroll-view>
      </view>
      <view class="hint" v-if="filteredRecords.length === 0">暂无记录</view>
      <view class="records" v-else>
        <view class="record" v-for="r in filteredRecords" :key="r.id">
          <view class="record-row">
            <view class="record-user">
              <image class="record-avatar" :src="avatarOf(recordMemberId(r))" mode="aspectFill" />
              <text class="record-name">{{ nicknameOf(recordMemberId(r)) }}</text>
            </view>
            <view class="record-meta">
              <text class="record-type" :class="recordTypeClass(r)">{{ recordTypeLabel(r) }}</text>
              <text class="record-amount" v-if="r.type !== 'remark'" :class="recordTypeClass(r)">{{ formatRecordAmount(r) }}</text>
              <text class="record-time">{{ formatTime(r.createdAt) }}</text>
            </view>
          </view>
          <view class="record-note" v-if="r.note">{{ r.note }}</view>
        </view>
      </view>
      
    </view>

    <view class="modal-mask" v-if="memberModalOpen" @click="closeMemberModal" />
    <view class="modal" v-if="memberModalOpen">
      <view class="modal-head">
        <view class="modal-title">{{ memberModalMode === 'edit' ? '编辑人员' : '新增人员' }}</view>
        <view class="modal-close" @click="closeMemberModal">×</view>
      </view>

      <view class="form">
        <template v-if="isMpWeixin">
          <button class="avatar-wrapper" open-type="chooseAvatar" @chooseavatar="onChooseAvatar" hover-class="none">
            <image class="avatar" :src="memberAvatar || fallbackAvatar" mode="aspectFill" />
            <view class="avatar-tip">点击选择头像（可选）</view>
          </button>
        </template>
        <template v-else>
          <view class="avatar-preview">
            <image class="avatar" :src="memberAvatar || fallbackAvatar" mode="aspectFill" />
          </view>
          <input class="input" v-model="memberAvatar" placeholder="头像 URL（可选）" />
        </template>
        <input class="input" v-model="memberNickname" placeholder="昵称" />
        <input class="input" v-if="isOwner" v-model="memberRemark" placeholder="备注（可选）" />
        <button class="btn" :disabled="memberSubmitting" @click="submitMember">
          {{ memberSubmitting ? '提交中…' : '保存' }}
        </button>
      </view>
    </view>

    <view class="modal-mask" v-if="recordModalOpen" @click="closeRecordModal" />
    <view class="modal" v-if="recordModalOpen">
      <view class="modal-head">
        <view class="modal-title">记账</view>
        <view class="modal-close" @click="closeRecordModal">×</view>
      </view>

      <view class="record-target" v-if="recordTarget">
        <image class="record-target-avatar" :src="recordTarget.avatarUrl || fallbackAvatar" mode="aspectFill" />
        <view class="record-target-body">
          <view class="record-target-name">{{ displayNickname(recordTarget.nickname) }}</view>
          <view class="record-target-sub">当前余额 {{ formatAmount(memberTotals[recordTarget.id]) }}</view>
        </view>
      </view>

      <view class="type-select">
        <view class="type-option" :class="{ active: recordType === 'expense' }" @click="setRecordType('expense')">
          <view class="type-radio" />
          <view class="type-label">支出</view>
        </view>
        <view class="type-option" :class="{ active: recordType === 'income' }" @click="setRecordType('income')">
          <view class="type-radio" />
          <view class="type-label">收入</view>
        </view>
      </view>

      <input class="input" type="digit" v-model="recordAmount" placeholder="金额" />
      <input class="input" v-model="recordNote" placeholder="备注（可选）" />

      <view class="modal-actions">
        <button class="confirm-btn" :disabled="recordSubmitting" @click="submitRecord">
          {{ recordSubmitting ? '提交中…' : '确认记账' }}
        </button>
      </view>
    </view>

    <view class="modal-mask" v-if="bindModalOpen" @click="closeBindModal" />
    <view class="modal" v-if="bindModalOpen">
      <view class="modal-head">
        <view class="modal-title">绑定人员</view>
        <view class="modal-close" @click="closeBindModal">×</view>
      </view>
      <view class="hint">请选择一个已有人员进行绑定（仅限未绑定的成员）。</view>
      <view class="bind-list" v-if="bindCandidates.length">
        <view class="bind-item" v-for="m in bindCandidates" :key="m.id">
          <image class="bind-avatar" :src="m.avatarUrl || fallbackAvatar" mode="aspectFill" />
          <view class="bind-info">
            <view class="bind-name">{{ displayNickname(m.nickname) }}</view>
            <view class="bind-sub">未绑定</view>
          </view>
          <button size="mini" class="bind-btn" :disabled="binding" @click="submitBind(m)">绑定</button>
        </view>
      </view>
      <view class="hint" v-else>暂无可绑定人员</view>
      <view class="modal-actions">
        <button size="mini" v-if="!currentUserId" @click="goLoginFromBind">去登录</button>
      </view>
    </view>

    <view class="modal-mask" v-if="qrModalOpen" @click="closeQRCode" />
    <view class="modal" v-if="qrModalOpen">
      <view class="modal-title">小程序码</view>
      <view v-if="qrLoading" class="hint">生成中…</view>
      <image v-else class="qr" :src="qrSrc" mode="widthFix" @click="previewQRCode" />
      <view class="modal-actions">
        <button size="mini" @click="closeQRCode">关闭</button>
      </view>
    </view>

    <view class="modal-mask" v-if="inviteQRModalOpen" @click="closeInviteCodeQR" />
    <view class="modal" v-if="inviteQRModalOpen">
      <view class="modal-title">邀请码二维码</view>
      <view v-if="inviteQRLoading" class="hint">生成中…</view>
      <canvas
        class="invite-qr-canvas"
        canvas-id="inviteQrCanvas"
        id="inviteQrCanvas"
        :style="{ width: `${inviteQRSize}px`, height: `${inviteQRSize}px` }"
        :width="inviteQRSize"
        :height="inviteQRSize"
      />
      <view class="hint">在首页点「扫码加入」即可识别</view>
      <view class="modal-actions">
        <button size="mini" @click="closeInviteCodeQR">关闭</button>
      </view>
    </view>

    <view class="fab-mask" v-if="actionMenuOpen && hasActions" @click="closeActionMenu" />
    <view class="fab" v-if="hasActions">
      <view class="fab-panel" :class="{ open: actionMenuOpen }">
        <button size="mini" class="action-btn" v-if="canOpenQRCode" @click="closeActionMenu(); openQRCode()">
          小程序码
        </button>
        <!-- #ifdef MP-WEIXIN -->
        <button
          size="mini"
          class="action-btn"
          v-if="canShare"
          open-type="share"
          @click="closeActionMenu"
        >
          分享
        </button>
        <!-- #endif -->
        <button size="mini" class="action-btn" v-if="canManageLedger" @click="closeActionMenu(); renameLedger()">
          改名
        </button>
        <button size="mini" class="action-btn" v-if="canManageLedger" @click="closeActionMenu(); toggleShare()">
          {{ ledger.shareDisabled ? '允许分享' : '禁止分享' }}
        </button>
        <button size="mini" class="action-btn danger" v-if="canManageLedger" @click="closeActionMenu(); endCurrent()">
          结束
        </button>
      </view>
      <button class="fab-toggle" :class="{ active: actionMenuOpen }" @click="toggleActionMenu">
        <image class="fab-icon" :src="actionMenuOpen ? closeIcon : moreIcon" mode="aspectFit" />
      </button>
    </view>
  </view>

  <view class="page" v-else-if="loading">
    <view class="card">
      <view class="title">加载中…</view>
      <view class="hint">正在获取记账簿数据</view>
    </view>
  </view>

  <view class="page" v-else-if="notFound">
    <view class="card">
      <view class="title">记账簿不存在</view>
      <view class="hint">该记账簿可能仅保存在创建设备中。</view>
      <button class="btn" @click="goList">返回列表</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, getCurrentInstance, nextTick, ref } from 'vue'
import { onLoad, onShareAppMessage, onShow } from '@dcloudio/uni-app'
import {
  addLedgerMember,
  addLedgerRecord,
  bindLedgerMember,
  endLedger,
  getLedgerInviteQRCode,
  getLedgerDetail,
  updateLedger,
  updateLedgerMember,
  updateLedgerName,
} from '../../utils/api'
import { makeInviteCodeQRMatrix } from '../../utils/qrcode'

const id = ref('')
const ledger = ref<any>(null)
const members = ref<any[]>([])
const records = ref<any[]>([])
const shareMode = ref(false)
const loading = ref(false)
const notFound = ref(false)
const isMpWeixin = ref(false)
// #ifdef MP-WEIXIN
isMpWeixin.value = true
// #endif

const memberModalOpen = ref(false)
const memberNickname = ref('')
const memberAvatar = ref('')
const memberRemark = ref('')
const memberSubmitting = ref(false)
const memberModalMode = ref<'add' | 'edit'>('add')
const editingMember = ref<any | null>(null)
const actionMenuOpen = ref(false)

const recordModalOpen = ref(false)
const recordTarget = ref<any | null>(null)
const recordType = ref<'income' | 'expense'>('expense')
const recordAmount = ref('')
const recordNote = ref('')
const recordSubmitting = ref(false)
const filterMemberId = ref('')
const qrModalOpen = ref(false)
const qrLoading = ref(false)
const qrSrc = ref('')
const inviteQRModalOpen = ref(false)
const inviteQRLoading = ref(false)
const inviteQRSize = 232
const currentUserId = ref('')
const bindModalOpen = ref(false)
const binding = ref(false)
const bindRequired = ref(false)

const fallbackAvatar =
  'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'
const addIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14"/><path d="M5 12h14"/></svg>'
const editIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="%23111" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5z"/></svg>'
const moreIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="%23ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="5" cy="12" r="1.8"/><circle cx="12" cy="12" r="1.8"/><circle cx="19" cy="12" r="1.8"/></svg>'
const closeIcon =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="%23ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18"/><path d="M6 6l12 12"/></svg>'

const isOwner = computed(() => {
  if (!currentUserId.value) return false
  const createdBy = ledger.value?.createdByUserId
  if (createdBy !== undefined && createdBy !== null && String(createdBy) !== '') {
    return String(createdBy) === currentUserId.value
  }
  const owner = members.value.find((m) => String(m?.role || '') === 'owner')
  return String(owner?.userId || '') === currentUserId.value
})
const isReadonly = computed(() => shareMode.value || ledger.value?.status === 'ended' || !isOwner.value)
const canOpenQRCode = computed(
  () => !!ledger.value && isOwner.value && ledger.value.status !== 'ended' && !ledger.value.shareDisabled,
)
const canShare = computed(() => !!ledger.value && isOwner.value && !ledger.value.shareDisabled)
const canManageLedger = computed(() => !!ledger.value && isOwner.value && ledger.value.status !== 'ended')
const hasActions = computed(() => canOpenQRCode.value || canShare.value || canManageLedger.value)
const canShowInvite = computed(() => {
  if (!ledger.value?.inviteCode) return false
  if (isOwner.value) return true
  if (ledger.value?.shareDisabled) return false
  return !!boundMember.value
})
const meMemberId = computed(() => {
  const me = members.value.find((m) => String(m?.role || '') === 'owner')
  return me?.id || ''
})
const filterMember = computed(() => members.value.find((m) => m.id === filterMemberId.value) || null)
const filterOptions = computed(() => {
  const list = members.value.filter((m) => String(m?.role || '') !== 'owner')
  return list.length > 0 ? list : members.value
})
const showFilterHint = computed(
  () =>
    !filterMember.value &&
    !isReadonly.value &&
    filterOptions.value.length > 0 &&
    filteredRecords.value.length > 0,
)

const memberTotals = computed(() => {
  const out: Record<string, number> = {}
  const hasScore = members.value.some((m) => m?.score !== undefined && m?.score !== null)
  if (!hasScore) {
    for (const r of records.value) {
      const fromId = String(r?.fromMemberId || '')
      const toId = String(r?.toMemberId || '')
      const delta = Number(r?.amount || 0)
      if (!fromId || !toId || !Number.isFinite(delta) || delta <= 0) continue
      out[fromId] = (out[fromId] || 0) - delta
      out[toId] = (out[toId] || 0) + delta
    }
    return out
  }
  for (const m of members.value) {
    const score = Number(m?.score || 0)
    out[m.id] = Number.isFinite(score) ? score : 0
  }
  return out
})

const filteredRecords = computed(() => {
  const meId = meMemberId.value
  const targetId = filterMemberId.value
  if (!targetId || !meId) return records.value
  return records.value.filter((r) => {
    const fromId = String(r?.fromMemberId || '')
    const toId = String(r?.toMemberId || '')
    return (
      (fromId === meId && toId === targetId) ||
      (fromId === targetId && toId === meId)
    )
  })
})

onLoad((q) => {
  const query = (q || {}) as any
  id.value = String(query.id || '')
  shareMode.value = String(query.share || query.readonly || '') === '1'
  bindRequired.value = shareMode.value || String(query.bind || '') === '1'
  syncCurrentUser()
  loadLedger()
})

onShow(() => {
  syncCurrentUser()
  loadLedger()
})

onShareAppMessage(() => {
  const name = ledger.value?.name || '记账簿'
  const path = `/pages/ledger/detail?id=${encodeURIComponent(id.value)}&share=1`
  return { title: `记账簿：${name}`, path }
})

async function loadLedger() {
  if (!id.value) {
    ledger.value = null
    members.value = []
    records.value = []
    notFound.value = false
    loading.value = false
    return
  }
  loading.value = true
  notFound.value = false
  try {
    const res = await getLedgerDetail(id.value)
    ledger.value = res.ledger
    members.value = res.members || []
    records.value = res.records || []
    maybeOpenBind()
  } catch (e: any) {
    ledger.value = null
    members.value = []
    records.value = []
    notFound.value = String(e?.code || '') === 'not_found'
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

function openMemberModal() {
  if (isReadonly.value) return
  memberModalMode.value = 'add'
  editingMember.value = null
  memberNickname.value = ''
  memberAvatar.value = ''
  memberRemark.value = ''
  memberModalOpen.value = true
}

function closeMemberModal() {
  if (memberSubmitting.value) return
  editingMember.value = null
  memberModalOpen.value = false
}

async function submitMember() {
  if (!ledger.value || memberSubmitting.value) return
  const nickname = memberNickname.value.trim()
  if (!nickname) {
    uni.showToast({ title: '请输入昵称', icon: 'none' })
    return
  }
  const prevRemark =
    memberModalMode.value === 'edit' && editingMember.value
      ? String(editingMember.value?.remark || '').trim()
      : ''
  let shouldReload = false
  memberSubmitting.value = true
  try {
    const remark = isOwner.value ? memberRemark.value.trim() : ''
    if (memberModalMode.value === 'edit' && editingMember.value) {
      const res = await updateLedgerMember(id.value, editingMember.value.id, {
        nickname,
        avatarUrl: memberAvatar.value,
        ...(isOwner.value ? { remark } : {}),
      })
      if (res?.member) {
        members.value = members.value.map((m) => (m.id === res.member.id ? res.member : m))
        if (isOwner.value && prevRemark !== remark) {
          shouldReload = true
        }
      }
    } else {
      const res = await addLedgerMember(id.value, {
        nickname,
        avatarUrl: memberAvatar.value,
        ...(isOwner.value ? { remark } : {}),
      })
      if (res?.member) members.value = [...members.value, res.member]
    }
    memberModalOpen.value = false
    if (shouldReload) {
      await loadLedger()
    }
  } catch (e: any) {
    uni.showToast({ title: e?.message || (memberModalMode.value === 'edit' ? '更新失败' : '新增失败'), icon: 'none' })
  } finally {
    memberSubmitting.value = false
  }
}

function onClickMember(member: any) {
  if (isMeMember(member) && canEditMember(member)) {
    openEditMember(member)
    return
  }
  if (isReadonly.value) return
  openRecordModal(member)
}

function closeRecordModal() {
  if (recordSubmitting.value) return
  recordModalOpen.value = false
}

function syncCurrentUser() {
  const u = (uni.getStorageSync('user') as any) || null
  currentUserId.value = u?.id ? String(u.id) : ''
}

function isMeMember(member: any): boolean {
  if (!currentUserId.value) return false
  return String(member?.userId || '') === currentUserId.value
}

function isOwnerMember(member: any): boolean {
  return String(member?.role || '') === 'owner'
}

function canEditMember(member: any): boolean {
  if (shareMode.value) return false
  return isOwner.value || isMeMember(member)
}

const boundMember = computed(() => {
  if (!currentUserId.value) return null
  return members.value.find((m) => String(m?.userId || '') === currentUserId.value) || null
})

const bindCandidates = computed(() => members.value.filter((m) => !m?.userId))

function maybeOpenBind() {
  if (!bindRequired.value) return
  if (!ledger.value) return
  if (boundMember.value) {
    bindModalOpen.value = false
    return
  }
  if (bindCandidates.value.length === 0) {
    bindModalOpen.value = false
    return
  }
  bindModalOpen.value = true
}

function closeBindModal() {
  if (binding.value) return
  bindModalOpen.value = false
}

async function submitBind(member: any) {
  if (!member || binding.value) return
  const token = (uni.getStorageSync('token') as string) || ''
  if (!token) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    return
  }
  binding.value = true
  try {
    const u = (uni.getStorageSync('user') as any) || null
    const payload = {
      memberId: String(member.id),
      nickname: String(u?.nickname || '').trim(),
      avatarUrl: String(u?.avatarUrl || '').trim(),
    }
    const res = await bindLedgerMember(id.value, payload)
    if (res?.member) {
      members.value = members.value.map((m) => (m.id === res.member.id ? res.member : m))
      bindModalOpen.value = false
    }
  } catch (e: any) {
    uni.showToast({ title: e?.message || '绑定失败', icon: 'none' })
  } finally {
    binding.value = false
  }
}

function goLoginFromBind() {
  uni.setStorageSync('scorehub.afterLogin', { to: 'ledger', url: `/pages/ledger/detail?id=${encodeURIComponent(id.value)}&bind=1`, ts: Date.now() })
  uni.switchTab({ url: '/pages/my/index' })
}

function ledgerQRCodeCacheKey(ledgerID: string): string {
  return `scorehub.ledgerInviteQRCode.${encodeURIComponent(String(ledgerID || '').trim())}`
}

function getCachedLedgerQRCode(ledgerID: string): string {
  const key = ledgerQRCodeCacheKey(ledgerID)
  if (!key) return ''
  try {
    const v: any = uni.getStorageSync(key)
    if (!v) return ''
    if (typeof v === 'string') return v
    if (typeof v === 'object' && v) {
      const src = String(v.src || '').trim()
      return src
    }
    return ''
  } catch (e) {
    return ''
  }
}

function setCachedLedgerQRCode(ledgerID: string, src: string) {
  const key = ledgerQRCodeCacheKey(ledgerID)
  if (!key) return
  const next = String(src || '').trim()
  if (!next) return
  try {
    uni.setStorageSync(key, { src: next, ts: Date.now() })
  } catch (e) {
    // ignore storage errors
  }
}

async function openQRCode() {
  if (ledger.value?.status === 'ended') {
    uni.showToast({ title: '已结束，不能加入', icon: 'none' })
    return
  }
  if (!isOwner.value) {
    uni.showToast({ title: '仅账主可分享', icon: 'none' })
    return
  }
  const token = (uni.getStorageSync('token') as string) || ''
  if (!token) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    return
  }
  qrModalOpen.value = true
  qrLoading.value = true
  try {
    const cached = getCachedLedgerQRCode(id.value)
    if (cached) {
      qrSrc.value = cached
      return
    }
    const fresh = await getLedgerInviteQRCode(id.value)
    qrSrc.value = fresh
    setCachedLedgerQRCode(id.value, fresh)
  } catch (e: any) {
    uni.showToast({ title: e?.message || '生成二维码失败', icon: 'none' })
    qrModalOpen.value = false
  } finally {
    qrLoading.value = false
  }
}

function closeQRCode() {
  qrModalOpen.value = false
}

function toggleActionMenu() {
  if (!hasActions.value) return
  actionMenuOpen.value = !actionMenuOpen.value
}

function closeActionMenu() {
  actionMenuOpen.value = false
}

function previewQRCode() {
  if (!qrSrc.value) return
  uni.previewImage({ urls: [qrSrc.value] })
}

function openRecordModal(member: any) {
  if (isReadonly.value || !member) return
  recordTarget.value = member
  recordType.value = 'expense'
  recordAmount.value = ''
  recordNote.value = ''
  recordModalOpen.value = true
}


function setRecordType(type: 'income' | 'expense') {
  recordType.value = type
}

async function submitRecord() {
  if (!ledger.value || recordSubmitting.value || !recordTarget.value) return
  const rawAmount = Number(recordAmount.value)
  if (!Number.isFinite(rawAmount) || rawAmount <= 0) {
    uni.showToast({ title: '请输入有效金额', icon: 'none' })
    return
  }
  if (!isTwoDecimals(rawAmount)) {
    uni.showToast({ title: '最多两位小数', icon: 'none' })
    return
  }
  const amount = roundToTwo(rawAmount)
  recordSubmitting.value = true
  try {
    const targetRemark = String(recordTarget.value?.remark || '').trim()
    const note = recordNote.value.trim() || targetRemark
    const res = await addLedgerRecord(id.value, {
      memberId: recordTarget.value.id,
      type: recordType.value,
      amount,
      note,
    })
    if (res?.record) {
      records.value = [res.record, ...records.value]
      applyRecordToMembers(res.record)
    }
    recordModalOpen.value = false
  } catch (e: any) {
    uni.showToast({ title: e?.message || '记账失败', icon: 'none' })
  } finally {
    recordSubmitting.value = false
  }
}

async function endCurrent() {
  if (!ledger.value) return
  const res = await new Promise<UniApp.ShowModalRes>((resolve) => {
    uni.showModal({ title: '结束记账', content: '确定结束当前记账簿？', success: resolve })
  })
  if (!res.confirm) return
  try {
    const updated = await endLedger(id.value)
    if (updated?.ledger) ledger.value = updated.ledger
  } catch (e: any) {
    uni.showToast({ title: e?.message || '结束失败', icon: 'none' })
  }
}

async function renameLedger() {
  const current = ledger.value?.name || ''
  uni.showModal({
    title: '修改名称',
    editable: true,
    placeholderText: current,
    success: async (res) => {
      if (!res.confirm) return
      const name = String((res as any).content || '').trim()
      if (!name) return
      try {
        const updated = await updateLedgerName(id.value, name)
        if (updated?.ledger) ledger.value = updated.ledger
      } catch (e: any) {
        uni.showToast({ title: e?.message || '修改失败', icon: 'none' })
      }
    },
  } as any)
}

async function toggleShare() {
  if (!ledger.value || !isOwner.value) return
  const next = !ledger.value.shareDisabled
  const title = next ? '禁止分享' : '允许分享'
  const content = next ? '禁止分享后，其他人不能再加入。' : '允许分享后，其他人可通过邀请码加入。'
  const res = await new Promise<UniApp.ShowModalRes>((resolve) => {
    uni.showModal({ title, content, success: resolve })
  })
  if (!res.confirm) return
  try {
    const updated = await updateLedger(id.value, { shareDisabled: next })
    if (updated?.ledger) ledger.value = updated.ledger
  } catch (e: any) {
    uni.showToast({ title: e?.message || '更新失败', icon: 'none' })
  }
}

function goList() {
  uni.navigateTo({ url: '/pages/ledger/list' })
}

function setFilter(memberId: string) {
  const idValue = String(memberId || '')
  if (!idValue) return
  filterMemberId.value = idValue
}

function clearFilter() {
  filterMemberId.value = ''
}

function openEditMember(member: any) {
  if (!member) return
  memberModalMode.value = 'edit'
  editingMember.value = member
  memberNickname.value = String(member?.nickname || '')
  memberAvatar.value = String(member?.avatarUrl || '')
  memberRemark.value = String(member?.remark || '')
  memberModalOpen.value = true
}

function displayNickname(v: any): string {
  const s = String(v || '').trim()
  return s || '未命名'
}

function nicknameOf(memberId: string): string {
  const m = members.value.find((it) => it.id === memberId)
  return displayNickname(m?.nickname)
}

function avatarOf(memberId: string): string {
  const m = members.value.find((it) => it.id === memberId)
  return m?.avatarUrl || fallbackAvatar
}

function recordMemberId(r: any): string {
  return String(r?.memberId || r?.fromMemberId || r?.toMemberId || '')
}

function applyRecordToMembers(record: any) {
  const fromId = String(record?.fromMemberId || '')
  const toId = String(record?.toMemberId || '')
  const delta = Number(record?.amount || 0)
  if (!fromId || !toId || !Number.isFinite(delta) || delta <= 0) return
  members.value = members.value.map((m) => {
    if (m.id === fromId) {
      const score = Number(m?.score || 0)
      return { ...m, score: (Number.isFinite(score) ? score : 0) - delta }
    }
    if (m.id === toId) {
      const score = Number(m?.score || 0)
      return { ...m, score: (Number.isFinite(score) ? score : 0) + delta }
    }
    return m
  })
}

function formatAmount(v: any): string {
  const n = Number(v || 0)
  if (!Number.isFinite(n)) return '0'
  return n.toFixed(2).replace(/\.00$/, '')
}

function roundToTwo(v: number): number {
  return Math.round(v * 100) / 100
}

function isTwoDecimals(v: number): boolean {
  if (!Number.isFinite(v)) return false
  return Math.abs(v * 100 - Math.round(v * 100)) < 1e-6
}

function recordTypeLabel(r: any): string {
  if (r?.type === 'remark') return '备注'
  return r?.type === 'income' ? '收入' : '支出'
}

function recordTypeClass(r: any): string {
  if (r?.type === 'remark') return 'remark'
  return r?.type === 'income' ? 'pos' : 'neg'
}

function formatRecordAmount(r: any): string {
  if (r?.type === 'remark') return ''
  const sign = r.type === 'income' ? '+' : '-'
  return `${sign}${formatAmount(r.amount)}`
}

function balanceTone(v: any): string {
  const n = Number(v || 0)
  if (n > 0) return 'pos'
  if (n < 0) return 'neg'
  return 'zero'
}

function formatTime(v: any): string {
  const d = new Date(String(v || ''))
  if (Number.isNaN(d.getTime())) return ''
  const now = new Date()
  const yyyy = String(d.getFullYear())
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  if (d.getFullYear() === now.getFullYear()) return `${mm}-${dd} ${hh}:${mi}`
  return `${yyyy}-${mm}-${dd} ${hh}:${mi}`
}

function copyInvite() {
  if (!canShowInvite.value) return
  uni.setClipboardData({ data: ledger.value.inviteCode })
}

async function openInviteCodeQR() {
  const code = String(ledger.value?.inviteCode || '').trim()
  if (!code || !canShowInvite.value) return

  // #ifndef MP-WEIXIN
  uni.showToast({ title: '请在微信小程序内使用', icon: 'none' })
  return
  // #endif

  // #ifdef MP-WEIXIN
  inviteQRModalOpen.value = true
  inviteQRLoading.value = true
  qrModalOpen.value = false

  try {
    await nextTick()
    await drawInviteCodeQR(code)
  } catch (e: any) {
    inviteQRModalOpen.value = false
    const raw = String(e?.message || '')
    const msg = raw || '生成二维码失败'
    uni.showToast({ title: msg, icon: 'none' })
  } finally {
    inviteQRLoading.value = false
  }
  // #endif
}

function closeInviteCodeQR() {
  inviteQRModalOpen.value = false
}

function drawInviteCodeQR(code: string): Promise<void> {
  const instance = getCurrentInstance()
  const proxy = (instance?.proxy as any) || undefined
  const matrix = makeInviteCodeQRMatrix(code)

  const n = matrix.length
  const margin = 4
  const moduleSize = Math.max(1, Math.floor(inviteQRSize / (n + margin * 2)))
  const drawSize = moduleSize * (n + margin * 2)

  const ctx = uni.createCanvasContext('inviteQrCanvas', proxy)
  ctx.setFillStyle('#ffffff')
  ctx.fillRect(0, 0, drawSize, drawSize)
  ctx.setFillStyle('#000000')
  for (let r = 0; r < n; r++) {
    for (let c = 0; c < n; c++) {
      if (!matrix[r][c]) continue
      const x = (c + margin) * moduleSize
      const y = (r + margin) * moduleSize
      ctx.fillRect(x, y, moduleSize, moduleSize)
    }
  }

  return new Promise((resolve) => {
    ctx.draw(false, resolve)
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
    memberAvatar.value = dataUrl
  } catch (err: any) {
    uni.showToast({ title: '头像处理失败', icon: 'none' })
  }
  // #endif
}
</script>

<style scoped>
.page {
  padding: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 24rpx;
  background: #f6f7fb;
  min-height: 100vh;
}
.card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(0, 0, 0, 0.06);
}
.hero {
  background: linear-gradient(135deg, #111 0%, #2b2b2b 100%);
  color: #fff;
  position: relative;
  overflow: hidden;
  border-radius: 20rpx;
}
.hero::before {
  content: '';
  position: absolute;
  right: -120rpx;
  top: -140rpx;
  width: 360rpx;
  height: 360rpx;
  border-radius: 999rpx;
  background: radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.18), rgba(255, 255, 255, 0));
  transform: rotate(12deg);
  pointer-events: none;
}
.hero::after {
  content: '';
  position: absolute;
  left: -140rpx;
  bottom: -180rpx;
  width: 420rpx;
  height: 420rpx;
  border-radius: 999rpx;
  background: radial-gradient(circle at 60% 40%, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0));
  transform: rotate(-10deg);
  pointer-events: none;
}
.row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.hero-row {
  align-items: flex-start;
  gap: 14rpx;
  position: relative;
  z-index: 1;
}
.name {
  font-size: 36rpx;
  font-weight: 700;
  flex: 1;
  min-width: 0;
  line-height: 1.25;
  word-break: break-all;
  white-space: normal;
}
.badge {
  font-size: 24rpx;
  padding: 8rpx 12rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.16);
  color: #fff;
  white-space: nowrap;
  flex: none;
  display: flex;
  align-items: center;
  gap: 8rpx;
  position: relative;
  z-index: 1;
}
.badge.ended {
  background: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.85);
}
.badge-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 999rpx;
  background: rgba(0, 200, 83, 0.95);
  box-shadow: 0 0 0 6rpx rgba(0, 200, 83, 0.18);
  flex: none;
}
.badge-dot.ended {
  background: rgba(255, 255, 255, 0.75);
  box-shadow: 0 0 0 6rpx rgba(255, 255, 255, 0.12);
}
.sub {
  margin-top: 8rpx;
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
  position: relative;
  z-index: 1;
}
.hero-sub {
  margin-top: 16rpx;
}
.pill {
  font-size: 24rpx;
  padding: 8rpx 12rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.14);
  color: rgba(255, 255, 255, 0.92);
  border: 1rpx solid rgba(255, 255, 255, 0.12);
}
.pill.warn {
  background: rgba(255, 255, 255, 0.26);
  border-color: rgba(255, 255, 255, 0.22);
  color: #fff;
}
.pill.code:active {
  opacity: 0.85;
}
.pill.code {
  display: flex;
  align-items: center;
  gap: 8rpx;
}
.qr-icon {
  width: 34rpx;
  height: 34rpx;
  border-radius: 8rpx;
  border: 1rpx solid rgba(255, 255, 255, 0.28);
  background: rgba(255, 255, 255, 0.12);
  position: relative;
  overflow: hidden;
}
.qr-icon:active {
  opacity: 0.85;
}
.qr-finder {
  position: absolute;
  width: 10rpx;
  height: 10rpx;
  border: 2rpx solid rgba(255, 255, 255, 0.92);
  border-radius: 3rpx;
  box-sizing: border-box;
}
.qr-finder-inner {
  position: absolute;
  left: 2rpx;
  top: 2rpx;
  width: 4rpx;
  height: 4rpx;
  border-radius: 2rpx;
  background: rgba(255, 255, 255, 0.92);
}
.qr-finder.tl {
  left: 4rpx;
  top: 4rpx;
}
.qr-finder.tr {
  right: 4rpx;
  top: 4rpx;
}
.qr-finder.bl {
  left: 4rpx;
  bottom: 4rpx;
}
.qr-dot {
  position: absolute;
  width: 4rpx;
  height: 4rpx;
  border-radius: 2rpx;
  background: rgba(255, 255, 255, 0.92);
}
.qr-dot.d1 {
  left: 16rpx;
  top: 8rpx;
}
.qr-dot.d2 {
  left: 22rpx;
  top: 14rpx;
}
.qr-dot.d3 {
  left: 16rpx;
  top: 18rpx;
}
.qr-dot.d4 {
  left: 22rpx;
  top: 22rpx;
}
.qr-dot.d5 {
  left: 12rpx;
  top: 14rpx;
}
.qr-dot.d6 {
  left: 20rpx;
  top: 10rpx;
}
.qr-dot.d7 {
  left: 14rpx;
  top: 24rpx;
}
.invite-qr-canvas {
  margin: 16rpx auto 10rpx;
  background: #fff;
  border-radius: 16rpx;
  box-shadow: 0 10rpx 30rpx rgba(0, 0, 0, 0.08);
}
.qr {
  width: 100%;
  border-radius: 12rpx;
  background: #f6f7fb;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  letter-spacing: 1rpx;
}
.pill.readonly {
  color: #fff;
  background: rgba(255, 255, 255, 0.22);
  border-color: rgba(255, 255, 255, 0.2);
}
.fab-mask {
  position: fixed;
  left: 0;
  right: 0;
  top: 0;
  bottom: 0;
  z-index: 40;
}
.fab {
  position: fixed;
  right: 24rpx;
  bottom: calc(28rpx + env(safe-area-inset-bottom));
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12rpx;
  z-index: 41;
}
.fab-panel {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0;
  padding: 6rpx 0;
  border-radius: 14rpx;
  background: #fff;
  border: 1rpx solid rgba(0, 0, 0, 0.06);
  box-shadow: 0 10rpx 26rpx rgba(0, 0, 0, 0.12);
  transform: translateY(10rpx);
  opacity: 0;
  pointer-events: none;
  transition: all 0.2s ease;
}
.fab-panel.open {
  transform: translateY(0);
  opacity: 1;
  pointer-events: auto;
}
.fab-panel .action-btn {
  width: 200rpx;
  text-align: left;
}
.fab-toggle {
  width: 70rpx;
  height: 70rpx;
  border-radius: 999rpx;
  background: rgba(120, 120, 120, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid rgba(255, 255, 255, 0.18);
  box-shadow: 0 10rpx 24rpx rgba(0, 0, 0, 0.16);
  transition: all 0.2s ease;
}
.fab-toggle::after {
  border: none;
}
.fab-toggle.active {
  background: rgba(120, 120, 120, 0.6);
  border-color: rgba(255, 255, 255, 0.22);
  box-shadow: 0 12rpx 26rpx rgba(0, 0, 0, 0.2);
}
.fab-toggle:active {
  transform: scale(0.98);
}
.fab-icon {
  width: 28rpx;
  height: 28rpx;
}
.action-btn {
  position: relative;
  background: transparent;
  color: #444;
  border-radius: 0;
  height: 64rpx;
  line-height: 64rpx;
  padding: 0 12rpx;
  font-size: 26rpx;
  font-weight: 500;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
}
.fab-panel .action-btn {
  background-image: linear-gradient(#eee, #eee);
  background-repeat: no-repeat;
  background-position: center bottom;
  background-size: 60% 1rpx;
}
.fab-panel .action-btn:last-child {
  background-image: none;
}
.action-btn::after {
  border: none;
}
.action-btn:active {
  opacity: 0.85;
}
.action-btn.danger {
  background: transparent;
  color: #d92d20;
}
.hero .action-btn {
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
}
.hero .action-btn.danger {
  background: rgba(255, 77, 79, 0.2);
  color: #ffd1d1;
}
.title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}
.title-actions {
  display: flex;
  align-items: center;
  gap: 10rpx;
}
.icon-btn {
  width: 60rpx;
  height: 60rpx;
  border-radius: 999rpx;
  background: #f6f7fb;
  color: #111;
  font-size: 30rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}
.icon-btn::after {
  border: none;
}
.icon-btn.primary {
  background: #e6e7ea;
  color: #111;
}
.icon-img {
  width: 28rpx;
  height: 28rpx;
}
.icon-img.small {
  width: 22rpx;
  height: 22rpx;
}
.title {
  font-size: 30rpx;
  font-weight: 600;
}
.tip {
  color: #888;
  font-size: 24rpx;
  margin-top: 6rpx;
}
.hint {
  color: #666;
  font-size: 26rpx;
  margin-top: 8rpx;
}
.member-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16rpx;
}
.member {
  background: #f6f7fb;
  border-radius: 16rpx;
  padding: 16rpx 12rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
  position: relative;
}
.member.disabled {
  opacity: 0.6;
}
.avatar-wrap {
  width: 96rpx;
  height: 96rpx;
  border-radius: 48rpx;
  overflow: hidden;
  background: #fff;
}
.member-tags {
  position: absolute;
  left: 8rpx;
  top: 8rpx;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  z-index: 2;
}
.tag {
  padding: 0 8rpx;
  height: 26rpx;
  line-height: 26rpx;
  border-radius: 8rpx;
  font-size: 18rpx;
  font-weight: 600;
  color: #fff;
  background: rgba(0, 0, 0, 0.7);
}
.tag.owner-tag {
  background: #111;
}
.tag.me-tag {
  background: #5b6bff;
}
.bind-list {
  margin-top: 16rpx;
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}
.bind-item {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 12rpx;
  border-radius: 14rpx;
  background: #f6f7fb;
}
.bind-avatar {
  width: 64rpx;
  height: 64rpx;
  border-radius: 32rpx;
  background: #fff;
  flex: none;
}
.bind-info {
  flex: 1;
  min-width: 0;
}
.bind-name {
  font-size: 28rpx;
  font-weight: 600;
}
.bind-sub {
  font-size: 22rpx;
  color: #666;
  margin-top: 4rpx;
}
.bind-btn {
  background: #111;
  color: #fff;
  border-radius: 999rpx;
  padding: 0 16rpx;
}
.bind-btn::after {
  border: none;
}
.member-edit {
  position: absolute;
  right: 6rpx;
  top: 6rpx;
  width: 44rpx;
  height: 44rpx;
  border-radius: 999rpx;
  background: #e6e7ea;
  color: #111;
  border: 1rpx solid #d6d7db;
  font-size: 22rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}
.member-edit::after {
  border: none;
}
.avatar {
  width: 96rpx;
  height: 96rpx;
  border-radius: 48rpx;
  background: #fff;
}
.member-name {
  font-size: 26rpx;
  font-weight: 600;
  text-align: center;
  line-height: 1.2;
}
.member-total {
  font-size: 24rpx;
}
.member-total.pos {
  color: #19be6b;
}
.member-total.neg {
  color: #e23d3d;
}
.member-total.zero {
  color: #666;
}
.records {
  margin-top: 12rpx;
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.record {
  background: #fff;
  border-radius: 16rpx;
  padding: 16rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.06);
}
.record-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}
.record-user {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
}
.record-avatar {
  width: 44rpx;
  height: 44rpx;
  border-radius: 22rpx;
  background: #fff;
}
.record-name {
  font-size: 26rpx;
  font-weight: 600;
}
.record-meta {
  display: flex;
  align-items: center;
  gap: 8rpx;
  font-size: 24rpx;
  color: #666;
}
.record-type.pos,
.record-amount.pos {
  color: #19be6b;
  font-weight: 600;
}
.record-type.neg,
.record-amount.neg {
  color: #e23d3d;
  font-weight: 600;
}
.record-type.remark {
  color: #9aa0a6;
  font-weight: 600;
}
.record-time {
  color: #999;
}
.record-note {
  margin-top: 8rpx;
  color: #666;
  font-size: 24rpx;
}
.filter-panel {
  margin-top: 14rpx;
}
.filter-title {
  font-size: 24rpx;
  color: #666;
  margin-bottom: 8rpx;
}
.filter-scroll {
  display: flex;
  white-space: nowrap;
}
.filter-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 8rpx 16rpx;
  margin-right: 10rpx;
  border-radius: 999rpx;
  background: #f1f2f4;
  color: #333;
  font-size: 24rpx;
  border: 1rpx solid transparent;
}
.filter-chip.active {
  background: #111;
  color: #fff;
  border-color: #111;
}
.record-btn {
  background: #111;
  color: #fff;
  border-radius: 999rpx;
  height: 60rpx;
  line-height: 60rpx;
  padding: 0 18rpx;
  font-size: 24rpx;
  font-weight: 600;
}
.record-btn::after {
  border: none;
}

.modal-mask {
  position: fixed;
  z-index: 1000;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.45);
}
.modal {
  position: fixed;
  z-index: 1001;
  left: 24rpx;
  right: 24rpx;
  top: 14%;
  background: #fff;
  border-radius: 18rpx;
  padding: 24rpx;
  box-shadow: 0 18rpx 48rpx rgba(0, 0, 0, 0.18);
}
.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}
.modal-title {
  font-size: 30rpx;
  font-weight: 600;
}
.modal-close {
  font-size: 36rpx;
  color: #666;
}
.form {
  margin-top: 16rpx;
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.input {
  background: #f6f7fb;
  border-radius: 12rpx;
  padding: 18rpx 16rpx;
  font-size: 28rpx;
}
.btn {
  margin-top: 8rpx;
}
.avatar-wrapper {
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
.avatar-tip {
  color: #666;
  font-size: 24rpx;
}
.avatar-preview {
  display: flex;
  justify-content: center;
}
.record-target {
  margin-top: 16rpx;
  display: flex;
  align-items: center;
  gap: 12rpx;
}
.record-target-avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 36rpx;
  background: #fff;
}
.record-target-name {
  font-size: 28rpx;
  font-weight: 600;
}
.record-target-sub {
  margin-top: 4rpx;
  color: #666;
  font-size: 24rpx;
}
.type-select {
  margin-top: 16rpx;
  display: flex;
  align-items: center;
  gap: 12rpx;
  background: #f1f2f5;
  border-radius: 999rpx;
  padding: 6rpx;
}
.type-option {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
  flex: 1;
  padding: 10rpx 0;
  border-radius: 999rpx;
  border: 1rpx solid transparent;
  background: transparent;
  color: #444;
}
.type-option.active {
  border-color: #111;
  background: #111;
  color: #fff;
  box-shadow: 0 8rpx 18rpx rgba(0, 0, 0, 0.15);
}
.type-radio {
  width: 18rpx;
  height: 18rpx;
  border-radius: 999rpx;
  border: 2rpx solid #999;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
}
.type-radio::after {
  content: '';
  width: 8rpx;
  height: 8rpx;
  border-radius: 999rpx;
  background: #fff;
  opacity: 0;
}
.type-option.active .type-radio {
  border-color: #fff;
}
.type-option.active .type-radio::after {
  opacity: 1;
}
.type-label {
  font-size: 28rpx;
  font-weight: 600;
}
.modal-actions {
  margin-top: 16rpx;
  display: flex;
}
.confirm-btn {
  background: #111;
  color: #fff;
  border-radius: 14rpx;
  height: 84rpx;
  line-height: 84rpx;
  width: 100%;
  font-size: 28rpx;
  font-weight: 600;
  box-shadow: 0 12rpx 28rpx rgba(0, 0, 0, 0.18);
}
.confirm-btn::after {
  border: none;
}
</style>
