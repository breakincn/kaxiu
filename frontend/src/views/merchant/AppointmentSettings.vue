<template>
  <div class="min-h-screen bg-gray-50 pb-6">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">预约设置</span>
    </header>

    <div class="px-4 mt-4">
      <div v-if="loading" class="text-gray-400 text-center py-10">加载中...</div>

      <div v-else class="space-y-4">
        <div class="bg-white rounded-xl p-4 shadow-sm">
          <div class="flex items-center gap-2 mb-4">
            <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10m-11 9h12a2 2 0 002-2V7a2 2 0 00-2-2H6a2 2 0 00-2 2v11a2 2 0 002 2z"/>
            </svg>
            <span class="font-medium text-gray-800">预约保护参数</span>
          </div>

          <div class="space-y-4">
            <div>
              <label class="text-sm font-medium text-gray-700 mb-2 block">预约前保留缓冲</label>
              <p class="text-xs text-gray-500 mb-2">预约开始前，预留给预约客户的客服时间</p>
              <div class="relative">
                <input
                  v-model.number="form.appointment_reserve_buffer_minutes"
                  type="number"
                  min="0"
                  max="120"
                  class="w-full px-3 py-2 pr-14 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
              </div>
            </div>

            <div>
              <label class="text-sm font-medium text-gray-700 mb-2 block">预约后宽限时间</label>
              <p class="text-xs text-gray-500 mb-2">客户在预约开始后仍可到店签到的宽限时长</p>
              <div class="relative">
                <input
                  v-model.number="form.appointment_grace_window_minutes"
                  type="number"
                  min="0"
                  max="180"
                  class="w-full px-3 py-2 pr-14 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
              </div>
            </div>

            <div>
              <label class="text-sm font-medium text-gray-700 mb-2 block">预约最大等待</label>
              <p class="text-xs text-gray-500 mb-2">系统允许预约客户因前序服务而产生的最大等待时间</p>
              <div class="relative">
                <input
                  v-model.number="form.appointment_max_wait_minutes"
                  type="number"
                  min="0"
                  max="180"
                  class="w-full px-3 py-2 pr-14 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
              </div>
            </div>

            <div>
              <label class="text-sm font-medium text-gray-700 mb-2 block">预约预测缓冲</label>
              <p class="text-xs text-gray-500 mb-2">用于估算服务可能延长的风险缓冲</p>
              <div class="relative">
                <input
                  v-model.number="form.appointment_prediction_buffer_minutes"
                  type="number"
                  min="0"
                  max="60"
                  class="w-full px-3 py-2 pr-14 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
              </div>
            </div>

            <div>
              <label class="text-sm font-medium text-gray-700 mb-2 block">预约时段展示粒度</label>
              <p class="text-xs text-gray-500 mb-2">用户预约时展示的时间网格，建议使用 15 分钟</p>
              <div class="relative">
                <input
                  v-model.number="form.appointment_slot_granularity_minutes"
                  type="number"
                  min="5"
                  max="60"
                  step="5"
                  class="w-full px-3 py-2 pr-14 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
              </div>
            </div>

            <div>
              <label class="text-sm font-medium text-gray-700 mb-2 block">昨天预约可改签到今天/明天阈值</label>
              <p class="text-xs text-gray-500 mb-2">仅影响昨天预约在今天的补救改签；超过该阈值时可改到今天或明天</p>
              <div class="relative">
                <input
                  v-model.number="form.appointment_reschedule_same_or_next_day_threshold_minutes"
                  type="number"
                  min="0"
                  max="1440"
                  class="w-full px-3 py-2 pr-14 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
              </div>
            </div>

            <div>
              <label class="text-sm font-medium text-gray-700 mb-2 block">昨天预约仅可改签到明天阈值</label>
              <p class="text-xs text-gray-500 mb-2">仅影响昨天预约在今天的补救改签；小于等于该阈值时禁止改签</p>
              <div class="relative">
                <input
                  v-model.number="form.appointment_reschedule_next_day_only_threshold_minutes"
                  type="number"
                  min="0"
                  max="1440"
                  class="w-full px-3 py-2 pr-14 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
              </div>
            </div>

            <div>
              <label class="text-sm font-medium text-gray-700 mb-2 block">系统优化性改签推荐</label>
              <p class="text-xs text-gray-500 mb-2">默认关闭。开启后，用户侧仅展示推荐时段与推荐原因，不会自动替用户改签。</p>
              <label class="flex items-center gap-3 px-3 py-3 border border-gray-200 rounded-lg">
                <input
                  v-model="form.appointment_reschedule_recommendation_enabled"
                  type="checkbox"
                  class="h-4 w-4"
                />
                <span class="text-sm text-gray-700">启用推荐卡片</span>
              </label>
            </div>
          </div>
        </div>

        <div class="bg-white rounded-xl p-4 shadow-sm">
          <div class="flex items-center gap-2 mb-4">
            <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <span class="font-medium text-gray-800">参数说明</span>
          </div>
          <p class="text-sm leading-6 text-gray-500">
            这些参数会同时影响预约页可选客服、客服模式现场派单、改签可用时段和预约到店等待判断。
          </p>
        </div>

        <button
          @click="save"
          :disabled="loading || saving || !isDirty"
          class="w-full mt-6 bg-primary text-white py-3 rounded-lg hover:bg-primary-dark font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
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
import { merchantApi } from '../../api'

const router = useRouter()
const loading = ref(true)
const saving = ref(false)
const initialSnapshot = ref('')

const form = ref({
  appointment_reserve_buffer_minutes: 10,
  appointment_grace_window_minutes: 15,
  appointment_max_wait_minutes: 15,
  appointment_prediction_buffer_minutes: 5,
  appointment_slot_granularity_minutes: 15,
  appointment_reschedule_same_or_next_day_threshold_minutes: 180,
  appointment_reschedule_next_day_only_threshold_minutes: 90,
  appointment_reschedule_recommendation_enabled: false
})

// 保存按钮与其他设置页保持一致，只有表单发生实际变化时才允许提交。
const buildSnapshot = () => JSON.stringify({
  appointment_reserve_buffer_minutes: Number(form.value.appointment_reserve_buffer_minutes ?? 0),
  appointment_grace_window_minutes: Number(form.value.appointment_grace_window_minutes ?? 0),
  appointment_max_wait_minutes: Number(form.value.appointment_max_wait_minutes ?? 0),
  appointment_prediction_buffer_minutes: Number(form.value.appointment_prediction_buffer_minutes ?? 0),
  appointment_slot_granularity_minutes: Number(form.value.appointment_slot_granularity_minutes ?? 0),
  appointment_reschedule_same_or_next_day_threshold_minutes: Number(form.value.appointment_reschedule_same_or_next_day_threshold_minutes ?? 0),
  appointment_reschedule_next_day_only_threshold_minutes: Number(form.value.appointment_reschedule_next_day_only_threshold_minutes ?? 0),
  appointment_reschedule_recommendation_enabled: !!form.value.appointment_reschedule_recommendation_enabled
})

const isDirty = computed(() => buildSnapshot() !== initialSnapshot.value)

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
    setTimeout(() => {
      if (router.currentRoute.value.path === '/merchant/appointment-settings') {
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
    const res = await merchantApi.getCurrentMerchant()
    const m = res.data?.data || {}
    if (!m.support_appointment) {
      alert('请先在“开启服务”中开启预约')
      router.replace('/merchant/services')
      return
    }
    form.value = {
      appointment_reserve_buffer_minutes: Number(m.appointment_reserve_buffer_minutes ?? 10),
      appointment_grace_window_minutes: Number(m.appointment_grace_window_minutes ?? 15),
      appointment_max_wait_minutes: Number(m.appointment_max_wait_minutes ?? 15),
      appointment_prediction_buffer_minutes: Number(m.appointment_prediction_buffer_minutes ?? 5),
      appointment_slot_granularity_minutes: Number(m.appointment_slot_granularity_minutes ?? 15),
      appointment_reschedule_same_or_next_day_threshold_minutes: Number(m.appointment_reschedule_same_or_next_day_threshold_minutes ?? 180),
      appointment_reschedule_next_day_only_threshold_minutes: Number(m.appointment_reschedule_next_day_only_threshold_minutes ?? 90),
      appointment_reschedule_recommendation_enabled: !!m.appointment_reschedule_recommendation_enabled
    }
    initialSnapshot.value = buildSnapshot()
  } catch (e) {
    alert(e?.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (loading.value || saving.value || !isDirty.value) return
  saving.value = true
  try {
    await merchantApi.updateCurrentMerchantServices({
      appointment_reserve_buffer_minutes: Number(form.value.appointment_reserve_buffer_minutes || 0),
      appointment_grace_window_minutes: Number(form.value.appointment_grace_window_minutes || 0),
      appointment_max_wait_minutes: Number(form.value.appointment_max_wait_minutes || 0),
      appointment_prediction_buffer_minutes: Number(form.value.appointment_prediction_buffer_minutes || 0),
      appointment_slot_granularity_minutes: Number(form.value.appointment_slot_granularity_minutes || 0),
      appointment_reschedule_same_or_next_day_threshold_minutes: Number(form.value.appointment_reschedule_same_or_next_day_threshold_minutes || 0),
      appointment_reschedule_next_day_only_threshold_minutes: Number(form.value.appointment_reschedule_next_day_only_threshold_minutes || 0),
      appointment_reschedule_recommendation_enabled: !!form.value.appointment_reschedule_recommendation_enabled
    })
    alert('保存成功')
    await load()
  } catch (e) {
    alert(e?.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  load()
})
</script>
