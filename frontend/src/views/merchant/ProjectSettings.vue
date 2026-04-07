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
            <div v-for="(project, index) in form.projects" :key="index" class="border border-gray-200 rounded-lg p-4 space-y-3">
              <div class="flex items-center justify-between">
                <div class="text-sm font-medium text-gray-700">项目 {{ index + 1 }}</div>
                <button 
                  @click="removeProject(index)"
                  class="text-red-500 hover:text-red-700"
                  v-if="form.projects.length > 1"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                  </svg>
                </button>
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

              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">{{ startPendingCountdownLabel }}</div>
                <input
                  v-model.number="project.start_pending_timeout_seconds"
                  type="number"
                  min="1"
                  max="3600"
                  placeholder="如 300"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
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

const startPendingCountdownLabel = computed(() => `${getStartCountdownLabel(merchantTerms.value)}（秒）`)

const normalizeProjectsState = (projects, removedIds = []) => JSON.stringify({
  projects: (projects || []).map((project) => ({
    id: project.id ?? null,
    name: String(project.name || '').trim(),
    duration: Number(project.duration || 0),
    bookable_online: project.bookable_online !== false,
    service_gap_minutes: Number(project.service_gap_minutes ?? 3),
    start_delay_seconds: Number(project.start_delay_seconds ?? 60),
    start_pending_timeout_seconds: Number(project.start_pending_timeout_seconds ?? 300),
    auto_assign_technician_delay_minutes: Number(project.auto_assign_technician_delay_minutes ?? 5),
    delay_tolerance_minutes: Number(project.delay_tolerance_minutes ?? 1),
    delay_compensation_mode: String(project.delay_compensation_mode || 'minutes_bucket'),
    delay_redeem_threshold_percent: Number(project.delay_redeem_threshold_percent ?? 100),
    delay_fixed_unit_value: Number(project.delay_fixed_unit_value ?? 0)
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
          start_pending_timeout_seconds: Number(p.start_pending_timeout_seconds ?? 300),
          auto_assign_technician_delay_minutes: Number(p.auto_assign_technician_delay_minutes ?? 5),
          delay_tolerance_minutes: Number(p.delay_tolerance_minutes ?? 1),
          delay_compensation_mode: String(p.delay_compensation_mode || 'minutes_bucket'),
          delay_redeem_threshold_percent: Number(p.delay_redeem_threshold_percent ?? 100),
          delay_fixed_unit_value: Number(p.delay_fixed_unit_value ?? 0)
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
    start_pending_timeout_seconds: 300,
    auto_assign_technician_delay_minutes: 5,
    delay_tolerance_minutes: 1,
    delay_compensation_mode: 'minutes_bucket',
    delay_redeem_threshold_percent: 100,
    delay_fixed_unit_value: 0
  })
}

const removeProject = (index) => {
  const p = form.value.projects[index]
  if (p && p.id) {
    removedProjectIds.value.push(p.id)
  }
  form.value.projects.splice(index, 1)
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
    const startPendingTimeoutSeconds = Number(project.start_pending_timeout_seconds ?? 300)
    if (!Number.isFinite(startPendingTimeoutSeconds) || startPendingTimeoutSeconds < 1 || startPendingTimeoutSeconds > 3600) {
      alert(`项目 ${i + 1} 的${getStartCountdownLabel(merchantTerms.value)}必须在 1-3600 秒之间`)
      return
    }
    const gapMinutes = Number(project.service_gap_minutes ?? 3)
    if (!Number.isFinite(gapMinutes) || gapMinutes < 0 || gapMinutes > 60) {
      alert(`项目 ${i + 1} 的服务间歇时间必须在 0-60 分钟之间`)
      return
    }
    const autoAssignDelayMinutes = Number(project.auto_assign_technician_delay_minutes ?? 5)
    if (!Number.isFinite(autoAssignDelayMinutes) || autoAssignDelayMinutes < 0 || autoAssignDelayMinutes > 180) {
      alert(`项目 ${i + 1} 的自动分配客服延迟时间必须在 0-180 分钟之间`)
      return
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
        start_pending_timeout_seconds: Number(p.start_pending_timeout_seconds ?? 300),
        auto_assign_technician_delay_minutes: Number(p.auto_assign_technician_delay_minutes ?? 5),
        delay_tolerance_minutes: Number(p.delay_tolerance_minutes ?? 1),
        delay_compensation_mode: String(p.delay_compensation_mode || 'minutes_bucket'),
        delay_redeem_threshold_percent: Number(p.delay_redeem_threshold_percent ?? 100),
        delay_fixed_unit_value: Number(p.delay_fixed_unit_value ?? 0)
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
