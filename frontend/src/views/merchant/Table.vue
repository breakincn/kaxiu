<template>
  <div class="min-h-screen bg-gray-50">
    <header v-if="!embedded" class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">看板</span>
      <div class="flex-1"></div>
    </header>

    <div class="px-4 pt-2 pb-4">
      <div class="bg-white rounded-xl shadow-sm px-4 pt-3 pb-4">
        <div v-if="!serviceOnly" class="grid grid-cols-4 gap-1">
          <button
            type="button"
            class="w-full px-2 py-2 rounded-lg text-sm font-medium border whitespace-nowrap"
            :class="activeTab === 'rooms' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectTab('rooms')"
          >
            房间
          </button>
          <button
            type="button"
            class="w-full px-2 py-2 rounded-lg text-sm font-medium border whitespace-nowrap"
            :class="activeTab === 'staff' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectTab('staff')"
          >
            客服
          </button>
          <button
            type="button"
            class="w-full px-2 py-2 rounded-lg text-sm font-medium border whitespace-nowrap"
            :class="activeTab === 'service' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectTab('service')"
          >
            服务
          </button>
          <button
            v-if="hasExceptionContent"
            type="button"
            class="w-full px-2 py-2 rounded-lg text-xs font-medium border whitespace-nowrap leading-tight"
            :class="activeTab === 'exception' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectTab('exception')"
          >
            异常中心
          </button>
        </div>

        <div v-if="activeTab === 'staff'" class="mt-3 flex gap-2">
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-sm font-medium border"
            :class="staffSubTab === 'operation' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectStaffSubTab('operation')"
          >
            运营客服
          </button>
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-sm font-medium border"
            :class="staffSubTab === 'professional' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectStaffSubTab('professional')"
          >
            专业客服
          </button>
        </div>

        <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>

        <div v-else class="mt-2">
          <div v-if="!activeTab" class="text-center text-gray-400 py-10">请选择上方标签查看内容</div>

          <div v-else-if="activeTab === 'rooms'">
            <div v-if="rooms.length === 0" class="text-center text-gray-400 py-10">暂无房间</div>
            <div v-else class="space-y-3">
              <div v-for="it in rooms" :key="it.room.id" class="border border-gray-100 rounded-xl p-4 bg-white">
                <div class="flex items-start justify-between gap-3">
                  <div class="flex-1">
                    <div class="flex items-center gap-2 flex-wrap justify-between">
                      <div class="flex items-center gap-2 flex-wrap">
                        <div class="text-gray-800 font-medium">{{ it.room.name }}</div>
                        <span class="px-2 py-0.5 rounded text-xs" :class="it.occupied ? 'bg-orange-50 text-orange-600' : 'bg-green-50 text-green-600'">
                          {{ it.occupied ? '使用中' : '空闲' }}
                        </span>
                        <span v-if="it.occupied" class="px-2 py-0.5 rounded text-xs bg-blue-50 text-blue-600">
                          {{ sessionStatusText(it.status) }}
                        </span>
                      </div>
                      <div v-if="it.occupied && (it.started_at || it.room_locked_at)" class="text-gray-500 text-sm text-right">
                        <div class="text-[11px] leading-none">占用时长</div>
                        <div class="mt-1 font-mono">{{ calculateElapsedTime(it.started_at || it.room_locked_at) }}</div>
                      </div>
                    </div>
                    <div v-if="it.occupied" class="text-gray-500 text-sm mt-1">
                      <div v-if="it.technician">{{ it.technician_role?.name || '工作人员' }}: {{ it.technician.account }} {{ it.technician.name }}</div>
                      <div v-if="it.started_at">开始：{{ formatTime(it.started_at) }}</div>
                      <div v-if="it.finish_at">结束：{{ formatTime(it.finish_at) }}</div>
                      <div v-if="it.finish_at">剩余：{{ calculateRemainTime(it.finish_at) }}</div>
                      <div v-if="it.phase_text" class="mt-1">
                        <span class="font-medium" :class="it.phase_class === 'start_pending' ? 'text-red-500' : (it.phase_class === 'finish_failed' ? 'text-red-500' : 'text-blue-600')">
                          {{ getPhaseDisplayText(it) }}
                        </span>
                        <span v-if="it.phase_class === 'start_pending'" class="ml-2 text-red-500 font-mono">
                          {{ formatCountdownSeconds(getDynamicStartRemainSeconds(it)) }}
                        </span>
                        <span v-else-if="it.phase_class === 'room_selecting'" class="ml-2 text-blue-600 font-mono">
                          {{ calculateRemainTime(it.room_select_deadline_at) }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeTab === 'staff'">
            <div v-if="groupedStaff.length === 0" class="text-center text-gray-400 py-10">暂无{{ staffSubTab === 'operation' ? '运营客服' : '专业客服' }}</div>
            <div v-else class="space-y-6">
              <div v-for="g in groupedStaff" :key="g.groupKey" class="mb-6">
                <div class="flex items-center justify-between mb-3">
                  <div class="flex items-center gap-2">
                    <div class="text-gray-800 font-medium">{{ g.roleName }}</div>
                    <span class="px-2 py-0.5 bg-blue-50 text-blue-600 rounded text-xs">{{ g.items.length }}人</span>
                  </div>
                </div>

                <div class="space-y-3">
                  <div v-for="it in g.items" :key="it.technician.id" class="border border-gray-100 rounded-xl p-4 bg-white">
                    <div class="flex items-start justify-between gap-3">
                      <div class="min-w-0 flex-1">
                        <div class="flex items-center gap-2">
                          <div class="text-gray-800 font-medium">{{ it.technician.name }}</div>
                          <span v-if="shouldShowStaffBadge(it)" class="px-2 py-0.5 rounded text-xs" :class="badgeClass(it)">
                            {{ badgeText(it) }}
                          </span>
                        </div>
                        <div class="text-gray-500 text-sm mt-1">
                          <div>岗位：{{ it.technician.service_role?.name || '-' }}　账号：{{ it.technician.account }}</div>
                          <div v-if="it.checked_in_at">签到：{{ formatTime(it.checked_in_at) }}</div>
                          <div v-if="it.room">房间：{{ it.room.name }}</div>
                          <div v-if="it.service_start_at">开始：{{ formatTime(it.service_start_at) }}</div>
                          <div v-if="it.service_finish_at">结束：{{ formatTime(it.service_finish_at) }}</div>
                          <div v-if="it.service_finish_at">剩余：{{ calculateRemainTime(it.service_finish_at) }}</div>
                          <div v-if="it.next_available_at">下次可服务：{{ formatTime(it.next_available_at) }}（{{ formatDuration(it.next_available_in_seconds) }}）</div>
                        </div>
                      </div>
                      <div class="shrink-0 self-start rounded-md bg-green-500 px-2.5 py-0.5 text-xs font-medium text-white whitespace-nowrap">
                        今日已完成 {{ Number(it.today_completed_count || 0) }} 单
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeTab === 'service'">
            <div class="pt-0 pb-4">
              <div v-if="!hideServiceFilter" class="bg-white rounded-xl shadow-sm p-4 mb-4">
                <div class="flex items-center gap-3">
                  <label class="text-sm font-medium text-gray-700">状态筛选：</label>
                  <select v-model="statusFilter" @change="resetServiceSessions" class="border border-gray-200 rounded-lg px-3 py-2 text-sm">
                    <option value="">全部</option>
                    <option value="created">已创建</option>
                    <option value="room_selecting">选房中</option>
                    <option value="room_locked">房间已锁定</option>
                    <option value="staff_selecting">选人中</option>
                    <option value="start_pending">{{ getPendingStartLabel(merchant, { queueMode: isQueueModeMerchant(merchant) }) }}</option>
                    <option value="delay_pending">延迟中</option>
                    <option value="serving">进行中</option>
                    <option value="auto_finishing">{{ getAutoFinishLabel(merchant) }}</option>
                    <option value="finished">已完成</option>
                    <option value="canceled">已取消</option>
                  </select>
                </div>
              </div>

              <div class="space-y-3">
                <div v-if="!technicianOwnOnly && isTechnicianAuth() && myServingSessions.length > 0">
                  <div class="bg-white rounded-xl shadow-sm p-4">
                    <div class="font-medium text-gray-800 mb-3 flex items-center gap-2">
                      <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                      </svg>
                      我的服务中
                    </div>
                    <div class="space-y-2">
                      <div v-for="session in myServingSessions" :key="session.id" class="border border-blue-100 rounded-lg p-3 bg-blue-50">
                        <ServiceSessionItem :session="session" :currentTime="currentTimeMs" @extend="openExtendModal" />
                      </div>
                    </div>
                  </div>
                </div>

                <div v-if="filteredOtherSessions.length === 0" class="text-center text-gray-400 py-10">
                  {{ statusFilter ? '暂无符合条件的服务单' : '暂无服务单' }}
                </div>

                <div v-else class="space-y-4">
                  <div v-for="group in groupedOtherSessions" :key="group.date" class="space-y-3">
                    <div class="font-medium text-gray-800">
                      {{ group.title }}
                      <span class="text-gray-500 text-sm font-normal">({{ group.items.length }})</span>
                    </div>

                    <div
                      v-for="session in group.items"
                      :key="session.id"
                      :class="serviceOnly ? 'rounded-2xl border border-gray-100 bg-white px-4 py-4 shadow-sm' : 'border border-gray-100 rounded-lg p-3'"
                    >
                      <template v-if="serviceOnly && technicianOwnOnly">
                        <div class="flex items-start justify-between gap-3">
                          <div class="flex items-center gap-2 flex-wrap">
                            <div class="text-[15px] font-semibold text-gray-800">服务单 #{{ session.id }}</div>
                            <span class="px-2 py-0.5 rounded-md text-xs font-medium" :class="getServiceBoardStatusClass(session.status)">
                              {{ getServiceBoardStatusText(session.status) }}
                            </span>
                          </div>
                          <div class="shrink-0 text-xs text-gray-500">{{ formatBoardDateTime(session.created_at) }}</div>
                        </div>
                        <div class="mt-3 space-y-1 text-sm leading-6 text-gray-600">
                          <div v-if="getServiceBoardCardText(session)">卡片：{{ getServiceBoardCardText(session) }}</div>
                          <div v-if="getServiceBoardUsageText(session)">卡号：{{ getServiceBoardUsageText(session) }}</div>
                          <div v-if="getServiceBoardCancelReason(session)">取消原因：{{ getServiceBoardCancelReason(session) }}</div>
                        </div>
                      </template>
                      <ServiceSessionItem v-else :session="session" :currentTime="currentTimeMs" @extend="openExtendModal" />
                    </div>
                  </div>
                </div>

                <button
                  v-if="canLoadMoreServiceSessions"
                  type="button"
                  class="w-full rounded-lg border border-orange-200 bg-orange-50 px-4 py-2 text-sm font-medium text-orange-600 disabled:opacity-50"
                  :disabled="serviceSessionsLoadingMore"
                  @click="loadMoreServiceSessions"
                >
                  {{ serviceSessionsLoadingMore ? '加载中...' : '更多' }}
                </button>
              </div>
            </div>
          </div>

          <div v-else-if="activeTab === 'exception' && hasExceptionContent">
            <slot name="exception-content" />
          </div>
        </div>
      </div>
    </div>

    <div v-if="showExtendModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
      <div class="bg-white w-full max-w-sm rounded-2xl p-4">
        <div class="flex items-center justify-between mb-3">
          <div class="font-medium text-gray-800">加钟</div>
          <button class="text-gray-500" @click="closeExtendModal">关闭</button>
        </div>

        <div class="text-gray-600 text-sm mb-3">延长服务时间（5~180分钟）</div>
        <div class="mb-4">
          <input v-model.number="extendMinutes" type="number" min="5" max="180" placeholder="分钟" class="w-full px-4 py-3 border border-gray-200 rounded-lg">
        </div>
        <div class="flex gap-2">
          <button @click="closeExtendModal" class="flex-1 px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium">取消</button>
          <button
            :disabled="!extendMinutes || extendMinutes < 5 || extendMinutes > 180 || extendLoading"
            @click="doExtendSession"
            class="flex-1 px-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ extendLoading ? '加钟中...' : '确认' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, useSlots } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi, serviceSessionApi } from '../../api'
import {
  getAutoFinishLabel,
  getPendingStartLabel,
  getServicePendingFinishLabel,
  getServicePendingStartLabel,
  isQueueModeMerchant
} from '../../utils/terms'
import { getMerchantId, isTechnicianAuth, getTechnicianId } from '../../utils/auth'
import { normalizeSessionStatus } from '../../utils/sessionStatus'
import ServiceSessionItem from '../../components/ServiceSessionItem.vue'

const props = defineProps({
  embedded: {
    type: Boolean,
    default: false
  },
  serviceOnly: {
    type: Boolean,
    default: false
  },
  technicianOwnOnly: {
    type: Boolean,
    default: false
  },
  hideServiceFilter: {
    type: Boolean,
    default: false
  }
})

const router = useRouter()
const slots = useSlots()
const hasExceptionContent = computed(() => !!slots['exception-content'])

const activeTab = ref(props.serviceOnly ? 'service' : (localStorage.getItem('tableActiveTab') || ''))
const staffSubTab = ref(localStorage.getItem('tableStaffSubTab') || 'professional')
const loading = ref(false)
const rooms = ref([])
const staff = ref([])
const currentTime = ref(new Date())
const currentTimeMs = computed(() => currentTime.value?.getTime?.() || Date.now())
const merchant = ref({})
const roleAttendanceMap = ref({})

const statusFilter = ref('')
const serviceSessions = ref([])
const loadedServiceSessionDates = ref([])
const serviceSessionCursorDate = ref('')
const serviceSessionsLoadingMore = ref(false)
const extendSession = ref(null)
const extendMinutes = ref(null)
const showExtendModal = ref(false)
const extendLoading = ref(false)

const canLoadMoreServiceSessions = computed(() => activeTab.value === 'service' && loadedServiceSessionDates.value.length > 0)

const isSessionOwnedByCurrentTechnician = (session) => {
  const techId = getTechnicianId()
  if (!techId) return false
  const candidateIds = [
    session?.technician_id,
    session?.last_technician_id,
    session?.technician?.id,
    session?.last_technician?.id,
    session?.initial_usage?.technician_id,
    session?.initial_usage?.technician?.id
  ]
  return candidateIds.some((value) => Number(value || 0) === techId)
}

const myServingSessions = computed(() => {
  if (!isTechnicianAuth()) return []
  const techId = getTechnicianId()
  if (!techId) return []
  return serviceSessions.value.filter((session) =>
    Number(session?.technician_id || 0) === techId &&
    ['delay_pending', 'serving', 'auto_finishing'].includes(normalizeSessionStatus(session.status))
  )
})

const filteredOtherSessions = computed(() => {
  let sessions = serviceSessions.value

  if (props.technicianOwnOnly && isTechnicianAuth()) {
    sessions = sessions.filter(isSessionOwnedByCurrentTechnician)
  } else if (isTechnicianAuth()) {
    const myIds = new Set(myServingSessions.value.map((session) => session.id))
    sessions = sessions.filter((session) => !myIds.has(session.id))
  }

  if (statusFilter.value) {
    sessions = sessions.filter((session) => normalizeSessionStatus(session.status) === statusFilter.value)
  }

  return sessions
})

const groupedOtherSessions = computed(() => {
  const groups = []
  const groupMap = new Map()

  for (const session of filteredOtherSessions.value) {
    const dateKey = getSessionDateKey(session)
    if (!groupMap.has(dateKey)) {
      const group = {
        date: dateKey,
        title: `${formatDisplayServiceSessionDate(dateKey)}服务单`,
        items: []
      }
      groupMap.set(dateKey, group)
      groups.push(group)
    }
    groupMap.get(dateKey).items.push(session)
  }

  return groups
})

const groupedStaff = computed(() => {
  const groups = []
  const groupIndex = {}

  for (const item of staff.value || []) {
    const roleKey = String(item?.technician?.service_role?.key || '').trim()
    const roleName = String(item?.technician?.service_role?.name || '').trim() || '未知岗位'
    const groupKey = roleKey || roleName

    let idx = groupIndex[groupKey]
    if (idx === undefined) {
      idx = groups.length
      groupIndex[groupKey] = idx
      groups.push({ groupKey, roleKey, roleName, items: [] })
    }
    groups[idx].items.push(item)
  }

  return groups
})

let timer = null

const updateCurrentTime = () => {
  currentTime.value = new Date()
}

const goBack = () => {
  router.back()
}

const formatBoardDateTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const yy = String(date.getFullYear()).slice(-2)
  const mm = String(date.getMonth() + 1).padStart(2, '0')
  const dd = String(date.getDate()).padStart(2, '0')
  const hh = String(date.getHours()).padStart(2, '0')
  const min = String(date.getMinutes()).padStart(2, '0')
  return `${yy}-${mm}-${dd} ${hh}:${min}`
}

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const pad = (num) => String(num).padStart(2, '0')
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const formatDuration = (seconds) => {
  const totalSeconds = Math.max(0, Number(seconds || 0))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const secs = Math.floor(totalSeconds % 60)
  const pad2 = (num) => String(num).padStart(2, '0')
  if (hours > 0) return `${hours}小时${pad2(minutes)}分${pad2(secs)}秒`
  return `${minutes}分${pad2(secs)}秒`
}

const formatCountdownSeconds = (seconds) => {
  const remain = Math.max(0, Math.floor(Number(seconds || 0)))
  const minutes = Math.floor(remain / 60)
  const secs = remain % 60
  return `${minutes}分${String(secs).padStart(2, '0')}秒`
}

const getDynamicStartRemainSeconds = (item) => {
  const baseRemain = Math.max(0, Math.floor(Number(item?.start_remain_seconds || 0)))
  if (baseRemain <= 0) return 0

  const snapshotMs = item?.now ? new Date(item.now).getTime() : 0
  if (!Number.isFinite(snapshotMs) || snapshotMs <= 0) return baseRemain

  const elapsedSeconds = Math.max(0, Math.floor((currentTimeMs.value - snapshotMs) / 1000))
  return Math.max(0, baseRemain - elapsedSeconds)
}

const getPhaseDisplayText = (item) => {
  if (!item) return ''
  if (item.phase_class === 'start_pending') return `${item.phase_text}剩余`
  return item.phase_text || ''
}

const calculateRemainTime = (endTime, durationSeconds = null) => {
  if (!endTime) return '-'
  let deadline
  if (durationSeconds !== null) {
    const start = new Date(endTime)
    deadline = new Date(start.getTime() + durationSeconds * 1000)
  } else {
    deadline = new Date(endTime)
  }

  const diffSeconds = Math.floor((deadline - currentTime.value) / 1000)
  const pad2 = (num) => String(num).padStart(2, '0')
  if (diffSeconds <= 0) return `0分${pad2(0)}秒`

  const hours = Math.floor(diffSeconds / 3600)
  const minutes = Math.floor((diffSeconds % 3600) / 60)
  const secs = diffSeconds % 60
  if (hours > 0) return `${hours}小时${pad2(minutes)}分${pad2(secs)}秒`
  return `${minutes}分${pad2(secs)}秒`
}

const calculateElapsedTime = (startTime) => {
  if (!startTime) return ''
  const start = new Date(startTime)
  const diffSeconds = Math.floor((currentTime.value - start) / 1000)
  const hours = Math.floor(diffSeconds / 3600)
  const minutes = Math.floor((diffSeconds % 3600) / 60)
  const secs = Math.floor(diffSeconds % 60)
  if (hours === 0) return `${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
  if (hours < 10) return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
}

const sessionStatusText = (status) => {
  const normalized = normalizeSessionStatus(status)
  if (normalized === 'created') return '已创建'
  if (normalized === 'room_selecting') return '选房中'
  if (normalized === 'room_locked') return '房间已锁定'
  if (normalized === 'staff_selecting') return '选人中'
  if (normalized === 'start_pending') return getPendingStartLabel(merchant.value, { queueMode: isQueueModeMerchant(merchant.value) })
  if (normalized === 'delay_pending') return getPendingStartLabel(merchant.value, { queueMode: isQueueModeMerchant(merchant.value) })
  if (normalized === 'timeout_waiting') return '过号等待'
  if (normalized === 'timeout_failed') return '过号失败'
  if (normalized === 'serving') return '服务中'
  if (normalized === 'auto_finishing') return getAutoFinishLabel(merchant.value)
  if (normalized === 'finished') return '已完成'
  if (normalized === 'canceled') return '已取消'
  return normalized || '-'
}

const getServiceBoardStatusText = (status) => sessionStatusText(status)

const getServiceBoardStatusClass = (status) => {
  const normalized = normalizeSessionStatus(status)
  if (normalized === 'canceled') return 'bg-red-50 text-red-500'
  if (normalized === 'finished' || normalized === 'serving') return 'bg-green-50 text-green-600'
  if (normalized === 'auto_finishing') return 'bg-orange-50 text-orange-600'
  if (normalized === 'start_pending' || normalized === 'delay_pending') return 'bg-blue-50 text-blue-600'
  return 'bg-gray-100 text-gray-500'
}

const getServiceBoardCardText = (session) => {
  const cardType = String(session?.card?.card_type || '').trim()
  const totalTimes = Number(session?.card?.total_times || 0)
  const remainTimes = Number(session?.card?.remain_times || 0)
  if (!cardType && totalTimes <= 0) return ''
  if (totalTimes > 0) return `${cardType || '-'}（剩余${remainTimes}次）`
  return cardType
}

const getServiceBoardUsageText = (session) => {
  const cardNo = String(session?.card?.card_no || '').trim()
  const totalTimes = Number(session?.card?.total_times || 0)
  const remainTimes = Number(session?.card?.remain_times || 0)
  const usedTimes = totalTimes > 0 ? Math.max(0, totalTimes - remainTimes) : 0
  if (!cardNo && totalTimes <= 0 && usedTimes <= 0) return ''
  return `${cardNo || '-'} 核销：${totalTimes > 0 ? totalTimes : '-'}\/${usedTimes > 0 ? usedTimes : '-'}`
}

const getServiceBoardCancelReason = (session) => {
  if (normalizeSessionStatus(session?.status) !== 'canceled') return ''
  const appointment = session?.appointment
  const appointmentStatus = String(appointment?.status || '').trim()
  const merchantCancelReason = String(appointment?.merchant_cancel_reason || '').trim()
  const disruptionReason = String(appointment?.disruption_reason || '').trim()
  const failedReason = String(appointment?.failed_reason || '').trim()
  const reasonCode = failedReason || disruptionReason

  const reasonTextMap = {
    service_unclosed_cross_day: '客户已到店但未开始服务，且跨日未完成结案',
    user_no_show: '预约用户未到店',
    appointment_state_inconsistent: '预约状态异常，已按失约处理',
    merchant_timeout_failed: '上钟超时'
  }

  if (appointmentStatus === 'no_show' || disruptionReason === 'user_no_show') return '预约用户未到店'
  if (merchantCancelReason) return merchantCancelReason
  if (reasonCode && reasonTextMap[reasonCode]) return reasonTextMap[reasonCode]
  if (failedReason) return failedReason
  if (Number(session?.start_timeout_count || 0) > 0) return '上钟超时'
  return '已取消'
}

const badgeText = (item) => {
  const status = String(item.service_status || '')
  if (!item.checked_in) return '未签到'

  const session = item.current_session
  if (session) {
    const sessionStatus = normalizeSessionStatus(session.status)
    if (sessionStatus === 'start_pending' && !session.start_confirmed_at) {
      return getServicePendingStartLabel(merchant.value, { queueMode: isQueueModeMerchant(merchant.value) })
    }
    if (sessionStatus === 'auto_finishing') return getAutoFinishLabel(merchant.value)
    return getServicePendingFinishLabel(merchant.value)
  }

  if (status === 'paused') return '暂停'
  if (status === 'idle') return '空闲'
  return status || '暂停'
}

const getRoleRequireAttendance = (roleKey) => {
  const key = String(roleKey || '')
  if (!key) return true
  const value = roleAttendanceMap.value[key]
  if (typeof value === 'boolean') return value
  return true
}

const shouldShowStaffBadge = (item) => {
  const requireAttendance = getRoleRequireAttendance(item?.technician?.service_role?.key)
  if (requireAttendance) return true
  return !!item?.current_session
}

const badgeClass = (item) => {
  const text = badgeText(item)
  if (text === '未签到') return 'bg-gray-100 text-gray-500'
  if (text === '空闲') return 'bg-green-50 text-green-600'
  if (text === getServicePendingStartLabel(merchant.value)) return 'bg-red-50 text-red-600'
  if (text === getServicePendingStartLabel(merchant.value, { queueMode: true })) return 'bg-red-50 text-red-600'
  if (text === getAutoFinishLabel(merchant.value)) return 'bg-red-50 text-red-600'
  if (text === getServicePendingFinishLabel(merchant.value)) return 'bg-orange-50 text-orange-600'
  if (text === '暂停') return 'bg-blue-50 text-blue-600'
  return 'bg-blue-50 text-blue-600'
}

const fetchMerchant = async () => {
  try {
    const merchantId = getMerchantId()
    if (!merchantId) return
    const res = await merchantApi.getMerchant(merchantId)
    merchant.value = res.data?.data || {}
  } catch (e) {
    console.error('加载商户信息失败:', e)
  }
}

const loadRoleAttendanceConfigs = async () => {
  try {
    const res = await merchantApi.getRoleAttendanceConfigs()
    const map = {}
    for (const item of res.data?.data || []) {
      if (!item || !item.service_role_key) continue
      map[String(item.service_role_key)] = !!item.require_attendance
    }
    roleAttendanceMap.value = map
  } catch (e) {
    roleAttendanceMap.value = {}
  }
}

const selectTab = async (tab) => {
  activeTab.value = tab
  if (!props.serviceOnly) {
    localStorage.setItem('tableActiveTab', tab)
  }
  await load()
}

const selectStaffSubTab = (tab) => {
  staffSubTab.value = tab
  localStorage.setItem('tableStaffSubTab', tab)
  if (activeTab.value === 'staff') {
    load()
  }
}

const load = async () => {
  if (activeTab.value === 'exception') return
  loading.value = true
  try {
    if (activeTab.value === 'rooms') {
      const res = await merchantApi.getTableRooms()
      rooms.value = res.data?.data || []
      return
    }
    if (activeTab.value === 'staff') {
      const type = staffSubTab.value === 'operation' ? 'operation' : 'professional'
      const res = await merchantApi.getTableStaff(type)
      staff.value = res.data?.data || []
      return
    }
    if (activeTab.value === 'service') {
      await resetServiceSessions()
    }
  } catch (e) {
    if (activeTab.value === 'rooms') rooms.value = []
    else if (activeTab.value === 'staff') staff.value = []
    else if (activeTab.value === 'service') serviceSessions.value = []
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const formatServiceSessionDate = (date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const getTodayServiceSessionDate = () => formatServiceSessionDate(new Date())

const getPreviousServiceSessionDate = (dateText) => {
  const [year, month, day] = String(dateText || '').split('-').map(Number)
  if (!year || !month || !day) return getTodayServiceSessionDate()
  const date = new Date(year, month - 1, day)
  date.setDate(date.getDate() - 1)
  return formatServiceSessionDate(date)
}

const formatDisplayServiceSessionDate = (dateText) => {
  const [year, month, day] = String(dateText || '').split('-').map(Number)
  if (!year || !month || !day) return '未知日期'
  return `${month}月${day}日`
}

const getSessionDateKey = (session) => {
  const raw = session?.created_at || session?.updated_at
  if (!raw) return '未知日期'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return '未知日期'
  return formatServiceSessionDate(date)
}

const mergeServiceSessions = (sessions, append = false) => {
  const merged = append ? [...serviceSessions.value] : []
  const seen = new Set(merged.map((session) => session.id))
  for (const session of sessions) {
    if (seen.has(session.id)) continue
    merged.push(session)
    seen.add(session.id)
  }
  merged.sort((a, b) => b.id - a.id)
  serviceSessions.value = merged
}

const fetchServiceSessionsByDate = async (dateText, append = false) => {
  try {
    const params = { date: dateText }
    if (statusFilter.value) params.status = statusFilter.value
    if (props.technicianOwnOnly && isTechnicianAuth()) {
      params.self_only = 1
    }
    const res = await serviceSessionApi.listSessions(params)
    const items = res.data?.data || []
    mergeServiceSessions(items, append)
    if (items.length > 0 && !append) {
      loadedServiceSessionDates.value = [dateText]
      serviceSessionCursorDate.value = dateText
    } else if (items.length > 0 && !loadedServiceSessionDates.value.includes(dateText)) {
      loadedServiceSessionDates.value = [...loadedServiceSessionDates.value, dateText]
      serviceSessionCursorDate.value = dateText
    }
    return items
  } catch (e) {
    console.error('获取服务单列表失败:', e)
    if (!append) {
      serviceSessions.value = []
      loadedServiceSessionDates.value = []
      serviceSessionCursorDate.value = ''
    }
    return []
  }
}

const findPreviousNonEmptyServiceSessionDate = async (startDateText, append = false) => {
  let dateText = startDateText
  for (let i = 0; i < 365; i += 1) {
    const items = await fetchServiceSessionsByDate(dateText, append)
    if (items.length > 0) return { dateText, items }
    dateText = getPreviousServiceSessionDate(dateText)
  }
  return { dateText: '', items: [] }
}

const resetServiceSessions = async () => {
  serviceSessions.value = []
  loadedServiceSessionDates.value = []
  serviceSessionCursorDate.value = ''
  await findPreviousNonEmptyServiceSessionDate(getTodayServiceSessionDate(), false)
}

const loadMoreServiceSessions = async () => {
  if (serviceSessionsLoadingMore.value || !serviceSessionCursorDate.value) return
  serviceSessionsLoadingMore.value = true
  try {
    await findPreviousNonEmptyServiceSessionDate(getPreviousServiceSessionDate(serviceSessionCursorDate.value), true)
  } finally {
    serviceSessionsLoadingMore.value = false
  }
}

const openExtendModal = (session) => {
  extendSession.value = session
  extendMinutes.value = null
  showExtendModal.value = true
}

const closeExtendModal = () => {
  showExtendModal.value = false
  extendSession.value = null
  extendMinutes.value = null
}

const doExtendSession = async () => {
  if (!extendMinutes.value || extendMinutes.value < 5 || extendMinutes.value > 180) {
    alert('请输入5~180分钟的加钟时长')
    return
  }

  extendLoading.value = true
  try {
    const res = await serviceSessionApi.extendDuration(extendSession.value.id, { minutes: extendMinutes.value })
    const updated = res.data?.data
    if (updated) {
      const idx = serviceSessions.value.findIndex((session) => session.id === updated.id)
      if (idx !== -1) serviceSessions.value[idx] = updated
    }
    closeExtendModal()
  } catch (e) {
    alert(e.response?.data?.error || '加钟失败')
  } finally {
    extendLoading.value = false
  }
}

onMounted(async () => {
  if (props.serviceOnly) {
    activeTab.value = 'service'
  } else {
    if (activeTab.value === 'exception' && !hasExceptionContent.value) {
      activeTab.value = 'rooms'
      localStorage.setItem('tableActiveTab', 'rooms')
    }
    if (!activeTab.value) {
      activeTab.value = 'rooms'
      localStorage.setItem('tableActiveTab', 'rooms')
    }
  }

  await fetchMerchant()
  await loadRoleAttendanceConfigs()
  await load()
  timer = setInterval(updateCurrentTime, 1000)
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
})
</script>
