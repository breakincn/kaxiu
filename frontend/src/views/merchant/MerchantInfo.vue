<template>
  <div class="min-h-screen bg-gray-50 pb-6">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">商家信息设置</span>
    </header>

    <!-- 营业时间设置 -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <span class="font-medium text-gray-800">营业时间</span>
        </div>

        <!-- 全天营业 -->
        <div class="mb-4">
          <label class="flex items-center gap-2 mb-2">
            <input type="checkbox" v-model="useAllDay" @change="handleAllDayToggle" class="w-4 h-4 text-primary">
            <span class="text-sm text-gray-700">全天营业（不分时间段）</span>
          </label>
          <div v-if="useAllDay" class="grid grid-cols-2 gap-3">
            <div>
              <label class="text-xs text-gray-500 mb-1 block">开始时间</label>
              <input
                v-model="form.all_day_start"
                type="time"
                class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              />
            </div>
            <div>
              <label class="text-xs text-gray-500 mb-1 block">结束时间</label>
              <input
                v-model="form.all_day_end"
                type="time"
                class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              />
            </div>
          </div>
        </div>

        <!-- 分时段营业 -->
        <div v-if="!useAllDay" class="space-y-4">
          <!-- 上午时间段 -->
          <div>
            <label class="text-sm font-medium text-gray-700 mb-2 block">上午营业时间</label>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="text-xs text-gray-500 mb-1 block">开始</label>
                <input
                  v-model="form.morning_start"
                  type="time"
                  class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>
              <div>
                <label class="text-xs text-gray-500 mb-1 block">结束</label>
                <input
                  v-model="form.morning_end"
                  type="time"
                  class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>
            </div>
          </div>

          <!-- 下午时间段 -->
          <div>
            <label class="text-sm font-medium text-gray-700 mb-2 block">下午营业时间</label>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="text-xs text-gray-500 mb-1 block">开始</label>
                <input
                  v-model="form.afternoon_start"
                  type="time"
                  class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>
              <div>
                <label class="text-xs text-gray-500 mb-1 block">结束</label>
                <input
                  v-model="form.afternoon_end"
                  type="time"
                  class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>
            </div>
          </div>

          <!-- 晚上时间段 -->
          <div>
            <label class="text-sm font-medium text-gray-700 mb-2 block">晚上营业时间</label>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="text-xs text-gray-500 mb-1 block">开始</label>
                <input
                  v-model="form.evening_start"
                  type="time"
                  class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>
              <div>
                <label class="text-xs text-gray-500 mb-1 block">结束</label>
                <input
                  v-model="form.evening_end"
                  type="time"
                  class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 地址设置 -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          <span class="font-medium text-gray-800">商家地址</span>
        </div>

        <div class="space-y-3">
          <!-- 省市区 -->
          <div class="grid grid-cols-3 gap-2">
            <div>
              <label class="text-xs text-gray-500 mb-1 block">省份</label>
              <input
                v-model="form.province"
                type="text"
                placeholder="如：浙江省"
                class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
              />
            </div>
            <div>
              <label class="text-xs text-gray-500 mb-1 block">城市</label>
              <input
                v-model="form.city"
                type="text"
                placeholder="如：杭州市"
                class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
              />
            </div>
            <div>
              <label class="text-xs text-gray-500 mb-1 block">区县</label>
              <input
                v-model="form.district"
                type="text"
                placeholder="如：西湖区"
                class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
              />
            </div>
          </div>

          <!-- 详细地址 -->
          <div>
            <label class="text-xs text-gray-500 mb-1 block">详细地址</label>
            <input
              v-model="form.address"
              type="text"
              placeholder="街道门牌号，如：西溪路1号"
              class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:border-primary text-sm"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 保存按钮 -->
    <div class="px-4 mt-6">
      <button
        @click="saveInfo"
        :disabled="saving"
        class="w-full py-3 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {{ saving ? '保存中...' : '保存设置' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

import { getMerchantId } from '../../utils/auth'

const router = useRouter()
const saving = ref(false)
const useAllDay = ref(false)

const form = ref({
  morning_start: '',
  morning_end: '',
  afternoon_start: '',
  afternoon_end: '',
  evening_start: '',
  evening_end: '',
  all_day_start: '',
  all_day_end: '',
  province: '',
  city: '',
  district: '',
  address: ''
})

const goBack = () => {
  router.back()
}

const handleAllDayToggle = () => {
  if (useAllDay.value) {
    // 切换到全天营业，清空分时段
    form.value.morning_start = ''
    form.value.morning_end = ''
    form.value.afternoon_start = ''
    form.value.afternoon_end = ''
    form.value.evening_start = ''
    form.value.evening_end = ''
  } else {
    // 切换到分时段，清空全天
    form.value.all_day_start = ''
    form.value.all_day_end = ''
  }
}

const fetchMerchantInfo = async () => {
  try {
    const merchantId = getMerchantId()
    if (!merchantId) {
      router.push('/login')
      return
    }

    const res = await merchantApi.getMerchant(merchantId)
    const data = res.data.data

    // 判断是全天营业还是分时段
    if (data.all_day_start || data.all_day_end) {
      useAllDay.value = true
      form.value.all_day_start = data.all_day_start || ''
      form.value.all_day_end = data.all_day_end || ''
    } else {
      useAllDay.value = false
      form.value.morning_start = data.morning_start || ''
      form.value.morning_end = data.morning_end || ''
      form.value.afternoon_start = data.afternoon_start || ''
      form.value.afternoon_end = data.afternoon_end || ''
      form.value.evening_start = data.evening_start || ''
      form.value.evening_end = data.evening_end || ''
    }

    form.value.province = data.province || ''
    form.value.city = data.city || ''
    form.value.district = data.district || ''
    form.value.address = data.address || ''
  } catch (err) {
    console.error('获取商户信息失败:', err)
  }
}

const saveInfo = async () => {
  if (saving.value) return

  saving.value = true
  try {
    await merchantApi.updateMerchantInfo(form.value)
    alert('保存成功')
    router.back()
  } catch (err) {
    alert(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchMerchantInfo()
})
</script>
