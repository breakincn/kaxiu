<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b">
      <div class="flex items-center gap-2">
        <span class="text-primary font-bold text-xl">卡包</span>
        <span class="text-gray-400 text-xs">kabao.me</span>
      </div>
      <router-link to="/user/cards" class="text-sm text-gray-500 hover:text-primary">
        切换用户端
      </router-link>
    </header>

    <!-- 商户信息 -->
    <div class="px-4 py-5 bg-white border-b">
      <h1 class="text-xl font-bold text-gray-800">{{ merchant.name }}</h1>
      <p class="text-gray-500 text-sm mt-1">管理后台</p>
    </div>

    <!-- 数据统计卡片 -->
    <div class="px-4 py-4 grid grid-cols-2 gap-4">
      <div class="bg-primary-light rounded-xl p-4">
        <div class="text-gray-600 text-sm mb-1">今日核销</div>
        <div class="text-3xl font-bold text-primary">{{ todayVerifyCount }}</div>
        <div class="text-gray-500 text-sm">次</div>
      </div>
      <div class="bg-secondary-light rounded-xl p-4">
        <div class="text-gray-600 text-sm mb-1">待处理预约</div>
        <div class="text-3xl font-bold text-secondary">{{ pendingAppointments }}</div>
        <div class="text-gray-500 text-sm">人</div>
      </div>
    </div>

    <!-- Tab 切换 -->
    <div class="px-4 flex gap-2 border-b bg-white">
      <button
        @click="currentTab = 'queue'"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'queue'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        排队管理
      </button>
      <button
        @click="currentTab = 'verify'"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'verify'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        快捷核销
      </button>
      <button
        @click="currentTab = 'notice'"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'notice'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        通知管理
      </button>
    </div>

    <!-- 排队管理 -->
    <div v-if="currentTab === 'queue'" class="px-4 py-4 space-y-4">
      <div v-for="appt in appointments" :key="appt.id" class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex justify-between items-start mb-2">
          <div>
            <div class="font-medium text-gray-800">用户 ID: {{ appt.user?.nickname || appt.user_id }}</div>
            <div class="text-gray-500 text-sm">预约时间: {{ appt.appointment_time }}</div>
          </div>
          <span :class="getStatusBadgeClass(appt.status)">
            {{ getStatusText(appt.status) }}
          </span>
        </div>
        
        <div class="flex gap-2 mt-3">
          <button
            v-if="appt.status === 'pending'"
            @click="confirmAppointment(appt.id)"
            class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
          >
            确认预约
          </button>
          <button
            v-if="appt.status === 'confirmed'"
            @click="finishAppointment(appt.id)"
            class="flex-1 py-2 bg-green-500 text-white rounded-lg text-sm font-medium"
          >
            完成服务 (扣次)
          </button>
          <button
            v-if="appt.status !== 'finished' && appt.status !== 'canceled'"
            @click="cancelAppointment(appt.id)"
            class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
          >
            取消
          </button>
        </div>
      </div>

      <div v-if="appointments.length === 0" class="text-center py-12 text-gray-400">
        暂无预约
      </div>
    </div>

    <!-- 快捷核销 -->
    <div v-if="currentTab === 'verify'" class="px-4 py-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <h3 class="font-medium text-gray-800 mb-4">输入核销码</h3>
        <input
          v-model="verifyCodeInput"
          type="text"
          placeholder="请输入用户的核销码"
          class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
        />
        <button
          @click="verifyCard"
          :disabled="!verifyCodeInput || verifying"
          class="w-full mt-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
        >
          {{ verifying ? '核销中...' : '确认核销' }}
        </button>
        
        <div v-if="verifyResult" class="mt-4 p-4 rounded-lg" :class="verifyResult.success ? 'bg-green-50' : 'bg-red-50'">
          <p :class="verifyResult.success ? 'text-green-600' : 'text-red-600'">
            {{ verifyResult.message }}
          </p>
        </div>
      </div>

      <!-- 今日核销记录 -->
      <div class="bg-white rounded-xl p-4 shadow-sm mt-4">
        <h3 class="font-medium text-gray-800 mb-4">今日核销记录</h3>
        <div v-if="todayUsages.length > 0" class="space-y-3">
          <div v-for="usage in todayUsages" :key="usage.id" class="flex justify-between items-center py-2 border-b last:border-0">
            <div>
              <div class="text-gray-800">{{ usage.card?.user?.nickname || '用户' }}</div>
              <div class="text-gray-400 text-sm">{{ usage.used_at }}</div>
            </div>
            <span class="text-green-500 text-sm">核销 {{ usage.used_times }} 次</span>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
          今日暂无核销
        </div>
      </div>
    </div>

    <!-- 通知管理 -->
    <div v-if="currentTab === 'notice'" class="px-4 py-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <h3 class="font-medium text-gray-800 mb-4">发布通知</h3>
        <div v-if="notices.length >= 3" class="mb-3 p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-700 text-sm">
          <p>已达到最大限制（3条），请先删除一条通知后再发布</p>
        </div>
        <input
          v-model="noticeForm.title"
          type="text"
          placeholder="通知标题"
          :disabled="notices.length >= 3"
          class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary mb-3 disabled:bg-gray-100"
        />
        <textarea
          v-model="noticeForm.content"
          placeholder="通知内容"
          rows="4"
          :disabled="notices.length >= 3"
          class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary resize-none disabled:bg-gray-100"
        ></textarea>
        <button
          @click="publishNotice"
          :disabled="!noticeForm.title || !noticeForm.content || notices.length >= 3"
          class="w-full mt-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
        >
          发布通知
        </button>
      </div>

      <!-- 历史通知 -->
      <div class="bg-white rounded-xl p-4 shadow-sm mt-4">
        <h3 class="font-medium text-gray-800 mb-4">已发布通知 ({{ notices.length }}/3)</h3>
        <div v-if="notices.length > 0" class="space-y-4">
          <div v-for="notice in notices" :key="notice.id" class="border-l-2 pl-3 relative" :class="notice.is_pinned ? 'border-yellow-500 bg-yellow-50' : 'border-primary'">
            <div class="flex items-start justify-between gap-2">
              <div class="flex-1">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-gray-800">{{ notice.title }}</span>
                  <span v-if="notice.is_pinned" class="px-2 py-0.5 bg-yellow-500 text-white text-xs rounded">置顶</span>
                </div>
                <div class="text-gray-500 text-sm mt-1">{{ notice.content }}</div>
                <div class="text-gray-400 text-xs mt-1">{{ notice.created_at }}</div>
              </div>
              <div class="flex flex-col gap-2">
                <button
                  @click="togglePin(notice.id)"
                  class="px-3 py-1 text-xs rounded"
                  :class="notice.is_pinned ? 'bg-gray-100 text-gray-600' : 'bg-yellow-100 text-yellow-600'"
                >
                  {{ notice.is_pinned ? '取消置顶' : '置顶' }}
                </button>
                <button
                  @click="deleteNotice(notice.id)"
                  class="px-3 py-1 bg-red-100 text-red-600 text-xs rounded"
                >
                  删除
                </button>
              </div>
            </div>
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
import { ref, onMounted, watch } from 'vue'
import { merchantApi, cardApi, appointmentApi, noticeApi, usageApi } from '../../api'

const merchantId = 1
const merchant = ref({})
const currentTab = ref('queue')

const todayVerifyCount = ref(0)
const pendingAppointments = ref(0)
const appointments = ref([])
const todayUsages = ref([])
const notices = ref([])

const verifyCodeInput = ref('')
const verifying = ref(false)
const verifyResult = ref(null)

const noticeForm = ref({
  title: '',
  content: ''
})

const fetchMerchant = async () => {
  try {
    const res = await merchantApi.getMerchant(merchantId)
    merchant.value = res.data.data
  } catch (err) {
    console.error('获取商户信息失败:', err)
  }
}

const fetchQueueStatus = async () => {
  try {
    const res = await merchantApi.getQueueStatus(merchantId)
    todayVerifyCount.value = res.data.data.today_verify_count || 0
    pendingAppointments.value = res.data.data.pending_appointments || 0
  } catch (err) {
    console.error('获取队列状态失败:', err)
  }
}

const fetchAppointments = async () => {
  try {
    const res = await appointmentApi.getMerchantAppointments(merchantId)
    appointments.value = (res.data.data || []).filter(a => a.status !== 'finished' && a.status !== 'canceled')
  } catch (err) {
    console.error('获取预约列表失败:', err)
  }
}

const fetchTodayUsages = async () => {
  try {
    const res = await usageApi.getMerchantUsages(merchantId)
    const today = new Date().toISOString().split('T')[0]
    todayUsages.value = (res.data.data || []).filter(u => u.used_at && u.used_at.startsWith(today))
  } catch (err) {
    console.error('获取核销记录失败:', err)
  }
}

const fetchNotices = async () => {
  try {
    const res = await noticeApi.getMerchantNotices(merchantId)
    notices.value = res.data.data || []
  } catch (err) {
    console.error('获取通知列表失败:', err)
  }
}

const confirmAppointment = async (id) => {
  try {
    await appointmentApi.confirmAppointment(id)
    fetchAppointments()
    fetchQueueStatus()
  } catch (err) {
    alert(err.response?.data?.error || '确认失败')
  }
}

const finishAppointment = async (id) => {
  try {
    await appointmentApi.finishAppointment(id)
    fetchAppointments()
    fetchQueueStatus()
  } catch (err) {
    alert(err.response?.data?.error || '完成失败')
  }
}

const cancelAppointment = async (id) => {
  try {
    await appointmentApi.cancelAppointment(id)
    fetchAppointments()
    fetchQueueStatus()
  } catch (err) {
    alert(err.response?.data?.error || '取消失败')
  }
}

const verifyCard = async () => {
  if (!verifyCodeInput.value || verifying.value) return
  
  verifying.value = true
  verifyResult.value = null
  
  try {
    const res = await cardApi.verifyCard(verifyCodeInput.value)
    verifyResult.value = {
      success: true,
      message: `核销成功！剩余次数: ${res.data.data.remain_times}`
    }
    verifyCodeInput.value = ''
    fetchQueueStatus()
    fetchTodayUsages()
  } catch (err) {
    verifyResult.value = {
      success: false,
      message: err.response?.data?.error || '核销失败'
    }
  } finally {
    verifying.value = false
  }
}

const publishNotice = async () => {
  if (!noticeForm.value.title || !noticeForm.value.content || notices.value.length >= 3) return
  
  try {
    await noticeApi.createNotice({
      merchant_id: merchantId,
      title: noticeForm.value.title,
      content: noticeForm.value.content
    })
    noticeForm.value = { title: '', content: '' }
    fetchNotices()
    alert('发布成功')
  } catch (err) {
    alert(err.response?.data?.error || '发布失败')
  }
}

const deleteNotice = async (id) => {
  if (!confirm('确定要删除这条通知吗？')) return
  
  try {
    await noticeApi.deleteNotice(id)
    fetchNotices()
    alert('删除成功')
  } catch (err) {
    alert(err.response?.data?.error || '删除失败')
  }
}

const togglePin = async (id) => {
  try {
    await noticeApi.togglePinNotice(id)
    fetchNotices()
  } catch (err) {
    alert(err.response?.data?.error || '操作失败')
  }
}

const getStatusBadgeClass = (status) => {
  const classes = {
    pending: 'px-2 py-1 rounded text-xs font-medium bg-orange-100 text-primary',
    confirmed: 'px-2 py-1 rounded text-xs font-medium bg-blue-100 text-blue-600',
    finished: 'px-2 py-1 rounded text-xs font-medium bg-green-100 text-green-600',
    canceled: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }
  return classes[status] || ''
}

const getStatusText = (status) => {
  const texts = {
    pending: '待确认',
    confirmed: '排队中',
    finished: '已完成',
    canceled: '已取消'
  }
  return texts[status] || status
}

watch(currentTab, (tab) => {
  if (tab === 'queue') {
    fetchAppointments()
  } else if (tab === 'verify') {
    fetchTodayUsages()
  } else if (tab === 'notice') {
    fetchNotices()
  }
})

onMounted(() => {
  fetchMerchant()
  fetchQueueStatus()
  fetchAppointments()
})
</script>
