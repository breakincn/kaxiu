<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">看板</span>
      <div class="flex-1"></div>
    </header>

    <div class="px-4 py-4">
      <div class="bg-white rounded-xl shadow-sm p-4">
        <div class="flex gap-2">
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-sm font-medium border"
            :class="activeTab === 'rooms' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectTab('rooms')"
          >
            房间
          </button>
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-sm font-medium border"
            :class="activeTab === 'staff' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectTab('staff')"
          >
            客服
          </button>
        </div>

        <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>

        <div v-else class="mt-4">
          <div v-if="activeTab === 'rooms'">
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
                      <div v-if="it.occupied && (it.started_at || it.room_locked_at)" class="text-gray-500 text-sm font-mono">
                        {{ calculateElapsedTime(it.started_at || it.room_locked_at) }}
                      </div>
                    </div>
                    <div v-if="it.occupied" class="text-gray-500 text-sm mt-1">
                      <div v-if="it.technician">{{ it.technician_role?.name || '工作人员' }}: {{ it.technician.account }} {{ it.technician.name }}</div>
                      <div v-if="it.started_at">开始：{{ formatTime(it.started_at) }}</div>
                      <div v-if="it.finish_at">结束：{{ formatTime(it.finish_at) }}</div>
                      <div v-if="it.finish_at">剩余：{{ formatDuration(it.remain_seconds) }}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-else>
            <div v-if="staff.length === 0" class="text-center text-gray-400 py-10">暂无专业客服</div>
            <div v-else class="space-y-3">
              <div v-for="it in staff" :key="it.technician.id" class="border border-gray-100 rounded-xl p-4">
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <div class="flex items-center gap-2">
                      <div class="text-gray-800 font-medium">{{ it.technician.name }}</div>
                      <span class="px-2 py-0.5 rounded text-xs" :class="badgeClass(it)">
                        {{ badgeText(it) }}
                      </span>
                    </div>
                    <div class="text-gray-500 text-sm mt-1">
                      <div>岗位：{{ it.technician.service_role?.name || '-' }}　账号：{{ it.technician.account }}</div>
                      <div v-if="it.checked_in_at">签到：{{ formatTime(it.checked_in_at) }}</div>
                      <div v-if="it.room">房间：{{ it.room.name }}</div>
                      <div v-if="it.service_start_at">开始：{{ formatTime(it.service_start_at) }}</div>
                      <div v-if="it.service_finish_at">结束：{{ formatTime(it.service_finish_at) }}</div>
                      <div v-if="it.service_finish_at">剩余：{{ formatDuration(it.remain_seconds) }}</div>
                      <div v-if="it.next_available_at">下次可服务：{{ formatTime(it.next_available_at) }}（{{ formatDuration(it.next_available_in_seconds) }}）</div>
                    </div>
                  </div>
                  <div class="text-gray-400 text-xs">ID: {{ it.technician.id }}</div>
                </div>
              </div>
            </div>
          </div>

        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

const router = useRouter()

const activeTab = ref('rooms')
const loading = ref(false)
const rooms = ref([])
const staff = ref([])
const currentTime = ref(new Date())

// 定时器
let timer = null

// 更新当前时间
const updateCurrentTime = () => {
  currentTime.value = new Date()
}

const goBack = () => {
  router.back()
}

const selectTab = async (t) => {
  activeTab.value = t
  await load()
}

const formatTime = (v) => {
  if (!v) return '-'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const formatDuration = (secs) => {
  const s = Math.max(0, Number(secs || 0))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const ss = Math.floor(s % 60)
  if (h > 0) return `${h}h${String(m).padStart(2, '0')}m`
  return `${m}m${String(ss).padStart(2, '0')}s`
}

const calculateElapsedTime = (startTime) => {
  if (!startTime) return ''
  const start = new Date(startTime)
  const diffMs = currentTime.value - start
  const diffSeconds = Math.floor(diffMs / 1000)
  const h = Math.floor(diffSeconds / 3600)
  const m = Math.floor((diffSeconds % 3600) / 60)
  const s = Math.floor(diffSeconds % 60)
  
  if (h === 0) {
    // 小时为0，只显示 MM:SS
    return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  } else if (h < 10) {
    // 小时第一位是0，显示 M:SS
    return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  } else {
    // 小时两位数，显示 HH:MM:SS
    return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }
}

const sessionStatusText = (st) => {
  const s = String(st || '').trim()
  if (s === 'created') return '已创建'
  if (s === 'room_selecting') return '选房中'
  if (s === 'room_locked') return '房间已锁定'
  if (s === 'staff_selecting') return '选人中'
  if (s === 'precheck_pending') return '待预结单'
  if (s === 'delay_pending') return '延迟中'
  if (s === 'serving') return '服务中'
  if (s === 'auto_finishing') return '待自动结单'
  if (s === 'finished') return '已完成'
  if (s === 'canceled') return '已取消'
  return s || '-'
}

const badgeText = (it) => {
  const st = String(it.service_status || '')
  if (!it.checked_in) return '未签到'
  // 会话状态：显示对应中文
  if (
    st === 'created' ||
    st === 'room_selecting' ||
    st === 'room_locked' ||
    st === 'staff_selecting' ||
    st === 'precheck_pending' ||
    st === 'delay_pending' ||
    st === 'serving' ||
    st === 'auto_finishing' ||
    st === 'finished' ||
    st === 'canceled'
  ) {
    return sessionStatusText(st)
  }

  // 签到状态
  if (st === 'available') return '可服务'
  if (st === 'idle') return '空闲'
  if (st === 'rest') return '休息'
  if (st === 'paused') return '暂停'
  return st || '未知'
}

const badgeClass = (it) => {
  const txt = badgeText(it)
  if (txt === '未签到') return 'bg-gray-100 text-gray-500'
  if (txt === '可服务' || txt === '空闲') return 'bg-green-50 text-green-600'
  if (txt === '服务中') return 'bg-orange-50 text-orange-600'
  return 'bg-blue-50 text-blue-600'
}

const load = async () => {
  loading.value = true
  try {
    if (activeTab.value === 'rooms') {
      const res = await merchantApi.getTableRooms()
      rooms.value = res.data?.data || []
    } else {
      const res = await merchantApi.getTableStaff()
      staff.value = res.data?.data || []
    }
  } catch (e) {
    if (activeTab.value === 'rooms') rooms.value = []
    else staff.value = []
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await load()
  // 启动定时器，每秒更新一次
  timer = setInterval(updateCurrentTime, 1000)
})

onUnmounted(() => {
  // 清理定时器
  if (timer) {
    clearInterval(timer)
    timer = null
  }
})
</script>
