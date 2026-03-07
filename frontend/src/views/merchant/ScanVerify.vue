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

    <div v-if="showHandCardModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4" @click="onHandCardMaskClick">
      <div class="w-full max-w-sm bg-white rounded-xl p-4 shadow-lg" @click.stop>
        <div class="flex items-center justify-between">
          <div class="text-gray-800 font-medium text-base">分配手牌</div>
          <button @click="closeHandCardModal" class="p-1 text-gray-500">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="text-gray-500 text-sm mt-1">请输入本次核销对应的手牌号</div>
        <input
          v-model="handCardInput"
          type="text"
          inputmode="numeric"
          pattern="[0-9]*"
          placeholder="例如：H001"
          class="w-full mt-3 px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
        />
        <div v-if="handCardError" class="text-red-600 text-sm mt-2">{{ handCardError }}</div>
        <div class="mt-4 flex gap-2">
          <button
            @click="submitHandCard(false)"
            :disabled="submittingHandCard"
            class="px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium disabled:opacity-50"
          >
            跳过分配
          </button>
          <button
            @click="submitHandCard(true)"
            :disabled="submittingHandCard"
            class="flex-1 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ submittingHandCard ? '提交中...' : '确认绑定' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Html5Qrcode } from 'html5-qrcode'
import { cardApi } from '../../api'
import PwaInstallGuide from '../../components/PwaInstallGuide.vue'

import { getMerchantToken } from '../../utils/auth'
import { replaceTerms } from '../../utils/terms'

const router = useRouter()
const route = useRoute()

const pageTitle = ref('扫码')

const mode = ref('verify')

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

const showHandCardModal = ref(false)
const submittingHandCard = ref(false)
const handCardInput = ref('')
const handCardError = ref('')
const pendingVerifyToken = ref('')


const currentCameraIndex = ref(0)
const cameras = ref([])

let html5QrCode = null
let lastScannedAt = 0
let jumpTimer = null

const goBack = () => {
  if (showHandCardModal.value) return
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
  if (verifying.value || showHandCardModal.value) return

  const code = (decodedText || '').trim()
  if (!code) return

  // 起单专用模式：仅允许 SS:<session_id>
  if (isStartOnlyMode() && !code.startsWith('SS:')) {
    resultSuccess.value = false
    resultText.value = replaceTerms('请扫描起单码')
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
      pendingVerifyToken.value = verifyToken
      handCardInput.value = ''
      handCardError.value = ''
      showHandCardModal.value = true
      resultSuccess.value = false
      resultText.value = '请先分配手牌，再完成核销'
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
    resultText.value = replaceTerms(`起单成功！服务单#${sid ?? '-'}，已开始计时。`)
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

const closeHandCardModal = () => {
  if (!showHandCardModal.value) return
  const no = String(handCardInput.value || '').trim()
  if (!no) {
    if (!confirm('关闭后本次核销不会提交，确定关闭吗？')) return
  }

  showHandCardModal.value = false
  pendingVerifyToken.value = ''
  handCardInput.value = ''
  handCardError.value = ''
  resultSuccess.value = false
  resultText.value = '已取消本次核销'
}

const onHandCardMaskClick = () => {
  closeHandCardModal()
}


const submitHandCard = async (doBind) => {
  if (submittingHandCard.value) return
  handCardError.value = ''

  const verifyToken = String(pendingVerifyToken.value || '').trim()
  if (!verifyToken) {
    showHandCardModal.value = false
    return
  }

  submittingHandCard.value = true
  try {
    const no = String(handCardInput.value || '').trim()
    if (doBind) {
      if (!no) {
        handCardError.value = '请输入手牌号'
        return
      }
    } else if (!confirm('确认本次核销跳过手牌分配吗？')) {
      return
    }
    const res = await cardApi.commitVerify(verifyToken, no, !doBind)
    showHandCardModal.value = false
    pendingVerifyToken.value = ''
    handCardInput.value = ''
    handleCommitSuccess(res?.data?.data || {})
  } catch (e) {
    handCardError.value = e?.response?.data?.error || '核销提交失败'
  } finally {
    submittingHandCard.value = false
  }
}

onMounted(() => {
  mode.value = String(route.query.mode || 'verify')
  pageTitle.value = isStartOnlyMode() ? replaceTerms('扫码起单') : replaceTerms('扫码核销')

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
