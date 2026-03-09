<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">修改密码</span>
    </header>

    <div class="px-4 py-6">
      <div class="bg-white rounded-xl shadow-sm p-6">
        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">原密码</label>
            <input
              v-model="form.oldPassword"
              type="password"
              placeholder="请输入原密码"
              class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">新密码</label>
            <input
              v-model="form.newPassword"
              type="password"
              placeholder="请输入新密码"
              class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">再次确认新密码</label>
            <input
              v-model="form.confirmPassword"
              type="password"
              placeholder="请再次输入新密码"
              class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent"
            />
          </div>

          <button
            type="submit"
            :disabled="saving"
            class="w-full bg-primary text-white py-3 rounded-lg font-medium disabled:opacity-50"
          >
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'
import { getMerchantActiveAuth, setTechnicianPasswordNeedReset } from '../../utils/auth'

const router = useRouter()
const saving = ref(false)
const form = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const goBack = () => {
  router.back()
}

const submit = async () => {
  if (getMerchantActiveAuth() !== 'staff') {
    alert('仅客服账号可修改密码')
    return
  }
  if (!form.value.oldPassword) {
    alert('请输入原密码')
    return
  }
  if (!form.value.newPassword) {
    alert('请输入新密码')
    return
  }
  if (form.value.newPassword.length < 8) {
    alert('新密码至少8位')
    return
  }
  if (!form.value.confirmPassword) {
    alert('请再次确认新密码')
    return
  }
  if (form.value.newPassword !== form.value.confirmPassword) {
    alert('两次输入的新密码不一致')
    return
  }

  saving.value = true
  try {
    await merchantApi.changeTechnicianPassword(form.value.oldPassword, form.value.newPassword)
    setTechnicianPasswordNeedReset(false)
    alert('密码修改成功')
    router.replace('/merchant/settings')
  } catch (e) {
    alert(e.response?.data?.error || '修改密码失败')
  } finally {
    saving.value = false
  }
}
</script>
