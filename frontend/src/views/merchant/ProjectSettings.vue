<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">项目设置</span>
    </header>

    <div class="px-4 py-6">
      <div v-if="loading" class="text-gray-400 text-center py-10">加载中...</div>

      <div v-else class="space-y-4">
        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">项目列表</div>
          </div>
          
          <div class="px-4 py-4 space-y-4">
            <div
              v-for="(project, index) in form.projects"
              :key="index"
              :class="[
                'border rounded-lg p-4 space-y-3',
                project._isNewUnsaved ? 'border-blue-500 bg-blue-50/40' : 'border-gray-200'
              ]"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <div class="text-sm font-medium text-gray-700">项目 {{ index + 1 }}</div>
                  <button
                    v-if="!project.is_default"
                    type="button"
                    @click="setDefaultProject(index)"
                    class="text-sm font-medium text-primary"
                  >
                    设为默认
                  </button>
                  <span v-else class="px-2 py-0.5 rounded-full bg-orange-50 text-orange-600 text-xs font-medium border border-orange-100">默认项目</span>
                </div>
                <div class="flex items-center gap-3">
                  <button 
                    @click="removeProject(index)"
                    class="text-red-500 hover:text-red-700"
                  >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                    </svg>
                  </button>
                </div>
              </div>
              
              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">项目名</div>
                <input
                  v-model="project.name"
                  type="text"
                  placeholder="如 A 项目(课型)"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
              
              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">项目服务时长（分钟）</div>
                <input
                  v-model.number="project.duration"
                  type="number"
                  min="1"
                  placeholder="如 45"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">服务开始延迟时间（秒）</div>
                <input
                  v-model.number="project.start_delay_seconds"
                  type="number"
                  min="0"
                  max="3600"
                  placeholder="如 60"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">服务间歇时间（分钟）</div>
                <input
                  v-model.number="project.service_gap_minutes"
                  type="number"
                  min="0"
                  max="60"
                  placeholder="如 3"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div v-if="isRoomServiceEnabled">
                <div class="text-sm font-medium text-gray-700 mb-2">自动分配房间延迟时间（分钟）</div>
                <input
                  v-model.number="project.room_select_timeout_minutes"
                  type="number"
                  min="0.5"
                  max="60"
                  step="0.5"
                  placeholder="如 1.5"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div v-if="isCustomerServiceModeEnabled">
                <div class="text-sm font-medium text-gray-700 mb-2">自动分配客服延迟时间（分钟）</div>
                <input
                  v-model.number="project.auto_assign_technician_delay_minutes"
                  type="number"
                  min="0"
                  max="180"
                  placeholder="如 5"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div v-if="isCustomerServiceModeEnabled">
                <div class="text-sm font-medium text-gray-700 mb-2">{{ startPendingCountdownLabel }}</div>
                <input
                  v-model.number="project.start_pending_timeout_minutes"
                  type="number"
                  min="1"
                  max="60"
                  placeholder="如 5"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">服务人数</div>
                <input
                  v-model.number="project.service_capacity"
                  type="number"
                  min="1"
                  max="999"
                  placeholder="如 1"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <div class="mt-1 text-xs text-gray-400">同一项目单次可服务的人数，如舞蹈课可设置为 15。</div>
              </div>

              <div>
                <div class="flex items-center justify-between mb-2">
                  <div class="text-sm font-medium text-gray-700">服务时间</div>
                  <div class="flex items-center gap-3">
                    <label class="flex items-center gap-1 text-sm text-gray-700">
                      <input
                        v-model="project.service_time_slot_mode"
                        type="radio"
                        value="weekly"
                        class="h-4 w-4 border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      周
                    </label>
                    <label class="flex items-center gap-1 text-sm text-gray-700">
                      <input
                        v-model="project.service_time_slot_mode"
                        type="radio"
                        value="monthly"
                        class="h-4 w-4 border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      月
                    </label>
                    <button
                      type="button"
                      @click="addServiceTimeSlot(project)"
                      class="text-sm font-medium text-primary"
                    >
                      + 添加时间
                    </button>
                  </div>
                </div>
                <div v-if="!project.service_time_slots || project.service_time_slots.length === 0" class="text-xs text-gray-400">未设置固定服务时间。</div>
                <div v-else class="space-y-2">
                  <div
                    v-for="(slot, slotIndex) in project.service_time_slots"
                    :key="slotIndex"
                    class="grid grid-cols-[1fr_1fr_auto] gap-2 items-center"
                  >
                    <select
                      v-if="slot.recurrence_type !== 'monthly'"
                      v-model.number="slot.weekday"
                      class="h-10 min-w-0 px-3 py-0 border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    >
                      <option
                        v-for="day in weekdayOptions"
                        :key="day.value"
                        :value="day.value"
                      >
                        {{ day.label }}
                      </option>
                    </select>
                    <select
                      v-else
                      v-model.number="slot.month_day"
                      class="h-10 min-w-0 px-3 py-0 border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    >
                      <option
                        v-for="day in monthDayOptions"
                        :key="day"
                        :value="day"
                      >
                        {{ day }}日
                      </option>
                    </select>
                    <div class="relative min-w-0">
                      <input
                        v-model="slot.start_time"
                        type="time"
                        class="service-time-input h-10 w-full min-w-0 px-3 pr-9 py-0 border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      />
                      <svg
                        class="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-700"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <circle cx="12" cy="12" r="9" stroke-width="2"/>
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 7v5l3 2"/>
                      </svg>
                    </div>
                    <button
                      type="button"
                      @click="removeServiceTimeSlot(project, slotIndex)"
                      class="h-10 px-2 text-red-500"
                    >
                      删除
                    </button>
                  </div>
                </div>
              </div>

              <div class="pt-2 border-t border-gray-100">
                <div class="text-sm font-medium text-gray-800 mb-2">拖堂补偿设置</div>
                <div class="text-xs text-gray-400 mb-3">用于预约拖堂账本累计与自动兑现。</div>
              </div>

              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">拖堂容忍分钟数</div>
                <input
                  v-model.number="project.delay_tolerance_minutes"
                  type="number"
                  min="0"
                  max="180"
                  placeholder="如 1"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">拖堂补偿模式</div>
                <select
                  v-model="project.delay_compensation_mode"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="minutes_bucket">分钟累计兑 1 次</option>
                  <option value="fixed_unit">固定值兑现</option>
                  <option value="amount_bucket">金额累计兑现</option>
                </select>
              </div>

              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">兑现阈值（服务时长百分比）</div>
                <input
                  v-model.number="project.delay_redeem_threshold_percent"
                  type="number"
                  min="1"
                  max="1000"
                  placeholder="如 100"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <div class="mt-1 text-xs text-gray-400">例如 60 表示累计达到一个服务单位的 60% 也可兑现。</div>
              </div>

              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">
                  {{ project.delay_compensation_mode === 'amount_bucket' ? '单个服务单位金额' : '固定兑现值' }}
                </div>
                <input
                  v-model.number="project.delay_fixed_unit_value"
                  type="number"
                  min="0"
                  placeholder="如 100"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <div class="mt-1 text-xs text-gray-400">
                  <template v-if="project.delay_compensation_mode === 'amount_bucket'">
                    用于金额卡按拖堂比例累计补偿值。
                  </template>
                  <template v-else-if="project.delay_compensation_mode === 'fixed_unit'">
                    达到阈值后每次自动补的固定次数/单位值。
                  </template>
                  <template v-else>
                    分钟累计模式下可留空为 0。
                  </template>
                </div>
              </div>

              <label class="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-3">
                <div>
                  <div class="text-sm font-medium text-gray-700">参与公开预约</div>
                  <div class="mt-1 text-xs text-gray-400">关闭后，该项目不会出现在用户预约项目和碎片计算中</div>
                </div>
                <input
                  v-model="project.bookable_online"
                  type="checkbox"
                  class="h-5 w-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
              </label>
            </div>
            
            <button
              @click="addProject"
              class="w-full py-2 border border-dashed border-gray-300 rounded-lg text-gray-600 hover:border-gray-400 hover:text-gray-800 transition-colors"
            >
              + 添加项目
            </button>
          </div>
        </div>

        <button
          @click="save"
          :disabled="loading || saving || !isDirty"
          class="w-full py-3 bg-primary text-white rounded-lg font-medium hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi, merchantProjectApi } from '../../api'
import { getStartCountdownLabel } from '../../utils/terms'

const router = useRouter()

const loading = ref(true)
const saving = ref(false)
const initialSnapshot = ref('')
const merchantTerms = ref(null)

const removedProjectIds = ref([])

const form = ref({
  projects: []
})

const weekdayOptions = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 7, label: '周日' }
]

const monthDayOptions = Array.from({ length: 31 }, (_, index) => index + 1)

const normalizeServiceTimeSlots = (slots) => (Array.isArray(slots) ? slots : [])
  .map(slot => ({
    recurrence_type: String(slot?.recurrence_type || 'weekly'),
    weekday: Number(slot?.weekday || 0),
    month_day: Number(slot?.month_day || 0),
    start_time: String(slot?.start_time || '').trim()
  }))
  .map(slot => {
    if (slot.recurrence_type === 'monthly') {
      return {
        recurrence_type: 'monthly',
        month_day: slot.month_day,
        start_time: slot.start_time
      }
    }
    return {
      recurrence_type: 'weekly',
      weekday: slot.weekday,
      start_time: slot.start_time
    }
  })
  .filter(slot => {
    if (!slot.start_time) return false
    if (slot.recurrence_type === 'monthly') {
      return slot.month_day >= 1 && slot.month_day <= 31
    }
    return slot.weekday >= 1 && slot.weekday <= 7
  })
  .sort((a, b) => {
    if (a.recurrence_type !== b.recurrence_type) {
      return a.recurrence_type.localeCompare(b.recurrence_type)
    }
    if (a.recurrence_type === 'monthly') {
      return (a.month_day - b.month_day) || a.start_time.localeCompare(b.start_time)
    }
    return (a.weekday - b.weekday) || a.start_time.localeCompare(b.start_time)
  })

const startPendingCountdownLabel = computed(() => `${getStartCountdownLabel(merchantTerms.value)}（分钟）`)
const isRoomServiceEnabled = computed(() => !!merchantTerms.value?.support_room)
const isCustomerServiceModeEnabled = computed(() => !!merchantTerms.value?.support_customer_service_mode)

const normalizeProjectsState = (projects, removedIds = []) => JSON.stringify({
  projects: (projects || []).map((project) => ({
    id: project.id ?? null,
    name: String(project.name || '').trim(),
    duration: Number(project.duration || 0),
    bookable_online: project.bookable_online !== false,
    service_gap_minutes: Number(project.service_gap_minutes ?? 3),
    start_delay_seconds: Number(project.start_delay_seconds ?? 60),
    room_select_timeout_minutes: Number(project.room_select_timeout_minutes ?? 1.5),
    start_pending_timeout_minutes: Number(project.start_pending_timeout_minutes ?? 5),
    service_capacity: Number(project.service_capacity ?? 1),
    service_time_slots: normalizeServiceTimeSlots(project.service_time_slots),
    auto_assign_technician_delay_minutes: Number(project.auto_assign_technician_delay_minutes ?? 5),
    delay_tolerance_minutes: Number(project.delay_tolerance_minutes ?? 1),
    delay_compensation_mode: String(project.delay_compensation_mode || 'minutes_bucket'),
    delay_redeem_threshold_percent: Number(project.delay_redeem_threshold_percent ?? 100),
    delay_fixed_unit_value: Number(project.delay_fixed_unit_value ?? 0),
    is_default: project.is_default === true
  })),
  removedProjectIds: [...removedIds].sort((a, b) => Number(a) - Number(b))
})

const isDirty = computed(() => (
  normalizeProjectsState(form.value.projects, removedProjectIds.value) !== initialSnapshot.value
))

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
    setTimeout(() => {
      if (router.currentRoute.value.path === '/merchant/project-settings') {
        router.push('/merchant/settings')
      }
    }, 80)
    return
  }
  router.push('/merchant/settings')
}

const load = async () => {
  loading.value = true
  try {
    const merchantRes = await merchantApi.getCurrentMerchant()
    merchantTerms.value = merchantRes.data?.data || null
    const res = await merchantProjectApi.list()
    const list = res.data?.data || []
    removedProjectIds.value = []
    form.value = {
      projects: list.length > 0
        ? list.map(p => ({
          id: p.id,
          name: p.name,
          duration: p.duration,
          bookable_online: p.bookable_online !== false,
          service_gap_minutes: Number(p.service_gap_minutes ?? 3),
          start_delay_seconds: Number(p.start_delay_seconds ?? 60),
          room_select_timeout_minutes: Number(p.room_select_timeout_seconds ?? 90) / 60,
          start_pending_timeout_minutes: Number(p.start_pending_timeout_seconds ?? 300) / 60,
          service_capacity: Number(p.service_capacity ?? 1),
          service_time_slot_mode: 'weekly',
          service_time_slots: normalizeServiceTimeSlots(p.service_time_slots),
          auto_assign_technician_delay_minutes: Number(p.auto_assign_technician_delay_minutes ?? 5),
          delay_tolerance_minutes: Number(p.delay_tolerance_minutes ?? 1),
          delay_compensation_mode: String(p.delay_compensation_mode || 'minutes_bucket'),
          delay_redeem_threshold_percent: Number(p.delay_redeem_threshold_percent ?? 100),
          delay_fixed_unit_value: Number(p.delay_fixed_unit_value ?? 0),
          is_default: p.is_default === true
        }))
        : []
    }
    initialSnapshot.value = normalizeProjectsState(form.value.projects, removedProjectIds.value)
  } catch (e) {
    console.error('加载项目设置失败', e)
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const addProject = () => {
  form.value.projects.push({
    id: null,
    name: '',
    duration: null,
    bookable_online: true,
    service_gap_minutes: 3,
    start_delay_seconds: 60,
    room_select_timeout_minutes: 1.5,
    start_pending_timeout_minutes: 5,
    service_capacity: 1,
    service_time_slot_mode: 'weekly',
    service_time_slots: [],
    auto_assign_technician_delay_minutes: 5,
    delay_tolerance_minutes: 1,
    delay_compensation_mode: 'minutes_bucket',
    delay_redeem_threshold_percent: 100,
    delay_fixed_unit_value: 0,
    is_default: form.value.projects.length === 0,
    _isNewUnsaved: true
  })
}

const addServiceTimeSlot = (project) => {
  if (!Array.isArray(project.service_time_slots)) {
    project.service_time_slots = []
  }
  if (project.service_time_slot_mode === 'monthly') {
    project.service_time_slots.push({
      recurrence_type: 'monthly',
      month_day: 1,
      start_time: '18:00'
    })
    return
  }
  project.service_time_slot_mode = 'weekly'
  project.service_time_slots.push({
    recurrence_type: 'weekly',
    weekday: 7,
    start_time: '18:00'
  })
}

const removeServiceTimeSlot = (project, slotIndex) => {
  if (!Array.isArray(project.service_time_slots)) {
    project.service_time_slots = []
    return
  }
  project.service_time_slots.splice(slotIndex, 1)
}

const setDefaultProject = (index) => {
  form.value.projects = form.value.projects.map((project, i) => ({
    ...project,
    is_default: i === index
  }))
}

const removeProject = (index) => {
  if (form.value.projects.length <= 1) {
    alert('项目列表至少保留一个项目')
    return
  }
  const p = form.value.projects[index]
  const projectName = String(p?.name || '').trim() || `项目 ${index + 1}`
  if (!window.confirm(`确定删除“${projectName}”吗？`)) {
    return
  }
  if (p && p.id) {
    removedProjectIds.value.push(p.id)
  }
  form.value.projects.splice(index, 1)
  if (!form.value.projects.some(project => project.is_default)) {
    form.value.projects = form.value.projects.map((project, i) => ({
      ...project,
      is_default: i === 0
    }))
  }
}

const save = async () => {
  if (saving.value) return
  
  // 验证项目列表
  for (let i = 0; i < form.value.projects.length; i++) {
    const project = form.value.projects[i]
    if (!project.name || !project.name.trim()) {
      alert(`项目 ${i + 1} 的名称不能为空`)
      return
    }
    if (!project.duration || project.duration < 1) {
      alert(`项目 ${i + 1} 的服务时长必须大于等于1`)
      return
    }
    const delaySeconds = Number(project.start_delay_seconds ?? 60)
    if (!Number.isFinite(delaySeconds) || delaySeconds < 0 || delaySeconds > 3600) {
      alert(`项目 ${i + 1} 的服务开始延迟时间必须在 0-3600 秒之间`)
      return
    }
    if (isRoomServiceEnabled.value) {
      const roomSelectTimeoutMinutes = Number(project.room_select_timeout_minutes ?? 1.5)
      if (!Number.isFinite(roomSelectTimeoutMinutes) || roomSelectTimeoutMinutes < 0.5 || roomSelectTimeoutMinutes > 60) {
        alert(`项目 ${i + 1} 的自动分配房间延迟时间必须在 0.5-60 分钟之间`)
        return
      }
    }
    if (isCustomerServiceModeEnabled.value) {
      const startPendingTimeoutMinutes = Number(project.start_pending_timeout_minutes ?? 5)
      if (!Number.isFinite(startPendingTimeoutMinutes) || startPendingTimeoutMinutes < 1 || startPendingTimeoutMinutes > 60) {
        alert(`项目 ${i + 1} 的${getStartCountdownLabel(merchantTerms.value)}必须在 1-60 分钟之间`)
        return
      }
    }
    const serviceCapacity = Number(project.service_capacity ?? 1)
    if (!Number.isFinite(serviceCapacity) || serviceCapacity < 1 || serviceCapacity > 999) {
      alert(`项目 ${i + 1} 的服务人数必须在 1-999 之间`)
      return
    }
    const serviceTimeSlots = Array.isArray(project.service_time_slots) ? project.service_time_slots : []
    for (let j = 0; j < serviceTimeSlots.length; j++) {
      const slot = serviceTimeSlots[j]
      const recurrenceType = String(slot.recurrence_type || 'weekly')
      if (!slot.start_time) {
        alert(`项目 ${i + 1} 的服务时间 ${j + 1} 必须选择开始时间`)
        return
      }
      if (recurrenceType === 'monthly') {
        if (!slot.month_day || slot.month_day < 1 || slot.month_day > 31) {
          alert(`项目 ${i + 1} 的服务时间 ${j + 1} 必须选择 1-31 日`)
          return
        }
      } else if (!slot.weekday || slot.weekday < 1 || slot.weekday > 7) {
        alert(`项目 ${i + 1} 的服务时间 ${j + 1} 必须选择星期`)
        return
      }
    }
    const gapMinutes = Number(project.service_gap_minutes ?? 3)
    if (!Number.isFinite(gapMinutes) || gapMinutes < 0 || gapMinutes > 60) {
      alert(`项目 ${i + 1} 的服务间歇时间必须在 0-60 分钟之间`)
      return
    }
    if (isCustomerServiceModeEnabled.value) {
      const autoAssignDelayMinutes = Number(project.auto_assign_technician_delay_minutes ?? 5)
      if (!Number.isFinite(autoAssignDelayMinutes) || autoAssignDelayMinutes < 0 || autoAssignDelayMinutes > 180) {
        alert(`项目 ${i + 1} 的自动分配客服延迟时间必须在 0-180 分钟之间`)
        return
      }
    }
    const toleranceMinutes = Number(project.delay_tolerance_minutes ?? 1)
    if (!Number.isFinite(toleranceMinutes) || toleranceMinutes < 0 || toleranceMinutes > 180) {
      alert(`项目 ${i + 1} 的拖堂容忍分钟数必须在 0-180 分钟之间`)
      return
    }
    const redeemPercent = Number(project.delay_redeem_threshold_percent ?? 100)
    if (!Number.isFinite(redeemPercent) || redeemPercent < 1 || redeemPercent > 1000) {
      alert(`项目 ${i + 1} 的兑现阈值百分比必须在 1-1000 之间`)
      return
    }
    const fixedUnitValue = Number(project.delay_fixed_unit_value ?? 0)
    if (!Number.isFinite(fixedUnitValue) || fixedUnitValue < 0) {
      alert(`项目 ${i + 1} 的固定兑现值不能小于 0`)
      return
    }
  }
  const defaultCount = form.value.projects.filter(project => project.is_default).length
  if (defaultCount !== 1) {
    alert('项目列表必须且只能有一个默认项目')
    return
  }

  saving.value = true
  try {
    // 先删除
    for (const id of removedProjectIds.value) {
      try {
        await merchantProjectApi.delete(id)
      } catch (e) {
        // ignore
      }
    }

    // 再新增/更新
    for (const p of form.value.projects) {
      const payload = {
        name: (p.name || '').trim(),
        duration: Number(p.duration || 0),
        bookable_online: p.bookable_online !== false,
        service_gap_minutes: Number(p.service_gap_minutes ?? 3),
        start_delay_seconds: Number(p.start_delay_seconds ?? 60),
        room_select_timeout_seconds: Math.round(Number(p.room_select_timeout_minutes ?? 1.5) * 60),
        start_pending_timeout_seconds: Number(p.start_pending_timeout_minutes ?? 5) * 60,
        service_capacity: Number(p.service_capacity ?? 1),
        service_time_slots: normalizeServiceTimeSlots(p.service_time_slots),
        auto_assign_technician_delay_minutes: Number(p.auto_assign_technician_delay_minutes ?? 5),
        delay_tolerance_minutes: Number(p.delay_tolerance_minutes ?? 1),
        delay_compensation_mode: String(p.delay_compensation_mode || 'minutes_bucket'),
        delay_redeem_threshold_percent: Number(p.delay_redeem_threshold_percent ?? 100),
        delay_fixed_unit_value: Number(p.delay_fixed_unit_value ?? 0),
        is_default: p.is_default === true
      }
      if (p.id) {
        await merchantProjectApi.update(p.id, payload)
      } else {
        await merchantProjectApi.create(payload)
      }
    }

    alert('保存成功')
    await load()
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  load()
})
</script>

<style scoped>
.service-time-input::-webkit-calendar-picker-indicator {
  opacity: 0;
}
</style>
