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
          v-if="canProjectManage"
          @click="goToProjectSettings"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
            </svg>
            <span class="text-gray-800 font-medium">项目设置</span>
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
          v-if="canAppointmentSettings"
          @click="goToAppointmentSettings"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10m-11 9h12a2 2 0 002-2V7a2 2 0 00-2-2H6a2 2 0 00-2 2v11a2 2 0 002 2z"/>
            </svg>
            <span class="text-gray-800 font-medium">预约设置</span>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>

        <button
          v-if="canQueueSettings"
          @click="goToQueueSettings"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h6m-6 4h10M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
            </svg>
            <span class="text-gray-800 font-medium">叫号设置</span>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>

        <button
          v-if="canHandCardManage"
          @click="goToHandCardSettings"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V8a2 2 0 00-2-2h-5m-4 0V5a2 2 0 114 0v1m-4 0a2 2 0 104 0m-5 8a2 2 0 100-4 2 2 0 000 4zm0 0c1.306 0 2.417.835 2.83 2M9 14a3.001 3.001 0 00-2.83 2M15 11h3m-3 4h2"/>
            </svg>
            <span class="text-gray-800 font-medium">手牌设置</span>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
          </svg>
        </button>

        <button
          v-if="canRoomNumberCardManage"
          @click="goToRoomNumberCardSettings"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/>
            </svg>
            <span class="text-gray-800 font-medium">房间号牌设置</span>
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
          v-if="isTechnicianSettings"
          @click="goToTechnicianPassword"
          class="w-full px-4 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors border-b border-gray-100"
        >
          <div class="flex items-center gap-3">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2h-1V9a5 5 0 00-10 0v2H6a2 2 0 00-2 2v6a2 2 0 002 2z"/>
            </svg>
            <span class="text-gray-800 font-medium">修改密码</span>
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

const canServiceUpdate = computed(() => hasMerchantPermission('merchant.service.manage'))
const canMerchantInfoUpdate = computed(() => hasMerchantPermission('merchant.info.manage'))
const canCustomerServiceManage = computed(() => {
  return hasMerchantPermission('merchant.cs.manage') && !!merchant.value?.support_customer_service
})

const canAppointmentSettings = computed(() => {
  return hasMerchantPermission('merchant.service.manage') && !!merchant.value?.support_appointment
})

const canProjectManage = computed(() => hasMerchantPermission('merchant.service.manage') && !!merchant.value?.support_project)
const canQueueSettings = computed(() => hasMerchantPermission('merchant.service.manage') && !!merchant.value?.support_queue)
const canHandCardManage = computed(() => hasMerchantPermission('merchant.service.manage') && !!merchant.value?.support_hand_card)
const canRoomNumberCardManage = computed(() => hasMerchantPermission('merchant.service.manage') && !!merchant.value?.support_room)

const merchant = ref({})
const isTechnicianSettings = computed(() => getMerchantActiveAuth() === 'staff')

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

const goToTechnicianPassword = () => {
  router.push('/merchant/technician-password')
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

const goToAppointmentSettings = () => {
  router.push('/merchant/appointment-settings')
}

const goToProjectSettings = () => {
  router.push('/merchant/project-settings')
}

const goToQueueSettings = () => {
  router.push('/merchant/queue-settings')
}

const goToHandCardSettings = () => {
  router.push('/merchant/hand-card-settings')
}

const goToRoomNumberCardSettings = () => {
  router.push('/merchant/room-number-card-settings')
}

onMounted(() => {
  ensureMerchantPermissionsLoaded()
  fetchMerchant()
})

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
