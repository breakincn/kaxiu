<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b">
      <div class="flex items-center gap-2">
        <span class="text-primary font-bold text-xl">卡包</span>
        <span class="text-gray-400 text-xs">kabao.me</span>
      </div>
      <div class="flex items-center gap-3">
        <router-link to="/user/scan-pay" class="p-1 text-gray-500 hover:text-primary">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4h-1a2 2 0 00-2 2v1m0 10v1a2 2 0 002 2h1m10-16h1a2 2 0 012 2v1m0 10v1a2 2 0 01-2 2h-1"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 11h8m-8 4h8"/>
          </svg>
        </router-link>
        <router-link to="/user/settings" class="p-1 text-gray-500 hover:text-primary">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
        </router-link>
      </div>
    </header>

    <!-- 问候区域 -->
    <div class="px-4 py-6">
      <div class="flex items-center gap-3">
        <span class="text-3xl">👋</span>
        <div>
          <h1 class="text-xl font-bold text-gray-800">你好，{{ userName }}</h1>
          <p class="text-gray-500 text-sm">今天想去哪里享受服务？</p>
        </div>
      </div>
    </div>

    <!-- 卡片包标题和筛选 -->
    <div class="px-4 mb-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-bold text-gray-800">我的卡片</h2>
        <div class="flex gap-2">
          <button
            @click="currentStatus = 'active'"
            :class="[
              'px-4 py-1.5 rounded-full text-sm font-medium transition-all',
              currentStatus === 'active' 
                ? 'bg-primary text-white' 
                : 'bg-gray-100 text-gray-500'
            ]"
          >
            进行中
          </button>
          <button
            @click="currentStatus = 'expired'"
            :class="[
              'px-4 py-1.5 rounded-full text-sm font-medium transition-all',
              currentStatus === 'expired' 
                ? 'bg-gray-600 text-white' 
                : 'bg-gray-100 text-gray-500'
            ]"
          >
            已失效
          </button>
        </div>
      </div>
    </div>

    <!-- 卡片列表 -->
    <div class="px-4 pb-6 space-y-4">
      <div
        v-for="(item, index) in displayItems"
        :key="item._key"
      >
        <template v-if="item._type === 'card'">
          <div
            @click="goToDetail(item.id)"
            :class="[
              'rounded-2xl p-4 text-white cursor-pointer transition-transform active:scale-[0.98]',
              index % 2 === 0 ? 'card-gradient-orange' : 'card-gradient-blue'
            ]"
          >
            <!-- 顶部：商户名称和版本标签 -->
            <div class="flex justify-between items-start mb-1">
              <div>
                <h3 class="text-lg font-bold">{{ item.merchant?.name }}</h3>
                <p class="text-white/70 text-xs mt-0.5">{{ item.card_type }}</p>
              </div>
              <div class="bg-white/20 px-2.5 py-0.5 rounded-full">
                <span class="text-xs font-medium">NO: G12345678981189</span>
              </div>
            </div>

            <!-- 底部：剩余次数和有效期 -->
            <div class="flex justify-between items-end mt-6">
              <div>
                <div class="text-white/70 text-xs mb-0.5">剩余次数</div>
                <div class="text-5xl font-bold leading-none">{{ item.remain_times }}</div>
              </div>
              <div class="text-right">
                <div class="text-white/70 text-xs mb-0.5">有效期至</div>
                <div class="text-sm font-medium">{{ formatDate(item.end_date) }}</div>
              </div>
            </div>
          </div>

          <!-- 置顶通知（仅卡片显示） -->
          <div 
            v-if="item.pinnedNotice" 
            class="mt-2 bg-yellow-50 border border-yellow-200 rounded-lg p-3 cursor-pointer"
            @click="goToDetail(item.id)"
          >
            <div class="flex items-center gap-2 mb-1">
              <svg class="w-4 h-4 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
              </svg>
              <span class="text-yellow-800 font-medium text-sm">{{ item.pinnedNotice.title }}</span>
              <span class="px-1.5 py-0.5 bg-yellow-500 text-white text-xs rounded">置顶</span>
            </div>
            <div class="text-yellow-700 text-xs line-clamp-1">{{ item.pinnedNotice.content }}</div>
          </div>
        </template>

        <div
          v-else
          class="rounded-2xl p-4 card-gradient-yellow cursor-not-allowed"
        >
          <div class="flex justify-between items-start mb-2">
            <div>
              <h3 class="text-lg font-bold text-gray-800">{{ item.merchant_name }}</h3>
              <p class="text-gray-600 text-xs mt-0.5">{{ item.card_name }}</p>
            </div>
            <div class="bg-white/80 px-2.5 py-0.5 rounded-full">
              <span class="text-xs font-medium text-gray-700">NO: {{ item.order_no }}</span>
            </div>
          </div>

          <div class="flex justify-between items-end mt-6">
            <div class="flex items-end gap-2">
              <div>
                <div class="text-gray-600 text-xs mb-0.5">卡片状态</div>
                <div class="text-xl font-bold text-gray-800">待商家确认</div>
              </div>
              <div class="text-xs text-gray-600 pb-0.5">{{ formatPaymentMethod(item.payment_method) }}</div>
            </div>
            <div class="text-right">
              <div class="text-gray-600 text-xs mb-0.5">{{ formatDateTime(item.paid_at) }}</div>
              <div class="text-gray-600 text-xs mb-0.5">已付款 ¥{{ (item.price / 100).toFixed(2) }}</div>
              <div class="text-sm font-medium text-yellow-800">{{ formatElapsed(item.paid_at) }}</div>
            </div>
          </div>
        </div>

      </div>

      <div v-if="displayItems.length === 0" class="text-center py-12 text-gray-400">
        暂无{{ currentStatus === 'active' ? '有效' : '失效' }}卡片
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { cardApi, noticeApi, shopApi } from '../../api'
import { formatDate } from '../../utils/dateFormat'

const router = useRouter()
const userName = ref('')
const currentStatus = ref('active')
const cards = ref([])
const pendingPaidOrders = ref([])
const userId = ref(null)

const nowTick = ref(Date.now())
let nowTimer = null
let pollTimer = null

// 从 localStorage 获取当前用户信息
const initUser = () => {
  const storedUserId = localStorage.getItem('userId')
  const storedUserName = localStorage.getItem('userName')
  
  if (!storedUserId) {
    // 如果没有登录，跳转到登录页
    router.push('/user/login')
    return
  }

  userId.value = parseInt(storedUserId)
  userName.value = storedUserName || '用户'
}

const fetchPendingOrders = async () => {
  if (!userId.value) return
  if (currentStatus.value !== 'active') {
    pendingPaidOrders.value = []
    return
  }
  try {
    const res = await shopApi.getDirectPurchases()
    const list = res.data.data || []
    pendingPaidOrders.value = list.filter(o => o && o.status === 'paid')
  } catch (err) {
    console.error('获取待确认订单失败:', err)
    pendingPaidOrders.value = []
  }
}

const displayItems = computed(() => {
  const items = []
  for (const o of pendingPaidOrders.value || []) {
    items.push({
      _type: 'pending',
      _key: `pending-${o.order_no}`,
      order_no: o.order_no,
      paid_at: o.paid_at,
      payment_method: o.payment_method,
      price: o.price,
      merchant_name: o.merchant?.name || '商户',
      card_name: o.card_template?.name || '卡片'
    })
  }
  for (const c of cards.value || []) {
    items.push({ ...c, _type: 'card', _key: `card-${c.id}` })
  }
  return items
})

const fetchCards = async () => {
  if (!userId.value) return
  
  try {
    const res = await cardApi.getUserCards(userId.value, currentStatus.value)
    const cardsData = res.data.data || []
    
    // 为每个卡片获取对应商户的置顶通知
    for (const card of cardsData) {
      if (card.merchant_id) {
        try {
          const noticesRes = await noticeApi.getMerchantNotices(card.merchant_id, 3)
          const notices = noticesRes.data.data || []
          // 找到置顶通知
          card.pinnedNotice = notices.find(n => n.is_pinned) || null
        } catch (err) {
          console.error('获取通知失败:', err)
          card.pinnedNotice = null
        }
      }
    }
    
    cards.value = cardsData
  } catch (err) {
    console.error('获取卡片失败:', err)
    if (err.response?.status === 401) {
      // token 过期或无效，跳转到登录页
      router.push('/user/login')
    }
  }
}

const goToDetail = (id) => {
  router.push(`/user/cards/${id}`)
}

const getStatusColor = (card) => {
  const now = new Date()
  const endDate = new Date(card.end_date)
  if (endDate < now || card.remain_times <= 0) {
    return 'bg-red-400'
  }
  const thirtyDaysLater = new Date()
  thirtyDaysLater.setDate(thirtyDaysLater.getDate() + 30)
  if (endDate < thirtyDaysLater) {
    return 'bg-yellow-400'
  }
  return 'bg-green-400'
}

function formatElapsed(fromTime) {
  if (!fromTime) return ''
  const fromTs = new Date(fromTime).getTime()
  if (!fromTs) return ''
  const diff = Math.max(0, Math.floor((nowTick.value - fromTs) / 1000))
  const h = Math.floor(diff / 3600)
  const m = Math.floor((diff % 3600) / 60)
  const s = diff % 60
  if (h > 0) return `${h}小时${m}分${s}秒`
  if (m > 0) return `${m}分${s}秒`
  return `${s}秒`
}

function formatPaymentMethod(method) {
  if (!method) return '未知支付方式'
  const methodMap = {
    'wechat': '微信支付',
    'alipay': '支付宝',
    'unionpay': '银联支付',
    'cash': '现金支付'
  }
  return methodMap[method] || method
}

function formatDateTime(dateTime) {
  if (!dateTime) return ''
  const date = new Date(dateTime)
  if (isNaN(date.getTime())) return ''
  
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  
  return `${year}-${month}-${day} ${hours}:${minutes}`
}

watch(currentStatus, async () => {
  await fetchCards()
  await fetchPendingOrders()
})

onMounted(() => {
  initUser()
  fetchCards()
  fetchPendingOrders()

  nowTimer = setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)

  pollTimer = setInterval(() => {
    fetchCards()
    fetchPendingOrders()
  }, 8000)
})

onUnmounted(() => {
  if (nowTimer) {
    clearInterval(nowTimer)
    nowTimer = null
  }
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})
</script>
