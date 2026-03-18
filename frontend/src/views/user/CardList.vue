<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b">
      <div class="flex items-center gap-2">
        <span class="text-primary font-bold text-xl">卡包</span>
        <span class="text-gray-400 text-xs">kabao.app</span>
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
            @click="onCardClick(item.id)"
            @touchstart="onCardTouchStart(item)"
            @touchmove="onCardTouchMove"
            @touchend="onCardTouchEnd"
            @touchcancel="onCardTouchEnd"
            :class="[
              'rounded-2xl p-4 cursor-pointer transition-transform active:scale-[0.98]',
              pressingCardId === item.id ? 'scale-[0.985] opacity-90' : '',
              'select-none',
              'kb-card'
            ]"
            style="-webkit-touch-callout: none;"
          >
            <!-- 顶部：商户名称和版本标签 -->
            <div class="flex justify-between items-start mb-1">
              <div>
                <h3 class="text-lg font-bold">{{ item.merchant?.name }}</h3>
                <p class="text-gray-500 text-xs mt-0.5">{{ item.card_type }}</p>
              </div>
              <div class="bg-gray-100 px-2.5 py-0.5 rounded-full">
                <span class="text-xs font-medium">NO: {{ item.card_no }}</span>
              </div>
            </div>

            <!-- 底部：剩余次数和有效期 -->
            <div v-if="!(item.locked && (item.remain_times === 0 || (item.remain_balance === 0 && item.card_type?.includes('储值'))) && currentStatus === 'active')" class="flex justify-between items-end" :class="item.pinnedNotice ? 'mb-2' : 'mb-3'">
              <div>
                <div class="text-gray-500 text-xs mb-0.5">{{ item.card_type?.includes('储值') ? '剩余余额' : '剩余次数' }}</div>
                <div class="flex items-start gap-2">
                  <div class="text-5xl font-bold leading-none">{{ item.card_type?.includes('储值') ? `¥${(item.remain_balance / 100).toFixed(2)}` : item.remain_times }}</div>
                  <span v-if="item.hasAppointment" class="mt-1 px-1.5 py-0.5 bg-red-500 text-white text-xs rounded flex-shrink-0 leading-none">预约</span>
                </div>
              </div>
              <div class="text-right">
                <div class="text-gray-500 text-xs mb-0.5">有效期至</div>
                <div class="text-sm font-medium">{{ formatDate(item.end_date) }}</div>
              </div>
            </div>

            <!-- 锁定提示（在卡片内部） -->
            <div v-if="item.locked" :class="[
              'px-3 py-2 rounded-lg card-gradient-yellow-solid',
              (item.remain_times === 0 || (item.remain_balance === 0 && item.card_type?.includes('储值'))) && currentStatus === 'active' ? 'mb-1 mt-4' : 'mt-1'
            ]">
              <div class="flex items-start gap-2">
                <svg class="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                </svg>
                <div class="flex-1">
                  <div class="text-red-600 text-sm font-medium">卡片已锁定</div>
                  <div class="text-red-500 text-xs mt-0.5">{{ item.locked_reason || '请联系商户处理' }}</div>
                </div>
              </div>
            </div>

            <!-- 置顶通知（在卡片内部底部） -->
            <div 
              v-if="item.pinnedNotice" 
              class="pt-2 border-t border-gray-100"
              @touchstart.stop
              @touchend.stop.prevent="goToDetailWithNotice(item.id)"
              @click.stop.prevent="goToDetailWithNotice(item.id)"
            >
              <div class="flex items-center gap-2 mb-1">
                <svg class="w-3 h-3 text-red-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
                </svg>
                <span class="text-red-600 font-medium text-xs truncate flex-1">{{ item.pinnedNotice.title }}</span>
                <span class="px-1.5 py-0.5 bg-red-500 text-white text-xs rounded flex-shrink-0">置顶</span>
              </div>
              <div class="text-red-500 text-xs line-clamp-2 pl-5">{{ item.pinnedNotice.content }}</div>
            </div>
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

    <div v-if="showActionSheet" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="closeActionSheet">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg px-5 py-6">
        <div class="text-center mb-4">
          <div class="text-gray-800 font-medium">{{ selectedCard?.merchant?.name || '商户' }}</div>
          <div class="text-gray-500 text-sm mt-1">{{ selectedCard?.card_type || '' }}</div>
        </div>
        <div class="space-y-3">
          <button
            v-if="hasActiveAppointment"
            @click="handleAppointmentAction"
            class="w-full py-3 rounded-xl border-2 border-primary text-primary font-medium"
          >
            查看预约
          </button>
          <button
            @click="openVerifyCodeFlowFromAction"
            class="w-full py-3 rounded-xl border-2 border-primary text-primary font-medium"
          >
            生成核销码
          </button>
          <button
            v-if="!hasActiveAppointment"
            @click="handleAppointmentAction"
            class="w-full py-3 rounded-xl border-2 border-primary text-primary font-medium"
          >
            我要预约
          </button>
          <button
            @click="openCardQrFromAction"
            class="w-full py-3 rounded-xl bg-primary text-white font-medium"
          >
            查看二维码
          </button>
        </div>
      </div>
    </div>

    <div v-if="showCardQrModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 select-none" @click.self="closeCardQrModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg overflow-hidden">
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between">
          <h3 class="font-medium text-lg">卡片二维码</h3>
          <button @click="closeCardQrModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5">
          <div class="text-center">
            <div class="text-gray-800 font-medium">{{ selectedCard?.merchant?.name || '商户' }}</div>
            <div class="text-gray-500 text-sm mt-1">{{ selectedCard?.card_type || '' }}</div>
          </div>

          <div class="mt-4 flex justify-center">
            <div
              class="select-none"
              style="-webkit-touch-callout: none; -webkit-user-select: none; user-select: none; pointer-events: none; touch-action: none;"
              @touchstart.prevent
              @touchmove.prevent
              @touchend.prevent
              @contextmenu.prevent
            >
              <canvas ref="cardQrCanvas" class="w-56 h-56" style="-webkit-touch-callout: none;"></canvas>
            </div>
          </div>

          <div class="mt-4 text-center text-gray-400 text-xs">
            请向商户出示此二维码用于查询卡片
          </div>
        </div>
      </div>
    </div>

    <div v-if="showVerifyProjectModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-[55]" @click.self="closeVerifyProjectModal">
      <div class="bg-white rounded-xl w-[90%] max-w-sm overflow-hidden">
        <div class="px-4 py-3 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">请选择项目</div>
          <button class="text-gray-400" @click="closeVerifyProjectModal">×</button>
        </div>
        <div class="p-4 max-h-[60vh] overflow-y-auto">
          <div v-if="!selectedCard?.projects || selectedCard.projects.length === 0" class="text-center text-gray-400 py-6">暂无可选项目</div>
          <label v-for="p in selectedCard.projects" :key="p.id" class="flex items-center gap-3 py-2">
            <input type="radio" name="verify_project" :value="p.id" v-model="selectedVerifyProjectId" />
            <div class="flex-1">
              <div class="text-gray-800">{{ p.name }}</div>
              <div v-if="p.duration" class="text-gray-400 text-xs">时长 {{ p.duration }} 分钟</div>
            </div>
          </label>
        </div>
        <div class="px-4 py-3 border-t flex gap-3">
          <button class="flex-1 py-2.5 rounded-lg border border-gray-200 text-gray-600" @click="closeVerifyProjectModal">取消</button>
          <button class="flex-1 py-2.5 rounded-lg bg-primary text-white disabled:opacity-50" :disabled="!selectedVerifyProjectId || generatingVerifyCode" @click="confirmVerifyProjectAndGenerate">确认</button>
        </div>
      </div>
    </div>

    <div v-if="showVerifyCodeModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[55]" @click.self="closeVerifyCodeModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg overflow-hidden">
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between">
          <h3 class="font-medium text-lg">到店出示核销码</h3>
          <button @click="closeVerifyCodeModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="px-5 py-5">
          <div class="text-center">
            <div class="text-gray-800 font-medium">{{ selectedCard?.merchant?.name || '商户' }}</div>
            <div class="text-gray-500 text-sm mt-1">{{ selectedCard?.card_type || '' }}</div>
          </div>
          <div v-if="verifyQrDataUrl" class="mt-4 flex justify-center">
            <img :src="verifyQrDataUrl" alt="核销二维码" class="w-56 h-56" />
          </div>
          <div v-if="verifyCodeProject" class="text-center text-gray-800 text-sm mt-3 font-medium">
            {{ verifyCodeProject.name }}
            <span v-if="verifyCodeProject.duration" class="text-gray-500">（{{ verifyCodeProject.duration }}分钟）</span>
          </div>
          <p v-if="codeExpireTime" class="text-center text-gray-400 text-sm mt-2">
            有效期至 {{ codeExpireTime }}
          </p>
        </div>
      </div>
    </div>

    <div v-if="showAppointmentModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="closeAppointmentModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg max-h-[80vh] overflow-hidden flex flex-col">
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between flex-shrink-0">
          <h3 class="font-medium text-lg">{{ appointmentModalTitle }}</h3>
          <button @click="closeAppointmentModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="overflow-y-auto flex-1">
          <div :class="availableTechnicians.length > 0 || selectedAppointmentProjectId || !hasAppointmentProjects ? 'px-5 py-4 border-b' : 'px-5 py-4'">
            <div class="text-sm font-medium text-gray-700 mb-2">选择项目</div>
            <div v-if="!hasAppointmentProjects" class="text-gray-400 text-sm">当前卡片未设置项目，将按默认服务时长预约</div>
            <div v-else class="space-y-2">
              <label v-for="p in selectedCard.projects" :key="p.id" class="flex items-center gap-3">
                <input type="radio" name="appt_project" :value="p.id" v-model="selectedAppointmentProjectId" />
                <div class="flex-1">
                  <div class="text-gray-800">{{ p.name }}</div>
                  <div v-if="p.duration" class="text-gray-400 text-xs">时长 {{ p.duration }} 分钟</div>
                </div>
              </label>
            </div>
          </div>

          <div
            v-if="availableTechnicians.length > 0 || loadingSlots"
            :class="selectedAppointmentProjectId ? 'px-5 py-4 border-b' : 'px-5 py-4'"
          >
            <div class="text-sm font-medium text-gray-700 mb-2">选择专业客服</div>
            <div v-if="availableTechnicians.length > 0" class="flex flex-wrap gap-2">
              <button
                v-for="t in displayedTechnicians"
                :key="t.id"
                type="button"
                @click="toggleTechnician(t.id)"
                :class="selectedTechnicianId === t.id ? 'bg-primary text-white' : 'bg-white border-2 border-gray-200 text-gray-700 hover:border-primary'"
                class="py-2 px-3 rounded-lg font-medium transition-all text-sm"
              >
                {{ t.name }}
              </button>
            </div>
            <div v-else class="h-10"></div>
          </div>

          <div
            v-if="selectedAppointmentProjectId || !hasAppointmentProjects"
            class="px-5 py-3"
          >
            <div class="text-sm font-medium text-gray-700 mb-3">
              选择明天预约时间
            </div>
            <div class="relative min-h-[220px]">
              <div v-if="timeSlotError && !loadingSlots" class="text-center py-8 text-gray-400">{{ timeSlotError }}</div>
              <div v-else-if="displayedTimeSlots.length === 0 && !loadingSlots" class="text-center py-8 text-gray-400">当前所选专业客服无可用时间段</div>
              <div v-else class="grid grid-cols-2 gap-3" :class="loadingSlots ? 'opacity-60 pointer-events-none' : ''">
                <button
                  v-for="slot in displayedTimeSlots"
                  :key="slot.time"
                  @click="selectTimeSlot(slot)"
                  :class="{
                    'bg-primary text-white': selectedTimeSlot === slot.time,
                    'bg-white border-2 border-gray-200 text-gray-700 hover:border-primary': selectedTimeSlot !== slot.time
                  }"
                  class="py-3 px-4 rounded-lg font-medium transition-all"
                >
                  <div>{{ formatSlotTime(slot.time) }}</div>
                </button>
              </div>
              <div
                v-if="loadingSlots"
                class="absolute inset-0 flex items-center justify-center bg-white/60 text-gray-400"
              >
                加载中...
              </div>
            </div>
          </div>
        </div>

        <div class="px-5 py-3 flex-shrink-0 bg-white">
          <button
            @click="openAppointmentConfirmModal"
            :disabled="!selectedTimeSlot || appointing"
            class="w-full py-3 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {{ appointing ? '预约中...' : '确认预约' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showAppointmentConfirmModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[60]" @click.self="closeAppointmentConfirmModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-md overflow-hidden">
        <div class="px-5 py-4 border-b">
          <h3 class="text-lg font-medium text-gray-800">确认预约</h3>
        </div>
        <div class="px-5 py-4 space-y-3 text-sm text-gray-700">
          <div>项目：{{ selectedAppointmentProjectName }}</div>
          <div>客服：{{ selectedTechnicianDisplayName }}</div>
          <div>时间：{{ selectedAppointmentTimeText }}</div>
          <div v-if="shouldAutoAssignTechnician" class="text-orange-600">
            未选择专业客服，提交后系统将自动分配客服。
          </div>
        </div>
        <div class="px-5 py-4 flex gap-3 bg-white">
          <button
            type="button"
            @click="closeAppointmentConfirmModal"
            :disabled="appointing"
            class="flex-1 py-3 rounded-lg border border-gray-200 text-gray-700 font-medium disabled:opacity-50"
          >
            取消
          </button>
          <button
            type="button"
            @click="confirmAppointment"
            :disabled="appointing"
            class="flex-1 py-3 rounded-lg bg-primary text-white font-medium disabled:opacity-50"
          >
            {{ appointing ? '预约中...' : '确认' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed, onUnmounted, nextTick, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { appointmentApi, cardApi, noticeApi, shopApi } from '../../api'
import { formatDate } from '../../utils/dateFormat'
import QRCode from 'qrcode'
import { LOW_PRIORITY_POLL_INTERVAL_MS } from '../../constants/polling'

const router = useRouter()
const userName = ref('')
const currentStatus = ref('active')
const cards = ref([])
const pendingPaidOrders = ref([])
const userId = ref(null)

const showCardQrModal = ref(false)
const showActionSheet = ref(false)
const showVerifyProjectModal = ref(false)
const showVerifyCodeModal = ref(false)
const showAppointmentModal = ref(false)
const showAppointmentConfirmModal = ref(false)
const selectedCard = ref(null)
const cardQrCanvas = ref(null)
const pressingCardId = ref(null)
const generatingVerifyCode = ref(false)
const selectedVerifyProjectId = ref(null)
const verifyCode = ref('')
const codeExpireTime = ref('')
const verifyQrDataUrl = ref('')
const verifyCodeProject = ref(null)
const hasActiveAppointment = ref(false)
const appointing = ref(false)
const preparingAppointmentModal = ref(false)
const loadingSlots = ref(false)
const selectedDate = ref('')
const selectedAppointmentProjectId = ref(null)
const selectedTechnicianId = ref(null)
const selectedTimeSlot = ref('')
const timeSlots = ref([])
const timeSlotError = ref('')
const availableTechnicians = ref([])

const prevBodyStyle = {
  userSelect: '',
  webkitUserSelect: '',
  webkitTouchCallout: ''
}

let longPressTimer = null
let longPressStart = null
let verifyStatusPollTimer = null
const suppressClickUntil = ref(0)
const verifyStatusChecking = ref(false)

const triggerHaptic = () => {
  try {
    if ('vibrate' in navigator && typeof navigator.vibrate === 'function') {
      navigator.vibrate(35)
    }
  } catch (_) {
    // ignore
  }
}

const nowTick = ref(Date.now())
let nowTimer = null
let pollTimer = null

// 从 localStorage 获取当前用户信息
const initUser = () => {
  const storedUserId = localStorage.getItem('userId')
  const storedUserName = localStorage.getItem('userName')
  
  if (!storedUserId) {
    // 如果没有登录，跳转到登录页
    router.push('/login')
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

const appointmentCardTitle = computed(() => {
  const merchantName = selectedCard.value?.merchant?.name || '商户'
  const cardName = selectedCard.value?.card_type || '卡片'
  return `${merchantName}-${cardName}`
})

const appointmentModalTitle = computed(() => `预约 ${appointmentCardTitle.value}`)

const selectedAppointmentProject = computed(() => {
  const list = selectedCard.value?.projects || []
  return list.find(p => Number(p.id) === Number(selectedAppointmentProjectId.value)) || null
})

const selectedAppointmentProjectName = computed(() => {
  return selectedAppointmentProject.value?.name || '未指定项目'
})

const selectedTechnicianDisplayName = computed(() => {
  if (!availableTechnicians.value.length) return '系统自动分配'
  const t = (availableTechnicians.value || []).find(item => Number(item.id) === Number(selectedTechnicianId.value))
  return t?.name || '系统自动分配'
})

const shouldAutoAssignTechnician = computed(() => {
  return (availableTechnicians.value || []).length > 0 && !selectedTechnicianId.value
})

const hasAppointmentProjects = computed(() => {
  return Array.isArray(selectedCard.value?.projects) && selectedCard.value.projects.length > 0
})

const selectedAppointmentTimeText = computed(() => {
  if (!selectedTimeSlot.value) return '-'
  const d = new Date(selectedTimeSlot.value)
  if (Number.isNaN(d.getTime())) return String(selectedTimeSlot.value).slice(0, 16).replace('T', ' ')
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd} ${hh}:${min}`
})

const displayedTimeSlots = computed(() => {
  const list = timeSlots.value || []
  if (selectedTechnicianId.value) {
    return list.filter(s => Array.isArray(s?.technician_ids) && s.technician_ids.includes(selectedTechnicianId.value))
  }
  return list
})

const displayedTechnicians = computed(() => {
  const list = availableTechnicians.value || []
  if (selectedTimeSlot.value && !selectedTechnicianId.value) {
    const slot = (timeSlots.value || []).find(s => s && s.time === selectedTimeSlot.value)
    const ids = Array.isArray(slot?.technician_ids) ? slot.technician_ids : []
    return list.filter(t => ids.includes(t.id))
  }
  return list
})

const fetchCards = async () => {
  if (!userId.value) return
  
  try {
    const res = await cardApi.getUserCards(userId.value, currentStatus.value)
    let cardsData = res.data.data || []

    // 锁定卡片优先展示：即使已过期，也要出现在“进行中”
    if (currentStatus.value === 'active') {
      try {
        const expiredRes = await cardApi.getUserCards(userId.value, 'expired')
        const expiredCards = (expiredRes.data.data || []).filter(c => c && c.locked)
        if (expiredCards.length > 0) {
          const seen = new Set((cardsData || []).map(c => c && c.id).filter(Boolean))
          for (const c of expiredCards) {
            if (c && c.id && !seen.has(c.id)) {
              seen.add(c.id)
              cardsData.push(c)
            }
          }
        }
      } catch (e) {
        // ignore
      }
    }

    // 已失效中不展示锁定卡片（避免与“进行中”重复）
    if (currentStatus.value === 'expired') {
      cardsData = (cardsData || []).filter(c => !(c && c.locked))
    }
    
    const enrichedCards = await Promise.all((cardsData || []).map(async (card) => {
      const enrichedCard = { ...card, pinnedNotice: null, hasAppointment: false }

      if (enrichedCard.merchant_id) {
        try {
          const noticesRes = await noticeApi.getMerchantNotices(enrichedCard.merchant_id, 3)
          const notices = noticesRes.data.data || []
          enrichedCard.pinnedNotice = notices.find(n => n.is_pinned) || null
        } catch (err) {
          console.error('获取通知失败:', err)
        }
      }

      if (currentStatus.value === 'active' && enrichedCard.id) {
        try {
          const appointmentRes = await appointmentApi.getCardAppointment(enrichedCard.id)
          const appointment = appointmentRes?.data?.data?.appointment
          enrichedCard.hasAppointment = Boolean(
            appointment &&
            ['pending', 'confirmed'].includes(String(appointment.status || '').trim())
          )
        } catch (_) {
          enrichedCard.hasAppointment = false
        }
      }

      return enrichedCard
    }))

    cards.value = enrichedCards
  } catch (err) {
    console.error('获取卡片失败:', err)
    if (err.response?.status === 401) {
      // token 过期或无效，跳转到登录页
      router.push('/login')
    }
  }
}

const goToDetail = (id) => {
  router.push(`/user/cards/${id}`)
}

const goToDetailWithNotice = (id) => {
  if (!id) return
  suppressClickUntil.value = Date.now() + 500
  router.push({
    path: `/user/cards/${id}`,
    query: {
      scrollToNotice: '1'
    }
  })
}

const onCardClick = (id) => {
  if (Date.now() < suppressClickUntil.value) return
  goToDetail(id)
}

const onCardTouchStart = (card) => {
  if (!card || !card.id) return
  if (longPressTimer) {
    clearTimeout(longPressTimer)
    longPressTimer = null
  }

  pressingCardId.value = card.id

  longPressStart = null
  longPressTimer = setTimeout(async () => {
    suppressClickUntil.value = Date.now() + 900
    triggerHaptic()
    openActionSheet(card)
  }, 820)
}

const onCardTouchMove = (e) => {
  if (!longPressTimer) return
  const t = e?.touches?.[0]
  if (!t) return

  if (!longPressStart) {
    longPressStart = { x: t.clientX, y: t.clientY }
    return
  }

  const dx = t.clientX - longPressStart.x
  const dy = t.clientY - longPressStart.y
  if (dx * dx + dy * dy > 12 * 12) {
    clearTimeout(longPressTimer)
    longPressTimer = null
    pressingCardId.value = null
  }
}

const onCardTouchEnd = () => {
  if (longPressTimer) {
    clearTimeout(longPressTimer)
    longPressTimer = null
  }
  pressingCardId.value = null
}

const openCardQrModal = async (card) => {
  selectedCard.value = card
  showCardQrModal.value = true
  try {
    prevBodyStyle.userSelect = document.body.style.userSelect
    prevBodyStyle.webkitUserSelect = document.body.style.webkitUserSelect
    prevBodyStyle.webkitTouchCallout = document.body.style.webkitTouchCallout
    document.documentElement.classList.add('kb-no-select')
    document.body.classList.add('kb-no-select')
    document.body.style.userSelect = 'none'
    document.body.style.webkitUserSelect = 'none'
    document.body.style.webkitTouchCallout = 'none'
  } catch (_) {
    // ignore
  }
  try {
    const content = `kabao-card:${card.id}`
    await nextTick()
    if (cardQrCanvas.value) {
      await QRCode.toCanvas(cardQrCanvas.value, content, {
        margin: 1,
        scale: 8,
        errorCorrectionLevel: 'M'
      })
    }
  } catch (e) {
    // ignore
  }
}

const openActionSheet = (card) => {
  selectedCard.value = card
  hasActiveAppointment.value = false
  showActionSheet.value = true
  fetchCardAppointmentStatus(card?.id)
}

const closeActionSheet = () => {
  showActionSheet.value = false
}

const fetchCardAppointmentStatus = async (cardId) => {
  const id = Number(cardId || 0)
  if (!id) {
    hasActiveAppointment.value = false
    return
  }

  try {
    const res = await appointmentApi.getCardAppointment(id)
    const appointment = res?.data?.data?.appointment
    hasActiveAppointment.value = Boolean(
      appointment &&
      ['pending', 'confirmed'].includes(String(appointment.status || '').trim())
    )
  } catch (_) {
    hasActiveAppointment.value = false
  }
}

const closeVerifyProjectModal = () => {
  if (generatingVerifyCode.value) return
  showVerifyProjectModal.value = false
  selectedVerifyProjectId.value = null
}

const closeVerifyCodeModal = () => {
  showVerifyCodeModal.value = false
  verifyCode.value = ''
  codeExpireTime.value = ''
  verifyQrDataUrl.value = ''
  verifyCodeProject.value = null
  stopVerifyStatusPoll()
}

const ensureSelectedCardForAppointment = async () => {
  const cardId = Number(selectedCard.value?.id || 0)
  if (!cardId) return false

  try {
    const res = await cardApi.getCard(cardId)
    const cardDetail = res.data.data || {}
    selectedCard.value = {
      ...selectedCard.value,
      ...cardDetail,
      merchant: cardDetail.merchant || selectedCard.value?.merchant || null
    }
  } catch (err) {
    console.error('获取卡片详情失败:', err)
  }

  if (Array.isArray(selectedCard.value?.projects) && selectedCard.value.projects.length > 0) {
    return true
  }

  try {
    const res = await cardApi.getCardProjects(cardId)
    const projects = Array.isArray(res.data.data) ? res.data.data : []
    selectedCard.value = {
      ...selectedCard.value,
      projects
    }
  } catch (err) {
    console.error('获取卡片项目失败:', err)
  }

  return Array.isArray(selectedCard.value?.projects) && selectedCard.value.projects.length > 0
}

const doGenerateVerifyCode = async (projectId) => {
  const payload = projectId ? { project_id: Number(projectId) } : undefined
  const res = await cardApi.generateVerifyCode(Number(selectedCard.value.id), payload)
  verifyCode.value = res.data.data.code
  const expireAt = new Date(res.data.data.expire_at * 1000)
  codeExpireTime.value = expireAt.toLocaleTimeString()

  if (projectId) {
    verifyCodeProject.value = (selectedCard.value?.projects || []).find(p => Number(p.id) === Number(projectId)) || null
  } else {
    verifyCodeProject.value = null
  }

  verifyQrDataUrl.value = await QRCode.toDataURL(verifyCode.value, {
    margin: 1,
    scale: 8,
    errorCorrectionLevel: 'M'
  })
}

const stopVerifyStatusPoll = () => {
  if (verifyStatusPollTimer) {
    clearInterval(verifyStatusPollTimer)
    verifyStatusPollTimer = null
  }
  verifyStatusChecking.value = false
}

const closeAllOverlayModals = () => {
  showActionSheet.value = false
  showCardQrModal.value = false
  showVerifyProjectModal.value = false
  showVerifyCodeModal.value = false
  showAppointmentConfirmModal.value = false
  showAppointmentModal.value = false
  stopVerifyStatusPoll()
  verifyCode.value = ''
  codeExpireTime.value = ''
  verifyQrDataUrl.value = ''
  verifyCodeProject.value = null
  selectedVerifyProjectId.value = null
  selectedAppointmentProjectId.value = null
  selectedTechnicianId.value = null
  selectedTimeSlot.value = ''
  timeSlots.value = []
  timeSlotError.value = ''
  availableTechnicians.value = []
  hasActiveAppointment.value = false
  pressingCardId.value = null
  selectedCard.value = null
  try {
    document.documentElement.classList.remove('kb-no-select')
    document.body.classList.remove('kb-no-select')
    document.body.style.userSelect = prevBodyStyle.userSelect
    document.body.style.webkitUserSelect = prevBodyStyle.webkitUserSelect
    document.body.style.webkitTouchCallout = prevBodyStyle.webkitTouchCallout
  } catch (_) {
    // ignore
  }
}

const waitForOverlayClosePaint = async () => {
  await nextTick()
  await new Promise(resolve => requestAnimationFrame(() => {
    requestAnimationFrame(() => resolve())
  }))
  await new Promise(resolve => setTimeout(resolve, 80))
}

const handlePageRestoreCleanup = () => {
  closeAllOverlayModals()
}

const handleVisibilityCleanup = () => {
  if (document.visibilityState === 'visible') {
    closeAllOverlayModals()
  }
}

const checkVerifyStatusAndMaybeJump = async () => {
  if (verifyStatusChecking.value) return
  if (!verifyCode.value) return

  verifyStatusChecking.value = true
  try {
    const res = await cardApi.getVerifyCodeStatus(verifyCode.value)
    const data = res?.data?.data || {}
    if (!data.used) return

    stopVerifyStatusPoll()
    const cardId = Number(selectedCard.value?.id || 0)
    closeAllOverlayModals()
    if (cardId > 0) {
      router.push(`/user/cards/${cardId}?scrollToUsages=1`)
    }
  } catch (_) {
    // ignore
  } finally {
    verifyStatusChecking.value = false
  }
}

const startVerifyStatusPoll = async () => {
  stopVerifyStatusPoll()
  if (!verifyCode.value) return

  await checkVerifyStatusAndMaybeJump()
  if (!verifyCode.value) return

  verifyStatusPollTimer = setInterval(() => {
    if (!verifyCode.value) {
      stopVerifyStatusPoll()
      return
    }
    checkVerifyStatusAndMaybeJump()
  }, 1000)
}

const openVerifyCodeFlowFromAction = async () => {
  closeActionSheet()
  if (!selectedCard.value?.id) return
  await ensureSelectedCardForAppointment()
  const projects = selectedCard.value?.projects || []
  if (projects.length > 1) {
    selectedVerifyProjectId.value = null
    showVerifyProjectModal.value = true
    return
  }

  generatingVerifyCode.value = true
  try {
    const onlyProjectId = projects.length === 1 ? projects[0].id : null
    await doGenerateVerifyCode(onlyProjectId)
    showVerifyCodeModal.value = true
    await startVerifyStatusPoll()
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generatingVerifyCode.value = false
  }
}

const handleAppointmentAction = async () => {
  const cardId = Number(selectedCard.value?.id || 0)
  if (!cardId) return

  if (hasActiveAppointment.value) {
    closeAllOverlayModals()
    await waitForOverlayClosePaint()
    router.push(`/user/cards/${cardId}?scrollToAppointment=1`)
    return
  }

  closeActionSheet()
  await openAppointmentModalFromAction()
}

const confirmVerifyProjectAndGenerate = async () => {
  if (!selectedVerifyProjectId.value || generatingVerifyCode.value) return
  generatingVerifyCode.value = true
  try {
    await doGenerateVerifyCode(selectedVerifyProjectId.value)
    showVerifyProjectModal.value = false
    showVerifyCodeModal.value = true
    await startVerifyStatusPoll()
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generatingVerifyCode.value = false
  }
}

const openCardQrFromAction = async () => {
  const card = selectedCard.value
  closeActionSheet()
  if (!card) return
  await openCardQrModal(card)
}

const getTomorrowDate = () => {
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  return tomorrow.toISOString().slice(0, 10)
}

const resetAppointmentState = () => {
  selectedDate.value = ''
  selectedAppointmentProjectId.value = null
  selectedTechnicianId.value = null
  selectedTimeSlot.value = ''
  timeSlots.value = []
  timeSlotError.value = ''
  availableTechnicians.value = []
  showAppointmentConfirmModal.value = false
}

const openAppointmentModalFromAction = async () => {
  closeActionSheet()
  if (!selectedCard.value?.merchant_id) {
    alert('卡片信息不完整，暂时无法预约')
    return
  }

  resetAppointmentState()
  selectedDate.value = getTomorrowDate()
  await ensureSelectedCardForAppointment()
  const projects = selectedCard.value?.projects || []
  if (projects.length > 0) {
    preparingAppointmentModal.value = true
    try {
      selectedAppointmentProjectId.value = Number(projects[0].id)
      await loadTimeSlots(selectedDate.value)
    } finally {
      preparingAppointmentModal.value = false
    }
  } else {
    await loadTimeSlots(selectedDate.value)
  }
  showAppointmentModal.value = true
}

const closeAppointmentModal = () => {
  showAppointmentModal.value = false
  resetAppointmentState()
}

const loadTimeSlots = async (date) => {
  if (!selectedCard.value?.merchant_id) return
  loadingSlots.value = true
  timeSlotError.value = ''
  try {
    const res = await appointmentApi.getAvailableTimeSlots(selectedCard.value.merchant_id, date, selectedAppointmentProjectId.value || undefined)
    timeSlots.value = res.data.data.time_slots || []
    availableTechnicians.value = res.data.data.technicians || []
    if (selectedTimeSlot.value) {
      const stillExists = timeSlots.value.some(slot => slot?.time === selectedTimeSlot.value)
      if (!stillExists) selectedTimeSlot.value = ''
    }
    if (selectedTechnicianId.value) {
      const stillExists = availableTechnicians.value.some(item => Number(item.id) === Number(selectedTechnicianId.value))
      if (!stillExists) selectedTechnicianId.value = null
    }
  } catch (err) {
    timeSlots.value = []
    availableTechnicians.value = []
    timeSlotError.value = `获取可用时间段失败: ${err.response?.data?.error || err.message}`
    alert(timeSlotError.value)
  } finally {
    loadingSlots.value = false
  }
}

const toggleTechnician = (id) => {
  const next = Number(id)
  if (!next) return
  if (selectedTechnicianId.value === next) {
    selectedTechnicianId.value = null
    return
  }
  selectedTechnicianId.value = next
  if (selectedTimeSlot.value) {
    const slot = (timeSlots.value || []).find(s => s && s.time === selectedTimeSlot.value)
    const ids = Array.isArray(slot?.technician_ids) ? slot.technician_ids : []
    if (ids.length > 0 && !ids.includes(next)) {
      selectedTimeSlot.value = ''
    }
  }
}

const selectTimeSlot = (slot) => {
  selectedTimeSlot.value = slot.time
  if (selectedTechnicianId.value) {
    const ids = Array.isArray(slot?.technician_ids) ? slot.technician_ids : []
    if (ids.length > 0 && !ids.includes(selectedTechnicianId.value)) {
      selectedTechnicianId.value = null
    }
  }
}

const formatSlotTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  return `${hours}:${minutes}`
}

const confirmAppointment = async () => {
  if (!selectedCard.value?.id || !selectedTimeSlot.value || appointing.value) return

  appointing.value = true
  try {
    const userId = localStorage.getItem('userId')
    if (!userId) {
      closeAllAppointmentModals()
      alert('请先登录')
      router.push('/login')
      return
    }

    const payload = {
      card_id: Number(selectedCard.value.id),
      merchant_id: selectedCard.value.merchant_id,
      user_id: parseInt(userId),
      technician_id: selectedTechnicianId.value ? Number(selectedTechnicianId.value) : null,
      appointment_time: selectedTimeSlot.value
    }
    if (selectedAppointmentProjectId.value) {
      payload.project_id = Number(selectedAppointmentProjectId.value)
    }

    await appointmentApi.createAppointment(payload)

    closeAppointmentConfirmModal()
    closeAppointmentModal()
  } catch (err) {
    closeAllAppointmentModals()
    alert(err.response?.data?.error || '预约失败')
  } finally {
    appointing.value = false
  }
}

const openAppointmentConfirmModal = () => {
  if (!selectedCard.value?.id || !selectedTimeSlot.value || appointing.value) return
  showAppointmentConfirmModal.value = true
}

const closeAppointmentConfirmModal = () => {
  if (appointing.value) return
  showAppointmentConfirmModal.value = false
}

const closeAllAppointmentModals = () => {
  showAppointmentConfirmModal.value = false
  showAppointmentModal.value = false
}

const closeCardQrModal = () => {
  showCardQrModal.value = false
  selectedCard.value = null
  try {
    document.documentElement.classList.remove('kb-no-select')
    document.body.classList.remove('kb-no-select')
    document.body.style.userSelect = prevBodyStyle.userSelect
    document.body.style.webkitUserSelect = prevBodyStyle.webkitUserSelect
    document.body.style.webkitTouchCallout = prevBodyStyle.webkitTouchCallout
  } catch (_) {
    // ignore
  }
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

watch(selectedAppointmentProjectId, async () => {
  if (preparingAppointmentModal.value) return
  selectedTimeSlot.value = ''
  timeSlotError.value = ''
  if (!selectedDate.value) return
  if (!selectedAppointmentProjectId.value && hasAppointmentProjects.value) return
  await loadTimeSlots(selectedDate.value)
})

onMounted(() => {
  initUser()
  fetchCards()
  fetchPendingOrders()
  window.addEventListener('pageshow', handlePageRestoreCleanup)
  document.addEventListener('visibilitychange', handleVisibilityCleanup)

  nowTimer = setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)

  pollTimer = setInterval(() => {
    fetchCards()
    fetchPendingOrders()
  }, LOW_PRIORITY_POLL_INTERVAL_MS)
})

onActivated(() => {
  closeAllOverlayModals()
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

  if (longPressTimer) {
    clearTimeout(longPressTimer)
    longPressTimer = null
  }

  window.removeEventListener('pageshow', handlePageRestoreCleanup)
  document.removeEventListener('visibilitychange', handleVisibilityCleanup)
  stopVerifyStatusPoll()
  closeActionSheet()
})
</script>

<style>
.kb-no-select, .kb-no-select * {
  -webkit-user-select: none !important;
  user-select: none !important;
  -webkit-touch-callout: none !important;
  -webkit-tap-highlight-color: rgba(0, 0, 0, 0) !important;
}

.card-gradient-yellow-solid {
  background: linear-gradient(135deg, #fff8e1 0%, #ffeeaa 100%);
  color: #e65100;
  border: 1px solid #ffe082;
  box-shadow: 0 2px 8px rgba(255, 167, 38, 0.2);
}
</style>
