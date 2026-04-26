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
            @click="onCardClick($event, item)"
            @touchstart="onCardTouchStart($event, item)"
            @touchmove="onCardTouchMove"
            @touchend="onCardTouchEnd($event, item)"
            @touchcancel="onCardTouchCancel"
            :class="[
              'rounded-2xl p-4 cursor-pointer transition-transform',
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
              <div
                :class="getCardNoTagClass(item)"
                class="px-2.5 py-0.5 rounded-full"
              >
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
                  <div v-if="getCardServiceTimeLines(item).length > 0" class="pt-1 max-w-[180px] text-left">
                    <div class="grid grid-cols-2 gap-x-3 gap-y-0.5">
                      <div
                        v-for="line in getCardServiceTimeLines(item)"
                        :key="line"
                        class="text-xs text-gray-700 leading-4 whitespace-nowrap"
                      >
                        {{ line }}
                      </div>
                    </div>
                  </div>
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
              @touchstart.stop="onNoticeTouchStart($event, item.id)"
              @touchmove.stop="onNoticeTouchMove"
              @touchend.stop.prevent="onNoticeTouchEnd($event, item.id)"
              @touchcancel.stop="onNoticeTouchCancel"
              @click.stop.prevent="onNoticeClick($event, item.id)"
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
            v-if="selectedCardHasStartPendingUsage"
            @click="openStartPendingUsageFromAction"
            class="w-full py-3 rounded-xl border-2 border-primary text-primary font-medium"
          >
            {{ selectedCardScanStartLabel }}
          </button>
          <button
            v-if="hasActiveAppointment"
            @click="handleAppointmentAction"
            class="w-full py-3 rounded-xl border-2 border-primary text-primary font-medium"
          >
            {{ selectedCardCanArriveNow ? '出示预约签到码' : '查看预约' }}
          </button>
          <button
            v-if="shouldShowSelectedCardVerifyButton && !selectedCardHasCurrentWindowGeneratedVerifyCode"
            @click="openVerifyCodeFlowFromAction"
            class="w-full py-3 rounded-xl border-2 border-primary text-primary font-medium"
          >
            {{ selectedCardMerchantClosed ? '暂停营业' : '生成核销码' }}
          </button>
          <button
            v-if="hasBookableMultiServiceSlots"
            @click="openMultiServiceBookingModalFromAction"
            class="w-full py-3 rounded-xl border-2 border-primary text-primary font-medium"
          >
            预约
          </button>
          <button
            v-if="!hasActiveAppointment && !selectedCardHasServiceTimeProject"
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
          <div v-if="selectedCardVerifyableProjects.length === 0" class="text-center text-gray-400 py-6">当前暂无可生成核销码的项目</div>
          <label v-for="p in selectedCardVerifyableProjects" :key="p.id" class="flex items-center gap-3 py-2">
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
          <h3 class="font-medium text-lg">{{ verifyCodeModalTitle }}</h3>
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
          <div class="mt-2 text-center text-gray-600 text-sm">
            {{ verifyCodeModalHint }}
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

    <div v-if="showMultiServiceBookingModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-[56]" @click.self="closeMultiServiceBookingModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg max-h-[80vh] overflow-hidden flex flex-col">
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between flex-shrink-0">
          <h3 class="font-medium text-lg">预约 {{ appointmentCardTitle }}</h3>
          <button @click="closeMultiServiceBookingModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="overflow-y-auto flex-1 px-5 py-4">
          <div v-if="loadingMultiServiceBookingSlots" class="py-10 text-center text-gray-400">正在加载可预约场次...</div>
          <div v-else-if="multiServiceBookingSlots.length === 0" class="py-10 text-center text-gray-400">当前暂无可预约的多人项目场次</div>
          <div v-else class="space-y-3">
            <div
              v-for="slot in multiServiceBookingSlots"
              :key="getMultiServiceBookingSlotKey(slot)"
              class="rounded-xl border border-gray-200 p-4"
            >
              <div class="flex items-start justify-between gap-4">
                <div class="min-w-0">
                  <div class="text-gray-800 font-medium">{{ slot.project_name }}</div>
                  <div class="text-gray-500 text-sm mt-1">{{ formatMultiServiceSlotDateTime(slot.slot_start_at) }}</div>
                  <div class="text-gray-400 text-xs mt-1">
                    已预约 {{ slot.booked_count || 0 }} / {{ slot.service_capacity || 0 }}
                    <span v-if="slot.checked_in_count">，已核销 {{ slot.checked_in_count }}</span>
                  </div>
                  <div class="flex flex-wrap gap-2 mt-2">
                    <span v-if="slot.current_user_booked" class="px-2 py-0.5 rounded-full bg-green-50 text-green-600 text-xs">已预约</span>
                    <span v-if="slot.is_full && !slot.current_user_booked" class="px-2 py-0.5 rounded-full bg-red-50 text-red-600 text-xs">已满</span>
                    <span v-else-if="!slot.current_user_booked" class="px-2 py-0.5 rounded-full bg-blue-50 text-blue-600 text-xs">可预约</span>
                    <span v-if="slot.current_user_booked && !slot.current_user_can_cancel" class="px-2 py-0.5 rounded-full bg-orange-50 text-orange-600 text-xs">当前取消计失约</span>
                  </div>
                </div>
                <div class="flex-shrink-0">
                  <button
                    v-if="slot.current_user_booked"
                    type="button"
                    class="px-4 py-2 rounded-lg border border-red-200 text-red-600 disabled:opacity-50"
                    :disabled="submittingMultiServiceBooking"
                    @click="cancelMultiServiceBooking(slot)"
                  >
                    取消预约
                  </button>
                  <button
                    v-else
                    type="button"
                    class="px-4 py-2 rounded-lg bg-primary text-white disabled:opacity-50"
                    :disabled="submittingMultiServiceBooking || slot.is_full"
                    @click="createMultiServiceBooking(slot)"
                  >
                    预约
                  </button>
                </div>
              </div>
              <div v-if="slot.current_user_booked && slot.current_user_cancel_deadline_at" class="text-xs text-gray-400 mt-3">
                无责任取消截止：{{ formatMultiServiceSlotDateTime(slot.current_user_cancel_deadline_at) }}
              </div>
            </div>
          </div>
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
              <label v-for="p in appointmentProjects" :key="p.id" class="flex items-center gap-3">
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
            <div v-if="selectedAppointmentTechnicianText" class="mt-2 rounded-lg bg-primary-light text-primary border border-primary/10 px-3 py-2 text-sm">
              已选客服：{{ selectedAppointmentTechnicianText }}
            </div>
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
            :disabled="!canSubmitAppointment"
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
            :disabled="!canSubmitAppointment"
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
import { appointmentApi, cardApi, noticeApi, shopApi, usageApi } from '../../api'
import { formatDate } from '../../utils/dateFormat'
import { normalizeSessionStatus } from '../../utils/sessionStatus'
import { getScanStartLabel } from '../../utils/terms'
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
const showMultiServiceBookingModal = ref(false)
const showAppointmentModal = ref(false)
const showAppointmentConfirmModal = ref(false)
const skipNextAppointmentProjectReload = ref(false)
const selectedCard = ref(null)
const cardQrCanvas = ref(null)
const pressingCardId = ref(null)
const generatingVerifyCode = ref(false)
const selectedVerifyProjectId = ref(null)
const verifyCode = ref('')
const codeExpireTime = ref('')
const verifyQrDataUrl = ref('')
const verifyCodeProject = ref(null)
const verifyCodeMode = ref('verify')
const hasActiveAppointment = ref(false)
const selectedCardAppointment = ref(null)
const selectedCardCanArriveNow = ref(false)
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
const loadingMultiServiceBookingSlots = ref(false)
const submittingMultiServiceBooking = ref(false)
const multiServiceBookingSlots = ref([])

const prevBodyStyle = {
  userSelect: '',
  webkitUserSelect: '',
  webkitTouchCallout: ''
}

const TAP_MOVE_THRESHOLD_PX = 8
const TAP_MOVE_THRESHOLD_SQ = TAP_MOVE_THRESHOLD_PX * TAP_MOVE_THRESHOLD_PX
const TAP_CANCEL_CLICK_SUPPRESS_MS = 350
const TAP_OPEN_CLICK_SUPPRESS_MS = 500
const LONG_PRESS_DURATION_MS = 820
const LONG_PRESS_CLICK_SUPPRESS_MS = 900

let longPressTimer = null
let verifyStatusPollTimer = null
const suppressClickUntil = ref(0)
const verifyStatusChecking = ref(false)
const activeCardId = ref(null)
let cardTouchStartX = 0
let cardTouchStartY = 0
let cardTouchMoved = false
let cardLongPressed = false
let noticeTouchCardId = null
let noticeTouchStartX = 0
let noticeTouchStartY = 0
let noticeTouchMoved = false

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

const appointmentProjects = computed(() => {
  const list = selectedCard.value?.projects || []
  return Array.isArray(list) ? list.filter(p => p?.bookable_online !== false) : []
})

const selectedAppointmentProject = computed(() => {
  const list = appointmentProjects.value || []
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

const selectedAppointmentTechnicianText = computed(() => {
  const technicianId = Number(selectedTechnicianId.value || 0)
  if (!technicianId) return ''
  const technician = (availableTechnicians.value || []).find(item => Number(item?.id || 0) === technicianId)
  if (!technician) return ''
  const name = String(technician?.name || '').trim() || `客服${technicianId}`
  const account = String(technician?.account || '').trim()
  return account ? `${name} - ${account}` : name
})

const shouldAutoAssignTechnician = computed(() => {
  return (availableTechnicians.value || []).length > 0 && !selectedTechnicianId.value
})

const hasAppointmentProjects = computed(() => {
  return appointmentProjects.value.length > 0
})

const selectedAppointmentTimeText = computed(() => {
  if (!selectedTimeSlot.value) return '-'
  const raw = String(selectedTimeSlot.value).trim().replace('T', ' ')
  const match = raw.match(/(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2})/)
  if (match) return `${match[1]} ${match[2]}`
  return raw.slice(0, 16)
})

const canSubmitAppointment = computed(() => {
  if (appointing.value || loadingSlots.value || timeSlotError.value) return false
  if (!selectedCard.value?.id || !selectedTimeSlot.value) return false
  return displayedTimeSlots.value.some(slot => slot?.time === selectedTimeSlot.value)
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

const selectedCardMerchantClosed = computed(() => selectedCard.value?.merchant?.is_open === false)
const selectedCardHasStartPendingUsage = computed(() => Boolean(selectedCard.value?.hasStartPendingUsage && selectedCard.value?.startPendingUsageSessionId))
const selectedCardScanStartLabel = computed(() => getScanStartLabel(selectedCard.value?.merchant))
const selectedCardHasServiceTimeProject = computed(() => getCardServiceTimeProjects(selectedCard.value).length > 0)
const selectedCardHasCurrentWindowGeneratedVerifyCode = computed(() => {
  const projects = Array.isArray(selectedCard.value?.projects) ? selectedCard.value.projects : []
  return projects.some(project => {
    if (Number(project?.service_capacity || 0) <= 1) return false
    return project?.multi_service_overview?.current_window_verify_code_generated === true
  })
})
const selectedCardVerifyableProjects = computed(() => {
  const projects = Array.isArray(selectedCard.value?.projects) ? selectedCard.value.projects : []
  const now = new Date(nowTick.value)
  return projects.filter(project => projectServiceTimeAllowed(project, now))
})
const shouldShowSelectedCardVerifyButton = computed(() => {
  if (selectedCardCanArriveNow.value) return false
  if (!selectedCard.value) return false
  return selectedCardVerifyableProjects.value.length > 0
})
const hasBookableMultiServiceSlots = computed(() => (multiServiceBookingSlots.value || []).length > 0)

const getSelectedCardMerchantClosedMessage = () => {
  const merchantName = selectedCard.value?.merchant?.name || '商户'
  return `${merchantName} 当前未营业`
}

const getUsageStartPendingSortTime = (usage) => {
  const candidates = [
    usage?.service_session_scheduled_start_at,
    usage?.service_session_updated_at,
    usage?.used_at,
    usage?.created_at
  ]
  for (const value of candidates) {
    const ms = new Date(value || '').getTime()
    if (!Number.isNaN(ms) && ms > 0) return ms
  }
  return Number.MAX_SAFE_INTEGER
}

const pickEarliestStartPendingUsage = (usages) => {
  return (usages || [])
    .filter(usage =>
      normalizeSessionStatus(usage?.service_session_status) === 'start_pending' &&
      !usage?.service_session_start_confirmed_at &&
      usage?.service_session_id
    )
    .sort((a, b) => {
      const byTime = getUsageStartPendingSortTime(a) - getUsageStartPendingSortTime(b)
      if (byTime !== 0) return byTime
      return Number(a?.id || 0) - Number(b?.id || 0)
    })[0] || null
}

const getCardNoTagClass = (card) => {
  if (card?.hasStartPendingUsage) return 'bg-red-100 text-red-700'
  if (card?.hasServingUsage) return 'bg-green-100 text-green-700'
  return 'bg-gray-100 text-gray-700'
}

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
      const enrichedCard = {
        ...card,
        pinnedNotice: null,
        hasAppointment: false,
        hasServingUsage: false,
        hasStartPendingUsage: false,
        startPendingUsageSessionId: '',
        projects: Array.isArray(card.projects) ? card.projects : []
      }

      if (enrichedCard.id) {
        try {
          const detailRes = await cardApi.getCard(enrichedCard.id)
          const detail = detailRes?.data?.data || {}
          if (Array.isArray(detail.projects)) {
            enrichedCard.projects = detail.projects
          }
        } catch (err) {
          console.error('获取卡片项目失败:', err)
        }
      }

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

        try {
          const usageRes = await usageApi.getCardUsages(enrichedCard.id)
          const usages = usageRes?.data?.data || []
          enrichedCard.hasServingUsage = usages.some(usage => normalizeSessionStatus(usage?.service_session_status) === 'serving')
          const startPendingUsage = pickEarliestStartPendingUsage(usages)
          enrichedCard.hasStartPendingUsage = Boolean(startPendingUsage)
          enrichedCard.startPendingUsageSessionId = startPendingUsage?.service_session_id ? String(startPendingUsage.service_session_id) : ''
        } catch (_) {
          enrichedCard.hasServingUsage = false
          enrichedCard.hasStartPendingUsage = false
          enrichedCard.startPendingUsageSessionId = ''
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

const setSuppressClick = (durationMs) => {
  suppressClickUntil.value = Math.max(suppressClickUntil.value, Date.now() + durationMs)
}

const clearCardLongPressTimer = () => {
  if (longPressTimer) {
    clearTimeout(longPressTimer)
    longPressTimer = null
  }
}

const resetCardTouchState = () => {
  clearCardLongPressTimer()
  pressingCardId.value = null
  activeCardId.value = null
  cardTouchStartX = 0
  cardTouchStartY = 0
  cardTouchMoved = false
  cardLongPressed = false
}

const resetNoticeTouchState = () => {
  noticeTouchCardId = null
  noticeTouchStartX = 0
  noticeTouchStartY = 0
  noticeTouchMoved = false
}

const hasTouchMovedPastThreshold = (touch, startX, startY) => {
  if (!touch) return false
  const dx = touch.clientX - startX
  const dy = touch.clientY - startY
  return Math.abs(dy) >= TAP_MOVE_THRESHOLD_PX || Math.abs(dx) >= TAP_MOVE_THRESHOLD_PX || (dx * dx + dy * dy >= TAP_MOVE_THRESHOLD_SQ)
}

const goToDetail = (card) => {
  const id = Number(card?.id || card || 0)
  if (!id) return
  if (card?.hasServingUsage) {
    router.push({
      path: `/user/cards/${id}`,
      query: {
        scrollToUsages: '1'
      }
    })
    return
  }
  router.push(`/user/cards/${id}`)
}

const goToDetailWithNotice = (id) => {
  if (!id) return
  setSuppressClick(TAP_OPEN_CLICK_SUPPRESS_MS)
  router.push({
    path: `/user/cards/${id}`,
    query: {
      scrollToNotice: '1'
    }
  })
}

const onCardClick = (_event, card) => {
  if (Date.now() < suppressClickUntil.value) return
  goToDetail(card)
}

const onCardTouchStart = (event, card) => {
  if (!card || !card.id) return
  clearCardLongPressTimer()
  const touch = event?.touches?.[0]
  activeCardId.value = card.id
  pressingCardId.value = card.id
  cardTouchStartX = touch?.clientX || 0
  cardTouchStartY = touch?.clientY || 0
  cardTouchMoved = false
  cardLongPressed = false

  longPressTimer = setTimeout(() => {
    longPressTimer = null
    if (activeCardId.value !== card.id || cardTouchMoved) return
    cardLongPressed = true
    pressingCardId.value = null
    setSuppressClick(LONG_PRESS_CLICK_SUPPRESS_MS)
    triggerHaptic()
    openActionSheet(card)
  }, LONG_PRESS_DURATION_MS)
}

const onCardTouchMove = (e) => {
  if (!activeCardId.value) return
  const t = e?.touches?.[0]
  if (!t) return
  if (!cardTouchMoved && hasTouchMovedPastThreshold(t, cardTouchStartX, cardTouchStartY)) {
    cardTouchMoved = true
    pressingCardId.value = null
    clearCardLongPressTimer()
    setSuppressClick(TAP_CANCEL_CLICK_SUPPRESS_MS)
  }
}

const onCardTouchEnd = (_event, card) => {
  const wasMoved = cardTouchMoved
  const wasLongPressed = cardLongPressed
  const shouldOpenDetail = Boolean(
    card?.id &&
    activeCardId.value === card.id &&
    !wasMoved &&
    !wasLongPressed
  )
  resetCardTouchState()
  if (!shouldOpenDetail) {
    if (wasMoved || wasLongPressed) {
      setSuppressClick(wasLongPressed ? LONG_PRESS_CLICK_SUPPRESS_MS : TAP_CANCEL_CLICK_SUPPRESS_MS)
    }
    return
  }
  setSuppressClick(TAP_OPEN_CLICK_SUPPRESS_MS)
  goToDetail(card)
}

const onCardTouchCancel = () => {
  if (activeCardId.value) {
    setSuppressClick(TAP_CANCEL_CLICK_SUPPRESS_MS)
  }
  resetCardTouchState()
}

const onNoticeTouchStart = (event, cardId) => {
  const touch = event?.touches?.[0]
  noticeTouchCardId = Number(cardId || 0) || null
  noticeTouchStartX = touch?.clientX || 0
  noticeTouchStartY = touch?.clientY || 0
  noticeTouchMoved = false
}

const onNoticeTouchMove = (event) => {
  if (!noticeTouchCardId) return
  const touch = event?.touches?.[0]
  if (!noticeTouchMoved && hasTouchMovedPastThreshold(touch, noticeTouchStartX, noticeTouchStartY)) {
    noticeTouchMoved = true
    setSuppressClick(TAP_CANCEL_CLICK_SUPPRESS_MS)
  }
}

const onNoticeTouchEnd = (_event, cardId) => {
  const shouldOpenNotice = Boolean(
    noticeTouchCardId &&
    Number(cardId || 0) === noticeTouchCardId &&
    !noticeTouchMoved
  )
  resetNoticeTouchState()
  if (!shouldOpenNotice) {
    setSuppressClick(TAP_CANCEL_CLICK_SUPPRESS_MS)
    return
  }
  goToDetailWithNotice(cardId)
}

const onNoticeTouchCancel = () => {
  if (noticeTouchCardId) {
    setSuppressClick(TAP_CANCEL_CLICK_SUPPRESS_MS)
  }
  resetNoticeTouchState()
}

const onNoticeClick = (_event, cardId) => {
  if (Date.now() < suppressClickUntil.value) return
  goToDetailWithNotice(cardId)
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
  selectedCardAppointment.value = null
  selectedCardCanArriveNow.value = false
  multiServiceBookingSlots.value = []
  showActionSheet.value = true
  fetchCardAppointmentStatus(card?.id)
  fetchSelectedCardMultiServiceBookingSlots(card?.id)
}

const closeActionSheet = () => {
  showActionSheet.value = false
}

const fetchCardAppointmentStatus = async (cardId) => {
  const id = Number(cardId || 0)
  if (!id) {
    hasActiveAppointment.value = false
    selectedCardAppointment.value = null
    selectedCardCanArriveNow.value = false
    return
  }

  try {
    const res = await appointmentApi.getCardAppointment(id)
    const appointment = res?.data?.data?.appointment
    selectedCardAppointment.value = appointment || null
    selectedCardCanArriveNow.value = Boolean(res?.data?.data?.can_arrive_now)
    hasActiveAppointment.value = Boolean(
      appointment &&
      ['pending', 'confirmed', 'arrived'].includes(String(appointment.status || '').trim())
    )
  } catch (_) {
    hasActiveAppointment.value = false
    selectedCardAppointment.value = null
    selectedCardCanArriveNow.value = false
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
  verifyCodeMode.value = 'verify'
  stopVerifyStatusPoll()
}

const closeMultiServiceBookingModal = () => {
  if (submittingMultiServiceBooking.value) return
  showMultiServiceBookingModal.value = false
}

const verifyCodeModalTitle = computed(() => {
  return verifyCodeMode.value === 'appointment_checkin' ? '到店出示预约到店码' : '到店出示核销码'
})

const verifyCodeModalHint = computed(() => {
  if (verifyCodeMode.value === 'appointment_checkin') {
    return '若到店后剩余服务时长仍达到该项目时长的一半，请向工作人员出示此码，由客服扫码后建单进入服务'
  }
  return '请向工作人员出示此码，由工作人员扫码完成到店核销'
})

const getCardServiceTimeProjects = (card) => {
  const projects = Array.isArray(card?.projects) ? card.projects : []
  return projects.filter((project) => {
    const slots = Array.isArray(project?.service_time_slots) ? project.service_time_slots : []
    return slots.length > 0
  })
}

const projectServiceTimeAllowed = (project, now = new Date()) => {
  const slots = Array.isArray(project?.service_time_slots) ? project.service_time_slots : []
  if (slots.length === 0) return true

  const durationMinutes = Number(project?.duration || 0)
  if (!Number.isFinite(durationMinutes) || durationMinutes <= 0) return false

  for (const slot of slots) {
    const startTime = String(slot?.start_time || '').trim()
    const match = startTime.match(/^(\d{2}):(\d{2})$/)
    if (!match) continue

    const hour = Number(match[1])
    const minute = Number(match[2])
    for (let offset = -1; offset <= 1; offset++) {
      const candidateDate = new Date(now)
      candidateDate.setDate(candidateDate.getDate() + offset)
      if (!serviceTimeSlotMatchesDate(slot, candidateDate)) continue

      const startAt = new Date(candidateDate)
      startAt.setHours(hour, minute, 0, 0)
      const windowStart = new Date(startAt.getTime() - 60 * 60 * 1000)
      const windowEnd = new Date(startAt.getTime() + durationMinutes * 60 * 1000 - 3 * 60 * 1000)
      if (now >= windowStart && now <= windowEnd) {
        return true
      }
    }
  }

  return false
}

const getProjectServiceSlotsByUpcomingTime = (project, limit = 1, now = new Date(nowTick.value)) => {
  const slots = Array.isArray(project?.service_time_slots) ? project.service_time_slots : []
  if (slots.length === 0 || limit <= 0) return []

  const times = []
  const seen = new Set()
  for (const [slotIndex, slot] of slots.entries()) {
    const startTime = String(slot?.start_time || '').trim()
    const match = startTime.match(/^(\d{2}):(\d{2})$/)
    if (!match) continue
    const hour = Number(match[1])
    const minute = Number(match[2])
    if (!Number.isFinite(hour) || !Number.isFinite(minute)) continue

    for (let offset = 0; offset <= 370; offset++) {
      const candidateDate = new Date(now)
      candidateDate.setDate(candidateDate.getDate() + offset)
      if (!serviceTimeSlotMatchesDate(slot, candidateDate)) continue

      const startAt = new Date(candidateDate)
      startAt.setHours(hour, minute, 0, 0)
      if (startAt.getTime() < now.getTime()) continue
      const key = `${slotIndex}:${startAt.getTime()}`
      if (!seen.has(key)) {
        seen.add(key)
        times.push({ slot, startAt, slotIndex })
      }
      break
    }
  }

  return times
    .sort((a, b) => {
      const diff = a.startAt.getTime() - b.startAt.getTime()
      return diff !== 0 ? diff : a.slotIndex - b.slotIndex
    })
    .slice(0, limit)
}

const formatServiceTimeSlotForCard = (slot) => {
  const startTime = String(slot?.start_time || '').trim()
  if (!startTime) return ''
  const weekdays = ['', '周一', '周二', '周三', '周四', '周五', '周六', '周日']
  if (String(slot?.recurrence_type || 'weekly') === 'monthly') {
    const monthDay = Number(slot?.month_day || 0)
    if (!Number.isFinite(monthDay) || monthDay <= 0) return ''
    return `${monthDay}号 ${startTime}`
  }
  const weekday = Number(slot?.weekday || 0)
  const weekdayText = weekdays[weekday] || ''
  return weekdayText ? `${weekdayText} ${startTime}` : startTime
}

const getCardServiceTimeLines = (card) => {
  const serviceTimeProjects = getCardServiceTimeProjects(card)

  if (serviceTimeProjects.length === 0) return []
  if (serviceTimeProjects.length === 1) {
    return getProjectServiceSlotsByUpcomingTime(serviceTimeProjects[0], 4).map((item) => formatServiceTimeSlotForCard(item.slot))
  }

  return serviceTimeProjects.map((project) => {
    const nextItem = getProjectServiceSlotsByUpcomingTime(project, 1)[0]
    if (!nextItem) return ''
    const name = String(project?.name || '').trim() || '项目'
    return `${name}：${formatServiceTimeSlotForCard(nextItem.slot)}`
  }).filter(Boolean)
}

const serviceTimeSlotMatchesDate = (slot, date) => {
  if (String(slot?.recurrence_type || 'weekly') === 'monthly') {
    return Number(slot?.month_day || 0) === date.getDate()
  }
  const jsDay = date.getDay()
  const weekday = jsDay === 0 ? 7 : jsDay
  return Number(slot?.weekday || 0) === weekday
}

const canGenerateVerifyCodeForProject = (projectId) => {
  if (!projectId) return true
  const project = (selectedCard.value?.projects || []).find(p => Number(p.id) === Number(projectId))
  if (!project || projectServiceTimeAllowed(project)) {
    return true
  }
  alert('当前不在卡片项目服务时间')
  return false
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

const doGenerateVerifyCode = async (projectId, options = {}) => {
  if (!options?.appointmentId && !canGenerateVerifyCodeForProject(projectId)) {
    return false
  }
  const payload = {}
  if (projectId) payload.project_id = Number(projectId)
  if (options?.appointmentId) payload.appointment_id = Number(options.appointmentId)
  const res = await cardApi.generateVerifyCode(Number(selectedCard.value.id), payload)
  verifyCode.value = res.data.data.code
  const expireAt = new Date(res.data.data.expire_at * 1000)
  codeExpireTime.value = expireAt.toLocaleTimeString()
  verifyCodeMode.value = options?.appointmentId ? 'appointment_checkin' : 'verify'

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
  return true
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
  showMultiServiceBookingModal.value = false
  showAppointmentConfirmModal.value = false
  showAppointmentModal.value = false
  stopVerifyStatusPoll()
  verifyCode.value = ''
  codeExpireTime.value = ''
  verifyQrDataUrl.value = ''
  verifyCodeProject.value = null
  verifyCodeMode.value = 'verify'
  selectedVerifyProjectId.value = null
  selectedAppointmentProjectId.value = null
  selectedTechnicianId.value = null
  selectedTimeSlot.value = ''
  timeSlots.value = []
  timeSlotError.value = ''
  availableTechnicians.value = []
  multiServiceBookingSlots.value = []
  loadingMultiServiceBookingSlots.value = false
  submittingMultiServiceBooking.value = false
  hasActiveAppointment.value = false
  selectedCardAppointment.value = null
  selectedCardCanArriveNow.value = false
  resetCardTouchState()
  resetNoticeTouchState()
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

const openAppointmentArrivalVerifyFlowFromAction = async () => {
  closeActionSheet()
  await ensureSelectedCardForAppointment()

  const appointmentProjectId = Number(selectedCardAppointment.value?.project_id || 0)
  const projects = selectedCard.value?.projects || []
  const fallbackProjectId = projects.length === 1 ? Number(projects[0].id || 0) : 0
  const projectId = appointmentProjectId > 0 ? appointmentProjectId : fallbackProjectId

  generatingVerifyCode.value = true
  try {
    await doGenerateVerifyCode(projectId > 0 ? projectId : null, { appointmentId: Number(selectedCardAppointment.value?.id || 0) })
    showVerifyCodeModal.value = true
    await startVerifyStatusPoll()
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generatingVerifyCode.value = false
  }
}

const openVerifyCodeFlowFromAction = async () => {
  if (selectedCardMerchantClosed.value) {
    alert(getSelectedCardMerchantClosedMessage())
    return
  }
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
    const generated = await doGenerateVerifyCode(onlyProjectId)
    if (!generated) return
    showVerifyCodeModal.value = true
    await startVerifyStatusPoll()
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generatingVerifyCode.value = false
  }
}

const getMultiServiceBookingSlotKey = (slot) => `${Number(slot?.project_id || 0)}:${String(slot?.slot_start_at || '').trim()}`

const formatMultiServiceSlotDateTime = (value) => formatDateTime(value)

const fetchSelectedCardMultiServiceBookingSlots = async (cardId) => {
  const id = Number(cardId || selectedCard.value?.id || 0)
  if (!id) {
    multiServiceBookingSlots.value = []
    return
  }
  loadingMultiServiceBookingSlots.value = true
  try {
    const res = await cardApi.getMultiServiceBookingSlots(id)
    multiServiceBookingSlots.value = Array.isArray(res?.data?.data) ? res.data.data : []
  } catch (_) {
    multiServiceBookingSlots.value = []
  } finally {
    loadingMultiServiceBookingSlots.value = false
  }
}

const openMultiServiceBookingModalFromAction = async () => {
  closeActionSheet()
  const cardId = Number(selectedCard.value?.id || 0)
  if (!cardId) return
  await ensureSelectedCardForAppointment()
  await fetchSelectedCardMultiServiceBookingSlots(cardId)
  showMultiServiceBookingModal.value = true
}

const refreshSelectedCardAfterMultiServiceBookingChange = async () => {
  const cardId = Number(selectedCard.value?.id || 0)
  if (!cardId) return
  await Promise.all([
    ensureSelectedCardForAppointment(),
    fetchSelectedCardMultiServiceBookingSlots(cardId),
    fetchCards()
  ])
}

const createMultiServiceBooking = async (slot) => {
  const cardId = Number(selectedCard.value?.id || 0)
  if (!cardId) return
  submittingMultiServiceBooking.value = true
  try {
    await cardApi.createMultiServiceBooking(cardId, {
      project_id: Number(slot?.project_id || 0),
      slot_start_at: String(slot?.slot_start_at || '').trim()
    })
    await refreshSelectedCardAfterMultiServiceBookingChange()
  } catch (err) {
    alert(err.response?.data?.error || '预约失败')
  } finally {
    submittingMultiServiceBooking.value = false
  }
}

const cancelMultiServiceBooking = async (slot) => {
  const cardId = Number(selectedCard.value?.id || 0)
  const bookingId = Number(slot?.current_user_booking_id || 0)
  if (!cardId || !bookingId) return
  submittingMultiServiceBooking.value = true
  try {
    await cardApi.cancelMultiServiceBooking(cardId, bookingId)
    await refreshSelectedCardAfterMultiServiceBookingChange()
  } catch (err) {
    alert(err.response?.data?.error || '取消预约失败')
  } finally {
    submittingMultiServiceBooking.value = false
  }
}

const openStartPendingUsageFromAction = () => {
  const cardId = Number(selectedCard.value?.id || 0)
  const sessionId = String(selectedCard.value?.startPendingUsageSessionId || '').trim()
  if (!cardId || !sessionId) return

  closeActionSheet()
  router.push({
    path: `/user/cards/${cardId}`,
    query: {
      scrollToUsages: '1',
      session_id: sessionId,
      openStartQr: '1'
    }
  })
}

const handleAppointmentAction = async () => {
  const cardId = Number(selectedCard.value?.id || 0)
  if (!cardId) return

  if (hasActiveAppointment.value) {
    if (selectedCardCanArriveNow.value && String(selectedCardAppointment.value?.status || '').trim() === 'confirmed') {
      await openAppointmentArrivalVerifyFlowFromAction()
      return
    }
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
  if (selectedCardMerchantClosed.value) {
    alert(getSelectedCardMerchantClosedMessage())
    return
  }
  generatingVerifyCode.value = true
  try {
    const generated = await doGenerateVerifyCode(selectedVerifyProjectId.value)
    if (!generated) return
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
  const projects = appointmentProjects.value || []
  let slotsLoaded = false
  if (projects.length > 0) {
    preparingAppointmentModal.value = true
    try {
      skipNextAppointmentProjectReload.value = true
      selectedAppointmentProjectId.value = Number(projects[0].id)
      slotsLoaded = await loadTimeSlots(selectedDate.value)
    } finally {
      preparingAppointmentModal.value = false
    }
  } else {
    slotsLoaded = await loadTimeSlots(selectedDate.value)
  }
  if (slotsLoaded) {
    showAppointmentModal.value = true
  }
}

const closeAppointmentModal = () => {
  showAppointmentModal.value = false
  resetAppointmentState()
}

const loadTimeSlots = async (date) => {
  if (!selectedCard.value?.merchant_id) return false
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
    return true
  } catch (err) {
    timeSlots.value = []
    availableTechnicians.value = []
    selectedTimeSlot.value = ''
    showAppointmentConfirmModal.value = false
    timeSlotError.value = `获取可用时间段失败: ${err.response?.data?.error || err.message}`
    alert(timeSlotError.value)
    return false
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
  const raw = String(timeStr).trim().replace('T', ' ')
  const match = raw.match(/\b(\d{2}):(\d{2})(?::\d{2})?\b/)
  if (match) return `${match[1]}:${match[2]}`
  return raw
}

const confirmAppointment = async () => {
  if (!canSubmitAppointment.value) return

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
  if (!canSubmitAppointment.value) return
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
  if (skipNextAppointmentProjectReload.value) {
    skipNextAppointmentProjectReload.value = false
    return
  }
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

  resetCardTouchState()
  resetNoticeTouchState()

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
