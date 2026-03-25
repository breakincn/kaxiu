<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">开启服务</span>
    </header>

    <div class="px-4 py-6">
      <div v-if="loading" class="text-gray-400 text-center py-10">加载中...</div>

      <div v-else class="space-y-4">
        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100 flex items-center justify-between">
            <div class="text-gray-800 font-medium">开启预约</div>
            <input type="checkbox" v-model="form.support_appointment" />
          </div>

          <div class="px-4 py-4 border-b border-gray-100 flex items-center justify-between">
            <div class="text-gray-800 font-medium">开启项目</div>
            <input type="checkbox" v-model="form.support_project" />
          </div>

		  <div class="px-4 py-4 border-b border-gray-100 flex items-center justify-between">
			<div class="text-gray-800 font-medium">开启客服</div>
			<input type="checkbox" v-model="form.support_customer_service" @change="onCustomerServiceChange" />
		  </div>

          <div class="px-4 py-4 border-b border-gray-100 flex items-center justify-between">
            <div class="text-gray-800 font-medium">客服模式</div>
            <input type="checkbox" v-model="form.support_customer_service_mode" :disabled="!form.support_customer_service" @change="onCustomerServiceModeChange" />
          </div>

          <div class="px-4 py-4 border-b border-gray-100 flex items-center justify-between">
            <div class="text-gray-800 font-medium">叫号模式</div>
            <input type="checkbox" v-model="form.support_queue" @change="onQueueModeChange" />
          </div>

          <div class="px-4 py-4 border-b border-gray-100 flex items-center justify-between">
            <div class="text-gray-800 font-medium">{{ replaceTerms('开启结单', merchantTerms) }}</div>
            <input type="checkbox" v-model="form.support_order_complete" :disabled="form.support_queue" />
          </div>

          <div class="px-4 py-4 border-b border-gray-100 flex items-center justify-between">
            <div class="text-gray-800 font-medium">开启手牌</div>
            <input type="checkbox" v-model="form.support_hand_card" />
          </div>

          <div class="px-4 py-4 flex items-center justify-between">
            <div class="text-gray-800 font-medium">开启房间</div>
            <input type="checkbox" v-model="form.support_room" :disabled="form.support_queue" />
          </div>

          <div class="px-4 py-4 border-b border-gray-100 flex items-center justify-between">
            <div class="text-gray-800 font-medium">开启直购售卡</div>
            <input type="checkbox" v-model="form.support_direct_sale" />
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
import { merchantApi } from '../../api'
import { replaceTerms } from '../../utils/terms'

const router = useRouter()

const loading = ref(true)
const saving = ref(false)
const initialSnapshot = ref('')
const lastSavedForm = ref(null)
const merchantTerms = ref(null)

const merchantBizTime = ref({
  all_day_start: '',
  all_day_end: '',
  morning_start: '',
  morning_end: '',
  afternoon_start: '',
  afternoon_end: '',
  evening_start: '',
  evening_end: ''
})

const hasBusinessTimeConfigured = () => {
  const m = merchantBizTime.value || {}
  const allDayOk = String(m.all_day_start || '').trim() !== '' && String(m.all_day_end || '').trim() !== ''
  if (allDayOk) return true
  const morningOk = String(m.morning_start || '').trim() !== '' && String(m.morning_end || '').trim() !== ''
  const afternoonOk = String(m.afternoon_start || '').trim() !== '' && String(m.afternoon_end || '').trim() !== ''
  const eveningOk = String(m.evening_start || '').trim() !== '' && String(m.evening_end || '').trim() !== ''
  return morningOk || afternoonOk || eveningOk
}

const form = ref({
  support_appointment: false,
  support_queue: false,
  support_room: false,
  support_direct_sale: false,
  support_customer_service: false,
  support_customer_service_mode: false,
  support_project: false,
  support_order_complete: false,
  support_hand_card: false,
  appointment_reserve_buffer_minutes: 10,
  appointment_grace_window_minutes: 15,
  appointment_prediction_buffer_minutes: 5
})

const normalizeForm = (value) => JSON.stringify({
  support_appointment: !!value.support_appointment,
  support_queue: !!value.support_queue,
  support_room: !!value.support_room,
  support_direct_sale: !!value.support_direct_sale,
  support_customer_service: !!value.support_customer_service,
  support_customer_service_mode: !!value.support_customer_service_mode,
  support_project: !!value.support_project,
  support_order_complete: !!value.support_order_complete,
  support_hand_card: !!value.support_hand_card,
  appointment_reserve_buffer_minutes: Number(value.appointment_reserve_buffer_minutes ?? 10),
  appointment_grace_window_minutes: Number(value.appointment_grace_window_minutes ?? 15),
  appointment_prediction_buffer_minutes: Number(value.appointment_prediction_buffer_minutes ?? 5)
})

const createFormState = (value = {}) => ({
  support_appointment: !!value.support_appointment,
  support_queue: !!value.support_queue,
  support_room: !!value.support_room,
  support_direct_sale: !!value.support_direct_sale,
  support_customer_service: !!value.support_customer_service,
  support_customer_service_mode: !!value.support_customer_service_mode,
  support_project: !!value.support_project,
  support_order_complete: !!value.support_order_complete,
  support_hand_card: !!value.support_hand_card,
  appointment_reserve_buffer_minutes: Number(value.appointment_reserve_buffer_minutes ?? 10),
  appointment_grace_window_minutes: Number(value.appointment_grace_window_minutes ?? 15),
  appointment_prediction_buffer_minutes: Number(value.appointment_prediction_buffer_minutes ?? 5)
})

const restoreLastSavedForm = () => {
  if (!lastSavedForm.value) return
  form.value = createFormState(lastSavedForm.value)
}

const isRuntimeSwitchBlockedError = (message) =>
  typeof message === 'string' &&
  (
    message.includes('当前有未完成服务会话') ||
    message.includes('当前有进行中的叫号服务') ||
    message.includes('请等待本轮服务全部完成后再切换')
  )

const isDirty = computed(() => normalizeForm(form.value) !== initialSnapshot.value)

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
    setTimeout(() => {
      if (router.currentRoute.value.path === '/merchant/services') {
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
    const m = res.data.data || {}
    merchantTerms.value = m

    merchantBizTime.value = {
      all_day_start: m.all_day_start || '',
      all_day_end: m.all_day_end || '',
      morning_start: m.morning_start || '',
      morning_end: m.morning_end || '',
      afternoon_start: m.afternoon_start || '',
      afternoon_end: m.afternoon_end || '',
      evening_start: m.evening_start || '',
      evening_end: m.evening_end || ''
    }

    form.value = createFormState(m)
    lastSavedForm.value = createFormState(m)
    initialSnapshot.value = normalizeForm(form.value)
  } catch (e) {
    console.error('加载商户服务配置失败', e)
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const onCustomerServiceChange = () => {
  // 关闭客服时，自动关闭客服模式
  if (!form.value.support_customer_service) {
    form.value.support_customer_service_mode = false
  }
}

const onCustomerServiceModeChange = () => {
  // 开启客服模式时，自动关闭叫号模式（互斥）
  if (form.value.support_customer_service_mode) {
    form.value.support_queue = false
  }
}

const onQueueModeChange = () => {
  // 开启叫号模式时，自动关闭客服模式（互斥）
  if (form.value.support_queue) {
    form.value.support_customer_service_mode = false
  }
}

const save = async () => {
  if (saving.value) return

  if (form.value.support_appointment && !hasBusinessTimeConfigured()) {
    alert('先设置营业时间，才能开启预约服务')
    form.value.support_appointment = false
    return
  }

  // 客服模式需要先开启客服
  if (form.value.support_customer_service_mode && !form.value.support_customer_service) {
    alert('开启客服模式前，请先开启“开启客服”')
    form.value.support_customer_service_mode = false
    return
  }

  saving.value = true
  try {
    await merchantApi.updateCurrentMerchantServices({
      support_appointment: form.value.support_appointment,
      support_queue: form.value.support_queue,
      support_room: form.value.support_room,
      support_direct_sale: form.value.support_direct_sale,
      support_customer_service: form.value.support_customer_service,
      support_customer_service_mode: form.value.support_customer_service_mode,
      support_project: form.value.support_project,
      support_order_complete: form.value.support_order_complete,
      support_hand_card: form.value.support_hand_card,
      appointment_reserve_buffer_minutes: Number(form.value.appointment_reserve_buffer_minutes || 0),
      appointment_grace_window_minutes: Number(form.value.appointment_grace_window_minutes || 0),
      appointment_prediction_buffer_minutes: Number(form.value.appointment_prediction_buffer_minutes || 0)
    })
    alert('保存成功')
    await load()
  } catch (e) {
    const errorMessage = e.response?.data?.error || '保存失败'
    if (isRuntimeSwitchBlockedError(errorMessage)) {
      restoreLastSavedForm()
    }
    alert(errorMessage)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  load()
})
</script>
