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
                class="w-4 h-4 mt-1 text-blue-600"
              />
              <div>
                <div class="text-gray-800 font-medium">自动叫号</div>
                <div class="text-gray-500 text-sm">自动结单后自动触发</div>
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

const form = ref({
  queue_prefix: '',
  queue_start_no: 1,
  queue_mode: 'auto'
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
    form.value = {
      queue_prefix: m.queue_prefix || '',
      queue_start_no: m.queue_start_no || 1,
      queue_mode: m.queue_mode || 'auto'
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

  saving.value = true
  try {
    await merchantApi.updateCurrentMerchantServices({
      queue_prefix: form.value.queue_prefix,
      queue_start_no: form.value.queue_start_no,
      queue_mode: form.value.queue_mode
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
