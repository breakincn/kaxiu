<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">设置</span>
    </header>

    <!-- 设置列表 -->
    <div class="px-4 py-4 space-y-4">
      <!-- 账户相关 -->
      <div class="bg-white rounded-xl shadow-sm overflow-hidden">
        <button
          v-if="canServiceUpdate"
          @click="goToServices"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6 2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"/>
            </svg>
            <span class="text-gray-800 font-medium">开启服务</span>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>

        <button
          v-if="canMerchantInfoUpdate"
          @click="goToMerchantInfo"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <span class="text-gray-800 font-medium">商家信息设置</span>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>

        <button
          v-if="canCustomerServiceManage"
          @click="goToCustomerService"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18 10c0 3.314-2.686 6-6 6s-6-2.686-6-6 2.686-6 6-6 6 2.686 6 6z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-4.35-4.35"/>
            </svg>
            <span class="text-gray-800 font-medium">客服设置</span>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>

        <button
          @click="goToBindPhone"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z"/>
            </svg>
            <span class="text-gray-800 font-medium">绑定手机号</span>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>
        
        <button
          @click="handleLogout"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
            </svg>
            <span class="text-red-500 font-medium">退出登录</span>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import { ensureMerchantPermissionsLoaded, merchantApi } from '../../api'
import { clearMerchantAuth, clearMerchantPermissionKeys, hasMerchantPermission, getMerchantActiveAuth, getTechnicianShopSlug } from '../../utils/auth'

const router = useRouter()

const canServiceUpdate = computed(() => hasMerchantPermission('merchant.service.update'))
const canMerchantInfoUpdate = computed(() => hasMerchantPermission('merchant.merchant.update'))
const canCustomerServiceManage = computed(() => {
  return hasMerchantPermission('merchant.cs.manage') && !!merchant.value?.support_customer_service
})

const merchant = ref({})

onMounted(() => {
  ensureMerchantPermissionsLoaded()
  fetchMerchant()
})

const fetchMerchant = async () => {
  try {
    const res = await merchantApi.getCurrentMerchant()
    merchant.value = res.data?.data || {}
  } catch (e) {
    merchant.value = {}
  }
}

const goBack = () => {
  router.back()
}

const goToBindPhone = () => {
  router.push('/merchant/bind-phone')
}

const goToServices = () => {
  router.push('/merchant/services')
}

const goToMerchantInfo = () => {
  router.push('/merchant/merchant-info')
}

const goToCustomerService = () => {
  router.push('/merchant/customer-service')
}

const handleLogout = () => {
  if (confirm('确定要退出登录吗？')) {
    // 检查当前是否是员工登录
    const isActiveStaff = getMerchantActiveAuth() === 'staff'
    const staffSlug = getTechnicianShopSlug()
    
    clearMerchantPermissionKeys()
    clearMerchantAuth()
    
    // 根据登录类型跳转到对应页面
    if (isActiveStaff && staffSlug) {
      // 员工退出后跳转到员工登录页面
      router.push(`/s/${staffSlug}/login`)
    } else {
      // 商户退出后跳转到商户登录页面
      router.push('/login')
    }
  }
}
</script>
