<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1 text-gray-600">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">扫码查询卡片</span>
      <div class="w-8"></div>
    </header>

    <PwaInstallGuide pageKey="merchant_scan_card" />

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

        <div v-if="searching" class="mb-3 flex items-center gap-2 p-3 bg-gray-50 border border-gray-100 text-gray-700 rounded-lg text-sm">
          <svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"></path>
          </svg>
          <span>查询中...</span>
        </div>

        <div class="rounded-lg overflow-hidden border border-gray-200 bg-black">
          <div id="qr-reader" class="w-full"></div>
        </div>

        <div class="mt-4 flex gap-2">
          <button
            @click="start"
            :disabled="starting || searching"
            class="flex-1 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ hasStarted ? '扫描中...' : (starting ? '启动中...' : '开始扫码') }}
          </button>
          <button
            @click="switchCamera"
            :disabled="starting || searching || !hasStarted"
            class="px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium disabled:opacity-50"
          >
            切换
          </button>
        </div>

        <p class="text-gray-400 text-xs mt-3">
          提示：用户端卡片二维码格式为 kabao-card:&lt;卡片ID&gt;
        </p>
      </div>
    </div>

    <div v-if="showAlert" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="closeAlert">
      <div class="bg-white rounded-2xl w-11/12 max-w-sm overflow-hidden">
        <div class="px-5 py-4 border-b">
          <div class="font-medium text-gray-800">提示</div>
        </div>
        <div class="px-5 py-5 text-gray-700 text-sm">
          {{ alertText }}
        </div>
        <div class="px-5 pb-5">
          <button @click="closeAlert" class="w-full py-3 bg-primary text-white rounded-lg font-medium">
            我知道了
          </button>
        </div>
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

const starting = ref(false)
const searching = ref(false)
const hasStarted = ref(false)
const errorText = ref('')
const showAlert = ref(false)
const alertText = ref('')

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

const parseCardIdFromQr = (text) => {
  const v = (text || '').trim()
  if (!v) return null
  if (v.startsWith('kabao-card:')) {
    const idStr = v.slice('kabao-card:'.length).trim()
    const id = parseInt(idStr)
    if (!Number.isFinite(id) || id <= 0) return null
    return id
  }
  return null
}

const parseUserCodeFromQr = (text) => {
  const v = (text || '').trim()
  if (!v) return null
  if (v.startsWith('kabao-user:')) {
    return v
  }
  return null
}

const start = async () => {
  if (starting.value || hasStarted.value) return

  errorText.value = ''
  showAlert.value = false
  alertText.value = ''

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
  if (searching.value) return

  const cardId = parseCardIdFromQr(decodedText)
  const userCode = parseUserCodeFromQr(decodedText)
  if (!cardId && !userCode) {
    showAlert.value = true
    alertText.value = '二维码格式不正确'
    return
  }

  searching.value = true
  try {
    await stop()
    if (cardId) {
      await cardApi.getMerchantCard(cardId)
      router.replace(`/merchant/cards/${cardId}`)
      return
    }

    router.replace({
      path: '/merchant',
      query: {
        tab: 'cards',
        user_code: userCode,
        from_scan: '1'
      }
    })
  } catch (err) {
    const status = err.response?.status
    if (status === 403) {
      showAlert.value = true
      alertText.value = '他家卡片，请出示我家卡片扫码'
    } else {
      showAlert.value = true
      alertText.value = err.response?.data?.error || '查询失败'
    }
  } finally {
    searching.value = false
  }
}

const closeAlert = () => {
  showAlert.value = false
  alertText.value = ''
}

onMounted(() => {
  const token = getMerchantToken()
  const id = getMerchantId()
  if (!token || !id) {
    router.replace('/merchant/login')
    return
  }

  loadCameras()
  start()
})

onUnmounted(() => {
  stop()
})
</script>
