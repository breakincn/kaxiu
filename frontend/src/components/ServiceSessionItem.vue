<template>
  <div class="space-y-2">
    <!-- 基本信息 -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="font-medium text-gray-800">服务单 #{{ session.id }}</span>
        <span class="px-2 py-0.5 rounded text-xs" :class="getStatusClass(session.status)">
          {{ getStatusText(session.status) }}
        </span>
      </div>
      <div class="text-gray-500 text-xs">
        {{ formatDateTime(session.created_at) }}
      </div>
    </div>

    <!-- 服务信息 -->
    <div class="text-sm text-gray-600 space-y-1">
      <div v-if="session.user">
        用户：{{ session.user.nickname || session.user.phone || `ID:${session.user.id}` }}
      </div>
      <div v-if="trackingNumber">
        单号：{{ trackingNumber }}
      </div>
      <div v-if="session.card">
        卡片：{{ session.card.card_type }} (剩余{{ session.card.remain_times }}次)
      </div>
      <div v-if="cardUsageDisplayText">
        {{ cardUsageDisplayText }}
      </div>
      <div v-if="shouldShowRoom">
        房间：{{ session.room.name }}
      </div>
      <div v-if="isAppointmentSession">
        预约号：#{{ session.source_id }}
      </div>
      <div v-if="effectiveTechnicianDisplayText">
        技师：{{ effectiveTechnicianDisplayText }}
      </div>
      <div v-if="cancelReasonText">
        取消原因：{{ cancelReasonText }}
      </div>
    </div>

    <!-- 时间信息 -->
    <div v-if="hasTimeInfo" class="text-xs text-gray-500 space-y-1">
      <div v-if="session.started_at">
        开始时间：{{ formatDateTime(session.started_at) }}
      </div>
      <div v-if="session.scheduled_finish_at">
        预计结束：{{ formatDateTime(session.scheduled_finish_at) }}
      </div>
      <div v-if="normalizeSessionStatus(session.status) !== 'canceled' && session.duration_minutes > 0">
        服务时长：{{ session.duration_minutes }}分钟
      </div>
      <div v-if="remainingSeconds !== null" class="text-blue-600 font-medium">
        服务剩余：{{ formatRemainingSeconds(remainingSeconds) }}
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="flex gap-2 pt-2">
      <button 
        v-if="canExtend" 
        @click="$emit('extend', session)"
        class="px-3 py-1 bg-blue-500 text-white rounded text-xs font-medium"
      >
        加钟
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { formatDateTime } from '../utils/dateFormat'
import { getAutoFinishLabel, getPendingStartLabel, replaceTerms } from '../utils/terms'
import { normalizeSessionStatus } from '../utils/sessionStatus'

const props = defineProps({
  session: {
    type: Object,
    required: true
  },
  currentTime: {
    type: Number,
    default: () => Date.now()
  }
})

defineEmits(['extend'])

const hasTimeInfo = computed(() => {
  return props.session.started_at || 
         props.session.scheduled_finish_at || 
         (normalizeSessionStatus(props.session.status) !== 'canceled' && props.session.duration_minutes > 0)
})

const isAppointmentSession = computed(() => {
  return String(props.session?.source_type || '').trim() === 'appointment' && Number(props.session?.source_id || 0) > 0
})

const effectiveTechnician = computed(() => {
  return props.session?.technician || props.session?.last_technician || props.session?.initial_usage?.technician || null
})

const effectiveTechnicianDisplayText = computed(() => {
  const account = String(effectiveTechnician.value?.account || '').trim()
  const name = String(effectiveTechnician.value?.name || '').trim()
  if (account && name) return `${account} - ${name}`
  return name || account || ''
})

const trackingNumber = computed(() => {
  const usageId = Number(props.session?.initial_usage_id || 0)
  if (usageId > 0) {
    return String(usageId).padStart(9, '0')
  }
  return String(props.session?.id || '')
})

const shouldShowRoom = computed(() => {
  return normalizeSessionStatus(props.session?.status) !== 'finished' && !!props.session?.room
})

const cardUsageDisplayText = computed(() => {
  const cardNo = String(props.session?.card?.card_no || '').trim()
  const totalTimes = Number(props.session?.card?.total_times || 0)
  const remainTimes = Number(props.session?.card?.remain_times || 0)
  const usedTimes = totalTimes > 0 ? Math.max(0, totalTimes - remainTimes) : 0
  if (!cardNo && totalTimes <= 0 && usedTimes <= 0) return ''
  const cardPart = `卡号：${cardNo || '-'}`
  const verifyPart = `核销：${totalTimes > 0 ? totalTimes : '-'}\/${usedTimes > 0 ? usedTimes : '-'}`
  return `${cardPart} ${verifyPart}`
})

const cancelReasonText = computed(() => {
  if (normalizeSessionStatus(props.session?.status) !== 'canceled') return ''

  const appointment = props.session?.appointment
  const appointmentStatus = String(appointment?.status || '').trim()
  const merchantCancelReason = String(appointment?.merchant_cancel_reason || '').trim()
  const disruptionReason = String(appointment?.disruption_reason || '').trim()
  const failedReason = String(appointment?.failed_reason || '').trim()
  const reasonCode = failedReason || disruptionReason

  const reasonTextMap = {
    service_unclosed_cross_day: '客户已到店但未开始服务，且跨日未完成结案',
    user_no_show: '预约用户未到店',
    appointment_state_inconsistent: '预约状态异常，已按失约处理',
    merchant_timeout_failed: replaceTerms('起单超时', props.session?.merchant)
  }

  if (appointmentStatus === 'no_show' || disruptionReason === 'user_no_show') {
    return '预约用户未到店'
  }
  if (merchantCancelReason) return merchantCancelReason
  if (reasonCode && reasonTextMap[reasonCode]) return reasonTextMap[reasonCode]
  if (failedReason) return failedReason
  if (Number(props.session?.start_timeout_count || 0) > 0) {
    return replaceTerms('起单超时', props.session?.merchant)
  }
  return '已取消'
})

const canExtend = computed(() => {
  return normalizeSessionStatus(props.session.status) === 'serving'
})

const remainingSeconds = computed(() => {
  const s = normalizeSessionStatus(props.session.status)
  if (s !== 'serving' && s !== 'auto_finishing') return null

  let finishAt = 0
  const finishAtRaw = props.session.scheduled_finish_at
  if (finishAtRaw) {
    finishAt = new Date(finishAtRaw).getTime()
  }
  if (!finishAt || Number.isNaN(finishAt)) {
    const startedAtRaw = props.session.started_at
    const durationMinutes = Number(props.session.duration_minutes || 0)
    if (!startedAtRaw || !Number.isFinite(durationMinutes) || durationMinutes <= 0) return null
    const startedAt = new Date(startedAtRaw).getTime()
    if (!startedAt || Number.isNaN(startedAt)) return null
    finishAt = startedAt + durationMinutes * 60 * 1000
  }

  const remain = Math.floor((finishAt - props.currentTime) / 1000)
  if (!Number.isFinite(remain)) return null
  return Math.max(0, remain)
})

const formatRemainingSeconds = (seconds) => {
  const n = Number(seconds)
  if (!Number.isFinite(n) || n < 0) return ''
  const totalSeconds = Math.floor(n)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const secs = totalSeconds % 60
  if (hours > 0) return `${hours}小时${minutes}分${secs}秒`
  if (minutes > 0) return `${minutes}分${secs}秒`
  return `${secs}秒`
}

const getStatusText = (status) => {
  const s = normalizeSessionStatus(status)
  const statusMap = {
    created: '已创建',
    room_selecting: '选房中',
    room_locked: '房间已锁定',
    staff_selecting: '选人中',
    start_pending: getPendingStartLabel(props.session?.merchant),
    delay_pending: getPendingStartLabel(props.session?.merchant),
    timeout_waiting: '过号等待',
    timeout_failed: '已过期',
    serving: '服务中',
    auto_finishing: getAutoFinishLabel(props.session?.merchant),
    finished: '已完成',
    canceled: '已取消'
  }
  return statusMap[s] || s
}

const getStatusClass = (status) => {
  const s = normalizeSessionStatus(status)
  const statusClassMap = {
    created: 'bg-gray-100 text-gray-600',
    room_selecting: 'bg-yellow-100 text-yellow-600',
    room_locked: 'bg-yellow-100 text-yellow-600',
    staff_selecting: 'bg-yellow-100 text-yellow-600',
    start_pending: 'bg-red-100 text-red-600',
    delay_pending: 'bg-blue-100 text-blue-600',
    timeout_waiting: 'bg-blue-100 text-blue-600',
    timeout_failed: 'bg-red-100 text-red-600',
    serving: 'bg-green-100 text-green-600',
    auto_finishing: 'bg-purple-100 text-purple-600',
    finished: 'bg-green-100 text-green-600',
    canceled: 'bg-red-100 text-red-600'
  }
  return statusClassMap[s] || 'bg-gray-100 text-gray-600'
}
</script>
