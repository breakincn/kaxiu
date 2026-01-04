<template>
  <div class="min-h-screen bg-gray-50 pb-6">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">{{ card.merchant?.name || '卡片详情' }}</span>
    </header>

    <!-- 卡片详情 -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          <span class="font-medium text-gray-800">卡片详情</span>
        </div>
        <div class="space-y-3.5">
          <div class="flex justify-between">
            <span class="text-gray-500">商户名称</span>
            <span class="text-gray-800">{{ card.merchant?.name }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">卡类型</span>
            <span class="text-gray-800">{{ card.card_type }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">开卡/充值</span>
            <span class="text-gray-800">{{ formatDateTime(card.recharge_at) }} / ¥{{ card.recharge_amount }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">总次数</span>
            <span class="text-gray-800">{{ card.total_times }} 次</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">已使用</span>
            <span class="text-gray-800">{{ card.used_times }} 次</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">上次使用</span>
            <span class="text-gray-800">{{ card.last_used_at ? formatDateTime(card.last_used_at) : '未使用' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">有效期</span>
            <span class="text-gray-800">{{ formatDate(card.start_date) }} 至 {{ formatDate(card.end_date) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 预约排队区域（如果支持且不在冷却中） -->
    <div v-if="card.merchant?.support_appointment && !isInCooldown" class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center gap-2 mb-3">
          <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <span class="font-medium text-gray-800">预约排队</span>
        </div>
        
        <div v-if="appointment" class="space-y-3">
          <div class="flex justify-between items-center">
            <span class="text-gray-500">我的预约</span>
            <span :class="getAppointmentStatusClass(appointment.status)">
              {{ getAppointmentStatusText(appointment.status) }}
            </span>
          </div>
          <div class="flex justify-between items-center">
            <div :class="getAppointmentTimeClass()" class="font-medium text-lg">
              {{ formatDateTime(appointment.appointment_time) }}
            </div>
            <div class="text-right">
              <div v-if="appointment.status === 'confirmed'" class="text-sm text-gray-600 mb-1">排队中</div>
              <div v-if="!isAppointmentPassed()" :class="getCountdownClass()" class="text-sm font-medium">
                {{ getCountdownText() }}
              </div>
              <div v-else class="text-sm text-gray-400">
                预约已过
              </div>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-4 pt-3 mt-3 border-t border-gray-200">
            <div>
              <div class="text-gray-400 text-xs">前面排队</div>
              <div class="text-2xl font-bold text-gray-800">{{ queueBefore }}<span class="text-sm font-normal">人</span></div>
            </div>
            <div>
              <div class="text-gray-400 text-xs">预计等待</div>
              <div class="text-2xl font-bold text-gray-800">{{ estimatedMinutes }}<span class="text-sm font-normal">分钟</span></div>
            </div>
          </div>
          <p class="text-xs text-gray-400">* 排队进度由商户服务确认后即时更新</p>
          
          <!-- 取消预约按钮 -->
          <button
            @click="cancelAppointment"
            :disabled="cancelButtonDisabled"
            class="w-full py-2.5 border-2 border-red-400 text-red-500 font-medium rounded-lg hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors mt-3"
          >
            {{ cancelButtonText }}
          </button>
        </div>
        
        <div v-else class="text-center">
          <button
            @click="showAppointmentModal"
            :disabled="appointing || isInCooldown"
            class="w-full py-3 border-2 border-primary text-primary font-medium rounded-lg hover:bg-primary-light disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {{ isInCooldown ? cooldownButtonText : '我要预约' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 核销码区域 -->
    <div v-if="shouldShowVerifyCode()" class="px-4 mt-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="text-center text-gray-600 mb-3">到店出示核销码</div>
        <button
          @click="generateCode"
          :disabled="generating || card.remain_times <= 0"
          class="w-full py-3 border-2 border-primary text-primary font-medium rounded-lg hover:bg-primary-light disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {{ generating ? '生成中...' : (verifyCode ? verifyCode : '生成核销码') }}
        </button>

			<div v-if="verifyQrDataUrl" class="mt-4 flex justify-center">
				<img :src="verifyQrDataUrl" alt="核销二维码" class="w-48 h-48" />
			</div>
        <p v-if="codeExpireTime" class="text-center text-gray-400 text-sm mt-2">
          有效期至 {{ codeExpireTime }}
        </p>
      </div>
    </div>

    <!-- 使用记录 -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
            </svg>
            <span class="font-medium text-gray-800">使用记录</span>
          </div>
          <span class="text-gray-600 text-sm">
            总数{{ card.total_times }}次/剩余{{ card.remain_times }}次
          </span>
        </div>
        <div v-if="usages.length > 0" class="space-y-2">
          <div v-for="usage in usages" :key="usage.id" class="flex justify-between items-center p-3 bg-white rounded-lg shadow-sm">
            <div>
              <div class="text-gray-800">核销次数: {{ usage.used_times }}</div>
              <div class="flex items-center gap-2">
                <span class="text-gray-500 text-sm">{{ getWeekDay(usage.used_at) }}</span>
                <span class="text-gray-400 text-sm">{{ formatDateTime(usage.used_at) }}</span>
              </div>
            </div>
            <span :class="usage.status === 'success' ? 'text-primary' : 'text-red-500'" class="text-sm font-medium">
              {{ usage.status === 'success' ? '成功' : '失败' }}
            </span>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
          暂无使用记录
        </div>
      </div>
    </div>

    <!-- 商户通知 -->
    <div v-if="notices.length > 0" ref="noticeAnchor" class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
          </svg>
          <span class="font-medium text-gray-800">商户通知</span>
        </div>
        <div class="space-y-4">
          <div v-for="notice in notices" :key="notice.id" class="bg-white rounded-lg p-3 shadow-sm" :class="notice.is_pinned ? 'border-l-4 border-red-500 bg-red-50' : 'border-l-4 border-orange-500 bg-orange-50'">
            <div class="flex items-center gap-2 mb-1">
              <span class="font-medium text-gray-800">{{ notice.title }}</span>
              <span v-if="notice.is_pinned" class="px-2 py-0.5 bg-red-500 text-white text-xs rounded">置顶</span>
            </div>
            <div class="text-gray-500 text-sm mt-1">{{ notice.content }}</div>
            <div class="text-gray-400 text-xs mt-1">{{ formatDateTime(notice.created_at) }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 动态占位元素：确保页面可以滚动到通知区域 -->
    <div v-if="notices.length > 0" :style="{ height: getBottomSpacerHeight() }"></div>

    <!-- 预约时间选择弹窗 -->
    <div v-if="showModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="closeModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg max-h-[80vh] overflow-hidden">
        <!-- 弹窗头部 -->
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between">
          <h3 class="font-medium text-lg">选择预约时间</h3>
          <button @click="closeModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <!-- 日期选择 -->
        <div class="px-5 py-3 border-b">
          <div class="flex gap-2">
            <button
              @click="selectDate('today')"
              :class="selectedDate === getTodayDate() ? 'bg-primary text-white' : 'bg-gray-100 text-gray-700'"
              class="flex-1 py-2 px-4 rounded-lg font-medium transition-colors"
            >
              今天
            </button>
            <button
              @click="selectDate('tomorrow')"
              :class="selectedDate === getTomorrowDate() ? 'bg-primary text-white' : 'bg-gray-100 text-gray-700'"
              class="flex-1 py-2 px-4 rounded-lg font-medium transition-colors"
            >
              明天
            </button>
          </div>
        </div>

        <!-- 时间段列表 -->
        <div class="px-5 py-4 overflow-y-auto" style="max-height: 400px;">
          <div v-if="loadingSlots" class="text-center py-8 text-gray-400">
            加载中...
          </div>
          <div v-else-if="timeSlots.length === 0" class="text-center py-8 text-gray-400">
            今日无可用时间段
          </div>
          <div v-else class="grid grid-cols-2 gap-3">
            <button
              v-for="slot in timeSlots"
              :key="slot.time"
              @click="selectTimeSlot(slot)"
              :disabled="!slot.available"
              :class="{
                'bg-primary text-white': selectedTimeSlot === slot.time && slot.available,
                'bg-gray-100 text-gray-400 cursor-not-allowed': !slot.available,
                'bg-white border-2 border-gray-200 text-gray-700 hover:border-primary': slot.available && selectedTimeSlot !== slot.time
              }"
              class="py-3 px-4 rounded-lg font-medium transition-all"
            >
              <div>{{ formatTime(slot.time) }}</div>
              <div v-if="!slot.available" class="text-xs mt-1">已被预约</div>
            </button>
          </div>
        </div>

        <!-- 弹窗底部 -->
        <div class="px-5 py-4 border-t">
          <button
            @click="confirmAppointment"
            :disabled="!selectedTimeSlot || appointing"
            class="w-full py-3 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {{ appointing ? '预纤中...' : '确认预约' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { cardApi, usageApi, noticeApi, appointmentApi } from '../../api'
import { formatDateTime, formatDate } from '../../utils/dateFormat'
import QRCode from 'qrcode'

const router = useRouter()
const route = useRoute()

const card = ref({})
const usages = ref([])
const notices = ref([])
const appointment = ref(null)
const queueBefore = ref(0)
const estimatedMinutes = ref(0)
const countdown = ref(0)
let countdownTimer = null

const cooldownUntil = ref(null)

const verifyCode = ref('')
const codeExpireTime = ref('')
const generating = ref(false)
const verifyQrDataUrl = ref('')
let verifyExpireTimer = null
const appointing = ref(false)
const canceling = ref(false)

const noticeAnchor = ref(null)

// 预约弹窗相关
const showModal = ref(false)
const selectedDate = ref('')
const selectedTimeSlot = ref('')
const timeSlots = ref([])
const loadingSlots = ref(false)

const goBack = () => {
  router.push('/user/cards')
}

const fetchCard = async () => {
  try {
    const res = await cardApi.getCard(route.params.id)
    card.value = res.data.data
    
    if (card.value.merchant_id) {
      fetchNotices(card.value.merchant_id)
    }
    
    fetchUsages()
    fetchAppointment()
  } catch (err) {
    console.error('获取卡片详情失败:', err)
  }
}

const fetchUsages = async () => {
  try {
    const res = await usageApi.getCardUsages(route.params.id)
    usages.value = res.data.data || []
  } catch (err) {
    console.error('获取使用记录失败:', err)
  }
}

const fetchNotices = async (merchantId) => {
  try {
    const res = await noticeApi.getMerchantNotices(merchantId, 5)
    notices.value = res.data.data || []
    
    // 如果需要滚动到通知区域
    if (route.query.scrollToNotice === '1' && notices.value.length > 0) {
      await scrollToNotice()
    }
  } catch (err) {
    console.error('获取通知失败:', err)
  }
}

const fetchAppointment = async () => {
  try {
    console.log('正在获取预约信息，卡片ID:', route.params.id)
    const res = await appointmentApi.getCardAppointment(route.params.id)
    console.log('预约信息响应:', res.data)
    const data = res.data.data
    cooldownUntil.value = data?.cooldown_until || null

    if (data?.appointment) {
      appointment.value = data.appointment
      queueBefore.value = data.queue_before || 0
      estimatedMinutes.value = data.estimated_minutes || 0
      console.log('预约信息已设置:', appointment.value)
      // 启动倒计时
      startCountdownTimer()
      return
    }

    console.log('未找到预约信息')
    appointment.value = null
    queueBefore.value = 0
    estimatedMinutes.value = 0
    stopCountdownTimer()
  } catch (err) {
    console.error('获取预约信息失败:', err)
    console.error('错误详情:', err.response?.data)
  }
}

const getCooldownRemainingSeconds = () => {
  if (!cooldownUntil.value) return 0
  const until = new Date(cooldownUntil.value).getTime()
  const now = Date.now()
  return Math.max(0, Math.floor((until - now) / 1000))
}

const isInCooldown = computed(() => {
  return getCooldownRemainingSeconds() > 0
})

const cooldownButtonText = computed(() => {
  const seconds = getCooldownRemainingSeconds()
  if (seconds <= 0) return '我要预约'
  const minutes = Math.ceil(seconds / 60)
  return `冷却中（约${minutes}分钟后可预约）`
})

const isAppointmentFailed = computed(() => {
  if (!appointment.value || !appointment.value.appointment_time) return false
  if (appointment.value.status !== 'pending' && appointment.value.status !== 'confirmed') return false
  const appointmentTimeMs = new Date(appointment.value.appointment_time).getTime()
  const nowMs = Date.now()
  return nowMs - appointmentTimeMs >= 30 * 60 * 1000
})

const cancelButtonDisabled = computed(() => {
  if (!appointment.value) return true
  if (isAppointmentFailed.value) return true
  return canceling.value || appointment.value.status === 'finished' || appointment.value.status === 'canceled'
})

const cancelButtonText = computed(() => {
  if (isAppointmentFailed.value) return '预约失败'
  return canceling.value ? '取消中...' : '取消预约'
})

const generateCode = async () => {
  if (generating.value || card.value.remain_times <= 0) return
  
  generating.value = true
  try {
    const res = await cardApi.generateVerifyCode(route.params.id)
    verifyCode.value = res.data.data.code
    const expireAt = new Date(res.data.data.expire_at * 1000)
    codeExpireTime.value = expireAt.toLocaleTimeString()

		verifyQrDataUrl.value = await QRCode.toDataURL(verifyCode.value, {
			margin: 1,
			scale: 8,
			errorCorrectionLevel: 'M'
		})

		if (verifyExpireTimer) {
			clearTimeout(verifyExpireTimer)
			verifyExpireTimer = null
		}
		const delayMs = Math.max(0, expireAt.getTime() - Date.now())
		verifyExpireTimer = setTimeout(() => {
			verifyCode.value = ''
			codeExpireTime.value = ''
			verifyQrDataUrl.value = ''
			verifyExpireTimer = null
		}, delayMs)
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generating.value = false
  }
}

// 显示预约弹窗
const showAppointmentModal = async () => {
  if (!card.value || !card.value.merchant_id) {
    alert('卡片信息加载中，请稍后再试')
    return
  }
  
  showModal.value = true
  selectedDate.value = getTodayDate()
  selectedTimeSlot.value = ''
  await loadTimeSlots(selectedDate.value)
}

// 关闭弹窗
const closeModal = () => {
  showModal.value = false
  selectedDate.value = ''
  selectedTimeSlot.value = ''
  timeSlots.value = []
}

// 获取今天日期
const getTodayDate = () => {
  return new Date().toISOString().slice(0, 10)
}

// 获取明天日期
const getTomorrowDate = () => {
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  return tomorrow.toISOString().slice(0, 10)
}

// 选择日期
const selectDate = async (type) => {
  selectedDate.value = type === 'today' ? getTodayDate() : getTomorrowDate()
  selectedTimeSlot.value = ''
  await loadTimeSlots(selectedDate.value)
}

// 加载可用时间段
const loadTimeSlots = async (date) => {
  if (!card.value.merchant_id) {
    console.error('商户ID不存在')
    return
  }
  
  loadingSlots.value = true
  try {
    console.log('正在获取时间段，商户ID:', card.value.merchant_id, '日期:', date)
    const res = await appointmentApi.getAvailableTimeSlots(card.value.merchant_id, date)
    console.log('获取时间段响应:', res.data)
    timeSlots.value = res.data.data.time_slots || []
  } catch (err) {
    console.error('获取可用时间段失败:', err)
    console.error('错误详情:', err.response?.data)
    alert(`获取可用时间段失败: ${err.response?.data?.error || err.message}`)
  } finally {
    loadingSlots.value = false
  }
}

// 选择时间段
const selectTimeSlot = (slot) => {
  if (!slot.available) return
  selectedTimeSlot.value = slot.time
}

// 格式化时间显示
const formatTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  return `${hours}:${minutes}`
}

// 确认预约
const confirmAppointment = async () => {
  if (!selectedTimeSlot.value || appointing.value) return
  
  appointing.value = true
  try {
    const userId = localStorage.getItem('userId')
    if (!userId) {
      alert('请先登录')
      router.push('/user/login')
      return
    }
    
    await appointmentApi.createAppointment({
      merchant_id: card.value.merchant_id,
      user_id: parseInt(userId),
      appointment_time: selectedTimeSlot.value
    })
    
    closeModal()
    await fetchAppointment()
    alert('预约成功！')
  } catch (err) {
    alert(err.response?.data?.error || '预约失败')
  } finally {
    appointing.value = false
  }
}

const cancelAppointment = async () => {
  if (canceling.value) return

  if (isAppointmentFailed.value) return
  
  if (!confirm('确定要取消预约吗？')) return
  
  canceling.value = true
  try {
    await appointmentApi.cancelAppointment(appointment.value.id)
    
    appointment.value = null
    queueBefore.value = 0
    estimatedMinutes.value = 0
    stopCountdownTimer()
    
    alert('已取消预约')
  } catch (err) {
    alert(err.response?.data?.error || '取消预约失败')
  } finally {
    canceling.value = false
  }
}

const getAppointmentStatusClass = (status) => {
  const classes = {
    pending: 'text-primary',
    confirmed: 'text-primary',
    finished: 'text-gray-600',
    canceled: 'text-gray-400'
  }
  return classes[status] || 'text-gray-500'
}

const getAppointmentStatusText = (status) => {
  const texts = {
    pending: '待确认',
    confirmed: '排队中',
    finished: '已完成',
    canceled: '已取消'
  }
  return texts[status] || status
}

const getWeekDay = (dateStr) => {
  if (!dateStr) return ''
  const weekDays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  const date = new Date(dateStr)
  return weekDays[date.getDay()]
}

// 计算倒计时（秒）
const calculateCountdown = () => {
  if (!appointment.value || !appointment.value.appointment_time) return 0
  const appointmentTime = new Date(appointment.value.appointment_time).getTime()
  const now = Date.now()
  return Math.floor((appointmentTime - now) / 1000)
}

// 更新倒计时
const updateCountdown = () => {
  countdown.value = calculateCountdown()
}

// 启动倒计时定时器
const startCountdownTimer = () => {
  stopCountdownTimer()
  updateCountdown()
  countdownTimer = setInterval(updateCountdown, 1000)
}

// 停止倒计时定时器
const stopCountdownTimer = () => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

// 判断预约是否已过（超过预约时间1分钟）
const isAppointmentPassed = () => {
  return countdown.value < -60
}

// 获取预约时间的颜色类
const getAppointmentTimeClass = () => {
  if (countdown.value < -60) {
    return 'text-gray-400' // 超过1分钟，灰色
  } else if (countdown.value > 0 && countdown.value <= 300) {
    return 'text-red-500' // 5分钟内，红色
  } else {
    return 'text-gray-800' // 默认使用正文色
  }
}

// 获取倒计时文字颜色类
const getCountdownClass = () => {
  if (countdown.value > 0 && countdown.value <= 300) {
    return 'text-red-500' // 5分钟内，红色
  } else {
    return 'text-gray-600' // 默认中性色
  }
}

// 获取倒计时文字
const getCountdownText = () => {
  if (countdown.value <= 0 && countdown.value > -60) {
    return '预约时间已到'
  }
  
  const totalSeconds = Math.abs(countdown.value)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  
  if (hours > 0) {
    return `${hours}小时${minutes}分${seconds}秒`
  } else if (minutes > 0) {
    return `${minutes}分${seconds}秒`
  } else {
    return `${seconds}秒`
  }
}

// 判断是否应该显示核销码区域
const shouldShowVerifyCode = () => {
  // 如果没有预约，显示核销码
  if (!appointment.value) {
    return true
  }
  
  // 如果有预约，判断条件
  // 1. 预约状态必须是已确认(confirmed)
  // 2. 当前时间距离预约时间小于等于5分钟（即倒计时 <= 300秒 且 > -60秒）
  if (appointment.value.status === 'confirmed') {
    // countdown.value > 0 表示还没到预约时间
    // countdown.value <= 300 表示距离预约时间小于等于5分钟
    // countdown.value > -60 表示还没有超过预约时间1分钟
    return countdown.value <= 300 && countdown.value > -60
  }
  
  // 其他状态（pending, finished, canceled）不显示核销码
  return false
}

const getBottomSpacerHeight = () => {
  // 当有通知时，添加底部占位高度，确保可以滚动到通知区域
  const windowHeight = window.innerHeight || 800
  const estimatedContentHeight = 700 // 估算页面内容高度
  const minSpacerHeight = Math.max(windowHeight - estimatedContentHeight, 300)
  return `${minSpacerHeight}px`
}

const scrollToNotice = async () => {
  // 等待DOM更新，包括动态占位元素的渲染
  await nextTick()
  // 再次等待，确保占位元素高度计算完成
  await new Promise(resolve => setTimeout(resolve, 100))
  
  const el = noticeAnchor.value
  if (!el) return
  try {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } catch (_) {
    // ignore
  }
}

onMounted(async () => {
  await fetchCard()
  // 如果有预约，启动倒计时
  if (appointment.value) {
    startCountdownTimer()
  }
})

onUnmounted(() => {
  stopCountdownTimer()
	if (verifyExpireTimer) {
		clearTimeout(verifyExpireTimer)
		verifyExpireTimer = null
	}
})
</script>
