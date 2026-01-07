<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center px-4">
    <div class="w-full max-w-md">
      <div class="text-center mb-8">
        <div class="flex items-center justify-center gap-2 mb-2">
          <span class="text-primary font-bold text-3xl">卡包</span>
          <span class="text-gray-400 text-sm">kabao.app</span>
        </div>
        <p class="text-gray-500">用户端注册</p>
      </div>

      <div class="bg-white rounded-2xl p-6 shadow-lg">
        <h2 class="text-xl font-bold text-gray-800 mb-6">创建账号</h2>

        <form @submit.prevent="handleRegister">
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">用户名</label>
            <input
              v-model="form.username"
              type="text"
              placeholder="请输入用户名"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
            />
          </div>

          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">密码</label>
            <input
              v-model="form.password"
              type="password"
              placeholder="请输入密码（至少6位）"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              required
              minlength="6"
            />
          </div>

          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">手机号（可选）</label>
            <input
                v-model="form.phone"
                type="tel"
                placeholder="请输入手机号"
                class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <div v-if="form.phone" class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">验证码</label>
            <div class="flex gap-2">
              <input
                  v-model="form.code"
                  type="text"
                  placeholder="请输入验证码"
                  class="flex-1 px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                  required
              />
              <button
                  type="button"
                  :disabled="sendingCode || countdown > 0"
                  class="px-3 py-3 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-200 transition-colors disabled:opacity-50 whitespace-nowrap shrink-0"
                  @click="sendCode"
              >
                {{ countdown > 0 ? `${countdown}s` : (sendingCode ? '发送中...' : '发送验证码') }}
              </button>
            </div>
          </div>

          <div class="mb-6">
            <label class="block text-gray-700 text-sm font-medium mb-2">昵称（可选）</label>
            <input
              v-model="form.nickname"
              type="text"
              placeholder="请输入昵称"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <button
            type="submit"
            :disabled="registering"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium hover:bg-primary-dark transition-colors disabled:opacity-50"
          >
            {{ registering ? '注册中...' : '注册并登录' }}
          </button>
        </form>

        <div class="mt-6 text-center">
          <router-link to="/login" class="text-sm text-primary hover:underline">
            返回登录
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi, smsApi } from '../../api'

const router = useRouter()

const form = ref({
  username: '',
  phone: '',
  code: '',
  password: '',
  nickname: ''
})

const sendingCode = ref(false)
const registering = ref(false)

const countdown = ref(0)
let timer = null

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

const sendCode = async () => {
  if (!form.value.phone) {
    alert('请先输入手机号')
    return
  }

  sendingCode.value = true
  try {
    const res = await smsApi.send(form.value.phone, 'user_register')
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

const handleRegister = async () => {
  registering.value = true
  try {
    const res = await authApi.register(form.value)
    const { token, user_id, nickname, username } = res.data.data

    localStorage.setItem('userToken', token)
    localStorage.setItem('userId', user_id)
    localStorage.setItem('userName', nickname || username)

    router.push('/user/cards')
  } catch (err) {
    alert(err.response?.data?.error || '注册失败，请重试')
  } finally {
    registering.value = false
  }
}
</script>
