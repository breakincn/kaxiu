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
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-sm font-medium border"
            :class="activeTab === 'service' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectTab('service')"
          >
            服务
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

        <div v-else class="mt-4">
          <div v-if="!activeTab" class="text-center text-gray-400 py-10">
            请选择上方标签查看内容
          </div>
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
                      <div v-if="it.occupied && (it.started_at || it.room_locked_at)" class="text-gray-500 text-sm font-mono">
                        {{ calculateElapsedTime(it.started_at || it.room_locked_at) }}
                      </div>
                    </div>
                    <div v-if="it.occupied" class="text-gray-500 text-sm mt-1">
                      <div v-if="it.technician">{{ it.technician_role?.name || '工作人员' }}: {{ it.technician.account }} {{ it.technician.name }}</div>
                      <div v-if="it.started_at">开始：{{ formatTime(it.started_at) }}</div>
                      <div v-if="it.finish_at">结束：{{ formatTime(it.finish_at) }}</div>
                      <div v-if="it.finish_at">剩余：{{ calculateRemainTime(it.finish_at) }}</div>
                      <div v-if="it.phase_text" class="mt-1">
                        <span class="font-medium" :class="it.phase_class === 'start_pending' ? 'text-red-500' : (it.phase_class === 'finish_failed' ? 'text-red-500' : 'text-blue-600')">
                          {{ it.phase_text }}
                        </span>
                        <span v-if="it.phase_class === 'start_pending' && config.startPendingTimeoutSeconds" class="ml-2 text-red-500 font-mono">
                          {{ calculateRemainTime(it.updated_at, config.startPendingTimeoutSeconds) }}
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
                      <div>
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
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 服务单看板内容 -->
          <div v-else-if="activeTab === 'service'">
            <div class="py-4">
              <!-- 筛选器 -->
              <div class="bg-white rounded-xl shadow-sm p-4 mb-4">
                <div class="flex items-center gap-3">
                  <label class="text-sm font-medium text-gray-700">状态筛选：</label>
                  <select v-model="statusFilter" @change="fetchServiceSessions" class="border border-gray-200 rounded-lg px-3 py-2 text-sm">
                    <option value="">全部</option>
                    <option value="created">已创建</option>
                    <option value="room_selecting">选房中</option>
                    <option value="room_locked">房间已锁定</option>
                    <option value="staff_selecting">选人中</option>
                    <option value="start_pending">{{ replaceTerms('待起单') }}</option>
                    <option value="delay_pending">延迟中</option>
                    <option value="serving">进行中</option>
                    <option value="auto_finishing">{{ replaceTerms('待自动结单') }}</option>
                    <option value="finished">已完成</option>
                    <option value="canceled">已取消</option>
                  </select>
                </div>
              </div>

              <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>

              <div v-else class="space-y-3">
                <!-- 我的服务中会话（技师视角） -->
                <div v-if="isTechnicianAuth() && myServingSessions.length > 0">
                  <div class="bg-white rounded-xl shadow-sm p-4">
                    <div class="font-medium text-gray-800 mb-3 flex items-center gap-2">
                      <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                      </svg>
                      我的服务中
                    </div>
                    <div class="space-y-2">
                      <div v-for="session in myServingSessions" :key="session.id" class="border border-blue-100 rounded-lg p-3 bg-blue-50">
                        <ServiceSessionItem :session="session" @extend="openExtendModal" />
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 其他会话 -->
                <div>
                  <div class="font-medium text-gray-800 mb-3">
                    {{ isTechnicianAuth() ? '其他服务单' : '全部服务单' }}
                    <span class="text-gray-500 text-sm font-normal">({{ filteredOtherSessions.length }})</span>
                  </div>
                  
                  <div v-if="filteredOtherSessions.length === 0" class="text-center text-gray-400 py-10">
                    {{ statusFilter ? '暂无符合条件的服务单' : '暂无服务单' }}
                  </div>
                  
                  <div v-else class="space-y-2">
                    <div v-for="session in filteredOtherSessions" :key="session.id" class="border border-gray-100 rounded-lg p-3">
                      <ServiceSessionItem :session="session" @extend="openExtendModal" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

        </div>
      </div>
    </div>

    <!-- 加钟弹窗 -->
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
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi, serviceSessionApi } from '../../api'
import { replaceTerms } from '../../utils/terms'
import { getMerchantId, isTechnicianAuth, getTechnicianId } from '../../utils/auth'
import ServiceSessionItem from '../../components/ServiceSessionItem.vue'

const router = useRouter()

defineProps({
  embedded: {
    type: Boolean,
    default: false
  }
})

// 从localStorage恢复选中的标签，默认为空（不默认选择）
const savedTab = localStorage.getItem('tableActiveTab')
const activeTab = ref(savedTab || '')
const savedStaffSubTab = localStorage.getItem('tableStaffSubTab')
const staffSubTab = ref(savedStaffSubTab || 'professional')
const loading = ref(false)
const rooms = ref([])
const staff = ref([])
const currentTime = ref(new Date())
const merchant = ref({})
const config = ref({})

const roleAttendanceMap = ref({})

// 服务单看板相关状态
const statusFilter = ref('')
const serviceSessions = ref([])
const extendSession = ref(null)
const extendMinutes = ref(null)
const showExtendModal = ref(false)
const extendLoading = ref(false)

// 技师视角：我的服务中会话
const myServingSessions = computed(() => {
  if (!isTechnicianAuth()) return []
  const techId = getTechnicianId()
  if (!techId) return []
  return serviceSessions.value.filter(s => 
    s.technician_id === techId && 
    ['delay_pending', 'serving', 'auto_finishing'].includes(s.status)
  )
})

// 其他会话（技师视角）或全部会话（商户视角）
const filteredOtherSessions = computed(() => {
  let sessions = isTechnicianAuth() 
    ? serviceSessions.value.filter(s => !myServingSessions.value.some(my => my.id === s.id))
    : serviceSessions.value
  
  if (statusFilter.value) {
    sessions = sessions.filter(s => s.status === statusFilter.value)
  }
  
  return sessions
})

const groupedStaff = computed(() => {
  const list = staff.value || []
  const groups = []
  const idx = {}

  list.forEach((it) => {
    const roleKey = String(it?.technician?.service_role?.key || '').trim()
    const roleName = String(it?.technician?.service_role?.name || '').trim() || '未知岗位'
    const groupKey = roleKey || roleName

    let gi = idx[groupKey]
    if (gi === undefined) {
      gi = groups.length
      idx[groupKey] = gi
      groups.push({ groupKey, roleKey, roleName, items: [] })
    }
    groups[gi].items.push(it)
  })

  return groups
})

// 定时器
let timer = null

// 更新当前时间
const updateCurrentTime = () => {
  currentTime.value = new Date()
}

const goBack = () => {
  router.back()
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

const fetchConfig = async () => {
  try {
    const res = await merchantApi.getConfig()
    config.value = res.data?.data || {}
  } catch (e) {
    console.error('加载配置失败:', e)
    config.value = {}
  }
}

const selectTab = async (t) => {
  activeTab.value = t
  // 保存到localStorage
  localStorage.setItem('tableActiveTab', t)
  await load()
}

const selectStaffSubTab = (t) => {
  staffSubTab.value = t
  localStorage.setItem('tableStaffSubTab', t)
  // 在客服页切换子项时，立即重新加载对应类型的数据
  if (activeTab.value === 'staff') {
    load()
  }
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
  const pad2 = (n) => String(n).padStart(2, '0')
  if (h > 0) return `${h}小时${pad2(m)}分${pad2(ss)}秒`
  return `${m}分${pad2(ss)}秒`
}

const calculateRemainTime = (endTime, durationSeconds = null) => {
  if (!endTime) return '-'
  
  let deadline
  if (durationSeconds !== null) {
    // 如果提供了持续时间，从开始时间计算截止时间
    const start = new Date(endTime)
    deadline = new Date(start.getTime() + durationSeconds * 1000)
  } else {
    // 直接使用结束时间
    deadline = new Date(endTime)
  }
  
  const now = currentTime.value
  const diffMs = deadline - now
  const diffSeconds = Math.floor(diffMs / 1000)

  const pad2 = (n) => String(n).padStart(2, '0')
  if (diffSeconds <= 0) return `0分${pad2(0)}秒`
  
  const h = Math.floor(diffSeconds / 3600)
  const m = Math.floor((diffSeconds % 3600) / 60)
  const s = diffSeconds % 60
  
  if (h > 0) return `${h}小时${pad2(m)}分${pad2(s)}秒`
  return `${m}分${pad2(s)}秒`
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
  if (s === 'start_pending') return replaceTerms('待起单', merchant.value)
  if (s === 'delay_pending') return '延迟中'
  if (s === 'serving') return '服务中'
  if (s === 'auto_finishing') return replaceTerms('待自动结单', merchant.value)
  if (s === 'finished') return '已完成'
  if (s === 'canceled') return '已取消'
  return s || '-'
}

const badgeText = (it) => {
  const st = String(it.service_status || '')
  if (!it.checked_in) return '未签到'

  const sess = it.current_session
  if (sess) {
    const sst = String(sess.status || '').trim()
    const startConfirmedAt = sess.start_confirmed_at
    if (sst === 'start_pending' && !startConfirmedAt) return replaceTerms('服务 待起单', merchant.value)
    if (sst === 'auto_finishing') return replaceTerms('待自动下钟', merchant.value)
    return replaceTerms('服务 待结单', merchant.value)
  }

  // 签到状态：仅保留空闲/暂停
  if (st === 'paused') return '暂停'
  if (st === 'idle') return '空闲'
  return st || '暂停'
}

const getRoleRequireAttendance = (roleKey) => {
  const k = String(roleKey || '')
  if (!k) return true
  const v = roleAttendanceMap.value[k]
  if (typeof v === 'boolean') return v
  return true
}

const shouldShowStaffBadge = (it) => {
  const roleKey = it?.technician?.service_role?.key
  const requireAttendance = getRoleRequireAttendance(roleKey)
  if (requireAttendance) return true
  // 不要求签到的岗位：不展示“未签到/空闲/暂停”等签到相关徽标
  // 但如果有服务会话（例如服务待起单/待结单等），仍然展示服务状态徽标
  return !!it?.current_session
}

const badgeClass = (it) => {
  const txt = badgeText(it)
  if (txt === '未签到') return 'bg-gray-100 text-gray-500'
  if (txt === '空闲') return 'bg-green-50 text-green-600'
  if (txt === replaceTerms('服务 待起单', merchant.value)) return 'bg-red-50 text-red-600'
  if (txt === replaceTerms('待自动下钟', merchant.value)) return 'bg-red-50 text-red-600'
  if (txt === replaceTerms('服务 待结单', merchant.value)) return 'bg-orange-50 text-orange-600'
  if (txt === '暂停') return 'bg-blue-50 text-blue-600'
  return 'bg-blue-50 text-blue-600'
}

const loadRoleAttendanceConfigs = async () => {
  try {
    const res = await merchantApi.getRoleAttendanceConfigs()
    const list = res.data?.data || []
    const m = {}
    list.forEach((it) => {
      if (!it || !it.service_role_key) return
      m[String(it.service_role_key)] = !!it.require_attendance
    })
    roleAttendanceMap.value = m
  } catch (e) {
    roleAttendanceMap.value = {}
  }
}

const load = async () => {
  loading.value = true
  try {
    if (activeTab.value === 'rooms') {
      const res = await merchantApi.getTableRooms()
      rooms.value = res.data?.data || []
    } else if (activeTab.value === 'staff') {
      const type = staffSubTab.value === 'operation' ? 'operation' : 'professional'
      const res = await merchantApi.getTableStaff(type)
      staff.value = res.data?.data || []
    } else if (activeTab.value === 'service') {
      await fetchServiceSessions()
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

const fetchServiceSessions = async () => {
  try {
    const params = {}
    if (statusFilter.value) params.status = statusFilter.value
    const res = await serviceSessionApi.listSessions(params)
    serviceSessions.value = res.data?.data || []
  } catch (e) {
    console.error('获取服务单列表失败:', e)
    serviceSessions.value = []
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
      const idx = serviceSessions.value.findIndex(s => s.id === updated.id)
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
  // 如果没有保存的标签，默认选择房间
  if (!activeTab.value) {
    activeTab.value = 'rooms'
    localStorage.setItem('tableActiveTab', 'rooms')
  }
  await fetchMerchant()
  await fetchConfig()
  await loadRoleAttendanceConfigs()
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
