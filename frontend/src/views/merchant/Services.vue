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
            <div class="text-gray-800 font-medium">开启叫号</div>
            <input type="checkbox" v-model="form.support_queue" />
          </div>

          <div v-if="form.support_queue" class="px-4 py-4 border-b border-gray-100 space-y-3">
            <div>
              <div class="text-sm font-medium text-gray-700 mb-2">叫号前缀</div>
              <input
                v-model="form.queue_prefix"
                type="text"
                placeholder="如 A"
                class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <div class="text-sm font-medium text-gray-700 mb-2">起始号码</div>
              <input
                v-model.number="form.queue_start_no"
                type="number"
                min="1"
                class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          <div class="px-4 py-4 flex items-center justify-between">
            <div class="text-gray-800 font-medium">开启直购售卡</div>
            <input type="checkbox" v-model="form.support_direct_sale" />
          </div>

		  <div v-if="form.support_direct_sale" class="px-4 py-4 flex items-center justify-between border-t border-gray-100">
			<div class="text-gray-800 font-medium">设置客服</div>
			<input type="checkbox" v-model="form.support_customer_service" />
		  </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm p-4">
          <div class="text-sm font-medium text-gray-700 mb-2">平均服务时长（分钟）</div>
          <input
            v-model.number="form.avg_service_minutes"
            type="number"
            min="1"
            class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        <button
          @click="save"
          :disabled="saving"
          class="w-full bg-blue-500 text-white py-3 rounded-lg hover:bg-blue-600 font-medium disabled:bg-gray-300 disabled:cursor-not-allowed"
        >
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

const router = useRouter()

const loading = ref(true)
const saving = ref(false)

const form = ref({
  support_appointment: false,
  support_queue: false,
  queue_prefix: '',
  queue_start_no: 1,
  support_direct_sale: false,
  support_customer_service: false,
  avg_service_minutes: 30
})

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
    form.value = {
      support_appointment: !!m.support_appointment,
      support_queue: !!m.support_queue,
      queue_prefix: m.queue_prefix || '',
      queue_start_no: m.queue_start_no || 1,
      support_direct_sale: !!m.support_direct_sale,
      support_customer_service: !!m.support_customer_service,
      avg_service_minutes: m.avg_service_minutes || 30
    }
  } catch (e) {
    console.error('加载商户服务配置失败', e)
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (saving.value) return
  if (form.value.support_queue && (!form.value.queue_start_no || form.value.queue_start_no < 1)) {
    alert('叫号起始号码必须大于等于1')
    return
  }
  if (!form.value.avg_service_minutes || form.value.avg_service_minutes < 1) {
    alert('平均服务时长必须大于等于1')
    return
  }

  saving.value = true
  try {
    await merchantApi.updateCurrentMerchantServices({
      support_appointment: form.value.support_appointment,
      support_queue: form.value.support_queue,
      queue_prefix: form.value.queue_prefix,
      queue_start_no: form.value.queue_start_no,
      support_direct_sale: form.value.support_direct_sale,
      support_customer_service: form.value.support_customer_service,
      avg_service_minutes: form.value.avg_service_minutes
    })
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
