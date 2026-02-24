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
      <div v-if="session.card">
        卡片：{{ session.card.card_type }} (剩余{{ session.card.remain_times }}次)
      </div>
      <div v-if="session.room">
        房间：{{ session.room.name }}
      </div>
      <div v-if="session.technician">
        技师：{{ session.technician.name || session.technician.account }}
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
      <div v-if="session.duration_minutes > 0">
        服务时长：{{ session.duration_minutes }}分钟
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
import { replaceTerms } from '../utils/terms'
import { normalizeSessionStatus } from '../utils/sessionStatus'

const props = defineProps({
  session: {
    type: Object,
    required: true
  }
})

defineEmits(['extend'])

const hasTimeInfo = computed(() => {
  return props.session.started_at || 
         props.session.scheduled_finish_at || 
         props.session.duration_minutes > 0
})

const canExtend = computed(() => {
  return normalizeSessionStatus(props.session.status) === 'serving'
})

const getStatusText = (status) => {
  const s = normalizeSessionStatus(status)
  const statusMap = {
    created: '已创建',
    room_selecting: '选房中',
    room_locked: '房间已锁定',
    staff_selecting: '选人中',
    start_pending: '待起单',
    delay_pending: '待上号',
    timeout_waiting: '过号等待',
    timeout_failed: '已过期',
    serving: '服务中',
    auto_finishing: '待结单',
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
    finished: 'bg-gray-100 text-gray-600',
    canceled: 'bg-gray-100 text-gray-600'
  }
  return statusClassMap[s] || 'bg-gray-100 text-gray-600'
}
</script>
