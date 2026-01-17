<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">服务单看板</span>
      <div class="flex-1"></div>
      <button @click="refresh" class="px-3 py-1.5 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium" :disabled="loading">
        刷新
      </button>
    </header>

    <div class="px-4 py-4">
      <!-- 筛选器 -->
      <div class="bg-white rounded-xl shadow-sm p-4 mb-4">
        <div class="flex items-center gap-3">
          <label class="text-sm font-medium text-gray-700">状态筛选：</label>
          <select v-model="statusFilter" class="border border-gray-200 rounded-lg px-3 py-2 text-sm">
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
        <div class="bg-white rounded-xl shadow-sm p-4">
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { serviceSessionApi } from '../../api'
import { formatDateTime, formatDate } from '../../utils/dateFormat'
import { getMerchantId, hasMerchantPermission, isTechnicianAuth, getTechnicianId } from '../../utils/auth'
import ServiceSessionItem from '../../components/ServiceSessionItem.vue'
import { replaceTerms } from '../../utils/terms'

const router = useRouter()

const loading = ref(false)
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

const goBack = () => router.back()

const refresh = async () => {
  await fetchServiceSessions()
}

const fetchServiceSessions = async () => {
  loading.value = true
  try {
    const params = {}
    if (statusFilter.value) params.status = statusFilter.value
    const res = await serviceSessionApi.listSessions(params)
    serviceSessions.value = res.data?.data || []
  } catch (e) {
    console.error('获取服务单列表失败:', e)
    serviceSessions.value = []
  } finally {
    loading.value = false
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
      // 更新会话列表中的对应项
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

onMounted(() => {
  fetchServiceSessions()
})
</script>
