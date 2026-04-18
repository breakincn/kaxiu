<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center justify-between border-b">
      <div class="flex items-center gap-2">
        <span class="text-primary font-bold text-xl">卡包</span>
        <span class="text-gray-400 text-xs">kabao.shop</span>
      </div>
      <div class="flex items-center gap-3">
        <button
          type="button"
          class="p-1 text-gray-500 hover:text-primary"
          @click="onTopScanClick"
          @touchstart="onTopScanTouchStart"
          @touchmove="onTopScanTouchMove"
          @touchend="onTopScanTouchEnd"
          @touchcancel="onTopScanTouchEnd"
          style="-webkit-touch-callout: none; -webkit-user-select: none; user-select: none;"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4h-1a2 2 0 00-2 2v1m0 10v1a2 2 0 002 2h1m10-16h1a2 2 0 012 2v1m0 10v1a2 2 0 01-2 2h-1"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 11h8m-8 4h8"/>
          </svg>
        </button>
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
          <p class="text-gray-500 text-sm mt-1">{{ currentAccountName }}</p>
        </div>
        <div class="flex gap-2">
          <router-link
            v-if="canDirectSaleManage"
            to="/merchant/shop-manage"
            class="px-3 py-2 bg-slate-600 text-white rounded-lg text-sm font-medium hover:bg-slate-700 transition-colors"
          >
            售卡管理
          </router-link>
          <router-link
            v-if="canCardIssue"
            to="/merchant/issue-card"
            class="px-3 py-2 bg-slate-600 text-white rounded-lg text-sm font-medium hover:bg-slate-700 transition-colors"
          >
            发卡/开卡
          </router-link>
        </div>
      </div>
    </div>

    <div v-if="showHandCardModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4" @click="onHandCardMaskClick">
      <div class="w-full max-w-sm bg-white rounded-xl p-4 shadow-lg" @click.stop>
        <div class="flex items-center justify-between">
          <div class="text-gray-800 font-medium text-base">分配手牌</div>
          <button @click="closeHandCardModal" class="p-1 text-gray-500">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="text-gray-500 text-sm mt-1">请输入本次核销对应的手牌号</div>
        <input
          v-model="handCardInput"
          type="text"
          inputmode="numeric"
          pattern="[0-9]*"
          placeholder="例如：H001"
          class="w-full mt-3 px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
        />
        <div v-if="handCardError" class="text-red-600 text-sm mt-2">{{ handCardError }}</div>
        <div class="mt-4 flex gap-2">
          <button
            @click="submitPendingVerifyHandCard(false)"
            :disabled="submittingHandCard"
            class="px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium disabled:opacity-50"
          >
            跳过分配
          </button>
          <button
            @click="submitPendingVerifyHandCard(true)"
            :disabled="submittingHandCard"
            class="flex-1 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ submittingHandCard ? '提交中...' : '确认绑定' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 营业状态按钮 -->
    <div v-if="canBusinessStatusUpdate" class="px-4 pt-4">
      <button
        @click="showBusinessStatusModal = true"
        :class="[
          'w-full py-3.5 rounded-lg font-medium text-base transition-colors',
          merchant.is_open
            ? 'bg-green-500 text-white hover:bg-green-600'
            : 'bg-red-500 text-white hover:bg-red-600'
        ]"
      >
        {{ merchant.is_open ? '营业中' : '打烊' }}
      </button>
    </div>

    <!-- 数据统计卡片 -->
    <div v-if="visibleStatsCount > 0" class="px-4 pt-3 pb-3 grid gap-3" :class="visibleStatsGridClass">
      <button
        v-if="canDirectSaleManage && merchant.support_direct_sale && pendingDirectPurchases > 0"
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-gray-100"
        @click="goToDirectPurchaseOrders"
      >
        <div class="text-gray-600 text-sm mb-1">待确认订单</div>
        <div class="text-3xl font-bold" :class="pendingDirectPurchases > 0 ? 'text-red-500' : 'text-gray-400'">{{ pendingDirectPurchases }}</div>
        <div class="text-gray-500 text-sm">单</div>
      </button>
      <button
        v-if="showAppointmentSummaryCard"
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-gray-100"
        @click="selectTab('appointment')"
      >
        <div class="text-gray-600 text-sm mb-1">待确认预约</div>
        <div class="text-3xl font-bold" :class="appointmentSummaryCount > 0 ? 'text-orange-500' : 'text-gray-400'">{{ appointmentSummaryCount }}</div>
        <div class="text-gray-500 text-sm">人</div>
      </button>
      <button
        v-if="showExceptionSummaryCard"
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-red-100"
        @click="selectTab(showTableTab ? 'table' : 'exception')"
      >
        <div class="text-gray-600 text-sm mb-1">待处理异常</div>
        <div class="text-3xl font-bold" :class="exceptionSummaryCount > 0 ? 'text-red-500' : 'text-gray-400'">{{ exceptionSummaryCount }}</div>
        <div class="text-gray-500 text-sm">单</div>
      </button>
    </div>

    <div v-if="showMerchantSchedulePublishingPanel" class="px-4 pt-1 pb-3">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-start justify-between gap-3">
          <div>
            <div class="font-medium text-gray-800">排班管理</div>
            <div class="text-sm text-gray-500 mt-1">先核对次日客服状态并标记请假，再确认正式发布次日排班。</div>
          </div>
          <button
            @click="handleSchedulePublishingPrimaryAction"
            :disabled="schedulePublishingSubmitting || schedulePublishingPrimaryAction.disabled"
            :class="schedulePublishingPrimaryAction.className"
            class="px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap disabled:opacity-50"
          >
            {{ schedulePublishingSubmitting ? schedulePublishingPrimaryAction.submittingText : schedulePublishingPrimaryAction.label }}
          </button>
        </div>
        <div class="mt-4">
          <div class="text-sm text-gray-500">
            次日排班日期：<span class="text-gray-700">{{ schedulePublishingDate }}</span>
          </div>
        </div>
        <div v-if="schedulePublishingError" class="mt-3 text-sm text-red-500">{{ schedulePublishingError }}</div>
        <div v-else-if="schedulePublishingLoading" class="mt-3 text-sm text-gray-400">读取排班中...</div>
        <div v-else-if="schedulePublishings.length === 0" class="mt-3 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-400 text-center">
          当前次日暂无客服排班视图
        </div>
        <div v-else class="mt-3 space-y-2">
          <div v-for="row in schedulePublishings" :key="getSchedulePublishingRowKey(row)" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="font-medium text-gray-800">{{ formatScheduleTechnicianLabel(row) }}</div>
                <div class="mt-1 text-sm text-gray-500">{{ formatDateTime(row.start_at) }} - {{ formatDateTime(row.end_at) }}</div>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <div class="px-2 py-1 rounded-full text-xs font-medium" :class="getSchedulePublishingStatusClass(getEffectiveSchedulePublishingStatus(row))">
                  {{ getSchedulePublishingStatusText(getEffectiveSchedulePublishingStatus(row)) }}
                </div>
                <button
                  v-if="getEffectiveSchedulePublishingStatus(row) === 'leave'"
                  @click="unmarkScheduleLeave(row)"
                  class="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm"
                >
                  销假
                </button>
              </div>
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              <button
                v-if="shouldShowScheduleAffectedAppointments(row)"
                @click="viewSchedulePublishingAffectedAppointments(row)"
                class="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm"
              >
                查看受影响预约
              </button>
              <button
                v-if="['published', 'unpublished', 'canceled'].includes(getEffectiveSchedulePublishingStatus(row))"
                @click="markScheduleLeave(row)"
                class="px-3 py-2 bg-orange-500 text-white rounded-lg text-sm"
              >
                {{ getEffectiveSchedulePublishingStatus(row) === 'published' ? '标记请假' : '请假' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tab 切换 -->
    <div class="px-4 border-b bg-white overflow-x-auto overscroll-x-contain [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      <div class="flex gap-2 min-w-max whitespace-nowrap">
      <button
        v-if="showVerifyTab"
        @click="selectTab('verify')"
        :class="[
          'shrink-0 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'verify'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        <span class="inline-flex items-center gap-[4px]">
          <span>扫码核销</span>
          <span
            v-if="todayVerifyCount > 0"
            :class="[
              'inline-flex min-w-[18px] h-[18px] px-1 items-center justify-center rounded-full text-[11px] leading-none font-semibold',
              currentTab === 'verify'
                ? 'bg-primary text-white'
                : 'bg-orange-500 text-white'
            ]"
          >
            {{ todayVerifyCount }}
          </span>
        </span>
      </button>
      <button
        v-if="showAppointmentTab"
        @click="selectTab('appointment')"
        :class="[
          'shrink-0 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'appointment'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        <span class="inline-flex items-center gap-[4px]">
          <span>预约</span>
          <span
            v-if="todayAppointmentCount > 0"
            :class="[
              'inline-flex min-w-[18px] h-[18px] px-1 items-center justify-center rounded-full text-[11px] leading-none font-semibold',
              currentTab === 'appointment'
                ? 'bg-primary text-white'
                : 'bg-orange-500 text-white'
            ]"
          >
            {{ todayAppointmentCount }}
          </span>
        </span>
      </button>
      <button
        v-if="showFinishTab"
        @click="selectTab('finish')"
        :class="[
          'shrink-0 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'finish'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        {{ replaceTerms('扫码结单', merchant) }}
      </button>
      <button
        v-if="showNoticeTab"
        @click="selectTab('notice')"
        :class="[
          'shrink-0 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'notice'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        通知
      </button>
      <button
        v-if="showServiceTab"
        @click="selectTab('service')"
        :class="[
          'shrink-0 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'service'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        <span class="inline-flex items-center gap-[4px]">
          <span>服务</span>
          <span
            v-if="pendingStartServiceCount > 0"
            :class="[
              'inline-flex min-w-[18px] h-[18px] px-1 items-center justify-center rounded-full text-[11px] leading-none font-semibold',
              currentTab === 'service'
                ? 'bg-primary text-white'
                : 'bg-orange-500 text-white'
            ]"
          >
            {{ pendingStartServiceCount }}
          </span>
        </span>
      </button>
      <button
        v-if="showCardsTab"
        @click="selectTab('cards')"
        :class="[
          'shrink-0 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'cards'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        {{ canVerify ? '卡片' : '售卡' }}
      </button>
      <button
        v-if="showTechnicianBoardTab"
        @click="selectTab('board')"
        :class="[
          'shrink-0 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'board'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        看板
      </button>
      <button
        v-if="showTableTab"
        @click="selectTab('table')"
        :class="[
          'shrink-0 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'table'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        看板
      </button>
      </div>
    </div>

    <!-- 看板 -->
    <div v-if="currentTab === 'table' && showTableTab" class="py-2">
      <Table :embedded="true">
        <template #exception-content>
          <div class="pb-4 space-y-4">
        <div class="rounded-xl border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-700">
          集中处理超时待处理、异常结案、履约争议、脏数据收口、补偿处理，以及改签/改派异常。
        </div>
        <div v-if="appointmentPanelGroups.length > 0" class="space-y-4">
          <div v-for="group in appointmentPanelGroups" :key="`table-${group.key}`" class="space-y-4">
            <div v-if="group.title" class="px-1 text-sm font-medium text-gray-500">{{ group.title }}</div>
            <div v-for="appt in group.items" :key="`table-${appt.id}`" class="bg-white rounded-xl p-4 shadow-sm">
              <div class="flex justify-between items-start">
                <div>
                  <div class="font-medium text-gray-800">{{ appt.user?.nickname || appt.user_id }} <span class="ml-2 text-gray-500 text-sm font-normal">{{ formatAppointmentTechnicianDisplay(appt) }}</span></div>
                  <div v-if="getAppointmentIdDisplay(appt)" class="text-gray-500 text-sm mt-1">预约号: {{ getAppointmentIdDisplay(appt) }}</div>
                  <div v-if="getAppointmentCardTypeDisplay(appt)" class="text-gray-500 text-sm mt-1">预约卡片: {{ getAppointmentCardTypeDisplay(appt) }}</div>
                  <div v-if="getAppointmentCardNoDisplay(appt)" class="text-gray-500 text-sm mt-1">预约卡号: {{ getAppointmentCardNoDisplay(appt) }}</div>
                  <div v-if="getAppointmentProjectDisplay(appt)" class="text-gray-500 text-sm mt-1">预约项目: {{ getAppointmentProjectDisplay(appt) }}</div>
                  <div class="text-gray-500 text-sm mt-1">预约时间: {{ formatDateTime(appt.appointment_time) }}</div>
                  <div v-if="appt.status === 'pending' && getPendingCountdown(appt) !== null" :class="getPendingCountdownClass(appt)" class="mt-1">
                    {{ getPendingCountdownDisplay(appt) }}
                  </div>
                  <div v-if="appt.status === 'confirmed' && getAppointmentCountdown(appt) !== null && !isServiceTimeExpired(appt)" :class="getServiceCountdownClass(appt)" class="mt-1">
                    服务开始: {{ getServiceCountdownDisplay(appt) }}
                  </div>
                  <div
                    v-if="getAppointmentRiskHint(appt)"
                    class="mt-2 rounded-lg px-3 py-2 text-sm"
                    :class="getAppointmentRiskHintClass(appt)"
                  >
                    {{ getAppointmentRiskHint(appt) }}
                  </div>
                  <div
                    v-if="isCrossDayUnfinishedAppointment(appt)"
                    class="mt-2 rounded-lg px-3 py-2 text-sm bg-red-50 text-red-700 border border-red-100 space-y-1"
                  >
                    <div>责任归属：{{ getLiabilityText(appt.liability_level) }}</div>
                    <div>异常原因：{{ getReasonText(appt.disruption_reason) }}</div>
                  </div>
                  <div
                    v-if="getLatestPendingRescheduleRequest(appt)"
                    class="mt-2 rounded-lg px-3 py-2 text-sm border"
                    :class="isMerchantConfirmationPending(appt) ? 'bg-orange-50 text-orange-700 border-orange-100' : 'bg-blue-50 text-blue-700 border-blue-100'"
                  >
                    <div class="font-medium">
                      {{ isMerchantConfirmationPending(appt) ? '待商户确认改签' : '已向用户发起改签提议' }}
                    </div>
                    <div class="mt-1">
                      提议时间：{{ formatDateTime(getLatestPendingRescheduleRequest(appt)?.new_appointment_time) }}
                    </div>
                    <div v-if="getRescheduleRequestTechnicianText(getLatestPendingRescheduleRequest(appt))" class="mt-1">
                      {{ getRescheduleRequestTechnicianText(getLatestPendingRescheduleRequest(appt)) }}
                    </div>
                  </div>
                  <div
                    v-if="getLatestPendingCancelRequest(appt)"
                    class="mt-2 rounded-lg px-3 py-2 text-sm border"
                    :class="isMerchantCancelConfirmationPending(appt) ? 'bg-orange-50 text-orange-700 border-orange-100' : 'bg-red-50 text-red-700 border-red-100'"
                  >
                    <div class="font-medium">
                      {{ isMerchantCancelConfirmationPending(appt) ? '待商户确认取消' : '已向用户发起取消申请' }}
                    </div>
                    <div v-if="getLatestPendingCancelRequest(appt)?.reason" class="mt-1">
                      取消原因：{{ getLatestPendingCancelRequest(appt)?.reason }}
                    </div>
                    <div v-if="getLatestPendingCancelRequest(appt)?.objection_note" class="mt-1">
                      异议说明：{{ getLatestPendingCancelRequest(appt)?.objection_note }}
                    </div>
                  </div>
                  <div v-if="appt.resolution_note" class="mt-2 rounded-lg px-3 py-2 text-sm bg-gray-50 text-gray-700 border border-gray-100">
                    处理备注: {{ appt.resolution_note }}
                  </div>
                  <div v-if="(appt.compensations || []).length > 0" class="mt-2 space-y-2">
                    <div
                      v-for="comp in appt.compensations"
                      :key="comp.id"
                      class="rounded-lg px-3 py-2 text-sm bg-gray-50 text-gray-700 border border-gray-100"
                    >
                      <div class="font-medium">补偿：{{ getCompensationTypeText(comp.type) }}</div>
                      <div v-if="getCompensationValueText(comp)" class="mt-1">{{ getCompensationValueText(comp) }}</div>
                      <div class="mt-1 text-gray-500">{{ comp.reason }}</div>
                    </div>
                  </div>
                  <div v-if="getLatestForceMajeureReliefRequest(appt)" class="mt-2 rounded-lg px-3 py-2 text-sm bg-gray-50 text-gray-700 border border-gray-100">
                    <div class="font-medium">不可抗力申请：{{ getForceMajeureStatusText(getLatestForceMajeureReliefRequest(appt)?.status) }}</div>
                    <div class="mt-1">发起方：{{ getForceMajeureActorText(getLatestForceMajeureReliefRequest(appt)?.proposed_by_type) }}</div>
                    <div v-if="getLatestForceMajeureReliefRequest(appt)?.reason" class="mt-1">原因：{{ getLatestForceMajeureReliefRequest(appt)?.reason }}</div>
                  </div>
                </div>
                <span :class="getStatusBadgeClass(appt)">
                  {{ getStatusText(appt) }}
                </span>
              </div>

              <div class="flex gap-2 mt-3">
                <button
                  @click="openAppointmentDetailModal(appt)"
                  class="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm"
                >
                  结算详情
                </button>
                <template v-if="canOperateAppointment(appt)">
                  <button
                    v-if="appt.status === 'pending' && !isPendingExpired(appt)"
                    @click="confirmAppointment(appt.id)"
                    class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                  >
                    确认预约
                  </button>
                  <button
                    v-if="appt.status === 'pending' && !isPendingExpired(appt) && !getLatestPendingCancelRequest(appt)"
                    @click="cancelAppointment(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    取消
                  </button>
                  <button
                    v-if="appt.status === 'pending' && isPendingExpired(appt)"
                    disabled
                    class="flex-1 py-2 bg-gray-100 text-gray-400 rounded-lg text-sm font-medium cursor-not-allowed"
                  >
                    未确认预约
                  </button>
                  <button
                    v-if="appt.status === 'confirmed' && !getLatestPendingCancelRequest(appt)"
                    @click="cancelAppointment(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    取消
                  </button>
                  <button
                    v-if="isMerchantCancelConfirmationPending(appt)"
                    @click="acceptAppointmentCancelRequest(appt)"
                    class="flex-1 py-2 bg-red-500 text-white rounded-lg text-sm font-medium"
                  >
                    同意取消
                  </button>
                  <button
                    v-if="isMerchantCancelConfirmationPending(appt)"
                    @click="rejectAppointmentCancelRequest(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    拒绝取消
                  </button>
                  <button
                    v-if="appt.status === 'arrived' && appt.service_session_id && !isCrossDayUnfinishedAppointment(appt) && !isAppointmentAlreadyInService(appt)"
                    @click="reassignAppointmentService(appt)"
                    class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                  >
                    改派其他客服
                  </button>
                  <button
                    v-if="isMerchantConfirmationPending(appt)"
                    @click="acceptAppointmentRescheduleRequest(appt)"
                    class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                  >
                    确认改签
                  </button>
                  <button
                    v-if="isMerchantConfirmationPending(appt)"
                    @click="rejectAppointmentRescheduleRequest(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    拒绝
                  </button>
                  <button
                    v-if="canCancelAppointmentRescheduleRequest(appt)"
                    @click="cancelAppointmentRescheduleRequest(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    撤销改签
                  </button>
                  <button
                    v-if="shouldShowAppointmentReschedule(appt)"
                    @click="openAppointmentRescheduleModal(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    发起改签
                  </button>
                  <button
                    v-if="isCrossDayUnfinishedAppointment(appt)"
                    @click="closeAppointmentException(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    异常结案
                  </button>
                  <button
                    v-if="shouldShowAppointmentCompensation(appt)"
                    @click="openAppointmentCompensationModal(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    补偿
                  </button>
                  <button
                    v-if="canCreateForceMajeureRelief(appt)"
                    @click="createMerchantForceMajeureRelief(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    不可抗力
                  </button>
                  <button
                    v-if="canAcceptForceMajeureRelief(appt)"
                    @click="acceptAppointmentForceMajeureRelief(appt)"
                    class="px-4 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                  >
                    确认可抗力
                  </button>
                  <button
                    v-if="canRejectForceMajeureRelief(appt)"
                    @click="rejectAppointmentForceMajeureRelief(appt)"
                    class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
                  >
                    拒绝可抗力
                  </button>
                </template>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="bg-white rounded-xl px-4 py-10 text-center text-gray-400">
          {{ appointmentPanelEmptyText }}
        </div>
          </div>
        </template>
      </Table>
    </div>

    <div v-if="currentTab === 'board' && showTechnicianBoardTab" class="py-2">
      <Table :embedded="true" :service-only="true" :technician-own-only="true" :hide-service-filter="true" />
    </div>

    <!-- 预约 / 异常中心 -->
    <div
      v-if="((currentTab === 'appointment' && showAppointmentTab) || (currentTab === 'exception' && showExceptionStandaloneTab))"
      class="px-4 py-4 space-y-4"
    >
      <div
        v-if="showTechnicianSchedulePublishingPanel"
        class="bg-white rounded-xl p-4 shadow-sm"
      >
        <div class="flex items-start justify-between gap-3">
          <div>
            <div class="font-medium text-gray-800">我的预约排班</div>
            <div class="text-sm text-gray-500 mt-1">{{ technicianSchedulePublishingSubtitle }}</div>
          </div>
          <button
            @click="handleSchedulePublishingPrimaryAction"
            :disabled="schedulePublishingSubmitting || schedulePublishingPrimaryAction.disabled"
            :class="schedulePublishingPrimaryAction.className"
            class="px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap disabled:opacity-50"
          >
            {{ schedulePublishingSubmitting ? schedulePublishingPrimaryAction.submittingText : schedulePublishingPrimaryAction.label }}
          </button>
        </div>
        <div class="mt-4 text-sm text-gray-500">
          排班日期：<span class="text-gray-700">{{ schedulePublishingDate }}</span>
        </div>
        <div v-if="technicianSchedulePublishingHint" class="mt-2 text-xs text-gray-500">
          {{ technicianSchedulePublishingHint }}
        </div>
        <div v-if="schedulePublishingError" class="mt-3 text-sm text-red-500">{{ schedulePublishingError }}</div>
        <div v-else-if="schedulePublishingLoading" class="mt-3 text-sm text-gray-400">读取排班中...</div>
        <div v-else-if="visibleSchedulePublishings.length === 0" class="mt-3 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-400 text-center">
          当前暂无可展示的预约排班
        </div>
        <div v-else class="mt-3 space-y-2">
          <div v-for="row in visibleSchedulePublishings" :key="getSchedulePublishingRowKey(row)" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
            <div class="mb-2 text-xs text-gray-400">
              {{ formatScheduleBookingOpenLabel(row) }}
            </div>
            <div class="font-medium text-gray-800">{{ formatDateTime(row.start_at) }} - {{ formatDateTime(row.end_at) }}</div>
            <div class="mt-2 flex items-center justify-between gap-3">
              <div class="text-sm text-gray-500 truncate">{{ row.technician?.name || '-' }}</div>
              <div class="flex items-center gap-2 shrink-0">
                <span class="px-2 py-1 rounded-full text-xs font-medium" :class="getSchedulePublishingStatusClass(getEffectiveSchedulePublishingStatus(row))">
                  {{ getSchedulePublishingStatusText(getEffectiveSchedulePublishingStatus(row)) }}
                </span>
                <span v-if="row.published_at" class="text-xs text-gray-400">{{ formatDateTime(row.published_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div
        v-if="showExceptionStandaloneContent"
        class="rounded-xl border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-700"
      >
        集中处理超时待处理、异常结案、履约争议、脏数据收口、补偿处理，以及改签/改派异常。
      </div>
      <div v-if="displayedAppointmentPanelGroups.length > 0" class="space-y-4">
        <div v-for="group in displayedAppointmentPanelGroups" :key="group.key" class="space-y-4">
          <div v-if="group.title" class="px-1 text-sm font-medium text-gray-500">{{ group.title }}</div>
          <div v-for="appt in group.items" :key="appt.id" class="bg-white rounded-xl p-4 shadow-sm">
          <div class="flex justify-between items-start">
            <div>
              <div class="font-medium text-gray-800">{{ appt.user?.nickname || appt.user_id }}</div>
              <div v-if="showAppointmentTechnicianLine(appt)" class="text-gray-500 text-sm mt-1">预约技师：{{ getAppointmentTechnicianLineText(appt) }}</div>
              <div v-if="getAppointmentIdDisplay(appt)" class="text-gray-500 text-sm mt-1">预约号: {{ getAppointmentIdDisplay(appt) }}</div>
              <div v-if="getAppointmentCardTypeDisplay(appt)" class="text-gray-500 text-sm mt-1">预约卡片: {{ getAppointmentCardTypeDisplay(appt) }}</div>
              <div v-if="getAppointmentCardNoDisplay(appt)" class="text-gray-500 text-sm mt-1">预约卡号: {{ getAppointmentCardNoDisplay(appt) }}</div>
              <div v-if="getAppointmentProjectDisplay(appt)" class="text-gray-500 text-sm mt-1">预约项目: {{ getAppointmentProjectDisplay(appt) }}</div>
              <div class="text-gray-500 text-sm mt-1">预约时间: {{ formatDateTime(appt.appointment_time) }}</div>
              <div v-if="appt.status === 'pending' && getPendingCountdown(appt) !== null" :class="getPendingCountdownClass(appt)" class="mt-1">
                {{ getPendingCountdownDisplay(appt) }}
              </div>
              <!-- 已确认预约的服务开始倒计时 -->
              <div v-if="appt.status === 'confirmed' && getAppointmentCountdown(appt) !== null && !isServiceTimeExpired(appt)" :class="getServiceCountdownClass(appt)" class="mt-1">
                服务开始: {{ getServiceCountdownDisplay(appt) }}
              </div>
              <div
                v-if="getAppointmentRiskHint(appt)"
                class="mt-2 rounded-lg px-3 py-2 text-sm"
                :class="getAppointmentRiskHintClass(appt)"
              >
                {{ getAppointmentRiskHint(appt) }}
              </div>
              <div
                v-if="isCrossDayUnfinishedAppointment(appt)"
                class="mt-2 rounded-lg px-3 py-2 text-sm bg-red-50 text-red-700 border border-red-100 space-y-1"
              >
                <div>责任归属：{{ getLiabilityText(appt.liability_level) }}</div>
                <div>异常原因：{{ getReasonText(appt.disruption_reason) }}</div>
              </div>
              <div
                v-if="getLatestPendingRescheduleRequest(appt)"
                class="mt-2 rounded-lg px-3 py-2 text-sm border"
                :class="isMerchantConfirmationPending(appt) ? 'bg-orange-50 text-orange-700 border-orange-100' : 'bg-blue-50 text-blue-700 border-blue-100'"
              >
                <div class="font-medium">
                  {{ isMerchantConfirmationPending(appt) ? '待商户确认改签' : '已向用户发起改签提议' }}
                </div>
                <div class="mt-1">
                  提议时间：{{ formatDateTime(getLatestPendingRescheduleRequest(appt)?.new_appointment_time) }}
                </div>
                <div v-if="getRescheduleRequestTechnicianText(getLatestPendingRescheduleRequest(appt))" class="mt-1">
                  {{ getRescheduleRequestTechnicianText(getLatestPendingRescheduleRequest(appt)) }}
                </div>
              </div>
              <div
                v-if="getLatestPendingCancelRequest(appt)"
                class="mt-2 rounded-lg px-3 py-2 text-sm border"
                :class="isMerchantCancelConfirmationPending(appt) ? 'bg-orange-50 text-orange-700 border-orange-100' : 'bg-red-50 text-red-700 border-red-100'"
              >
                <div class="font-medium">
                  {{ isMerchantCancelConfirmationPending(appt) ? '待商户确认取消' : '已向用户发起取消申请' }}
                </div>
                <div v-if="getLatestPendingCancelRequest(appt)?.reason" class="mt-1">
                  取消原因：{{ getLatestPendingCancelRequest(appt)?.reason }}
                </div>
                <div v-if="getLatestPendingCancelRequest(appt)?.objection_note" class="mt-1">
                  异议说明：{{ getLatestPendingCancelRequest(appt)?.objection_note }}
                </div>
              </div>
              <div v-if="appt.resolution_note" class="mt-2 rounded-lg px-3 py-2 text-sm bg-gray-50 text-gray-700 border border-gray-100">
                处理备注: {{ appt.resolution_note }}
              </div>
              <div v-if="(appt.compensations || []).length > 0" class="mt-2 space-y-2">
                <div
                  v-for="comp in appt.compensations"
                  :key="comp.id"
                  class="rounded-lg px-3 py-2 text-sm bg-gray-50 text-gray-700 border border-gray-100"
                >
                  <div class="font-medium">补偿：{{ getCompensationTypeText(comp.type) }}</div>
                  <div v-if="getCompensationValueText(comp)" class="mt-1">{{ getCompensationValueText(comp) }}</div>
                  <div class="mt-1 text-gray-500">{{ comp.reason }}</div>
                </div>
              </div>
              <div v-if="getLatestForceMajeureReliefRequest(appt)" class="mt-2 rounded-lg px-3 py-2 text-sm bg-gray-50 text-gray-700 border border-gray-100">
                <div class="font-medium">不可抗力申请：{{ getForceMajeureStatusText(getLatestForceMajeureReliefRequest(appt)?.status) }}</div>
                <div class="mt-1">发起方：{{ getForceMajeureActorText(getLatestForceMajeureReliefRequest(appt)?.proposed_by_type) }}</div>
                <div v-if="getLatestForceMajeureReliefRequest(appt)?.reason" class="mt-1">原因：{{ getLatestForceMajeureReliefRequest(appt)?.reason }}</div>
              </div>
            </div>
            <span :class="getStatusBadgeClass(appt)">
              {{ getStatusText(appt) }}
            </span>
          </div>

          <div class="flex gap-2 mt-3">
            <button
              @click="openAppointmentDetailModal(appt)"
              class="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm"
            >
              结算详情
            </button>
            <template v-if="canOperateAppointment(appt)">
              <button
                v-if="appt.status === 'pending' && !isPendingExpired(appt)"
                @click="confirmAppointment(appt.id)"
                class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
              >
                确认预约
              </button>
              <button
                v-if="appt.status === 'pending' && !isPendingExpired(appt) && !getLatestPendingCancelRequest(appt)"
                @click="cancelAppointment(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                取消
              </button>
              <button
                v-if="appt.status === 'pending' && isPendingExpired(appt)"
                disabled
                class="flex-1 py-2 bg-gray-100 text-gray-400 rounded-lg text-sm font-medium cursor-not-allowed"
              >
                未确认预约
              </button>
              <button
                v-if="appt.status === 'confirmed' && !getLatestPendingCancelRequest(appt)"
                @click="cancelAppointment(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                取消
              </button>
              <button
                v-if="isMerchantCancelConfirmationPending(appt)"
                @click="acceptAppointmentCancelRequest(appt)"
                class="flex-1 py-2 bg-red-500 text-white rounded-lg text-sm font-medium"
              >
                同意取消
              </button>
              <button
                v-if="isMerchantCancelConfirmationPending(appt)"
                @click="rejectAppointmentCancelRequest(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                拒绝取消
              </button>
              <button
                v-if="appt.status === 'arrived' && appt.service_session_id && !isCrossDayUnfinishedAppointment(appt) && !isAppointmentAlreadyInService(appt)"
                @click="reassignAppointmentService(appt)"
                class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
              >
                改派其他客服
              </button>
              <button
                v-if="isMerchantConfirmationPending(appt)"
                @click="acceptAppointmentRescheduleRequest(appt)"
                class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
              >
                确认改签
              </button>
              <button
                v-if="isMerchantConfirmationPending(appt)"
                @click="rejectAppointmentRescheduleRequest(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                拒绝
              </button>
              <button
                v-if="canCancelAppointmentRescheduleRequest(appt)"
                @click="cancelAppointmentRescheduleRequest(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                撤销改签
              </button>
              <button
                v-if="shouldShowAppointmentReschedule(appt)"
                @click="openAppointmentRescheduleModal(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                发起改签
              </button>
              <button
                v-if="isCrossDayUnfinishedAppointment(appt)"
                @click="closeAppointmentException(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                异常结案
              </button>
              <button
                v-if="shouldShowAppointmentCompensation(appt)"
                @click="openAppointmentCompensationModal(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                补偿
              </button>
              <button
                v-if="canCreateForceMajeureRelief(appt)"
                @click="createMerchantForceMajeureRelief(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                不可抗力
              </button>
              <button
                v-if="canAcceptForceMajeureRelief(appt)"
                @click="acceptAppointmentForceMajeureRelief(appt)"
                class="px-4 py-2 bg-primary text-white rounded-lg text-sm font-medium"
              >
                确认可抗力
              </button>
              <button
                v-if="canRejectForceMajeureRelief(appt)"
                @click="rejectAppointmentForceMajeureRelief(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                拒绝可抗力
              </button>
            </template>
          </div>
        </div>
        </div>
        <button
          v-if="showAppointmentGroupsLoadMore"
          type="button"
          class="w-full rounded-lg border border-orange-200 bg-orange-50 px-4 py-3 text-sm font-medium text-orange-600 disabled:opacity-50"
          :disabled="appointmentGroupsLoadingMore"
          @click="loadMoreAppointmentGroups"
        >
          {{ appointmentGroupsLoadingMore ? '加载中...' : '更多' }}
        </button>
      </div>
      <div v-else class="text-center py-12 text-gray-400">
        {{ appointmentPanelEmptyText }}
      </div>
    </div>

    <!-- 扫码核销 -->
    <div v-if="currentTab === 'verify' && showVerifyTab" class="px-4 py-4">
      <!-- 默认显示大按钮 -->
      <div v-if="!showVerifyInput" class="bg-white rounded-xl p-4 shadow-sm">
        <button
          @click="goScanVerify"
          class="w-full py-3 bg-primary text-white rounded-lg font-medium"
        >
          扫码核销
        </button>

        <!-- 叫号控制按钮（运营客服端） -->
        <div v-if="showQueueControlInVerify" class="mt-4 flex gap-2">
          <button
            @click="startMerchantQueue"
            :disabled="queueStatusUpdating || !merchantQueuePaused"
            :class="[
              'flex-1 py-3 rounded-lg font-medium transition-colors',
              merchantQueuePaused
                ? 'bg-green-500 text-white hover:bg-green-600'
                : 'bg-gray-100 text-gray-400 cursor-not-allowed'
            ]"
          >
            {{ queueStatusUpdating ? '处理中...' : '开始叫号' }}
          </button>
          <button
            @click="pauseMerchantQueue"
            :disabled="queueStatusUpdating || merchantQueuePaused"
            :class="[
              'flex-1 py-3 rounded-lg font-medium transition-colors',
              !merchantQueuePaused
                ? 'bg-orange-500 text-white hover:bg-orange-600'
                : 'bg-gray-100 text-gray-400 cursor-not-allowed'
            ]"
          >
            {{ queueStatusUpdating ? '处理中...' : (isQueueEnded ? '结束叫号' : '暂停叫号') }}
          </button>
        </div>
        <div v-if="showQueueControlInVerify" class="mt-2 text-center text-xs text-gray-500">
          {{ merchant.queue_mode === 'auto' ? '自动叫号' : '手动叫号' }}：{{ isQueueEnded ? '叫号已结束' : (merchantQueuePaused ? '叫号已暂停' : '叫号进行中') }}
        </div>
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

      <div v-if="merchant?.support_hand_card" class="bg-white rounded-xl p-4 shadow-sm mt-4">
        <h3 class="font-medium text-gray-800 mb-4">归还手牌</h3>
        <div class="flex gap-2">
          <input
            v-model="returnHandCardNo"
            type="text"
            inputmode="numeric"
            pattern="[0-9]*"
            placeholder="请输入手牌号"
            class="flex-1 px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
          />
          <button
            @click="queryHandCardForReturn"
            :disabled="!returnHandCardNo || queryingReturnHandCard"
            class="px-4 py-3 bg-primary text-white rounded-lg text-sm font-medium disabled:opacity-50"
          >
            {{ queryingReturnHandCard ? '查询中...' : '查询' }}
          </button>
        </div>

        <div v-if="returnHandCardError" class="text-red-600 text-sm mt-3">{{ returnHandCardError }}</div>

        <div v-if="returnHandCardUsage" class="mt-4 p-4 bg-gray-50 rounded-lg">
          <div class="text-gray-800 font-medium">{{ returnHandCardUsage.card?.user?.nickname || '用户' }}</div>
          <div class="text-gray-500 text-sm mt-2">卡号：{{ returnHandCardUsage.card?.card_no || '-' }}</div>
          <div class="text-gray-500 text-sm mt-1">单号：{{ getUsageTrackingNumber(returnHandCardUsage) }}</div>
          <div class="text-gray-500 text-sm mt-1">项目：{{ returnHandCardUsage.project?.name || '-' }}</div>
          <div class="text-gray-500 text-sm mt-1">手牌：{{ returnHandCardUsage.hand_card_no || '-' }}</div>
          <div class="text-gray-500 text-sm mt-1">核销时间：{{ formatDateTime(returnHandCardUsage.used_at) }}</div>
          <div v-if="returnHandCardUsage.service_session?.room" class="text-gray-500 text-sm mt-1">房间：{{ returnHandCardUsage.service_session.room.name || '-' }}</div>

          <div class="mt-4 flex gap-2">
            <button
              @click="confirmReturnHandCard"
              :disabled="returningHandCard"
              class="flex-1 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
            >
              {{ returningHandCard ? '归还中...' : '确认归还' }}
            </button>
            <button
              @click="cancelReturnHandCard"
              :disabled="returningHandCard"
              class="flex-1 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium disabled:opacity-50"
            >
              取消
            </button>
          </div>
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
          <div v-for="usage in todayUsages" :key="usage.id" class="flex justify-between items-start py-3 border-b last:border-0">
            <div class="flex-1">
              <div class="text-gray-800 font-medium">{{ usage.card?.user?.nickname || '用户' }}</div>
              <div class="text-gray-500 text-sm mt-1">单号：{{ getUsageTrackingNumber(usage) }}</div>
              <div class="text-gray-500 text-sm mt-1">卡号：{{ usage.card?.card_no || '-' }}</div>
              <div v-if="merchant?.support_hand_card" class="text-gray-500 text-sm mt-1">手牌：{{ usage.hand_card_no || '-' }} (<span v-if="!usage.hand_card_no && !isUsageHandCardReturned(usage)" class="text-red-500">未分配</span><span v-else-if="isUsageHandCardReturned(usage)">{{ getHandCardStatusText(usage) }}</span><span v-else-if="normalizeSessionStatus(usage.service_session_status) === 'serving'">{{ getHandCardStatusText(usage) }}</span><span v-else class="text-red-500">未归还</span>)</div>
              <div v-if="getUsageAppointmentNumber(usage)" class="text-gray-500 text-sm mt-1">
                预约号：#{{ getUsageAppointmentNumber(usage) }}
              </div>
              <div class="text-gray-500 text-sm mt-1">项目：{{ usage.project?.name || '-' }}</div>
              <div class="text-gray-500 text-sm mt-1">状态：{{ getUsageServiceStatusText(usage) }}</div>
              <div v-if="getUsageServiceTechnicianLabel(usage)" class="text-gray-500 text-sm mt-1">
                {{ getUsageServiceTechnicianLabel(usage) }}
              </div>
              <div v-if="shouldShowUsagePendingReassign(usage)" class="text-amber-600 text-sm mt-1">
                {{ usage.service_technician_unavailable_reason || ('当前' + replaceTerms('客服', merchant) + '不可服务') }}
              </div>
              <div v-if="getUsageServiceRemainingSeconds(usage) !== null" class="text-gray-500 text-sm mt-1">
                服务剩余：{{ formatRemainingSeconds(getUsageServiceRemainingSeconds(usage)) }}
              </div>
              <div class="text-gray-400 text-sm mt-1">{{ formatDateTime(usage.used_at) }}</div>
            </div>
            <div class="text-right">
              <div class="text-sm">
                核销次数：<span class="text-gray-700">{{ getUsageCountDisplayText(usage).totalTimes }}</span> / <span :class="getUsageCountColorClass(usage)">{{ getUsageCountDisplayText(usage).usedTimes }}</span> 次
              </div>
              <div class="text-gray-500 text-xs mt-1">
                {{ getVerifyOperatorPrimaryInfo(usage) }}
              </div>
              <button
                v-if="shouldShowUsagePendingReassign(usage)"
                @click="reassignUsagePendingItem(usage)"
                :disabled="isUsagePendingReassigning(usage)"
                class="mt-2 px-3 py-2 bg-orange-500 text-white rounded-lg text-xs font-medium disabled:opacity-60"
              >
                {{ isUsagePendingReassigning(usage) ? '重分配中...' : '重新分配' }}
              </button>
            </div>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
          今日暂无核销
        </div>
      </div>
    </div>

    <!-- 扫码结单 -->
    <div v-if="currentTab === 'finish' && showFinishTab" class="px-4 py-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <button
          @click="goScanFinish"
          class="w-full py-3 bg-primary text-white rounded-lg font-medium"
        >
          {{ replaceTerms('扫码结单', merchant) }}
        </button>
      </div>

      <!-- 今日结单记录 -->
      <div class="bg-white rounded-xl p-4 shadow-sm mt-4">
        <h3 class="font-medium text-gray-800 mb-4">{{ replaceTerms('今日结单记录', merchant) }}</h3>
        <div v-if="todayFinishedUsages.length > 0" class="space-y-3">
          <div v-for="usage in todayFinishedUsages" :key="usage.id" class="flex justify-between items-start py-3 border-b last:border-0">
            <div class="flex-1">
              <div class="text-gray-800 font-medium">{{ usage.card?.user?.nickname || '用户' }}</div>
              <div class="text-gray-500 text-sm mt-1">单号：{{ getUsageTrackingNumber(usage) }}</div>
              <div class="text-gray-500 text-sm mt-1">卡号：{{ usage.card?.card_no || '-' }}</div>
              <div class="text-gray-500 text-sm mt-1">项目：{{ usage.project?.name || '-' }}</div>
              <div class="text-gray-500 text-sm mt-1">状态：{{ getUsageServiceStatusText(usage) }}</div>
              <div class="text-gray-400 text-sm mt-1">{{ formatDateTime(usage.finished_at) }}</div>
            </div>
            <div class="text-right">
              <div class="text-sm">
                {{ replaceTerms('结单', merchant) }}：<span class="text-gray-700">{{ getUsageCountDisplayText(usage).totalTimes }}</span> / <span :class="getUsageCountColorClass(usage)">{{ getUsageCountDisplayText(usage).usedTimes }}</span> 次
              </div>
            </div>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
          {{ replaceTerms('今日暂无结单', merchant) }}
        </div>
      </div>
    </div>

    <!-- 通知管理 -->
    <div v-if="currentTab === 'notice' && showNoticeTab" class="px-4 py-4">
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
    <div v-if="currentTab === 'cards' && showCardsTab" class="px-4 py-4">
      <div v-if="cardsError && canVerify" class="bg-gray-50 border border-gray-100 text-gray-700 rounded-lg p-3 text-sm mb-4">
        {{ cardsError }}
      </div>

      <div v-if="routeUserCode && canVerify" ref="userCodeAnchor" class="bg-white rounded-xl p-4 shadow-sm mb-4 flex items-center justify-between">
        <div class="text-sm text-gray-700">当前仅显示该用户的卡片</div>
        <button type="button" class="text-sm text-primary" @click="clearUserCodeFilter">清除筛选</button>
      </div>

      <div class="bg-white rounded-xl p-4 shadow-sm mb-4">
        <div class="grid grid-cols-1 gap-3" :class="canVerify ? 'sm:grid-cols-2' : ''">
          <input
            v-if="canVerify"
            v-model="cardSearch.card_no"
            class="border border-gray-200 rounded-lg px-3 py-2 text-sm"
            placeholder="按卡号搜索"
          />
          <select
            v-model="cardSearch.card_type"
            class="border border-gray-200 rounded-lg px-3 py-2 text-sm"
          >
            <option value="">{{ canVerify ? '全部卡片类型' : '全部售卡类型' }}</option>
            <option
              v-for="tpl in (canVerify ? cardTemplates : sellTemplates)"
              :key="tpl.id"
              :value="tpl.name"
            >
              {{ tpl.name }}（{{ getCardTypeLabel(tpl.card_type) }}）
            </option>
          </select>
        </div>

        <div class="flex gap-2 mt-3 items-center" :class="!canVerify ? 'justify-start' : ''">
          <button
            v-if="canVerify"
            @click="searchCards"
            class="px-4 py-2 bg-primary text-white text-sm rounded-lg"
          >
            查询
          </button>
          <button
            v-if="canVerify"
            @click="resetCardSearch"
            class="px-4 py-2 bg-gray-100 text-gray-700 text-sm rounded-lg"
          >
            重置
          </button>

          <div v-if="canVerify" class="flex-1"></div>
          <button
            v-if="(!isTechnicianAuth() && merchant.support_direct_sale) || (isTechnicianAuth() && canSellCards && canVerify)"
            type="button"
            @click="loadSellTemplates"
            class="px-4 py-2 bg-slate-600 text-white text-sm rounded-lg"
          >
            售卡
          </button>
        </div>
      </div>

      <div v-if="currentDisplay === 'cards' && canVerify && cardsLoading" class="text-center py-12 text-gray-400">
        加载中...
      </div>

      <div v-else-if="currentDisplay === 'cards' && canVerify">
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

            <div v-if="merchant?.support_hand_card && card?.locked" class="mt-3 px-3 py-2 rounded-lg bg-red-50 border border-red-100">
              <div class="flex items-start gap-2">
                <svg class="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                </svg>
                <div class="flex-1">
                  <div class="text-red-600 text-sm font-medium">卡片已锁定</div>
                  <div class="text-red-500 text-xs mt-0.5">{{ getLockedHandCardTipText(card) }}</div>
                </div>
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

        <!-- 动态占位元素：当卡片数量少时增加底部高度，确保可以滚动到锚点 -->
        <div v-if="issuedCards.length > 0 && issuedCards.length <= 2" :style="{ height: getBottomSpacerHeight() }"></div>

        <div v-if="issuedCards.length === 0" class="text-center py-12 text-gray-400">
          暂无已发卡
        </div>
      </div>

      <!-- 售卡模板列表 -->
      <div v-if="currentDisplay === 'sellTemplates'">
        <div class="mb-4">
          <p class="text-sm text-gray-500">长按卡片模板生成售卡二维码</p>
        </div>
        <div v-if="filteredSellTemplates.length === 0" class="text-center py-12 text-gray-400">
          {{ sellTemplates.length === 0 ? '暂无在售卡片模板' : '没有找到匹配的卡片模板' }}
        </div>
        
        <div v-else class="template-list">
          <div 
            v-for="tpl in filteredSellTemplates" 
            :key="tpl.id" 
            class="template-item"
          >
            <div
              class="template-card"
              @click="openSellQrModal(tpl)"
              @touchstart="(e) => onTemplateTouchStart(e, tpl)"
              @touchmove="onTemplateTouchMove"
              @touchend="onTemplateTouchEnd"
              @touchcancel="onTemplateTouchEnd"
              style="-webkit-touch-callout: none; -webkit-user-select: none; user-select: none;"
            >
              <div class="template-info">
                <div class="template-name">{{ tpl.name }}</div>
                <div class="template-meta">
                  <span class="type-tag">{{ getCardTypeLabel(tpl.card_type) }}</span>
                  <span class="price">¥{{ (tpl.price / 100).toFixed(2) }}</span>
                </div>
                <div class="template-detail">
                  <span v-if="tpl.card_type !== 'balance'">{{ tpl.total_times }}次</span>
                  <span v-else>充值{{ (tpl.recharge_amount / 100).toFixed(0) }}元</span>
                  <span v-if="tpl.valid_days > 0">· {{ tpl.valid_days }}天有效</span>
                  <span v-else>· 永久有效</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 服务/会话 -->
    <div v-if="currentTab === 'service' && showServiceTab" class="px-4 py-4 space-y-4">
      <!-- 签到/状态 -->
      <div v-if="showTechnicianAttendancePanel" class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-center justify-between">
          <div>
            <div class="font-medium text-gray-800">工作人员签到</div>
            <div class="text-gray-500 text-sm mt-1" v-if="isTechnicianAuth()">当前账号：{{ getTechnicianName() }}</div>
            <div class="text-gray-500 text-sm mt-1" v-else>请使用工作人员账号登录进行签到</div>
          </div>
          <div v-if="isTechnicianAuth()" class="flex items-center gap-2">
            <button
              v-if="isTechnicianNotCheckedIn"
              :disabled="attendanceLoading"
              @click="doCheckIn"
              class="px-4 py-2 rounded-lg text-sm font-medium"
              :class="attendanceLoading ? 'bg-gray-100 text-gray-400' : 'bg-green-500 text-white'"
            >
              上班签到
            </button>
            <button
              v-else-if="showCheckOutButton"
              :disabled="attendanceLoading"
              @click="doCheckOut"
              class="px-4 py-2 rounded-lg text-sm font-medium"
              :class="attendanceLoading ? 'bg-gray-100 text-gray-400' : 'bg-primary text-white'"
            >
              下班签到
            </button>
          </div>
        </div>

        <div v-if="isTechnicianAuth()" class="mt-3">
          <div class="flex items-center justify-between">
            <div class="text-sm text-gray-600">
              当前状态：<span class="font-medium">{{ technicianCurrentStatusText }}</span>
            </div>
            <div v-if="!isTechnicianNotCheckedIn" class="flex items-center gap-2">
              <template v-if="technicianCurrentStatus === 'service_pending_settlement'">
                <button
                  :disabled="setNextPausedLoading"
                  @click="setNextStatusPaused"
                  class="px-4 py-2 rounded-lg text-sm font-medium"
                  :class="setNextPausedLoading ? 'bg-gray-100 text-gray-400' : 'bg-orange-600 text-white'"
                >
                  {{ replaceTerms('结单后暂停', merchant) }}
                </button>
              </template>

              <template v-else>
                <select
                  :value="statusSelectValue"
                  @change="handleAttendanceStatusSelectChange"
                  class="border border-gray-200 rounded-lg px-3 py-2 text-sm"
                  :class="canManualUpdateStatus ? 'w-32' : 'w-44'"
                  :disabled="!canManualUpdateStatus || attendanceUpdating"
                >
                  <!-- 叫号模式下的状态选项 -->
                  <template v-if="isQueueModeView">
                    <option value="not_checked_in" disabled>未签到</option>
                    <option value="idle">空闲</option>
                    <option value="paused" :disabled="!canManualUpdateStatus">暂停</option>
                    <option value="service_pending_presettlement" disabled>{{ getMerchantPendingStartLabel({ queueMode: true }) }}</option>
                    <option value="service_pending_settlement" disabled>上号</option>
                  </template>
                  <!-- 非叫号模式保持原有选项 -->
                  <template v-else>
                    <option value="not_checked_in" disabled>未签到</option>
                    <option value="idle">空闲</option>
                    <option value="paused" :disabled="!canManualUpdateStatus">暂停</option>
                    <option value="service_pending_presettlement" disabled>{{ getMerchantServicePendingStartLabel() }}</option>
                    <option value="service_pending_settlement" disabled>{{ getMerchantServicePendingFinishLabel() }}</option>
                  </template>
                </select>

              </template>
            </div>
          </div>
        </div>

        <div v-if="isTechnicianAuth()" class="mt-4 space-y-3">
          <div v-if="isTechnicianNotCheckedIn" class="w-full py-3 bg-gray-100 text-gray-600 rounded-lg text-center">
            {{ attendanceBlockedHint }}
          </div>
          <button
            v-else-if="showScanStartButton"
            @click="goScanStart"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium"
          >
            {{ getMerchantScanStartLabel() }}
          </button>
        </div>
      </div>

      <div v-if="isTechnicianAuth() && serviceTabUpcomingAppointments.length > 0" class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-center justify-between">
          <div>
            <div class="font-medium text-gray-800">1小时内我的预约</div>
            <div class="text-gray-500 text-sm mt-1">可在这里核对即将到店的预约，并在本页扫码上钟时兼容预约签到。</div>
          </div>
        </div>
        <div class="mt-3 space-y-3">
          <div
            v-for="appt in serviceTabUpcomingAppointments"
            :key="`service-upcoming-${appt.id}`"
            class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="font-medium text-gray-800">{{ appt.user?.nickname || '用户' }}</div>
                <div class="mt-1 text-sm text-gray-500">预约号: {{ getAppointmentIdDisplay(appt) || '-' }}</div>
                <div class="mt-1 text-sm text-gray-500">预约卡片: {{ getAppointmentCardTypeDisplay(appt) || '-' }}</div>
                <div class="mt-1 text-sm text-gray-500">预约卡号: {{ getAppointmentCardNoDisplay(appt) || '-' }}</div>
                <div class="mt-1 text-sm text-gray-500">预约项目: {{ appt.project?.name || '-' }}（{{ getAppointmentServiceMinutes(appt) }}分钟）</div>
                <div class="mt-1 text-sm text-gray-500">预约时间: {{ formatDateTime(appt.appointment_time) }}</div>
              </div>
              <span :class="getStatusBadgeClass(appt)">
                {{ getStatusText(appt) }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 叫号控制（专业客服端） -->
      <div v-if="showQueueControlInService && !queueBlockedByAttendance" class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-center justify-between">
          <div>
            <div class="font-medium text-gray-800">叫号管理</div>
            <div class="text-gray-500 text-sm mt-1">
              商户叫号状态: {{ isQueueEnded ? '已结束' : (merchantQueuePaused ? '已暂停' : '进行中') }}
              <span v-if="technicianQueuePaused" class="text-orange-500 ml-1">·您已暂停叫号</span>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button
              v-if="merchantQueuePaused"
              @click="startTechnicianQueue"
              :disabled="queueStatusUpdating"
              class="px-4 py-2 bg-green-500 text-white rounded-lg text-sm font-medium hover:bg-green-600 disabled:opacity-50"
            >
              {{ queueStatusUpdating ? '处理中...' : '开始叫号' }}
            </button>
            <template v-else>
              <button
                v-if="!technicianQueuePaused"
                @click="pauseTechnicianQueue"
                :disabled="queueStatusUpdating"
                class="px-4 py-2 bg-orange-500 text-white rounded-lg text-sm font-medium hover:bg-orange-600 disabled:opacity-50"
              >
                {{ queueStatusUpdating ? '处理中...' : (isQueueEnded ? '结束叫号' : '暂停叫号') }}
              </button>
              <button
                v-else
                @click="resumeTechnicianQueue"
                :disabled="queueStatusUpdating"
                class="px-4 py-2 bg-green-500 text-white rounded-lg text-sm font-medium hover:bg-green-600 disabled:opacity-50"
              >
                {{ queueStatusUpdating ? '处理中...' : '恢复叫号' }}
              </button>
              <button
                v-if="shouldShowContinueCall"
                @click="doContinueCall()"
                :disabled="continueCallLoading || queueStatusUpdating || continueCallBlockedSeconds > 0"
                class="px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium hover:bg-blue-600 disabled:opacity-50"
              >
                {{ continueCallLoading ? '处理中...' : (continueCallBlockedSeconds > 0 ? `等待${getMerchantPendingStartLabel({ queueMode: true }).replace('待', '')}(${continueCallBlockedSeconds}s)` : '继续叫号') }}
              </button>
            </template>
          </div>
        </div>

        <div class="mt-4">
          <div class="text-gray-800 font-medium">待叫号</div>
          <div v-if="queuePendingLoading" class="text-gray-500 text-sm mt-2">加载中...</div>
          <div v-else-if="!queueWaitingList.length" class="text-gray-500 text-sm mt-2">暂无待叫号用户</div>
          <div v-else class="mt-2 space-y-2">
            <div v-for="it in queueWaitingList" :key="String(it.usage_id)" class="flex items-start justify-between bg-gray-50 rounded-lg px-3 py-2">
              <div class="flex-1">
                <div class="text-gray-800 text-sm font-medium">
                  叫号顺序: {{ it.queue_no || '-' }}
                  <span v-if="it.user_nickname" class="text-gray-600 font-normal ml-2">{{ it.user_nickname }}</span>
                </div>
                <div class="text-gray-600 text-sm mt-1 font-mono">单号: {{ formatSessionNo(it.usage_id) }}</div>
                <div v-if="it.project_name" class="text-gray-600 text-sm mt-1">项目: {{ it.project_name }}</div>
                <div v-if="it.technician_name" class="text-gray-600 text-sm mt-1">当前客服: {{ it.technician_name }}</div>
                <div v-if="it.technician_unavailable_reason" class="text-red-500 text-sm mt-1 font-medium">
                  {{ it.technician_unavailable_reason }}
                </div>
                <div
                  v-if="getQueueWaitingItemStartPendingRemainingSeconds(it) !== null"
                  :class="['text-sm mt-1 font-medium', getQueueWaitingItemStartPendingRemainingClass(it)]"
                >
                  {{ getStartCountdownLabel(merchant, { queueMode: true }) }}：{{ formatRemainingSeconds(getQueueWaitingItemStartPendingRemainingSeconds(it)) }}
                </div>
              </div>
              <div class="ml-3 flex flex-col items-end gap-2">
                <div class="text-gray-500 text-sm whitespace-nowrap">{{ getQueueWaitingItemPhaseText(it) }}</div>
                <button
                  v-if="shouldShowQueuePendingReassign(it)"
                  @click="reassignQueuePendingItem(it)"
                  :disabled="isQueuePendingReassigning(it)"
                  class="px-3 py-1.5 bg-orange-500 text-white rounded-lg text-xs font-medium hover:bg-orange-600 disabled:opacity-50"
                >
                  {{ isQueuePendingReassigning(it) ? '重分配中...' : '重新分配' }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="mt-4">
          <div class="text-gray-800 font-medium">服务中</div>
          <div v-if="queuePendingLoading" class="text-gray-500 text-sm mt-2">加载中...</div>
          <div v-else-if="!queueServingList.length" class="text-gray-500 text-sm mt-2">暂无服务中用户</div>
          <div v-else class="mt-2 space-y-2">
            <div v-for="it in queueServingList" :key="`serving-${String(it.session_id || it.usage_id)}`" class="flex items-start justify-between bg-green-50 rounded-lg px-3 py-2">
              <div class="flex-1">
                <div class="text-gray-800 text-sm font-medium">
                  叫号顺序: {{ it.queue_no || '-' }}
                  <span v-if="it.user_nickname" class="text-gray-600 font-normal ml-2">{{ it.user_nickname }}</span>
                </div>
                <div class="text-gray-600 text-sm mt-1 font-mono">单号: {{ formatSessionNo(it.usage_id) }}</div>
                <div v-if="it.project_name" class="text-gray-600 text-sm mt-1">项目: {{ it.project_name }}</div>
                <div
                  v-if="getQueueServingItemRemainingSeconds(it) !== null"
                  :class="['text-sm mt-1 font-medium', getQueueServingItemRemainingClass(it)]"
                >
                  服务剩余：{{ formatRemainingSeconds(getQueueServingItemRemainingSeconds(it)) }}
                </div>
              </div>
              <div class="text-green-600 text-sm ml-3 whitespace-nowrap">{{ getQueueServingItemPhaseText(it) }}</div>
            </div>
          </div>
        </div>

        <div class="mt-4">
          <div class="text-gray-800 font-medium">超时过号等待</div>
          <div v-if="queueTimeoutWaitingLoading" class="text-gray-500 text-sm mt-2">加载中...</div>
          <div v-else-if="!queueTimeoutWaitingList.length" class="text-gray-500 text-sm mt-2">暂无超时过号等待用户</div>
          <div v-else class="mt-2 space-y-2">
            <div v-for="it in queueTimeoutWaitingList" :key="String(it.session_id || it.usage_id)" class="flex items-start justify-between bg-orange-50 rounded-lg px-3 py-2">
              <div class="flex-1">
                <div class="text-gray-800 text-sm font-medium">
                  <span class="text-orange-600">超时过号等待</span>
                  <span v-if="it.user_nickname" class="text-gray-600 font-normal ml-2">{{ it.user_nickname }}</span>
                </div>
                <div v-if="formatTimeoutWaitingTitle(it)" class="text-gray-600 text-sm mt-1">
                  {{ formatTimeoutWaitingTitle(it) }}
                </div>
                <div class="text-gray-600 text-sm mt-1 font-mono">单号: {{ formatSessionNo(it.usage_id) }}</div>
                <div v-if="it.project_name" class="text-gray-600 text-sm mt-1">项目: {{ it.project_name }}</div>
                <div v-if="it.timeout_at" class="text-gray-500 text-xs mt-1">超时: {{ formatDateTime(it.timeout_at) }}</div>
              </div>
              <div class="text-orange-600 text-sm ml-3 whitespace-nowrap">等待插队</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 房间管理入口 -->
      <div v-if="showRoomManageCard" class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-center justify-between">
          <div>
            <div class="font-medium text-gray-800">{{ (isQueueModeView && isTechnicianAuth()) ? '叫号信息' : '房间管理' }}</div>

            <!-- 叫号模式 + 专业客服：显示窗口/台号、叫号、单号、项目 -->
            <template v-if="isQueueModeView && isTechnicianAuth() && !queueBlockedByAttendance">
              <div v-if="queueCallInfo?.window_no" class="text-gray-700 text-sm mt-1">
                {{ windowTerm }}: {{ queueCallInfo.window_no }}
              </div>
              <div class="text-gray-700 text-sm mt-1">
                叫号: {{ queueCallQueueNoText }}
              </div>
              <div v-if="queueCallInfo?.session" class="text-gray-700 text-sm mt-1">
                阶段: {{ queueCallPhaseText }}
              </div>
              <div
                v-if="getQueueCallSessionStartPendingRemainingSeconds() !== null"
                :class="['text-sm mt-1 font-medium', getQueueCallSessionStartPendingRemainingClass()]"
              >
                {{ getStartCountdownLabel(merchant, { queueMode: true }) }}：{{ formatRemainingSeconds(getQueueCallSessionStartPendingRemainingSeconds()) }}
              </div>
              <div v-if="queueCallInfo?.tracking_id" class="text-gray-700 text-sm mt-1 font-mono">
                单号: {{ formatSessionNo(queueCallInfo.tracking_id) }}
              </div>
              <div v-if="queueCallInfo?.session?.project_name" class="text-gray-700 text-sm mt-1">
                项目: {{ queueCallInfo.session.project_name }}
              </div>
              <div v-if="getQueueCallSessionRemainingSeconds() !== null" class="text-blue-600 text-sm mt-1 font-medium">
                服务剩余：{{ formatRemainingSeconds(getQueueCallSessionRemainingSeconds()) }}
              </div>
            </template>

            <!-- 非叫号模式：保留原房间管理展示 -->
            <template v-else>
              <div v-if="roomManageSession" class="text-gray-700 text-sm mt-1">
                {{ roomManagePhaseText }} 房间号: {{ roomManageRoomText }}
              </div>
              <div v-if="roomManageSession" class="text-gray-700 text-sm mt-1 font-mono">
                单号: {{ formatSessionNo(roomManageTrackingId) }}
              </div>
              <div v-if="roomManageSession" class="text-gray-700 text-sm mt-1">
                项目: {{ roomManageProjectName }}
              </div>
              <div
                v-if="getRoomManageStartPendingCountdownSeconds() !== null"
                :class="['text-sm mt-1 font-medium', getRemainingSecondsClass(getRoomManageStartPendingCountdownSeconds())]"
              >
                {{ getStartCountdownLabel(merchant) }}：{{ formatStartCountdownSeconds(getRoomManageStartPendingCountdownSeconds()) }}
              </div>
              <div
                v-if="getRoomManageOccupancyDurationText()"
                class="text-sm mt-1 font-medium text-gray-700"
              >
                房间占用时间：{{ getRoomManageOccupancyDurationText() }}
              </div>
            </template>
          </div>

          <!-- 叫号模式 + 专业客服：不展示房间管理按钮 -->
          <button
            v-if="!(isQueueModeView && isTechnicianAuth()) && canRoomManage && merchant?.support_room"
            @click="router.push('/merchant/rooms')"
            class="px-4 py-2 bg-slate-600 text-white rounded-lg text-sm font-medium"
          >
            管理房间
          </button>
        </div>
      </div>

      <div class="bg-white rounded-xl p-4 shadow-sm">
        <h3 class="font-medium text-gray-800 mb-4">{{ getMerchantTodayStartRecordLabel() }}</h3>
        <div v-if="startUsagesLoading" class="text-center text-gray-400 py-4">
          加载中...
        </div>
        <div v-else-if="todayStartUsages.length > 0" class="space-y-3">
          <div v-for="usage in todayStartUsages" :key="usage.id" class="py-3 border-b last:border-0">
            <div class="flex justify-between items-start">
              <div class="flex-1 min-w-0">
                <div class="text-gray-800 font-medium">{{ usage.card?.user?.nickname || '用户' }}</div>
                <div class="text-gray-500 text-sm mt-1">卡号：{{ usage.card?.card_no || '-' }}</div>
                <div class="text-gray-500 text-sm mt-1">单号：{{ getUsageTrackingNumber(usage) }}</div>
                <div v-if="getUsageAppointmentNumber(usage)" class="text-gray-500 text-sm mt-1">
                  预约号：#{{ getUsageAppointmentNumber(usage) }}
                </div>
                <div v-if="getUsageRoomText(usage)" class="text-gray-500 text-sm mt-1">
                  {{ getUsageRoomText(usage) }}
                </div>
                <div class="text-gray-500 text-sm mt-1">项目：{{ formatProjectNameWithDuration(usage.project) }}</div>
              </div>
              <div class="text-right flex-shrink-0 ml-3">
                <div class="text-sm">
                  核销次数：<span class="text-gray-700">{{ getUsageCountDisplayText(usage).totalTimes }}</span> / <span :class="getUsageCountColorClass(usage)">{{ getUsageCountDisplayText(usage).usedTimes }}</span> 次
                </div>
              </div>
            </div>
            <div v-if="getUsageApprovedExtendInfoLines(usage).length > 0" class="w-full text-green-700 text-sm mt-1 rounded-lg bg-green-50 px-2 py-1 leading-6">
              <div v-for="line in getUsageApprovedExtendInfoLines(usage)" :key="line" class="whitespace-nowrap">{{ line }}</div>
            </div>
            <div
              v-if="shouldShowUsageServiceRemainingSeconds(usage)"
              :class="['text-sm mt-1 font-medium', getRemainingSecondsClass(getUsageServiceRemainingSeconds(usage))]"
            >
              服务剩余：{{ formatRemainingSeconds(getUsageServiceRemainingSeconds(usage)) }}
            </div>
            <div
              v-if="getUsageStartPendingCountdownSeconds(usage) !== null"
              :class="['text-sm mt-1 font-medium', getRemainingSecondsClass(getUsageStartPendingCountdownSeconds(usage))]"
            >
              {{ getStartCountdownLabel(merchant, { queueMode: isQueueModeMerchant(merchant) }) }}：{{ formatStartCountdownSeconds(getUsageStartPendingCountdownSeconds(usage)) }}
            </div>
            <div v-if="getUsageRoomOccupancyDurationText(usage)" class="text-gray-500 text-sm mt-1">
              房间占用时间：{{ getUsageRoomOccupancyDurationText(usage) }}
            </div>
            <div class="text-gray-500 text-sm mt-1">状态：{{ getUsageServiceStatusText(usage) }}</div>
            <div class="text-gray-400 text-sm mt-1">{{ formatDateTime(usage.used_at) }}</div>
            <button
              v-if="hasPendingExtendRequest(usage)"
              type="button"
              class="mt-3 px-3 py-1.5 rounded bg-orange-500 text-white text-xs font-medium"
              @click="openExtendRequestReview(usage)"
            >
              加钟申请
            </button>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
          {{ getMerchantTodayNoStartRecordLabel() }}
        </div>
      </div>

      
    </div>

    <div v-if="showExtendRequestReviewModal && selectedExtendRequest" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4" @click.self="closeExtendRequestReview">
      <div class="bg-white w-full max-w-sm rounded-2xl p-4">
        <div class="flex items-center justify-between mb-3">
          <div class="font-medium text-gray-800">加钟申请</div>
          <button class="text-gray-500" @click="closeExtendRequestReview">关闭</button>
        </div>

        <div class="text-gray-700 text-sm space-y-2 mb-4">
          <div>用户：{{ selectedExtendUsage?.card?.user?.nickname || '用户' }}</div>
          <div>单号：{{ getUsageTrackingNumber(selectedExtendUsage) }}</div>
          <div>项目：{{ selectedExtendRequest.project?.name || '-' }}</div>
          <div>加钟时长：{{ selectedExtendRequest.minutes || 0 }} 分钟</div>
          <div>申请时间：{{ formatDateTime(selectedExtendRequest.created_at) }}</div>
        </div>
        <textarea
          v-model="extendRejectReason"
          rows="3"
          placeholder="拒绝原因（拒绝时选填）"
          class="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm mb-4"
        ></textarea>
        <div class="flex gap-2">
          <button
            @click="rejectExtendRequest"
            :disabled="extendReviewLoading"
            class="flex-1 px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium disabled:opacity-50"
          >
            拒绝
          </button>
          <button
            :disabled="extendReviewLoading"
            @click="approveExtendRequest"
            class="flex-1 px-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ extendReviewLoading ? '处理中...' : '确定' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 技师加钟弹窗 -->
    <div v-if="showExtendModalVisible" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
      <div class="bg-white w-full max-w-sm rounded-2xl p-4">
        <div class="flex items-center justify-between mb-3">
          <div class="font-medium text-gray-800">加钟</div>
          <button class="text-gray-500" @click="closeExtendModal">关闭</button>
        </div>

        <div class="text-gray-600 text-sm mb-3">延长服务时间（5~180分钟）</div>
        <div class="mb-4">
          <input v-model.number="extendMinutes" type="number" min="5" max="180" placeholder="分钟" class="w-full px-4 py-3 border border-gray-200 rounded-lg">
        </div>
        <div class="flex gap-2">
          <button @click="closeExtendModal" class="flex-1 px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium">取消</button>
          <button :disabled="!extendMinutes || extendMinutes < 5 || extendMinutes > 180 || extendLoadingIds.has(extendSession?.id)" @click="doExtendSession" class="flex-1 px-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50">
            {{ extendLoadingIds.has(extendSession?.id) ? '加钟中...' : '确认' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showAppointmentRescheduleModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4" @click.self="closeAppointmentRescheduleModal">
      <div class="bg-white w-full max-w-lg rounded-2xl p-4 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-3">
          <div class="font-medium text-gray-800">改签到新时间</div>
          <button class="text-gray-500" @click="closeAppointmentRescheduleModal">关闭</button>
        </div>
        <div class="text-sm text-gray-600">当前预约：{{ rescheduleAppointmentTarget?.user?.nickname || '-' }} / {{ getAppointmentProjectDisplay(rescheduleAppointmentTarget) || '默认项目' }}</div>
        <div
          v-if="appointmentRescheduleEligibility && !appointmentRescheduleEligibility.allowed"
          class="mt-3 rounded-lg bg-orange-50 text-orange-700 border border-orange-100 px-3 py-2 text-sm"
        >
          {{ appointmentRescheduleEligibility.reason || '当前不可改签' }}
        </div>
        <div class="mt-3">
          <div class="text-sm text-gray-600 mb-1">新日期</div>
          <input
            v-model="appointmentRescheduleForm.date"
            type="date"
            :min="appointmentRescheduleDateMin"
            :max="appointmentRescheduleDateMax"
            :disabled="appointmentRescheduleEligibility && !appointmentRescheduleEligibility.allowed"
            class="w-full px-4 py-3 border border-gray-200 rounded-lg disabled:bg-gray-50 disabled:text-gray-400"
          />
          <div v-if="appointmentRescheduleDateHint" class="text-xs text-gray-400 mt-2">{{ appointmentRescheduleDateHint }}</div>
        </div>
        <div class="mt-3">
          <div class="text-sm text-gray-600 mb-1">可选时间</div>
          <div v-if="appointmentRescheduleComparisonHint" class="mb-2 text-xs text-gray-500">{{ appointmentRescheduleComparisonHint }}</div>
          <div v-if="appointmentRescheduleLoading" class="text-sm text-gray-400 py-3">加载中...</div>
          <div v-else-if="appointmentRescheduleSlots.length === 0" class="text-sm text-gray-400 py-3">该日期暂无可改签时间</div>
          <div v-else class="flex flex-wrap gap-2">
            <button
              v-for="slot in appointmentRescheduleSlots"
              :key="slot.time"
              type="button"
              @click="selectAppointmentRescheduleSlot(slot)"
              :class="selectedAppointmentRescheduleTime === slot.time ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
              class="px-3 py-2 rounded-lg border text-sm text-left"
            >
              <div>{{ slot.label || slot.time.slice(11, 16) }}</div>
              <div
                v-if="slot.comparison_label"
                class="mt-1 text-[11px]"
                :class="selectedAppointmentRescheduleTime === slot.time ? 'text-white/80' : getRescheduleSlotComparisonClass(slot.comparison_kind)"
              >
                {{ slot.comparison_label }}
              </div>
            </button>
          </div>
        </div>
        <div v-if="selectedAppointmentRescheduleCandidates.length > 0" class="mt-3">
          <div class="text-sm text-gray-600 mb-1">可选客服</div>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="item in selectedAppointmentRescheduleCandidates"
              :key="item.technician_id"
              type="button"
              @click="toggleAppointmentRescheduleTechnician(item.technician_id)"
              :class="appointmentRescheduleForm.technician_id === item.technician_id ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
              class="px-3 py-2 rounded-lg border text-sm"
            >
              {{ item.label }}
            </button>
          </div>
        </div>
        <div v-if="selectedAppointmentRescheduleTechnicianText" class="mt-3 rounded-lg bg-primary-light text-primary border border-primary/10 px-3 py-2 text-sm">
          已选客服：{{ selectedAppointmentRescheduleTechnicianText }}
        </div>
        <div class="mt-3">
          <div class="text-sm text-gray-600 mb-1">改签原因</div>
          <textarea v-model="appointmentRescheduleForm.reason" rows="3" class="w-full px-4 py-3 border border-gray-200 rounded-lg" placeholder="例如：客户主动改到明天下午"></textarea>
        </div>
        <div class="mt-4 flex gap-2">
          <button @click="closeAppointmentRescheduleModal" class="flex-1 px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium">取消</button>
          <button @click="submitAppointmentReschedule" :disabled="appointmentRescheduleSubmitting || !selectedAppointmentRescheduleTime || !appointmentRescheduleForm.reason.trim() || (appointmentRescheduleEligibility && !appointmentRescheduleEligibility.allowed)" class="flex-1 px-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50">
            {{ appointmentRescheduleSubmitting ? '提交中...' : '确认改签' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showAppointmentCompensationModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4" @click.self="closeAppointmentCompensationModal">
      <div class="bg-white w-full max-w-lg rounded-2xl p-4">
        <div class="flex items-center justify-between mb-3">
          <div class="font-medium text-gray-800">预约补偿</div>
          <button class="text-gray-500" @click="closeAppointmentCompensationModal">关闭</button>
        </div>
        <div class="text-sm text-gray-600">仅在门店未兑现预约承诺时使用。补偿会形成正式记录并立即执行。</div>
        <div class="mt-3">
          <div class="text-sm text-gray-600 mb-1">补偿类型</div>
          <select v-model="appointmentCompensationForm.type" class="w-full px-4 py-3 border border-gray-200 rounded-lg">
            <option value="extra_times">补次数</option>
            <option value="extend_minutes">补时长</option>
            <option value="discount_note">优惠减免记录</option>
            <option value="other_note">其他补偿记录</option>
          </select>
        </div>
        <div v-if="['extra_times', 'extend_minutes'].includes(appointmentCompensationForm.type)" class="mt-3">
          <div class="text-sm text-gray-600 mb-1">{{ appointmentCompensationForm.type === 'extra_times' ? '补偿次数' : '补偿分钟数' }}</div>
          <input v-model.number="appointmentCompensationForm.value" type="number" min="1" class="w-full px-4 py-3 border border-gray-200 rounded-lg" />
        </div>
        <div class="mt-3">
          <div class="text-sm text-gray-600 mb-1">补偿原因</div>
          <input v-model="appointmentCompensationForm.reason" type="text" class="w-full px-4 py-3 border border-gray-200 rounded-lg" placeholder="例如：到店等待超过15分钟" />
        </div>
        <div class="mt-3">
          <div class="text-sm text-gray-600 mb-1">备注</div>
          <textarea v-model="appointmentCompensationForm.remark" rows="3" class="w-full px-4 py-3 border border-gray-200 rounded-lg" placeholder="可填写赠送内容或减免说明"></textarea>
        </div>
        <div class="mt-4 flex gap-2">
          <button @click="closeAppointmentCompensationModal" class="flex-1 px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium">取消</button>
          <button @click="submitAppointmentCompensation" :disabled="appointmentCompensationSubmitting || !appointmentCompensationForm.reason.trim()" class="flex-1 px-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50">
            {{ appointmentCompensationSubmitting ? '处理中...' : '确认补偿' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showAppointmentDetailModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4" @click.self="closeAppointmentDetailModal">
      <div class="bg-white w-full max-w-2xl rounded-2xl overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div>
            <div class="font-medium text-gray-800">预约结算详情</div>
            <div class="text-sm text-gray-500 mt-1">
              {{ appointmentDetailTarget?.user?.nickname || appointmentDetailTarget?.user_id || '-' }}
              <span class="mx-1">/</span>
              {{ getAppointmentProjectDisplay(appointmentDetailTarget) || '默认项目' }}
            </div>
          </div>
          <button class="text-gray-500" @click="closeAppointmentDetailModal">关闭</button>
        </div>
        <div class="p-5 max-h-[75vh] overflow-y-auto">
          <div v-if="appointmentDetailLoading" class="py-12 text-center text-gray-400">加载中...</div>
          <div v-else-if="appointmentDetailError" class="py-12 text-center text-red-500">{{ appointmentDetailError }}</div>
          <div v-else class="space-y-4">
            <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">预约号</div>
                <div class="mt-1 font-medium text-gray-800">{{ getAppointmentIdDisplay(appointmentDetailTarget) || '-' }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">结算状态</div>
                <div class="mt-1 font-medium text-gray-800">{{ getAppointmentSettlementStatusText(appointmentDetailSettlement?.settlement_status_snapshot || appointmentDetailTarget?.settlement_status_snapshot) || '待结算' }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">责任归属</div>
                <div class="mt-1 font-medium text-gray-800">{{ getLiabilityText(appointmentDetailSettlement?.liability_level || appointmentDetailTarget?.liability_level) }}</div>
              </div>
            </div>
            <div v-if="appointmentDetailTarget?.reserved_start_at || appointmentDetailTarget?.reserved_end_at || appointmentDetailTarget?.cancel_deadline_at" class="rounded-xl border border-gray-200 bg-white px-4 py-4 text-sm text-gray-700 space-y-2">
              <div class="font-medium text-gray-800">预约规则信息</div>
              <div v-if="appointmentDetailTarget?.reserved_start_at">锁定开始：{{ formatDateTime(appointmentDetailTarget.reserved_start_at) }}</div>
              <div v-if="appointmentDetailTarget?.reserved_end_at">锁定结束：{{ formatDateTime(appointmentDetailTarget.reserved_end_at) }}</div>
              <div v-if="appointmentDetailTarget?.cancel_deadline_at">最晚可直接取消：{{ formatDateTime(appointmentDetailTarget.cancel_deadline_at) }}</div>
              <div class="pt-2 border-t border-gray-100 text-xs text-gray-500 space-y-1">
                <div>执行口径：当前按 `balanced` 规则执行。</div>
                <div>取消截止口径：默认按预约前一日 16:00 作为直接取消截止时间，超过后应走取消申请。</div>
                <div>晚间补开口径：默认按预约前一日 18:00 判断次日晚间资源补开与调度。</div>
              </div>
            </div>
            <div v-if="appointmentDetailSettlement?.latest_reason || appointmentDetailTarget?.disruption_reason" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-700">
              原因：{{ getReasonText(appointmentDetailSettlement?.latest_reason || appointmentDetailTarget?.disruption_reason) }}
            </div>
            <div v-if="appointmentDetailTarget?.merchant_cancel_reason || appointmentDetailTarget?.user_rebuttal_note" class="rounded-xl border border-gray-200 bg-white px-4 py-4 text-sm text-gray-700 space-y-3">
              <div v-if="appointmentDetailTarget?.merchant_cancel_reason">
                <div class="font-medium text-gray-800">商户取消原因</div>
                <div class="mt-1">{{ appointmentDetailTarget.merchant_cancel_reason }}</div>
              </div>
              <div v-if="appointmentDetailTarget?.user_rebuttal_note">
                <div class="font-medium text-gray-800">用户抗辩</div>
                <div class="mt-1">{{ appointmentDetailTarget.user_rebuttal_note }}</div>
              </div>
            </div>
            <div v-if="appointmentDetailTarget?.technician_id" class="rounded-xl border border-gray-200 bg-white px-4 py-4 text-sm text-gray-700">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <div class="font-medium text-gray-800">客服月度异常统计</div>
                  <div class="mt-1 text-gray-500">查看当前预约客服在指定月份的免责与责任账本统计</div>
                </div>
                <div class="flex items-center gap-2">
                  <button
                    @click="viewTechnicianMonthlyDisruptions(appointmentDetailTarget)"
                    class="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm"
                  >
                    查看统计
                  </button>
                  <button
                    v-if="appointmentDetailHasRepairOverview"
                    @click="viewAppointmentRepairOverview(appointmentDetailTarget)"
                    class="px-4 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm"
                  >
                    异常修复
                  </button>
                </div>
              </div>
            </div>
            <div v-if="appointmentDetailSummary" class="rounded-xl border border-gray-200 bg-white px-4 py-4">
              <div class="font-medium text-gray-800">拖堂累计</div>
              <div class="grid grid-cols-2 gap-3 mt-3 text-sm">
                <div class="rounded-lg bg-gray-50 px-3 py-3">
                  <div class="text-xs text-gray-400">累计拖堂</div>
                  <div class="mt-1 font-medium text-gray-800">{{ appointmentDetailSummary.total_delay_minutes || 0 }} 分钟</div>
                </div>
                <div class="rounded-lg bg-gray-50 px-3 py-3">
                  <div class="text-xs text-gray-400">计入补偿桶</div>
                  <div class="mt-1 font-medium text-gray-800">{{ appointmentDetailSummary.total_credited_minutes || 0 }} 分钟</div>
                </div>
              </div>
            </div>
            <div v-if="appointmentDetailDelayLedgers.length > 0" class="space-y-2">
              <div class="font-medium text-gray-800">拖堂账本</div>
              <div v-for="ledger in appointmentDetailDelayLedgers" :key="ledger.id" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-700">
                <div class="font-medium text-gray-800">延迟 {{ ledger.delay_minutes }} 分钟</div>
                <div class="mt-1">计入补偿桶 {{ ledger.credited_minutes }} 分钟</div>
                <div v-if="ledger.delay_compensation_value > 0" class="mt-1">累计补偿值 {{ ledger.delay_compensation_value }}</div>
                <div class="mt-1 text-gray-500">账本状态：{{ getDelayLedgerStatusText(ledger) }}</div>
              </div>
            </div>
            <div v-if="appointmentDetailCompensations.length > 0" class="space-y-2">
              <div class="font-medium text-gray-800">正式补偿记录</div>
              <div v-for="comp in appointmentDetailCompensations" :key="comp.id" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-700">
                <div class="font-medium text-gray-800">{{ getCompensationTypeText(comp.type) }}</div>
                <div v-if="getCompensationValueText(comp)" class="mt-1">{{ getCompensationValueText(comp) }}</div>
                <div v-if="comp.reason" class="mt-1 text-gray-500">{{ getReasonText(comp.reason) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showAppointmentRepairOverviewModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4" @click.self="closeAppointmentRepairOverviewModal">
      <div class="bg-white w-full max-w-3xl rounded-2xl overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div>
            <div class="font-medium text-gray-800">请假异常修复详情</div>
            <div class="text-sm text-gray-500 mt-1">{{ appointmentRepairOverviewScheduleLabel || '当前预约关联排班' }}</div>
          </div>
          <button class="text-gray-500" @click="closeAppointmentRepairOverviewModal">关闭</button>
        </div>
        <div class="p-5 max-h-[75vh] overflow-y-auto">
          <div v-if="appointmentRepairOverviewLoading" class="py-12 text-center text-gray-400">加载中...</div>
          <div v-else-if="appointmentRepairOverviewError" class="py-12 text-center text-red-500">{{ appointmentRepairOverviewError }}</div>
          <div v-else class="space-y-4">
            <div v-if="appointmentRepairOverviewReason" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-700">
              {{ appointmentRepairOverviewReason }}
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">受影响预约</div>
                <div class="mt-1 font-medium text-gray-800">{{ appointmentRepairAffectedAppointments.length }} 笔</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">保护修复槽</div>
                <div class="mt-1 font-medium text-gray-800">{{ appointmentRepairProtectedSlots.length }} 个</div>
              </div>
            </div>
            <div>
              <div class="font-medium text-gray-800 mb-2">受影响预约</div>
              <div v-if="appointmentRepairAffectedAppointments.length === 0" class="rounded-xl border border-dashed border-gray-200 px-4 py-8 text-center text-sm text-gray-400">
                当前无受影响预约
              </div>
              <div v-for="item in appointmentRepairItems" :key="item.appointment?.id || item.id" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-700 mb-2">
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <div class="font-medium text-gray-800">{{ item.appointment?.user?.nickname || item.appointment?.user_id || `用户${item.appointment?.user_id || ''}` }}</div>
                    <div class="mt-1 text-gray-500">预约时间：{{ formatDateTime(item.appointment?.appointment_time) }}</div>
                    <div class="mt-1 text-gray-500">状态：{{ getAppointmentStatusText(item.appointment?.status) }}</div>
                    <div v-if="item.appointment?.disruption_reason" class="mt-1 text-gray-500">原因：{{ getReasonText(item.appointment?.disruption_reason) }}</div>
                  </div>
                  <div class="px-2 py-1 rounded-full text-xs font-medium whitespace-nowrap" :class="getRepairDecisionClass(item.decision)">
                    {{ getRepairDecisionText(item.decision) }}
                  </div>
                </div>
                <div v-if="item.reason" class="mt-3 rounded-lg bg-white px-3 py-3 border border-gray-100 text-gray-700">
                  {{ item.reason }}
                </div>
                <div v-if="item.candidate_time" class="mt-2 text-gray-500">判定命中时段：{{ formatDateTime(item.candidate_time) }}</div>
                <div v-if="item.recommendations?.length" class="mt-3">
                  <div class="text-xs text-gray-400 mb-2">保护性改签建议</div>
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
                    <div v-for="slot in item.recommendations" :key="`${item.appointment?.id}-${slot.time}`" class="rounded-lg bg-white px-3 py-3 border border-gray-100">
                      <div class="font-medium text-gray-800">{{ formatDateTime(slot.time) }}</div>
                      <div v-if="slot.comparison_label" class="mt-1 text-xs text-blue-600">{{ slot.comparison_label }}</div>
                      <div v-if="slot.recommendation_reason" class="mt-1 text-xs text-gray-500">{{ slot.recommendation_reason }}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div>
              <div class="font-medium text-gray-800 mb-2">保护修复槽</div>
              <div v-if="appointmentRepairProtectedSlots.length === 0" class="rounded-xl border border-dashed border-gray-200 px-4 py-8 text-center text-sm text-gray-400">
                当前无保护修复槽
              </div>
              <div v-for="slot in appointmentRepairProtectedSlots" :key="slot.id" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-700 mb-2">
                <div class="font-medium text-gray-800">{{ formatDateTime(slot.start_at) }} - {{ formatDateTime(slot.end_at) }}</div>
                <div v-if="slot.appointment_id" class="mt-1 text-gray-500">关联预约：#{{ slot.appointment_id }}</div>
                <div v-if="slot.status" class="mt-1 text-gray-500">状态：{{ slot.status }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showTechnicianMonthlyDisruptionModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4" @click.self="closeTechnicianMonthlyDisruptionModal">
      <div class="bg-white w-full max-w-2xl rounded-2xl overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div>
            <div class="font-medium text-gray-800">客服月度异常统计</div>
            <div class="text-sm text-gray-500 mt-1">
              {{ technicianMonthlyDisruptionTechnicianName || '-' }}
              <span class="mx-1">/</span>
              {{ technicianMonthlyDisruptionMonth || '-' }}
            </div>
          </div>
          <button class="text-gray-500" @click="closeTechnicianMonthlyDisruptionModal">关闭</button>
        </div>
        <div class="p-5 max-h-[75vh] overflow-y-auto">
          <div class="flex items-end gap-3 mb-4">
            <div>
              <div class="text-xs text-gray-400 mb-1">统计月份</div>
              <input
                v-model="technicianMonthlyDisruptionMonth"
                type="month"
                class="px-3 py-2 border border-gray-300 rounded-lg text-sm"
              />
            </div>
            <button
              @click="reloadTechnicianMonthlyDisruptions"
              :disabled="technicianMonthlyDisruptionLoading || !technicianMonthlyDisruptionTechnicianId"
              class="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {{ technicianMonthlyDisruptionLoading ? '加载中...' : '刷新统计' }}
            </button>
          </div>

          <div v-if="technicianMonthlyDisruptionLoading" class="py-12 text-center text-gray-400">加载中...</div>
          <div v-else-if="technicianMonthlyDisruptionError" class="py-12 text-center text-red-500">{{ technicianMonthlyDisruptionError }}</div>
          <div v-else-if="technicianMonthlyDisruptionData" class="space-y-4">
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">请假导致未履约</div>
                <div class="mt-1 font-medium text-gray-800">{{ Number(technicianMonthlyDisruptionData.leave_disruption_count || 0) }} 次</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">商户免责次数</div>
                <div class="mt-1 font-medium text-gray-800">{{ Number(technicianMonthlyDisruptionData.merchant_exempt_count || 0) }} 次</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">客服责任次数</div>
                <div class="mt-1 font-medium text-gray-800">{{ Number(technicianMonthlyDisruptionData.technician_chargeable_count || 0) }} 次</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4">
                <div class="text-xs text-gray-400">首次免责</div>
                <div class="mt-1 font-medium" :class="technicianMonthlyDisruptionData.first_exempt_used ? 'text-orange-600' : 'text-green-600'">
                  {{ technicianMonthlyDisruptionData.first_exempt_used ? '已使用' : '未使用' }}
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-gray-200 bg-white px-4 py-4 text-sm text-gray-700">
              <div class="font-medium text-gray-800">统计说明</div>
              <div class="mt-2 space-y-1 text-gray-600">
                <div>月份：{{ technicianMonthlyDisruptionData.month || technicianMonthlyDisruptionMonth || '-' }}</div>
                <div>账本明细数：{{ technicianMonthlyDisruptionItems.length }} 条</div>
              </div>
            </div>

            <div class="space-y-2">
              <div class="font-medium text-gray-800">账本明细</div>
              <div v-if="technicianMonthlyDisruptionItems.length === 0" class="rounded-xl border border-dashed border-gray-200 px-4 py-8 text-center text-sm text-gray-400">
                本月暂无异常统计明细
              </div>
              <div v-for="(item, index) in technicianMonthlyDisruptionItems" :key="item.id || `${item.appointment_id || 'appt'}-${index}`" class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-700">
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <div class="font-medium text-gray-800">{{ getReasonText(item.latest_reason || item.reason || item.disruption_reason) || '异常账本' }}</div>
                    <div v-if="item.appointment_time || item.reserved_start_at" class="mt-1 text-gray-500">预约时间：{{ formatDateTime(item.reserved_start_at || item.appointment_time) }}</div>
                    <div v-if="item.created_at" class="mt-1 text-gray-500">记录时间：{{ formatDateTime(item.created_at) }}</div>
                    <div v-if="item.liability_level" class="mt-1 text-gray-500">责任归属：{{ getLiabilityText(item.liability_level) }}</div>
                    <div v-if="item.note || item.remark" class="mt-1 text-gray-500">备注：{{ item.note || item.remark }}</div>
                  </div>
                  <div class="text-right shrink-0">
                    <div v-if="item.exempt_applied !== undefined" class="text-xs" :class="item.exempt_applied ? 'text-green-600' : 'text-gray-400'">
                      {{ item.exempt_applied ? '已免责' : '未免责' }}
                    </div>
                    <div v-if="item.delay_minutes" class="mt-1 text-xs text-gray-500">延迟 {{ item.delay_minutes }} 分钟</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 营业状态切换弹窗 -->
    <div v-if="showBusinessStatusModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="showBusinessStatusModal = false">
      <div class="bg-white rounded-2xl w-11/12 max-w-sm overflow-hidden">
        <!-- 弹窗头部 -->
        <div class="px-5 py-4 border-b">
          <h3 class="font-medium text-lg text-gray-800">切换营业状态</h3>
        </div>

        <!-- 弹窗内容 -->
        <div class="px-5 py-6">
          <p class="text-gray-600 mb-6">
            {{ merchant.is_open ? '确定要切换为打烊状态吗？' : '确定要切换为营业中状态吗？' }}
          </p>
          <div class="flex gap-3">
            <button
              @click="showBusinessStatusModal = false"
              class="flex-1 py-2.5 border border-gray-300 text-gray-700 rounded-lg font-medium hover:bg-gray-50 transition-colors"
            >
              取消
            </button>
            <button
              @click="confirmToggleBusinessStatus"
              :class="[
                'flex-1 py-2.5 rounded-lg font-medium transition-colors',
                merchant.is_open
                  ? 'bg-red-500 text-white hover:bg-red-600'
                  : 'bg-green-500 text-white hover:bg-green-600'
              ]"
            >
              {{ merchant.is_open ? '确认打烊' : '确认营业' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 售卡二维码弹窗 -->
    <div v-if="showSellQrModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 select-none" @click.self="closeSellQrModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg overflow-hidden">
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between">
          <h3 class="font-medium text-lg">{{ getTechnicianName() }}的售卡二维码</h3>
          <button @click="closeSellQrModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5">
          <div class="text-center">
            <div class="text-gray-800 font-medium">{{ sellSelectedTemplateName }}</div>
            <div class="text-gray-500 text-sm mt-1">请客户扫码购买</div>
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
              <canvas ref="sellQrCanvas" class="w-56 h-56" style="-webkit-touch-callout: none;"></canvas>
            </div>
          </div>

          <div class="mt-4 text-center text-gray-400 text-xs">
            请向客户出示此二维码用于购买卡片
          </div>
        </div>
      </div>
    </div>

    <!-- 扫码错误弹窗 -->
    <div v-if="showErrorModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 m-4 max-w-sm w-full">
        <div class="flex items-center mb-4">
          <div class="w-12 h-12 bg-red-100 rounded-full flex items-center justify-center mr-3">
            <svg class="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </div>
          <h3 class="text-lg font-medium text-gray-900">扫码失败</h3>
        </div>
        <p class="text-gray-600 mb-6">{{ errorMessage }}</p>
        <div class="flex gap-3">
          <button
            @click="closeErrorModal"
            class="flex-1 py-2 bg-primary text-white rounded-lg font-medium"
          >
            确定
          </button>
        </div>
        <p class="text-center text-gray-400 text-xs mt-3">
          10秒后自动关闭
        </p>
      </div>
    </div>

    <div v-if="showServiceDurationConfirmModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 select-none" @click.self="cancelServiceDurationConfirm">
      <div class="bg-white rounded-xl p-6 m-4 max-w-sm w-full">
        <div class="flex items-center mb-4">
          <div class="w-12 h-12 bg-orange-100 rounded-full flex items-center justify-center mr-3">
            <svg class="w-6 h-6 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M12 19a7 7 0 110-14 7 7 0 010 14z"/>
            </svg>
          </div>
          <h3 class="text-lg font-medium text-gray-900">完成当前服务</h3>
        </div>
        <p class="text-gray-600 whitespace-pre-line mb-6">{{ serviceDurationConfirmMessage }}</p>
        <div class="flex gap-3">
          <button
            @click="cancelServiceDurationConfirm"
            class="flex-1 py-2.5 bg-gray-100 text-gray-700 rounded-lg font-medium"
          >
            取消
          </button>
          <button
            @click="confirmServiceDurationConfirm"
            class="flex-1 py-2.5 bg-primary text-white rounded-lg font-medium"
          >
            确认
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, onActivated, watch, nextTick, computed } from 'vue'
import { useRouter, useRoute, onBeforeRouteLeave } from 'vue-router'
import { ensureMerchantPermissionsLoaded, merchantApi, appointmentApi, shopApi, attendanceApi, serviceSessionApi, usageApi, noticeApi, cardApi, queueApi, isHandledAuthRedirectError } from '../../api'
import { clearMerchantAuth, clearMerchantPermissionKeys, hasMerchantPermission, getMerchantActiveAuth, getMerchantId, getTechnicianShopSlug } from '../../utils/auth'
import {
  getAutoFinishLabel,
  getPendingStartLabel,
  getScanStartLabel,
  getServicePendingFinishLabel,
  getServicePendingStartLabel,
  getStartCountdownLabel,
  isQueueModeMerchant,
  replaceTerms
} from '../../utils/terms'
import { normalizeSessionStatus } from '../../utils/sessionStatus'
import { formatDateTime, formatDate } from '../../utils/dateFormat'
import { DATA_POLL_INTERVAL_MS, REFRESH_GUARD_INTERVAL_MS } from '../../constants/polling'
import Table from './Table.vue'
import QRCode from 'qrcode'

const router = useRouter()
const route = useRoute()
let topScanLongPressTimer = null
let topScanStart = null
const suppressTopScanClickUntil = ref(0)
const prevTopScanBodyStyle = {
  userSelect: '',
  webkitUserSelect: '',
  webkitTouchCallout: ''
}
const merchantId = ref(null)
const merchant = ref({})
const isQueueModeView = computed(() => isQueueModeMerchant(merchant.value))
const schedulerHealth = ref(null)
const schedulerHealthError = ref('')
const schedulerHealthLoaded = ref(false)
let schedulerHealthTimer = null

const getMerchantPendingStartLabel = (options = {}) => getPendingStartLabel(merchant.value, options)
const getMerchantAutoFinishLabel = () => getAutoFinishLabel(merchant.value)
const getMerchantServicePendingStartLabel = (options = {}) => getServicePendingStartLabel(merchant.value, options)
const getMerchantServicePendingFinishLabel = () => getServicePendingFinishLabel(merchant.value)
const getMerchantScanStartLabel = () => getScanStartLabel(merchant.value)
const getMerchantTodayStartRecordLabel = () => (isQueueModeView.value ? '今日上号记录' : replaceTerms('今日起单记录', merchant.value))
const getMerchantTodayNoStartRecordLabel = () => (isQueueModeView.value ? '今日暂无上号' : replaceTerms('今日暂无起单', merchant.value))
const supportsPendingStartReassignBeforeLeave = () => {
  if (merchant.value?.support_customer_service_mode) return true
  return !!merchant.value?.support_queue && !!merchant.value?.support_multi_customer_service
}
const getPendingStartLeavePromptLabel = () => {
  return getMerchantServicePendingStartLabel({ queueMode: isQueueModeMerchant(merchant.value) })
}

const getLocalDateKey = (date = new Date()) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const canBusinessStatusUpdate = computed(() => hasMerchantPermission('merchant.business_status.manage'))
const canDirectSaleManage = computed(() => hasMerchantPermission('merchant.direct_sale.manage'))
const canCardIssue = computed(() => hasMerchantPermission('merchant.card.issue'))
const canFinishVerify = computed(() => hasMerchantPermission('merchant.card.finish'))
const canVerify = computed(() => hasMerchantPermission('merchant.card.verify'))
const canNoticeManage = computed(() => hasMerchantPermission('merchant.notice.manage'))
const canAppointmentView = computed(() => hasMerchantPermission('merchant.appointment.view'))
const canAppointmentManage = computed(() => hasMerchantPermission('merchant.appointment.manage'))
const canRoomManage = computed(() => hasMerchantPermission('merchant.service.manage'))
const canQueueCalling = computed(() => hasMerchantPermission('merchant.queue.calling'))
const canTableView = computed(() => hasMerchantPermission('merchant.table.view'))
const showMerchantSchedulePublishingPanel = computed(() => false)
const showTechnicianSchedulePublishingPanel = computed(() => {
  if (!isTechnicianAuth() || currentTab.value !== 'appointment' || !showAppointmentTab.value) {
    return false
  }
  if (schedulePublishingLoading.value || schedulePublishingError.value) {
    return true
  }
  if (schedulePublishings.value.length === 0) {
    return true
  }
  return visibleSchedulePublishings.value.length > 0
})

// 统计卡片显示个数
const visibleStatsCount = computed(() => {
  let count = 0
  if (canDirectSaleManage.value && merchant.value.support_direct_sale && pendingDirectPurchases.value > 0) count++
  if (showAppointmentSummaryCard.value) count++
  if (showExceptionSummaryCard.value) count++
  if (canVerify.value && todayVerifyCount.value > 0) count++
  return count
})

const visibleStatsGridClass = computed(() => {
  if (visibleStatsCount.value >= 4) return 'grid-cols-2'
  if (visibleStatsCount.value === 3) return 'grid-cols-3'
  if (visibleStatsCount.value === 2) return 'grid-cols-2'
  return 'grid-cols-1'
})

const schedulerHealthItems = computed(() => Array.isArray(schedulerHealth.value?.items) ? schedulerHealth.value.items : [])

const schedulerHealthBadgeText = computed(() => {
  const status = String(schedulerHealth.value?.overall_status || '').trim()
  if (!status) return '未读取'
  if (status === 'healthy') return '正常'
  if (status === 'degraded') return '异常'
  return '未知'
})

const schedulerHealthBadgeClass = computed(() => {
  const status = String(schedulerHealth.value?.overall_status || '').trim()
  if (status === 'healthy') return 'bg-green-50 text-green-700'
  if (status === 'degraded') return 'bg-red-50 text-red-600'
  return 'bg-gray-100 text-gray-500'
})

const schedulerHealthSummary = computed(() => {
  const generatedAt = schedulerHealth.value?.generated_at ? formatDateTime(schedulerHealth.value.generated_at) : '-'
  const threshold = Number(schedulerHealth.value?.stale_threshold_minutes || 0)
  const message = String(schedulerHealth.value?.overall_message || '').trim() || '未读取调度器状态'
  return `最近检查：${generatedAt} · 阈值：${threshold || '-'} 分钟 · ${message}`
})

// Tab 显示控制
const showVerifyTab = computed(() => {
  // 有核销权限
  console.log('showVerifyTab:', canVerify.value)
  return canVerify.value
})

const showStartTab = computed(() => {
  return false
})

const canAccessAppointmentTab = computed(() => {
  return !!merchant.value?.support_appointment && (canAppointmentView.value || canAppointmentManage.value)
})

const showAppointmentTab = computed(() => {
  return canAccessAppointmentTab.value && (todayAppointmentCount.value > 0 || showTechnicianSchedulePublishingTab.value)
})

const showAppointmentSummaryCard = computed(() => {
  return showAppointmentTab.value && appointmentSummaryCount.value > 0
})

const showExceptionTab = computed(() => {
  return canAccessAppointmentTab.value
})
const showExceptionStandaloneTab = computed(() => {
  return showExceptionTab.value && !showTableTab.value
})
const showExceptionContentInTable = computed(() => {
  return currentTab.value === 'table' && showTableTab.value && showExceptionTab.value
})
const showExceptionStandaloneContent = computed(() => {
  return currentTab.value === 'exception' && showExceptionStandaloneTab.value
})

const showExceptionSummaryCard = computed(() => {
  return showExceptionTab.value && exceptionSummaryCount.value > 0
})

const showFinishTab = computed(() => {
	return false
})

const showNoticeTab = computed(() => {
  // 有通知管理权限
  return canNoticeManage.value
})

const canCardSell = computed(() => hasMerchantPermission('merchant.card.sell'))

const showCardsTab = computed(() => {
  return canVerify.value || canCardSell.value
})

const showTableTab = computed(() => {
  return canTableView.value
})

const showServiceTab = computed(() => {
  // 按岗位独立配置签到：不再依赖商户全局开关
  return isTechnicianAuth()
})

const showTechnicianBoardTab = computed(() => {
  return isTechnicianAuth()
})
const currentTab = ref('appointment')
const routeUserCode = ref('')
const userCodeAnchor = ref(null)

const DASHBOARD_ACTIVE_TAB_STORAGE_KEY = 'merchant_dashboard_active_tab'

const selectTab = (tab) => {
  currentTab.value = tab
  try {
    localStorage.setItem(DASHBOARD_ACTIVE_TAB_STORAGE_KEY, String(tab))
  } catch (e) {
    // ignore
  }
}

const normalizeDashboardTab = (tab) => {
  const normalizedTab = tab === 'start' ? 'service' : tab
  if (normalizedTab === 'exception' && showTableTab.value) {
    return 'table'
  }
  return normalizedTab
}

const queuePendingList = ref([])
const queuePendingLoading = ref(false)
const queuePendingReassigningMap = ref({})

const queueTimeoutWaitingList = ref([])
const queueTimeoutWaitingLoading = ref(false)

const queueCallInfo = ref(null)
const serviceTabRefreshing = ref(false)
const serviceTabRefreshQueued = ref(false)

let queuePendingFirstLoaded = false
let serviceTabBoundaryState = new Map()
let startTabBoundaryState = new Map()

const queuePendingSignature = (list) => {
  if (!Array.isArray(list) || list.length === 0) return ''
  return list
    .map(it => {
      const usageId = Number(it?.usage_id || 0)
      const queueNo = Number(it?.queue_no || 0)
      const st = String(it?.session_status || '')
      const sc = it?.start_confirmed_at ? String(it.start_confirmed_at) : ''
      const started = it?.started_at ? String(it.started_at) : ''
      const scheduled = it?.scheduled_finish_at ? String(it.scheduled_finish_at) : ''
      const duration = Number(it?.duration_minutes || 0)
      const tech = it?.technician_id != null ? String(it.technician_id) : ''
      const techReason = String(it?.technician_unavailable_reason || '')
      return `${usageId}:${queueNo}:${st}:${sc}:${started}:${scheduled}:${duration}:${tech}:${techReason}`
    })
    .join('|')
}

const getQueueWaitingItemPhaseText = (it) => {
  if (!it) return '-'
  const st = normalizeSessionStatus(it.session_status)
  if (st === 'start_pending' && !it.start_confirmed_at) return getMerchantPendingStartLabel({ queueMode: true })
  if (st === 'staff_selecting') return '待分配'
  if (st === 'timeout_waiting') return '超时过号等待'
  return st || '-'
}

const shouldShowQueuePendingReassign = (it) => {
  if (!canQueueCalling.value) return false
  if (!it || !it.session_id) return false
  const st = normalizeSessionStatus(it.session_status)
  return st === 'start_pending' && !it.start_confirmed_at && !!String(it.technician_unavailable_reason || '').trim()
}

const getUsagePendingReassignPayload = (usage) => {
  if (!usage) return null
  return {
    session_id: usage.service_session_id,
    session_status: usage.service_session_status,
    start_confirmed_at: usage.service_session_start_confirmed_at,
    technician_unavailable_reason: usage.service_technician_unavailable_reason
  }
}

const shouldShowUsagePendingReassign = (usage) => {
  return shouldShowQueuePendingReassign(getUsagePendingReassignPayload(usage))
}

const isUsagePendingReassigning = (usage) => {
  return isQueuePendingReassigning(getUsagePendingReassignPayload(usage))
}

const isQueuePendingReassigning = (it) => {
  const sid = Number(it?.session_id || 0)
  if (!sid) return false
  return !!queuePendingReassigningMap.value[sid]
}

const setQueuePendingReassigning = (sessionId, value) => {
  const sid = Number(sessionId || 0)
  if (!sid) return
  queuePendingReassigningMap.value = {
    ...queuePendingReassigningMap.value,
    [sid]: !!value
  }
}

const reassignQueuePendingItem = async (it, reason = '当前客服不可服务，运营发起重新分配') => {
  const sessionId = Number(it?.session_id || 0)
  if (!sessionId || isQueuePendingReassigning(it)) return false
  setQueuePendingReassigning(sessionId, true)
  try {
    const res = await queueApi.reassignCurrentPending(sessionId, reason)
    const data = res.data?.data || {}
    const toName = String(data.to_technician_name || '').trim()
    alert(toName ? `已重新分配给 ${toName}` : '已重新分配')
    await fetchServiceSessions()
    if (merchant.value?.support_queue) {
      await fetchQueuePendingList(true)
      await fetchQueueCallInfo()
      await fetchTodayUsages()
    }
    return true
  } catch (e) {
    alert(e.response?.data?.error || '重新分配失败')
    return false
  } finally {
    setQueuePendingReassigning(sessionId, false)
  }
}

const reassignUsagePendingItem = async (usage) => {
  return reassignQueuePendingItem(
    getUsagePendingReassignPayload(usage),
    '当前客服不可服务，商户首页核销记录发起重新分配'
  )
}

const getQueueServingItemPhaseText = (it) => {
  if (!it) return '-'
  const st = normalizeSessionStatus(it.session_status)
  if (st === 'delay_pending') return getMerchantPendingStartLabel({ queueMode: true })
  if (st === 'serving') return '服务中'
  if (st === 'auto_finishing') return getMerchantAutoFinishLabel()
  return st || '-'
}

const queueWaitingList = computed(() => {
  return (queuePendingList.value || []).filter(it => {
    const st = normalizeSessionStatus(it?.session_status)
    return st === 'start_pending' || st === 'staff_selecting'
  })
})

const queueServingList = computed(() => {
  return (queuePendingList.value || []).filter(it => {
    const st = normalizeSessionStatus(it?.session_status)
    return st === 'delay_pending' || st === 'serving' || st === 'auto_finishing'
  })
})

const getQueueServingItemRemainingSeconds = (it) => {
  if (!it) return null
  const st = normalizeSessionStatus(it.session_status)
  if (st !== 'serving' && st !== 'auto_finishing') return null

  let finishAt = 0
  const scheduledFinishAtRaw = it.scheduled_finish_at
  if (scheduledFinishAtRaw) {
    finishAt = new Date(scheduledFinishAtRaw).getTime()
  }
  if (!finishAt || Number.isNaN(finishAt)) {
    const startedAtRaw = it.started_at
    const durationMinutes = Number(it.duration_minutes || 0)
    if (!startedAtRaw || !Number.isFinite(durationMinutes) || durationMinutes <= 0) return null
    const startedAt = new Date(startedAtRaw).getTime()
    if (!startedAt || Number.isNaN(startedAt)) return null
    finishAt = startedAt + durationMinutes * 60 * 1000
  }

  const remain = Math.floor((finishAt - currentTime.value) / 1000)
  if (!Number.isFinite(remain)) return null
  return Math.max(0, remain)
}

const getQueueServingItemRemainingClass = (it) => {
  const remain = getQueueServingItemRemainingSeconds(it)
  return getRemainingSecondsClass(remain)
}

const getStartPendingRemainingSeconds = (sessionLike) => {
  if (!sessionLike) return null
  const status = normalizeSessionStatus(sessionLike.session_status || sessionLike.status)
  if (status !== 'start_pending') return null
  if (sessionLike.start_confirmed_at) return null

  const timeoutSeconds = Number(sessionLike.start_pending_timeout_seconds || 0) > 0
    ? Number(sessionLike.start_pending_timeout_seconds)
    : 180

  const baseRaw = sessionLike.updated_at || sessionLike.created_at
  if (baseRaw) {
    const baseTime = new Date(baseRaw).getTime()
    if (Number.isFinite(baseTime) && baseTime > 0) {
      const remain = Math.floor((baseTime + timeoutSeconds * 1000 - currentTime.value) / 1000)
      if (Number.isFinite(remain)) {
        return Math.max(0, remain)
      }
    }
  }

  const remainFromServer = Number(sessionLike.start_pending_remaining_seconds || 0)
  const fetchedAt = Number(sessionLike._countdown_fetched_at || 0)
  if (Number.isFinite(remainFromServer) && remainFromServer > 0 && Number.isFinite(fetchedAt) && fetchedAt > 0) {
    const elapsed = Math.floor((currentTime.value - fetchedAt) / 1000)
    return Math.max(0, remainFromServer - Math.max(0, elapsed))
  }

  if (Number.isFinite(remainFromServer) && remainFromServer > 0) {
    return Math.floor(remainFromServer)
  }

  return null
}

const getQueueWaitingItemStartPendingRemainingSeconds = (it) => {
  return getStartPendingRemainingSeconds(it)
}

const getQueueWaitingItemStartPendingRemainingClass = (it) => {
  const remain = getQueueWaitingItemStartPendingRemainingSeconds(it)
  if (remain === null) return 'text-blue-600'
  if (remain <= 60) return 'text-red-500'
  return 'text-blue-600'
}

const patchQueuePendingList = (nextList) => {
  const prev = queuePendingList.value
  const next = Array.isArray(nextList) ? nextList : []
  const fetchedAt = Date.now()

  const byUsageIdPrev = new Map()
  for (const it of prev) {
    const id = Number(it?.usage_id || 0)
    if (id > 0) byUsageIdPrev.set(id, it)
  }
  const nextIds = new Set()

  // 原地更新 & 新增
  for (const raw of next) {
    const id = Number(raw?.usage_id || 0)
    if (!id) continue
    nextIds.add(id)
    const existed = byUsageIdPrev.get(id)
    if (existed) {
      existed.queue_no = raw.queue_no
      existed.queue_called_at = raw.queue_called_at
      existed.session_id = raw.session_id
      existed.session_status = raw.session_status
      existed.start_confirmed_at = raw.start_confirmed_at
      existed.start_pending_timeout_seconds = raw.start_pending_timeout_seconds
      existed.start_pending_remaining_seconds = raw.start_pending_remaining_seconds
      existed.started_at = raw.started_at
      existed.scheduled_finish_at = raw.scheduled_finish_at
      existed.duration_minutes = raw.duration_minutes
      existed.technician_id = raw.technician_id
      existed.technician_name = raw.technician_name
      existed.technician_available = raw.technician_available
      existed.technician_unavailable_reason = raw.technician_unavailable_reason
      existed.project_name = raw.project_name
      existed.user_nickname = raw.user_nickname
      existed.updated_at = raw.updated_at
      existed.created_at = raw.created_at
      existed._countdown_fetched_at = fetchedAt
    } else {
      raw._countdown_fetched_at = fetchedAt
      prev.push(raw)
    }
  }

  // 删除已不存在的
  for (let i = prev.length - 1; i >= 0; i--) {
    const id = Number(prev[i]?.usage_id || 0)
    if (id && !nextIds.has(id)) {
      prev.splice(i, 1)
    }
  }
}

const patchTodayStartUsages = (nextList) => {
  const prev = todayStartUsages.value
  const next = Array.isArray(nextList) ? nextList : []
  const fetchedAt = Date.now()

  const byIdPrev = new Map()
  for (const item of prev) {
    const id = Number(item?.id || 0)
    if (id > 0) byIdPrev.set(id, item)
  }

  const ordered = []
  for (const raw of next) {
    const id = Number(raw?.id || 0)
    if (!id) continue

    const existed = byIdPrev.get(id)
    if (existed) {
      for (const key of Object.keys(existed)) {
        if (key.startsWith('_')) continue
        if (!(key in raw)) {
          delete existed[key]
        }
      }
      Object.assign(existed, raw)
      existed._countdown_fetched_at = fetchedAt
      ordered.push(existed)
    } else {
      raw._countdown_fetched_at = fetchedAt
      ordered.push(raw)
    }
  }

  prev.splice(0, prev.length, ...ordered)
}

let lastQueuePendingSig = ''
const fetchQueuePendingList = async (silent = false) => {
  if (!showQueueControlInService.value) return
  if (!silent && !queuePendingFirstLoaded) queuePendingLoading.value = true
  try {
    const res = await queueApi.getPendingList({})
    const next = res.data?.data || []
    const sig = queuePendingSignature(next)
    if (sig !== lastQueuePendingSig) {
      patchQueuePendingList(next)
      lastQueuePendingSig = sig
    }
    queuePendingFirstLoaded = true
  } catch (e) {
    // 静默轮询失败不清空列表，避免 UI 闪烁；首次加载失败则按空处理
    if (!queuePendingFirstLoaded) {
      queuePendingList.value = []
      queuePendingFirstLoaded = true
    }
  } finally {
    if (!silent && queuePendingLoading.value) queuePendingLoading.value = false
  }
}

const fetchQueueCallInfo = async () => {
  if (!showQueueControlInService.value) return
  try {
    const res = await queueApi.getCallInfo()
    const next = res.data?.data || null
    if (next?.session) {
      next.session._countdown_fetched_at = Date.now()
    }
    queueCallInfo.value = next
  } catch (e) {
    // 静默失败不置空，避免 UI 抖动
  }
}

let queueTimeoutWaitingFirstLoaded = false
const fetchQueueTimeoutWaitingList = async (silent = false) => {
  if (!showQueueControlInService.value) return
  if (!silent && !queueTimeoutWaitingFirstLoaded) queueTimeoutWaitingLoading.value = true
  try {
    const res = await queueApi.getTimeoutWaitingList({})
    const next = Array.isArray(res.data?.data) ? res.data.data : []
    queueTimeoutWaitingList.value = next
    queueTimeoutWaitingFirstLoaded = true
  } catch (e) {
    if (!queueTimeoutWaitingFirstLoaded) {
      queueTimeoutWaitingList.value = []
      queueTimeoutWaitingFirstLoaded = true
    }
  } finally {
    if (!silent && queueTimeoutWaitingLoading.value) queueTimeoutWaitingLoading.value = false
  }
}

const formatTimeoutWaitingTitle = (it) => {
  if (!it) return ''
  const wno = String(it.window_no || '').trim()
  const tname = String(it.technician_name || '').trim()
  const parts = []
  if (wno) parts.push(`${windowTerm.value}${wno}`)
  if (tname) parts.push(tname)
  return parts.join(' ')
}

const getDefaultTab = () => {
  if (showAppointmentTab.value) {
    return 'appointment'
  }
  if (showExceptionStandaloneTab.value) {
    return 'exception'
  }
  if (showVerifyTab.value) {
    return 'verify'
  } else if (showFinishTab.value) {
    return 'finish'
  } else if (showNoticeTab.value) {
    return 'notice'
  } else if (showCardsTab.value) {
    return 'cards'
  } else if (showTableTab.value) {
    return 'table'
  } else if (showServiceTab.value) {
    return 'service'
  } else if (showTechnicianBoardTab.value) {
    return 'board'
  } else {
    return 'appointment'
  }
}

const getFirstVisibleTab = () => {
  if (showAppointmentTab.value) return 'appointment'
  if (showExceptionStandaloneTab.value) return 'exception'
  if (showVerifyTab.value) return 'verify'
  if (showFinishTab.value) return 'finish'
  if (showNoticeTab.value) return 'notice'
  if (showCardsTab.value) return 'cards'
  if (showTableTab.value) return 'table'
  if (showServiceTab.value) return 'service'
  if (showTechnicianBoardTab.value) return 'board'
  return 'appointment'
}

const showSellView = ref(false)
const showSellQrView = ref(false)
const showSellQrModal = ref(false)
const sellSelectedTemplate = ref(null)
const sellQrDataUrl = ref('')
const sellQrCanvas = ref(null)

const activeSellTemplates = computed(() => {
  return (cardTemplates.value || []).filter(t => t && t.is_active)
})

const sellSelectedTemplateName = computed(() => {
  return sellSelectedTemplate.value && sellSelectedTemplate.value.name ? sellSelectedTemplate.value.name : ''
})

const currentAccountName = computed(() => {
  // 如果是工作人员登录，显示工作人员账号和称谓
  if (isTechnicianAuth()) {
    const technicianAccount = sessionStorage.getItem('technicianAccount')
    const technicianName = sessionStorage.getItem('technicianName')
    const technicianRoleName = sessionStorage.getItem('technicianRoleName')
    
    if (technicianAccount && technicianName) {
      return `${technicianRoleName}: ${technicianAccount} ${technicianName}`
    }
    if (technicianAccount) return `${technicianRoleName}: ${technicianAccount}`
    const technicianCode = sessionStorage.getItem('technicianCode')
    if (technicianCode && technicianName) {
      return `${technicianRoleName}: ${technicianCode} ${technicianName}`
    }
    if (technicianCode) return `${technicianRoleName}: ${technicianCode}`
    return technicianRoleName
  }
  // 如果是商户登录，显示商户手机号
  const merchantPhone = localStorage.getItem('merchantPhone')
  return merchantPhone || '商户'
})

const isTechnicianAuth = () => {
  return sessionStorage.getItem('merchantActiveAuth') === 'staff'
}

const getTechnicianId = () => {
  const raw = sessionStorage.getItem('technicianId')
  if (!raw) return null
  const n = parseInt(String(raw), 10)
  return Number.isFinite(n) && n > 0 ? n : null
}

const getTechnicianName = () => {
  const name = sessionStorage.getItem('technicianName')
  const code = sessionStorage.getItem('technicianCode')
  if (name) return name
  if (code) return `技师${code}`
  return '技师'
}

// 窗口/台号名词（商户可自定义）
const windowTerm = computed(() => {
  return merchant.value?.queue_window_term || '窗口'
})

// 当前技师信息（用于读取 window_no 等）
const technicianMe = ref(null)
const technicianWindowNo = computed(() => {
  const no = technicianMe.value?.window_no
  return String(no || '').trim() || ''
})

const showTechnicianAttendancePanel = computed(() => {
  // 仅技师账号需要展示；并且岗位开启签到才展示
  if (!isTechnicianAuth()) return false
  const requireAttendance = technicianMe.value?.service_role?.require_attendance
  if (typeof requireAttendance === 'boolean') return requireAttendance
  return true
})

const queueBlockedByAttendance = computed(() => {
  if (!merchant.value?.support_queue) return false
  if (!isTechnicianAuth()) return false
  const requireAttendance = technicianMe.value?.service_role?.require_attendance
  const roleRequiresAttendance = typeof requireAttendance === 'boolean' ? requireAttendance : true
  if (!roleRequiresAttendance) return false
  return isTechnicianNotCheckedIn.value
})

const attendanceBlockedHint = computed(() => {
  if (isQueueModeView.value && queueBlockedByAttendance.value) {
    return '上班签到后 才可进行叫号和扫码上号'
  }
  return `上班签到后 才可${getMerchantScanStartLabel()}`
})

const queueCallQueueNoText = computed(() => {
  if (!merchant.value?.support_queue) return '-'
  const info = queueCallInfo.value
  if (!info) return '-'
  const no = Number(info.queue_no || 0)
  if (!Number.isFinite(no) || no <= 0) return '-'
  const prefix = String(info.queue_prefix || '')
  return `${prefix}${no}`
})

const canSellCards = computed(() => {
  return (
    canCardSell.value &&
    !!merchant.value.support_direct_sale &&
    isTechnicianAuth() &&
    !!getTechnicianId()
  )
})
const scanUserCodeActive = ref(false)

const todayVerifyCount = ref(0)
const pendingAppointments = ref(0)
const pendingDirectPurchases = ref(0)
const appointments = ref([])
const appointmentTechnicianDirectory = ref([])
const showAppointmentRescheduleModal = ref(false)
const rescheduleAppointmentTarget = ref(null)
const appointmentRescheduleEligibility = ref(null)
const appointmentRescheduleLoading = ref(false)
const appointmentRescheduleSubmitting = ref(false)
const appointmentRescheduleSlots = ref([])
const appointmentRescheduleTechnicians = ref([])
const selectedAppointmentRescheduleTime = ref('')
const appointmentRescheduleDateMin = computed(() => {
  const dates = appointmentRescheduleEligibility.value?.allowed_dates || []
  return dates[0] || ''
})
const appointmentRescheduleDateMax = computed(() => {
  const dates = appointmentRescheduleEligibility.value?.allowed_dates || []
  return dates.length > 0 ? dates[dates.length - 1] : ''
})
const appointmentRescheduleDateHint = computed(() => {
  const eligibility = appointmentRescheduleEligibility.value
  if (!eligibility?.allowed) return ''
  if (eligibility.rule_mode === 'same_day_only' && appointmentRescheduleSlots.value.length === 0 && !appointmentRescheduleLoading.value) {
    return '原预约日内暂无可改签时段'
  }
  if (eligibility.rule_mode === 'same_day_only') {
    return '当前规则仅允许改签到原预约日内的其他营业时段'
  }
  return ''
})
const appointmentRescheduleComparisonHint = computed(() => {
  const eligibility = appointmentRescheduleEligibility.value
  if (!eligibility?.allowed) return ''
  if (eligibility.rule_mode === 'same_day_only') {
    return '当前仅允许处理原预约日内的改签，系统会优先标注排布更优的时段。'
  }
  return ''
})
const selectedAppointmentRescheduleCandidates = computed(() => {
  const slot = (appointmentRescheduleSlots.value || []).find(item => item.time === selectedAppointmentRescheduleTime.value)
  const candidates = slot?.technician_candidates || []
  const byId = new Map((appointmentRescheduleTechnicians.value || []).map(item => [Number(item.id || 0), item]))
  const currentTechnicianId = Number(rescheduleAppointmentTarget.value?.technician_id || 0)
  return candidates.map(item => ({
    technician_id: Number(item.technician_id || 0),
    label: (() => {
      const technician = byId.get(Number(item.technician_id || 0))
      const name = String(technician?.name || '').trim() || `客服${item.technician_id}`
      return name
    })()
  })).filter(item => !currentTechnicianId || item.technician_id !== currentTechnicianId)
})
const selectedAppointmentRescheduleTechnicianText = computed(() => {
  const technicianId = Number(appointmentRescheduleForm.value.technician_id || 0)
  if (!technicianId) return ''
  const technician = (appointmentRescheduleTechnicians.value || []).find(item => Number(item.id || 0) === technicianId)
  if (!technician) return ''
  const name = String(technician.name || '').trim() || `客服${technicianId}`
  const account = String(technician.account || '').trim()
  return account ? `${name} - ${account}` : name
})
const appointmentRescheduleForm = ref({
  date: '',
  technician_id: null,
  reason: ''
})
const showAppointmentCompensationModal = ref(false)
const compensationAppointmentTarget = ref(null)
const appointmentCompensationSubmitting = ref(false)
const appointmentCompensationForm = ref({
  type: 'extra_times',
  value: 1,
  reason: '',
  remark: ''
})
const showAppointmentDetailModal = ref(false)
const appointmentDetailTarget = ref(null)
const appointmentDetailLoading = ref(false)
const appointmentDetailError = ref('')
const appointmentDetailSettlement = ref(null)
const appointmentDetailDelayLedgers = ref([])
const appointmentDetailSummary = ref(null)
const appointmentDetailHasRepairOverview = ref(false)
const appointmentDetailCompensations = computed(() => appointmentDetailSummary.value?.compensations || [])
const showTechnicianMonthlyDisruptionModal = ref(false)
const technicianMonthlyDisruptionTechnicianId = ref(null)
const technicianMonthlyDisruptionTechnicianName = ref('')
const technicianMonthlyDisruptionMonth = ref('')
const technicianMonthlyDisruptionLoading = ref(false)
const technicianMonthlyDisruptionError = ref('')
const technicianMonthlyDisruptionData = ref(null)
const schedulePublishingDate = ref('')
const schedulePublishingLoading = ref(false)
const schedulePublishingSubmitting = ref(false)
const schedulePublishingError = ref('')
const schedulePublishings = ref([])
const showAppointmentRepairOverviewModal = ref(false)
const appointmentRepairOverviewLoading = ref(false)
const appointmentRepairOverviewError = ref('')
const appointmentRepairOverviewReason = ref('')
const appointmentRepairOverviewSchedule = ref(null)
const appointmentRepairAffectedAppointments = ref([])
const appointmentRepairItems = ref([])
const appointmentRepairProtectedSlots = ref([])
const schedulePublishingActionSubmitting = ref(false)
const schedulePublishingNow = computed(() => new Date(currentTime.value || Date.now()))
const formatScheduleDateValue = (date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
const parseScheduleDateValue = (value) => {
  const match = String(value || '').trim().match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!match) return null
  const date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]), 0, 0, 0, 0)
  if (Number.isNaN(date.getTime())) return null
  return date
}
const formatScheduleDayLabel = (date) => {
  if (!date || Number.isNaN(date.getTime())) return ''
  return `${date.getDate()}日`
}
const formatScheduleBookingOpenLabel = (row) => {
  const source = row?.start_at || row?.publish_date
  if (!source) return ''
  const date = new Date(source)
  if (Number.isNaN(date.getTime())) return ''
  const openDate = new Date(date)
  openDate.setDate(openDate.getDate() - 1)
  return `${formatScheduleDayLabel(openDate)} 10:00 开放预约`
}
const getSchedulePublishCutoff = (date) => {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 10, 0, 0, 0)
}
const getScheduleWithdrawCutoff = (date) => {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 10, 30, 0, 0)
}
const getDefaultSchedulePublishingDate = () => {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0, 0)
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)
  return formatScheduleDateValue(tomorrow)
}
const currentSchedulePublishingDate = computed(() => parseScheduleDateValue(schedulePublishingDate.value))
const getSchedulePublishingWindowStateForDate = (target, now = schedulePublishingNow.value) => {
  if (!target || !isTechnicianAuth()) {
    return {
      canPublish: true,
      canWithdraw: false,
      publishBlockedReason: '',
      withdrawBlockedReason: ''
    }
  }
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0, 0)
  const targetDay = new Date(target.getFullYear(), target.getMonth(), target.getDate(), 0, 0, 0, 0)
  const publishCutoff = getSchedulePublishCutoff(targetDay)
  const withdrawCutoff = getScheduleWithdrawCutoff(targetDay)
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)

  if (targetDay.getTime() === today.getTime()) {
    return {
      canPublish: now < publishCutoff,
      canWithdraw: now < withdrawCutoff,
      publishBlockedReason: now < publishCutoff ? '' : '今日预约排班已过 10:00 发布时间',
      withdrawBlockedReason: now < withdrawCutoff ? '' : '今日预约排班已过 10:30 撤销截止时间'
    }
  }

  if (targetDay.getTime() === tomorrow.getTime()) {
    const publishCutoff = getSchedulePublishCutoff(today)
    const withdrawCutoff = getScheduleWithdrawCutoff(today)
    return {
      canPublish: now < publishCutoff,
      canWithdraw: now < withdrawCutoff,
      publishBlockedReason: now < publishCutoff ? '' : '次日预约排班已过今日 10:00 发布时间',
      withdrawBlockedReason: now < withdrawCutoff ? '' : '次日预约排班已过今日 10:30 撤销截止时间'
    }
  }

  return {
    canPublish: false,
    canWithdraw: false,
    publishBlockedReason: '仅支持发布次日预约排班',
    withdrawBlockedReason: '仅支持撤销次日预约排班'
  }
}
const technicianSchedulePublishingWindowState = computed(() => {
  return getSchedulePublishingWindowStateForDate(currentSchedulePublishingDate.value, schedulePublishingNow.value)
})
const showTechnicianSchedulePublishingTab = computed(() => {
  if (!isTechnicianAuth() || !canAccessAppointmentTab.value) return false
  if (schedulePublishingLoading.value || schedulePublishingError.value) return true
  const rows = visibleSchedulePublishings.value || []
  const hasPublishableRows = rows.some(row => ['unpublished', 'canceled'].includes(getEffectiveSchedulePublishingStatus(row)))
  const hasPublishedRows = rows.some(row => getEffectiveSchedulePublishingStatus(row) === 'published')
  const hasRepublishableCanceledRows = rows.some(row => getEffectiveSchedulePublishingStatus(row) === 'canceled')
  return (technicianSchedulePublishingWindowState.value.canPublish && hasPublishableRows) ||
    (technicianSchedulePublishingWindowState.value.canWithdraw && (hasPublishedRows || hasRepublishableCanceledRows))
})
const technicianSchedulePublishingSubtitle = computed(() => {
  if (hasPublishedScheduleRows.value) {
    const target = currentSchedulePublishingDate.value
    if (target) {
      const withdrawDate = new Date(target)
      withdrawDate.setDate(withdrawDate.getDate() - 1)
      return `${formatScheduleDayLabel(target)}的预约排班已发布，${formatScheduleDayLabel(withdrawDate)}10:30 前可撤销`
    }
  }
  return '请在今天 10:00 前发布明天的预约安排，10:30 前可撤销。'
})
const technicianSchedulePublishingHint = computed(() => {
  if (!isTechnicianAuth()) return ''
  if (hasPublishedScheduleRows.value) {
    return technicianSchedulePublishingWindowState.value.withdrawBlockedReason || '当前排班已发布，可在截止前撤销。'
  }
  if (!hasPublishableScheduleRows.value) {
    return '当前没有待发布的预约排班'
  }
  return technicianSchedulePublishingWindowState.value.publishBlockedReason
})
const appointmentRepairOverviewScheduleLabel = computed(() => {
  const schedule = appointmentRepairOverviewSchedule.value
  if (!schedule?.publish_date && !schedule?.start_at) return ''
  const parts = []
  if (schedule.publish_date) parts.push(formatDate(schedule.publish_date))
  if (schedule.start_at) parts.push(formatDateTime(schedule.start_at))
  if (schedule.end_at) parts.push(`至 ${formatDateTime(schedule.end_at)}`)
  return parts.join(' ')
})
const technicianMonthlyDisruptionItems = computed(() => {
  const data = technicianMonthlyDisruptionData.value
  if (!data) return []
  if (Array.isArray(data.items)) return data.items
  if (Array.isArray(data.records)) return data.records
  if (Array.isArray(data.ledgers)) return data.ledgers
  return []
})
const unassignedAppointments = computed(() => {
  if (!isTechnicianAuth()) return []
  return (appointments.value || []).filter(a => isTechnicianVisibleUnassignedAppointment(a))
})
const assignedAppointments = computed(() => {
  if (!isTechnicianAuth()) return appointments.value || []
  const currentTechnicianId = getTechnicianId()
  if (!currentTechnicianId) return []
  return (appointments.value || []).filter(a => Number(a?.technician_id) === Number(currentTechnicianId))
})
const buildExceptionGroups = (items, scopeKey = '', scopeTitle = '') => {
  const source = Array.isArray(items) ? items : []
  const groups = []
  const consumed = new Set()

  const pushGroup = (key, title, predicate) => {
    const bucket = []
    for (const appt of source) {
      const id = Number(appt?.id || 0)
      if (id > 0 && consumed.has(id)) continue
      if (!predicate(appt)) continue
      bucket.push(appt)
      if (id > 0) consumed.add(id)
    }
    if (bucket.length === 0) return
    groups.push({
      key: scopeKey ? `${scopeKey}-${key}` : key,
      title: scopeTitle ? `${scopeTitle} / ${title}` : title,
      items: bucket
    })
  }

  pushGroup('data-cleanup', '脏数据收口', appt => isAppointmentDataCleanup(appt))
  pushGroup('timeout', '超时待处理', appt => isHistoricalArrivedAppointment(appt))
  pushGroup('reschedule', '改签/改派异常', appt => hasAppointmentRescheduleIssue(appt))
  pushGroup('dispute', '履约争议', appt => hasAppointmentDisputeIssue(appt))
  pushGroup('compensation', '补偿处理', appt => needsAppointmentCompensationFollowUp(appt))
  pushGroup('settled', '已结案', appt => isAppointmentSettled(appt))
  pushGroup('other', '其他异常', () => true)

  return groups
}

const appointmentFlowGroups = computed(() => {
  const list = appointments.value || []
  if (!isTechnicianAuth()) {
    const normalAppointments = list.filter(a => !isExceptionAppointment(a))
    const active = normalAppointments.filter(a => !isAppointmentSettled(a))
    const settled = normalAppointments.filter(a => isAppointmentSettled(a))
    const groups = []
    if (active.length > 0) groups.push({ key: 'active', title: '待服务', items: active })
    if (settled.length > 0) groups.push({ key: 'settled', title: '已结束', items: settled })
    return groups
  }

  const unassigned = unassignedAppointments.value.filter(a => !isExceptionAppointment(a))
  const mine = assignedAppointments.value.filter(a => !isExceptionAppointment(a))
  const myActive = mine.filter(a => !isAppointmentSettled(a))
  const mySettled = mine.filter(a => isAppointmentSettled(a))
  const groups = []
  if (unassigned.length > 0) {
    groups.push({ key: 'unassigned', title: '待分配', items: unassigned })
  }
  if (myActive.length > 0) {
    groups.push({
      key: 'assigned',
      title: unassigned.length > 0 ? '我的预约' : '',
      items: myActive
    })
  }
  if (mySettled.length > 0) {
    groups.push({ key: 'assigned-settled', title: '已结束', items: mySettled })
  }
  return groups
})

const formatAppointmentDateKey = (date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const formatAppointmentGroupDate = (dateText) => {
  const [year, month, day] = String(dateText || '').split('-').map(Number)
  if (!year || !month || !day) return '未知日期预约'
  return `${month}月${day}日预约`
}

const getAppointmentGroupDateKey = (appt) => {
  const appointmentTimeMs = getAppointmentTimeMs(appt)
  if (appointmentTimeMs !== null) {
    return formatAppointmentDateKey(new Date(appointmentTimeMs))
  }
  if (appt?.created_at) {
    const createdAt = new Date(appt.created_at)
    if (!Number.isNaN(createdAt.getTime())) {
      return formatAppointmentDateKey(createdAt)
    }
  }
  return '未知日期'
}

const appointmentDateGroups = computed(() => {
  const allItems = appointmentFlowGroups.value.flatMap(group => Array.isArray(group?.items) ? group.items : [])
  const groups = []
  const groupMap = new Map()

  for (const appt of allItems) {
    const dateKey = getAppointmentGroupDateKey(appt)
    if (!groupMap.has(dateKey)) {
      const group = {
        key: `date-${dateKey}`,
        title: `${formatAppointmentGroupDate(dateKey)} (${0})`,
        date: dateKey,
        items: []
      }
      groupMap.set(dateKey, group)
      groups.push(group)
    }
    groupMap.get(dateKey).items.push(appt)
  }

  groups.sort((left, right) => String(right.date || '').localeCompare(String(left.date || '')))
  groups.forEach((group) => {
    group.items.sort((left, right) => {
      const leftMs = getAppointmentTimeMs(left) ?? 0
      const rightMs = getAppointmentTimeMs(right) ?? 0
      return rightMs - leftMs
    })
    group.title = `${formatAppointmentGroupDate(group.date)} (${group.items.length})`
  })

  return groups
})

const isTodayAppointment = (appt) => {
  const appointmentTimeMs = getAppointmentTimeMs(appt)
  if (appointmentTimeMs === null) return false
  const todayKey = formatAppointmentDateKey(new Date(currentTime.value || Date.now()))
  return getAppointmentGroupDateKey(appt) === todayKey
}

const todayAppointmentCount = computed(() => {
  if (!canAccessAppointmentTab.value) return 0
  return appointmentFlowGroups.value
    .flatMap(group => Array.isArray(group?.items) ? group.items : [])
    .filter(appt => isTodayAppointment(appt))
    .length
})

const appointmentDisplayDateGroups = computed(() => {
  if (!isTechnicianAuth()) return appointmentDateGroups.value
  const todayKey = formatAppointmentDateKey(new Date(currentTime.value || Date.now()))
  return appointmentDateGroups.value.filter(group => group?.date === todayKey)
})

const visibleAppointmentGroupDays = ref(1)
const appointmentGroupsLoadingMore = ref(false)

const exceptionGroups = computed(() => {
  if (!isTechnicianAuth()) {
    return buildExceptionGroups((appointments.value || []).filter(a => isExceptionAppointment(a)))
  }

  const groups = []
  const unassigned = unassignedAppointments.value.filter(a => isExceptionAppointment(a))
  const mine = assignedAppointments.value.filter(a => isExceptionAppointment(a))
  groups.push(...buildExceptionGroups(unassigned, 'unassigned', '待分配异常'))
  groups.push(...buildExceptionGroups(mine, 'assigned', '我的异常'))
  return groups
})

const appointmentPanelGroups = computed(() => {
  return (showExceptionStandaloneContent.value || showExceptionContentInTable.value) ? exceptionGroups.value : appointmentFlowGroups.value
})

const displayedAppointmentPanelGroups = computed(() => {
  if (showExceptionStandaloneContent.value || showExceptionContentInTable.value) {
    return appointmentPanelGroups.value
  }
  return appointmentDisplayDateGroups.value.slice(0, visibleAppointmentGroupDays.value)
})

const showAppointmentGroupsLoadMore = computed(() => {
  if (showExceptionStandaloneContent.value || showExceptionContentInTable.value) return false
  return appointmentDisplayDateGroups.value.length > visibleAppointmentGroupDays.value
})

const resetVisibleAppointmentGroups = () => {
  visibleAppointmentGroupDays.value = 1
}

const loadMoreAppointmentGroups = async () => {
  if (appointmentGroupsLoadingMore.value) return
  if (!showAppointmentGroupsLoadMore.value) return
  appointmentGroupsLoadingMore.value = true
  try {
    visibleAppointmentGroupDays.value += 1
  } finally {
    appointmentGroupsLoadingMore.value = false
  }
}

const appointmentPanelEmptyText = computed(() => {
  return (showExceptionStandaloneContent.value || showExceptionContentInTable.value) ? '暂无异常单' : '暂无预约'
})

const exceptionSummaryCount = computed(() => {
  if (!showExceptionTab.value) return 0
  return (appointments.value || []).filter(a => isExceptionAppointment(a) && !isAppointmentSettled(a)).length
})

const appointmentSummaryCount = computed(() => {
  if (!showAppointmentTab.value) return 0
  // 统计卡文案是“待确认预约”，这里只统计真正仍待商户确认的 pending，
  // 已 confirmed / arrived 的预约应继续留在列表里，但不应再占用顶部待确认数字。
  return (appointments.value || []).filter(a => !isExceptionAppointment(a) && a?.status === 'pending').length
})

const isAppointmentAlreadyInService = (appt) => {
  if (!appt) return false
  const sessionStatus = normalizeSessionStatus(appt.service_session?.status || appt.session_status)
  if (sessionStatus === 'serving' || sessionStatus === 'auto_finishing') return true
  if (isAppointmentSettled(appt)) return false
  return !!appt.actual_start_at && appt.status === 'arrived'
}

const serviceTabUpcomingAppointments = computed(() => {
  if (!isTechnicianAuth()) return []
  const now = currentTime.value
  const windowMs = 60 * 60 * 1000
  return assignedAppointments.value
    .filter(appt => {
      if (isExceptionAppointment(appt) || isAppointmentSettled(appt)) return false
      if (isAppointmentAlreadyInService(appt)) return false
      const appointmentTimeMs = getAppointmentTimeMs(appt)
      if (appointmentTimeMs === null) return false
      return Math.abs(appointmentTimeMs - now) <= windowMs
    })
    .sort((left, right) => {
      const leftMs = getAppointmentTimeMs(left) ?? 0
      const rightMs = getAppointmentTimeMs(right) ?? 0
      return leftMs - rightMs
    })
})
const todayUsages = ref([])
const todayStartUsages = ref([])
const startUsagesLoading = ref(false)
let startUsagesRefreshing = false
const todayFinishedUsages = ref([])
const notices = ref([])
const currentTime = ref(Date.now())
let countdownTimer = null
let serviceSessionTimer = null
let lastServiceTabRefreshAt = 0

const verifyCodeInput = ref('')
const verifying = ref(false)
const verifyResult = ref(null)
const showVerifyInput = ref(false)
const showHandCardModal = ref(false)
const submittingHandCard = ref(false)
const handCardInput = ref('')
const handCardError = ref('')
const pendingVerifyToken = ref('')
const PENDING_VERIFY_STORAGE_KEY = 'kabao_pending_verify_commit'

const returnHandCardNo = ref('')
const queryingReturnHandCard = ref(false)
const returnHandCardError = ref('')
const returnHandCardUsage = ref(null)
const returningHandCard = ref(false)

const noticeForm = ref({
  title: '',
  content: ''
})

// 扫码错误弹窗
const showErrorModal = ref(false)
const errorMessage = ref('')
let errorTimer = null

const currentView = ref('cards') // 'cards' | 'sellTemplates'
const issuedCards = ref([])
const sellTemplates = ref([])
const displayMode = ref('auto') // 'auto' | 'cards' | 'sellTemplates'

// 售卡模板长按相关
const templateLongPressTimer = ref(null)
const templatePressStart = ref(null)
const filteredSellTemplates = computed(() => {
  if (!sellTemplates.value.length) return []
  
  let filtered = sellTemplates.value
  
  // 按卡片类型过滤
  if (cardSearch.value.card_type) {
    filtered = filtered.filter(tpl => tpl.name === cardSearch.value.card_type)
  }
  
  return filtered
})

const currentDisplay = computed(() => {
  // 如果手动指定了显示模式，优先使用，但要检查权限
  if (displayMode.value !== 'auto') {
    // 如果是卡片模式但没有核销权限，则显示售卡模板
    if (displayMode.value === 'cards' && !canVerify.value) {
      return canSellCards.value ? 'sellTemplates' : 'cards'
    }
    return displayMode.value
  }
  // 如果没有核销权限但有售卡权限，默认显示售卡模板
  if (!canVerify.value && canSellCards.value) {
    return 'sellTemplates'
  }
  return 'cards'
})
const cardsLoading = ref(false)
const cardsError = ref('')
const expandedCardId = ref(null)

const cardSearch = ref({
  card_no: '',
  card_type: ''
})

watch(
  () => cardSearch.value.card_type,
  async () => {
    if (currentTab.value !== 'cards') return
    if (currentDisplay.value !== 'cards') return
    if (!canVerify.value) return
    await fetchIssuedCards()
  }
)

watch(
  () => route.query.error,
  (v) => {
    if (!v) return
    const tabParam = route.query.tab
    showErrorModalWithMessage(String(v))
    // 清除URL中的错误参数，避免重复显示；保留 tab
    router.replace({ path: '/merchant', query: { tab: tabParam || 'appointment' } })
  }
)

const cardTemplates = ref([])

const showBusinessStatusModal = ref(false)

// service tab: 签到

const attendanceLoading = ref(false)
const attendanceUpdating = ref(false)
const attendanceStatus = ref('not_checked_in') // 用户在下拉框中选择的状态
const serverAttendanceStatus = ref('not_checked_in') // 服务器中的真实状态
const attendanceStatusDirty = ref(false)
const attendanceStatusConfirming = ref(false)
const setNextPausedLoading = ref(false)

// 叫号状态管理
const merchantQueuePaused = ref(false)
const technicianQueuePaused = ref(false)
const queueStatusLoading = ref(false)
const queueStatusUpdating = ref(false)

const isInBusinessHours = ref(true)
const queueEndedAt = ref(null)

const isQueueEnded = computed(() => {
  return !isInBusinessHours.value && !!queueEndedAt.value
})

// 继续叫号按钮可见性规则（专业客服端）
const shouldShowContinueCall = computed(() => {
  if (!merchant.value) return false
  if (!merchant.value.support_queue) return false
  if (merchant.value.queue_mode !== 'manual') return false // 仅手动叫号显示
  if (isQueueEnded.value) return false
  if (technicianQueuePaused.value) return false
  if (merchantQueuePaused.value) return false
  // 手动叫号：始终显示"继续叫号"按钮
  // - 有进行中服务：完成服务并分配下一号  
  // - 无进行中服务：直接分配下一号给当前技师
  return true
})

const continueCallLoading = ref(false)

const continueCallBlockedSeconds = ref(0)
let continueCallBlockedTimer = null

const stopContinueCallBlockedTimer = () => {
  if (continueCallBlockedTimer) {
    clearInterval(continueCallBlockedTimer)
    continueCallBlockedTimer = null
  }
}

const startContinueCallBlockedTimer = (seconds) => {
  stopContinueCallBlockedTimer()
  const n = Number(seconds || 0)
  continueCallBlockedSeconds.value = Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
  if (continueCallBlockedSeconds.value <= 0) return
  continueCallBlockedTimer = setInterval(() => {
    if (continueCallBlockedSeconds.value > 0) {
      continueCallBlockedSeconds.value -= 1
    }
    if (continueCallBlockedSeconds.value <= 0) {
      stopContinueCallBlockedTimer()
    }
  }, 1000)
}

const syncContinueCallBlockedFromPendingSession = () => {
  if (!isTechnicianAuth()) return
  if (!merchant.value?.support_queue) return
  if (merchant.value?.queue_mode !== 'manual') return

  const s = pendingStartSession.value
  if (!s) {
    if (continueCallBlockedSeconds.value > 0) {
      stopContinueCallBlockedTimer()
      continueCallBlockedSeconds.value = 0
    }
    return
  }

  const remainFromServer = Number(s.start_pending_remaining_seconds || 0)
  if (Number.isFinite(remainFromServer) && remainFromServer > 0) {
    const remain = Math.floor(remainFromServer)
    if (continueCallBlockedSeconds.value <= 0 || Math.abs(remain - continueCallBlockedSeconds.value) > 2) {
      startContinueCallBlockedTimer(remain)
    }
    return
  }

  const timeoutSeconds = Number(s.start_pending_timeout_seconds || 0) > 0 ? Number(s.start_pending_timeout_seconds) : 180
  const baseRaw = s.updated_at || s.created_at
  if (!baseRaw) return

  const baseTime = new Date(baseRaw).getTime()
  if (!Number.isFinite(baseTime) || baseTime <= 0) return

  const deadline = baseTime + timeoutSeconds * 1000
  const remain = Math.floor((deadline - Date.now()) / 1000)
  if (remain > 0) {
    if (continueCallBlockedSeconds.value <= 0 || Math.abs(remain - continueCallBlockedSeconds.value) > 2) {
      startContinueCallBlockedTimer(remain)
    }
    return
  }

  if (continueCallBlockedSeconds.value > 0) {
    stopContinueCallBlockedTimer()
    continueCallBlockedSeconds.value = 0
  }
}

const showServiceDurationConfirmModal = ref(false)
const serviceDurationConfirmMessage = ref('')
let serviceDurationConfirmResolve = null

const openServiceDurationConfirm = (msg) => {
  serviceDurationConfirmMessage.value = String(msg || '')
  showServiceDurationConfirmModal.value = true
  return new Promise(resolve => {
    serviceDurationConfirmResolve = resolve
  })
}

const confirmServiceDurationConfirm = () => {
  showServiceDurationConfirmModal.value = false
  const resolve = serviceDurationConfirmResolve
  serviceDurationConfirmResolve = null
  if (resolve) resolve(true)
}

const cancelServiceDurationConfirm = () => {
  showServiceDurationConfirmModal.value = false
  const resolve = serviceDurationConfirmResolve
  serviceDurationConfirmResolve = null
  if (resolve) resolve(false)
}

const doContinueCall = async (forceFinish = false) => {
  console.log('[DEBUG] doContinueCall called, forceFinish:', forceFinish)
  if (continueCallLoading.value) {
    console.log('[DEBUG] continueCallLoading is true, returning')
    return
  }
  continueCallLoading.value = true
  console.log('[DEBUG] continueCallLoading set to true')
  try {
    console.log('[DEBUG] calling queueApi.continueCall()')
    const res = await queueApi.continueCall()
    console.log('[DEBUG] continueCall response:', res)
    const data = res?.data?.data || {}
    console.log('[DEBUG] response data:', data)

    // 待上号倒计时未结束：后端返回 need_wait + remaining seconds
    if (data?.need_wait) {
      const remain = Number(data?.pending_remaining_seconds || 0)
      if (remain > 0) {
        startContinueCallBlockedTimer(remain)
      }
      if (data?.reason) {
        alert(String(data.reason))
      }
      await fetchServiceSessions()
      return
    }
    
    // 优先判断是否需要二次确认（服务时长未达标）
    console.log('[DEBUG] checking need_confirm:', data?.need_confirm, 'forceFinish:', forceFinish)
    if (data?.need_confirm && !forceFinish) {
      console.log('[DEBUG] entering confirmation flow')
      const servedMinutes = Number(data.served_minutes || 0)
      const requiredMinutes = Number(data.required_minutes || 0)
      const remainingMinutes = Number(data.remaining_minutes || 0)
      
      const confirmMsg = `当前服务已进行 ${servedMinutes} 分钟，项目要求服务时长为 ${requiredMinutes} 分钟，还差 ${remainingMinutes} 分钟。\n\n确定要结束服务并继续叫号吗？`
      console.log('[DEBUG] calling openServiceDurationConfirm with message:', confirmMsg)
      
      const ok = await openServiceDurationConfirm(confirmMsg)
      console.log('[DEBUG] confirmation result:', ok)
      if (!ok) {
        console.log('[DEBUG] user cancelled, returning')
        continueCallLoading.value = false
        return
      }
      console.log('[DEBUG] user confirmed, calling doContinueCallForce')
      continueCallLoading.value = false
      await doContinueCallForce()
      return
    }
    console.log('[DEBUG] no confirmation needed, proceeding with normal flow')
    
    // 处理其他情况（reason 提示、成功完成等）
    if (data?.reason) {
      alert(String(data.reason))
    } else {
      const nextUsageId = Number(data?.next_usage_id || 0)
      if (nextUsageId > 0) {
        if (Number(data?.skipped_session_id || 0) > 0) {
          alert(`已跳过${getMerchantPendingStartLabel({ queueMode: true })}，并已触发下一号`)
        } else {
          alert('已完成当前服务，并已触发下一号')
        }
      } else {
        alert('已完成当前服务')
      }
    }
    await fetchQueueCallingStatus()
    await fetchServiceSessions()
    await fetchTodayUsages()
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    continueCallLoading.value = false
  }
}

// 强制结束当前服务并继续叫号
const doContinueCallForce = async () => {
  if (continueCallLoading.value) return
  continueCallLoading.value = true
  try {
    const res = await queueApi.continueCallForce()
    const data = res?.data?.data || {}
    
    if (data?.reason) {
      alert(String(data.reason))
    } else {
      const nextUsageId = Number(data?.next_usage_id || 0)
      if (nextUsageId > 0) {
        alert('已强制结束当前服务，并已触发下一号')
      } else {
        alert('已强制结束当前服务')
      }
    }
    await fetchQueueCallingStatus()
    await fetchServiceSessions()
    await fetchTodayUsages()
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    continueCallLoading.value = false
  }
}

const serviceSessions = ref([])
const sessionLoading = ref(false)
const sessionStatusFilter = ref('')

const pendingStartSession = computed(() => {
  if (!isTechnicianAuth()) return null
  const techId = getTechnicianId()
  if (!techId) return null
  const sess = serviceSessions.value
    .filter(s => s.technician_id === techId && normalizeSessionStatus(s.status) === 'start_pending' && !s.start_confirmed_at)
    .sort((a, b) => b.id - a.id)[0]
  return sess || null
})

const roomManageSession = computed(() => {
  if (!isTechnicianAuth()) return null
  const techId = getTechnicianId()
  if (!techId) return null
  const activeStatuses = ['start_pending', 'delay_pending', 'serving', 'auto_finishing']
  const sess = serviceSessions.value
    .filter(s => s.technician_id === techId && activeStatuses.includes(normalizeSessionStatus(s.status)))
    .sort((a, b) => b.id - a.id)[0]
  return sess || null
})

const roomManageUsage = computed(() => {
  const usageID = Number(roomManageSession.value?.initial_usage_id || 0)
  if (!usageID) return null
  return (todayStartUsages.value || []).find(usage => Number(usage?.id || 0) === usageID) || null
})

const showRoomManageCard = computed(() => {
  if (isQueueModeView.value && isTechnicianAuth()) {
    return !queueBlockedByAttendance.value || !!queueCallInfo.value
  }
  if (!isTechnicianAuth()) return false
  if (roomManageSession.value && !roomManageUsage.value) return true
  return !!canRoomManage.value && !!merchant.value?.support_room && todayStartUsages.value.length === 0
})

const pendingStartServiceCount = computed(() => {
  if (!isTechnicianAuth()) return 0
  const techId = getTechnicianId()
  if (!techId) return 0
  return (serviceSessions.value || []).filter((session) => {
    if (Number(session?.technician_id || 0) !== Number(techId)) return false
    if (session?.start_confirmed_at) return false
    const status = normalizeSessionStatus(session?.status)
    return status === 'start_pending' || status === 'delay_pending'
  }).length
})

const roomManagePhaseText = computed(() => {
  const s = roomManageSession.value
  if (!s) return ''
  if (normalizeSessionStatus(s.status) === 'start_pending' && !s.start_confirmed_at) return getMerchantPendingStartLabel()
  return '服务中'
})

const queueCallPhaseText = computed(() => {
  const info = queueCallInfo.value
  const s = info?.session
  if (!s) return ''
  if (normalizeSessionStatus(s.status) === 'start_pending' && !s.start_confirmed_at) return getMerchantPendingStartLabel({ queueMode: true })
  return '服务中'
})

const getQueueCallSessionStartPendingRemainingSeconds = () => {
  return getStartPendingRemainingSeconds(queueCallInfo.value?.session)
}

const getQueueCallSessionStartPendingRemainingClass = () => {
  const remain = getQueueCallSessionStartPendingRemainingSeconds()
  if (remain === null) return 'text-blue-600'
  if (remain <= 60) return 'text-red-500'
  return 'text-blue-600'
}

const roomManageRoomText = computed(() => {
  const s = roomManageSession.value
  if (!s) return '-'
  const roomName = s.room?.name
  if (roomName) return roomName
  if (s.room_id) return String(s.room_id)
  return '-'
})

const roomManageTrackingId = computed(() => {
  const s = roomManageSession.value
  if (!s) return null
  return s.initial_usage_id || s.id
})

const roomManageProjectName = computed(() => {
  const s = roomManageSession.value
  if (!s) return '-'
  // 直接从 ServiceSession 中获取 Project 信息，不再依赖 Usage 数据
  return formatProjectNameWithDuration(s.project)
})

const getSessionRemainingSeconds = (sess) => {
  if (!sess) return null
  const s = normalizeSessionStatus(sess.status)
  if (s !== 'serving' && s !== 'auto_finishing') return null

  let finishAt = 0
  const finishAtRaw = sess.scheduled_finish_at
  if (finishAtRaw) {
    finishAt = new Date(finishAtRaw).getTime()
  }
  if (!finishAt || Number.isNaN(finishAt)) {
    const startedAtRaw = sess.started_at
    const durationMinutes = Number(sess.duration_minutes || 0)
    if (!startedAtRaw || !Number.isFinite(durationMinutes) || durationMinutes <= 0) return null
    const startedAt = new Date(startedAtRaw).getTime()
    if (!startedAt || Number.isNaN(startedAt)) return null
    finishAt = startedAt + durationMinutes * 60 * 1000
  }

  const remain = Math.floor((finishAt - currentTime.value) / 1000)
  if (!Number.isFinite(remain)) return null
  return Math.max(0, remain)
}

const getRoomManageSessionRemainingSeconds = () => {
  return getSessionRemainingSeconds(roomManageSession.value)
}

const getRoomManageStartPendingCountdownSeconds = () => {
  return getStartPendingRemainingSeconds(roomManageSession.value)
}

const getRoomManageServiceCountdownSeconds = () => {
  const sess = roomManageSession.value
  if (!sess) return null
  const status = normalizeSessionStatus(sess.status)
  if (status !== 'serving' && status !== 'auto_finishing') return null
  return getRoomManageSessionRemainingSeconds()
}

const getSessionOccupancyDurationText = (sess) => {
  if (!sess) return ''
  const startRaw = sess.started_at || sess.room_locked_at
  if (!startRaw) return ''
  const startAt = new Date(startRaw).getTime()
  if (!Number.isFinite(startAt) || startAt <= 0) return ''
  const elapsedSeconds = Math.max(0, Math.floor((currentTime.value - startAt) / 1000))
  const hours = Math.floor(elapsedSeconds / 3600)
  const minutes = Math.floor((elapsedSeconds % 3600) / 60)
  const seconds = elapsedSeconds % 60
  if (hours === 0) {
    return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
  }
  if (hours < 10) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
  }
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

const getRoomManageOccupancyDurationText = () => {
  return getSessionOccupancyDurationText(roomManageSession.value)
}

// 格式化项目名称（加上时长）
const formatProjectNameWithDuration = (project) => {
  if (!project || !project.name) return '-'
  const duration = Number(project.duration)
  if (Number.isFinite(duration) && duration > 0) {
    return `${project.name}（${duration}分钟）`
  }
  return project.name
}

const pendingStartRoomText = computed(() => {
  const s = pendingStartSession.value
  if (!s) return '-'
  const roomName = s.room?.name
  if (roomName) return roomName
  if (s.room_id) return String(s.room_id)
  return '-'
})

const formatSessionNo = (id) => {
  const n = Number(id)
  if (!Number.isFinite(n) || n <= 0) return '-'
  return String(n).padStart(9, '0')
}

// 技师当前服务状态（用于显示）
const isTechnicianNotCheckedIn = computed(() => {
  return isTechnicianAuth() && (serverAttendanceStatus.value === 'not_checked_in' || serverAttendanceStatus.value === 'rest')
})

const technicianCurrentStatus = computed(() => {
  if (!isTechnicianAuth()) return null
  const techId = getTechnicianId()
  if (!techId) return null
  const sess = serviceSessions.value.find(s => s.technician_id === techId && ['room_selecting', 'room_locked', 'staff_selecting', 'start_pending', 'delay_pending', 'serving', 'auto_finishing'].includes(normalizeSessionStatus(s.status)))
  if (!sess) {
    // 没有活跃会话，返回服务器中的签到状态 idle/paused
    return serverAttendanceStatus.value
  }
  const baseStatus = normalizeSessionStatus(sess.status)
  if (['room_selecting', 'room_locked', 'staff_selecting'].includes(baseStatus)) return 'service_pending_presettlement'
  if ((baseStatus === 'start_pending' || baseStatus === 'delay_pending') && !sess.start_confirmed_at) return 'service_pending_presettlement'
  return 'service_pending_settlement'
})

const technicianCurrentStatusText = computed(() => {
  const st = technicianCurrentStatus.value
  // 叫号模式下的状态显示
  if (isQueueModeView.value) {
    if (st === 'not_checked_in') return '未签到'
    if (st === 'idle') return '空闲'
    if (st === 'paused') return '暂停'
    if (st === 'service_pending_presettlement') return getMerchantPendingStartLabel({ queueMode: true })
    if (st === 'service_pending_settlement') return getMerchantPendingStartLabel({ queueMode: true }).replace('待', '')
    if (st === 'busy') return getMerchantPendingStartLabel({ queueMode: true }).replace('待', '')
    if (st === 'rest') return '未签到 休息中'
  }
  // 非叫号模式保持原有显示
  if (st === 'not_checked_in') return '未签到'
  if (st === 'idle') return '空闲'
  if (st === 'paused') return '暂停'
  if (st === 'busy') return '忙碌'
  if (st === 'rest') return '未签到 休息中'
  if (st === 'service_pending_presettlement') return getMerchantServicePendingStartLabel()
  if (st === 'service_pending_settlement') return getMerchantServicePendingFinishLabel()
  return st || '-'
})

const canManualUpdateStatus = computed(() => {
  // 允许在"空闲"时手动更新，也允许在"暂停"时手动更新到任何状态
  const currentStatus = technicianCurrentStatus.value
  return currentStatus === 'idle' || currentStatus === 'paused'
})

const hasActiveServingSession = computed(() => {
  if (!isTechnicianAuth()) return false
  const techId = Number(getTechnicianId() || 0)
  if (!techId) return false
  return (serviceSessions.value || []).some((session) => {
    if (Number(session?.technician_id || 0) !== techId) return false
    const status = normalizeSessionStatus(session?.status)
    return status === 'serving' || status === 'auto_finishing'
  })
})

const showScanStartButton = computed(() => {
  return isTechnicianAuth() && !isTechnicianNotCheckedIn.value && !hasActiveServingSession.value
})

const showCheckOutButton = computed(() => {
  return isTechnicianAuth() && !isTechnicianNotCheckedIn.value && !hasActiveServingSession.value
})

const applyAttendanceStatusFromServer = (status) => {
  const nextStatus = String(status || 'not_checked_in')
  serverAttendanceStatus.value = nextStatus
  if (!attendanceStatusDirty.value && !attendanceStatusConfirming.value && !attendanceUpdating.value) {
    attendanceStatus.value = nextStatus
  }
}

const resetAttendanceDraftToServer = () => {
  attendanceStatusDirty.value = false
  attendanceStatusConfirming.value = false
  attendanceStatus.value = String(serverAttendanceStatus.value || 'not_checked_in')
}

const submitAttendanceStatusChange = async (targetStatus) => {
  const nextStatus = String(targetStatus || '')
  if (!nextStatus) return false
  if (nextStatus === String(serverAttendanceStatus.value || '')) {
    resetAttendanceDraftToServer()
    return true
  }
  if (nextStatus === 'paused' && String(serverAttendanceStatus.value || '') !== 'paused') {
    const canContinue = await maybeReassignPendingBeforeLeave('暂停服务')
    if (!canContinue) {
      resetAttendanceDraftToServer()
      return false
    }
  }

  attendanceUpdating.value = true
  attendanceStatusConfirming.value = false
  try {
    await attendanceApi.updateStatus({ status: nextStatus })
    applyAttendanceStatusFromServer(nextStatus)
    attendanceStatusDirty.value = false

    if (merchant.value?.support_queue) {
      if (nextStatus === 'paused') {
        try {
          await queueApi.updateTechnicianQueuePaused(true)
          technicianQueuePaused.value = true
          await fetchQueueCallingStatus()
        } catch (e) {
          // ignore
        }
      }
      if (nextStatus === 'idle') {
        try {
          await queueApi.updateTechnicianQueuePaused(false)
          technicianQueuePaused.value = false
          await fetchQueueCallingStatus()
        } catch (e) {
          // ignore
        }
      }
    }

    return true
  } catch (e) {
    resetAttendanceDraftToServer()
    alert(e.response?.data?.error || '更新失败')
    return false
  } finally {
    attendanceUpdating.value = false
    attendanceStatusConfirming.value = false
  }
}

const statusSelectValue = computed({
  get() {
    // 可手动更新时：显示用户选择的状态
    // 不可手动更新(下拉框 disabled)时：显示真实的当前状态，避免与“当前状态”文案不一致
    if (!canManualUpdateStatus.value) {
      return String(technicianCurrentStatus.value || '')
    }
    return String(attendanceStatus.value || '')
  },
})

const handleAttendanceStatusSelectChange = async (event) => {
  if (!canManualUpdateStatus.value) return
  const nextStatus = String(event?.target?.value || '')
  if (nextStatus !== 'paused' && nextStatus !== 'idle') {
    if (event?.target) {
      event.target.value = String(statusSelectValue.value || '')
    }
    return
  }
  if (nextStatus === String(serverAttendanceStatus.value || '')) {
    resetAttendanceDraftToServer()
    if (event?.target) {
      event.target.value = String(statusSelectValue.value || '')
    }
    return
  }

  attendanceStatusConfirming.value = true
  const label = nextStatus === 'paused' ? '暂停' : '空闲'
  const confirmed = confirm(`确认要将当前状态更新为“${label}”吗？`)
  if (!confirmed) {
    attendanceStatusConfirming.value = false
    resetAttendanceDraftToServer()
    if (event?.target) {
      event.target.value = String(statusSelectValue.value || '')
    }
    return
  }

  attendanceStatus.value = nextStatus
  attendanceStatusDirty.value = true
  if (event?.target) {
    event.target.value = nextStatus
  }
  await submitAttendanceStatusChange(nextStatus)
}

// 技师视角：我的服务中会话
const myServingSessions = computed(() => {
  if (!isTechnicianAuth()) return []
  const techId = getTechnicianId()
  if (!techId) return []
  return serviceSessions.value.filter(s => s.technician_id === techId && ['delay_pending', 'serving', 'auto_finishing'].includes(normalizeSessionStatus(s.status)))
})

// 其他会话（技师视角）或全部会话（商户视角）
const filteredOtherSessions = computed(() => {
  if (!isTechnicianAuth()) return serviceSessions.value
  const myIds = new Set(myServingSessions.value.map(s => s.id))
  return serviceSessions.value.filter(s => !myIds.has(s.id))
})

// 加钟弹窗状态
const extendLoadingIds = ref(new Set())
const showExtendModalVisible = ref(false)
const extendSession = ref(null)
const extendMinutes = ref(null)
const showExtendRequestReviewModal = ref(false)
const selectedExtendUsage = ref(null)
const extendReviewLoading = ref(false)
const extendRejectReason = ref('')

const selectedExtendRequest = computed(() => selectedExtendUsage.value?.latest_extend_request || null)

const hasPendingExtendRequest = (usage) => {
  return usage?.latest_extend_request?.status === 'pending'
}

const isUsageServiceClosed = (usage) => {
  if (!usage) return false
  if (String(usage.status || '').trim() === 'success') return true
  return normalizeSessionStatus(usage.service_session_status) === 'finished'
}

const formatExtendClockProjectDurationText = (projectName, minutes) => {
  const name = String(projectName || '').trim() || '-'
  const duration = Number(minutes || 0)
  return duration > 0 ? `${name}（${duration}分钟）` : name
}

const getUsageFirstClockMinutes = (usage, req) => {
  const projectDuration = Number(usage?.project?.duration || 0)
  if (projectDuration > 0) return projectDuration
  const totalDuration = Number(usage?.service_session_duration_minutes || 0)
  const extendMinutes = Number(req?.minutes || 0)
  if (totalDuration > extendMinutes) return totalDuration - extendMinutes
  return totalDuration
}

const getUsageApprovedExtendInfoLines = (usage) => {
  const req = usage?.latest_extend_request
  if (!req || String(req.status || '').trim() !== 'approved') return []
  const minutes = Number(req.minutes || 0)
  const projectName = String(req.project?.name || '').trim()
  if (isUsageServiceClosed(usage)) {
    return [
      `第一个钟：${formatExtendClockProjectDurationText(usage?.project?.name, getUsageFirstClockMinutes(usage, req))}`,
      `第二个钟：${formatExtendClockProjectDurationText(projectName, minutes)}`
    ]
  }
  let before = Number(req.before_remaining_seconds || 0)
  let after = Number(req.after_remaining_seconds || 0)
  if (after <= 0 && minutes > 0) {
    const currentRemain = getUsageServiceRemainingSeconds(usage)
    if (currentRemain !== null) {
      after = currentRemain
      before = Math.max(0, currentRemain - minutes * 60)
    }
  }
  const beforeText = formatRemainingSeconds(before)
  const afterText = formatRemainingSeconds(after)
  const projectText = projectName ? `（${projectName} ${minutes}分钟）` : `（${minutes}分钟）`
  const firstLine = `已加钟${projectText}`
  if (!beforeText || !afterText) return [firstLine]
  return [firstLine, `加钟前剩余 ${beforeText}`, `加钟后剩余 ${afterText}`]
}

const hasUsageApprovedExtendAfterRemaining = (usage) => {
  return getUsageApprovedExtendInfoLines(usage).some(line => String(line || '').startsWith('加钟后剩余'))
}

const shouldShowUsageServiceRemainingSeconds = (usage) => {
  return getUsageServiceRemainingSeconds(usage) !== null && !hasUsageApprovedExtendAfterRemaining(usage)
}

const getCardTypeLabel = (type) => {
  const labels = { times: '次数卡', lesson: '课时卡', balance: '充值卡' }
  return labels[type] || type
}

const goScanVerify = () => {
  router.push({
    path: '/merchant/scan-verify',
    query: {
      mode: 'verify',
      queue_mode: isQueueModeMerchant(merchant.value) ? '1' : '0',
      start_term: merchant.value?.start_term || '',
      finish_term: merchant.value?.finish_term || ''
    }
  })
}

const goScanStart = () => {
  if (hasActiveServingSession.value) return
  router.push({
    path: '/merchant/scan-verify',
    query: {
      mode: 'start',
      queue_mode: isQueueModeMerchant(merchant.value) ? '1' : '0',
      start_term: merchant.value?.start_term || '',
      finish_term: merchant.value?.finish_term || ''
    }
  })
}

// 叫号状态管理方法
const fetchQueueCallingStatus = async () => {
  if (!canQueueCalling.value) return
  queueStatusLoading.value = true
  try {
    const res = await queueApi.getCallingStatus()
    const data = res.data?.data || {}
    merchantQueuePaused.value = !!data.queue_paused
    isInBusinessHours.value = data.is_in_business_hours !== false
    queueEndedAt.value = data.queue_ended_at || null
    if (data.technician_queue_paused !== null && data.technician_queue_paused !== undefined) {
      technicianQueuePaused.value = !!data.technician_queue_paused
    }
  } catch (e) {
    console.error('获取叫号状态失败', e)
  } finally {
    queueStatusLoading.value = false
  }
}

const startMerchantQueue = async () => {
  if (queueStatusUpdating.value) return
  queueStatusUpdating.value = true
  try {
    // 调用 callNext API 恢复叫号并触发下一个
    const res = await queueApi.callNext()
    merchantQueuePaused.value = false
    const nextUsageId = res.data?.data?.next_usage_id
    if (nextUsageId && nextUsageId > 0) {
      alert('叫号已开始，已触发下一个叫号')
    } else {
      alert('叫号已开始（当前没有等待叫号的用户）')
    }
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    queueStatusUpdating.value = false
  }
}

const pauseMerchantQueue = async () => {
  if (queueStatusUpdating.value) return
  queueStatusUpdating.value = true
  try {
    await queueApi.updateCallingStatus(true)
    merchantQueuePaused.value = true
    await fetchQueueCallingStatus()
    alert(isQueueEnded.value ? '叫号已结束' : '叫号已暂停')
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    queueStatusUpdating.value = false
  }
}

const startTechnicianQueue = async () => {
  if (queueStatusUpdating.value) return
  queueStatusUpdating.value = true
  try {
    // 专业客服点击"开始叫号"也是启动商户的整个叫号服务
    const res = await queueApi.callNext()
    merchantQueuePaused.value = false
    const nextUsageId = res.data?.data?.next_usage_id
    if (nextUsageId && nextUsageId > 0) {
      alert('叫号已开始，已触发下一个叫号')
    } else {
      alert('叫号已开始（当前没有等待叫号的用户）')
    }
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    queueStatusUpdating.value = false
  }
}

const pauseTechnicianQueue = async () => {
  if (queueStatusUpdating.value) return
  const canContinue = await maybeReassignPendingBeforeLeave('暂停叫号')
  if (!canContinue) return
  queueStatusUpdating.value = true
  try {
    // 专业客服点击"暂停叫号"是暂停自己的叫号服务
    await queueApi.updateTechnicianQueuePaused(true)
    technicianQueuePaused.value = true
    // 叫号模式下：同步将技师当前状态改为“暂停”
    if (merchant.value?.support_queue) {
      try {
        await attendanceApi.updateStatus({ status: 'paused' })
        applyAttendanceStatusFromServer('paused')
        attendanceStatusDirty.value = false
      } catch (e) {
        // ignore
      }
    }
    await fetchQueueCallingStatus()
    alert(isQueueEnded.value ? '叫号已结束' : '您的叫号已暂停')
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    queueStatusUpdating.value = false
  }
}

const resumeTechnicianQueue = async () => {
  if (queueStatusUpdating.value) return
  queueStatusUpdating.value = true
  try {
    await queueApi.updateTechnicianQueuePaused(false)
    technicianQueuePaused.value = false
    // 叫号模式下：同步将技师当前状态改为“空闲”
    if (merchant.value?.support_queue) {
      try {
        await attendanceApi.updateStatus({ status: 'idle' })
        applyAttendanceStatusFromServer('idle')
        attendanceStatusDirty.value = false
      } catch (e) {
        // ignore
      }
    }
    alert('您的叫号已恢复')
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    queueStatusUpdating.value = false
  }
}

// 是否显示叫号控制区域（运营客服端 - 扫码核销页）
const showQueueControlInVerify = computed(() => {
  // 条件：开启了叫号 + 有叫号权限 + 非技师账号（运营客服或商户）（支持手动和自动叫号）
  return merchant.value?.support_queue && 
         canQueueCalling.value &&
         !isTechnicianAuth()
})

// 是否显示叫号控制区域（专业客服端 - 服务标签页）
const showQueueControlInService = computed(() => {
  // 条件：开启了叫号 + 有叫号权限 + 技师账号（支持手动和自动叫号）
  return merchant.value?.support_queue && 
         canQueueCalling.value &&
         isTechnicianAuth()
})

// 兼容旧模板引用：当前 finish tab 未启用，但需要保留方法以避免编译报错
const goScanFinish = () => {
  goScanVerify()
}

const showExtendModal = (session) => {
  extendSession.value = session
  extendMinutes.value = null
  showExtendModalVisible.value = true
}

const closeExtendModal = () => {
  showExtendModalVisible.value = false
  extendSession.value = null
  extendMinutes.value = null
}

const openExtendRequestReview = (usage) => {
  if (!hasPendingExtendRequest(usage)) return
  selectedExtendUsage.value = usage
  extendRejectReason.value = ''
  showExtendRequestReviewModal.value = true
}

const closeExtendRequestReview = () => {
  showExtendRequestReviewModal.value = false
  selectedExtendUsage.value = null
  extendRejectReason.value = ''
}

const refreshAfterExtendReview = async () => {
  closeExtendRequestReview()
  await Promise.all([
    fetchTodayStartUsages({ silent: true }),
    fetchServiceSessions()
  ])
}

const approveExtendRequest = async () => {
  const requestID = Number(selectedExtendRequest.value?.id || 0)
  if (!requestID || extendReviewLoading.value) return
  extendReviewLoading.value = true
  try {
    await serviceSessionApi.approveExtendRequest(requestID)
    await refreshAfterExtendReview()
  } catch (e) {
    alert(e.response?.data?.error || '确认加钟失败')
  } finally {
    extendReviewLoading.value = false
  }
}

const rejectExtendRequest = async () => {
  const requestID = Number(selectedExtendRequest.value?.id || 0)
  if (!requestID || extendReviewLoading.value) return
  extendReviewLoading.value = true
  try {
    await serviceSessionApi.rejectExtendRequest(requestID, { reject_reason: extendRejectReason.value })
    await refreshAfterExtendReview()
  } catch (e) {
    alert(e.response?.data?.error || '拒绝加钟失败')
  } finally {
    extendReviewLoading.value = false
  }
}

const doExtendSession = async () => {
  if (!extendSession.value?.id || !extendMinutes.value || extendMinutes.value < 5 || extendMinutes.value > 180) {
    alert('请输入5~180分钟的加钟时长')
    return
  }
  extendLoadingIds.value.add(extendSession.value.id)
  try {
    const res = await serviceSessionApi.extendDuration(extendSession.value.id, { minutes: extendMinutes.value })
    const updated = res.data?.data
    if (updated) {
      // 更新会话列表中的对应项
      const idx = serviceSessions.value.findIndex(s => s.id === updated.id)
      if (idx !== -1) serviceSessions.value[idx] = updated
    }
    closeExtendModal()
  } catch (e) {
    alert(e.response?.data?.error || '加钟失败')
  } finally {
    extendLoadingIds.value.delete(extendSession.value.id)
  }
}

const onTopScanClick = () => {
  if (Date.now() < suppressTopScanClickUntil.value) return
  // 顶部扫码入口也按同样规则：
  // - 只有结单权限：进入起单模式（只起单，不核销）
  // - 同时有核销+结单：进入智能模式（优先核销，满足条件才结单）
  // - 只有核销：进入核销模式
  if (!canVerify.value && canFinishVerify.value) {
    goScanStart()
    return
  }
  goScanVerify()
}

const onTopScanTouchStart = (e) => {
  if (topScanLongPressTimer) {
    clearTimeout(topScanLongPressTimer)
    topScanLongPressTimer = null
  }
  topScanStart = null
  try {
    prevTopScanBodyStyle.userSelect = document.body.style.userSelect
    prevTopScanBodyStyle.webkitUserSelect = document.body.style.webkitUserSelect
    prevTopScanBodyStyle.webkitTouchCallout = document.body.style.webkitTouchCallout
    document.documentElement.classList.add('kb-no-select')
    document.body.classList.add('kb-no-select')
    document.body.style.userSelect = 'none'
    document.body.style.webkitUserSelect = 'none'
    document.body.style.webkitTouchCallout = 'none'
  } catch (_) {
    // ignore
  }
  topScanLongPressTimer = setTimeout(() => {
    suppressTopScanClickUntil.value = Date.now() + 900
    router.push('/merchant/scan-card')
  }, 820)
}

const onTopScanTouchMove = (e) => {
  if (!topScanLongPressTimer) return
  const t = e?.touches?.[0]
  if (!t) return
  if (!topScanStart) {
    topScanStart = { x: t.clientX, y: t.clientY }
    return
  }
  const dx = t.clientX - topScanStart.x
  const dy = t.clientY - topScanStart.y
  if (dx * dx + dy * dy > 12 * 12) {
    clearTimeout(topScanLongPressTimer)
    topScanLongPressTimer = null
  }
}

const onTopScanTouchEnd = () => {
  if (topScanLongPressTimer) {
    clearTimeout(topScanLongPressTimer)
    topScanLongPressTimer = null
  }
  try {
    document.documentElement.classList.remove('kb-no-select')
    document.body.classList.remove('kb-no-select')
    document.body.style.userSelect = prevTopScanBodyStyle.userSelect
    document.body.style.webkitUserSelect = prevTopScanBodyStyle.webkitUserSelect
    document.body.style.webkitTouchCallout = prevTopScanBodyStyle.webkitTouchCallout
  } catch (_) {
    // ignore
  }
}

const goToDirectPurchaseOrders = () => {
  if (!merchant.value.support_direct_sale) return
  router.push({ path: '/merchant/shop-manage', query: { tab: 'orders' } })
}

// 错误弹窗处理
const closeErrorModal = () => {
  if (errorTimer) {
    clearTimeout(errorTimer)
    errorTimer = null
  }
  showErrorModal.value = false
}

const showErrorModalWithMessage = (message) => {
  errorMessage.value = message
  showErrorModal.value = true
  
  // 10秒后自动关闭
  if (errorTimer) clearTimeout(errorTimer)
  errorTimer = setTimeout(() => {
    showErrorModal.value = false
  }, 10000)
}

// 获取操作人名称
const getOperatorName = (usage) => {
  // 如果有技师信息，显示技师姓名或账号
  if (usage.technician) {
    return usage.technician.name || usage.technician.account || '技师'
  }
  // 如果没有技师信息，显示商户店名（商户老板号操作）
  if (usage.merchant) {
    return usage.merchant.name || '店铺'
  }
  return '-'
}

// 获取核销记录的完整操作人信息（包括结单人员）
const getVerifyOperatorInfo = (usage) => {
  const operatorInfo = []
  
  // 核销人员
  if (usage.technician) {
    operatorInfo.push(`核销：${usage.technician.name || usage.technician.account || '技师'}`)
  } else if (usage.merchant) {
    operatorInfo.push(`核销：${usage.merchant.name || '店铺'}`)
  }
  
  // 如果已结单且有结单服务，显示结单人员
  if (usage.status === 'success' && usage.finished_at && merchant.value?.support_customer_service) {
    if (usage.technician_id && usage.technician) {
      // 技师结单
      operatorInfo.push(`${replaceTerms('结单', merchant.value)}：${usage.technician.name || usage.technician.account || '技师'}`)
    } else if (!usage.technician_id && usage.merchant) {
      // 商户老板结单
      operatorInfo.push(`${replaceTerms('结单', merchant.value)}：${usage.merchant.name || '店铺'}`)
    }
  }
  
  return operatorInfo.join(' / ')
}

const getVerifyOperatorPrimaryInfo = (usage) => {
  const fullText = getVerifyOperatorInfo(usage)
  if (!fullText) return '-'
  return fullText.split(' / ')[0]
}

const getUsageServiceTechnicianLabel = (usage) => {
  const technician = usage?.service_technician
  if (!technician) return ''
  const roleName = String(technician?.service_role?.name || '').trim() || replaceTerms('客服', merchant.value)
  const account = String(technician?.account || technician?.code || '').trim()
  const name = String(technician?.name || '').trim()
  if (account && name) return `${roleName}：${account} - ${name}`
  if (account) return `${roleName}：${account}`
  if (name) return `${roleName}：${name}`
  return ''
}

const getUsageTrackingNumber = (usage) => {
  if (!usage?.id) return ''
  return String(usage.id).padStart(9, '0')
}

const getUsageAppointmentNumber = (usage) => {
  const appointmentID = Number(usage?.appointment_id || 0)
  return appointmentID > 0 ? appointmentID : ''
}

const getUsageServiceRemainingSeconds = (usage) => {
  if (!usage) return null
  const s = normalizeSessionStatus(usage.service_session_status)
  if (s !== 'serving' && s !== 'auto_finishing') return null

  let finishAt = 0
  const finishAtRaw = usage.service_session_scheduled_finish_at
  if (finishAtRaw) {
    finishAt = new Date(finishAtRaw).getTime()
  }
  if (!finishAt || Number.isNaN(finishAt)) {
    const startedAtRaw = usage.service_session_started_at
    const durationMinutes = Number(usage.service_session_duration_minutes || 0)
    if (!startedAtRaw || !Number.isFinite(durationMinutes) || durationMinutes <= 0) return null
    const startedAt = new Date(startedAtRaw).getTime()
    if (!startedAt || Number.isNaN(startedAt)) return null
    finishAt = startedAt + durationMinutes * 60 * 1000
  }

  const remain = Math.floor((finishAt - currentTime.value) / 1000)
  if (!Number.isFinite(remain)) return null
  return Math.max(0, remain)
}

const getUsageStartPendingCountdownSeconds = (usage) => {
  if (!usage) return null
  return getStartPendingRemainingSeconds({
    status: usage.service_session_status,
    start_confirmed_at: usage.service_session_start_confirmed_at,
    start_pending_timeout_seconds: usage.service_session_start_pending_timeout_seconds,
    start_pending_remaining_seconds: usage.service_session_start_pending_remaining_seconds,
    updated_at: usage.service_session_updated_at,
    created_at: usage.service_session_created_at,
    _countdown_fetched_at: usage._countdown_fetched_at
  })
}

const getUsageRoomText = (usage) => {
  const room = usage?.service_room
  if (!room) return ''
  const label = String(room.name || room.code || '').trim()
  return label ? `房间号：${label}` : ''
}

const getUsageRoomOccupancyDurationText = (usage) => {
  if (!usage?.service_room) return ''
  const status = normalizeSessionStatus(usage.service_session_status)
  if (!['start_pending', 'delay_pending', 'serving', 'auto_finishing'].includes(status)) return ''
  return getSessionOccupancyDurationText({
    started_at: usage.service_session_started_at,
    room_locked_at: usage.room_locked_at
  })
}

const getQueueCallSessionRemainingSeconds = () => {
  return getSessionRemainingSeconds(queueCallInfo.value?.session)
}

const formatRemainingSeconds = (seconds) => {
  const n = Number(seconds)
  if (!Number.isFinite(n) || n < 0) return ''
  const totalSeconds = Math.floor(n)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const secs = totalSeconds % 60
  if (hours > 0) return `${hours}小时${minutes}分${secs}秒`
  if (minutes > 0) return `${minutes}分${secs}秒`
  return `${secs}秒`
}

const formatServiceCountdownSeconds = (seconds) => {
  const n = Number(seconds)
  if (!Number.isFinite(n) || n < 0) return ''
  const totalSeconds = Math.floor(n)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const secs = totalSeconds % 60
  const pad2 = (value) => String(value).padStart(2, '0')
  if (hours > 0) return `${hours}小时${pad2(minutes)}分${pad2(secs)}秒`
  return `${minutes}分${pad2(secs)}秒`
}

const formatStartCountdownSeconds = (seconds) => {
  const n = Number(seconds)
  if (!Number.isFinite(n) || n < 0) return ''
  const totalSeconds = Math.floor(n)
  const minutes = Math.floor(totalSeconds / 60)
  const secs = totalSeconds % 60
  return `${minutes}:${String(secs).padStart(2, '0')}`
}

const getRemainingSecondsClass = (seconds) => {
  const remain = Number(seconds)
  if (!Number.isFinite(remain)) return 'text-blue-600'
  if (remain <= 60) return 'text-red-500'
  return 'text-blue-600'
}

const getUsageServiceStatusText = (usage) => {
  if (!usage) return '-'

  // 已结单
  if ((usage.status === 'success' && usage.finished_at) || normalizeSessionStatus(usage.service_session_status) === 'finished') {
    return replaceTerms('已结单', merchant.value)
  }

  // 优先按服务单状态展示（避免将待选房间等阶段误显示为“待结单/待下钟”）
  if (usage.service_session_status) {
    const s = normalizeSessionStatus(usage.service_session_status)
    if (s === 'finished') return '完成'
    if (s === 'room_selecting') return '待选房间'
    if (s === 'room_locked') return '房间已锁定'
    if (s === 'staff_selecting') return replaceTerms('待选客服', merchant.value)
    if (s === 'start_pending') {
      return getMerchantPendingStartLabel({ queueMode: isQueueModeMerchant(merchant.value) })
    }
    if (s === 'delay_pending') return getMerchantPendingStartLabel({ queueMode: isQueueModeMerchant(merchant.value) })
    if (s === 'serving') return replaceTerms('服务中', merchant.value)
    if (s === 'auto_finishing') return getMerchantAutoFinishLabel()
    if (s === 'created') return '已创建'
    if (s === 'canceled') return '已取消'
  }

  // 待起单
  if (normalizeSessionStatus(usage.service_session_status) === 'start_pending') {
    return getMerchantPendingStartLabel({ queueMode: isQueueModeMerchant(merchant.value) })
  }

  // 结单超时（超过12小时仍未结单）
  if (usage.used_at && !usage.finished_at) {
    const usedAt = new Date(usage.used_at)
    if (!Number.isNaN(usedAt.getTime())) {
      const diffMs = Date.now() - usedAt.getTime()
      if (diffMs > 12 * 60 * 60 * 1000) {
        return replaceTerms('结单超时', merchant.value)
      }
    }
  }

  return '-'
}

// 获取核销次数显示文本（总次数 / 当前次数）
const getUsageCountDisplayText = (usage) => {
  if (!usage || !usage.card) return { totalTimes: '-', usedTimes: '-' }
  
  const card = usage.card
  const totalTimes = Number(usage.card_total_times_snapshot ?? card.total_times ?? 0)
  if (usage.card_used_times_snapshot !== undefined && usage.card_used_times_snapshot !== null) {
    return { totalTimes, usedTimes: Number(usage.card_used_times_snapshot || 0) }
  }

  const currentRemainTimes = Number(card.remain_times || 0)
  
  // 计算该usage在其所属卡片的核销序号
  // 需要找出同一卡片在todayUsages中的所有记录，按时间排序后计算序号
  const cardUsages = todayUsages.value.filter(u => u.card_id === usage.card_id && u.status !== 'failed')
  
  // 按核销时间升序排列（早的在前）
  cardUsages.sort((a, b) => {
    const timeA = a.used_at ? new Date(a.used_at).getTime() : 0
    const timeB = b.used_at ? new Date(b.used_at).getTime() : 0
    return timeA - timeB
  })
  
  // 计算当前usage的索引位置
  const currentIndex = cardUsages.findIndex(u => u.id === usage.id)
  if (currentIndex === -1) {
    // 如果找不到，降级处理
    const usedTimes = totalTimes - currentRemainTimes
    return { totalTimes, usedTimes }
  }
  
  // 从最新状态（卡片当前剩余次数）反推这条记录的序号
  // 当前已使用总次数 = total_times - remain_times
  const currentUsedTotal = totalTimes - currentRemainTimes
  
  // 计算该记录之后还有多少次核销
  let usagesAfterCurrent = 0
  for (let i = currentIndex + 1; i < cardUsages.length; i++) {
    usagesAfterCurrent += (cardUsages[i].used_times || 1)
  }
  
  // 该记录的序号 = 当前总使用次数 - 之后的核销次数
  const usedTimes = currentUsedTotal - usagesAfterCurrent
  
  return { totalTimes, usedTimes }
}

// 获取当前次数的颜色类
const getUsageCountColorClass = (usage) => {
  if (!usage) return 'text-gray-700'
  
  const status = normalizeSessionStatus(usage.service_session_status) || usage.status
  
  // 服务中：绿色
  if (status === 'serving') {
    return 'text-green-600'
  }
  
  // 完成：默认色
  if (status === 'success' || status === 'finished') {
    return 'text-gray-700'
  }
  
  // 其他状态：红色
  return 'text-red-600'
}

const getSchedulerItemStatusText = (status) => {
  if (status === 'healthy') return '正常'
  if (status === 'stale') return '超时'
  if (status === 'missing') return '未上报'
  if (status === 'error') return '异常'
  return '未知'
}

const getSchedulerItemRowClass = (status) => {
  if (status === 'healthy') return 'border-green-100 bg-green-50/40'
  if (status === 'stale' || status === 'error') return 'border-red-100 bg-red-50/40'
  return 'border-gray-200 bg-gray-50/60'
}

const getSchedulerItemTextClass = (status) => {
  if (status === 'healthy') return 'text-green-700'
  if (status === 'stale' || status === 'error') return 'text-red-600'
  return 'text-gray-500'
}

const formatSchedulerTickAt = (value) => {
  if (!value) return '未上报'
  return formatDateTime(value)
}

const formatSchedulerAge = (seconds) => {
  const total = Number(seconds || 0)
  if (!Number.isFinite(total) || total <= 0) return '刚刚'
  const mins = Math.floor(total / 60)
  const secs = total % 60
  if (mins > 0) return `${mins}分${secs}秒前`
  return `${secs}秒前`
}

const fetchSchedulerHealth = async (silent = false) => {
  if (!silent) {
    schedulerHealthError.value = ''
  }
  try {
    const res = await merchantApi.getSchedulerHealth({})
    schedulerHealth.value = res.data?.data || null
    schedulerHealthLoaded.value = true
    schedulerHealthError.value = ''
  } catch (err) {
    if (!silent) {
      schedulerHealthError.value = err.response?.data?.error || '读取调度器健康状态失败'
    }
    schedulerHealthLoaded.value = true
  }
}

const startSchedulerHealthTimer = () => {
  if (schedulerHealthTimer) return
  schedulerHealthTimer = setInterval(() => {
    if (!merchantId.value) return
    fetchSchedulerHealth(true)
  }, 30000)
}

const stopSchedulerHealthTimer = () => {
  if (!schedulerHealthTimer) return
  clearInterval(schedulerHealthTimer)
  schedulerHealthTimer = null
}

const fetchMerchant = async () => {
  try {
    const res = await merchantApi.getMerchant(merchantId.value)
    merchant.value = res.data.data
  } catch (err) {
    console.error('获取商户信息失败:', err)
  }
}

const fetchCurrentTechnicianMe = async () => {
  if (!isTechnicianAuth()) return
  try {
    const res = await merchantApi.getCurrentTechnician()
    technicianMe.value = res.data?.data || null
  } catch (e) {
    technicianMe.value = null
  }
}

const loadCardTemplates = async () => {
  try {
    const res = await shopApi.getCardTemplates()
    const list = res.data.data || []
    cardTemplates.value = list.filter(t => t && (t.card_type === 'times' || t.card_type === 'lesson' || t.card_type === 'balance'))
    sellQrDataUrl.value = ''
  } catch (e) {
    console.error('加载卡片模板失败', e)
  }
}

const searchCards = async () => {
  displayMode.value = 'cards'
  await fetchIssuedCards()
}

const loadSellTemplates = async () => {
  console.log('loadSellTemplates 被调用, isTechnicianAuth():', isTechnicianAuth())
  displayMode.value = 'sellTemplates'  // 手动设置为售卡模式
  console.log('displayMode 设置为:', displayMode.value)
  console.log('currentDisplay 现在是:', currentDisplay.value)
  
  try {
    const res = await shopApi.getCardTemplates()
    console.log('API 响应:', res)
    sellTemplates.value = (res.data.data || []).filter(t => t && t.is_active)
    console.log('售卡模板数据:', sellTemplates.value)
  } catch (e) {
    console.error('加载售卡模板失败', e)
    if (e.response?.status === 403) {
      alert('您没有售卡权限，请联系管理员开通')
    } else {
      alert('加载售卡模板失败，请稍后重试')
    }
    sellTemplates.value = []
    displayMode.value = 'auto'  // 出错时重置为自动模式
  }
}

const openSellQrModal = async (tpl) => {
  if (!tpl || !tpl.id) return
  const techId = getTechnicianId()
  if (!techId) {
    alert('技师信息丢失')
    return
  }
  
  sellSelectedTemplate.value = tpl
  showSellQrModal.value = true
  
  // 等待DOM更新
  await nextTick()
  
  try {
    const url = `${window.location.origin}/shop?card_template_id=${tpl.id}&tech_id=${techId}`
    const canvas = sellQrCanvas.value
    if (canvas) {
      await QRCode.toCanvas(canvas, url, {
        width: 224,
        margin: 1,
        color: {
          dark: '#000000',
          light: '#FFFFFF'
        }
      })
    }
  } catch (e) {
    console.error('生成售卡二维码失败', e)
    alert('生成二维码失败')
  }
}

const closeSellQrModal = () => {
  showSellQrModal.value = false
  sellSelectedTemplate.value = null
}

// 售卡模板长按事件处理
const onTemplateTouchStart = (e, tpl) => {
  if (templateLongPressTimer.value) {
    clearTimeout(templateLongPressTimer.value)
    templateLongPressTimer.value = null
  }
  templatePressStart.value = null
  
  templateLongPressTimer.value = setTimeout(() => {
    openSellQrModal(tpl)
  }, 820)
}

const onTemplateTouchMove = (e) => {
  if (!templateLongPressTimer.value) return
  const t = e?.touches?.[0]
  if (!t) return
  if (!templatePressStart.value) {
    templatePressStart.value = { x: t.clientX, y: t.clientY }
    return
  }
  const dx = t.clientX - templatePressStart.value.x
  const dy = t.clientY - templatePressStart.value.y
  if (dx * dx + dy * dy > 12 * 12) {
    clearTimeout(templateLongPressTimer.value)
    templateLongPressTimer.value = null
  }
}

const onTemplateTouchEnd = () => {
  if (templateLongPressTimer.value) {
    clearTimeout(templateLongPressTimer.value)
    templateLongPressTimer.value = null
  }
}

const resetCardSearch = async () => {
  cardSearch.value = { card_no: '', card_type: '' }
  expandedCardId.value = null
  displayMode.value = 'auto'
  await fetchIssuedCards()
}

const clearUserCodeFilter = async () => {
  routeUserCode.value = ''
  scanUserCodeActive.value = false
  await router.replace({ path: '/merchant', query: { tab: 'cards' } })
  if (canVerify.value) {
    await fetchIssuedCards()
  }
}

const scrollToUserCodeHint = async () => {
  // 等待DOM更新，包括动态占位元素的渲染
  await nextTick()
  // 再次等待，确保占位元素高度计算完成
  await new Promise(resolve => setTimeout(resolve, 100))
  
  const el = userCodeAnchor.value
  if (!el) return
  try {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } catch (_) {
    // ignore
  }
}

const getBottomSpacerHeight = () => {
  // 当卡片数量少时，添加底部占位高度，确保可以滚动到锚点
  // 计算逻辑：窗口高度 - 已有内容的估算高度
  const windowHeight = window.innerHeight || 800
  const estimatedContentHeight = 600 // 头部 + Tab + 搜索框 + 提示框 + 1-2张卡片
  const minSpacerHeight = Math.max(windowHeight - estimatedContentHeight, 200)
  return `${minSpacerHeight}px`
}

const cleanupScanQuery = async () => {
  try {
    await router.replace({ path: '/merchant', query: { tab: 'cards' } })
  } catch (_) {
    // ignore
  }
}

const fetchQueueStatus = async () => {
  try {
    const res = await merchantApi.getQueueStatus(merchantId.value)
    pendingAppointments.value = res.data.data.pending_appointments || 0
  } catch (err) {
    console.error('获取队列状态失败:', err)
  }
}

const fetchPendingDirectPurchases = async () => {
  if (!canDirectSaleManage.value || !merchant.value.support_direct_sale) {
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
    appointmentTechnicianDirectory.value = []
    resetVisibleAppointmentGroups()
    if (currentTab.value === 'appointment' && !showAppointmentTab.value) {
      selectTab(getFirstVisibleTab())
    }
    return
  }
  try {
    const [appointmentsRes, techniciansRes] = await Promise.all([
      appointmentApi.getMerchantAppointments(merchantId.value),
      appointmentApi.getMerchantTechnicians(merchantId.value)
    ])
    appointments.value = (appointmentsRes.data.data || [])
      .filter(a => a.status !== 'canceled')
      .sort((a, b) => new Date(b?.appointment_time || 0).getTime() - new Date(a?.appointment_time || 0).getTime())
    appointmentTechnicianDirectory.value = techniciansRes.data?.data || []
    resetVisibleAppointmentGroups()
    if (currentTab.value === 'appointment' && !showAppointmentTab.value) {
      selectTab(getFirstVisibleTab())
    }
  } catch (err) {
    console.error('获取预约列表失败:', err)
  }
}

const fetchTodayUsages = async () => {
  try {
    const today = getLocalDateKey()
    const res = await usageApi.getMerchantUsages(merchantId.value, {
      date: today,
      limit: 200
    })
    todayUsages.value = (res.data.data || []).filter((u) => {
      return u.used_at && u.used_at.startsWith(today) && u.status !== 'failed'
    })
    todayVerifyCount.value = todayUsages.value.length
  } catch (err) {
    console.error('获取核销记录失败:', err)
  }
}

const fetchTodayStartUsages = async ({ silent = false } = {}) => {
  if (startUsagesRefreshing) return
  startUsagesRefreshing = true
  if (!silent) {
    startUsagesLoading.value = true
  }
  try {
    const today = getLocalDateKey()
    const currentTechnicianId = getTechnicianId()
    if (!currentTechnicianId) {
      // 非技师账号/无法获取技师ID：保持旧数据不闪烁，但结束 loading
      return
    }
    const res = await usageApi.getMerchantUsages(merchantId.value, {
      date: today,
      technician_id: currentTechnicianId,
      only_with_session: 1,
      limit: 50
    })
    // 后端已过滤，但这里仍做一次兜底，确保只显示“当前技师 + 今日 + 有会话”的记录
    const nextList = (res.data.data || []).filter((u) => {
      if (!u || !u.used_at || !u.used_at.startsWith(today)) return false
      const techId = u?.service_technician?.id || u?.technician_id
      if (Number(techId) !== Number(currentTechnicianId)) return false
      return !!u.service_session_status
    })
    patchTodayStartUsages(nextList)
  } catch (err) {
    console.error('获取上号记录失败:', err)
    // 失败时保留旧数据，避免“暂无”闪烁
  }
  finally {
    if (!silent) {
      startUsagesLoading.value = false
    }
    startUsagesRefreshing = false
  }
}

const fetchTodayFinishedUsages = async () => {
  try {
    const today = getLocalDateKey()
    const res = await usageApi.getMerchantUsages(merchantId.value)
    const currentTechnicianId = getTechnicianId()
    const isTechnician = isTechnicianAuth()
    
    // 只显示今天已结单的记录，且是当前用户的操作
    todayFinishedUsages.value = (res.data.data || []).filter(u => 
      u.finished_at && 
      u.finished_at.startsWith(today) &&
      u.status === 'success' &&
      // 过滤当前用户的记录
      (isTechnician ? 
        // 技师账号：只显示自己结单的记录
        (u.technician_id === currentTechnicianId) :
        // 商户老板号：只显示没有技师ID的记录（即老板操作的记录）
        (!u.technician_id)
      )
    )
  } catch (err) {
    console.error('获取结号记录失败:', err)
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
    if (cardSearch.value.card_no) params.card_no = cardSearch.value.card_no
    if (cardSearch.value.card_type) params.card_type = cardSearch.value.card_type
    if (routeUserCode.value) params.user_code = routeUserCode.value

    const res = await cardApi.getMerchantCards(merchantId.value, params)
    let cardsList = res.data.data || []
    
    // 过滤掉剩余次数为0的卡片
    cardsList = cardsList.filter(card => card.remain_times > 0)
    
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

const getAppointmentEffectiveStartAt = (appt) => {
  return appt?.reserved_start_at || appt?.appointment_time || null
}

const shouldUseCancelRequest = (appt) => {
  const startAt = getAppointmentEffectiveStartAt(appt)
  if (!startAt) return false
  const startMs = new Date(startAt).getTime()
  if (!Number.isFinite(startMs)) return false
  return Date.now() >= (startMs - 5 * 60 * 60 * 1000)
}

const getRepairDecisionText = (decision) => {
  if (decision === 'repairable') return '可优先修复'
  if (decision === 'cancel_only') return '需走取消分流'
  if (decision === 'not_affected') return '未命中异常'
  return '待判定'
}

const getRepairDecisionClass = (decision) => {
  if (decision === 'repairable') return 'bg-green-50 text-green-700'
  if (decision === 'cancel_only') return 'bg-orange-50 text-orange-700'
  return 'bg-gray-100 text-gray-600'
}

const maybeRedirectToMerchantRescheduleBeforeCancel = async (appt) => {
  if (!appt?.id) return false
  try {
    const res = await appointmentApi.getMerchantRescheduleEligibility(appt.id)
    const eligibility = res?.data?.data || null
    if (!eligibility?.allowed) return false
    const message = eligibility.default_date
      ? `当前预约仍可改签。是否先查看 ${eligibility.default_date} 的改签时段，再决定是否取消？`
      : '当前预约仍可改签。是否先查看改签时段，再决定是否取消？'
    if (!window.confirm(message)) return false
    await openAppointmentRescheduleModal(appt)
    return true
  } catch (_) {
    return false
  }
}

const cancelAppointment = async (appt) => {
  if (!appt?.id) return
  if (await maybeRedirectToMerchantRescheduleBeforeCancel(appt)) {
    return
  }
  if (!confirm('确定要取消这个预约吗？此操作不可撤销。')) {
    return
  }

  try {
    if (shouldUseCancelRequest(appt)) {
      const reason = window.prompt('距预约开始不足5小时，请填写取消申请原因')
      if (reason == null) return
      const trimmedReason = String(reason || '').trim()
      if (!trimmedReason) {
        alert('取消申请原因不能为空')
        return
      }
      await appointmentApi.createMerchantCancelRequest(appt.id, { reason: trimmedReason })
      alert('取消申请已提交，待用户确认')
    } else {
      const reason = window.prompt('请输入商户取消原因')
      if (reason == null) return
      const trimmedReason = String(reason || '').trim()
      if (!trimmedReason) {
        alert('商户取消原因不能为空')
        return
      }
      await appointmentApi.cancelMerchantAppointment(appt.id, { reason: trimmedReason })
      alert('预约已取消')
    }
    fetchAppointments()
    fetchQueueStatus()
  } catch (err) {
    if (isHandledAuthRedirectError(err)) return
    alert(err.response?.data?.error || '取消失败')
  }
}

const verifyCard = async () => {
  if (!verifyCodeInput.value || verifying.value) return
  
  verifying.value = true
  verifyResult.value = null
  
  try {
    const res = await cardApi.verifyCard(verifyCodeInput.value)
    const data = res?.data?.data || {}
    const extraMessages = []
    if (data.appointment_status === 'arrived') {
      if (Number(data.predicted_delay_minutes || 0) > 0) {
        extraMessages.push(`预约客户已到店，预计延迟 ${data.predicted_delay_minutes} 分钟`)
      } else {
        extraMessages.push('预约客户已到店，已进入服务闭环')
      }
    }
    verifyResult.value = {
      success: true,
      message: [`核销成功！剩余次数: ${data.remain_times}`, ...extraMessages].join('；')
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

const clearPendingVerifyCommitState = (clearRoute = true) => {
  pendingVerifyToken.value = ''
  handCardInput.value = ''
  handCardError.value = ''
  showHandCardModal.value = false
  try {
    sessionStorage.removeItem(PENDING_VERIFY_STORAGE_KEY)
  } catch (_) {
    // ignore
  }
  if (clearRoute && route.query.pending_verify_commit) {
    const nextQuery = { ...route.query }
    delete nextQuery.pending_verify_commit
    router.replace({ path: route.path, query: nextQuery })
  }
}

const openHandCardModalFromPendingVerify = () => {
  if (route.query.pending_verify_commit !== '1') return

  let payload = null
  try {
    payload = JSON.parse(sessionStorage.getItem(PENDING_VERIFY_STORAGE_KEY) || 'null')
  } catch (_) {
    payload = null
  }

  const verifyToken = String(payload?.verify_token || '').trim()
  if (!verifyToken) {
    clearPendingVerifyCommitState(true)
    return
  }

  pendingVerifyToken.value = verifyToken
  handCardInput.value = ''
  handCardError.value = ''
  showHandCardModal.value = true

  const nextQuery = { ...route.query }
  delete nextQuery.pending_verify_commit
  router.replace({ path: route.path, query: nextQuery })
}

const submitPendingVerifyHandCard = async (doBind) => {
  if (submittingHandCard.value) return
  handCardError.value = ''

  const verifyToken = String(pendingVerifyToken.value || '').trim()
  if (!verifyToken) {
    clearPendingVerifyCommitState(false)
    return
  }

  submittingHandCard.value = true
  try {
    const no = String(handCardInput.value || '').trim()
    if (doBind) {
      if (!no) {
        handCardError.value = '请输入手牌号'
        return
      }
    } else if (!confirm('确认本次核销跳过手牌分配吗？')) {
      return
    }

    const res = await cardApi.commitVerify(verifyToken, no, !doBind)
    clearPendingVerifyCommitState(false)
    fetchQueueStatus()
    fetchTodayUsages()
    alert(`核销成功！剩余次数: ${res?.data?.data?.remain_times ?? '-'}`)
  } catch (e) {
    handCardError.value = e?.response?.data?.error || '核销提交失败'
  } finally {
    submittingHandCard.value = false
  }
}

const closeHandCardModal = () => {
  if (!showHandCardModal.value) return
  if (!confirm('关闭后本次核销不会提交，确定关闭吗？')) return
  clearPendingVerifyCommitState(false)
}

const onHandCardMaskClick = () => {
  closeHandCardModal()
}

const cancelReturnHandCard = () => {
  returnHandCardNo.value = ''
  returnHandCardError.value = ''
  returnHandCardUsage.value = null
}

const queryHandCardForReturn = async () => {
  const no = String(returnHandCardNo.value || '').trim()
  if (!no || queryingReturnHandCard.value) return

  queryingReturnHandCard.value = true
  returnHandCardError.value = ''
  returnHandCardUsage.value = null
  try {
    const res = await cardApi.queryHandCardForReturn(no)
    returnHandCardUsage.value = res?.data?.data || null
    if (!returnHandCardUsage.value) {
      returnHandCardError.value = '未找到可归还的记录'
    }
  } catch (e) {
    returnHandCardError.value = e?.response?.data?.error || '查询失败'
  } finally {
    queryingReturnHandCard.value = false
  }
}

const confirmReturnHandCard = async () => {
  const no = String(returnHandCardNo.value || '').trim()
  if (!no || returningHandCard.value) return

  // 检查核销记录状态，如果不是已结单状态，需要二次确认
  const usage = returnHandCardUsage.value
  if (usage && usage.status !== 'success') {
    if (!confirm('服务尚未结束，是否提前归还手牌')) {
      return // 用户取消，不执行归还
    }
  }

  returningHandCard.value = true
  returnHandCardError.value = ''
  try {
    await cardApi.returnHandCard(no)
    alert('归还成功')
    cancelReturnHandCard()
    fetchQueueStatus()
    fetchTodayUsages()
    if (canVerify.value) {
      fetchIssuedCards()
    }
  } catch (e) {
    returnHandCardError.value = e?.response?.data?.error || '归还失败'
  } finally {
    returningHandCard.value = false
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

const confirmToggleBusinessStatus = async () => {
  try {
    const newStatus = !merchant.value.is_open
    await merchantApi.toggleBusinessStatus({ is_open: newStatus })
    merchant.value.is_open = newStatus
    showBusinessStatusModal.value = false
    alert(newStatus ? '已切换为营业中' : '已切换为打烊')
  } catch (err) {
    alert(err.response?.data?.error || '操作失败')
  }
}

const getStatusBadgeClass = (appt) => {
  if (!appt) return ''

  if (appt.status === 'pending' && isPendingExpired(appt)) {
    return 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }

  if (appt.status === 'confirmed' && isWriteOffExpired(appt)) {
    return 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }

  if (isHistoricalArrivedAppointment(appt)) {
    return 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-red-50 text-red-700'
  }

  const classes = {
    pending: 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-primary-light text-primary',
    confirmed: 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-700',
    arrived: 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-amber-50 text-amber-700',
    completed: 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-700',
    failed: 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500',
    no_show: 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500',
    canceled: 'shrink-0 whitespace-nowrap px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }
  return classes[appt.status] || ''
}

const getStatusText = (appt) => {
  if (!appt) return ''

  if (appt.status === 'pending' && !appt.technician) {
    return '待分配'
  }

  if (appt.status === 'pending' && isPendingExpired(appt)) {
    return '过期未确认'
  }

  if (appt.status === 'confirmed' && isWriteOffExpired(appt)) {
    return '已过服务时间'
  }

  if (isHistoricalArrivedAppointment(appt)) {
    return '超时待处理'
  }

  const texts = {
    pending: '待确认',
    confirmed: '待到店',
    arrived: '已到店',
    completed: '已完成',
    failed: '分配失败',
    no_show: '已失约',
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

  let deadlineMs = 0
  if (appt.reserved_end_at) {
    const reservedEndMs = new Date(appt.reserved_end_at).getTime()
    if (Number.isFinite(reservedEndMs) && reservedEndMs > 0) {
      const minServiceMinutes = Math.max(0, Math.floor(Number(appt.late_arrival_min_service_minutes || 0)))
      deadlineMs = reservedEndMs - minServiceMinutes * 60 * 1000
    }
  }
  if (!deadlineMs) {
    const appointmentTime = new Date(appt.appointment_time).getTime()
    deadlineMs = appointmentTime + 15 * 60 * 1000
  }
  return currentTime.value > deadlineMs
}

const isTechnicianVisibleUnassignedAppointment = (appt) => {
  if (!appt || appt.technician_id) return false
  if (isAppointmentSettled(appt)) return false
  const status = getAppointmentNormalizedStatus(appt)
  if (status === 'pending') {
    return !isPendingExpired(appt)
  }
  if (status === 'confirmed') {
    return !isWriteOffExpired(appt)
  }
  return status === 'arrived'
}

// 判断是否已过服务时间（不显示倒计时）
const isServiceTimeExpired = (appt) => {
  if (!appt || appt.status !== 'confirmed' || !appt.appointment_time) return false

  const appointmentTime = new Date(appt.appointment_time).getTime()
  const serviceMinutes = getAppointmentServiceMinutes(appt)
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

const formatAppointmentTechnicianDisplay = (appt) => {
  if (!appt || !appt.technician) return '待分配'
  const roleName = appt.technician?.service_role?.name || ''
  const account = appt.technician?.account || appt.technician?.code || ''
  const left = String(roleName || '').trim()
  const right = String(account || '').trim()
  const text = `${left} ${right}`.trim()
  return text || '待分配'
}

const getAppointmentTechnicianLineText = (appt) => {
  const account = String(appt?.technician?.account || appt?.technician?.code || '').trim()
  const name = String(appt?.technician?.name || '').trim()
  if (account && name) return `${account} - ${name}`
  return account || name || '待分配'
}

const showAppointmentTechnicianLine = (appt) => {
  if (isTechnicianAuth()) return false
  return !!String(getAppointmentTechnicianLineText(appt) || '').trim() && !!appt?.technician
}

const getAppointmentCardTypeDisplay = (appt) => {
  return String(appt?.card?.card_type || '').trim()
}

const getAppointmentIdDisplay = (appt) => {
  const id = Number(appt?.id || 0)
  return id > 0 ? `#${id}` : ''
}

const getAppointmentCardNoDisplay = (appt) => {
  return String(appt?.card?.card_no || '').trim()
}

const getAppointmentDisplayWaitState = (appt) => String(appt?.display_wait_state || '').trim()

const getAppointmentDisplayWaitMessage = (appt) => String(appt?.display_wait_message || '').trim()

const isCrossDayUnfinishedAppointment = (appt) => getAppointmentDisplayWaitState(appt) === 'cross_day_unfinished'

const getAppointmentNormalizedStatus = (appt) => {
  const value = String(appt?.status || '').trim()
  return value === 'finished' ? 'completed' : value
}

const isAppointmentSettled = (appt) => {
  const sessionStatus = normalizeSessionStatus(appt?.service_session?.status || appt?.session_status)
  if (sessionStatus === 'finished') return true
  if (appt?.completed_at) return true
  if (getAppointmentDisruptionReason(appt) === 'service_completed' && !hasAppointmentLiabilityIssue(appt)) return true
  return ['completed', 'failed', 'no_show', 'canceled'].includes(getAppointmentNormalizedStatus(appt))
}

const getAppointmentDisruptionReason = (appt) => String(appt?.disruption_reason || '').trim()

const getAppointmentLiabilityLevel = (appt) => String(appt?.liability_level || '').trim()

const isAppointmentDataCleanup = (appt) => getAppointmentDisruptionReason(appt) === 'appointment_state_inconsistent'

const hasAppointmentLiabilityIssue = (appt) => {
  const value = getAppointmentLiabilityLevel(appt)
  return !!value && value !== 'none'
}

const getAppointmentTimeMs = (appt) => {
  const raw = appt?.appointment_time
  if (!raw) return null
  const ts = new Date(raw).getTime()
  return Number.isFinite(ts) ? ts : null
}

const isSameCalendarDay = (leftMs, rightMs) => {
  const left = new Date(leftMs)
  const right = new Date(rightMs)
  return left.getFullYear() === right.getFullYear() &&
    left.getMonth() === right.getMonth() &&
    left.getDate() === right.getDate()
}

const isHistoricalArrivedAppointment = (appt) => {
  if (isCrossDayUnfinishedAppointment(appt)) return true
  if (!appt || appt.status !== 'arrived' || appt.actual_start_at) return false
  const appointmentTimeMs = getAppointmentTimeMs(appt)
  if (appointmentTimeMs === null) return false
  return !isSameCalendarDay(appointmentTimeMs, currentTime.value)
}

const hasAppointmentCompensationRecords = (appt) => {
  return Array.isArray(appt?.compensations) && appt.compensations.length > 0
}

const hasAppointmentRescheduleIssue = (appt) => {
  return !!getLatestPendingRescheduleRequest(appt)
}

const hasAppointmentForceMajeureIssue = (appt) => {
  return !!getLatestForceMajeureReliefRequest(appt)
}

const isPlainUserNoShowAppointment = (appt) => {
  if (!appt) return false
  if (getAppointmentNormalizedStatus(appt) !== 'no_show') return false
  if (getAppointmentDisruptionReason(appt) !== 'user_no_show') return false
  return getAppointmentLiabilityLevel(appt) === 'user'
}

const hasAppointmentDisruptionFlag = (appt) => {
  if (isPlainUserNoShowAppointment(appt)) return false
  const reason = getAppointmentDisruptionReason(appt)
  if (reason === 'service_completed') return hasAppointmentLiabilityIssue(appt)
  return !!reason || hasAppointmentLiabilityIssue(appt)
}

const hasAppointmentDisputeIssue = (appt) => {
  if (!appt) return false
  if (hasAppointmentForceMajeureIssue(appt)) return true
  if (hasAppointmentLiabilityIssue(appt) || hasAppointmentDisruptionFlag(appt)) {
    return !isHistoricalArrivedAppointment(appt)
  }
  return false
}

const needsAppointmentCompensationFollowUp = (appt) => {
  if (!appt || isAppointmentSettled(appt)) return false
  if (hasAppointmentCompensationRecords(appt)) return true
  return hasAppointmentDisruptionFlag(appt) && shouldShowAppointmentCompensation(appt)
}

const isExceptionAppointment = (appt) => {
  if (!appt) return false
  if (isHistoricalArrivedAppointment(appt)) return true
  if (isAppointmentDataCleanup(appt)) return true
  if (hasAppointmentRescheduleIssue(appt)) return true
  if (hasAppointmentForceMajeureIssue(appt)) return true
  if (hasAppointmentCompensationRecords(appt)) return true
  if (hasAppointmentDisruptionFlag(appt)) return true
  return false
}

const canOperateAppointment = (appt) => {
  if (canAppointmentManage.value && getMerchantActiveAuth() === 'merchant') {
    return true
  }
  if (!isTechnicianAuth()) return false
  const currentTechnicianId = getTechnicianId()
  if (!currentTechnicianId) return false
  if (!appt?.technician_id) {
    return !!canAppointmentManage.value
  }
  return Number(appt?.technician_id) === Number(currentTechnicianId)
}

const getAppointmentRiskHint = (appt) => {
  if (isAppointmentAlreadyInService(appt)) {
    return '客户已到店，已在服务中'
  }
  const displayMessage = getAppointmentDisplayWaitMessage(appt)
  if (displayMessage) {
    return displayMessage
  }
  return ''
}

const getAppointmentRiskHintClass = (appt) => {
  if (isHistoricalArrivedAppointment(appt)) {
    return 'bg-red-50 text-red-700 border border-red-100'
  }
  if (getAppointmentDisplayWaitState(appt) === 'active_waiting' || appt?.status === 'arrived') {
    return 'bg-amber-50 text-amber-700 border border-amber-100'
  }
  return 'bg-primary-light text-primary border border-primary/10'
}

const shouldShowAppointmentReschedule = (appt) => {
  if (!appt) return false
  if (isCrossDayUnfinishedAppointment(appt)) return false
  if (isAppointmentAlreadyInService(appt)) return false
  // 改签是重新安排预约，只有“还没彻底结束”的预约才允许改到新时间。
  return (appt.status === 'confirmed' || appt.status === 'arrived') && !getLatestPendingRescheduleRequest(appt)
}

const getLatestPendingRescheduleRequest = (appt) => {
  const list = Array.isArray(appt?.reschedule_requests) ? appt.reschedule_requests : []
  return list.find(item => item?.status === 'pending_user' || item?.status === 'pending_merchant') || null
}

const getLatestPendingCancelRequest = (appt) => {
  const list = Array.isArray(appt?.cancel_requests) ? appt.cancel_requests : []
  return list.find(item => item?.status === 'pending_user' || item?.status === 'pending_merchant') || null
}

const getLatestForceMajeureReliefRequest = (appt) => {
  const list = Array.isArray(appt?.force_majeure_relief_requests) ? appt.force_majeure_relief_requests : []
  if (list.length === 0) return null
  return [...list].sort((a, b) => Number(b?.id || 0) - Number(a?.id || 0))[0]
}

const getRescheduleRequestTechnicianText = (req) => {
  const technicianId = Number(req?.new_technician_id || 0)
  if (!technicianId) return ''
  const technician = (appointmentTechnicianDirectory.value || []).find(item => Number(item?.id || 0) === technicianId)
  if (!technician) return `改签客服：${technicianId}`
  const name = String(technician?.name || '').trim() || `改签客服${technicianId}`
  const account = String(technician?.account || '').trim()
  return account ? `改签客服：${name} - ${account}` : `改签客服：${name}`
}

const isMerchantConfirmationPending = (appt) => {
  const req = getLatestPendingRescheduleRequest(appt)
  return req?.status === 'pending_merchant'
}

const isMerchantCancelConfirmationPending = (appt) => {
  const req = getLatestPendingCancelRequest(appt)
  return req?.status === 'pending_merchant'
}

const canCancelAppointmentRescheduleRequest = (appt) => {
  const req = getLatestPendingRescheduleRequest(appt)
  if (!req) return false
  const proposer = String(req?.proposed_by_type || '').trim()
  return proposer === 'merchant' || proposer === 'staff'
}

const canCreateForceMajeureRelief = (appt) => {
  const latest = getLatestForceMajeureReliefRequest(appt)
  if (latest && String(latest.status || '').trim() === 'pending') return false
  const compensations = Array.isArray(appt?.compensations) ? appt.compensations : []
  return compensations.some(item => ['merchant_breach', 'merchant_failure_offset'].includes(String(item?.source_type || item?.reason || '').trim()))
}

const canAcceptForceMajeureRelief = (appt) => {
  const latest = getLatestForceMajeureReliefRequest(appt)
  if (!latest || String(latest.status || '').trim() !== 'pending') return false
  const proposer = String(latest.proposed_by_type || '').trim()
  return proposer !== 'merchant' && proposer !== 'staff'
}

const canRejectForceMajeureRelief = (appt) => canAcceptForceMajeureRelief(appt)

const shouldShowAppointmentCompensation = (appt) => {
  if (!appt) return false
  if (isAppointmentAlreadyInService(appt)) return false
  // 补偿不是常驻动作，只在门店承诺已经受损或服务异常结束后开放。
  if (appt.status === 'arrived') {
    return Number(appt?.predicted_delay_minutes || 0) > 0 || !!appt?.service_session_id
  }
  return appt.status === 'completed' || appt.status === 'failed'
}

const getCompensationTypeText = (type) => {
  if (type === 'extra_times') return '补次数'
  if (type === 'extend_minutes') return '补时长'
  if (type === 'manual_adjustment') return '补额度'
  if (type === 'discount_note') return '优惠减免'
  return '其他补偿'
}

const getCompensationValueText = (comp) => {
  const value = Number(comp?.value || 0)
  if (comp?.type === 'extra_times' && value > 0) return `增加 ${value} 次`
  if (comp?.type === 'extend_minutes' && value > 0) return `增加 ${value} 分钟`
  if (comp?.type === 'manual_adjustment' && value > 0) return `增加 ${value} 额度`
  if (comp?.remark) return comp.remark
  return ''
}

const getAppointmentSettlementStatusText = (status) => {
  const value = String(status || '').trim()
  if (!value) return ''
  const map = {
    pending: '待结算',
    settled: '已结算',
    frozen: '已冻结',
    refunded: '已退款',
    transferred: '已迁移',
    offset: '已对冲'
  }
  return map[value] || value
}

const getLiabilityText = (level) => {
  const value = String(level || '').trim()
  const map = {
    none: '无责任',
    user: '用户责任',
    merchant: '商户责任',
    pending_merchant: '商户待判定',
    merchant_exempt: '商户免责',
    technician_chargeable: '客服承担'
  }
  return map[value] || (value || '待判定')
}

const getReasonText = (reason) => {
  const value = String(reason || '').trim()
  const map = {
    merchant_breach: '商户违约补偿',
    merchant_failure_offset: '恢复性对冲',
    delay_bucket_redeem: '拖堂补偿兑现',
    service_unclosed_cross_day: '客户已到店但未开始服务，且跨日未完成结案',
    appointment_state_inconsistent: '预约状态与履约事实不一致',
    user_no_show: '用户未到店',
    risk_released: '风险解除',
    technician_leave: '客服请假',
    force_majeure_relief_accepted: '不可抗力救济已生效'
  }
  return map[value] || value
}

const getDelayLedgerStatusText = (ledger) => {
  const redeemStatus = String(ledger?.redeem_status || '').trim()
  const ledgerStatus = String(ledger?.ledger_status || '').trim()
  if (redeemStatus === 'redeemed' || ledgerStatus === 'redeemed') return '已兑现'
  if (redeemStatus === 'skipped' || ledgerStatus === 'ignored') return '已跳过'
  return '累计中'
}

const getForceMajeureStatusText = (status) => {
  const value = String(status || '').trim()
  const map = {
    pending: '待对方确认',
    accepted: '已确认生效',
    rejected: '已拒绝',
    canceled: '已撤销'
  }
  return map[value] || value
}

const getForceMajeureActorText = (actorType) => {
  const value = String(actorType || '').trim()
  if (value === 'user') return '用户'
  if (value === 'merchant') return '商户'
  if (value === 'staff') return '客服'
  return value || '-'
}

const getAppointmentReassignCandidates = async (appt) => {
  const res = await attendanceApi.listAvailableTechnicians()
  const currentTechnicianId = Number(appt?.technician_id || 0)
  return (res.data?.data || []).filter(item => {
    const technicianId = Number(item?.technician_id || item?.technician?.id || 0)
    if (!technicianId || technicianId === currentTechnicianId) return false
    const roleKey = String(item?.technician?.service_role?.key || '')
    return roleKey !== 'store_manager' && roleKey !== 'front_desk'
  })
}

const keepWaitingForAppointment = async (appt) => {
  await fetchAppointments()
  if (appt?.service_session_id) {
    await fetchServiceSessions()
  }
  alert('已保持原客服等待顺序，系统会在该客服释放后优先推进此预约')
}

const reassignAppointmentService = async (appt) => {
  if (!appt?.service_session_id) {
    alert('当前预约尚未生成服务会话，无法改派')
    return
  }
  try {
    const candidates = await getAppointmentReassignCandidates(appt)
    if (candidates.length === 0) {
      alert('当前没有其他空闲客服可改派')
      return
    }
    const promptText = candidates.map(item => {
      const technicianId = Number(item?.technician_id || item?.technician?.id || 0)
      const name = item?.technician?.account || item?.technician?.name || item?.technician?.code || `客服${technicianId}`
      const roleName = item?.technician?.service_role?.name || ''
      return `${technicianId}: ${roleName ? `${roleName} / ` : ''}${name}`
    }).join('\n')
    const nextTechnicianId = window.prompt(`请输入要改派的客服ID：\n${promptText}`)
    const parsedTechnicianId = Number(nextTechnicianId || 0)
    if (!parsedTechnicianId) return
    await serviceSessionApi.chooseTechnician(appt.service_session_id, { technician_id: parsedTechnicianId })
    alert('已改派客服，预约将按新的客服继续推进')
    await fetchAppointments()
    await fetchServiceSessions()
  } catch (err) {
    alert(err.response?.data?.error || '改派失败')
  }
}

const closeAppointmentException = async (appt) => {
  if (!isCrossDayUnfinishedAppointment(appt)) {
    alert('当前预约不属于跨日未闭环异常')
    return
  }
  const reason = window.prompt('请输入异常结案说明，例如：客户已到店但当日未开始服务，按商户履约异常结案')
  const trimmedReason = String(reason || '').trim()
  if (!trimmedReason) return
  try {
    await appointmentApi.closeMerchantAppointmentException(appt.id, { reason: trimmedReason })
    alert('已完成异常结案，预约已转入已结束')
    await fetchAppointments()
    await fetchServiceSessions()
  } catch (err) {
    alert(err.response?.data?.error || '异常结案失败')
  }
}

const resetAppointmentRescheduleForm = () => {
  appointmentRescheduleEligibility.value = null
  appointmentRescheduleTechnicians.value = []
  appointmentRescheduleForm.value = {
    date: '',
    technician_id: null,
    reason: ''
  }
  appointmentRescheduleSlots.value = []
  selectedAppointmentRescheduleTime.value = ''
}

const closeAppointmentRescheduleModal = () => {
  showAppointmentRescheduleModal.value = false
  rescheduleAppointmentTarget.value = null
  appointmentRescheduleLoading.value = false
  appointmentRescheduleSubmitting.value = false
  resetAppointmentRescheduleForm()
}

const buildAppointmentMinuteKey = (value) => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  // 这里按字符串取到分钟即可，避免 new Date() 受时区解析影响，导致原预约同一分钟没有被正确排除。
  const normalized = raw.replace('T', ' ').replace(/\.\d+$/, '').replace(/\//g, '-')
  const match = normalized.match(/(\d{4})-(\d{2})-(\d{2})\s+(\d{2}):(\d{2})/)
  if (match) {
    return `${match[1]}-${match[2]}-${match[3]} ${match[4]}:${match[5]}`
  }
  return normalized.slice(0, 16)
}

const getRescheduleSlotComparisonClass = (kind) => {
  if (kind === 'better') return 'text-green-600'
  if (kind === 'not_worse') return 'text-orange-500'
  return 'text-gray-400'
}

const fetchAppointmentRescheduleSlots = async () => {
  const appt = rescheduleAppointmentTarget.value
  if (!appt || !appointmentRescheduleForm.value.date) {
    appointmentRescheduleSlots.value = []
    appointmentRescheduleTechnicians.value = []
    return
  }
  appointmentRescheduleLoading.value = true
  try {
    const res = await appointmentApi.getMerchantRescheduleSlots(appt.id, appointmentRescheduleForm.value.date)
    appointmentRescheduleEligibility.value = res.data?.data?.eligibility || appointmentRescheduleEligibility.value
    appointmentRescheduleTechnicians.value = res.data?.data?.technicians || []
    const currentMinute = buildAppointmentMinuteKey(appt?.appointment_time)
    const rawSlots = (res.data?.data?.time_slots || []).filter(slot => {
      const slotMinute = buildAppointmentMinuteKey(slot?.time)
      return !currentMinute || slotMinute !== currentMinute
    })
    appointmentRescheduleSlots.value = rawSlots.map(slot => ({
      ...slot,
      label: slot?.time ? String(slot.time).slice(11, 16) : ''
    }))
    if (!appointmentRescheduleSlots.value.some(item => item.time === selectedAppointmentRescheduleTime.value)) {
      selectedAppointmentRescheduleTime.value = ''
      appointmentRescheduleForm.value.technician_id = null
    }
  } catch (err) {
    appointmentRescheduleSlots.value = []
    if (err.response?.data?.error) {
      alert(err.response.data.error)
      return
    }
    alert(err.response?.data?.error || '获取可改签时间失败')
  } finally {
    appointmentRescheduleLoading.value = false
  }
}

const openAppointmentRescheduleModal = async (appt) => {
  rescheduleAppointmentTarget.value = appt
  resetAppointmentRescheduleForm()
  try {
    const eligibilityRes = await appointmentApi.getMerchantRescheduleEligibility(appt.id)
    appointmentRescheduleEligibility.value = eligibilityRes.data?.data || null
    appointmentRescheduleForm.value.date = appointmentRescheduleEligibility.value?.default_date || ''
    appointmentRescheduleForm.value.reason = String(appt?.reschedule_reason || '').trim()
    showAppointmentRescheduleModal.value = true
    if (appointmentRescheduleEligibility.value?.allowed && appointmentRescheduleForm.value.date) {
      await fetchAppointmentRescheduleSlots()
    }
  } catch (err) {
    alert(err.response?.data?.error || '获取改签资格失败')
  }
}

const selectAppointmentRescheduleSlot = (slot) => {
  selectedAppointmentRescheduleTime.value = slot?.time || ''
  appointmentRescheduleForm.value.technician_id = null
}

const toggleAppointmentRescheduleTechnician = (technicianId) => {
  const next = Number(technicianId || 0)
  if (!next) return
  appointmentRescheduleForm.value.technician_id = appointmentRescheduleForm.value.technician_id === next ? null : next
}

const submitAppointmentReschedule = async () => {
  const appt = rescheduleAppointmentTarget.value
  if (!appt || !selectedAppointmentRescheduleTime.value) return
  appointmentRescheduleSubmitting.value = true
  try {
    const payload = {
      appointment_time: selectedAppointmentRescheduleTime.value,
      technician_id: appointmentRescheduleForm.value.technician_id ? Number(appointmentRescheduleForm.value.technician_id) : null,
      reason: String(appointmentRescheduleForm.value.reason || '').trim()
    }
    await appointmentApi.createMerchantRescheduleRequest(appt.id, payload)
    alert('改签提议已发送，等待用户确认')
    closeAppointmentRescheduleModal()
    await fetchAppointments()
  } catch (err) {
    alert(err.response?.data?.error || '改签失败')
  } finally {
    appointmentRescheduleSubmitting.value = false
  }
}

const acceptAppointmentRescheduleRequest = async (appt) => {
  const req = getLatestPendingRescheduleRequest(appt)
  if (!req) return
  try {
    await appointmentApi.acceptMerchantRescheduleRequest(appt.id, req.id)
    alert('已确认用户改签申请，系统已生成新预约')
    await fetchAppointments()
  } catch (err) {
    alert(err.response?.data?.error || '确认改签失败')
  }
}

const rejectAppointmentRescheduleRequest = async (appt) => {
  const req = getLatestPendingRescheduleRequest(appt)
  if (!req) return
  try {
    await appointmentApi.rejectMerchantRescheduleRequest(appt.id, req.id)
    alert('已拒绝该改签申请')
    await fetchAppointments()
  } catch (err) {
    alert(err.response?.data?.error || '拒绝改签失败')
  }
}

const acceptAppointmentCancelRequest = async (appt) => {
  const req = getLatestPendingCancelRequest(appt)
  if (!req) return
  try {
    await appointmentApi.acceptMerchantCancelRequest(appt.id, req.id)
    alert('已同意用户取消申请，预约已取消')
    await fetchAppointments()
  } catch (err) {
    if (isHandledAuthRedirectError(err)) return
    alert(err.response?.data?.error || '同意取消失败')
  }
}

const rejectAppointmentCancelRequest = async (appt) => {
  const req = getLatestPendingCancelRequest(appt)
  if (!req) return
  const note = window.prompt('请输入拒绝取消的说明', String(req?.objection_note || '').trim())
  if (note == null) return
  const trimmedNote = String(note || '').trim()
  if (!trimmedNote) {
    alert('拒绝说明不能为空')
    return
  }
  try {
    await appointmentApi.rejectMerchantCancelRequest(appt.id, req.id, { objection_note: trimmedNote })
    alert('已拒绝该取消申请')
    await fetchAppointments()
  } catch (err) {
    if (isHandledAuthRedirectError(err)) return
    alert(err.response?.data?.error || '拒绝取消失败')
  }
}

const cancelAppointmentRescheduleRequest = async (appt) => {
  const req = getLatestPendingRescheduleRequest(appt)
  if (!req) return
  try {
    await appointmentApi.cancelMerchantRescheduleRequest(appt.id, req.id)
    alert('已撤销改签提议')
    await fetchAppointments()
  } catch (err) {
    alert(err.response?.data?.error || '撤销改签失败')
  }
}

const closeAppointmentCompensationModal = () => {
  showAppointmentCompensationModal.value = false
  compensationAppointmentTarget.value = null
  appointmentCompensationSubmitting.value = false
  appointmentCompensationForm.value = {
    type: 'extra_times',
    value: 1,
    reason: '',
    remark: ''
  }
}

const openAppointmentCompensationModal = (appt) => {
  compensationAppointmentTarget.value = appt
  appointmentCompensationForm.value = {
    type: 'extra_times',
    value: 1,
    reason: '',
    remark: ''
  }
  showAppointmentCompensationModal.value = true
}

const closeAppointmentDetailModal = () => {
  showAppointmentDetailModal.value = false
  appointmentDetailTarget.value = null
  appointmentDetailLoading.value = false
  appointmentDetailError.value = ''
  appointmentDetailSettlement.value = null
  appointmentDetailDelayLedgers.value = []
  appointmentDetailSummary.value = null
  appointmentDetailHasRepairOverview.value = false
}

const closeTechnicianMonthlyDisruptionModal = () => {
  showTechnicianMonthlyDisruptionModal.value = false
  technicianMonthlyDisruptionTechnicianId.value = null
  technicianMonthlyDisruptionTechnicianName.value = ''
  technicianMonthlyDisruptionMonth.value = ''
  technicianMonthlyDisruptionLoading.value = false
  technicianMonthlyDisruptionError.value = ''
  technicianMonthlyDisruptionData.value = null
}

const closeAppointmentRepairOverviewModal = () => {
  showAppointmentRepairOverviewModal.value = false
  appointmentRepairOverviewLoading.value = false
  appointmentRepairOverviewError.value = ''
  appointmentRepairOverviewReason.value = ''
  appointmentRepairOverviewSchedule.value = null
  appointmentRepairAffectedAppointments.value = []
  appointmentRepairItems.value = []
  appointmentRepairProtectedSlots.value = []
}

const loadAppointmentDetailRepairOverviewVisibility = async (appointmentId) => {
  if (!appointmentId) {
    appointmentDetailHasRepairOverview.value = false
    return
  }
  try {
    const res = await appointmentApi.getMerchantAppointmentRepairOverview(appointmentId)
    if (Number(appointmentDetailTarget.value?.id || 0) !== Number(appointmentId)) return
    appointmentDetailHasRepairOverview.value = hasAppointmentRepairOverviewContent(res?.data?.data || {})
  } catch (err) {
    if (Number(appointmentDetailTarget.value?.id || 0) !== Number(appointmentId)) return
    appointmentDetailHasRepairOverview.value = false
  }
}

const openAppointmentDetailModal = async (appt) => {
  appointmentDetailTarget.value = appt
  appointmentDetailLoading.value = true
  appointmentDetailError.value = ''
  appointmentDetailSettlement.value = null
  appointmentDetailDelayLedgers.value = []
  appointmentDetailSummary.value = null
  appointmentDetailHasRepairOverview.value = false
  showAppointmentDetailModal.value = true
  loadAppointmentDetailRepairOverviewVisibility(appt.id)
  try {
    const [settlementRes, delayRes, summaryRes] = await Promise.all([
      appointmentApi.getMerchantSettlement(appt.id),
      appointmentApi.getMerchantDelayLedgers(appt.id),
      appointmentApi.getMerchantCompensationSummary(appt.id)
    ])
    appointmentDetailSettlement.value = settlementRes.data?.data || null
    appointmentDetailDelayLedgers.value = delayRes.data?.data || []
    appointmentDetailSummary.value = summaryRes.data?.data || null
  } catch (err) {
    appointmentDetailError.value = err.response?.data?.error || '读取预约结算详情失败'
  } finally {
    appointmentDetailLoading.value = false
  }
}

const loadTechnicianMonthlyDisruptions = async () => {
  if (!technicianMonthlyDisruptionTechnicianId.value || !technicianMonthlyDisruptionMonth.value) return
  technicianMonthlyDisruptionLoading.value = true
  technicianMonthlyDisruptionError.value = ''
  try {
    const res = await appointmentApi.getTechnicianMonthlyDisruptions(
      technicianMonthlyDisruptionTechnicianId.value,
      technicianMonthlyDisruptionMonth.value
    )
    technicianMonthlyDisruptionData.value = res?.data?.data || {}
  } catch (err) {
    technicianMonthlyDisruptionError.value = err.response?.data?.error || '读取月度异常统计失败'
    technicianMonthlyDisruptionData.value = null
  } finally {
    technicianMonthlyDisruptionLoading.value = false
  }
}

const reloadTechnicianMonthlyDisruptions = async () => {
  await loadTechnicianMonthlyDisruptions()
}

const getSchedulePublishingStatusText = (status) => {
  if (status === 'published') return '已发布'
  if (status === 'unpublished') return '待发布'
  if (status === 'leave') return '请假不可预约'
  if (status === 'canceled') return '待发布'
  return status || '未知'
}

const getSchedulePublishingStatusClass = (status) => {
  if (status === 'published') return 'bg-green-50 text-green-700'
  if (status === 'unpublished') return 'bg-blue-50 text-blue-700'
  if (status === 'leave') return 'bg-gray-100 text-gray-500'
  if (status === 'canceled') return 'bg-blue-50 text-blue-700'
  return 'bg-gray-100 text-gray-600'
}

const getSchedulePublishingRowKey = (row) => {
  if (row?.id) return `schedule-${row.id}`
  return `schedule-${row?.technician_id || 'merchant'}-${row?.start_at || ''}-${row?.end_at || ''}`
}

const formatScheduleTechnicianLabel = (row) => {
  const technicianId = Number(row?.technician_id || row?.technician?.id || 0)
  const fromRow = row?.technician || null
  const fromDirectory = (appointmentTechnicianDirectory.value || []).find(item => Number(item?.id || 0) === technicianId) || null
  const technician = fromRow || fromDirectory
  const account = String(technician?.account || technician?.code || '').trim()
  const name = String(technician?.name || row?.technician_name || '').trim()
  if (account && name) return `客服#${account} ${name}`
  if (account) return `客服#${account}`
  if (name) return `客服#${technicianId || '-'} ${name}`
  return `客服#${technicianId || '-'}`
}

const hasCanceledScheduleRows = computed(() => {
  return (schedulePublishings.value || []).some(row => row?.status === 'canceled')
})

const getEffectiveSchedulePublishingStatus = (row) => {
  const status = String(row?.status || '').trim()
  if (status === 'published' && hasCanceledScheduleRows.value) return 'canceled'
  return status
}

const getSchedulePublishingTargetDateFromRow = (row) => {
  if (row?.publish_date) {
    const parsed = parseScheduleDateValue(row.publish_date)
    if (parsed) return parsed
  }
  if (row?.start_at) {
    const parsed = new Date(row.start_at)
    if (!Number.isNaN(parsed.getTime())) {
      return new Date(parsed.getFullYear(), parsed.getMonth(), parsed.getDate(), 0, 0, 0, 0)
    }
  }
  return null
}

const isExpiredSchedulePublishingRow = (row) => {
  const status = getEffectiveSchedulePublishingStatus(row)
  if (!['unpublished', 'canceled'].includes(status)) return false
  const targetDate = getSchedulePublishingTargetDateFromRow(row)
  if (!targetDate) return false
  const windowState = getSchedulePublishingWindowStateForDate(targetDate, schedulePublishingNow.value)
  return status === 'canceled' ? !windowState.canWithdraw : !windowState.canPublish
}

const visibleSchedulePublishings = computed(() => {
  return (schedulePublishings.value || []).filter(row => !isExpiredSchedulePublishingRow(row))
})

const getPublishableScheduleRows = () => {
  return visibleSchedulePublishings.value.filter(row => ['unpublished', 'canceled'].includes(getEffectiveSchedulePublishingStatus(row)))
}

const hasPublishableScheduleRows = computed(() => {
  return getPublishableScheduleRows().length > 0
})

const shouldShowScheduleAffectedAppointments = (row) => {
  return Number(row?.affected_appointments_count || 0) > 0 || Number(row?.protected_repair_slots_count || 0) > 0
}

const hasPublishedScheduleRows = computed(() => {
  return visibleSchedulePublishings.value.some(row => getEffectiveSchedulePublishingStatus(row) === 'published')
})

const latestPublishedScheduleAt = computed(() => {
  let latest = null
  for (const row of visibleSchedulePublishings.value) {
    if (getEffectiveSchedulePublishingStatus(row) !== 'published') continue
    const raw = String(row?.published_at || '').trim()
    if (!raw) continue
    const dt = new Date(raw)
    if (Number.isNaN(dt.getTime())) continue
    if (!latest || dt.getTime() > latest.getTime()) {
      latest = dt
    }
  }
  return latest
})

const canWithdrawPublishedScheduleRows = computed(() => {
  if (isTechnicianAuth()) {
    return hasPublishedScheduleRows.value && technicianSchedulePublishingWindowState.value.canWithdraw
  }
  if (!hasPublishedScheduleRows.value) return false
  const latest = latestPublishedScheduleAt.value
  if (!latest) return false
  return Date.now() - latest.getTime() <= 30 * 60 * 1000
})

const schedulePublishingPrimaryAction = computed(() => {
  if (isTechnicianAuth()) {
    if (canWithdrawPublishedScheduleRows.value) {
      return {
        mode: 'withdraw',
        label: '撤销安排',
        submittingText: '撤销中...',
        disabled: false,
        className: 'bg-orange-500 text-white'
      }
    }
    if (hasPublishedScheduleRows.value) {
      return {
        mode: 'published',
        label: '已发布安排',
        submittingText: '处理中...',
        disabled: true,
        className: 'bg-green-500 text-white'
      }
    }
    const canPublish = technicianSchedulePublishingWindowState.value.canPublish
    const canRepublishCanceled = technicianSchedulePublishingWindowState.value.canWithdraw && hasCanceledScheduleRows.value
    const canPublishRows = hasPublishableScheduleRows.value
    return {
      mode: 'publish',
      label: '发布次日安排',
      submittingText: '发布中...',
      disabled: !(canPublish || canRepublishCanceled) || !canPublishRows,
      className: (canPublish || canRepublishCanceled) && canPublishRows ? 'bg-orange-500 text-white' : 'bg-gray-100 text-gray-400'
    }
  }
  if (canWithdrawPublishedScheduleRows.value) {
    return {
      mode: 'withdraw',
      label: '撤销发布',
      submittingText: '撤销中...',
      disabled: false,
      className: 'bg-orange-500 text-white'
    }
  }
  if (hasPublishedScheduleRows.value) {
    return {
      mode: 'published',
      label: '已发布预约',
      submittingText: '处理中...',
      disabled: true,
      className: 'bg-green-500 text-white'
    }
  }
  return {
    mode: 'publish',
    label: '发布预约排班',
    submittingText: '发布中...',
    disabled: !hasPublishableScheduleRows.value,
    className: hasPublishableScheduleRows.value ? 'bg-orange-500 text-white' : 'bg-gray-100 text-gray-400'
  }
})

const fetchSchedulePublishings = async () => {
  if (!schedulePublishingDate.value) return
  schedulePublishingLoading.value = true
  schedulePublishingError.value = ''
  try {
    const res = await attendanceApi.listSchedulePublishings(schedulePublishingDate.value)
    const data = res?.data?.data || {}
    schedulePublishings.value = Array.isArray(data.publishings) ? data.publishings : []
  } catch (err) {
    schedulePublishingError.value = err.response?.data?.error || '读取排班发布失败'
    schedulePublishings.value = []
  } finally {
    schedulePublishingLoading.value = false
  }
}

const syncSchedulePublishingTargetDate = async () => {
  const nextDate = getDefaultSchedulePublishingDate()
  if (schedulePublishingDate.value === nextDate) return
  schedulePublishingDate.value = nextDate
  await fetchSchedulePublishings()
}

const handleSchedulePublishingPrimaryAction = async () => {
  if (schedulePublishingPrimaryAction.value.mode === 'withdraw') {
    await withdrawNextDaySchedules()
    return
  }
  if (schedulePublishingPrimaryAction.value.mode === 'publish') {
    await publishNextDaySchedules()
  }
}

const publishNextDaySchedules = async () => {
  const publishableRows = getPublishableScheduleRows()
  if (publishableRows.length === 0) {
    alert(isTechnicianAuth() ? '当前没有待发布的预约排班' : '当前没有待发布的次日排班')
    return
  }
  const promptText = isTechnicianAuth()
    ? `确认发布 ${schedulePublishingDate.value} 的预约排班吗？\n\n发布后该日期的预约会按你当前排班对外开放。`
    : (() => {
        const preview = publishableRows.slice(0, 6).map(row => formatScheduleTechnicianLabel(row)).join('\n')
        const remain = publishableRows.length > 6 ? `\n等 ${publishableRows.length} 位客服` : ''
        return `确认正式发布 ${schedulePublishingDate.value} 的次日排班吗？\n\n本次将发布以下客服：\n${preview}${remain}\n\n已标记请假的客服不会被发布。`
      })()
  if (!window.confirm(promptText)) {
    return
  }
  schedulePublishingSubmitting.value = true
  try {
    await attendanceApi.publishNextDaySchedule(schedulePublishingDate.value)
    alert(isTechnicianAuth() ? '预约排班已发布' : '次日排班已发布')
    await fetchSchedulePublishings()
  } catch (err) {
    alert(err.response?.data?.error || '发布次日排班失败')
  } finally {
    schedulePublishingSubmitting.value = false
  }
}

const withdrawNextDaySchedules = async () => {
  if (!canWithdrawPublishedScheduleRows.value) {
    alert('当前没有已发布排班可撤销')
    return
  }
  const confirmText = isTechnicianAuth()
    ? `确认撤销 ${schedulePublishingDate.value} 的预约排班吗？\n\n撤销后会同时取消该日期已分配给你的预约。`
    : `确认撤销 ${schedulePublishingDate.value} 的已发布排班吗？\n\n仅支持发布后30分钟内撤销。\n撤销后会同时单方面取消该日期已预约用户的预约，已请假客服会保留请假状态。`
  if (!window.confirm(confirmText)) {
    return
  }
  schedulePublishingSubmitting.value = true
  try {
    const res = await attendanceApi.withdrawNextDaySchedule(schedulePublishingDate.value)
    const canceledAppointments = Number(res?.data?.data?.canceled_appointments || 0)
    alert(isTechnicianAuth()
      ? `已撤销该日期排班，并取消 ${canceledAppointments} 个分配给你的预约`
      : `已撤销该日期的已发布排班，并取消 ${canceledAppointments} 个已预约用户预约`)
    await fetchSchedulePublishings()
  } catch (err) {
    alert(err.response?.data?.error || '撤销发布失败')
  } finally {
    schedulePublishingSubmitting.value = false
  }
}

const fillAppointmentRepairOverview = (data) => {
  appointmentRepairOverviewSchedule.value = data.schedule || null
  appointmentRepairOverviewReason.value = String(data.reason || '').trim()
  appointmentRepairAffectedAppointments.value = Array.isArray(data.affected_appointments) ? data.affected_appointments : []
  appointmentRepairItems.value = Array.isArray(data.affected_appointment_repairs) ? data.affected_appointment_repairs : []
  appointmentRepairProtectedSlots.value = Array.isArray(data.protected_repair_slots) ? data.protected_repair_slots : []
}

const hasAppointmentRepairOverviewContent = (data) => {
  const affectedCount = Number(data?.affected_appointments_count || 0)
  const protectedCount = Number(data?.protected_repair_slots_count || 0)
  if (affectedCount > 0 || protectedCount > 0) return true
  const affectedAppointments = Array.isArray(data?.affected_appointments) ? data.affected_appointments : []
  const protectedRepairSlots = Array.isArray(data?.protected_repair_slots) ? data.protected_repair_slots : []
  return affectedAppointments.length > 0 || protectedRepairSlots.length > 0
}

const viewSchedulePublishingAffectedAppointments = async (schedule) => {
  if (!schedule?.id) return
  showAppointmentRepairOverviewModal.value = true
  appointmentRepairOverviewLoading.value = true
  appointmentRepairOverviewError.value = ''
  appointmentRepairOverviewReason.value = ''
  appointmentRepairOverviewSchedule.value = null
  appointmentRepairAffectedAppointments.value = []
  appointmentRepairItems.value = []
  appointmentRepairProtectedSlots.value = []
  try {
    const res = await attendanceApi.getScheduleAffectedAppointments(schedule.id)
    fillAppointmentRepairOverview(res?.data?.data || {})
  } catch (err) {
    appointmentRepairOverviewError.value = err.response?.data?.error || '读取受影响预约失败'
  } finally {
    appointmentRepairOverviewLoading.value = false
  }
}

const markScheduleLeave = async (schedule) => {
  if (schedulePublishingActionSubmitting.value) return
  if (!schedule?.technician_id) return
  const status = getEffectiveSchedulePublishingStatus(schedule)
  const confirmText = status === 'unpublished' || status === 'canceled'
    ? '确认给该客服请假吗？请假后会标记为“请假不可预约”，且不会进入后续预约排班发布。'
    : '确认将这条排班标记为请假吗？系统会立即扫描受影响预约并生成保护性改签建议。'
  if (!window.confirm(confirmText)) return
  schedulePublishingActionSubmitting.value = true
  try {
    let res = null
    if (status === 'published' && schedule?.id) {
      res = await attendanceApi.markScheduleLeave(schedule.id)
    } else {
      await attendanceApi.markScheduleLeaveByTechnician({
        date: schedulePublishingDate.value,
        technician_id: schedule.technician_id
      })
      alert('已标记请假，该客服不会进入次日正式发布')
    }
    await fetchSchedulePublishings()
    if (res) {
      const repairData = res?.data?.data || {}
      if (hasAppointmentRepairOverviewContent(repairData)) {
        alert('已标记请假并生成异常修复结果')
        showAppointmentRepairOverviewModal.value = true
        appointmentRepairOverviewLoading.value = false
        appointmentRepairOverviewError.value = ''
        fillAppointmentRepairOverview(repairData)
      } else {
        alert('已标记请假')
      }
    }
  } catch (err) {
    alert(err.response?.data?.error || '标记请假失败')
  } finally {
    schedulePublishingActionSubmitting.value = false
  }
}

const unmarkScheduleLeave = async (schedule) => {
  if (schedulePublishingActionSubmitting.value) return
  if (!schedule?.technician_id) return
  if (!window.confirm('确认销假吗？销假后该客服会恢复到可重新编排/可发布状态。')) return
  schedulePublishingActionSubmitting.value = true
  try {
    await attendanceApi.unmarkScheduleLeaveByTechnician({
      date: schedulePublishingDate.value,
      technician_id: schedule.technician_id
    })
    alert('已销假')
    await fetchSchedulePublishings()
  } catch (err) {
    alert(err.response?.data?.error || '销假失败')
  } finally {
    schedulePublishingActionSubmitting.value = false
  }
}

const viewTechnicianMonthlyDisruptions = async (appt) => {
  const technicianId = Number(appt?.technician_id || 0)
  if (!technicianId) return
  const now = new Date()
  technicianMonthlyDisruptionTechnicianId.value = technicianId
  technicianMonthlyDisruptionTechnicianName.value = String(appt?.technician?.name || '').trim() || `客服${technicianId}`
  technicianMonthlyDisruptionMonth.value = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  technicianMonthlyDisruptionData.value = null
  technicianMonthlyDisruptionError.value = ''
  showTechnicianMonthlyDisruptionModal.value = true
  await loadTechnicianMonthlyDisruptions()
}

const viewAppointmentRepairOverview = async (appt) => {
  if (!appt?.id) return
  showAppointmentRepairOverviewModal.value = true
  appointmentRepairOverviewLoading.value = true
  appointmentRepairOverviewError.value = ''
  appointmentRepairOverviewReason.value = ''
  appointmentRepairOverviewSchedule.value = null
  appointmentRepairAffectedAppointments.value = []
  appointmentRepairItems.value = []
  appointmentRepairProtectedSlots.value = []
  try {
    const res = await appointmentApi.getMerchantAppointmentRepairOverview(appt.id)
    fillAppointmentRepairOverview(res?.data?.data || {})
  } catch (err) {
    appointmentRepairOverviewError.value = err.response?.data?.error || '读取异常修复详情失败'
  } finally {
    appointmentRepairOverviewLoading.value = false
  }
}

const createMerchantForceMajeureRelief = async (appt) => {
  const reason = window.prompt('请输入不可抗力原因，例如：停电、突发设施故障')
  if (reason == null) return
  const trimmedReason = String(reason || '').trim()
  if (!trimmedReason) {
    alert('不可抗力原因不能为空')
    return
  }
  const evidence = window.prompt('可补充举证说明（可选）') || ''
  try {
    await appointmentApi.createMerchantForceMajeureRelief(appt.id, {
      reason: trimmedReason,
      evidence_note: String(evidence || '').trim()
    })
    alert('不可抗力申请已提交，待用户确认')
    await fetchAppointments()
  } catch (err) {
    alert(err.response?.data?.error || '提交不可抗力申请失败')
  }
}

const acceptAppointmentForceMajeureRelief = async (appt) => {
  const request = getLatestForceMajeureReliefRequest(appt)
  if (!request) return
  try {
    await appointmentApi.acceptForceMajeureRelief(request.id)
    alert('已确认不可抗力申请')
    await fetchAppointments()
  } catch (err) {
    alert(err.response?.data?.error || '确认不可抗力申请失败')
  }
}

const rejectAppointmentForceMajeureRelief = async (appt) => {
  const request = getLatestForceMajeureReliefRequest(appt)
  if (!request) return
  try {
    await appointmentApi.rejectForceMajeureRelief(request.id)
    alert('已拒绝不可抗力申请')
    await fetchAppointments()
  } catch (err) {
    alert(err.response?.data?.error || '拒绝不可抗力申请失败')
  }
}

const submitAppointmentCompensation = async () => {
  const appt = compensationAppointmentTarget.value
  if (!appt) return
  appointmentCompensationSubmitting.value = true
  try {
    await appointmentApi.createCompensation(appt.id, {
      type: appointmentCompensationForm.value.type,
      value: Number(appointmentCompensationForm.value.value || 0),
      reason: String(appointmentCompensationForm.value.reason || '').trim(),
      remark: String(appointmentCompensationForm.value.remark || '').trim()
    })
    alert('补偿已执行并记录')
    closeAppointmentCompensationModal()
    await fetchAppointments()
    if (appt?.service_session_id) {
      await fetchServiceSessions()
    }
  } catch (err) {
    alert(err.response?.data?.error || '补偿失败')
  } finally {
    appointmentCompensationSubmitting.value = false
  }
}

const getAppointmentProjectName = (appt) => {
  const name = appt?.project?.name || ''
  const trimmed = String(name || '').trim()
  return trimmed
}

const getAppointmentProjectDisplay = (appt) => {
  const name = appt?.project?.name || ''
  const duration = appt?.project?.duration
  const nameTrimmed = String(name || '').trim()
  if (!nameTrimmed) return ''
  if (duration && duration > 0) {
    return `${nameTrimmed}（${duration}分钟）`
  }
  return nameTrimmed
}

const getAppointmentServiceMinutes = (appt) => {
  const duration = Number(appt?.project?.duration || 0)
  return duration > 0 ? duration : 30
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
  if (countdown === null) return 'text-gray-500 text-sm font-medium mt-1'

  if (countdown <= 0) {
    return 'text-green-600 text-sm font-medium mt-1'
  }

  if (countdown <= 900) {
    return 'text-red-500 text-sm font-medium mt-1'
  }

  return 'text-gray-500 text-sm font-medium mt-1'
}

// 获取服务开始倒计时显示文本
const getServiceCountdownDisplay = (appt) => {
  const countdown = getAppointmentCountdown(appt)
  if (countdown === null) return ''
  
  if (countdown <= 0) {
    return '已开始'
  }
  
  const totalSeconds = countdown
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  
  if (hours > 0) {
    return `${hours}小时${minutes}分钟${seconds}秒`
  } else if (minutes > 0) {
    return `${minutes}分钟${seconds}秒`
  } else {
    return `${seconds}秒`
  }
}

// 获取服务开始倒计时颜色类（15分钟以内红色）
const getServiceCountdownClass = (appt) => {
  const countdown = getAppointmentCountdown(appt)
  if (countdown === null) return 'text-gray-500 text-sm font-medium mt-1'
  
  if (countdown <= 0) {
    return 'text-green-600 text-sm font-medium mt-1'
  }
  
  // 15分钟以内（900秒）用红色显示
  if (countdown <= 900) {
    return 'text-red-500 text-sm font-medium mt-1'
  }
  
  return 'text-gray-500 text-sm font-medium mt-1'
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

watch(currentTime, () => {
  syncServiceTabRefreshOnCountdownBoundary()
  syncStartTabRefreshOnCountdownBoundary()
  if (isTechnicianAuth()) {
    void syncSchedulePublishingTargetDate()
  }
})

watch(
  () => appointmentRescheduleForm.value.date,
  async (nextDate, prevDate) => {
    if (!showAppointmentRescheduleModal.value || !nextDate || nextDate === prevDate) return
    await fetchAppointmentRescheduleSlots()
  }
)

watch(currentTab, (tab) => {
  const normalizedTab = normalizeDashboardTab(tab)
  if (normalizedTab !== tab) {
    selectTab(normalizedTab)
    return
  }
  if (tab !== 'cards' && scanUserCodeActive.value) {
    scanUserCodeActive.value = false
    routeUserCode.value = ''
  }
  if (tab !== 'service') {
    stopServiceSessionTimer()
  }
  // 倒计时：appointment/exception/verify/service 以及带异常区块的看板需要每秒刷新 currentTime
  if (tab === 'appointment' || tab === 'exception' || tab === 'verify' || tab === 'service' || (tab === 'table' && showExceptionTab.value)) {
    startCountdownTimer()
  } else {
    stopCountdownTimer()
  }

  if (tab === 'appointment' || tab === 'exception' || (tab === 'table' && showExceptionTab.value)) {
    clearCountdownBoundaryState()
    fetchAppointments()
    return
  }
  if (tab === 'verify') {
    clearCountdownBoundaryState()
    // 重置为默认状态
    showVerifyInput.value = false
    verifyCodeInput.value = ''
    verifyResult.value = null
    fetchTodayUsages()
    return
  }
  if (tab === 'finish') {
    clearCountdownBoundaryState()
    // 结单Tab显示今日结单记录
    fetchTodayFinishedUsages()
    return
  }
  if (tab === 'cards') {
    clearCountdownBoundaryState()
    // 重置显示模式为自动，让computed决定显示什么
    displayMode.value = 'auto'
    // 如果默认显示售卡模板，则加载售卡模板数据
    if (currentDisplay.value === 'sellTemplates') {
      loadSellTemplates()
    } else if (canVerify.value) {
      fetchIssuedCards()
    }
    return
  }
  if (tab === 'notice') {
    clearCountdownBoundaryState()
    fetchNotices()
    return
  }
  if (tab === 'service') {
    clearCountdownBoundaryState()
    refreshServiceTabPartialData({ silent: false, force: true })
    startServiceSessionTimer()
    return
  }
  clearCountdownBoundaryState()
})

const startServiceSessionTimer = () => {
  stopServiceSessionTimer()
  serviceSessionTimer = setInterval(() => {
    if (currentTab.value === 'service') {
      refreshServiceTabPartialData({ silent: true })
      return
    }
  }, DATA_POLL_INTERVAL_MS)
}

const stopServiceSessionTimer = () => {
  if (serviceSessionTimer) {
    clearInterval(serviceSessionTimer)
    serviceSessionTimer = null
  }
}

watch(
  () => route.query.user_code,
  async (v) => {
    if (v) {
      routeUserCode.value = String(v)
      scanUserCodeActive.value = String(route.query.from_scan || '') === '1'
      if (currentTab.value === 'cards') {
        if (canVerify.value) {
          await fetchIssuedCards()
        }
        if (scanUserCodeActive.value) {
          await scrollToUserCodeHint()
          await cleanupScanQuery()
        }
      }
      return
    }

    if (!scanUserCodeActive.value) {
      routeUserCode.value = ''
      if (currentTab.value === 'cards') {
        if (canVerify.value) {
          await fetchIssuedCards()
        }
      }
    }
  }
)

onMounted(async () => {
  console.log('Merchant Dashboard mounted')
  console.log('localStorage merchantId:', localStorage.getItem('merchantId'))

  // 尝试从 localStorage 恢复上次选择的 tab
  try {
    const savedTab = localStorage.getItem(DASHBOARD_ACTIVE_TAB_STORAGE_KEY)
    if (savedTab && ['verify', 'appointment', 'exception', 'start', 'finish', 'notice', 'cards', 'table', 'service'].includes(savedTab)) {
      const normalizedSavedTab = normalizeDashboardTab(savedTab)
      selectTab(normalizedSavedTab)
      console.log('从 localStorage 恢复 tab:', normalizedSavedTab)
    }
  } catch (e) {
    // ignore
  }

  // 等待权限加载完成
  await ensureMerchantPermissionsLoaded()
  console.log('Permissions loaded, checking permissions:', {
    canAppointmentView: canAppointmentView.value,
    canAppointmentManage: canAppointmentManage.value,
    canVerify: canVerify.value,
    canFinishVerify: canFinishVerify.value
  })
  
  // 检查查询参数，自动切换到指定Tab（优先级高于 localStorage）
  const tabParam = route.query.tab
  if (tabParam && ['verify', 'appointment', 'exception', 'start', 'finish', 'notice', 'cards', 'table', 'service'].includes(tabParam)) {
    selectTab(normalizeDashboardTab(tabParam))
  }

  // 检查错误参数，显示错误弹窗
  const errorParam = route.query.error
  if (errorParam) {
    showErrorModalWithMessage(String(errorParam))
    // 清除URL中的错误参数，避免刷新时重复显示
    router.replace({ path: '/merchant', query: { tab: tabParam || 'appointment' } })
  }

  const userCodeParam = route.query.user_code
  routeUserCode.value = userCodeParam ? String(userCodeParam) : ''
  scanUserCodeActive.value = !!userCodeParam && String(route.query.from_scan || '') === '1'
  
  const storedMerchantId = getMerchantId()
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
  await fetchMerchant()
  console.log('Merchant loaded:', merchant.value)
  schedulePublishingDate.value = getDefaultSchedulePublishingDate()
  fetchSchedulePublishings()

  await fetchCurrentTechnicianMe()
  nextTick(() => {
    openHandCardModalFromPendingVerify()
  })

  // 根据权限选择默认Tab
  if (!tabParam) {
    // 如果已经从 localStorage 恢复了 tab，并且该 tab 有权限显示，则保持不变
    const restoredTab = currentTab.value
      if (restoredTab && ['verify', 'appointment', 'exception', 'start', 'finish', 'notice', 'cards', 'board', 'table', 'service'].includes(restoredTab)) {
        // 检查恢复的 tab 是否有权限显示
      const canShowRestoredTab = 
        (restoredTab === 'appointment' && showAppointmentTab.value) ||
        (restoredTab === 'exception' && showExceptionStandaloneTab.value) ||
        (restoredTab === 'verify' && showVerifyTab.value) ||
        (restoredTab === 'finish' && showFinishTab.value) ||
        (restoredTab === 'notice' && showNoticeTab.value) ||
        (restoredTab === 'cards' && showCardsTab.value) ||
        (restoredTab === 'board' && showTechnicianBoardTab.value) ||
        (restoredTab === 'table' && showTableTab.value) ||
        (restoredTab === 'service' && showServiceTab.value)
      
      if (canShowRestoredTab) {
        // 保持恢复的 tab，不需要改变（也不需要重新保存到 localStorage）
        console.log('保持从 localStorage 恢复的 tab:', restoredTab)
      } else {
        // 恢复的 tab 没有权限，设置默认 tab
        console.log('恢复的 tab 没有权限，设置默认 tab')
        currentTab.value = getDefaultTab()
        // 重新保存新的默认 tab
        try {
          localStorage.setItem(DASHBOARD_ACTIVE_TAB_STORAGE_KEY, currentTab.value)
        } catch (e) {
          // ignore
        }
      }
    } else {
      // 没有恢复有效的 tab，设置默认 tab
      console.log('没有恢复有效的 tab，设置默认 tab')
      currentTab.value = getDefaultTab()
      // 保存默认 tab
      try {
        localStorage.setItem(DASHBOARD_ACTIVE_TAB_STORAGE_KEY, currentTab.value)
      } catch (e) {
        // ignore
      }
    }
    } else {
      const tabVisibleMap = {
      verify: showVerifyTab.value,
      appointment: showAppointmentTab.value,
      exception: showExceptionStandaloneTab.value,
      finish: showFinishTab.value,
      notice: showNoticeTab.value,
      cards: showCardsTab.value,
      board: showTechnicianBoardTab.value,
      table: showTableTab.value,
      service: showServiceTab.value
    }
    if (!tabVisibleMap[currentTab.value]) {
      selectTab(getFirstVisibleTab())
    }
  }

  fetchQueueStatus()
  fetchPendingDirectPurchases()
  fetchAppointments()
  if (showVerifyTab.value) {
    fetchTodayUsages()
  }
  loadCardTemplates() // 加载卡片模板
  
  // 如果是技师登录，获取当前签到状态
  if (isTechnicianAuth()) {
    await fetchCurrentAttendanceStatus()
  }

  // 获取叫号状态（有权限时）
  await fetchQueueCallingStatus()
  
  // 根据最终的 currentTab 加载对应的数据
  if (currentTab.value === 'appointment' || currentTab.value === 'exception' || (currentTab.value === 'table' && showExceptionTab.value)) {
    fetchAppointments()
    startCountdownTimer()
  } else if (currentTab.value === 'verify') {
    fetchTodayUsages()
    startCountdownTimer()
  } else if (currentTab.value === 'finish') {
    fetchTodayFinishedUsages()
  } else if (currentTab.value === 'cards') {
    // 重置显示模式为自动，让computed决定显示什么
    displayMode.value = 'auto'
    // 如果默认显示售卡模板，则加载售卡模板数据
    if (currentDisplay.value === 'sellTemplates') {
      loadSellTemplates()
    } else if (canVerify.value) {
      fetchIssuedCards()
    }
  } else if (currentTab.value === 'notice') {
    fetchNotices()
  } else if (currentTab.value === 'service') {
    refreshServiceTabPartialData({ silent: false, force: true })
    startCountdownTimer()
    startServiceSessionTimer()
  }
  if (currentTab.value !== 'service' && showServiceTab.value) {
    fetchServiceSessions()
  }
})

const copyText = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    alert('已复制')
  } catch (e) {
    alert('复制失败')
  }
}

const maybeReassignPendingBeforeLeave = async (actionText) => {
  if (!isTechnicianAuth()) return true
  if (!supportsPendingStartReassignBeforeLeave()) return true
  const sess = pendingStartSession.value
  if (!sess || !sess.id) return true
  const confirmed = confirm(`你当前还有1个${getPendingStartLeavePromptLabel()}用户，是否在${actionText}前先尝试转交给其他空闲客服？`)
  if (!confirmed) return true
  const ok = await reassignQueuePendingItem({ session_id: sess.id }, `专业客服在${actionText}前发起自助转交`)
  if (!ok) return false
  return true
}

const doCheckIn = async () => {
  if (!isTechnicianAuth()) return
  attendanceLoading.value = true
  try {
    const res = await attendanceApi.checkIn({})
    const msg = res?.data?.message
    alert(msg || '签到成功')
    // 重新获取最新状态
    await fetchCurrentAttendanceStatus()
  } catch (e) {
    alert(e.response?.data?.error || '签到失败')
  } finally {
    attendanceLoading.value = false
  }
}

const doCheckOut = async () => {
  if (!isTechnicianAuth()) return
  if (hasActiveServingSession.value) {
    alert('当前服务完成后，才可以下班签到')
    return
  }
  
  // 添加确认弹窗
  const confirmed = confirm('确认要下班签到吗？')
  if (!confirmed) return
  const canContinue = await maybeReassignPendingBeforeLeave('下班')
  if (!canContinue) return
  
  attendanceLoading.value = true
  try {
    const res = await attendanceApi.checkOut({})
    // 下班签到成功后，同步服务器状态
    applyAttendanceStatusFromServer('rest')
    attendanceStatusDirty.value = false
    attendanceStatusConfirming.value = false
    const msg = res?.data?.message
    alert(msg || '下班签到成功')
  } catch (e) {
    alert(e.response?.data?.error || '下班签到失败')
  } finally {
    attendanceLoading.value = false
  }
}

const updateAttendanceStatus = async () => {
  if (!isTechnicianAuth()) return
  await submitAttendanceStatusChange(attendanceStatus.value)
}

// 设置结号后自动暂停
const setNextStatusPaused = async () => {
  if (!isTechnicianAuth()) return
  setNextPausedLoading.value = true
  try {
    await attendanceApi.updateStatus({ status: 'paused' })
    applyAttendanceStatusFromServer('paused')
    attendanceStatusDirty.value = false
    alert(replaceTerms('已设置：结单后自动暂停', merchant.value))
  } catch (e) {
    alert(e.response?.data?.error || '设置失败')
  } finally {
    setNextPausedLoading.value = false
  }
}

const getSessionStatusText = (status) => {
  const statusTextMap = {
    room_selecting: '选房中',
    room_locked: '房间已锁定',
    staff_selecting: '选人中',
    start_pending: getMerchantPendingStartLabel(),
    delay_pending: getMerchantPendingStartLabel({ queueMode: true }),
    timeout_waiting: '过号等待',
    timeout_failed: '过号失败',
    serving: '进行中',
    auto_finishing: getMerchantAutoFinishLabel(),
    finished: '已完成',
    canceled: '已取消'
  }
  return statusTextMap[status] || status || '-'
}

const fetchServiceSessions = async () => {
  sessionLoading.value = true
  try {
    const params = {}
    if (sessionStatusFilter.value) params.status = sessionStatusFilter.value
    const res = await serviceSessionApi.listSessions(params)
    serviceSessions.value = res.data?.data || []
    syncContinueCallBlockedFromPendingSession()
  } catch (e) {
    serviceSessions.value = []
  } finally {
    sessionLoading.value = false
  }
}

const fetchCurrentAttendanceStatus = async () => {
  if (!isTechnicianAuth()) return
  console.log('fetchCurrentAttendanceStatus: 开始获取技师状态')
  try {
    const res = await attendanceApi.getCurrentStatus()
    const attendance = res.data?.data
    console.log('fetchCurrentAttendanceStatus: 接口返回', attendance)
    if (attendance && attendance.status) {
      applyAttendanceStatusFromServer(attendance.status)
      console.log('从服务器恢复技师状态:', attendance.status)
    } else {
      applyAttendanceStatusFromServer('not_checked_in')
      console.log('接口返回空或无status，置为未签到')
    }
  } catch (e) {
    applyAttendanceStatusFromServer('not_checked_in')
    console.error('获取技师状态失败，置为未签到:', e)
  }
}

const clearServiceTabBoundaryState = () => {
  serviceTabBoundaryState = new Map()
}

const clearStartTabBoundaryState = () => {
  startTabBoundaryState = new Map()
}

const clearCountdownBoundaryState = () => {
  clearServiceTabBoundaryState()
  clearStartTabBoundaryState()
}

const collectServiceTabCountdowns = () => {
  const items = []
  const pushItem = (key, seconds) => {
    if (!key || seconds === null || seconds === undefined) return
    const remain = Number(seconds)
    if (!Number.isFinite(remain)) return
    items.push({ key, remain })
  }

  for (const session of serviceSessions.value || []) {
    const sessionId = Number(session?.id || 0)
    if (!sessionId) continue
    pushItem(`session:start_pending:${sessionId}`, getStartPendingRemainingSeconds(session))
    pushItem(`session:serving:${sessionId}`, getSessionRemainingSeconds(session))
  }

  for (const item of queuePendingList.value || []) {
    const usageId = Number(item?.usage_id || 0)
    const sessionId = Number(item?.session_id || 0)
    const baseId = sessionId || usageId
    if (!baseId) continue
    pushItem(`queue_pending:start_pending:${baseId}`, getQueueWaitingItemStartPendingRemainingSeconds(item))
    pushItem(`queue_pending:serving:${baseId}`, getQueueServingItemRemainingSeconds(item))
  }

  const queueSession = queueCallInfo.value?.session
  if (queueSession) {
    const sessionId = Number(queueSession?.id || queueSession?.session_id || queueCallInfo.value?.tracking_id || 0)
    if (sessionId) {
      pushItem(`queue_call:start_pending:${sessionId}`, getQueueCallSessionStartPendingRemainingSeconds())
      pushItem(`queue_call:serving:${sessionId}`, getQueueCallSessionRemainingSeconds())
    }
  }

  return items
}

const collectStartTabCountdowns = () => {
  const items = []
  for (const usage of todayStartUsages.value || []) {
    const usageId = Number(usage?.id || 0)
    if (!usageId) continue
    const servingRemain = getUsageServiceRemainingSeconds(usage)
    if (servingRemain !== null && servingRemain !== undefined) {
      const normalizedRemain = Number(servingRemain)
      if (Number.isFinite(normalizedRemain)) {
        items.push({ key: `start_usage:serving:${usageId}`, remain: normalizedRemain })
      }
    }
    const startRemain = getUsageStartPendingCountdownSeconds(usage)
    if (startRemain !== null && startRemain !== undefined) {
      const normalizedRemain = Number(startRemain)
      if (Number.isFinite(normalizedRemain)) {
        items.push({ key: `start_usage:start_pending:${usageId}`, remain: normalizedRemain })
      }
    }
  }
  return items
}

const refreshServiceTabPartialData = async ({ silent = true, force = false } = {}) => {
  if (currentTab.value !== 'service') return

  const now = Date.now()
  if (!force && serviceTabRefreshing.value) {
    serviceTabRefreshQueued.value = true
    return
  }
  if (!force && now - lastServiceTabRefreshAt < REFRESH_GUARD_INTERVAL_MS) {
    return
  }

  serviceTabRefreshing.value = true
  lastServiceTabRefreshAt = now
  try {
    await Promise.allSettled([
      fetchCurrentTechnicianMe(),
      fetchCurrentAttendanceStatus(),
      fetchQueueCallingStatus(),
      fetchServiceSessions(),
      fetchQueuePendingList(silent),
      fetchQueueTimeoutWaitingList(silent),
      fetchQueueCallInfo(),
      fetchTodayStartUsages({ silent }),
      fetchTodayUsages(),
      fetchQueueStatus()
    ])
  } finally {
    serviceTabRefreshing.value = false
    if (serviceTabRefreshQueued.value) {
      serviceTabRefreshQueued.value = false
      refreshServiceTabPartialData({ silent: true, force: true })
    }
  }
}

const syncServiceTabRefreshOnCountdownBoundary = () => {
  if (currentTab.value !== 'service') {
    clearServiceTabBoundaryState()
    return
  }

  const nextState = new Map()
  let shouldRefresh = false
  for (const item of collectServiceTabCountdowns()) {
    const remain = Math.max(0, Math.floor(Number(item.remain || 0)))
    nextState.set(item.key, remain)
    const prevRemain = serviceTabBoundaryState.get(item.key)
    if (prevRemain === undefined) continue
    if (prevRemain > 0 && remain <= 0) {
      shouldRefresh = true
    }
  }
  serviceTabBoundaryState = nextState

  if (shouldRefresh) {
    refreshServiceTabPartialData({ silent: true, force: true })
  }
}

const syncStartTabRefreshOnCountdownBoundary = () => {
  if (currentTab.value !== 'start') {
    clearStartTabBoundaryState()
    return
  }

  const nextState = new Map()
  let shouldRefresh = false
  for (const item of collectStartTabCountdowns()) {
    const remain = Math.max(0, Math.floor(Number(item.remain || 0)))
    nextState.set(item.key, remain)
    const prevRemain = startTabBoundaryState.get(item.key)
    if (prevRemain === undefined) continue
    if (prevRemain > 0 && remain <= 0) {
      shouldRefresh = true
    }
  }
  startTabBoundaryState = nextState

  if (shouldRefresh) {
    fetchTodayStartUsages({ silent: true })
  }
}

onBeforeRouteLeave(() => {
  scanUserCodeActive.value = false
  routeUserCode.value = ''
})

onUnmounted(() => {
  stopCountdownTimer()
  stopServiceSessionTimer()
  stopContinueCallBlockedTimer()
  clearCountdownBoundaryState()
  serviceTabRefreshQueued.value = false
  scanUserCodeActive.value = false
  routeUserCode.value = ''
  if (errorTimer) {
    clearTimeout(errorTimer)
    errorTimer = null
  }
})

onActivated(() => {
  if (!merchantId.value) return
  fetchMerchant()
  
  // 根据当前Tab刷新对应数据
  if (currentTab.value === 'verify') {
    fetchTodayUsages()
  } else if (currentTab.value === 'finish') {
    fetchTodayFinishedUsages()
  } else if (currentTab.value === 'service') {
    refreshServiceTabPartialData({ silent: false, force: true })
    startCountdownTimer()
    startServiceSessionTimer()
  }
  if (currentTab.value !== 'service' && showServiceTab.value) {
    fetchServiceSessions()
  }
  if (currentTab.value !== 'verify' && showVerifyTab.value) {
    fetchTodayUsages()
  }
})

watch(
  () => route.query.pending_verify_commit,
  () => {
    nextTick(() => {
      openHandCardModalFromPendingVerify()
    })
  }
)

const isUsageHandCardReturned = (usage) => {
  return Boolean(usage?.hand_card_returned_at)
}

const getHandCardStatusText = (usage) => {
  if (!usage) return '未分配'
  if (isUsageHandCardReturned(usage)) return '已归还'
  if (!usage.hand_card_no) return '未分配'
  if (usage.hand_card_assigned_at) return '已分配'
  return '未分配'
}

const parseLockedReasonHandCards = (reason) => {
  const r = String(reason || '').trim()
  if (!r) return { count: 0, list: [] }

  // 优先解析冒号后的手牌列表（支持中文冒号/英文冒号）
  let listPart = ''
  const idxCN = r.indexOf('：')
  const idxEN = r.indexOf(':')
  const idx = idxCN >= 0 ? idxCN : idxEN
  if (idx >= 0 && idx + 1 < r.length) {
    listPart = r.slice(idx + 1)
  }
  const list = (listPart || '')
    .split(',')
    .map(s => String(s || '').trim())
    .filter(Boolean)

  if (list.length > 0) return { count: list.length, list }

  // 兜底：解析“你有N个未归还手牌”中的 N
  const m = r.match(/你有\s*(\d+)\s*个未归还手牌/)
  if (m && m[1]) {
    const n = parseInt(m[1], 10)
    if (Number.isFinite(n) && n >= 0) return { count: n, list: [] }
  }
  return { count: 0, list: [] }
}

const getLockedHandCardTipText = (card) => {
  const info = parseLockedReasonHandCards(card?.locked_reason)
  const n = info.count
  const listText = info.list.length > 0 ? info.list.join(',') : ''
  const prefix = `尚未归还${Number.isFinite(n) ? n : 0}个手牌`
  if (listText) return `${prefix}：${listText}`
  return prefix
}
</script>

<style scoped>
/* 售卡模板样式 */
.template-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.template-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.template-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--kb-surface);
  border-radius: 12px;
  box-shadow: var(--kb-shadow);
  cursor: pointer;
  transition: transform 0.2s;
}

.template-card:hover {
  transform: translateY(-2px);
}

.template-card:active {
  transform: scale(0.98);
}

.template-info {
  flex: 1;
}

.template-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--kb-text);
}

.template-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.type-tag {
  padding: 2px 8px;
  background: var(--kb-primary-soft);
  color: var(--kb-primary-dark);
  border-radius: 4px;
  font-size: 12px;
}

.price {
  font-size: 16px;
  font-weight: 600;
  color: var(--kb-text);
}

.template-detail {
  font-size: 12px;
  color: var(--kb-text-muted);
}
</style>
