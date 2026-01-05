<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">{{ currentPhone ? '更换手机号' : '绑定手机号' }}</span>
    </header>

    <!-- 加载中 -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="text-gray-400">加载中...</div>
    </div>

    <!-- 已绑定手机号 -->
    <div v-else-if="currentPhone && !isChanging" class="px-4 py-6">
      <div class="bg-white rounded-xl shadow-sm p-6">
        <div class="text-center">
          <div class="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg class="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
          </div>
          <h3 class="text-lg font-medium text-gray-800 mb-2">已绑定手机号</h3>
          <p class="text-2xl font-semibold text-gray-800 mb-6 no-underline" style="text-decoration: none;">{{ currentPhone }}</p>
          
          <button
            @click="startChanging"
            class="w-full bg-blue-500 text-white py-3 rounded-lg hover:bg-blue-600 font-medium"
          >
            更换手机号
          </button>
        </div>
      </div>
      
      <!-- 温馨提示 -->
      <div v-if="!isTechnician()" class="mt-4 px-4 py-3 bg-yellow-50 rounded-lg">
        <p class="text-sm text-yellow-600">
          <svg class="w-4 h-4 inline-block mr-1" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
          </svg>
          更换手机号需要验证商户密码和新手机号验证码，请确保新手机号可以正常接收验证码
        </p>
      </div>
    </div>

    <!-- 绑定/换绑表单 -->
    <div v-else class="px-4 py-6">
      <div class="bg-white rounded-xl shadow-sm p-6">
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <!-- 换绑时需要验证密码 -->
          <div v-if="currentPhone && !isTechnician()">
            <label class="block text-sm font-medium text-gray-700 mb-2">商户密码</label>
            <input
              v-model="password"
              type="password"
              placeholder="请输入商户密码"
              class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

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
            {{ currentPhone ? '确认更换' : '确认绑定' }}
          </button>
          
          <!-- 取消换绑按钮 -->
          <button
            v-if="isChanging"
            type="button"
            @click="cancelChanging"
            class="w-full bg-gray-200 text-gray-700 py-3 rounded-lg hover:bg-gray-300 font-medium mt-3"
          >
            取消
          </button>
        </form>
      </div>

      <!-- 温馨提示 -->
      <div v-if="!isChanging" class="mt-4 px-4 py-3 bg-blue-50 rounded-lg">
        <p class="text-sm text-blue-600">
          <svg class="w-4 h-4 inline-block mr-1" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"/>
          </svg>
          {{ isTechnician() ? '绑定手机号后，将用于该账号的联系方式' : '绑定手机号后，可以使用手机号登录商户管理后台' }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

import { getMerchantActiveAuth } from '../../utils/auth'

const router = useRouter()
const loading = ref(true)
const currentPhone = ref('')
const isChanging = ref(false)
const phone = ref('')
const code = ref('')
const password = ref('')
const countdown = ref(0)
let timer = null

const isTechnician = () => getMerchantActiveAuth() === 'technician'

// 获取当前账号信息
const fetchMerchantInfo = async () => {
  loading.value = true
  try {
    if (isTechnician()) {
      const response = await merchantApi.getCurrentTechnician()
      const tech = response.data.data
      if (tech.phone) currentPhone.value = tech.phone
    } else {
      const response = await merchantApi.getCurrentMerchant()
      const merchant = response.data.data
      if (merchant.phone) currentPhone.value = merchant.phone
    }
  } catch (error) {
    console.error('获取商户信息失败:', error)
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  router.back()
}

const startChanging = () => {
  isChanging.value = true
  phone.value = ''
  code.value = ''
  password.value = ''
}

const cancelChanging = () => {
  isChanging.value = false
  phone.value = ''
  code.value = ''
  password.value = ''
  if (countdown.value > 0) {
    clearInterval(timer)
    countdown.value = 0
  }
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
    // 使用SMS API发送验证码
    const response = await import('../../api').then(({ default: api }) => 
      api.post('/sms/send', {
        phone: phone.value,
        purpose: isTechnician() ? 'technician_bind_phone' : 'merchant_bind_phone'
      })
    )
    
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
  
  // 换绑时需要验证密码（仅商户账号）
  if (!isTechnician() && currentPhone.value && !password.value) {
    alert('请输入商户密码')
    return
  }

  try {
    if (isTechnician()) {
      await merchantApi.bindTechnicianPhone(phone.value, code.value)
    } else {
      await merchantApi.bindPhone(phone.value, code.value, password.value)
    }
    
    alert(currentPhone.value ? '更换成功！' : '绑定成功！')
    // 更新当前手机号
    currentPhone.value = phone.value
    isChanging.value = false
    phone.value = ''
    code.value = ''
    password.value = ''
    
    // 更新localStorage中的手机号（仅商户账号）
    if (!isTechnician()) {
      localStorage.setItem('merchantPhone', phone.value)
    }
  } catch (error) {
    alert(error.response?.data?.error || (currentPhone.value ? '更换失败' : '绑定失败'))
  }
}

// 页面加载时获取商户信息
onMounted(() => {
  fetchMerchantInfo()
})

// 清理定时器
onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<style scoped>
/* 禁止浏览器自动识别电话号码为链接 */
a[href^="tel:"] {
  color: inherit !important;
  text-decoration: none !important;
  pointer-events: none;
}

/* 确保所有链接样式被禁用 */
p a {
  color: inherit !important;
  text-decoration: none !important;
  pointer-events: none;
}
</style>
