<template>
  <div class="min-h-screen bg-gray-50 pb-6">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">卡片详情</span>
    </header>

    <!-- 卡片详情 -->
    <div class="px-4 mt-4">
      <div class="bg-gray-50 rounded-2xl p-5 shadow-md border border-gray-200">
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
            <span class="text-gray-800">{{ card.recharge_at }} / ¥{{ card.recharge_amount }}</span>
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
            <span class="text-gray-800">{{ card.last_used_at || '未使用' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">有效期</span>
            <span class="text-gray-800">{{ card.start_date }} 至 {{ card.end_date }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 预约排队区域（如果支持） -->
    <div v-if="card.merchant?.support_appointment" class="px-4 mt-4">
      <div class="bg-blue-50 rounded-2xl p-5 shadow-md border border-blue-100">
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
          <div class="text-primary font-medium text-lg">{{ appointment.appointment_time }}</div>
          <div class="grid grid-cols-2 gap-4 pt-3 mt-3 border-t border-blue-200">
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
        </div>
        
        <div v-else class="text-center text-gray-400 py-4">
          暂无预约
        </div>
      </div>
    </div>

    <!-- 核销码区域 -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="text-center text-gray-600 mb-3">到店出示核销码</div>
        <button
          @click="generateCode"
          :disabled="generating || card.remain_times <= 0"
          class="w-full py-3 border-2 border-primary text-primary font-medium rounded-lg hover:bg-primary-light disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {{ generating ? '生成中...' : (verifyCode ? verifyCode : '生成核销码') }}
        </button>
        <p v-if="codeExpireTime" class="text-center text-gray-400 text-sm mt-2">
          有效期至 {{ codeExpireTime }}
        </p>
      </div>
    </div>

    <!-- 使用记录 -->
    <div class="px-4 mt-4">
      <div class="bg-green-50 rounded-2xl p-5 shadow-md border border-green-100">
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
                <span class="text-gray-400 text-sm">{{ usage.used_at }}</span>
              </div>
            </div>
            <span :class="usage.status === 'success' ? 'text-green-500' : 'text-red-500'" class="text-sm font-medium">
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
    <div class="px-4 mt-4">
      <div class="bg-orange-50 rounded-2xl p-5 shadow-md border border-orange-100">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
          </svg>
          <span class="font-medium text-gray-800">商户通知</span>
        </div>
        <div v-if="notices.length > 0" class="space-y-4">
          <div v-for="notice in notices" :key="notice.id" class="bg-white rounded-lg p-3 shadow-sm" :class="notice.is_pinned ? 'border-l-4 border-yellow-500' : 'border-l-4 border-primary'">
            <div class="flex items-center gap-2 mb-1">
              <span class="font-medium text-gray-800">{{ notice.title }}</span>
              <span v-if="notice.is_pinned" class="px-2 py-0.5 bg-yellow-500 text-white text-xs rounded">置顶</span>
            </div>
            <div class="text-gray-500 text-sm mt-1">{{ notice.content }}</div>
            <div class="text-gray-400 text-xs mt-1">{{ notice.created_at }}</div>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
          暂无通知
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { cardApi, usageApi, noticeApi, appointmentApi } from '../../api'

const router = useRouter()
const route = useRoute()

const card = ref({})
const usages = ref([])
const notices = ref([])
const appointment = ref(null)
const queueBefore = ref(0)
const estimatedMinutes = ref(0)

const verifyCode = ref('')
const codeExpireTime = ref('')
const generating = ref(false)

const goBack = () => {
  router.back()
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
  } catch (err) {
    console.error('获取通知失败:', err)
  }
}

const fetchAppointment = async () => {
  try {
    const res = await appointmentApi.getCardAppointment(route.params.id)
    if (res.data.data) {
      appointment.value = res.data.data.appointment
      queueBefore.value = res.data.data.queue_before || 0
      estimatedMinutes.value = res.data.data.estimated_minutes || 0
    }
  } catch (err) {
    console.error('获取预约信息失败:', err)
  }
}

const generateCode = async () => {
  if (generating.value || card.value.remain_times <= 0) return
  
  generating.value = true
  try {
    const res = await cardApi.generateVerifyCode(route.params.id)
    verifyCode.value = res.data.data.code
    const expireAt = new Date(res.data.data.expire_at * 1000)
    codeExpireTime.value = expireAt.toLocaleTimeString()
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generating.value = false
  }
}

const getAppointmentStatusClass = (status) => {
  const classes = {
    pending: 'text-primary',
    confirmed: 'text-blue-500',
    finished: 'text-green-500',
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

onMounted(fetchCard)
</script>
