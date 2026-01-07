<template>
  <div class="min-h-screen bg-gray-50 pb-6">
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1 text-gray-600">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">卡片详情</span>
      <div class="w-8"></div>
    </header>

    <div class="px-4 mt-4 space-y-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div v-if="loading" class="text-center text-gray-400 py-8">
          加载中...
        </div>

        <div v-else>
          <div class="flex items-start justify-between">
            <div>
              <div class="text-lg font-bold text-gray-800">{{ card.merchant?.name || '商户' }}</div>
              <div class="text-gray-500 text-xs mt-1">{{ card.card_type }}</div>
            </div>
            <div class="bg-gray-100 px-3 py-1 rounded-full">
              <span class="text-xs font-medium text-gray-700">NO: {{ card.card_no || '-' }}</span>
            </div>
          </div>

          <div class="flex justify-between items-end mt-6">
            <div>
              <div class="text-gray-500 text-xs mb-1">剩余次数</div>
              <div class="text-5xl font-bold leading-none text-gray-900">{{ card.remain_times }}</div>
            </div>
            <div class="text-right">
              <div class="text-gray-500 text-xs mb-1">有效期至</div>
              <div class="text-sm font-medium text-gray-800">{{ formatDate(card.end_date) }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="space-y-3.5">
          <div class="flex justify-between">
            <span class="text-gray-500">用户</span>
            <span class="text-gray-800">{{ card.user?.nickname || card.user?.phone || card.user_id }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">卡类型</span>
            <span class="text-gray-800">{{ card.card_type }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">开卡/充值</span>
            <span class="text-gray-800">{{ formatDateTime(card.recharge_at) }} / ¥{{ card.recharge_amount || 0 }}</span>
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

      <div class="bg-white rounded-xl p-4 shadow-sm">
        <button
          @click="goScanVerify"
          class="w-full py-3 bg-orange-500 text-white rounded-lg font-medium"
        >
          扫码核销
        </button>
        <p class="text-gray-400 text-xs mt-3">
          提示：请扫描用户端生成的核销二维码
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { cardApi } from '../../api'
import { formatDateTime, formatDate } from '../../utils/dateFormat'

import { getMerchantId, getMerchantToken } from '../../utils/auth'

const router = useRouter()
const route = useRoute()

const loading = ref(false)
const card = ref({})

const goBack = () => {
  router.back()
}

const fetchCard = async () => {
  loading.value = true
  try {
    const res = await cardApi.getMerchantCard(route.params.id)
    card.value = res.data.data
  } catch (err) {
    alert(err.response?.data?.error || '获取卡片失败')
  } finally {
    loading.value = false
  }
}

const goScanVerify = () => {
  router.push('/merchant/scan-verify')
}

onMounted(() => {
  const token = getMerchantToken()
  const id = getMerchantId()
  if (!token || !id) {
    router.replace('/login')
    return
  }
  fetchCard()
})
</script>
