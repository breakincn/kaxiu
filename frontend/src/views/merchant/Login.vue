<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center px-4">
    <div class="w-full max-w-md">
      <div class="text-center mb-8">
        <div class="flex items-center justify-center gap-2 mb-2">
          <span class="text-primary font-bold text-3xl">卡包</span>
          <span class="text-gray-400 text-sm">kabao.shop</span>
        </div>
        <p class="text-gray-500">{{ isTechnicianLogin ? '技师端登录' : '商户端登录' }}</p>
      </div>

      <div class="bg-white rounded-2xl p-6 shadow-lg">
        <h2 class="text-xl font-bold text-gray-800 mb-6">{{ isTechnicianLogin ? '技师登录' : '商户管理后台' }}</h2>
        
        <form @submit.prevent="handleLogin">
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">{{ isTechnicianLogin ? '技师账号' : '账号/手机号' }}</label>
            <input
              v-model="phone"
              type="text"
              :placeholder="isTechnicianLogin ? '请输入技师账号（如js0001）' : '请输入账号/手机号'"
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

        <div class="mt-6 text-center space-y-2">
          <div>
            <span class="text-sm text-gray-500">还没有账号？</span>
            <button @click="showRegister = true" class="text-sm text-primary hover:underline ml-1">
              立即注册
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 注册弹窗 -->
    <div v-if="showRegister" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="showRegister = false">
      <div class="bg-white rounded-2xl w-full max-w-md max-h-[90vh] overflow-y-auto p-6">
        <div class="flex items-center justify-between mb-6">
          <h3 class="text-xl font-bold text-gray-800">商户注册</h3>
          <button @click="showRegister = false" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <form @submit.prevent="handleRegister">
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">手机号</label>
            <input
              v-model="registerForm.phone"
              type="tel"
              placeholder="请输入手机号"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
            />
          </div>

          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">验证码</label>
            <div class="flex gap-2">
              <input
                v-model="registerForm.code"
                type="text"
                placeholder="请输入验证码"
                class="flex-1 px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                required
              />
              <button
                type="button"
                :disabled="sendingCode || countdown > 0"
                class="px-3 py-3 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-200 transition-colors disabled:opacity-50 whitespace-nowrap shrink-0"
                @click="sendRegisterCode"
              >
                {{ countdown > 0 ? `${countdown}s` : (sendingCode ? '发送中...' : '发送验证码') }}
              </button>
            </div>
          </div>

          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">密码</label>
            <input
              v-model="registerForm.password"
              type="password"
              placeholder="请输入密码（至少6位）"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
              minlength="6"
            />
          </div>

          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">商户名称</label>
            <input
              v-model="registerForm.name"
              type="text"
              placeholder="请输入商户名称"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
            />
          </div>

          <div class="mb-6">
            <label class="block text-gray-700 text-sm font-medium mb-2">商户类型</label>
            <input
              v-model="registerForm.type"
              type="text"
              placeholder="如：理发、美容、健身等"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <div class="mb-6">
            <label class="block text-gray-700 text-sm font-medium mb-2">邀请码</label>
            <input
              v-model="registerForm.invite_code"
              type="text"
              placeholder="请输入邀请码"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
            />
          </div>

          <button
            type="submit"
            :disabled="registering"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium hover:bg-primary-dark transition-colors disabled:opacity-50"
          >
            {{ registering ? '注册中...' : '注册' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ensureMerchantPermissionsLoaded, merchantApi, shopApi, smsApi } from '../../api'

import { setMerchantActiveAuth, setTechnicianShopSlug } from '../../utils/auth'

const router = useRouter()
const route = useRoute()

const isTechnicianLogin = ref(false)
const shopSlug = ref('')

isTechnicianLogin.value = typeof route.params?.slug === 'string' && route.path.startsWith('/shop/')
shopSlug.value = isTechnicianLogin.value ? String(route.params.slug) : ''
const phone = ref('')
const password = ref('')
const loading = ref(false)

const showRegister = ref(false)
const registering = ref(false)
const sendingCode = ref(false)

const countdown = ref(0)
let timer = null

const registerForm = ref({
  phone: '',
  code: '',
  password: '',
  name: '',
  type: '',
  invite_code: ''
})

const startCountdown = () => {
  countdown.value = 60
  timer = setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0) {
      clearInterval(timer)
      timer = null
    }
  }, 1000)
}

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
})

const sendRegisterCode = async () => {
  if (!registerForm.value.phone) {
    alert('请先输入手机号')
    return
  }

  sendingCode.value = true
  try {
    const res = await smsApi.send(registerForm.value.phone, 'merchant_register')
    const debugCode = res.data?.data?.debug_code
    if (debugCode) {
      alert(`验证码已发送（开发模式）：${debugCode}`)
    } else {
      alert('验证码已发送')
    }
    startCountdown()
  } catch (err) {
    alert(err.response?.data?.error || '发送失败，请重试')
  } finally {
    sendingCode.value = false
  }
}

const handleLogin = async () => {
  loading.value = true
  
  try {
    const res = isTechnicianLogin.value
      ? await shopApi.technicianLogin(shopSlug.value, phone.value, password.value)
      : await merchantApi.login(phone.value, password.value)
    console.log('登录响应:', res.data)
    console.log('完整响应对象:', res)
    
    // 检查响应结构
    if (!res.data || !res.data.merchant) {
      console.error('登录响应结构异常:', res.data)
      throw new Error('登录响应数据格式错误')
    }
    
    console.log('商户信息:', res.data.merchant)
    
    // 保存登录状态：商户与技师分离，避免互相覆盖
    if (res.data.technician && res.data.technician.id) {
      // 技师登录（来自 /shop/:slug/login）
      setMerchantActiveAuth('technician')
      setTechnicianShopSlug(shopSlug.value)

      localStorage.setItem('technicianToken', res.data.token)
      localStorage.setItem('technicianMerchantId', res.data.merchant.id)
      localStorage.setItem('technicianMerchantName', res.data.merchant.name)
      localStorage.setItem('technicianMerchantPhone', res.data.merchant.phone)

      sessionStorage.setItem('technicianId', res.data.technician.id)
      sessionStorage.setItem('technicianName', res.data.technician.name || '')
      sessionStorage.setItem('technicianCode', res.data.technician.code || '')
      sessionStorage.setItem('technicianAccount', res.data.technician.account || '')
    } else {
      // 商户登录（来自 /merchant/login）
      setMerchantActiveAuth('merchant')
      setTechnicianShopSlug('')

      localStorage.setItem('merchantToken', res.data.token)
      localStorage.setItem('merchantId', res.data.merchant.id)
      localStorage.setItem('merchantName', res.data.merchant.name)
      localStorage.setItem('merchantPhone', res.data.merchant.phone)

      sessionStorage.removeItem('technicianId')
      sessionStorage.removeItem('technicianName')
      sessionStorage.removeItem('technicianCode')
      sessionStorage.removeItem('technicianAccount')
    }

    await ensureMerchantPermissionsLoaded()
    
    console.log('登录信息已保存，准备跳转...')
    console.log('merchantToken:', localStorage.getItem('merchantToken'))
    console.log('merchantId:', localStorage.getItem('merchantId'))
    console.log('technicianToken:', localStorage.getItem('technicianToken'))
    console.log('technicianMerchantId:', localStorage.getItem('technicianMerchantId'))
    
    console.log('即将执行 router.push("/merchant")')
    router.push('/merchant').then(() => {
      console.log('router.push 成功')
    }).catch(err => {
      console.error('router.push 失败:', err)
    })
  } catch (err) {
    console.error('登录失败:', err)
    console.error('错误响应:', err.response)
    alert(err.response?.data?.error || '登录失败，请重试')
  } finally {
    loading.value = false
  }
}

const handleRegister = async () => {
  registering.value = true
  
  try {
    console.log('正在注册，数据:', registerForm.value)
    const res = await merchantApi.register(registerForm.value)
    console.log('注册响应:', res.data)
    
    alert('注册成功！请登录')
    
    // 关闭注册弹窗，填充登录表单
    showRegister.value = false
    phone.value = registerForm.value.phone
    password.value = registerForm.value.password
    
    // 清空注册表单
    registerForm.value = {
      phone: '',
      code: '',
      password: '',
      name: '',
      type: '',
      invite_code: ''
    }
  } catch (err) {
    console.error('注册失败:', err)
    console.error('错误响应:', err.response)
    alert(err.response?.data?.error || '注册失败，请重试')
  } finally {
    registering.value = false
  }
}
</script>
