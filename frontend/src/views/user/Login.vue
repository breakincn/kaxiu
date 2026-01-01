<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center px-4">
    <div class="w-full max-w-md">
      <div class="text-center mb-8">
        <div class="flex items-center justify-center gap-2 mb-2">
          <span class="text-primary font-bold text-3xl">卡包</span>
          <span class="text-gray-400 text-sm">kabao.me</span>
        </div>
        <p class="text-gray-500">用户端登录</p>
      </div>

      <div class="bg-white rounded-2xl p-6 shadow-lg">
        <h2 class="text-xl font-bold text-gray-800 mb-6">欢迎回来</h2>
        
        <form @submit.prevent="handleLogin">
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">手机号</label>
            <input
              v-model="username"
              type="text"
              placeholder="请输入手机号"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
            />
          </div>

          <div class="mb-6">
            <label class="block text-gray-700 text-sm font-medium mb-2">密码</label>
            <input
              v-model="password"
              type="password"
              placeholder="请输入密码"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
            />
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium hover:bg-primary-dark transition-colors disabled:opacity-50"
          >
            {{ loading ? '登录中...' : '登录' }}
          </button>
        </form>

        <div class="mt-6 text-center">
          <div class="mb-2">
            <span class="text-sm text-gray-500">还没有账号？</span>
            <router-link to="/user/register" class="text-sm text-primary hover:underline ml-1">
              立即注册
            </router-link>
          </div>
          <router-link to="/merchant/login" class="text-sm text-primary hover:underline">
            切换到商户端登录
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../../api'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)

const handleLogin = async () => {
  loading.value = true
  
  try {
    const res = await authApi.login(username.value, password.value)
    const { token, user_id, nickname } = res.data.data
    
    // 保存登录状态
    localStorage.setItem('userToken', token)
    localStorage.setItem('userId', user_id)
    localStorage.setItem('userName', nickname || username.value)
    
    // 跳转到用户卡片列表
    router.push('/user/cards')
  } catch (err) {
    alert(err.response?.data?.error || '登录失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>
