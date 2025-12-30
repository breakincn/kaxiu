<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b">
      <div class="flex items-center gap-2">
        <span class="text-primary font-bold text-xl">卡包</span>
        <span class="text-gray-400 text-xs">kabao.me</span>
      </div>
      <router-link to="/user/settings" class="p-1 text-gray-500 hover:text-primary">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
        </svg>
      </router-link>
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
        <h2 class="text-lg font-bold text-gray-800">卡片包</h2>
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
        v-for="(card, index) in cards"
        :key="card.id"
      >
        <div
          @click="goToDetail(card.id)"
          :class="[
            'rounded-2xl p-4 text-white cursor-pointer transition-transform active:scale-[0.98]',
            index % 2 === 0 ? 'card-gradient-orange' : 'card-gradient-blue'
          ]"
        >
          <!-- 顶部：商户名称和版本标签 -->
          <div class="flex justify-between items-start mb-1">
            <div>
              <h3 class="text-lg font-bold">{{ card.merchant?.name }}</h3>
              <p class="text-white/70 text-xs mt-0.5">{{ card.card_type }}</p>
            </div>
            <div class="bg-white/20 px-2.5 py-0.5 rounded-full">
              <span class="text-xs font-medium">NO: G12345678981189</span>
            </div>
          </div>

          <!-- 底部：剩余次数和有效期 -->
          <div class="flex justify-between items-end mt-6">
            <div>
              <div class="text-white/70 text-xs mb-0.5">剩余次数</div>
              <div class="text-5xl font-bold leading-none">{{ card.remain_times }}</div>
            </div>
            <div class="text-right">
              <div class="text-white/70 text-xs mb-0.5">有效期至</div>
              <div class="text-sm font-medium">{{ formatDate(card.end_date) }}</div>
            </div>
          </div>
        </div>

        <!-- 置顶通知 -->
        <div 
          v-if="card.pinnedNotice" 
          class="mt-2 bg-yellow-50 border border-yellow-200 rounded-lg p-3 cursor-pointer"
          @click="goToDetail(card.id)"
        >
          <div class="flex items-center gap-2 mb-1">
            <svg class="w-4 h-4 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
            </svg>
            <span class="text-yellow-800 font-medium text-sm">{{ card.pinnedNotice.title }}</span>
            <span class="px-1.5 py-0.5 bg-yellow-500 text-white text-xs rounded">置顶</span>
          </div>
          <div class="text-yellow-700 text-xs line-clamp-1">{{ card.pinnedNotice.content }}</div>
        </div>
      </div>

      <div v-if="cards.length === 0" class="text-center py-12 text-gray-400">
        暂无{{ currentStatus === 'active' ? '有效' : '失效' }}卡片
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { cardApi, noticeApi } from '../../api'
import { formatDate } from '../../utils/dateFormat'

const router = useRouter()
const userName = ref('')
const currentStatus = ref('active')
const cards = ref([])
const userId = ref(null)

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

watch(currentStatus, fetchCards)

onMounted(() => {
  initUser()
  fetchCards()
})
</script>
