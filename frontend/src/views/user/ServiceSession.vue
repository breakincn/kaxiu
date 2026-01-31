<template>
  <div class="min-h-screen bg-gray-50 pb-6">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">到店服务</span>
    </header>

    <div class="px-4 mt-4 space-y-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center justify-between">
          <div>
            <div class="font-medium text-gray-800">服务单 #{{ session?.id || '-' }}</div>
            <div class="text-gray-500 text-sm mt-1">状态：{{ statusText(session?.status) }}</div>
          </div>
          <button
            class="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium"
            :disabled="loading"
            @click="refresh"
          >
            刷新
          </button>
        </div>

        <div class="mt-3 text-gray-500 text-sm">
          房间：{{ session?.room?.name || '-' }} / 工作人员：{{ session?.technician?.name || session?.technician?.account || '-' }}
        </div>

        <div v-if="nextStepText" class="mt-3 p-3 bg-primary-light border border-gray-100 rounded-lg text-sm text-primary">
          {{ nextStepText }}
        </div>
      </div>

      <div v-if="needsRoom" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="font-medium text-gray-800 mb-3">选择房间</div>

        <div v-if="roomsLoading" class="text-gray-500 text-sm">加载房间中...</div>
        <div v-else-if="rooms.length === 0" class="text-gray-500 text-sm">暂无可用房间</div>
        <div v-else class="space-y-2">
          <button
            v-for="r in rooms"
            :key="r.id"
            class="w-full px-4 py-3 border border-gray-200 rounded-lg text-left hover:bg-gray-50"
            @click="chooseRoom(r.id)"
            :disabled="actionLoading"
          >
            {{ r.name }}
          </button>
        </div>

        <div v-if="roomAutoAdjustedMsg" class="mt-3 p-3 bg-primary-light border border-gray-100 rounded-lg text-sm text-primary">
          {{ roomAutoAdjustedMsg }}
        </div>
      </div>

      <!-- 已选择的房间 -->
      <div v-if="session?.room && canChooseTechnician" class="bg-primary-light border border-gray-100 rounded-2xl p-4">
        <div class="text-sm text-primary font-medium mb-1">已选择房间</div>
        <div class="text-lg font-bold text-primary">{{ session.room.name }}</div>
      </div>

      <div v-if="isStaffSelectCooling" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="font-medium text-gray-800 mb-3">选择工作人员</div>
        <div class="text-gray-500 text-sm">{{ staffSelectCooldownText }}</div>
      </div>

      <div v-else-if="canChooseTechnician" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="font-medium text-gray-800 mb-3">选择工作人员</div>

        <div v-if="techLoading" class="text-gray-500 text-sm">加载工作人员中...</div>
        <div v-else-if="technicians.length === 0" class="text-gray-500 text-sm">暂无可选工作人员（需今日上班签到且未下班且可服务）</div>
        <div v-else class="space-y-2">
          <button
            v-for="t in technicians"
            :key="t.technician_id"
            class="w-full px-4 py-3 border border-gray-200 rounded-lg text-left hover:bg-gray-50"
            @click="chooseTechnician(t.technician_id)"
            :disabled="actionLoading"
          >
            {{ t.technician?.name || t.technician?.account || ('ID:' + t.technician_id) }}
          </button>
        </div>
      </div>

      <div v-if="canExtend" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="font-medium text-gray-800 mb-3">加钟</div>
        <div class="text-gray-600 text-sm mb-3">延长服务时间（5~180分钟）</div>
        <div class="flex items-center gap-2">
          <input v-model.number="extendMinutes" type="number" min="5" max="180" placeholder="分钟" class="flex-1 px-4 py-3 border border-gray-200 rounded-lg">
          <button :disabled="extendLoading || !extendMinutes || extendMinutes < 5 || extendMinutes > 180" @click="doExtend" class="px-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50">
            {{ extendLoading ? '加钟中...' : '加钟' }}
          </button>
        </div>
      </div>

      <div v-if="errorText" class="p-3 bg-gray-50 border border-gray-100 rounded-lg text-sm text-gray-700">
        {{ errorText }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { replaceTerms } from '../../utils/terms'
import { normalizeSessionStatus } from '../../utils/sessionStatus'
import { userServiceSessionApi } from '../../api/index'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const actionLoading = ref(false)
const roomsLoading = ref(false)
const techLoading = ref(false)
const extendLoading = ref(false)

const session = ref(null)
const rooms = ref([])
const technicians = ref([])
const roomAutoAdjustedMsg = ref('')
const errorText = ref('')
const extendMinutes = ref(null)


const nowTick = ref(Date.now())
let nowTickTimer = null

const startNowTickTimer = () => {
  if (nowTickTimer) return
  nowTickTimer = setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
}


const sessionId = computed(() => String(route.params.id || ''))

const staffSelectCooldownLeftMs = computed(() => {
  const dl = session.value?.staff_select_cooldown_until
  if (!dl) return 0
  const ms = new Date(dl).getTime()
  if (Number.isNaN(ms)) return 0
  return Math.max(0, ms - nowTick.value)
})

const isStaffSelectCooling = computed(() => {
  return staffSelectCooldownLeftMs.value > 0
})

const staffSelectCooldownText = computed(() => {
  const ms = staffSelectCooldownLeftMs.value
  if (!ms) return ''
  const totalSeconds = Math.floor(ms / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `当前没有空闲客服，${minutes}分${seconds}秒后可再次选择客服`
})

const needsRoom = computed(() => {
  if (!session.value) return false
  return normalizeSessionStatus(session.value.status) === 'room_selecting'
})

const canChooseTechnician = computed(() => {
  if (!session.value) return false
  if (isStaffSelectCooling.value) return false
  const st = normalizeSessionStatus(session.value.status)
  return st === 'staff_selecting' || st === 'room_locked'
})

const canExtend = computed(() => {
  return normalizeSessionStatus(session.value?.status) === 'serving'
})

const nextStepText = computed(() => {
  const step = route.query.next_step
  const map = {
    room_select: '请先选择房间',
    staff_select: '请选择工作人员'
  }
  return step ? map[step] : ''
})

const statusText = (s) => {
  const st = normalizeSessionStatus(s)
  const statusMap = {
    room_selecting: '选房中',
    room_locked: '房间已锁定',
    staff_selecting: '选人中',
    start_pending: replaceTerms('待起单'),
    delay_pending: '延迟中',
    serving: '进行中',
    auto_finishing: replaceTerms('待自动结单'),
    finished: '已完成'
  }
  return statusMap[st] || st || '-'
}

const goBack = () => router.back()


const resumeIfCanceled = async () => {
  if (!session.value) return false
  if (normalizeSessionStatus(session.value.status) !== 'canceled') return false
  try {
    const res = await userServiceSessionApi.resume(sessionId.value)
    session.value = res.data?.data || session.value
    return true
  } catch (e) {
    errorText.value = e.response?.data?.error || '服务单恢复失败'
    return false
  }
}


const refresh = async () => {
  errorText.value = ''
  roomAutoAdjustedMsg.value = ''
  loading.value = true
  try {
    const res = await userServiceSessionApi.getSession(sessionId.value)
    session.value = res.data?.data || null

    // 若服务单已取消：自动恢复后再继续按状态加载
    if (await resumeIfCanceled()) {
      const res2 = await userServiceSessionApi.getSession(sessionId.value)
      session.value = res2.data?.data || session.value
    }

    // 根据状态加载列表
    if (needsRoom.value) {
      await loadRooms()
    }
    if (canChooseTechnician.value) {
      await loadTechnicians()
    }
  } catch (e) {
    errorText.value = e.response?.data?.error || '加载失败'
  } finally {
    loading.value = false
  }
}

const loadRooms = async () => {
  roomsLoading.value = true
  try {
    const res = await userServiceSessionApi.listRooms(sessionId.value)
    rooms.value = res.data?.data || []
  } catch (e) {
    rooms.value = []
  } finally {
    roomsLoading.value = false
  }
}

const loadTechnicians = async () => {
  techLoading.value = true
  try {
    const res = await userServiceSessionApi.listTechnicians(sessionId.value)
    technicians.value = res.data?.data || []
  } catch (e) {
    technicians.value = []
    errorText.value = e.response?.data?.error || ''
  } finally {
    techLoading.value = false
  }
}

const chooseRoom = async (roomId) => {
  actionLoading.value = true
  errorText.value = ''
  roomAutoAdjustedMsg.value = ''
  try {
    // 防止页面展示的状态与后端实际状态不一致：提交前先同步一次最新会话状态
    try {
      const sres = await userServiceSessionApi.getSession(sessionId.value)
      session.value = sres.data?.data || session.value
    } catch (_) {
      // ignore
    }

    // 若会话已取消，先尝试恢复，再继续
    if (await resumeIfCanceled()) {
      try {
        const sres2 = await userServiceSessionApi.getSession(sessionId.value)
        session.value = sres2.data?.data || session.value
      } catch (_) {
        // ignore
      }
    }

    // 若当前已不在选房状态，直接刷新页面按最新状态展示（避免 400）
    if (session.value?.status !== 'room_selecting') {
      const msg = '当前状态不可选房'
      errorText.value = msg
      alert(msg)
      await refresh()
      return
    }

    const res = await userServiceSessionApi.chooseRoom(sessionId.value, { room_id: roomId })
    const data = res.data || {}
    session.value = data.data || null
    if (data.room_auto_adjusted) {
      roomAutoAdjustedMsg.value = data.message || '房间已自动调整'
    }
    await refresh()
  } catch (e) {
    const msg = e.response?.data?.error || '选房失败'
    errorText.value = msg
    alert(msg)
    await refresh()
  } finally {
    actionLoading.value = false
  }
}

const chooseTechnician = async (technicianId) => {
  actionLoading.value = true
  errorText.value = ''
  try {
    const res = await userServiceSessionApi.chooseTechnician(sessionId.value, { technician_id: technicianId })
    const data = res.data || {}
    session.value = data.data || null

    const cardId = session.value?.card_id
    const sid = session.value?.id
    if (cardId && sid) {
      await router.replace({
        path: `/user/cards/${cardId}`,
        query: { session_id: String(sid) }
      })
    }
  } catch (e) {
    errorText.value = e.response?.data?.error || '选人失败'
  } finally {
    actionLoading.value = false
  }
}

const doExtend = async () => {
  if (!extendMinutes.value || extendMinutes.value < 5 || extendMinutes.value > 180) {
    errorText.value = '请输入5~180分钟的加钟时长'
    return
  }
  extendLoading.value = true
  errorText.value = ''
  try {
    const res = await userServiceSessionApi.extend(sessionId.value, { minutes: extendMinutes.value })
    session.value = res.data?.data || null
    extendMinutes.value = null
  } catch (e) {
    errorText.value = e.response?.data?.error || '加钟失败'
  } finally {
    extendLoading.value = false
  }
}

onMounted(async () => {
  startNowTickTimer()
  await refresh()
})
</script>
