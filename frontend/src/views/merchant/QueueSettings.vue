<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">叫号设置</span>
    </header>

    <div class="px-4 py-6">
      <div v-if="loading" class="text-gray-400 text-center py-10">加载中...</div>

      <div v-else class="space-y-4">
        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">叫号前缀</div>
          </div>
          <div class="px-4 py-4">
            <input
              v-model="form.queue_prefix"
              type="text"
              placeholder="如 A"
              class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">起始号码</div>
          </div>
          <div class="px-4 py-4">
            <input
              v-model.number="form.queue_start_no"
              type="number"
              min="1"
              class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">叫号方式</div>
          </div>
          <div class="px-4 py-4 space-y-3">
            <label class="flex items-start gap-3">
              <input
                type="radio"
                v-model="form.queue_mode"
                value="auto"
                :disabled="!supportOrderComplete"
                class="w-4 h-4 mt-1 text-blue-600"
              />
              <div>
                <div class="text-gray-800 font-medium">自动叫号</div>
                <div class="text-gray-500 text-sm">自动结单后自动触发</div>
                <div v-if="!supportOrderComplete" class="text-gray-500 text-sm">需先开启结单服务</div>
              </div>
            </label>
            <label class="flex items-start gap-3">
              <input
                type="radio"
                v-model="form.queue_mode"
                value="manual"
                class="w-4 h-4 mt-1 text-blue-600"
              />
              <div>
                <div class="text-gray-800 font-medium">人工叫号</div>
                <div class="text-gray-500 text-sm">服务结束后人工手动触发</div>
              </div>
            </label>

            <!-- 开启多个客服 -->
            <div class="mt-4 pt-4 border-t border-gray-100">
              <label class="flex items-center gap-3">
                <input
                  type="checkbox"
                  v-model="form.support_multi_customer_service"
                  class="w-4 h-4 text-blue-600 rounded"
                />
                <div>
                  <div class="text-gray-800 font-medium">开启多个客服</div>
                  <div class="text-gray-500 text-sm">允许同时叫多个号，每个技师按完成顺序自动领取下一号</div>
                </div>
              </label>
            </div>

            <!-- 自定义窗口名词：仅开启多个客服时显示 -->
            <div v-if="form.support_multi_customer_service" class="mt-4 pt-4 border-t border-gray-100">
              <div class="text-gray-700 text-sm font-medium mb-2">自定义叫号窗口名词</div>
              <input
                v-model="form.queue_window_term"
                type="text"
                placeholder="默认：窗口，可自定义如：台号、工位等"
                class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <div class="text-gray-500 text-xs mt-2">在“设置客服”中为每个专业客服设置服务窗口时，使用此名词</div>
            </div>

            <div v-if="form.support_multi_customer_service" class="mt-4 pt-4 border-t border-gray-100">
              <div class="text-gray-700 text-sm font-medium mb-2">等待上号时间</div>
              <div class="relative">
                <input
                  v-model.number="form.queue_waiting_start_seconds"
                  type="number"
                  min="1"
                  max="3600"
                  class="w-full px-4 py-3 pr-12 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">秒</span>
              </div>
              <div class="text-gray-500 text-xs mt-2">叫号后进入待上号状态的倒计时时间，默认 180 秒</div>
            </div>

            <div v-if="form.support_multi_customer_service" class="mt-4 pt-4 border-t border-gray-100">
              <div class="text-gray-700 text-sm font-medium mb-2">超时过号等待时间</div>
              <div class="relative">
                <input
                  v-model.number="form.queue_timeout_waiting_minutes"
                  type="number"
                  min="1"
                  max="1440"
                  class="w-full px-4 py-3 pr-14 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">分钟</span>
              </div>
              <div class="text-gray-500 text-xs mt-2">用户超时过号后，保留等待重新上号的时长，默认 15 分钟</div>
            </div>
          </div>
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
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

const router = useRouter()

const loading = ref(true)
const saving = ref(false)

const supportOrderComplete = ref(false)

const form = ref({
  queue_prefix: '',
  queue_start_no: 1,
  queue_mode: 'auto',
  queue_window_term: '窗口',
  queue_waiting_start_seconds: 180,
  queue_timeout_waiting_minutes: 15,
  support_multi_customer_service: false
})

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
    setTimeout(() => {
      if (router.currentRoute.value.path === '/merchant/queue-settings') {
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
    supportOrderComplete.value = !!m.support_order_complete
    form.value = {
      queue_prefix: m.queue_prefix || '',
      queue_start_no: m.queue_start_no || 1,
      queue_mode: m.queue_mode || 'auto',
      queue_window_term: m.queue_window_term || '窗口',
      queue_waiting_start_seconds: m.queue_waiting_start_seconds || 180,
      queue_timeout_waiting_minutes: Math.max(1, Math.floor(Number(m.queue_timeout_waiting_seconds || 900) / 60)),
      support_multi_customer_service: !!m.support_multi_customer_service
    }

    if (!supportOrderComplete.value && form.value.queue_mode === 'auto') {
      form.value.queue_mode = 'manual'
    }
  } catch (e) {
    alert(e?.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (saving.value) return

  if (!form.value.queue_start_no || form.value.queue_start_no < 1) {
    alert('起始号码必须大于等于1')
    return
  }

  const mode = String(form.value.queue_mode || '').trim()
  if (mode !== 'auto' && mode !== 'manual') {
    alert('叫号方式无效')
    return
  }

  if (mode === 'auto' && !supportOrderComplete.value) {
    alert('先开启结单服务，才能开启自动叫号')
    form.value.queue_mode = 'manual'
    return
  }

  if (!form.value.queue_waiting_start_seconds || form.value.queue_waiting_start_seconds < 1 || form.value.queue_waiting_start_seconds > 3600) {
    alert('等待上号时间范围应为1-3600秒')
    return
  }

  if (!form.value.queue_timeout_waiting_minutes || form.value.queue_timeout_waiting_minutes < 1) {
    alert('超时过号等待时间必须大于0分钟')
    return
  }

  saving.value = true
  try {
    await merchantApi.updateCurrentMerchantServices({
      queue_prefix: form.value.queue_prefix,
      queue_start_no: form.value.queue_start_no,
      queue_mode: form.value.queue_mode,
      queue_window_term: form.value.queue_window_term,
      queue_waiting_start_seconds: form.value.queue_waiting_start_seconds,
      queue_timeout_waiting_seconds: Number(form.value.queue_timeout_waiting_minutes) * 60,
      support_multi_customer_service: !!form.value.support_multi_customer_service
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
