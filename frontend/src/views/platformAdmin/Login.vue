<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center px-4">
    <div class="w-full max-w-md">
      <div class="text-center mb-8">
        <div class="flex items-center justify-center gap-2 mb-2">
          <span class="text-primary font-bold text-3xl">卡包</span>
          <span class="text-gray-400 text-sm">平台后台</span>
        </div>
        <p class="text-gray-500">请输入平台管理员 Token</p>
      </div>

      <div class="bg-white rounded-2xl p-6 shadow-lg">
        <form @submit.prevent="submit">
          <div class="mb-6">
            <label class="block text-gray-700 text-sm font-medium mb-2">PLATFORM_ADMIN_TOKEN</label>
            <input
              v-model="token"
              type="password"
              placeholder="请输入 Token"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
            />
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ loading ? '验证中...' : '进入平台后台' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { platformAdminApi } from '../../api'

const router = useRouter()
const token = ref(localStorage.getItem('platformAdminToken') || '')
const loading = ref(false)

const submit = async () => {
  if (!token.value) return
  loading.value = true
  try {
    localStorage.setItem('platformAdminToken', token.value)
    await platformAdminApi.listServiceRoles()
    router.replace('/platform-admin')
  } catch (e) {
    localStorage.removeItem('platformAdminToken')
    alert(e.response?.data?.error || 'Token 无效')
  } finally {
    loading.value = false
  }
}
</script>
