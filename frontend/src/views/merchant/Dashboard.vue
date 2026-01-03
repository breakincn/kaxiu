<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b">
      <div class="flex items-center gap-2">
        <span class="text-primary font-bold text-xl">卡包</span>
        <span class="text-gray-400 text-xs">kabao.me</span>
      </div>
      <div class="flex items-center gap-3">
        <router-link to="/merchant/scan-card" class="p-1 text-gray-500 hover:text-primary">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4h-1a2 2 0 00-2 2v1m0 10v1a2 2 0 002 2h1m10-16h1a2 2 0 012 2v1m0 10v1a2 2 0 01-2 2h-1"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 11h8m-8 4h8"/>
          </svg>
        </router-link>
        <router-link to="/merchant/settings" class="p-1 text-gray-500 hover:text-primary">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
        </router-link>
      </div>
    </header>

    <!-- 商户信息 -->
    <div class="px-4 py-5 bg-white border-b">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h1 class="text-xl font-bold text-gray-800">{{ merchant.name }}</h1>
          <p class="text-gray-500 text-sm mt-1">管理后台</p>
        </div>
        <div class="flex gap-2">
          <router-link
            v-if="merchant.support_direct_sale"
            to="/merchant/shop-manage"
            class="px-3 py-2 bg-slate-600 text-white rounded-lg text-sm font-medium hover:bg-slate-700 transition-colors"
          >
            售卡管理
          </router-link>
          <router-link
            to="/merchant/issue-card"
            class="px-3 py-2 bg-slate-600 text-white rounded-lg text-sm font-medium hover:bg-slate-700 transition-colors"
          >
            发卡/开卡
          </router-link>
        </div>
      </div>
    </div>

    <!-- 数据统计卡片 -->
    <div class="px-4 py-4 grid grid-cols-3 gap-3">
      <button
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-gray-100"
        @click="goToDirectPurchaseOrders"
      >
        <div class="text-gray-600 text-sm mb-1">待确认订单</div>
        <div class="text-3xl font-bold" :class="pendingDirectPurchases > 0 ? 'text-red-500' : 'text-gray-400'">{{ pendingDirectPurchases }}</div>
        <div class="text-gray-500 text-sm">单</div>
      </button>
      <button
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-gray-100"
        @click="currentTab = 'queue'"
      >
        <div class="text-gray-600 text-sm mb-1">待处理预约</div>
        <div class="text-3xl font-bold" :class="pendingAppointments > 0 ? 'text-orange-500' : 'text-gray-400'">{{ pendingAppointments }}</div>
        <div class="text-gray-500 text-sm">人</div>
      </button>
      <button
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-gray-100"
        @click="currentTab = 'verify'"
      >
        <div class="text-gray-600 text-sm mb-1">今日核销</div>
        <div class="text-3xl font-bold" :class="todayVerifyCount > 0 ? 'text-secondary' : 'text-gray-400'">{{ todayVerifyCount }}</div>
        <div class="text-gray-500 text-sm">次</div>
      </button>
    </div>

    <!-- Tab 切换 -->
    <div class="px-4 flex gap-2 border-b bg-white">
      <button
        v-if="merchant.support_appointment"
        @click="currentTab = 'queue'"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'queue'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        排队
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
        扫码核销
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
        通知
      </button>
      <button
        @click="currentTab = 'cards'"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'cards'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        卡片
      </button>
    </div>

    <!-- 排队管理 -->
    <div v-if="currentTab === 'queue'" class="px-4 py-4 space-y-4">
      <div v-for="appt in appointments" :key="appt.id" class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex justify-between items-start mb-2">
          <div>
            <div class="font-medium text-gray-800">用户 ID: {{ appt.user?.nickname || appt.user_id }}</div>
            <div class="text-gray-500 text-sm">预约时间: {{ formatDateTime(appt.appointment_time) }}</div>
            <!-- 待确认预约的倒计时 -->
            <div v-if="appt.status === 'pending' && getPendingCountdown(appt) !== null" :class="getPendingCountdownClass(appt)">
              {{ getPendingCountdownDisplay(appt) }}
            </div>
            <!-- 已确认预约的倒计时 -->
            <div v-if="appt.status === 'confirmed' && getAppointmentCountdown(appt) !== null && !isServiceTimeExpired(appt)" :class="getCountdownClass(appt)">
              {{ getCountdownDisplay(appt) }}
            </div>
          </div>
          <span :class="getStatusBadgeClass(appt)">
            {{ getStatusText(appt) }}
          </span>
        </div>
        
        <div class="flex gap-2 mt-3">
          <button
            v-if="appt.status === 'pending' && !isPendingExpired(appt)"
            @click="confirmAppointment(appt.id)"
            class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
          >
            确认预约
          </button>
          <button
            v-if="appt.status === 'pending' && isPendingExpired(appt)"
            disabled
            class="flex-1 py-2 bg-gray-100 text-gray-400 rounded-lg text-sm font-medium cursor-not-allowed"
          >
            未确认预约
          </button>
          <button
            v-if="shouldShowFinishButton(appt) && !isWriteOffExpired(appt)"
            @click="finishAppointment(appt.id)"
            class="flex-1 py-2 bg-gray-900 text-white rounded-lg text-sm font-medium"
          >
            完成服务 (扣次)
          </button>
          <button
            v-if="appt.status === 'confirmed' && isWriteOffExpired(appt)"
            disabled
            class="flex-1 py-2 bg-gray-100 text-gray-400 rounded-lg text-sm font-medium cursor-not-allowed"
          >
            未核销
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

    <!-- 扫码核销 -->
    <div v-if="currentTab === 'verify'" class="px-4 py-4">
      <!-- 默认显示大按钮 -->
      <div v-if="!showVerifyInput" class="bg-white rounded-xl p-4 shadow-sm">
        <button
          @click="goScanVerify"
          class="w-full py-3 bg-primary text-white rounded-lg font-medium"
        >
          扫码核销
        </button>
      </div>

      <!-- 输入核销码区域 -->
      <div v-else class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-medium text-gray-800">输入核销码</h3>
          <button
            @click="showVerifyInput = false; verifyCodeInput = ''; verifyResult = null"
            class="text-gray-500 hover:text-gray-700 text-sm"
          >
            取消
          </button>
        </div>
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
        
        <div v-if="verifyResult" class="mt-4 p-4 rounded-lg" :class="verifyResult.success ? 'bg-primary-light' : 'bg-gray-50'">
          <p :class="verifyResult.success ? 'text-primary' : 'text-gray-700'">
            {{ verifyResult.message }}
          </p>
        </div>
      </div>

      <!-- 今日核销记录 -->
      <div class="bg-white rounded-xl p-4 shadow-sm mt-4">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-medium text-gray-800">今日核销记录</h3>
          <button
            v-if="!showVerifyInput"
            @click="showVerifyInput = true"
            class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium"
          >
            输入核销码
          </button>
          <button
            v-else
            @click="showVerifyInput = false; verifyCodeInput = ''; verifyResult = null"
            class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium"
          >
            扫码核销
          </button>
        </div>
        <div v-if="todayUsages.length > 0" class="space-y-3">
          <div v-for="usage in todayUsages" :key="usage.id" class="flex justify-between items-center py-2 border-b last:border-0">
            <div>
              <div class="text-gray-800">{{ usage.card?.user?.nickname || '用户' }}</div>
              <div class="text-gray-400 text-sm">{{ formatDateTime(usage.used_at) }}</div>
            </div>
            <span class="text-gray-700 text-sm">核销 {{ usage.used_times }} 次</span>
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
        <div v-if="notices.length >= 3" class="mb-3 p-3 bg-primary-light border border-gray-100 rounded-lg text-gray-700 text-sm">
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
          <div v-for="notice in notices" :key="notice.id" class="border-l-2 pl-3 relative" :class="notice.is_pinned ? 'border-primary bg-primary-light' : 'border-primary'">
            <div class="flex items-start justify-between gap-2">
              <div class="flex-1">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-gray-800">{{ notice.title }}</span>
                  <span v-if="notice.is_pinned" class="px-2 py-0.5 bg-primary-light text-primary text-xs rounded">置顶</span>
                </div>
                <div class="text-gray-500 text-sm mt-1">{{ notice.content }}</div>
                <div class="text-gray-400 text-xs mt-1">{{ formatDateTime(notice.created_at) }}</div>
              </div>
              <div class="flex flex-col gap-2">
                <button
                  @click="togglePin(notice.id)"
                  class="px-3 py-1 text-xs rounded"
                  :class="notice.is_pinned ? 'bg-gray-100 text-gray-600' : 'bg-primary-light text-primary'"
                >
                  {{ notice.is_pinned ? '取消置顶' : '置顶' }}
                </button>
                <button
                  @click="deleteNotice(notice.id)"
                  class="px-3 py-1 bg-gray-100 text-gray-700 text-xs rounded"
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

    <!-- 卡片管理 -->
    <div v-if="currentTab === 'cards'" class="px-4 py-4">
      <div v-if="cardsError" class="bg-gray-50 border border-gray-100 text-gray-700 rounded-lg p-3 text-sm mb-4">
        {{ cardsError }}
      </div>

      <div class="bg-white rounded-xl p-4 shadow-sm mb-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <input
            v-model="cardSearch.phone"
            class="border border-gray-200 rounded-lg px-3 py-2 text-sm"
            placeholder="按手机号搜索"
          />
          <input
            v-model="cardSearch.nickname"
            class="border border-gray-200 rounded-lg px-3 py-2 text-sm"
            placeholder="按用户名(昵称)搜索"
          />
          <input
            v-model="cardSearch.card_no"
            class="border border-gray-200 rounded-lg px-3 py-2 text-sm"
            placeholder="按卡号搜索"
          />
          <input
            v-model="cardSearch.card_type"
            class="border border-gray-200 rounded-lg px-3 py-2 text-sm"
            placeholder="按卡片类型搜索"
          />
        </div>

        <div class="flex gap-2 mt-3">
          <button
            @click="searchCards"
            class="px-4 py-2 bg-primary text-white text-sm rounded-lg"
          >
            查询
          </button>
          <button
            @click="resetCardSearch"
            class="px-4 py-2 bg-gray-100 text-gray-700 text-sm rounded-lg"
          >
            重置
          </button>
        </div>
      </div>

      <div v-if="cardsLoading" class="text-center py-12 text-gray-400">
        加载中...
      </div>

      <div v-else>
        <div v-for="(card, index) in issuedCards" :key="card.id" class="mb-6">
          <div
            @click="toggleCardExpand(card.id)"
            :class="[
              'rounded-2xl p-4 cursor-pointer transition-transform active:scale-[0.98]',
              'kb-card'
            ]"
          >
            <div class="flex justify-between items-start mb-1">
              <div>
                <h3 class="text-lg font-bold">{{ card.user?.nickname || card.user_id }}</h3>
                <p class="text-gray-500 text-xs mt-0.5">{{ card.card_type }}</p>
              </div>
              <div class="bg-gray-100 px-2.5 py-0.5 rounded-full">
                <span class="text-xs font-medium">NO: {{ card.card_no || '-' }}</span>
              </div>
            </div>

            <div class="flex justify-between items-end mt-6">
              <div>
                <div class="text-gray-500 text-xs mb-0.5">剩余次数</div>
                <div class="text-5xl font-bold leading-none">{{ card.remain_times }}</div>
              </div>
              <div class="text-right">
                <div class="text-gray-500 text-xs mb-0.5">有效期至</div>
                <div class="text-sm font-medium">{{ formatDate(card.end_date) }}</div>
              </div>
            </div>
          </div>

          <div v-if="expandedCardId === card.id" class="mt-3 bg-gray-50 rounded-2xl p-5 shadow-md border border-gray-200">
            <div class="space-y-3.5">
              <div class="flex justify-between">
                <span class="text-gray-500">用户</span>
                <span class="text-gray-800">{{ card.user?.nickname || card.user_id }}</span>
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

        <div v-if="issuedCards.length === 0" class="text-center py-12 text-gray-400">
          暂无已发卡
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { merchantApi, cardApi, appointmentApi, noticeApi, usageApi, shopApi } from '../../api'
import { formatDateTime, formatDate } from '../../utils/dateFormat'

const router = useRouter()
const route = useRoute()
const merchantId = ref(null)
const merchant = ref({})
const currentTab = ref('queue')

const todayVerifyCount = ref(0)
const pendingAppointments = ref(0)
const pendingDirectPurchases = ref(0)
const appointments = ref([])
const todayUsages = ref([])
const notices = ref([])
const currentTime = ref(Date.now())
let countdownTimer = null

const verifyCodeInput = ref('')
const verifying = ref(false)
const verifyResult = ref(null)
const showVerifyInput = ref(false)

const noticeForm = ref({
  title: '',
  content: ''
})

const issuedCards = ref([])
const cardsLoading = ref(false)
const cardsError = ref('')
const expandedCardId = ref(null)

const cardSearch = ref({
  phone: '',
  nickname: '',
  card_no: '',
  card_type: ''
})

const goScanVerify = () => {
  router.push('/merchant/scan-verify')
}

const goToDirectPurchaseOrders = () => {
  router.push({ path: '/merchant/shop-manage', query: { tab: 'orders' } })
}

const fetchMerchant = async () => {
  try {
    const res = await merchantApi.getMerchant(merchantId.value)
    merchant.value = res.data.data
  } catch (err) {
    console.error('获取商户信息失败:', err)
  }
}

const searchCards = async () => {
  expandedCardId.value = null
  await fetchIssuedCards()
}

const resetCardSearch = async () => {
  cardSearch.value = { phone: '', nickname: '', card_no: '', card_type: '' }
  expandedCardId.value = null
  await fetchIssuedCards()
}

const fetchQueueStatus = async () => {
  try {
    const res = await merchantApi.getQueueStatus(merchantId.value)
    todayVerifyCount.value = res.data.data.today_verify_count || 0
    pendingAppointments.value = res.data.data.pending_appointments || 0
  } catch (err) {
    console.error('获取队列状态失败:', err)
  }
}

const fetchPendingDirectPurchases = async () => {
  if (!merchant.value.support_direct_sale) {
    pendingDirectPurchases.value = 0
    return
  }
  try {
    const res = await shopApi.getMerchantDirectPurchases()
    const list = res.data.data || []
    pendingDirectPurchases.value = list.filter(o => o && o.status === 'paid').length
  } catch (err) {
    console.error('获取待确认订单失败:', err)
  }
}

const fetchAppointments = async () => {
  if (!merchant.value.support_appointment) {
    appointments.value = []
    return
  }
  try {
    const res = await appointmentApi.getMerchantAppointments(merchantId.value)
    appointments.value = (res.data.data || []).filter(a => a.status !== 'finished' && a.status !== 'canceled')
  } catch (err) {
    console.error('获取预约列表失败:', err)
  }
}

const fetchTodayUsages = async () => {
  try {
    const res = await usageApi.getMerchantUsages(merchantId.value)
    const today = new Date().toISOString().split('T')[0]
    todayUsages.value = (res.data.data || []).filter(u => u.used_at && u.used_at.startsWith(today))
  } catch (err) {
    console.error('获取核销记录失败:', err)
  }
}

const fetchNotices = async () => {
  try {
    const res = await noticeApi.getMerchantNotices(merchantId.value)
    notices.value = res.data.data || []
  } catch (err) {
    console.error('获取通知列表失败:', err)
  }
}

const fetchIssuedCards = async () => {
  if (!merchantId.value) return
  if (cardsLoading.value) return

  cardsLoading.value = true
  cardsError.value = ''
  try {
    const params = {}
    if (cardSearch.value.phone) params.phone = cardSearch.value.phone
    if (cardSearch.value.nickname) params.nickname = cardSearch.value.nickname
    if (cardSearch.value.card_no) params.card_no = cardSearch.value.card_no
    if (cardSearch.value.card_type) params.card_type = cardSearch.value.card_type

    const res = await cardApi.getMerchantCards(merchantId.value, params)
    let cardsList = res.data.data || []
    
    // 排序：先按创建时间降序，再按最近使用时间降序
    cardsList.sort((a, b) => {
      // 先按 created_at 降序排列
      const createTimeA = new Date(a.created_at).getTime()
      const createTimeB = new Date(b.created_at).getTime()
      if (createTimeA !== createTimeB) {
        return createTimeB - createTimeA
      }
      // 如果创建时间相同，按 last_used_at 降序排列
      const lastUsedA = a.last_used_at ? new Date(a.last_used_at).getTime() : 0
      const lastUsedB = b.last_used_at ? new Date(b.last_used_at).getTime() : 0
      return lastUsedB - lastUsedA
    })
    
    issuedCards.value = cardsList
  } catch (err) {
    cardsError.value = err.response?.data?.error || '获取卡片列表失败'
  } finally {
    cardsLoading.value = false
  }
}

const toggleCardExpand = (cardId) => {
  expandedCardId.value = expandedCardId.value === cardId ? null : cardId
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
  if (!confirm('确定要取消这个预约吗？此操作不可撤销。')) {
    return
  }
  
  try {
    await appointmentApi.cancelAppointment(id)
    fetchAppointments()
    fetchQueueStatus()
    alert('预约已取消')
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
    
    // 核销成功后2秒关闭输入框
    setTimeout(() => {
      showVerifyInput.value = false
      verifyResult.value = null
    }, 2000)
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
      merchant_id: merchantId.value,
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

const getStatusBadgeClass = (appt) => {
  if (!appt) return ''

  if (appt.status === 'pending' && isPendingExpired(appt)) {
    return 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }

  if (appt.status === 'confirmed' && isWriteOffExpired(appt)) {
    return 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }

  const classes = {
    pending: 'px-2 py-1 rounded text-xs font-medium bg-primary-light text-primary',
    confirmed: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-700',
    finished: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-700',
    canceled: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }
  return classes[appt.status] || ''
}

const getStatusText = (appt) => {
  if (!appt) return ''

  if (appt.status === 'pending' && isPendingExpired(appt)) {
    return '过期未确认'
  }

  if (appt.status === 'confirmed' && isWriteOffExpired(appt)) {
    return '已过服务时间'
  }

  const texts = {
    pending: '待确认',
    confirmed: '排队中',
    finished: '已完成',
    canceled: '已取消'
  }
  return texts[appt.status] || appt.status
}

const isPendingExpired = (appt) => {
  if (!appt || appt.status !== 'pending' || !appt.appointment_time) return false
  const appointmentTime = new Date(appt.appointment_time).getTime()
  return currentTime.value > appointmentTime
}

const isWriteOffExpired = (appt) => {
  if (!appt || appt.status !== 'confirmed' || !appt.appointment_time) return false

  const appointmentTime = new Date(appt.appointment_time).getTime()
  let serviceMinutes = merchant.value.avg_service_minutes
  if (!serviceMinutes || serviceMinutes <= 0) serviceMinutes = 30
  const deadlineMs = appointmentTime + (serviceMinutes + 30) * 60 * 1000
  return currentTime.value > deadlineMs
}

// 判断是否已过服务时间（不显示倒计时）
const isServiceTimeExpired = (appt) => {
  if (!appt || appt.status !== 'confirmed' || !appt.appointment_time) return false

  const appointmentTime = new Date(appt.appointment_time).getTime()
  let serviceMinutes = merchant.value.avg_service_minutes
  if (!serviceMinutes || serviceMinutes <= 0) serviceMinutes = 30
  const serviceDeadlineMs = appointmentTime + serviceMinutes * 60 * 1000
  return currentTime.value > serviceDeadlineMs
}

// 计算待确认预约倒计时（秒）
const getPendingCountdown = (appt) => {
  if (!appt || appt.status !== 'pending' || !appt.appointment_time) return null
  const appointmentTime = new Date(appt.appointment_time).getTime()
  const now = currentTime.value
  return Math.floor((appointmentTime - now) / 1000)
}

// 获取待确认预约倒计时显示文本
const getPendingCountdownDisplay = (appt) => {
  const countdown = getPendingCountdown(appt)
  if (countdown === null) return ''
  
  if (countdown <= 0) {
    return '预约时间已过'
  }
  
  const totalSeconds = Math.abs(countdown)
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

// 获取待确认预约倒计时颜色类
const getPendingCountdownClass = (appt) => {
  const countdown = getPendingCountdown(appt)
  if (countdown === null || countdown <= 0) {
    return 'text-gray-400 text-sm font-medium mt-1'
  }

  // 预约时间临近时用主色提示，其余用弱化文本（避免红绿灯）
  if (countdown <= 600) {
    return 'text-primary text-sm font-medium mt-1'
  }
  return 'text-gray-500 text-sm font-medium mt-1'
}

// 计算预约倒计时（秒）
const getAppointmentCountdown = (appt) => {
  if (!appt || !appt.appointment_time) return null
  const appointmentTime = new Date(appt.appointment_time).getTime()
  const now = currentTime.value
  return Math.floor((appointmentTime - now) / 1000)
}

// 获取倒计时显示文本
const getCountdownDisplay = (appt) => {
  const countdown = getAppointmentCountdown(appt)
  if (countdown === null) return ''
  
  // 预约时间已过，显示服务时间倒计时
  if (countdown <= 0) {
    const elapsed = Math.abs(countdown)
    const hours = Math.floor(elapsed / 3600)
    const minutes = Math.floor((elapsed % 3600) / 60)
    const seconds = elapsed % 60
    
    let timeText = ''
    if (hours > 0) {
      timeText = `${hours}小时${minutes}分${seconds}秒`
    } else if (minutes > 0) {
      timeText = `${minutes}分${seconds}秒`
    } else {
      timeText = `${seconds}秒`
    }
    
    return `已服务时间 ${timeText}`
  }
  
  // 预约时间未到，显示倒计时
  const totalSeconds = Math.abs(countdown)
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

const getCountdownClass = (appt) => {
  if (isServiceTimeExpired(appt)) {
    return 'text-gray-400 text-sm font-medium mt-1'
  }
  
  const countdown = getAppointmentCountdown(appt)
  if (countdown === null) return 'text-primary text-sm font-medium mt-1'

  // 统一色相：临近用主色，其余用灰
  if (countdown <= 600) {
    return 'text-primary text-sm font-medium mt-1'
  }
  return 'text-gray-500 text-sm font-medium mt-1'
}

// 判断是否应该显示完成服务按钮
const shouldShowFinishButton = (appt) => {
  if (appt.status !== 'confirmed') return false
  if (!appt.appointment_time) return false
  
  const appointmentTime = new Date(appt.appointment_time).getTime()
  const now = currentTime.value
  const elapsed = now - appointmentTime // 已过的时间（毫秒）
  
  // 需要过了预约时间 + 服务时长 - 1分钟 才显示按钮
  let serviceMinutes = merchant.value.avg_service_minutes
  if (!serviceMinutes || serviceMinutes <= 0) serviceMinutes = 30
  const requiredTime = (serviceMinutes - 1) * 60 * 1000
  
  return elapsed >= requiredTime
}

// 启动倒计时定时器
const startCountdownTimer = () => {
  stopCountdownTimer()
  countdownTimer = setInterval(() => {
    currentTime.value = Date.now()
  }, 1000)
}

// 停止倒计时定时器
const stopCountdownTimer = () => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

watch(currentTab, (tab) => {
  if (tab === 'queue') {
    fetchAppointments()
    startCountdownTimer()
  } else {
    stopCountdownTimer()
    if (tab === 'verify') {
      // 重置为默认状态
      showVerifyInput.value = false
      verifyCodeInput.value = ''
      verifyResult.value = null
      fetchTodayUsages()
    } else if (tab === 'notice') {
      fetchNotices()
    } else if (tab === 'cards') {
      fetchIssuedCards()
    }
  }
})

onMounted(() => {
  // 在页面最顶部打印一个标记，确保能看到
  document.title = 'Dashboard Loaded'
  console.error('=== Dashboard MOUNTED ===')
  console.log('Dashboard mounted, checking localStorage...')
  console.log('localStorage merchantToken:', localStorage.getItem('merchantToken'))
  console.log('localStorage merchantId:', localStorage.getItem('merchantId'))
  
  // 检查查询参数，自动切换到指定Tab
  const tabParam = route.query.tab
  if (tabParam && ['queue', 'verify', 'notice', 'cards'].includes(tabParam)) {
    currentTab.value = tabParam
  }
  
  const storedMerchantId = localStorage.getItem('merchantId')
  if (!storedMerchantId) {
    console.log('No merchantId found, redirecting to login')
    router.replace('/merchant/login')
    return
  }

  const parsedMerchantId = Number.parseInt(storedMerchantId, 10)
  if (Number.isNaN(parsedMerchantId) || parsedMerchantId <= 0) {
    console.log('Invalid merchantId:', storedMerchantId, 'redirecting to login')
    router.replace('/merchant/login')
    return
  }

  console.log('Valid merchantId:', parsedMerchantId, 'loading data...')
  merchantId.value = parsedMerchantId
  fetchMerchant().then(() => {
    if (!merchant.value.support_appointment && currentTab.value === 'queue') {
      currentTab.value = 'verify'
    }
    fetchQueueStatus()
    fetchPendingDirectPurchases()
    fetchAppointments()
    if (merchant.value.support_appointment) {
      startCountdownTimer()
    }
  })
})

onUnmounted(() => {
  stopCountdownTimer()
})
</script>
