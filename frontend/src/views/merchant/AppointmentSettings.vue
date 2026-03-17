<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">预约设置</span>
    </header>

    <div class="px-4 py-6">
      <div v-if="loading" class="text-gray-400 text-center py-10">加载中...</div>

      <div v-else class="space-y-4">
        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">预约前保留缓冲</div>
            <div class="text-gray-500 text-sm mt-1">预约开始前，预留给预约客户的客服时间</div>
          </div>
          <div class="px-4 py-4">
            <div class="relative">
              <input v-model.number="form.appointment_reserve_buffer_minutes" type="number" min="0" max="120" class="w-full px-4 py-3 pr-14 border border-gray-300 rounded-lg" />
              <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
            </div>
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">预约后宽限时间</div>
            <div class="text-gray-500 text-sm mt-1">客户在预约开始后仍可到店签到的宽限时长</div>
          </div>
          <div class="px-4 py-4">
            <div class="relative">
              <input v-model.number="form.appointment_grace_window_minutes" type="number" min="0" max="180" class="w-full px-4 py-3 pr-14 border border-gray-300 rounded-lg" />
              <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
            </div>
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">预约最大等待</div>
            <div class="text-gray-500 text-sm mt-1">系统允许预约客户因前序服务而产生的最大等待时间</div>
          </div>
          <div class="px-4 py-4">
            <div class="relative">
              <input v-model.number="form.appointment_max_wait_minutes" type="number" min="0" max="180" class="w-full px-4 py-3 pr-14 border border-gray-300 rounded-lg" />
              <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
            </div>
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">预约预测缓冲</div>
            <div class="text-gray-500 text-sm mt-1">用于估算服务可能延长的风险缓冲</div>
          </div>
          <div class="px-4 py-4">
            <div class="relative">
              <input v-model.number="form.appointment_prediction_buffer_minutes" type="number" min="0" max="60" class="w-full px-4 py-3 pr-14 border border-gray-300 rounded-lg" />
              <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
            </div>
          </div>
        </div>

        <div class="text-xs text-gray-400 px-1">
          这些参数会同时影响预约页可选客服、客服模式现场派单、改签可用时段和预约到店等待判断。
        </div>

        <button
          @click="save"
          :disabled="saving"
          class="w-full bg-primary text-white py-3 rounded-lg hover:bg-primary-dark font-medium disabled:bg-gray-300 disabled:cursor-not-allowed"
        >
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

const router = useRouter()
const loading = ref(true)
const saving = ref(false)

const form = ref({
  appointment_reserve_buffer_minutes: 10,
  appointment_grace_window_minutes: 15,
  appointment_max_wait_minutes: 15,
  appointment_prediction_buffer_minutes: 5
})

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
      appointment_prediction_buffer_minutes: Number(m.appointment_prediction_buffer_minutes ?? 5)
    }
  } catch (e) {
    alert(e?.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (saving.value) return
  saving.value = true
  try {
    await merchantApi.updateCurrentMerchantServices({
      appointment_reserve_buffer_minutes: Number(form.value.appointment_reserve_buffer_minutes || 0),
      appointment_grace_window_minutes: Number(form.value.appointment_grace_window_minutes || 0),
      appointment_max_wait_minutes: Number(form.value.appointment_max_wait_minutes || 0),
      appointment_prediction_buffer_minutes: Number(form.value.appointment_prediction_buffer_minutes || 0)
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
