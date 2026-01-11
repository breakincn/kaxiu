<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">手牌设置</span>
    </header>

    <div class="px-4 py-6">
      <div v-if="loading" class="text-gray-400 text-center py-10">加载中...</div>

      <div v-else class="space-y-4">
        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">手牌编号设置</div>
          </div>
          
          <div class="px-4 py-4 space-y-4">
            <div>
              <div class="text-sm font-medium text-gray-700 mb-2">手牌前缀</div>
              <input
                v-model="form.hand_card_prefix"
                type="text"
                placeholder="如 H"
                class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            
            <div>
              <div class="text-sm font-medium text-gray-700 mb-2">起始号码</div>
              <input
                v-model.number="form.hand_card_start_no"
                type="number"
                min="1"
                class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            
            <div>
              <div class="text-sm font-medium text-gray-700 mb-2">结束号码</div>
              <input
                v-model.number="form.hand_card_end_no"
                type="number"
                min="1"
                class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm p-4">
          <div class="text-sm text-gray-600">
            <div class="mb-2">预览：</div>
            <div class="font-mono text-lg">
              {{ form.hand_card_prefix || 'H' }}{{ form.hand_card_start_no || 1 }} - 
              {{ form.hand_card_prefix || 'H' }}{{ form.hand_card_end_no || 100 }}
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
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

const router = useRouter()

const loading = ref(true)
const saving = ref(false)

const form = ref({
  hand_card_prefix: 'H',
  hand_card_start_no: 1,
  hand_card_end_no: 100
})

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
    setTimeout(() => {
      if (router.currentRoute.value.path === '/merchant/hand-card-settings') {
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
      hand_card_prefix: m.hand_card_prefix || 'H',
      hand_card_start_no: m.hand_card_start_no || 1,
      hand_card_end_no: m.hand_card_end_no || 100
    }
  } catch (e) {
    console.error('加载手牌设置失败', e)
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (saving.value) return
  
  if (!form.value.hand_card_start_no || form.value.hand_card_start_no < 1) {
    alert('起始号码必须大于等于1')
    return
  }
  
  if (!form.value.hand_card_end_no || form.value.hand_card_end_no < 1) {
    alert('结束号码必须大于等于1')
    return
  }
  
  if (form.value.hand_card_end_no < form.value.hand_card_start_no) {
    alert('结束号码必须大于等于起始号码')
    return
  }

  saving.value = true
  try {
    await merchantApi.updateCurrentMerchantServices({
      hand_card_prefix: form.value.hand_card_prefix,
      hand_card_start_no: form.value.hand_card_start_no,
      hand_card_end_no: form.value.hand_card_end_no
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
