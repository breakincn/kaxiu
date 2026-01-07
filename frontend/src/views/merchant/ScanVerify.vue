<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1 text-gray-600">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">{{ pageTitle }}</span>
      <div class="w-8"></div>
    </header>

    <PwaInstallGuide pageKey="merchant_scan_verify" />

    <div class="px-4 py-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-center justify-between mb-3">
          <h3 class="font-medium text-gray-800">相机扫码</h3>
          <button
            v-if="hasStarted"
            @click="stop"
            class="px-3 py-1.5 bg-gray-100 text-gray-700 rounded-lg text-sm"
          >
            停止
          </button>
        </div>

        <div v-if="errorText" class="mb-3 p-3 bg-gray-50 border border-gray-100 text-gray-700 rounded-lg text-sm">
          {{ errorText }}
        </div>

        <div v-if="resultText" class="mb-3 p-3 rounded-lg text-sm" :class="resultSuccess ? 'bg-primary-light text-primary border border-gray-100' : 'bg-gray-50 text-gray-700 border border-gray-100'">
          {{ resultText }}
        </div>

        <div class="rounded-lg overflow-hidden border border-gray-200 bg-black">
          <div id="qr-reader" class="w-full"></div>
        </div>

        <div class="mt-4 flex gap-2">
          <button
            @click="start"
            :disabled="starting || verifying"
            class="flex-1 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ hasStarted ? '扫描中...' : (starting ? '启动中...' : '开始扫码') }}
          </button>
          <button
            @click="switchCamera"
            :disabled="starting || verifying || !hasStarted"
            class="px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium disabled:opacity-50"
          >
            切换
          </button>
        </div>

        <p class="text-gray-400 text-xs mt-3">
          提示：请允许浏览器使用摄像头权限，建议使用微信内置浏览器 / Safari / Chrome。
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Html5Qrcode } from 'html5-qrcode'
import { cardApi } from '../../api'
import PwaInstallGuide from '../../components/PwaInstallGuide.vue'

import { getMerchantId, getMerchantToken } from '../../utils/auth'

const router = useRouter()

const pageTitle = ref('扫码')

const starting = ref(false)
const verifying = ref(false)
const hasStarted = ref(false)
const errorText = ref('')
const resultText = ref('')
const resultSuccess = ref(false)

const currentCameraIndex = ref(0)
const cameras = ref([])

let html5QrCode = null
let lastScannedAt = 0

const goBack = () => {
  router.back()
}

const loadCameras = async () => {
  try {
    const allCameras = await Html5Qrcode.getCameras()
    cameras.value = allCameras
    
    // 默认选择后置摄像头（environment facing mode）
    if (allCameras && allCameras.length > 0) {
      const backCamera = allCameras.find(camera => 
        camera.label.toLowerCase().includes('back') || 
        camera.label.toLowerCase().includes('environment') ||
        camera.label.toLowerCase().includes('后') ||
        camera.label.toLowerCase().includes('主')
      )
      
      if (backCamera) {
        currentCameraIndex.value = allCameras.indexOf(backCamera)
      } else {
        // 如果找不到后置摄像头，尝试通过 facingMode 判断
        const environmentCamera = allCameras.find(camera => 
          !camera.label.toLowerCase().includes('front') &&
          !camera.label.toLowerCase().includes('user') &&
          !camera.label.toLowerCase().includes('前') &&
          !camera.label.toLowerCase().includes('自')
        )
        if (environmentCamera) {
          currentCameraIndex.value = allCameras.indexOf(environmentCamera)
        }
      }
    }
  } catch (e) {
    cameras.value = []
  }
}

const ensureInstance = () => {
  if (!html5QrCode) {
    html5QrCode = new Html5Qrcode('qr-reader')
  }
}

const start = async () => {
  if (starting.value || hasStarted.value) return

  errorText.value = ''
  resultText.value = ''

  starting.value = true
  try {
    ensureInstance()
    await loadCameras()

    const cameraId = cameras.value[currentCameraIndex.value]?.id

    const config = {
      fps: 10,
      qrbox: { width: 250, height: 250 },
      aspectRatio: 1.777778
    }

    const constraints = cameraId
      ? { deviceId: { exact: cameraId } }
      : { facingMode: { exact: 'environment' } }

    await html5QrCode.start(
      constraints,
      config,
      async (decodedText) => {
        const now = Date.now()
        if (now - lastScannedAt < 1200) return
        lastScannedAt = now

        await onDecoded(decodedText)
      },
      () => {}
    )

    hasStarted.value = true
  } catch (e) {
    errorText.value = e?.message || '启动摄像头失败'
    hasStarted.value = false
  } finally {
    starting.value = false
  }
}

const stop = async () => {
  if (!html5QrCode || !hasStarted.value) return
  try {
    await html5QrCode.stop()
    await html5QrCode.clear()
  } catch (_) {
    // ignore
  } finally {
    hasStarted.value = false
  }
}

const switchCamera = async () => {
  if (!hasStarted.value) return
  if (!cameras.value || cameras.value.length <= 1) return

  currentCameraIndex.value = (currentCameraIndex.value + 1) % cameras.value.length
  await stop()
  await start()
}

const onDecoded = async (decodedText) => {
  if (verifying.value) return

  const code = (decodedText || '').trim()
  if (!code) return

  verifying.value = true
  try {
    const res = await cardApi.scanVerify(code)
    const action = res?.data?.data?.action
    resultSuccess.value = true
    if (action === 'finish') {
      resultText.value = '结单成功！'
    } else {
      const remainTimes = res?.data?.data?.remain_times
      resultText.value = `核销成功！剩余次数: ${remainTimes ?? '-'}`
    }

    // 刷新商户后台数据：回到 dashboard 并切到 verify tab
    setTimeout(() => {
      router.replace({ path: '/merchant', query: { tab: 'verify' } })
    }, 700)
  } catch (err) {
    resultSuccess.value = false
    resultText.value = err.response?.data?.error || '扫码失败'
  } finally {
    verifying.value = false
  }
}

onMounted(() => {
  pageTitle.value = '扫码'

  const token = getMerchantToken()
  const id = getMerchantId()
  if (!token || !id) {
    router.replace('/login')
    return
  }

  loadCameras()
  // 默认自动启动一次
  start()
})

onUnmounted(() => {
  stop()
})
</script>
