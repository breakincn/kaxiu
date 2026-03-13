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
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Html5Qrcode } from 'html5-qrcode'
import { cardApi } from '../../api'
import PwaInstallGuide from '../../components/PwaInstallGuide.vue'

import { getMerchantToken } from '../../utils/auth'
import { getScanStartLabel, getStartCodePromptLabel, getStartSuccessLabel, replaceTerms } from '../../utils/terms'

const router = useRouter()
const route = useRoute()

const pageTitle = ref('扫码')

const mode = ref('verify')

const routeTermMerchant = computed(() => {
  const queueMode = String(route.query.queue_mode || '').trim() === '1'
  return {
    start_term: String(route.query.start_term || '').trim(),
    finish_term: String(route.query.finish_term || '').trim(),
    support_queue: queueMode,
    queue_mode: queueMode ? 'manual' : '',
    support_customer_service_mode: false
  }
})

const getReturnPath = () => {
  const p = String(route.query.return_path || '').trim()
  return p || '/merchant'
}

const getReturnTab = () => {
  const t = String(route.query.tab || '').trim()
  if (t) return t
  return isStartOnlyMode() ? 'start' : 'verify'
}

const isStartOnlyMode = () => {
  return String(mode.value || '').trim() === 'start'
}

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
let jumpTimer = null
const PENDING_VERIFY_STORAGE_KEY = 'kabao_pending_verify_commit'

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

  // 开始服务专用模式：仅允许 SS:<session_id>
  if (isStartOnlyMode() && !code.startsWith('SS:')) {
    resultSuccess.value = false
    resultText.value = getStartCodePromptLabel(routeTermMerchant.value)
    return
  }

  verifying.value = true
  try {
    if (isStartOnlyMode()) {
      const res = await cardApi.scanVerify(code)
      handleCommitSuccess(res?.data?.data || {})
      return
    }

    const prepareRes = await cardApi.prepareVerify(code)
    const prepareData = prepareRes?.data?.data || {}
    const verifyToken = String(prepareData.verify_token || '').trim()
    if (!verifyToken) {
      throw new Error('核销令牌生成失败')
    }
    if (prepareData.need_hand_card) {
      try {
        sessionStorage.setItem(PENDING_VERIFY_STORAGE_KEY, JSON.stringify({
          verify_token: verifyToken,
          created_at: Date.now()
        }))
      } catch (_) {
        throw new Error('暂存核销状态失败，请重试')
      }
      await stop()
      router.replace({
        path: getReturnPath(),
        query: {
          tab: getReturnTab(),
          pending_verify_commit: '1'
        }
      })
      return
    }

    const commitRes = await cardApi.commitVerify(verifyToken, '', false)
    handleCommitSuccess(commitRes?.data?.data || {})
  } catch (err) {
    resultSuccess.value = false
    const errorMsg = err.response?.data?.error || '扫码失败'
    resultText.value = errorMsg
  } finally {
    verifying.value = false
  }
}

const handleCommitSuccess = (data) => {
  const action = data?.action || 'verify'
  resultSuccess.value = true
  if (action === 'start') {
    const sid = data?.session_id
    resultText.value = getStartSuccessLabel(routeTermMerchant.value, sid)
  } else {
    const remainTimes = data?.remain_times
    resultText.value = `核销成功！剩余次数: ${remainTimes ?? '-'}`
  }

  const returnPath = getReturnPath()
  const backTab = getReturnTab()
  const jump = () => {
    router.replace({ path: returnPath, query: { tab: backTab } })
  }
  if (jumpTimer) {
    clearTimeout(jumpTimer)
  }
  jumpTimer = setTimeout(() => {
    jump()
  }, 1000)
}

onMounted(() => {
  mode.value = String(route.query.mode || 'verify')
  pageTitle.value = isStartOnlyMode()
    ? getScanStartLabel(routeTermMerchant.value)
    : replaceTerms('扫码核销', routeTermMerchant.value)

  const token = getMerchantToken()
  if (!token) {
    router.replace('/login')
    return
  }

  loadCameras()
  // 默认自动启动一次
  start()
})

onUnmounted(() => {
  stop()
  if (jumpTimer) {
    clearTimeout(jumpTimer)
    jumpTimer = null
  }
})
</script>
