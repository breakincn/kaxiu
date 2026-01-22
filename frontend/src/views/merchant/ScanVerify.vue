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

      <div v-if="supportHandCard" class="bg-white rounded-xl p-4 shadow-sm mt-4">
        <div class="flex items-center justify-between mb-3">
          <h3 class="font-medium text-gray-800">归还手牌</h3>
          <button
            v-if="returnQueryUsage"
            @click="clearHandCardReturn()"
            class="px-3 py-1.5 bg-gray-100 text-gray-700 rounded-lg text-sm"
          >
            取消
          </button>
        </div>

        <div class="flex gap-2">
          <input
            v-model="handCardReturnInput"
            type="text"
            placeholder="请输入手牌号"
            class="flex-1 px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
          />
          <button
            @click="queryHandCardReturn"
            :disabled="returnQuerying || !handCardReturnInput"
            class="px-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ returnQuerying ? '查询中...' : '查询' }}
          </button>
        </div>
        <div v-if="returnError" class="text-red-600 text-sm mt-2">{{ returnError }}</div>

        <div v-if="returnQueryUsage" class="mt-3 p-3 bg-gray-50 rounded-lg">
          <div class="text-gray-800 text-sm font-medium">
            {{ returnQueryUsage.card?.user?.nickname || '用户' }} / 卡号：{{ returnQueryUsage.card?.card_no || '-' }}
          </div>
          <div class="text-gray-500 text-sm mt-1">项目：{{ returnQueryUsage.project?.name || '-' }}</div>
          <div class="text-gray-500 text-sm mt-1">手牌：{{ returnQueryUsage.hand_card_no || '-' }}</div>
          <div class="text-gray-500 text-sm mt-1">分配时间：{{ formatDateTime(returnQueryUsage.hand_card_assigned_at) }}</div>

          <button
            @click="confirmReturnHandCard"
            :disabled="returnConfirming"
            class="w-full mt-3 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ returnConfirming ? '归还中...' : '确认归还' }}
          </button>
        </div>

        <div v-if="returnSuccessText" class="mt-3 p-3 bg-primary-light text-primary rounded-lg text-sm">
          {{ returnSuccessText }}
        </div>
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
import { cardApi, merchantApi } from '../../api'
import PwaInstallGuide from '../../components/PwaInstallGuide.vue'

import { getMerchantId, getMerchantToken } from '../../utils/auth'
import { replaceTerms } from '../../utils/terms'
import { formatDateTime } from '../../utils/dateFormat'

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

const supportHandCard = ref(false)
const showHandCardModal = ref(false)
const submittingHandCard = ref(false)
const handCardInput = ref('')
const handCardError = ref('')
const pendingBindUsageId = ref(null)
const pendingJump = ref(null)

const handCardReturnInput = ref('')
const returnQuerying = ref(false)
const returnConfirming = ref(false)
const returnError = ref('')
const returnSuccessText = ref('')
const returnQueryUsage = ref(null)

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
  if (verifying.value) return

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
    const res = await cardApi.scanVerify(code)
    const action = res?.data?.data?.action || 'verify'
    const usageId = res?.data?.data?.usage_id
    resultSuccess.value = true
    if (action === 'start') {
      const sid = res?.data?.data?.session_id
      resultText.value = replaceTerms(`起单成功！服务单#${sid ?? '-'}，已开始计时。`)
    } else {
      const remainTimes = res?.data?.data?.remain_times
      resultText.value = `核销成功！剩余次数: ${remainTimes ?? '-'}`
    }

    // 回到 dashboard 并切到对应 tab
    const backTab = action === 'start' ? getReturnTab() : getReturnTab()
    console.log('扫码成功，将在30秒后跳转到', backTab, 'tab')
    
    const returnPath = getReturnPath()
    const jump = () => {
      router.replace({ path: returnPath, query: { tab: backTab } })
    }

    // 开启手牌 + 核销成功：先跳回上一页，再由上一页弹窗分配手牌
    if (action === 'verify' && supportHandCard.value && usageId) {
      if (jumpTimer) {
        clearTimeout(jumpTimer)
        jumpTimer = null
      }
      router.replace({
        path: returnPath,
        query: {
          tab: backTab,
          hand_card_usage_id: usageId
        }
      })
      return
    }

    // 清除之前的定时器（如果有）
    if (jumpTimer) {
      clearTimeout(jumpTimer)
    }
    jumpTimer = setTimeout(() => {
      jump()
    }, 1000)
  } catch (err) {
    resultSuccess.value = false
    const errorMsg = err.response?.data?.error || '扫码失败'
    resultText.value = errorMsg

    // 直接跳回 dashboard 并带上错误信息（避免 back + replace 导致 Dashboard 不刷新）
    const backTab = getReturnTab()
    const returnPath = getReturnPath()
    router.replace({ path: returnPath, query: { error: errorMsg, tab: backTab } })
  } finally {
    verifying.value = false
  }
}

const closeHandCardModal = () => {
  if (!showHandCardModal.value) return
  const no = String(handCardInput.value || '').trim()
  if (!no) {
    if (!confirm('你尚未分配手牌，确定关闭吗？')) return
  }

  showHandCardModal.value = false
  pendingBindUsageId.value = null
  const jump = pendingJump.value
  pendingJump.value = null
  if (jumpTimer) {
    clearTimeout(jumpTimer)
    jumpTimer = null
  }
  setTimeout(() => {
    jump?.()
  }, 200)
}

const onHandCardMaskClick = () => {
  closeHandCardModal()
}

const clearHandCardReturn = () => {
  handCardReturnInput.value = ''
  returnQueryUsage.value = null
  returnError.value = ''
  returnSuccessText.value = ''
}

const queryHandCardReturn = async () => {
  if (returnQuerying.value) return
  returnError.value = ''
  returnSuccessText.value = ''
  const no = String(handCardReturnInput.value || '').trim()
  if (!no) return
  returnQuerying.value = true
  try {
    const res = await cardApi.queryHandCardForReturn(no)
    returnQueryUsage.value = res?.data?.data || null
    if (!returnQueryUsage.value) {
      returnError.value = '未找到待归还记录'
    }
  } catch (e) {
    returnQueryUsage.value = null
    returnError.value = e?.response?.data?.error || '查询失败'
  } finally {
    returnQuerying.value = false
  }
}

const confirmReturnHandCard = async () => {
  if (returnConfirming.value) return
  returnError.value = ''
  returnSuccessText.value = ''
  const usage = returnQueryUsage.value
  const no = String(usage?.hand_card_no || handCardReturnInput.value || '').trim()
  if (!no) return
  if (!confirm('确定已归还该手牌吗？')) return

  returnConfirming.value = true
  try {
    await cardApi.returnHandCard(no)
    returnSuccessText.value = '归还成功'
    setTimeout(() => {
      clearHandCardReturn()
    }, 800)
  } catch (e) {
    returnError.value = e?.response?.data?.error || '归还失败'
  } finally {
    returnConfirming.value = false
  }
}

const submitHandCard = async (doBind) => {
  if (submittingHandCard.value) return
  handCardError.value = ''

  const usageId = pendingBindUsageId.value
  if (!usageId) {
    showHandCardModal.value = false
    pendingJump.value?.()
    pendingJump.value = null
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
      await cardApi.bindUsageHandCard(usageId, no)
    }
    showHandCardModal.value = false
    pendingBindUsageId.value = null
    const jump = pendingJump.value
    pendingJump.value = null
    if (jumpTimer) {
      clearTimeout(jumpTimer)
      jumpTimer = null
    }
    setTimeout(() => {
      jump?.()
    }, 200)
  } catch (e) {
    handCardError.value = e?.response?.data?.error || '绑定手牌失败'
  } finally {
    submittingHandCard.value = false
  }
}

onMounted(() => {
  mode.value = String(route.query.mode || 'verify')
  pageTitle.value = isStartOnlyMode() ? replaceTerms('扫码起单') : replaceTerms('扫码核销')

  const token = getMerchantToken()
  const id = getMerchantId()
  if (!token || !id) {
    router.replace('/login')
    return
  }

  loadCameras()
  // 默认自动启动一次
  start()

  merchantApi.getCurrentMerchant().then(res => {
    const m = res?.data?.data || {}
    supportHandCard.value = !!m.support_hand_card
  }).catch(() => {
    supportHandCard.value = false
  })
})

onUnmounted(() => {
  stop()
  if (jumpTimer) {
    clearTimeout(jumpTimer)
    jumpTimer = null
  }
})
</script>
