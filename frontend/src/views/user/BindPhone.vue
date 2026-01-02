<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">绑定手机号</span>
    </header>

    <!-- 表单内容 -->
    <div class="px-4 py-6">
      <div class="bg-white rounded-xl shadow-sm p-6">
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <!-- 手机号输入 -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">手机号</label>
            <input
              v-model="phone"
              type="tel"
              placeholder="请输入手机号"
              maxlength="11"
              class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <!-- 验证码输入 -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">验证码</label>
            <div class="flex gap-2">
              <input
                v-model="code"
                type="text"
                placeholder="请输入验证码"
                maxlength="6"
                class="flex-1 px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <button
                type="button"
                @click="sendCode"
                :disabled="countdown > 0"
                class="px-4 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-300 disabled:cursor-not-allowed whitespace-nowrap"
              >
                {{ countdown > 0 ? `${countdown}秒后重试` : '发送验证码' }}
              </button>
            </div>
          </div>

          <!-- 提交按钮 -->
          <button
            type="submit"
            class="w-full bg-blue-500 text-white py-3 rounded-lg hover:bg-blue-600 font-medium mt-6"
          >
            确认绑定
          </button>
        </form>
      </div>

      <!-- 温馨提示 -->
      <div class="mt-4 px-4 py-3 bg-blue-50 rounded-lg">
        <p class="text-sm text-blue-600">
          <svg class="w-4 h-4 inline-block mr-1" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"/>
          </svg>
          绑定手机号后，可以使用手机号登录账户
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'

const router = useRouter()
const phone = ref('')
const code = ref('')
const countdown = ref(0)
let timer = null

const goBack = () => {
  router.back()
}

const sendCode = async () => {
  if (!phone.value) {
    alert('请输入手机号')
    return
  }
  if (!/^1[3-9]\d{9}$/.test(phone.value)) {
    alert('请输入正确的手机号')
    return
  }

  try {
    const response = await api.post('/sms/send', {
      phone: phone.value,
      purpose: 'user_bind_phone'
    })
    
    // 显示调试验证码（开发环境）
    if (response.data.data?.debug_code) {
      alert(`验证码已发送（开发环境）：${response.data.data.debug_code}`)
    } else {
      alert('验证码已发送，请查收短信')
    }
    
    // 开始倒计时
    countdown.value = 60
    timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)
  } catch (error) {
    alert(error.response?.data?.error || '发送验证码失败')
  }
}

const handleSubmit = async () => {
  if (!phone.value) {
    alert('请输入手机号')
    return
  }
  if (!/^1[3-9]\d{9}$/.test(phone.value)) {
    alert('请输入正确的手机号')
    return
  }
  if (!code.value) {
    alert('请输入验证码')
    return
  }

  try {
    await api.post('/user/bind-phone', {
      phone: phone.value,
      code: code.value
    })
    
    alert('绑定成功！')
    router.push('/user/settings')
  } catch (error) {
    alert(error.response?.data?.error || '绑定失败')
  }
}

// 清理定时器
import { onUnmounted } from 'vue'
onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>
