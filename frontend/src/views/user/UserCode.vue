<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">我的用户码</span>
    </header>

    <div class="px-4 py-6">
      <div class="bg-white rounded-xl shadow-sm p-6">
        <div class="text-center">
          <div class="text-gray-800 font-medium">请商户扫码查询我的卡片</div>
          <div class="text-gray-500 text-sm mt-1">用户码有效期 5 分钟</div>

          <div class="mt-5 flex items-center justify-center">
            <div class="rounded-2xl bg-white p-4 border border-gray-100" style="-webkit-touch-callout: none; -webkit-user-select: none; user-select: none;">
              <canvas ref="qrCanvas" class="block" style="width: 220px; height: 220px;" />
            </div>
          </div>

          <div class="mt-4">
            <div v-if="loading" class="text-gray-400">加载中...</div>
            <div v-else class="text-sm" :class="remainSeconds > 0 ? 'text-gray-600' : 'text-red-600'">
              {{ remainSeconds > 0 ? `剩余 ${remainSeconds} 秒` : '已过期，请刷新' }}
            </div>
          </div>

          <button
            type="button"
            class="mt-5 w-full bg-primary text-white py-3 rounded-lg font-medium disabled:opacity-50"
            @click="refresh"
            :disabled="loading"
          >
            刷新用户码
          </button>

          <p class="text-gray-400 text-xs mt-3">
            提示：请在商户面前打开此页面，避免截图传播。
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import QRCode from 'qrcode'
import { userApi } from '../../api'

const router = useRouter()
const qrCanvas = ref(null)

const loading = ref(false)
const code = ref('')
const expiresAt = ref(0)
const nowTick = ref(Date.now())
const isVisible = ref(true)
let tickTimer = null

const remainSeconds = computed(() => {
  const expMs = (expiresAt.value || 0) * 1000
  if (!expMs) return 0
  const diff = Math.floor((expMs - nowTick.value) / 1000)
  return diff > 0 ? diff : 0
})

// 监听剩余时间，当过期时自动刷新
watch(remainSeconds, async (newValue) => {
  if (newValue === 0 && isVisible.value && !loading.value) {
    // 用户码过期且页面可见时自动刷新
    await refresh()
  }
})

const goBack = () => {
  router.back()
}

const renderQr = async () => {
  try {
    await nextTick()
    if (!qrCanvas.value || !code.value) return
    await QRCode.toCanvas(qrCanvas.value, code.value, {
      margin: 1,
      scale: 8,
      errorCorrectionLevel: 'M'
    })
  } catch (_) {
    // ignore
  }
}

const refresh = async () => {
  if (loading.value) return
  loading.value = true
  try {
    const res = await userApi.getUserCode()
    const data = res.data.data || {}
    code.value = String(data.code || '')
    expiresAt.value = Number(data.expires_at || 0)
    await renderQr()
  } catch (err) {
    alert(err.response?.data?.error || '获取用户码失败')
  } finally {
    loading.value = false
  }
}

// 页面可见性变化处理
const handleVisibilityChange = () => {
  isVisible.value = !document.hidden
  if (isVisible.value && remainSeconds.value === 0 && !loading.value) {
    // 页面重新可见且用户码已过期时刷新
    refresh()
  }
}

onMounted(async () => {
  tickTimer = setInterval(() => {
    nowTick.value = Date.now()
  }, 500)

  // 监听页面可见性变化
  document.addEventListener('visibilitychange', handleVisibilityChange)

  await refresh()
})

onUnmounted(() => {
  if (tickTimer) clearInterval(tickTimer)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>
