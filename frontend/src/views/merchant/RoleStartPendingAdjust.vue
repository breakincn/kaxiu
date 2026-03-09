<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">参数设置</span>
    </header>

    <div class="px-4 py-4">
      <div class="bg-white rounded-xl shadow-sm p-4">
        <div class="text-gray-800 font-medium">{{ role?.name || '-' }}</div>
      </div>

      <div class="bg-white rounded-xl shadow-sm p-4 mt-4">
        <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>
        <div v-else>
          <label class="block text-gray-800 text-sm font-medium mb-2">{{ label }}</label>
          <div class="relative">
            <input
              v-model.number="seconds"
              type="number"
              min="1"
              max="3600"
              class="w-full px-4 py-3 pr-12 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
            <span class="absolute inset-y-0 right-4 flex items-center text-sm text-gray-500">秒</span>
          </div>
          <div class="text-gray-500 text-xs mt-2">默认 300 秒，支持按岗位单独配置。</div>
        </div>
      </div>

      <button
        type="button"
        @click="save"
        :disabled="loading || saving || !isDirty"
        class="w-full mt-6 py-3 bg-primary text-white rounded-lg font-medium hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {{ saving ? '保存中...' : '保存' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { merchantApi } from '../../api'
import { replaceTerms } from '../../utils/terms'

const router = useRouter()
const route = useRoute()
const startTerm = String(route.query.start_term || '').trim()
const termMerchant = startTerm ? { start_term: startTerm } : null

const roleKey = ref(String(route.params.roleKey || ''))
const loading = ref(false)
const saving = ref(false)
const initialSeconds = ref(null)
const role = ref(route.query.role_name ? { name: String(route.query.role_name) } : null)
const seconds = ref(300)
const label = ref(replaceTerms('待起单超时秒数', termMerchant))

const isDirty = computed(() => Number(seconds.value || 0) !== Number(initialSeconds.value || 0))

const goBack = () => {
  router.back()
}

const load = async () => {
  loading.value = true
  try {
    const res = await merchantApi.getRoleStartPendingSetting(roleKey.value)
    const data = res.data?.data || {}
    role.value = data.role || role.value
    seconds.value = Number(data.start_pending_timeout_seconds || 300)
    initialSeconds.value = Number(data.start_pending_timeout_seconds || 300)
    label.value = String(data.start_pending_timeout_label || replaceTerms('待起单超时秒数', termMerchant))
  } catch (e) {
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (saving.value) return
  const n = Number(seconds.value || 0)
  if (!Number.isFinite(n) || n < 1 || n > 3600) {
    alert(`${label.value}范围应为1-3600秒`)
    return
  }
  saving.value = true
  try {
    await merchantApi.setRoleStartPendingSetting(roleKey.value, Math.floor(n))
    alert('保存成功')
    await load()
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
