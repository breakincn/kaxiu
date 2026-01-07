<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1 text-gray-600">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">扫码进店</span>
      <div class="w-8"></div>
    </header>

    <PwaInstallGuide pageKey="user_scan_shop_pay" />

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

        <div v-if="errorText" class="mb-3 p-3 bg-red-50 border border-red-100 text-red-600 rounded-lg text-sm">
          {{ errorText }}
        </div>

        <div v-if="statusText" class="mb-3 p-3 rounded-lg text-sm bg-blue-50 text-blue-700 border border-blue-100">
          {{ statusText }}
        </div>

        <div class="rounded-lg overflow-hidden border border-gray-200 bg-black">
          <div id="qr-reader-pay" class="w-full"></div>
        </div>

        <div class="mt-4 flex gap-2">
          <button
            @click="start"
            :disabled="starting || resolving"
            class="flex-1 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ hasStarted ? '扫描中...' : (starting ? '启动中...' : '开始扫码') }}
          </button>
          <button
            @click="switchCamera"
            :disabled="starting || resolving || !hasStarted"
            class="px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium disabled:opacity-50"
          >
            切换
          </button>
        </div>

        <p class="text-gray-400 text-xs mt-3">
          提示：请扫描商户售卡二维码，识别后将进入对应店铺页。
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Html5Qrcode } from 'html5-qrcode'
import PwaInstallGuide from '../../components/PwaInstallGuide.vue'

const router = useRouter()

const starting = ref(false)
const resolving = ref(false)
const hasStarted = ref(false)
const errorText = ref('')
const statusText = ref('')

const currentCameraIndex = ref(0)
const cameras = ref([])
const hasAutoSelectedBackCamera = ref(false)
const hasUserSwitchedCamera = ref(false)

let html5QrCode = null
let lastScannedAt = 0

const goBack = () => {
  router.back()
}

const loadCameras = async () => {
  try {
    cameras.value = await Html5Qrcode.getCameras()

    if (!hasAutoSelectedBackCamera.value && cameras.value.length > 0) {
      const idx = cameras.value.findIndex((c) => {
        const label = String(c?.label || '').toLowerCase()
        return (
          label.includes('back') ||
          label.includes('rear') ||
          label.includes('environment') ||
          label.includes('后') ||
          label.includes('背') ||
          label.includes('外')
        )
      })
      if (idx >= 0) {
        currentCameraIndex.value = idx
      }
      hasAutoSelectedBackCamera.value = true
    }
  } catch (e) {
    cameras.value = []
  }
}

const ensureInstance = () => {
  if (!html5QrCode) {
    html5QrCode = new Html5Qrcode('qr-reader-pay')
  }
}

const start = async () => {
  if (starting.value || hasStarted.value) return

  errorText.value = ''
  statusText.value = ''
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

    const constraints = (hasUserSwitchedCamera.value && cameraId)
      ? { deviceId: { exact: cameraId } }
      : { facingMode: 'environment' }

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
  } finally {
    hasStarted.value = false
  }
}

const switchCamera = async () => {
  if (!hasStarted.value) return
  if (!cameras.value || cameras.value.length <= 1) return

  hasUserSwitchedCamera.value = true
  currentCameraIndex.value = (currentCameraIndex.value + 1) % cameras.value.length
  await stop()
  await start()
}

const parseShopTarget = (rawText) => {
  const text = (rawText || '').trim()
  if (!text) return null

  try {
    if (text.startsWith('http://') || text.startsWith('https://')) {
      const u = new URL(text)
      const path = u.pathname || ''
      const parts = path.split('/').filter(Boolean)
      if (parts[0] === 'shop' && parts[1]) {
        if (parts[1] === 'id' && parts[2]) {
          return { kind: 'id', value: parts[2] }
        }
        return { kind: 'slug', value: parts[1] }
      }
      return null
    }

    const parts = text.split('/').filter(Boolean)
    if (parts[0] === 'shop' && parts[1]) {
      if (parts[1] === 'id' && parts[2]) {
        return { kind: 'id', value: parts[2] }
      }
      return { kind: 'slug', value: parts[1] }
    }

    return null
  } catch (_) {
    return null
  }
}

const onDecoded = async (decodedText) => {
  if (resolving.value) return

  const target = parseShopTarget(decodedText)
  if (!target) {
    errorText.value = '二维码不是商户售卡码'
    return
  }

  resolving.value = true
  try {
    statusText.value = '识别成功，进入店铺中...'

    const path = target.kind === 'slug'
      ? `/s/${encodeURIComponent(target.value)}`
      : `/s/id/${encodeURIComponent(target.value)}`

    await stop()
    router.replace(path)
  } catch (err) {
    errorText.value = err?.message || '跳转失败'
    statusText.value = ''
  } finally {
    resolving.value = false
  }
}

onMounted(() => {
  const token = localStorage.getItem('userToken')
  const id = localStorage.getItem('userId')
  if (!token || !id) {
    router.replace('/login')
    return
  }

  loadCameras()
  start()
})

onUnmounted(() => {
  stop()
})
</script>
