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
            <div class="font-medium text-gray-800">会话 #{{ session?.id || '-' }}</div>
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

      <div v-if="canChooseTechnician" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="font-medium text-gray-800 mb-3">选择工作人员</div>

        <div v-if="techLoading" class="text-gray-500 text-sm">加载工作人员中...</div>
        <div v-else-if="technicians.length === 0" class="text-gray-500 text-sm">暂无可选工作人员（需最近15分钟签到且可服务）</div>
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

      <div v-if="precheckCode" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="font-medium text-gray-800 mb-2">预结单码</div>
        <div class="text-gray-600 text-sm">请向工作人员出示此码，由工作人员扫码开始计时</div>
        <div class="mt-3 flex items-center gap-2">
          <div class="flex-1 px-4 py-3 border border-gray-200 rounded-lg font-mono">{{ precheckCode }}</div>
          <button class="px-4 py-3 bg-primary text-white rounded-lg font-medium" @click="copy(precheckCode)">
            复制
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
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { userServiceSessionApi } from '../../api'

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

const precheckCode = ref('')
const roomAutoAdjustedMsg = ref('')
const errorText = ref('')
const extendMinutes = ref(null)

const sessionId = computed(() => String(route.params.id || ''))

const needsRoom = computed(() => {
  if (!session.value) return false
  return session.value.status === 'room_selecting'
})

const canChooseTechnician = computed(() => {
  if (!session.value) return false
  return session.value.status === 'staff_selecting' || session.value.status === 'room_locked'
})

const canExtend = computed(() => {
  return session.value?.status === 'serving'
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
  const m = {
    room_selecting: '选房中',
    room_locked: '房间已锁定',
    staff_selecting: '选人中',
    precheck_pending: '待预结单',
    delay_pending: '延迟中',
    serving: '进行中',
    auto_finishing: '待自动结单',
    finished: '已完成'
  }
  return m[s] || s || '-'
}

const goBack = () => router.back()

const copy = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    alert('已复制')
  } catch (_) {
    alert('复制失败')
  }
}

const refresh = async () => {
  errorText.value = ''
  roomAutoAdjustedMsg.value = ''
  loading.value = true
  try {
    const res = await userServiceSessionApi.getSession(sessionId.value)
    session.value = res.data?.data || null

    // 根据状态加载列表
    if (needsRoom.value) {
      await loadRooms()
    }
    if (canChooseTechnician.value) {
      await loadTechnicians()
    }

    if (session.value?.status === 'precheck_pending' && session.value?.id) {
      precheckCode.value = `SS:${session.value.id}`
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
  } finally {
    techLoading.value = false
  }
}

const chooseRoom = async (roomId) => {
  actionLoading.value = true
  errorText.value = ''
  roomAutoAdjustedMsg.value = ''
  try {
    const res = await userServiceSessionApi.chooseRoom(sessionId.value, { room_id: roomId })
    const data = res.data || {}
    session.value = data.data || null
    if (data.room_auto_adjusted) {
      roomAutoAdjustedMsg.value = data.message || '房间已自动调整'
    }
    await refresh()
  } catch (e) {
    errorText.value = e.response?.data?.error || '选房失败'
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
    precheckCode.value = data.precheck_code || (session.value?.id ? `SS:${session.value.id}` : '')
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
  await refresh()
})
</script>
