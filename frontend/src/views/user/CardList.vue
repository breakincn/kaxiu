<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b">
      <div class="flex items-center gap-2">
        <span class="text-primary font-bold text-xl">卡包</span>
        <span class="text-gray-400 text-xs">kabao.me</span>
      </div>
      <router-link to="/merchant" class="text-sm text-gray-500 hover:text-primary">
        切换商户端
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
        @click="goToDetail(card.id)"
        :class="[
          'rounded-2xl p-5 text-white cursor-pointer transition-transform active:scale-[0.98]',
          index % 2 === 0 ? 'card-gradient-orange' : 'card-gradient-blue'
        ]"
      >
        <div class="flex justify-between items-start mb-2">
          <div>
            <h3 class="text-lg font-bold">{{ card.merchant?.name }}</h3>
            <p class="text-white/80 text-sm">{{ card.card_type }}</p>
          </div>
          <div class="text-right">
            <div class="text-4xl font-bold">{{ card.remain_times }}</div>
            <div class="text-white/80 text-xs">剩余次数</div>
          </div>
        </div>
        <div class="text-white/60 text-xs mb-2">卡号: {{ card.card_no }}</div>
        <div class="flex justify-between items-center mt-4">
          <div class="flex items-center gap-1">
            <span class="w-2 h-2 rounded-full" :class="getStatusColor(card)"></span>
            <span class="text-sm text-white/80">有效期至 {{ card.end_date }}</span>
          </div>
          <div class="flex items-center gap-1 text-white font-medium">
            立即使用
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
            </svg>
          </div>
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
import { cardApi } from '../../api'

const router = useRouter()
const userName = ref('张三')
const currentStatus = ref('active')
const cards = ref([])
const userId = 1

const fetchCards = async () => {
  try {
    const res = await cardApi.getUserCards(userId, currentStatus.value)
    cards.value = res.data.data || []
  } catch (err) {
    console.error('获取卡片失败:', err)
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

onMounted(fetchCards)
</script>
