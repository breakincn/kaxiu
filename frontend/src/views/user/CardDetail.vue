<template>
  <div class="min-h-screen bg-gray-50 pb-6">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">{{ card.merchant?.name || '卡片详情' }}</span>
    </header>

    <!-- 卡片详情与营业时间 -->
    <div ref="usagesAnchor" class="px-4 mt-4">
      <div ref="cardSummarySection" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          <span class="font-medium text-gray-800">卡片详情</span>
        </div>
        <div class="space-y-3.5">
          <div class="flex justify-between">
            <span class="text-gray-500">卡号</span>
            <span class="text-gray-800">{{ card.card_no }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">卡类型</span>
            <span class="text-gray-800">{{ card.card_type }}</span>
          </div>
          <div v-if="card.projects && card.projects.length > 0" class="flex justify-between">
            <span class="text-gray-500">包含项目</span>
            <span class="text-gray-800 text-right max-w-[70%]">
              {{ card.projects.map(p => p.name).join('、') }}
            </span>
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
          <div v-if="getMerchantAddress()" class="flex justify-between">
            <span class="text-gray-500">地址</span>
            <span class="text-gray-800 text-right max-w-[70%]">{{ getMerchantAddress() }}</span>
          </div>
        </div>

        <div v-if="card?.locked" class="mt-4 p-3 rounded-lg bg-red-50 border border-red-100">
          <div class="text-red-600 font-medium">卡片已锁定</div>
          <div class="text-red-500 text-sm mt-1">{{ card.locked_reason || '请联系商户处理' }}</div>
        </div>

        <div v-if="getMerchantBusinessHours()" class="pt-4 mt-4 border-t border-gray-100">
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <svg :class="isMerchantOpen() ? 'text-green-500' : 'text-red-500'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
              <span class="font-medium text-gray-800">{{ isMerchantOpen() ? '营业时间' : '暂停营业' }}</span>
            </div>
            <div v-if="isMerchantOpen()" class="text-sm leading-relaxed text-gray-500 text-right" v-html="getMerchantBusinessHours()"></div>
            <span v-if="!isMerchantOpen()" class="bg-red-500 text-white text-sm font-medium px-3 py-1 rounded">打烊</span>
          </div>
        </div>

        <div class="live-service-section pt-4 mt-4 border-t border-gray-100">
          <div class="live-service-card" :class="`is-${getLiveServiceStatusLevel(liveServiceStatus)}`">
            <div class="live-service-header">
              <div class="min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <svg class="w-5 h-5 text-blue-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                  </svg>
                  <span class="font-medium text-gray-800">门店实时服务状态</span>
                </div>
                <div class="live-service-title">
                  {{ liveServiceStatus?.status_text || '正在读取门店实时状态' }}
                </div>
                <div class="live-service-summary">
                  {{ formatLiveServiceSummary(liveServiceStatus) || liveServiceError || '根据当前服务进度进行估算' }}
                </div>
              </div>
              <button class="live-service-refresh" :disabled="liveServiceLoading" @click="refreshLiveServiceStatus">
                {{ liveServiceLoading ? '刷新中' : '刷新' }}
              </button>
            </div>

            <div v-if="liveServiceStatus" class="mt-3">
              <div class="live-service-metrics">
                <div>
                  <span>待服务</span>
                  <strong>{{ formatLiveServiceCount(liveServiceStatus.counts?.waiting_count) }}</strong>
                </div>
                <div>
                  <span>服务中</span>
                  <strong>{{ formatLiveServiceCount(liveServiceStatus.counts?.serving_count) }}</strong>
                </div>
                <div>
                  <span>空闲客服</span>
                  <strong>{{ formatLiveServiceCount(liveServiceStatus.counts?.idle_staff_count) }}</strong>
                </div>
              </div>

              <div v-if="liveServiceStatus.queue" class="live-service-inline">
                <span>当前叫到 {{ formatLiveQueueCurrentNo(liveServiceStatus) }}</span>
                <span>待叫号 {{ liveServiceStatus.queue.waiting_queue_count || 0 }} 人</span>
              </div>

              <div class="live-service-estimate" :class="getLiveServiceEstimateClass(liveServiceStatus)">
                <div>
                  <span v-if="shouldShowLiveServiceWaitLabel(liveServiceStatus)">预计等待</span>
                  <strong>{{ formatLiveServiceWait(liveServiceStatus) }}</strong>
                  <p>{{ formatLiveServiceRecommendation(liveServiceStatus) }}</p>
                </div>
              </div>

              <div class="live-service-footer">
                <span>{{ liveServiceStatus.confidence_text || '结果仅供参考' }}</span>
                <span v-if="liveServiceStatus.generated_at">{{ formatLiveServiceGeneratedAt(liveServiceStatus.generated_at) }}</span>
              </div>
            </div>

            <div v-else class="live-service-empty">
              {{ liveServiceError || '正在读取门店实时状态...' }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 预约状态展示：仅保留查看，不再在详情页提供入口操作 -->
    <div v-if="card.merchant?.support_appointment && !card?.locked && appointment && shouldShowAppointmentStatusCard" class="px-4 mt-4">
      <div ref="appointmentAnchor" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center gap-2 mb-3">
          <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <span class="font-medium text-gray-800">预约状态</span>
        </div>

        <div class="space-y-3">
          <div class="flex justify-between items-center">
            <div>
              <span class="text-gray-500">我的预约</span>
              <span class="ml-2 text-gray-600 text-sm">
                {{ appointment.technician ? ((appointment.technician.service_role?.name || '客服') + '：' + appointment.technician.name) : '待分配' }}
              </span>
              <div v-if="appointment.id" class="text-gray-400 text-sm mt-1">预约号：#{{ appointment.id }}</div>
            </div>
            <span :class="getAppointmentStatusClass(appointment)">
              {{ getAppointmentStatusText(appointment) }}
            </span>
          </div>
          <div class="flex justify-between items-center">
            <div :class="getAppointmentTimeClass()" class="font-medium text-lg">
              <div v-if="getAppointmentProjectDisplay(appointment)" class="text-gray-500 text-sm font-normal mb-1">
                预约项目: {{ getAppointmentProjectDisplay(appointment) }}
              </div>
              {{ formatDateTime(appointment.appointment_time) }}
            </div>
            <div class="text-right">
              <div v-if="shouldShowAppointmentCountdownLabel" class="text-sm text-gray-600 mb-1">{{ getAppointmentCountdownLabel() }}</div>
              <div v-if="shouldShowAppointmentCountdownValue" :class="getCountdownClass()" class="text-sm font-medium">
                {{ getCountdownText() }}
              </div>
              <div v-else-if="shouldShowAppointmentPassedText" class="text-sm text-gray-400">
                预约已过
              </div>
            </div>
          </div>
          <div v-if="appointmentDelayMetricVisible" class="pt-3 mt-3 border-t border-gray-100">
            <div class="text-gray-400 text-xs">{{ appointmentDelayLabel }}</div>
            <div class="text-2xl font-bold text-gray-800">{{ appointmentDelayMinutes }}<span class="text-sm font-normal">分钟</span></div>
          </div>
          <div
            v-if="appointmentWaitingHint"
            class="rounded-lg px-3 py-3 text-sm border"
            :class="isHistoricalArrivedAppointment(appointment) ? 'bg-red-50 text-red-700 border-red-100' : 'bg-amber-50 text-amber-700 border-amber-100'"
          >
            {{ appointmentWaitingHint }}
          </div>
          <div v-if="appointment.reserved_start_at || appointment.reserved_end_at || appointment.cancel_deadline_at" class="grid grid-cols-1 gap-2 pt-3 mt-3 border-t border-gray-100 text-sm text-gray-600">
            <div v-if="appointment.reserved_start_at">锁定开始：{{ formatDateTime(appointment.reserved_start_at) }}</div>
            <div v-if="appointment.reserved_end_at">锁定结束：{{ formatDateTime(appointment.reserved_end_at) }}</div>
            <div v-if="appointment.cancel_deadline_at">最晚可直接取消：{{ formatDateTime(appointment.cancel_deadline_at) }}</div>
          </div>
          <div
            v-if="latestAppointmentRescheduleRequest"
            class="rounded-lg px-3 py-3 text-sm border"
            :class="'bg-orange-50 text-orange-700 border-orange-100'"
          >
            <div class="font-medium">
              {{ isUserRescheduleConfirmationPending ? '商户发起了改签提议，请确认' : '存在待处理的改签提议' }}
            </div>
            <div class="mt-1">提议时间：{{ formatDateTime(latestAppointmentRescheduleRequest.new_appointment_time) }}</div>
            <div v-if="latestAppointmentRescheduleTechnicianText" class="mt-1">改签客服：{{ latestAppointmentRescheduleTechnicianText }}</div>
            <div v-if="latestAppointmentRescheduleRequest.reason" class="mt-1">原因：{{ latestAppointmentRescheduleRequest.reason }}</div>
          </div>
          <div
            v-if="latestAppointmentCancelRequest"
            class="rounded-lg px-3 py-3 text-sm border"
            :class="isUserCancelConfirmationPending ? 'bg-red-50 text-red-700 border-red-100' : (isMerchantReviewingUserCancelRequest ? 'bg-orange-50 text-orange-700 border-orange-100' : 'bg-gray-50 text-gray-700 border-gray-200')"
          >
            <div class="font-medium">
              {{ isUserCancelConfirmationPending ? '商户发起了取消申请，请确认' : (isMerchantReviewingUserCancelRequest ? '你已发起取消申请，待商户判定' : '取消申请记录') }}
            </div>
            <div v-if="latestAppointmentCancelRequest.reason" class="mt-1">取消原因：{{ latestAppointmentCancelRequest.reason }}</div>
            <div v-if="latestAppointmentCancelRequest.objection_note" class="mt-1">抗辩说明：{{ latestAppointmentCancelRequest.objection_note }}</div>
            <div v-if="latestAppointmentCancelRequest.created_at" class="mt-1">申请时间：{{ formatDateTime(latestAppointmentCancelRequest.created_at) }}</div>
          </div>
          <div v-if="appointmentRescheduleRecommendationVisible" class="rounded-xl border border-blue-200 bg-blue-50 px-4 py-4 space-y-3">
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium text-blue-900">系统优化性改签推荐</div>
              <div class="text-xs text-blue-500">仅推荐，不自动改签</div>
            </div>
            <div v-if="appointmentRescheduleRecommendationReason" class="text-sm text-blue-700">
              {{ appointmentRescheduleRecommendationReason }}
            </div>
            <div v-if="appointmentRescheduleRecommendations.length > 0" class="grid grid-cols-2 gap-3">
              <div v-for="slot in appointmentRescheduleRecommendations" :key="slot.time" class="rounded-lg bg-white px-3 py-3 border border-blue-100 text-sm text-gray-700">
                <div class="font-medium text-gray-800">{{ formatDateTime(slot.time) }}</div>
                <div v-if="slot.comparison_label" class="mt-1 text-blue-600">{{ slot.comparison_label }}</div>
                <div v-if="slot.recommendation_reason" class="mt-1 text-xs text-gray-500">{{ slot.recommendation_reason }}</div>
              </div>
            </div>
          </div>
          <p class="text-xs text-gray-400">* 预约履约状态会随商户服务推进及时更新</p>
          <div v-if="showAppointmentSettlementButton" class="mt-3">
            <button
              type="button"
              @click="openAppointmentSettlementModal"
              class="px-4 py-2 rounded-lg bg-gray-100 text-gray-700 text-sm font-medium hover:bg-gray-200 transition-colors"
            >
              结算与补偿
            </button>
          </div>
          <div class="space-y-2 mt-3">
            <button
              v-if="showAppointmentArrivalVerifyButton"
              @click="openAppointmentArrivalVerifyFlow"
              class="w-full py-2.5 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark transition-colors"
            >
              出示预约签到码
            </button>
            <button
              v-if="showUserRescheduleAction"
              @click="openUserRescheduleModal"
              class="w-full py-2.5 border-2 border-primary text-primary font-medium rounded-lg hover:bg-primary-light transition-colors"
            >
              申请改签
            </button>
            <button
              v-if="isUserRescheduleConfirmationPending"
              @click="acceptUserRescheduleRequest"
              class="w-full py-2.5 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark transition-colors"
            >
              同意改签
            </button>
            <button
              v-if="isUserRescheduleConfirmationPending"
              @click="rejectUserRescheduleRequest"
              class="w-full py-2.5 border-2 border-gray-300 text-gray-700 font-medium rounded-lg hover:bg-gray-50 transition-colors"
            >
              拒绝改签
            </button>
            <button
              v-if="canCancelUserRescheduleRequest"
              @click="cancelUserRescheduleRequest"
              class="w-full py-2.5 border-2 border-gray-300 text-gray-700 font-medium rounded-lg hover:bg-gray-50 transition-colors"
            >
              撤销改签
            </button>
            <button
              v-if="showCancelAppointmentAction"
              @click="cancelAppointment"
              :disabled="cancelButtonDisabled"
              class="w-full py-2.5 border-2 border-red-400 text-red-500 font-medium rounded-lg hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {{ cancelButtonText }}
            </button>
            <button
              v-if="isUserCancelConfirmationPending"
              @click="acceptUserCancelRequest"
              class="w-full py-2.5 bg-red-500 text-white font-medium rounded-lg hover:bg-red-600 transition-colors"
            >
              同意取消
            </button>
            <button
              v-if="isUserCancelConfirmationPending"
              @click="rejectUserCancelRequest"
              class="w-full py-2.5 border-2 border-gray-300 text-gray-700 font-medium rounded-lg hover:bg-gray-50 transition-colors"
            >
              拒绝取消
            </button>
            <button
              v-if="canSubmitUserRebuttal"
              @click="submitUserRebuttal"
              class="w-full py-2.5 border-2 border-orange-300 text-orange-600 font-medium rounded-lg hover:bg-orange-50 transition-colors"
            >
              {{ appointment?.user_rebuttal_note ? '修改抗辩意见' : '提交抗辩意见' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 使用记录 -->
    <div class="px-4 mt-4">
      <div ref="usageRecordsSection" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div
          class="flex items-center justify-between"
          :class="usageRecordsCollapsed ? 'cursor-pointer mb-0' : 'mb-4'"
          @click="handleUsageHeaderClick"
        >
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="p-1 -m-1 text-gray-600 rounded hover:bg-gray-50"
              @click.stop="toggleUsageRecordsCollapsed"
              :aria-label="usageRecordsCollapsed ? '展开使用记录' : '折叠使用记录'"
            >
              <svg v-if="!usageRecordsCollapsed" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6M9 8h6m2 13H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
              </svg>
              <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7h16M4 12h10M4 17h16"/>
              </svg>
            </button>
            <span class="font-medium text-gray-800">使用记录</span>
            <span v-if="usageRecordsCollapsed" class="ml-2 text-sm font-medium text-orange-500">已折叠</span>
          </div>
          <span class="text-gray-600 text-sm">
            总数{{ card.total_times }}次/剩余{{ card.remain_times }}次
          </span>
        </div>
        <div v-if="!usageRecordsCollapsed && displayUsages.length > 0">
          <div
            v-for="(usage, index) in visibleUsages"
            :key="usage.id"
            class="grid grid-cols-[1fr_auto] items-start px-1 py-3"
            :class="index > 0 ? 'border-t border-gray-100' : ''"
            @touchstart="(e) => onUsageTouchStart(e, usage)"
            @touchmove="onUsageTouchMove"
            @touchend="onUsageTouchEnd"
            @touchcancel="onUsageTouchEnd"
            @contextmenu.prevent
            style="-webkit-touch-callout: none;"
          >
            <div class="min-w-0">
              <!-- 已分配手牌显示 -->
              <div v-if="card?.merchant?.support_hand_card && usage.hand_card_no && !isUsageHandCardReturned(usage)" class="text-red-500 font-medium mb-2">
                {{ usage.status === 'success' ? '未归还手牌' : '已分配手牌' }}：<span class="text-lg font-bold">{{ usage.hand_card_no }}</span>
              </div>
              <div class="text-gray-800">核销次数：{{ card.total_times }} / <span :class="getUsageCurrentTimesClass(usage, index)">{{ getUsageSequence(index) }}</span></div>
              <div v-if="getUsageAppointmentNumber(usage)" class="text-gray-400 text-sm mt-0.5">
                预约号：#{{ getUsageAppointmentNumber(usage) }}
              </div>
              <div class="text-gray-400 text-sm mt-0.5">
                单号：{{ getUsageTrackingNumber(usage) }}
              </div>
              <div v-if="getUsageSyntheticStatusText(usage)" class="text-gray-400 text-sm mt-0.5">
                状态：{{ getUsageSyntheticStatusText(usage) }}
              </div>
              <div v-if="getUsageQueueDisplayText(usage)" class="text-sm mt-0.5 font-medium">
                叫号：<span :class="getUsageQueueNoClass(usage)">{{ getUsageQueueDisplayText(usage) }}</span>
              </div>
              <div v-if="getUsageQueueOperatorInfo(usage)" class="col-span-2 text-gray-400 text-sm mt-0.5 whitespace-nowrap">
                叫号人员：{{ getUsageQueueOperatorInfo(usage) }}
              </div>
              <div v-if="getUsageQueueCountdownText(usage)" class="text-xs mt-0.5 font-mono" :class="getUsageQueueCountdownClass(usage)">
                {{ getUsageQueueCountdownText(usage) }}
              </div>
              <div v-if="isUsageSessionFinishedButUsageInProgress(usage) && getUsageServiceEndAtTextForFinishedSession(usage)" class="text-gray-400 text-sm mt-0.5">
                服务结束：{{ getUsageServiceEndAtTextForFinishedSession(usage) }}
              </div>
              <div v-else-if="usage?.status === 'in_progress' && shouldShowServiceStartTime(usage) && getUsageServiceStartAtText(usage)" class="text-gray-400 text-sm mt-0.5">
                服务开始：{{ getUsageServiceStartAtText(usage) }}
              </div>
              <div v-if="shouldShowServiceRemainTime(usage) && getUsageServiceRemainText(usage)" class="text-gray-400 text-sm mt-0.5 font-mono">
                剩余时间：{{ getUsageServiceRemainText(usage) }}
              </div>
              <div v-if="usage?.status === 'success' && usage?.finished_at" class="text-gray-400 text-sm mt-0.5">
                服务结束：{{ formatDateTime(usage.finished_at) }}
              </div>
              <div v-else-if="getUsageSyntheticServiceEndText(usage)" class="text-gray-400 text-sm mt-0.5">
                服务结束：{{ getUsageSyntheticServiceEndText(usage) }}
              </div>
              <div v-if="getUsageRoomInfo(usage)" class="text-gray-400 text-sm mt-0.5">
                {{ getUsageRoomInfo(usage) }}
              </div>
              <div v-if="getUsageWindowInfo(usage)" class="text-gray-400 text-sm mt-0.5">
                {{ getUsageWindowInfo(usage) }}
              </div>
              <div v-if="getUsageProjectText(usage)" class="text-gray-400 text-sm mt-0.5">
                {{ getUsageProjectText(usage) }}
              </div>
              <div v-if="getUsageApprovedExtendInfoLines(usage).length > 0" class="w-full text-green-700 text-sm mt-1 rounded-lg bg-green-50 px-2 py-1 leading-6">
                <div v-for="line in getUsageApprovedExtendInfoLines(usage)" :key="line" class="whitespace-nowrap">{{ line }}</div>
              </div>
              <div v-if="getUsageFailureReasonText(usage)" class="text-gray-400 text-sm mt-0.5">
                原因：{{ getUsageFailureReasonText(usage) }}
              </div>
              <div v-if="card?.merchant?.support_hand_card && !(usage.hand_card_no && !isUsageHandCardReturned(usage))" class="text-gray-400 text-sm mt-0.5">
                手牌：{{ usage.hand_card_no || '-' }}（<span v-if="!usage.hand_card_no && !isUsageHandCardReturned(usage)" :class="isUsageHandCardPendingAssignment(usage) ? 'text-orange-500' : 'text-red-500'">{{ isUsageHandCardPendingAssignment(usage) ? '待分配' : '未分配' }}</span><span v-else>{{ getHandCardStatusText(usage) }}</span>）
              </div>
            </div>
            <div class="text-right flex-shrink-0 ml-3">
              <div v-if="isUsageSessionFinishedButUsageInProgress(usage)" class="text-sm font-medium text-gray-800">
                完成
              </div>
              <div v-if="!isUsageSessionFinishedButUsageInProgress(usage)" :class="getUsageStatusClass(usage)" class="text-sm font-medium">
                {{ getUsageStatusText(usage) }}
              </div>
              <button
                v-if="usage?.can_revoke"
                class="mt-1 px-2 py-1 text-xs border border-red-400 text-red-500 rounded disabled:opacity-50"
                :disabled="revokeLoading"
                @click.stop="doRevokeUsage(usage)"
              >
                {{ revokeLoading ? '撤销中...' : '撤销核销' }}
              </button>
              <div v-if="getUsageStatusCountdownText(usage)" class="text-xs mt-0.5 font-mono" :class="getUsageStatusCountdownClass(usage)">
                {{ getUsageStatusCountdownText(usage) }}
              </div>
              <button
                v-if="shouldShowUsageExtendButton(usage)"
                class="mt-2 px-2 py-1 text-xs rounded font-medium disabled:opacity-60"
                :class="getUsageExtendButtonClass(usage)"
                :disabled="isUsageExtendButtonDisabled(usage) || extendRequestSubmitting"
                @click.stop="handleUsageExtendClick(usage)"
              >
                {{ getUsageExtendButtonText(usage) }}
              </button>
            </div>
            <div v-if="getUsageOperatorInfo(usage)" class="col-span-2 flex items-center justify-between text-gray-400 text-sm mt-0.5">
              <span v-if="getUsageOperatorInfo(usage)">{{ getUsageOperatorInfo(usage) }}</span>
            </div>
            <div v-if="getUsageServiceStaffInfo(usage)" class="col-span-2 flex items-center justify-between text-gray-400 text-sm mt-0.5">
              <span v-if="getUsageServiceStaffInfo(usage)">{{ getUsageServiceStaffInfo(usage) }}</span>
            </div>
            <div v-if="isSyntheticAppointmentUsage(usage) && showAppointmentSettlementButton" class="col-span-2 mt-2">
              <button
                type="button"
                @click.stop="openAppointmentSettlementModal"
                class="px-4 py-2 rounded-lg bg-gray-100 text-gray-700 text-sm font-medium hover:bg-gray-200 transition-colors"
              >
                结算与补偿
              </button>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-gray-500 text-sm">{{ getWeekDay(usage.used_at) }}</span>
              <span class="text-gray-400 text-sm">{{ formatDateTime(usage.used_at) }}</span>
            </div>
          </div>
          <button
            v-if="hasMoreUsages"
            type="button"
            class="w-full flex items-center justify-start gap-2 pt-3 mt-1 border-t border-gray-100 text-sm text-gray-500"
            @click="loadMoreUsages"
          >
            <span>更多</span>
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
            </svg>
          </button>
        </div>
        <div v-else-if="!usageRecordsCollapsed" class="text-center text-gray-400 py-4">
          暂无使用记录
        </div>
      </div>
    </div>

    <div v-if="showUsageQrModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 select-none" @click.self="closeUsageQrModal" @contextmenu.prevent>
      <div class="bg-white rounded-2xl w-11/12 max-w-lg overflow-hidden">
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between">
          <h3 class="font-medium text-lg">{{ usageQrTitle }}</h3>
          <button @click="closeUsageQrModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5">
          <div class="text-center">
            <div class="text-gray-800 font-medium">{{ card?.merchant?.name || '商户' }}</div>
            <div class="text-gray-500 text-sm mt-1">{{ card?.card_type || '' }}</div>
          </div>

          <div v-if="isPrecheckModal" class="mt-2 text-center text-gray-600 text-sm">
            <template v-if="usagePrecheckDone">
              {{ replaceTerms('起单完成，进入服务', card?.merchant) }}
            </template>
            <template v-else>
              请向工作人员出示此码，由工作人员扫码进入服务
            </template>
          </div>

          <div v-if="usageQrDataUrl && !(isPrecheckModal && usagePrecheckDone)" class="mt-4 flex justify-center">
            <div
              class="select-none"
              style="-webkit-touch-callout: none; -webkit-user-select: none; user-select: none; pointer-events: none; touch-action: none;"
              @touchstart.prevent
              @touchmove.prevent
              @touchend.prevent
              @contextmenu.prevent
            >
              <img :src="usageQrDataUrl" :alt="usageQrAlt" class="w-56 h-56" style="-webkit-touch-callout: none;" />
            </div>
          </div>

          <!-- 房间号与客服人员信息 -->
          <div v-if="selectedUsage && (selectedUsage.service_room || selectedUsage.service_technician)" class="mt-3 text-center text-sm text-gray-600">
            <template v-if="selectedUsage.service_room && selectedUsage.service_technician">
              房间: {{ selectedUsage.service_room.name || selectedUsage.service_room.code }}  {{ selectedUsage.service_technician.service_role?.name || '技师' }}: {{ selectedUsage.service_technician.account }} {{ selectedUsage.service_technician.name }}
            </template>
            <template v-else-if="selectedUsage.service_room">
              房间: {{ selectedUsage.service_room.name || selectedUsage.service_room.code }}
            </template>
            <template v-else-if="selectedUsage.service_technician">
              {{ selectedUsage.service_technician.service_role?.name || '技师' }}: {{ selectedUsage.service_technician.account }} {{ selectedUsage.service_technician.name }}
            </template>
          </div>

          <div v-if="selectedUsage && getUsageProjectText(selectedUsage)" class="mt-2 text-center text-sm text-gray-600">
            {{ getUsageProjectText(selectedUsage) }}
          </div>

          
        </div>
      </div>
    </div>

    <!-- 商户通知 -->
    <div v-if="notices.length > 0" ref="noticeAnchor" class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
          </svg>
          <span class="font-medium text-gray-800">商户通知</span>
        </div>
        <div class="space-y-4">
          <div v-for="notice in notices" :key="notice.id" class="bg-white rounded-lg p-3 shadow-sm" :class="notice.is_pinned ? 'border-l-4 border-red-500 bg-red-50' : 'border-l-4 border-orange-500 bg-orange-50'">
            <div class="flex items-center gap-2 mb-1">
              <span class="font-medium text-gray-800">{{ notice.title }}</span>
              <span v-if="notice.is_pinned" class="px-2 py-0.5 bg-red-500 text-white text-xs rounded">置顶</span>
            </div>
            <div class="text-gray-500 text-sm mt-1">{{ notice.content }}</div>
            <div class="text-gray-400 text-xs mt-1">{{ formatDateTime(notice.created_at) }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 动态占位元素：仅在需要滚动到通知区域时显示，确保页面可以滚动到通知区域 -->
    <div v-if="bottomSpacerHeight > 0" :style="{ height: `${bottomSpacerHeight}px` }"></div>

    <div v-if="showUserRescheduleModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="closeUserRescheduleModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg max-h-[80vh] overflow-hidden flex flex-col">
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between flex-shrink-0">
          <h3 class="font-medium text-lg">申请改签到新时间</h3>
          <button @click="closeUserRescheduleModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="overflow-y-auto flex-1">
          <div class="px-5 py-3 border-b text-sm text-gray-600">
            当前预约：{{ getAppointmentProjectDisplay(appointment) || '默认项目' }}
          </div>
          <div
            v-if="userRescheduleEligibility && !userRescheduleEligibility.allowed"
            class="mx-5 mt-4 rounded-lg bg-orange-50 text-orange-700 border border-orange-100 px-3 py-3 text-sm"
          >
            {{ userRescheduleEligibility.reason || '当前不可改签' }}
          </div>
          <div class="px-5 py-3 border-b">
            <div class="text-sm font-medium text-gray-700 mb-2">新日期</div>
            <input
              v-model="userRescheduleDate"
              type="date"
              :min="userRescheduleDateMin"
              :max="userRescheduleDateMax"
              :disabled="userRescheduleEligibility && !userRescheduleEligibility.allowed"
              class="w-full px-3 py-2 border border-gray-200 rounded-lg disabled:bg-gray-50 disabled:text-gray-400"
            />
            <div v-if="userRescheduleDateHint" class="text-xs text-gray-400 mt-2">{{ userRescheduleDateHint }}</div>
          </div>
          <div class="px-5 py-4">
            <div v-if="userRescheduleComparisonHint" class="text-xs text-gray-500 mb-3">{{ userRescheduleComparisonHint }}</div>
            <div v-if="userRescheduleLoading" class="text-center py-8 text-gray-400">加载中...</div>
            <div v-else-if="userRescheduleError" class="text-center py-8 text-gray-400">{{ userRescheduleError }}</div>
            <div v-else-if="userRescheduleSlots.length === 0" class="text-center py-8 text-gray-400">暂无可改签时间段</div>
            <div v-else-if="userDisplayedRescheduleSlots.length === 0" class="text-center py-8 text-gray-400">当前所选专业客服无可用时间段</div>
            <div v-else class="grid grid-cols-2 gap-3">
              <button
                v-for="slot in userDisplayedRescheduleSlots"
                :key="slot.time"
                @click="selectUserRescheduleSlot(slot)"
                :class="{
                  'bg-primary text-white': userRescheduleTime === slot.time,
                  'bg-white border-2 border-gray-200 text-gray-700 hover:border-primary': userRescheduleTime !== slot.time
                }"
                class="py-3 px-4 rounded-lg font-medium transition-all text-left"
              >
                <div>{{ formatTime(slot.time) }}</div>
                <div
                  v-if="slot.comparison_label"
                  class="text-[11px] mt-1"
                  :class="userRescheduleTime === slot.time ? 'text-white/80' : getUserRescheduleSlotComparisonClass(slot.comparison_kind)"
                >
                  {{ slot.comparison_label }}
                </div>
              </button>
            </div>
          </div>
          <div v-if="userDisplayedRescheduleTechnicians.length > 0" class="px-5 py-3 border-t">
            <div class="text-sm font-medium text-gray-700 mb-2">可选客服</div>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="t in userDisplayedRescheduleTechnicians"
                :key="t.id"
                type="button"
                @click="toggleUserRescheduleTechnician(t.id)"
                :class="userRescheduleTechnicianId === t.id ? 'bg-primary text-white' : 'bg-white border-2 border-gray-200 text-gray-700 hover:border-primary'"
                class="py-2 px-3 rounded-lg font-medium transition-all text-sm"
              >
                <div>{{ t.name }}</div>
              </button>
            </div>
            <div v-if="selectedUserRescheduleTechnicianText" class="mt-2 rounded-lg bg-primary-light text-primary border border-primary/10 px-3 py-2 text-sm">
              已选客服：{{ selectedUserRescheduleTechnicianText }}
            </div>
          </div>
          <div class="px-5 py-3 border-t">
            <div class="text-sm font-medium text-gray-700 mb-2">改签原因</div>
            <textarea v-model="userRescheduleReason" rows="3" class="w-full px-3 py-2 border border-gray-200 rounded-lg" placeholder="例如：我下午更方便到店"></textarea>
          </div>
        </div>
        <div class="px-5 py-4 border-t flex-shrink-0 bg-white">
          <button
            @click="submitUserRescheduleRequest"
            :disabled="!userRescheduleTime || userRescheduleSubmitting || (userRescheduleEligibility && !userRescheduleEligibility.allowed)"
            class="w-full py-3 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {{ userRescheduleSubmitting ? '提交中...' : '提交改签申请' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showExtendRequestModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-[55]" @click.self="closeExtendRequestModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-sm overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">申请加钟</div>
          <button class="text-gray-500 text-sm" @click="closeExtendRequestModal">关闭</button>
        </div>
        <div class="px-5 py-4">
          <div class="text-sm text-gray-600 mb-3">选择本卡包含的加钟项目</div>
          <div class="space-y-2 max-h-[45vh] overflow-y-auto">
            <label
              v-for="project in availableExtendProjects"
              :key="project.id"
              class="flex items-center justify-between gap-3 rounded-lg border border-gray-200 px-3 py-3 text-sm"
              :class="Number(selectedExtendProjectId || 0) === Number(project.id) ? 'border-primary bg-primary-light' : 'bg-white'"
            >
              <span class="flex items-center gap-2">
                <input type="radio" name="extend_project" :value="project.id" v-model="selectedExtendProjectId" />
                <span class="text-gray-800">{{ project.name }}</span>
              </span>
              <span class="text-gray-500">{{ project.duration }}分钟</span>
            </label>
          </div>
          <div v-if="selectedExtendProject" class="mt-4 rounded-lg bg-orange-50 text-orange-700 px-3 py-3 text-sm">
            确认申请加钟「{{ selectedExtendProject.name }}」{{ selectedExtendProject.duration }}分钟？
          </div>
        </div>
        <div class="px-5 pb-5 flex gap-3">
          <button class="flex-1 py-3 rounded-lg bg-gray-100 text-gray-700 font-medium" @click="closeExtendRequestModal">取消</button>
          <button
            class="flex-1 py-3 rounded-lg bg-primary text-white font-medium disabled:opacity-50"
            :disabled="!selectedExtendProjectId || extendRequestSubmitting"
            @click="submitExtendRequest"
          >
            {{ extendRequestSubmitting ? '申请中...' : '确认申请' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showExtendFailureModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-[55]" @click.self="closeExtendFailureModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-sm overflow-hidden">
        <div class="px-5 py-4 border-b">
          <div class="font-medium text-gray-800">加钟失败</div>
        </div>
        <div class="px-5 py-4 text-sm text-gray-700">
          {{ selectedExtendFailureReason || '客服拒绝加钟申请' }}
        </div>
        <div class="px-5 pb-5 flex gap-3">
          <button class="flex-1 py-3 rounded-lg bg-gray-100 text-gray-700 font-medium" @click="closeExtendFailureModal">取消</button>
          <button class="flex-1 py-3 rounded-lg bg-primary text-white font-medium" @click="reapplyExtendRequest">重新申请</button>
        </div>
      </div>
    </div>

    <div v-if="showProjectModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-[55]" @click.self="closeProjectModal">
      <div class="bg-white rounded-xl w-[90%] max-w-sm overflow-hidden">
        <div class="px-4 py-3 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">请选择项目</div>
          <button class="text-gray-400" @click="closeProjectModal">×</button>
        </div>
        <div class="p-4 max-h-[60vh] overflow-y-auto">
          <div v-if="!card?.projects || card.projects.length === 0" class="text-center text-gray-400 py-6">暂无可选项目</div>
          <label v-for="p in card.projects" :key="p.id" class="flex items-center gap-3 py-2">
            <input type="radio" name="verify_project_detail" :value="p.id" v-model="selectedProjectId" />
            <div class="flex-1">
              <div class="text-gray-800">{{ p.name }}</div>
              <div v-if="p.duration" class="text-gray-400 text-xs">时长 {{ p.duration }} 分钟</div>
            </div>
          </label>
        </div>
        <div class="px-4 py-3 border-t flex gap-3">
          <button class="flex-1 py-2.5 rounded-lg border border-gray-200 text-gray-600" @click="closeProjectModal">取消</button>
          <button class="flex-1 py-2.5 rounded-lg bg-primary text-white disabled:opacity-50" :disabled="!selectedProjectId || generating" @click="confirmProjectAndGenerate">确认</button>
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
            <div class="text-gray-800 font-medium">{{ card?.merchant?.name || '商户' }}</div>
            <div class="text-gray-500 text-sm mt-1">{{ card?.card_type || '' }}</div>
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

    <div v-if="showAppointmentSettlementModal && appointment" class="fixed inset-0 bg-black/50 flex items-center justify-center z-[55]" @click.self="closeAppointmentSettlementModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg max-h-[80vh] overflow-hidden flex flex-col">
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between">
          <h3 class="font-medium text-lg">结算与补偿</h3>
          <button @click="closeAppointmentSettlementModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="px-5 py-5 overflow-y-auto space-y-3">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium text-gray-800">结算状态</div>
            <div
              v-if="appointmentSettlementStatusText"
              class="px-2 py-1 rounded-full text-xs font-medium"
              :class="getAppointmentSettlementStatusClass(appointmentSettlement?.settlement_status_snapshot || appointment.settlement_status_snapshot)"
            >
              {{ appointmentSettlementStatusText }}
            </div>
          </div>
          <div v-if="appointmentSettlement" class="grid grid-cols-2 gap-3 text-sm">
            <div class="rounded-lg bg-gray-50 px-3 py-3 border border-gray-100">
              <div class="text-xs text-gray-400">结算状态</div>
              <div class="mt-1 font-medium text-gray-800">{{ appointmentSettlementStatusText || '待结算' }}</div>
            </div>
            <div class="rounded-lg bg-gray-50 px-3 py-3 border border-gray-100">
              <div class="text-xs text-gray-400">责任归属</div>
              <div class="mt-1 font-medium text-gray-800">{{ getLiabilityText(appointmentSettlement.liability_level || appointment.liability_level) }}</div>
            </div>
          </div>
          <div v-if="appointmentSettlement?.latest_reason || appointment.disruption_reason" class="text-sm text-gray-600">
            原因：{{ getReasonText(appointmentSettlement?.latest_reason || appointment.disruption_reason) }}
          </div>
          <div v-if="appointment.merchant_cancel_reason" class="rounded-lg bg-gray-50 px-3 py-3 border border-gray-100 text-sm text-gray-700">
            <div class="font-medium text-gray-800">商户取消原因</div>
            <div class="mt-1">{{ appointment.merchant_cancel_reason }}</div>
          </div>
          <div v-if="appointment.user_rebuttal_note" class="rounded-lg bg-gray-50 px-3 py-3 border border-gray-100 text-sm text-gray-700">
            <div class="font-medium text-gray-800">我的抗辩</div>
            <div class="mt-1">{{ appointment.user_rebuttal_note }}</div>
          </div>
          <div v-if="appointmentDelayLedgerItems.length > 0" class="space-y-2">
            <div class="text-sm font-medium text-gray-700">拖堂账本</div>
            <div v-for="ledger in appointmentDelayLedgerItems" :key="ledger.id" class="rounded-lg bg-gray-50 px-3 py-3 border border-gray-100 text-sm text-gray-700">
              <div class="font-medium text-gray-800">延迟 {{ ledger.delay_minutes }} 分钟</div>
              <div class="mt-1">计入补偿桶 {{ ledger.credited_minutes }} 分钟</div>
              <div v-if="ledger.delay_compensation_value > 0" class="mt-1">累计补偿值 {{ ledger.delay_compensation_value }}</div>
              <div class="mt-1 text-gray-500">账本状态：{{ getDelayLedgerStatusText(ledger) }}</div>
            </div>
          </div>
          <div v-if="(appointment.compensations || []).length > 0" class="space-y-2">
            <div class="text-sm font-medium text-gray-700">补偿结果</div>
            <div v-for="comp in appointment.compensations" :key="comp.id" class="rounded-lg bg-gray-50 px-3 py-3 border border-gray-100 text-sm text-gray-700">
              <div class="font-medium text-gray-800">{{ getCompensationTypeText(comp.type) }}</div>
              <div v-if="getCompensationValueText(comp)" class="mt-1">{{ getCompensationValueText(comp) }}</div>
              <div v-if="comp.reason" class="mt-1 text-gray-500">{{ getReasonText(comp.reason) }}</div>
            </div>
          </div>
          <div v-if="latestForceMajeureReliefRequest" class="rounded-lg bg-gray-50 px-3 py-3 border border-gray-100 text-sm text-gray-700">
            <div class="font-medium text-gray-800">不可抗力申请</div>
            <div class="mt-1">状态：{{ getForceMajeureStatusText(latestForceMajeureReliefRequest.status) }}</div>
            <div class="mt-1">发起方：{{ getForceMajeureActorText(latestForceMajeureReliefRequest.proposed_by_type) }}</div>
            <div v-if="latestForceMajeureReliefRequest.reason" class="mt-1">原因：{{ latestForceMajeureReliefRequest.reason }}</div>
            <div v-if="latestForceMajeureReliefRequest.evidence_note" class="mt-1 text-gray-500">举证：{{ latestForceMajeureReliefRequest.evidence_note }}</div>
          </div>
          <div class="flex gap-2">
            <button
              v-if="canCreateForceMajeureRelief"
              @click="createUserForceMajeureRelief"
              class="flex-1 py-2.5 border-2 border-primary text-primary font-medium rounded-lg hover:bg-primary-light transition-colors"
            >
              申请不可抗力
            </button>
            <button
              v-if="canAcceptForceMajeureRelief"
              @click="acceptForceMajeureRelief"
              class="flex-1 py-2.5 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark transition-colors"
            >
              确认不可抗力
            </button>
            <button
              v-if="canRejectForceMajeureRelief"
              @click="rejectForceMajeureRelief"
              class="flex-1 py-2.5 border-2 border-gray-300 text-gray-700 font-medium rounded-lg hover:bg-gray-50 transition-colors"
            >
              拒绝申请
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { cardApi, usageApi, noticeApi, appointmentApi, shopApi, userServiceSessionApi, isHandledAuthRedirectError } from '../../api'
import { formatDateTime, formatDate } from '../../utils/dateFormat'
import QRCode from 'qrcode'
import { FAST_POLL_INTERVAL_MS, DATA_POLL_INTERVAL_MS } from '../../constants/polling'

import {
  getAutoFinishLabel,
  getPendingFinishLabel,
  getPendingStartLabel,
  getStartQrCodeLabel,
  getStartCountdownLabel,
  getStartTimeoutLabel,
  replaceTerms
} from '../../utils/terms'
import { normalizeSessionStatus } from '../../utils/sessionStatus'

const router = useRouter()
const route = useRoute()

const card = ref({})
const usages = ref([])
const usagesSnapshotAtMs = ref(0)
const notices = ref([])
const appointment = ref(null)
const appointmentSettlement = ref(null)
const appointmentDelayLedgers = ref([])
const canArriveNow = ref(false)
const countdown = ref(0)
const usageRecordsCollapsed = ref(false)
const visibleUsageCount = ref(10)
const showAppointmentSettlementModal = ref(false)
let countdownTimer = null
const liveServiceStatus = ref(null)
const liveServiceLoading = ref(false)
const liveServiceError = ref('')
let liveServicePollTimer = null
const LIVE_SERVICE_POLL_INTERVAL_MS = 30000
const showExtendRequestModal = ref(false)
const showExtendFailureModal = ref(false)
const selectedExtendUsage = ref(null)
const selectedExtendProjectId = ref(null)
const extendRequestSubmitting = ref(false)

const getAppointmentDisplayWaitState = (appt) => String(appt?.display_wait_state || '').trim()

const getAppointmentDisplayWaitMessage = (appt) => String(appt?.display_wait_message || '').trim()

const getAppointmentCurrentEstimatedDelayMinutes = (appt) => Number(appt?.current_estimated_delay_minutes || appt?.predicted_delay_minutes || 0)

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
  if (getAppointmentDisplayWaitState(appt) === 'cross_day_unfinished') return true
  if (!appt || appt.status !== 'arrived' || appt.actual_start_at) return false
  const appointmentTimeMs = getAppointmentTimeMs(appt)
  if (appointmentTimeMs === null) return false
  return !isSameCalendarDay(appointmentTimeMs, Date.now())
}


const verifyCode = ref('')
const codeExpireTime = ref('')
const generating = ref(false)
const verifyQrDataUrl = ref('')
const verifyCodeProject = ref(null) // 当前核销码对应的项目
const verifyCodeMode = ref('verify')
const showVerifyCodeModal = ref(false)
let verifyExpireTimer = null

let verifyStatusPollTimer = null
const verifyStatusChecking = ref(false)
const hasJumpedToRoomSelect = ref(false)
const showProjectModal = ref(false)
const selectedProjectId = ref(null)
const canceling = ref(false)
const appointmentDelayLedgerItems = computed(() => {
  const appointmentId = Number(appointment.value?.id || 0)
  if (!appointmentId) return []
  return (appointmentDelayLedgers.value || []).filter(item => Number(item?.appointment_id || 0) === appointmentId)
})
const latestForceMajeureReliefRequest = computed(() => {
  const list = Array.isArray(appointment.value?.force_majeure_relief_requests) ? appointment.value.force_majeure_relief_requests : []
  if (list.length === 0) return null
  return [...list].sort((a, b) => Number(b?.id || 0) - Number(a?.id || 0))[0]
})
const appointmentDelayMinutes = computed(() => {
  const appt = appointment.value
  if (!appt) return 0
  if (isHistoricalArrivedAppointment(appt)) return 0
  return getAppointmentCurrentEstimatedDelayMinutes(appt)
})
const appointmentDelayMetricVisible = computed(() => appointmentDelayMinutes.value > 0)
const appointmentDelayLabel = computed(() => {
  const appt = appointment.value
  if (!appt) return '预计延迟'
  const state = getAppointmentDisplayWaitState(appt)
  if (state === 'risk_pending') return '到店后可能延迟'
  if (appt.status === 'arrived') return '当前预计延迟'
  return '预计延迟'
})
const appointmentWaitingHint = computed(() => {
  const appt = appointment.value
  if (!appt || appt.status !== 'arrived') return ''
  const displayMessage = getAppointmentDisplayWaitMessage(appt)
  if (displayMessage) {
    return displayMessage
  }
  return ''
})
const canCreateForceMajeureRelief = computed(() => {
  const latest = latestForceMajeureReliefRequest.value
  if (latest && String(latest.status || '').trim() === 'pending') return false
  const compensations = Array.isArray(appointment.value?.compensations) ? appointment.value.compensations : []
  return compensations.some(item => ['merchant_breach', 'merchant_failure_offset'].includes(String(item?.source_type || item?.reason || '').trim()))
})
const canAcceptForceMajeureRelief = computed(() => {
  const latest = latestForceMajeureReliefRequest.value
  if (!latest || String(latest.status || '').trim() !== 'pending') return false
  return String(latest.proposed_by_type || '').trim() !== 'user'
})
const canRejectForceMajeureRelief = computed(() => canAcceptForceMajeureRelief.value)
const appointmentSettlementStatusText = computed(() => {
  const apptStatus = String(appointment.value?.status || '').trim()
  if (apptStatus === 'no_show') {
    return isBalanceStyleCard() ? '预约已完成扣额' : '预约已完成扣次'
  }
  return getAppointmentSettlementStatusText(
    appointmentSettlement.value?.settlement_status_snapshot || appointment.value?.settlement_status_snapshot
  )
})
const appointmentHasUsageRecord = computed(() => {
  const usageID = Number(appointment.value?.usage_id || 0)
  if (!usageID) return false
  return (usages.value || []).some(item => Number(item?.id || 0) === usageID)
})
const isAppointmentServiceWindowExpired = computed(() => {
  if (!appointment.value) return false
  const deadlineMs = getAppointmentArrivalDeadlineMs(appointment.value)
  if (deadlineMs > 0) return nowTick.value > deadlineMs
  return false
})
const shouldArchiveAppointmentToUsage = computed(() => {
  const appt = appointment.value
  if (!appt) return false
  return String(appt.status || '').trim() === 'no_show' && isAppointmentServiceWindowExpired.value
})
const appointmentLinkedUsage = computed(() => {
  const appt = appointment.value
  if (!appt) return null
  const appointmentID = Number(appt.id || 0)
  const usageID = Number(appt.usage_id || 0)
  const sessionID = Number(appt.service_session_id || 0)
  return (usages.value || []).find(item => {
    const itemUsageID = Number(item?.id || 0)
    const itemAppointmentID = Number(item?.appointment_id || 0)
    const itemSessionID = Number(item?.service_session_id || 0)
    if (usageID > 0 && itemUsageID === usageID) return true
    if (sessionID > 0 && itemSessionID === sessionID) return true
    if (appointmentID > 0 && itemAppointmentID === appointmentID) return true
    return false
  }) || null
})
const shouldHideAppointmentStatusCardAfterStart = computed(() => {
  const linkedUsage = appointmentLinkedUsage.value
  if (!linkedUsage) return false
  const sessStatus = normalizeSessionStatus(linkedUsage?.service_session_status)
  if (['serving', 'auto_finishing', 'finished'].includes(sessStatus)) return true
  if (linkedUsage?.service_session_start_confirmed_at) return true
  if (String(linkedUsage?.status || '').trim() === 'success') return true
  return false
})
const shouldShowAppointmentStatusCard = computed(() => {
  return Boolean(appointment.value) && !shouldArchiveAppointmentToUsage.value && !shouldHideAppointmentStatusCardAfterStart.value
})
const hasAppointmentSettlementContent = computed(() => {
  const appt = appointment.value
  if (!appt) return false
  return Boolean(
    appointmentSettlement.value ||
    appointmentDelayLedgerItems.value.length > 0 ||
    (appt.compensations || []).length > 0 ||
    latestForceMajeureReliefRequest.value ||
    appt.disruption_reason ||
    appt.merchant_cancel_reason ||
    appt.user_rebuttal_note
  )
})
const showAppointmentSettlementButton = computed(() => {
  const appt = appointment.value
  if (!appt || !hasAppointmentSettlementContent.value) return false
  const status = String(appt.status || '').trim()
  return isAppointmentServiceWindowExpired.value && (status === 'no_show' || status === 'failed')
})
const syntheticAppointmentUsage = computed(() => {
  const appt = appointment.value
  if (!appt || !shouldArchiveAppointmentToUsage.value || appointmentHasUsageRecord.value) return null
  const appointmentID = Number(appt.id || 0)
  if (!appointmentID) return null
  return {
    id: `appointment-${appointmentID}`,
    appointment_id: appointmentID,
    is_synthetic_appointment_usage: true,
    status: 'success',
    used_at: appt.appointment_time,
    finished_at: null,
    project_id: appt.project_id,
    project: appt.project || null,
    merchant: card.value?.merchant || null,
    used_times: 1
  }
})
const displayUsages = computed(() => {
  const list = Array.isArray(usages.value) ? [...usages.value] : []
  if (syntheticAppointmentUsage.value) {
    list.push(syntheticAppointmentUsage.value)
  }
  list.sort((left, right) => {
    const leftUsedAt = getUsageUsedAtMs(left)
    const rightUsedAt = getUsageUsedAtMs(right)
    if (leftUsedAt !== rightUsedAt) return rightUsedAt - leftUsedAt

    const leftId = Number(left?.id || 0)
    const rightId = Number(right?.id || 0)
    if (Number.isFinite(leftId) && Number.isFinite(rightId) && leftId !== rightId) {
      return rightId - leftId
    }
    return 0
  })
  return list.filter((usage) => shouldDisplayUsageRecord(usage))
})

const showUsageQrModal = ref(false)
const selectedUsage = ref(null)
const usageQrDataUrl = ref('')
const usagePrecheckDone = ref(false)

const qrMode = ref('finish')
const qrSessionId = ref('')

const sessionIdFromQuery = computed(() => {
  const v = String(route.query.session_id || '').trim()
  return v
})
const autoOpenStartQrDone = ref(false)

const isPrecheckModal = computed(() => {
  return qrMode.value === 'start'
})

const precheckCode = computed(() => {
  const sid = sessionIdFromQuery.value || String(qrSessionId.value || '').trim()
  return sid ? `SS:${sid}` : ''
})

const usageQrTitle = computed(() => {
  if (isPrecheckModal.value && usagePrecheckDone.value) return '即将进入服务'
  
  // 叫号模式下显示“扫码上号二维码”
  const merchant = card.value?.merchant
  const rawSessStatus = String(selectedUsage.value?.service_session_status || '').trim()
  const isQueueSessionByPrefix = rawSessStatus.startsWith('qs_') || rawSessStatus.startsWith('qm_') || rawSessStatus.startsWith('qms_') || rawSessStatus.startsWith('qmm_')
  const isQueueMode = !merchant?.support_customer_service_mode && merchant?.support_queue && (merchant?.queue_mode === 'auto' || merchant?.queue_mode === 'manual')
  const isMultiQueueMode = merchant?.support_queue && merchant?.queue_mode === 'auto' && merchant?.support_multi_customer_service
  const isQueueSession = isQueueSessionByPrefix || isQueueMode || isMultiQueueMode
  const sessStatus = normalizeSessionStatus(selectedUsage.value?.service_session_status)
  if (isQueueSession && sessStatus === 'start_pending') {
    return '扫码上号二维码'
  }
  if (isQueueSession && sessStatus === 'delay_pending') {
    return '扫码上号二维码'
  }
  if (isQueueSession && sessStatus === 'timeout_waiting') {
    return '扫码上号二维码'
  }
  
  return getStartQrCodeLabel(card.value?.merchant)
})

const usageQrAlt = computed(() => {
  return getStartQrCodeLabel(card.value?.merchant)
})

let usageLongPressTimer = null
let usageTouchStartX = 0
let usageTouchStartY = 0
let usageTouchMoved = false

const revokeLoading = ref(false)

const isUsageStartTimeout = (usage) => {
  const merchant = card.value?.merchant
  const supportCSMode = Boolean(merchant?.support_customer_service_mode)
  if (!supportCSMode) return false
  const supportRoom = Boolean(card.value?.merchant?.support_room)
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const precheckedAt = usage?.service_session_start_confirmed_at
  const cnt = Number(usage?.start_timeout_count || 0)
  const precheckDl = getPrecheckDeadlineAtMs(usage)
  const hasTech = Boolean(usage?.service_technician)
  // 已经选定/自动分配了客服：通常不再视为上钟超时
  // 但 start_pending 超时场景仍应成立（否则倒计时结束后状态不会变化）
  if (hasTech && sessStatus !== 'start_pending') return false
  const roomReleasedByCancel = supportRoom && sessStatus === 'canceled' && !precheckedAt && !usage?.service_room && !usage?.service_technician
  if (roomReleasedByCancel) return false
  return (
    (cnt > 0 && sessStatus === 'staff_selecting') ||
    (sessStatus === 'start_pending' && !precheckedAt && precheckDl && Date.now() >= precheckDl) ||
    (sessStatus === 'canceled' && !precheckedAt && !hasTech)
  )
}

const getQueueSessionMeta = (usage, merchant) => {
  const raw = String(usage?.service_session_status || '').trim()
  if (raw.startsWith('qm_') || raw.startsWith('qmm_')) {
    return { isQueueSession: true, isMultiQueueSession: true }
  }
  if (raw.startsWith('qs_') || raw.startsWith('qms_')) {
    return { isQueueSession: true, isMultiQueueSession: false }
  }
  const isQueueMode = !merchant?.support_customer_service_mode && merchant?.support_queue && (merchant?.queue_mode === 'auto' || merchant?.queue_mode === 'manual')
  const isMultiQueueMode = merchant?.support_queue && merchant?.queue_mode === 'auto' && merchant?.support_multi_customer_service
  return {
    isQueueSession: Boolean(isQueueMode || isMultiQueueMode),
    isMultiQueueSession: Boolean(isMultiQueueMode)
  }
}

const getQueueSessionModeKey = (usage, merchant) => {
  const raw = String(usage?.service_session_status || '').trim()
  if (raw.startsWith('qs_')) return 'qs'
  if (raw.startsWith('qm_')) return 'qm'
  if (raw.startsWith('qms_')) return 'qms'
  if (raw.startsWith('qmm_')) return 'qmm'

  if (merchant?.support_queue) {
    if (merchant?.queue_mode === 'auto') {
      return merchant?.support_multi_customer_service ? 'qm' : 'qs'
    }
    if (merchant?.queue_mode === 'manual') {
      return merchant?.support_multi_customer_service ? 'qmm' : 'qms'
    }
  }
  return ''
}

const getCardStartPendingLabel = () => getPendingStartLabel(card.value?.merchant)
const getCardPendingFinishLabel = () => getPendingFinishLabel(card.value?.merchant)
const getCardAutoFinishLabel = () => getAutoFinishLabel(card.value?.merchant)
const getCardStartTimeoutLabel = () => getStartTimeoutLabel(card.value?.merchant)
const getCardStartTimeoutPrompt = () => `${getStartTimeoutLabel(card.value?.merchant, '客服').replace(' 重新选择客服', '')}，请重新选择客服`
const getCardStartTimeoutLongPressPrompt = () => `${getStartTimeoutLabel(card.value?.merchant, '客服').replace(' 重新选择客服', '')}，请长按该记录重新选择客服`

const getUsageStatusText = (usage) => {
  const s = String(usage?.status || '').trim()
  if (s === 'in_progress') {
    const supportCS = Boolean(card.value?.merchant?.support_customer_service)
    const supportCSMode = Boolean(card.value?.merchant?.support_customer_service_mode)
    const supportRoom = Boolean(card.value?.merchant?.support_room)
    const sessStatus = normalizeSessionStatus(usage?.service_session_status)
    const precheckedAt = usage?.service_session_start_confirmed_at
    const now = nowTick.value
    const merchant = card.value?.merchant
    const { isQueueSession, isMultiQueueSession } = getQueueSessionMeta(usage, merchant)
    
    if (sessStatus === 'finished') return '完成'

    // 自动叫号单窗口：过号插队窗口期
    if (sessStatus === 'timeout_waiting') {
      if (isTimeDrivenTimeoutWaitingExpired(usage, now)) return '已过期'
      return '超时过号等待'
    }

    if (sessStatus === 'timeout_failed') {
      return '已过期'
    }
    
    // 叫号模式下 delay_pending 显示为"待扫码上号"
    if (sessStatus === 'delay_pending') {
      if (isQueueSession) {
        // 多窗口叫号：只有已分配到具体技师/窗口后才进入“待扫码上号”的交互
        if (isMultiQueueSession && !usage?.service_technician) return '待叫号'
        return '待扫码上号'
      }
      return '待服务开始'
    }
    
    // 优先按会话状态本身展示（不要依赖当前商户开关；历史会话在关闭客服后仍需正确展示）
    if (sessStatus === 'start_pending') {
      if (isQueueSession && isMultiQueueSession) {
        const term = String(merchant?.queue_window_term || '窗口')
        const tech = usage?.service_technician
        const wno = String(tech?.window_no || '').trim()
        const tname = String(tech?.name || '').trim()
        if (wno && tname) return `${term}${wno}:${tname} 待上号`
        if (wno) return `${term}${wno} 待上号`
        if (tname) return `${tname} 待上号`
        return '待上号'
      }
      if (isQueueSession) {
        return '待上号'
      }
      return getCardStartPendingLabel()
    }
    if (sessStatus === 'serving') return replaceTerms('服务中', card.value?.merchant)
    if (sessStatus === 'auto_finishing') return getCardAutoFinishLabel()
    if (supportCSMode && supportRoom && sessStatus === 'room_selecting') return '待选房间'
    if (sessStatus === 'room_locked' || sessStatus === 'staff_selecting') {
      if (isQueueSession) {
        return '待叫号'
      }
      if (usage?.service_technician) {
        return getCardStartPendingLabel()
      }
      // 若客服模式已关闭：走不开启客服模式的流程，不允许再进入"待选客服"
      const supportCSMode = Boolean(merchant?.support_customer_service_mode)
      if (!supportCSMode) {
        // 叫号模式（自动或手动）：staff_selecting 统一视为排队中
        if (merchant?.support_queue && (merchant?.queue_mode === 'auto' || merchant?.queue_mode === 'manual')) {
          return '待叫号'
        }
        return getCardStartPendingLabel()
      }
      return '待选客服'
    }
    // 客服模式下会话已取消但 usage 仍在进行中：视为上钟/选择超时
    if (sessStatus === 'canceled' && !precheckedAt) {
      const supportCSModeCanceled = Boolean(merchant?.support_customer_service_mode)
      if (supportCSModeCanceled) {
        if (supportRoom && !usage?.service_room && !usage?.service_technician) {
          return '服务超时重新选择房间'
        }
        // 已经选定/自动分配了客服：应回到待开始服务
        if (usage?.service_technician) {
          return getCardStartPendingLabel()
        }
        return getCardStartTimeoutLabel()
      }

      return '已失效'
    }
    // 上钟超时统一优先判断（避免兜底到待结束服务）
    const supportCSMode2 = Boolean(card.value?.merchant?.support_customer_service_mode)
    if (supportCSMode2) {
      const cnt = Number(usage?.start_timeout_count || 0)
      if (cnt > 0 && sessStatus === 'staff_selecting' && !usage?.service_technician) {
        return getCardStartTimeoutLabel()
      }
    }
    if (supportCSMode2 && sessStatus === 'staff_selecting') {
      // 已经选定/自动分配了客服：应回到待开始服务
      if (usage?.service_technician) return getCardStartPendingLabel()
      return '待选客服'
    }
    if (supportCSMode2 && sessStatus === 'start_pending' && !precheckedAt) {
      const dl = getPrecheckDeadlineAtMs(usage)
      if (dl && now < dl) return getCardStartPendingLabel()
      if (dl && now >= dl) return getCardStartTimeoutLabel()
    }
    // 未上钟成功（未确认开始服务）时，永远不要进入“待结束服务”兜底
    if (supportCSMode2 && !precheckedAt) {
      // start_pending 且已超时：应立刻显示"上钟超时 重新选择客服"（无需刷新页面）
      if (sessStatus === 'start_pending') {
        const dl = getPrecheckDeadlineAtMs(usage)
        if (dl && now >= dl) return getCardStartTimeoutLabel()
      }
      return getCardStartPendingLabel()
    }
    return getCardPendingFinishLabel()
  }
  if (s === 'success') return '完成'
  if (s === 'failed') {
    const sessStatus = normalizeSessionStatus(usage?.service_session_status)
    if (sessStatus === 'timeout_failed') return '已过期'
    return '失败'
  }
  if (s === 'canceled') return '已取消'
  if (s === 'pending') return '未开始'
  return s || '-'
}

const getUsageStatusClass = (usage) => {
  const s = String(usage?.status || '').trim()
  if (s === 'in_progress') {
    const supportCS = Boolean(card.value?.merchant?.support_customer_service)
    const supportCSMode = Boolean(card.value?.merchant?.support_customer_service_mode)
    const supportRoom = Boolean(card.value?.merchant?.support_room)
    const sessStatus = normalizeSessionStatus(usage?.service_session_status)
    const precheckedAt = usage?.service_session_start_confirmed_at
    const now = nowTick.value
    const { isQueueSession } = getQueueSessionMeta(usage, card.value?.merchant)
    if (sessStatus === 'finished') return 'text-gray-600'
    // 服务中：绿色
    if (sessStatus === 'serving') return 'text-green-500'
    if (isQueueSession && sessStatus === 'timeout_waiting') {
      if (isTimeDrivenTimeoutWaitingExpired(usage, now)) return 'text-red-500'
      return 'text-blue-500'
    }
    if (supportCSMode && supportRoom && sessStatus === 'room_selecting') return 'text-orange-500'
    if (sessStatus === 'room_locked' || sessStatus === 'staff_selecting') return 'text-orange-500'
    // 会话已取消但 usage 仍在进行中：视为上钟超时
    if (sessStatus === 'canceled' && !precheckedAt) {
      // 已经选定/自动分配了客服：不再视为上钟超时
      if (usage?.service_technician) {
        return 'text-red-500'
      }
      return 'text-red-500'
    }
    // 上钟超时统一优先判断（避免兜底到待结束服务样式）
    const supportCSMode3 = Boolean(card.value?.merchant?.support_customer_service_mode)
    if (supportCSMode3) {
      const cnt = Number(usage?.start_timeout_count || 0)
      if (cnt > 0 && sessStatus === 'staff_selecting') {
        return 'text-red-500'
      }
    }
    if (supportCSMode3 && sessStatus === 'staff_selecting') {
      return 'text-orange-500'
    }
    if (supportCSMode3 && sessStatus === 'start_pending' && !precheckedAt) {
      const dl = getPrecheckDeadlineAtMs(usage)
      if (dl && now < dl) return 'text-red-500'
    }
    if (supportCSMode3 && sessStatus === 'start_pending' && !precheckedAt && getPrecheckDeadlineAtMs(usage) && now >= getPrecheckDeadlineAtMs(usage)) {
      return 'text-red-500'
    }
    // 未上钟成功（未确认开始服务）时，永远不要进入待结束服务蓝色兜底
    if (supportCSMode3 && !precheckedAt) {
      return 'text-red-500'
    }
    return 'text-blue-500'
  }
  if (s === 'success') return ''
  if (s === 'failed') return 'text-red-500'
  return ''
}

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

const isUsageHandCardPendingAssignment = (usage) => {
  if (!card.value?.merchant?.support_hand_card) return false
  if (!usage || usage.hand_card_no) return false
  const status = String(usage.status || '').trim()
  if (status !== 'success' && status !== 'in_progress') return false
  const usedAtMs = getUsageUsedAtMs(usage)
  if (!usedAtMs) return false
  return nowTick.value- usedAtMs < 20 * 1000
}

const isUsageSessionFinishedButUsageInProgress = () => false

const getUsageServiceEndAtTextForFinishedSession = () => ''

const stopVerifyStatusPoll = () => {
  if (verifyStatusPollTimer) {
    clearInterval(verifyStatusPollTimer)
    verifyStatusPollTimer = null
  }
  verifyStatusChecking.value = false
}

const checkVerifyStatusAndMaybeJump = async () => {
  if (verifyStatusChecking.value) return
  if (hasJumpedToRoomSelect.value) return
  if (!verifyCode.value) return

  verifyStatusChecking.value = true
  try {
    const res = await cardApi.getVerifyCodeStatus(verifyCode.value)
    const data = res?.data?.data || {}
    const used = Boolean(data.used)
    const supportRoom = Boolean(data.merchant_support_room)
    const nextStep = String(data.next_step || '')
    const sessionId = data.session_id

    if (used && supportRoom && nextStep === 'room_select' && sessionId) {
      hasJumpedToRoomSelect.value = true
      stopVerifyStatusPoll()
      closeVerifyCodeModal()
      router.push({ path: `/user/service-sessions/${sessionId}`, query: { next_step: 'room_select' } })
      return
    }

    // 已核销但无需跳转（例如：未开启房间/客服，或 next_step 为空）：刷新当前页面数据
    if (used) {
      stopVerifyStatusPoll()
      try {
        await fetchCard()
      } catch (_) {
        // ignore
      }
      closeVerifyCodeModal()
      verifyCode.value = ''
      codeExpireTime.value = ''
      verifyQrDataUrl.value = ''
      verifyCodeProject.value = null
    }
  } catch (_) {
    // ignore
  } finally {
    verifyStatusChecking.value = false
  }
}

const startVerifyStatusPoll = async () => {
  stopVerifyStatusPoll()
  hasJumpedToRoomSelect.value = false
  if (!verifyCode.value) return

  await checkVerifyStatusAndMaybeJump()
  if (hasJumpedToRoomSelect.value) return

  verifyStatusPollTimer = setInterval(() => {
    // 核销码被清空/过期后停止轮询
    if (!verifyCode.value) {
      stopVerifyStatusPoll()
      return
    }
    checkVerifyStatusAndMaybeJump()
  }, FAST_POLL_INTERVAL_MS)
}

const formatExpireTime = (expireAtUnix) => {
  const ts = Number(expireAtUnix || 0)
  if (!ts) return ''
  const d = new Date(ts * 1000)
  return d.toLocaleString()
}

const getUsageUsedAtMs = (usage) => {
  const v = usage?.used_at
  if (!v) return 0
  const ms = new Date(v).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const FAILED_USAGE_VISIBLE_WINDOW_MS = 48 * 60 * 60 * 1000

const getTimeMs = (value) => {
  if (!value) return 0
  const ms = new Date(value).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageFailureAtMs = (usage) => {
  return (
    getUsageUsedAtMs(usage) ||
    getTimeMs(usage?.finished_at) ||
    getTimeMs(usage?.service_session_finished_at) ||
    getTimeMs(usage?.service_session_updated_at) ||
    getTimeMs(usage?.created_at)
  )
}

const shouldDisplayUsageRecord = (usage) => {
  if (String(usage?.status || '').trim() !== 'failed') return true
  const failedAtMs = getUsageFailureAtMs(usage)
  if (!failedAtMs) return true
  return nowTick.value - failedAtMs <= FAILED_USAGE_VISIBLE_WINDOW_MS
}

const getUsageSessionUpdatedAtMs = (usage) => {
  const v = usage?.service_session_updated_at
  if (!v) return 0
  const ms = new Date(v).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageSessionStartConfirmedAtMs = (usage) => {
  const v = usage?.service_session_start_confirmed_at
  if (!v) return 0
  const ms = new Date(v).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageSessionScheduledStartAtMs = (usage) => {
  const v = usage?.service_session_scheduled_start_at
  if (!v) return 0
  const ms = new Date(v).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageSessionStartedAtMs = (usage) => {
  const v = usage?.service_session_started_at
  if (!v) return 0
  const ms = new Date(v).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageSessionScheduledFinishAtMs = (usage) => {
  const v = usage?.service_session_scheduled_finish_at
  if (!v) return 0
  const ms = new Date(v).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageServiceDurationMinutes = (usage) => {
  const fromSession = Number(usage?.service_session_duration_minutes || 0)
  if (Number.isFinite(fromSession) && fromSession > 0) return fromSession
  const fromProject = Number(usage?.project?.duration || 0)
  if (Number.isFinite(fromProject) && fromProject > 0) return fromProject
  return 50
}

const getUsageProject = (usage) => {
  if (usage?.project && Number(usage.project?.id || 0) > 0) return usage.project
  const projectID = Number(usage?.project_id || 0)
  if (projectID <= 0) return null
  return (card.value?.projects || []).find((item) => Number(item?.id || 0) === projectID) || null
}

const serviceTimeSlotMatchesDate = (slot, date) => {
  if (String(slot?.recurrence_type || 'weekly') === 'monthly') {
    return Number(slot?.month_day || 0) === date.getDate()
  }
  const jsDay = date.getDay()
  const weekday = jsDay === 0 ? 7 : jsDay
  return Number(slot?.weekday || 0) === weekday
}

const isUsageProjectServiceTimeAllowed = (usage, now = new Date()) => {
  const project = getUsageProject(usage)
  const slots = Array.isArray(project?.service_time_slots) ? project.service_time_slots : []
  if (slots.length === 0) return true

  const durationMinutes = getUsageServiceDurationMinutes(usage)
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

const ensureUsageProjectServiceTimeAllowed = (usage) => {
  if (isUsageProjectServiceTimeAllowed(usage)) return true
  alert('当前不在卡片项目服务时间')
  return false
}

const getUsageProjectStartPendingTimeoutSeconds = (usage) => {
  const fromUsageProject = Number(usage?.project?.start_pending_timeout_seconds || 0)
  if (Number.isFinite(fromUsageProject) && fromUsageProject > 0) return fromUsageProject

  const projectID = Number(usage?.project_id || 0)
  if (projectID > 0) {
    const project = (card.value?.projects || []).find((item) => Number(item?.id || 0) === projectID)
    const fromCardProject = Number(project?.start_pending_timeout_seconds || 0)
    if (Number.isFinite(fromCardProject) && fromCardProject > 0) return fromCardProject
  }

  return 0
}

const getStartPendingTimeoutMs = (usage) => {
  const fromProject = getUsageProjectStartPendingTimeoutSeconds(usage)
  if (fromProject > 0) return fromProject * 1000

  const fromSession = Number(usage?.service_session_start_pending_timeout_seconds || 0)
  if (Number.isFinite(fromSession) && fromSession > 0) return fromSession * 1000

  const fromCard = Number(card.value?.start_pending_timeout_seconds || 0)
  if (!Number.isFinite(fromCard) || fromCard <= 0) return 0
  return fromCard * 1000
}

const getUsageSessionStartPendingRemainingSeconds = (usage) => {
  const n = Number(usage?.service_session_start_pending_remaining_seconds || 0)
  if (!Number.isFinite(n) || n <= 0) return 0
  return Math.floor(n)
}

const getQueueStartPendingRemainMs = (usage, nowMs) => {
  const configuredTimeoutMs = getStartPendingTimeoutMs(usage)
  const configuredTimeoutSeconds = configuredTimeoutMs > 0 ? Math.floor(configuredTimeoutMs / 1000) : 0
  const sessionTimeoutSeconds = Number(usage?.service_session_start_pending_timeout_seconds || 0)
  const shouldRecalculateByProjectTimeout = configuredTimeoutSeconds > 0 &&
    Number.isFinite(sessionTimeoutSeconds) &&
    sessionTimeoutSeconds > 0 &&
    configuredTimeoutSeconds !== Math.floor(sessionTimeoutSeconds)

  const baseMs = getUsageSessionUpdatedAtMs(usage)
  if (shouldRecalculateByProjectTimeout && baseMs > 0) {
    const diff = baseMs + configuredTimeoutMs - nowMs
    if (!Number.isFinite(diff) || diff <= 0) return 0
    return diff
  }

  const remainSeconds = getUsageSessionStartPendingRemainingSeconds(usage)
  if (remainSeconds > 0) {
    const snapshotAt = Number(usagesSnapshotAtMs.value || 0)
    const elapsed = snapshotAt > 0 ? nowMs - snapshotAt : 0
    const remain = remainSeconds * 1000 - (Number.isFinite(elapsed) ? elapsed : 0)
    if (remain > 0) return remain
    return 0
  }
  if (!baseMs) return 0
  const timeoutMs = configuredTimeoutMs > 0 ? configuredTimeoutMs : 180 * 1000
  const deadlineMs = baseMs + timeoutMs
  const diff = deadlineMs - nowMs
  if (!Number.isFinite(diff) || diff <= 0) return 0
  return diff
}

const getStartScanTimeoutMs = () => {
  const fromCard = Number(card.value?.start_scan_timeout_seconds || 0)
  if (Number.isFinite(fromCard) && fromCard > 0) return fromCard * 1000
  return 60 * 1000
}

const getUsageServiceStartAtMs = (usage) => {
  // 优先使用真正进入服务中的时间（service_session_started_at）
  // 这才是实际的服务开始时间，而不是起单确认时间或核销时间
  const startedAtMs = getUsageSessionStartedAtMs(usage)
  if (startedAtMs) return startedAtMs

  // 如果没有真正开始服务的时间，才使用起单确认时间作为兜底
  const confirmedAtMs = getUsageSessionStartConfirmedAtMs(usage)
  if (confirmedAtMs) return confirmedAtMs

  // 若未扫码起单，则按后端调度逻辑推算：updated_at + start_pending_timeout_seconds + 60s
  const sessUpdatedAtMs = getUsageSessionUpdatedAtMs(usage)
  const startPendingTimeoutMs = getStartPendingTimeoutMs(usage)
  const startScanTimeoutMs = getStartScanTimeoutMs()
  if (sessUpdatedAtMs && startPendingTimeoutMs) return sessUpdatedAtMs + startPendingTimeoutMs + startScanTimeoutMs

  // 无服务会话信息时退化：以核销时间作为服务开始时间
  const usedAtMs = getUsageUsedAtMs(usage)
  if (usedAtMs) return usedAtMs
  return 0
}

const getUsageServiceFinishAtMs = (usage) => {
  const finishAtMs = getUsageSessionScheduledFinishAtMs(usage)
  if (finishAtMs) return finishAtMs

  const startedAtMs = getUsageSessionStartedAtMs(usage)
  const durationMinutes = getUsageServiceDurationMinutes(usage)
  if (startedAtMs && durationMinutes > 0) return startedAtMs + durationMinutes * 60 * 1000

  // 兼容旧逻辑：按“起单确认/预估开始时间 + 时长”推算
  const startAtMs = getUsageServiceStartAtMs(usage)
  if (!startAtMs) return 0
  if (durationMinutes > 0) return startAtMs + durationMinutes * 60 * 1000
  return 0
}

const getUsageRoomSelectDeadlineAtMs = (usage) => {
  const raw = usage?.room_select_deadline_at
  if (!raw) return 0
  const ms = new Date(raw).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageStaffSelectCooldownDeadlineAtMs = (usage) => {
  const raw = usage?.staff_select_cooldown_until
  if (!raw) return 0
  const ms = new Date(raw).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageAutoAssignDelayMinutes = (usage) => {
  const fromUsageProject = Number(usage?.project?.auto_assign_technician_delay_minutes)
  if (Number.isFinite(fromUsageProject) && fromUsageProject >= 0) {
    return fromUsageProject
  }

  const projectID = Number(usage?.project_id || 0)
  if (projectID > 0) {
    const project = (card.value?.projects || []).find((item) => Number(item?.id || 0) === projectID)
    const fromCardProject = Number(project?.auto_assign_technician_delay_minutes)
    if (Number.isFinite(fromCardProject) && fromCardProject >= 0) {
      return fromCardProject
    }
  }

  return 5
}

const getUsageStaffSelectAutoAssignDeadlineAtMs = (usage) => {
  const baseRaw = usage?.staff_select_entered_at || usage?.room_locked_at
  if (!baseRaw) return 0
  const baseMs = new Date(baseRaw).getTime()
  if (!Number.isFinite(baseMs) || baseMs <= 0) return 0
  return baseMs + getUsageAutoAssignDelayMinutes(usage) * 60 * 1000
}

const getUsageStartTimeoutAutoAssignDeadlineAtMs = (usage) => {
  const raw = usage?.staff_select_entered_at
  if (!raw) return 0
  const enteredAtMs = new Date(raw).getTime()
  if (!Number.isFinite(enteredAtMs) || enteredAtMs <= 0) return 0
  return enteredAtMs + getUsageAutoAssignDelayMinutes(usage) * 60 * 1000
}

const getUsageTimeoutWaitingDeadlineAtMs = (usage) => {
  const merchantSeconds = Number(card.value?.merchant?.queue_timeout_waiting_seconds || 0)
  const timeoutSeconds = merchantSeconds > 0 ? merchantSeconds : 15 * 60
  const baseRaw = usage?.service_session_updated_at || usage?.updated_at || usage?.used_at
  if (!baseRaw) return 0
  const baseMs = new Date(baseRaw).getTime()
  if (!Number.isFinite(baseMs) || baseMs <= 0) return 0
  return baseMs + timeoutSeconds * 1000
}

const isTimeDrivenTimeoutWaitingExpired = (usage, nowMs = nowTick.value) => {
  const mode = getQueueSessionModeKey(usage, card.value?.merchant)
  if (mode !== 'qm') return false
  const deadlineMs = getUsageTimeoutWaitingDeadlineAtMs(usage)
  return deadlineMs > 0 && nowMs >= deadlineMs
}

const getUsageAutoFinishingDeadlineAtMs = (usage) => {
  const raw = usage?.service_session_finished_at
  if (!raw) return 0
  const ms = new Date(raw).getTime()
  return Number.isFinite(ms) ? ms : 0
}

const getUsageServiceStartAtText = (usage) => {
  if (String(usage?.status || '').trim() !== 'in_progress') return ''
  const ms = getUsageServiceStartAtMs(usage)
  if (!ms) return ''
  return formatDateTime(new Date(ms))
}

const shouldShowServiceStartTime = (usage) => {
  if (String(usage?.status || '').trim() !== 'in_progress') return false
  
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const precheckedAt = usage?.service_session_start_confirmed_at
  
  // 仅在真正进入服务中(serving)后才显示服务开始时间
  // 避免在待上钟/待开始服务(start_pending)阶段显示预估时间
  if (sessStatus !== 'serving' && sessStatus !== 'auto_finishing') return false
  if (!precheckedAt) return false
  
  return true
}

const shouldShowServiceRemainTime = (usage) => {
  if (String(usage?.status || '').trim() !== 'in_progress') return false

  const sessStatus = normalizeSessionStatus(usage?.service_session_status)

  // 仅在真实服务阶段显示剩余时间，避免“待上钟/待开始服务”阶段误显示
  if (sessStatus !== 'serving' && sessStatus !== 'auto_finishing') return false

  const finishAtMs = getUsageServiceFinishAtMs(usage)
  if (!finishAtMs) return false
  return finishAtMs > nowTick.value
}

const getUsageServiceRemainText = (usage) => {
  if (!shouldShowServiceRemainTime(usage)) return ''
  const finishAtMs = getUsageServiceFinishAtMs(usage)
  if (!finishAtMs) return ''
  const now = nowTick.value
  const diff = finishAtMs - now
  if (diff <= 0) return ''

  const totalSeconds = Math.floor(diff / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  const pad2 = (n) => String(n).padStart(2, '0')
  if (hours > 0) return `${hours}小时${pad2(minutes)}分${pad2(seconds)}秒`
  return `${minutes}分${pad2(seconds)}秒`
}

const getUsageTrackingNumber = (usage) => {
  if (isSyntheticAppointmentUsage(usage)) {
    return replaceTerms('未起单', card.value?.merchant)
  }
  if (!usage?.id) return ''
  return String(usage.id).padStart(9, '0')
}

const getSyntheticUsageAppointment = (usage) => {
  if (!isSyntheticAppointmentUsage(usage)) return null
  const usageAppointmentId = Number(usage?.appointment_id || 0)
  const currentAppointmentId = Number(appointment.value?.id || 0)
  if (usageAppointmentId > 0 && usageAppointmentId === currentAppointmentId) {
    return appointment.value
  }
  return null
}

const getUsageSyntheticStatusText = (usage) => {
  const appt = getSyntheticUsageAppointment(usage)
  if (!appt) return ''
  const status = String(appt.status || '').trim()
  const reason = String(appt.disruption_reason || appt.failed_reason || '').trim()
  if (status === 'no_show' && reason === 'user_no_show') {
    return '失约-用户未到店'
  }
  return ''
}

const getUsageSyntheticServiceEndText = (usage) => {
  const appt = getSyntheticUsageAppointment(usage)
  if (!appt) return ''
  const deadlineMs = getAppointmentArrivalDeadlineMs(appt)
  if (!Number.isFinite(deadlineMs) || deadlineMs <= 0) return ''
  return formatDateTime(new Date(deadlineMs))
}

const getUsageFailureReasonText = (usage) => {
  if (String(usage?.status || '').trim() !== 'failed') return ''

  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const rawReason = String(usage?.failure_reason || usage?.failed_reason || usage?.disruption_reason || '').trim()
  if (rawReason) return getReasonText(rawReason)

  if (sessStatus === 'canceled') {
    return '服务已取消'
  }
  return '本次核销失败'
}

const getUsageSequence = (index) => {
  const list = displayUsages.value || []
  if (!card.value || !list[index]) return 0
  if (list[index].status === 'failed') return '-'
  
  let seq = (card.value.total_times || 0) - (card.value.remain_times || 0)
  for (let i = 0; i < index; i++) {
    const u = list[i]
    if (u && u.status !== 'failed') {
      seq -= (u.used_times || 0)
    }
  }
  return seq
}

const getUsageCurrentTimesClass = (usage, index) => {
  const v = getUsageSequence(index)
  if (v === '-' || v === '' || v === null || typeof v === 'undefined') return ''

  const s = String(usage?.status || '').trim()
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)

  // 完成：默认色
  if (s === 'success' || sessStatus === 'finished') return ''

  // 服务中：绿色
  if (s === 'in_progress' && sessStatus === 'serving') return 'text-green-500'

  // 非“服务中/完成”：蓝色
  return 'text-blue-500'
}

const getPrecheckDeadlineAtMs = (usage) => {
  const ms = getUsageSessionUpdatedAtMs(usage)
  if (!ms) return 0
  const startPendingTimeoutMs = getStartPendingTimeoutMs(usage)
  if (!startPendingTimeoutMs) return 0
  return ms + startPendingTimeoutMs
}

const nowTick = ref(Date.now())
let nowTickTimer = null

const startNowTickTimer = () => {
  if (nowTickTimer) return
  nowTickTimer = setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
}

const stopNowTickTimer = () => {
  if (nowTickTimer) {
    clearInterval(nowTickTimer)
    nowTickTimer = null
  }
}

let autoAssignPollTimer = null
const stopAutoAssignPoll = () => {
  if (autoAssignPollTimer) {
    clearInterval(autoAssignPollTimer)
    autoAssignPollTimer = null
  }
}

const startAutoAssignPollIfNeeded = () => {
  const m = autoAssignCountdownMap.value || {}
  const hasAny = Object.keys(m).length > 0
  if (!hasAny) {
    stopAutoAssignPoll()
    return
  }
  if (autoAssignPollTimer) return
  autoAssignPollTimer = setInterval(() => {
    const m2 = autoAssignCountdownMap.value || {}
    if (Object.keys(m2).length === 0) {
      stopAutoAssignPoll()
      return
    }
    fetchUsages()
  }, DATA_POLL_INTERVAL_MS)
}

const shouldLivePollUsages = () => {
  const merchant = card.value?.merchant
  const list = usages.value || []
  if (!merchant || !list.length) return false

  return list.some((usage) => {
    if (String(usage?.status || '').trim() !== 'in_progress') return false

    const sessStatus = normalizeSessionStatus(usage?.service_session_status)
    const { isQueueSession } = getQueueSessionMeta(usage, merchant)
    if (!isQueueSession) return false

    return ['delay_pending', 'start_pending', 'serving', 'auto_finishing', 'timeout_waiting'].includes(sessStatus)
  })
}

let usageLivePollTimer = null

const stopUsageLivePoll = () => {
  if (usageLivePollTimer) {
    clearInterval(usageLivePollTimer)
    usageLivePollTimer = null
  }
}

const startUsageLivePollIfNeeded = () => {
  if (!shouldLivePollUsages()) {
    stopUsageLivePoll()
    return
  }
  if (usageLivePollTimer) return

  usageLivePollTimer = setInterval(() => {
    if (document.hidden) return
    if (!shouldLivePollUsages()) {
      stopUsageLivePoll()
      return
    }
    fetchUsages()
  }, DATA_POLL_INTERVAL_MS)
}

const handleVisibilityRefresh = () => {
  if (document.hidden) return
  refreshLiveServiceStatus()
  if (shouldLivePollUsages()) {
    fetchUsages()
  }
}

const usageDeadlineTriggeredKeys = new Set()
const usageDeadlineRetryState = new Map()
let usageDeadlineMonitorRunning = false

const getUsageCountdownRefreshMilestones = (usage) => {
  const merchant = card.value?.merchant
  const supportRoom = Boolean(merchant?.support_room)
  const supportCSMode = Boolean(merchant?.support_customer_service_mode)
  const supportCS = Boolean(merchant?.support_customer_service)
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const precheckedAt = usage?.service_session_start_confirmed_at
  const { isQueueSession, isMultiQueueSession } = getQueueSessionMeta(usage, merchant)
  const milestones = []
  const usageId = Number(usage?.id || 0)
  if (!usageId) return milestones

  if (isQueueSession && sessStatus === 'start_pending' && !precheckedAt) {
    if (!(isMultiQueueSession && !usage?.service_technician)) {
      const deadlineMs = getPrecheckDeadlineAtMs(usage)
      if (deadlineMs > 0) {
        milestones.push({ key: `usage:${usageId}:queue_start_pending_deadline`, deadlineMs, kind: 'queue_start_pending', usageId })
      }
    }
  }

  if (!isQueueSession && supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    const deadlineMs = getPrecheckDeadlineAtMs(usage)
    if (deadlineMs > 0) {
      milestones.push({ key: `usage:${usageId}:cs_start_pending_deadline`, deadlineMs, kind: 'cs_start_pending', usageId })
    }
  }

  if (supportCSMode && supportRoom && sessStatus === 'room_selecting') {
    const deadlineMs = getUsageRoomSelectDeadlineAtMs(usage)
    if (deadlineMs > 0) {
      milestones.push({ key: `usage:${usageId}:room_select_deadline`, deadlineMs, kind: 'room_select', usageId })
    }
  }

  if (supportCS && (sessStatus === 'room_locked' || sessStatus === 'staff_selecting')) {
    const cooldownDeadlineMs = getUsageStaffSelectCooldownDeadlineAtMs(usage)
    if (cooldownDeadlineMs > 0) {
      milestones.push({ key: `usage:${usageId}:staff_select_cooldown_deadline`, deadlineMs: cooldownDeadlineMs, kind: 'staff_select_cooldown', usageId })
    } else {
      const autoAssignDeadlineMs = getUsageStaffSelectAutoAssignDeadlineAtMs(usage)
      if (autoAssignDeadlineMs > 0) {
        milestones.push({ key: `usage:${usageId}:staff_select_auto_assign_deadline`, deadlineMs: autoAssignDeadlineMs, kind: 'staff_select_auto_assign', usageId })
      }
    }
  }

  if (supportCS && isUsageStartTimeout(usage)) {
    const deadlineMs = getUsageStartTimeoutAutoAssignDeadlineAtMs(usage)
    if (deadlineMs > 0) {
      milestones.push({ key: `usage:${usageId}:start_timeout_auto_assign_deadline`, deadlineMs, kind: 'start_timeout_auto_assign', usageId })
    }
  }

  if (!isQueueSession && supportCS && sessStatus === 'delay_pending') {
    const deadlineMs = getUsageSessionScheduledStartAtMs(usage)
    if (deadlineMs > 0) {
      milestones.push({ key: `usage:${usageId}:cs_delay_pending_deadline`, deadlineMs, kind: 'cs_delay_pending', usageId })
    }
  }

  if (isQueueSession && sessStatus === 'timeout_waiting') {
    const mode = getQueueSessionModeKey(usage, merchant)
    if (mode === 'qm' || mode === 'qms' || mode === 'qmm') {
      const deadlineMs = getUsageTimeoutWaitingDeadlineAtMs(usage)
      if (deadlineMs > 0) {
        milestones.push({ key: `usage:${usageId}:queue_timeout_waiting_deadline`, deadlineMs, kind: 'queue_timeout_waiting', usageId })
      }
    }
  }

  if (sessStatus === 'serving') {
    const deadlineMs = getUsageServiceFinishAtMs(usage)
    if (deadlineMs > 0) {
      milestones.push({ key: `usage:${usageId}:service_finish_deadline`, deadlineMs, kind: 'service_finish', usageId })
    }
  }

  if (sessStatus === 'auto_finishing') {
    const deadlineMs = getUsageAutoFinishingDeadlineAtMs(usage)
    if (deadlineMs > 0) {
      milestones.push({ key: `usage:${usageId}:auto_finishing_deadline`, deadlineMs, kind: 'auto_finishing', usageId })
    }
  }

  return milestones
}

const usageStillMatchesMilestoneKind = (usage, kind) => {
  if (!usage) return false
  const merchant = card.value?.merchant
  const supportRoom = Boolean(merchant?.support_room)
  const supportCSMode = Boolean(merchant?.support_customer_service_mode)
  const supportCS = Boolean(merchant?.support_customer_service)
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const precheckedAt = usage?.service_session_start_confirmed_at
  const { isQueueSession, isMultiQueueSession } = getQueueSessionMeta(usage, merchant)

  if (kind === 'queue_start_pending') {
    return isQueueSession && sessStatus === 'start_pending' && !precheckedAt && !(isMultiQueueSession && !usage?.service_technician)
  }
  if (kind === 'cs_start_pending') {
    return !isQueueSession && supportCS && sessStatus === 'start_pending' && !precheckedAt
  }
  if (kind === 'room_select') {
    return supportCSMode && supportRoom && sessStatus === 'room_selecting'
  }
  if (kind === 'staff_select_cooldown') {
    return supportCS && (sessStatus === 'room_locked' || sessStatus === 'staff_selecting') && getUsageStaffSelectCooldownDeadlineAtMs(usage) > 0
  }
  if (kind === 'staff_select_auto_assign') {
    return supportCS && (sessStatus === 'room_locked' || sessStatus === 'staff_selecting') && getUsageStaffSelectCooldownDeadlineAtMs(usage) <= 0 && getUsageStaffSelectAutoAssignDeadlineAtMs(usage) > 0
  }
  if (kind === 'start_timeout_auto_assign') {
    return supportCS && isUsageStartTimeout(usage) && getUsageStartTimeoutAutoAssignDeadlineAtMs(usage) > 0
  }
  if (kind === 'cs_delay_pending') {
    return !isQueueSession && supportCS && sessStatus === 'delay_pending' && getUsageSessionScheduledStartAtMs(usage) > 0
  }
  if (kind === 'queue_timeout_waiting') {
    return isQueueSession && sessStatus === 'timeout_waiting' && getUsageTimeoutWaitingDeadlineAtMs(usage) > 0
  }
  if (kind === 'service_finish') {
    return sessStatus === 'serving'
  }
  if (kind === 'auto_finishing') {
    return sessStatus === 'auto_finishing'
  }
  return false
}

const clearInactiveUsageDeadlineState = () => {
  const activeKeys = new Set()
  for (const usage of usages.value || []) {
    for (const milestone of getUsageCountdownRefreshMilestones(usage)) {
      activeKeys.add(milestone.key)
    }
  }

  for (const key of Array.from(usageDeadlineTriggeredKeys)) {
    if (!activeKeys.has(key)) {
      usageDeadlineTriggeredKeys.delete(key)
    }
  }

  for (const [key] of Array.from(usageDeadlineRetryState.entries())) {
    if (!activeKeys.has(key)) {
      usageDeadlineRetryState.delete(key)
    }
  }
}

const scheduleUsageDeadlineRetry = (milestone, attempts = 4) => {
  if (!milestone?.key) return
  usageDeadlineRetryState.set(milestone.key, {
    key: milestone.key,
    usageId: milestone.usageId,
    kind: milestone.kind,
    remainingAttempts: attempts,
    nextAtMs: nowTick.value + 1000
  })
}

const triggerUsageDeadlineRefresh = async (milestone, options = {}) => {
  if (!milestone?.key) return
  const { withRetry = true } = options
  await fetchUsages()
  if (!withRetry) return

  const latest = (usages.value || []).find(u => Number(u?.id || 0) === Number(milestone.usageId || 0))
  if (latest && usageStillMatchesMilestoneKind(latest, milestone.kind)) {
    scheduleUsageDeadlineRetry(milestone, 4)
    return
  }

  usageDeadlineRetryState.delete(milestone.key)
}

const monitorUsageCountdownMilestones = async () => {
  const due = []
  for (const usage of usages.value || []) {
    for (const milestone of getUsageCountdownRefreshMilestones(usage)) {
      if (milestone.deadlineMs > nowTick.value) continue
      if (usageDeadlineTriggeredKeys.has(milestone.key)) continue
      usageDeadlineTriggeredKeys.add(milestone.key)
      due.push(milestone)
    }
  }

  if (due.length === 0) return

  for (const milestone of due) {
    await triggerUsageDeadlineRefresh(milestone, { withRetry: true })
  }
}

const processUsageDeadlineRetries = async () => {
  const dueRetries = Array.from(usageDeadlineRetryState.values())
    .filter(item => Number(item?.nextAtMs || 0) <= nowTick.value)

  if (dueRetries.length === 0) return

  for (const item of dueRetries) {
    const latest = (usages.value || []).find(u => Number(u?.id || 0) === Number(item.usageId || 0))
    if (!latest || !usageStillMatchesMilestoneKind(latest, item.kind)) {
      usageDeadlineRetryState.delete(item.key)
      continue
    }
    if (Number(item.remainingAttempts || 0) <= 0) {
      usageDeadlineRetryState.delete(item.key)
      continue
    }

    usageDeadlineRetryState.set(item.key, {
      ...item,
      remainingAttempts: Number(item.remainingAttempts || 0) - 1,
      nextAtMs: nowTick.value + 1000
    })
    await triggerUsageDeadlineRefresh(item, { withRetry: false })
  }
}

const runUsageDeadlineMonitorTick = async () => {
  if (usageDeadlineMonitorRunning) return
  usageDeadlineMonitorRunning = true
  try {
    await monitorUsageCountdownMilestones()
    await processUsageDeadlineRetries()
  } finally {
    usageDeadlineMonitorRunning = false
  }
}

const getUsageStatusCountdownText = (usage) => {
  const s = String(usage?.status || '').trim()
  if (s !== 'in_progress') return ''

  const merchant = card.value?.merchant
  const supportRoom = Boolean(card.value?.merchant?.support_room)
  const supportCSMode = Boolean(card.value?.merchant?.support_customer_service_mode)
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const precheckedAt = usage?.service_session_start_confirmed_at
  const now = nowTick.value
  const { isQueueSession, isMultiQueueSession } = getQueueSessionMeta(usage, merchant)

  // 叫号模式待上号倒计时：与客服端保持同一口径
  if (isQueueSession && sessStatus === 'start_pending' && !precheckedAt) {
    if (isMultiQueueSession && !usage?.service_technician) return ''
    const diff = getQueueStartPendingRemainMs(usage, now)
    if (diff <= 0) return ''
    const totalSeconds = Math.floor(diff / 1000)
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = totalSeconds % 60
    const pad2 = (n) => String(n).padStart(2, '0')
    return `${getStartCountdownLabel(card.value?.merchant, { queueMode: true })} ${minutes}:${pad2(seconds)}`
  }

  // 待选房间倒计时（90秒）
  if (supportCSMode && supportRoom && sessStatus === 'room_selecting' && usage?.room_select_deadline_at) {
    const deadline = new Date(usage.room_select_deadline_at).getTime()
    const diff = deadline - now
    if (diff > 0) {
      const totalSeconds = Math.floor(diff / 1000)
      const minutes = Math.floor(totalSeconds / 60)
      const seconds = totalSeconds % 60
      return `${minutes}分${seconds}秒后自动分配房间`
    }
  }

  // 待选客服倒计时
  if (supportCS && (sessStatus === 'room_locked' || sessStatus === 'staff_selecting')) {
    if (!isUsageProjectServiceTimeAllowed(usage, new Date(now))) {
      return '当前不在卡片服务时间'
    }

    // 冷却期提示（无空闲客服）
    if (usage?.staff_select_cooldown_until) {
      const dl = new Date(usage.staff_select_cooldown_until).getTime()
      const diff = dl - now
      if (diff > 0) {
        const totalSeconds = Math.floor(diff / 1000)
        const minutes = Math.floor(totalSeconds / 60)
        const seconds = totalSeconds % 60
        return `${minutes}分${seconds}秒后可再次选择客服`
      }
    }

    // 自动分配客服倒计时（优先使用项目配置，默认5分钟）
    // 优先使用 staff_select_entered_at（用户进入选择客服页的时间），否则使用 room_locked_at（锁房时间）
    const baseMs = usage?.staff_select_entered_at
      ? new Date(usage.staff_select_entered_at).getTime()
      : usage?.room_locked_at
      ? new Date(usage.room_locked_at).getTime()
      : 0
    if (!baseMs || Number.isNaN(baseMs)) return ''
    const deadline = baseMs + getUsageAutoAssignDelayMinutes(usage) * 60 * 1000
    const diff = deadline - now
    if (diff > 0) {
      const totalSeconds = Math.floor(diff / 1000)
      const minutes = Math.floor(totalSeconds / 60)
      const seconds = totalSeconds % 60
      return `${minutes}分${seconds}秒后自动分配客服`
    }
    // 只有当用户已进入选客服页（staff_select_entered_at 有值）时才显示"正在自动分配客服"
    if (usage?.staff_select_entered_at) {
      return '正在自动分配客服...'
    }
    return ''
  }

  // 上钟超时后重新选择客服：如果已经开始计时（staff_select_entered_at）则展示自动分配倒计时
  if (supportCS && isUsageStartTimeout(usage)) {
    if (!isUsageProjectServiceTimeAllowed(usage, new Date(now))) {
      return '当前不在卡片服务时间'
    }

    const enteredAtMs = usage?.staff_select_entered_at ? new Date(usage.staff_select_entered_at).getTime() : 0
    if (!enteredAtMs || Number.isNaN(enteredAtMs)) return ''
    const deadline = enteredAtMs + getUsageAutoAssignDelayMinutes(usage) * 60 * 1000
    const diff = deadline - now
    if (diff > 0) {
      const totalSeconds = Math.floor(diff / 1000)
      const minutes = Math.floor(totalSeconds / 60)
      const seconds = totalSeconds % 60
      return `${minutes}分${seconds}秒后自动分配客服`
    }
    return '正在自动分配客服...'
  }

  // 待开始服务倒计时
  if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    const dl = getPrecheckDeadlineAtMs(usage)
    if (dl) {
      const diff = dl - now
      if (diff > 0) {
        const totalSeconds = Math.floor(diff / 1000)
        const minutes = Math.floor(totalSeconds / 60)
        const seconds = totalSeconds % 60
        const pad2 = (n) => String(n).padStart(2, '0')
        return `${getStartCountdownLabel(card.value?.merchant)} ${minutes}:${pad2(seconds)}`
      }
      // 超时后不显示倒计时
      return ''
    }
  }

  if (isQueueSession && sessStatus === 'timeout_waiting') {
    const mode = getQueueSessionModeKey(usage, merchant)
    if (mode === 'qm') {
      const deadlineMs = getUsageTimeoutWaitingDeadlineAtMs(usage)
      const diff = deadlineMs - now
      if (deadlineMs > 0 && diff > 0) {
        const totalSeconds = Math.floor(diff / 1000)
        const minutes = Math.floor(totalSeconds / 60)
        const seconds = totalSeconds % 60
        return `${minutes}分${seconds}秒后过期`
      }
      return ''
    }
    if (mode === 'qs') return '过3号失效'
    if (mode === 'qms' || mode === 'qmm') {
      const deadlineMs = getUsageTimeoutWaitingDeadlineAtMs(usage)
      const diff = deadlineMs - now
      if (deadlineMs > 0 && diff > 0) {
        const totalSeconds = Math.floor(diff / 1000)
        const minutes = Math.floor(totalSeconds / 60)
        const seconds = totalSeconds % 60
        return `超3号且${minutes}分${seconds}秒`
      }
      return '待超3号失效'
    }
    return ''
  }

  // 待自动下钟/结单倒计时
  if (sessStatus === 'auto_finishing' && usage?.service_session_finished_at) {
    const finishAt = new Date(usage.service_session_finished_at).getTime()
    const diff = finishAt - now
    if (diff > 0) {
      const totalSeconds = Math.floor(diff / 1000)
      const minutes = Math.floor(totalSeconds / 60)
      const seconds = totalSeconds % 60
      return `${minutes}分${seconds}秒后自动${replaceTerms('结单', card.value?.merchant)}`
    }
  }

  return ''
}

const getUsageStatusCountdownClass = (usage) => {
  const s = String(usage?.status || '').trim()
  if (s !== 'in_progress') return 'text-gray-400'

  const supportRoom = Boolean(card.value?.merchant?.support_room)
  const supportCSMode = Boolean(card.value?.merchant?.support_customer_service_mode)
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const precheckedAt = usage?.service_session_start_confirmed_at
  const { isQueueSession, isMultiQueueSession } = getQueueSessionMeta(usage, card.value?.merchant)

  if (isQueueSession && sessStatus === 'start_pending' && !precheckedAt) {
    if (isMultiQueueSession && !usage?.service_technician) return 'text-gray-400'
    return 'text-red-500'
  }

  // 待选房间倒计时（橙色）
  if (supportCSMode && supportRoom && sessStatus === 'room_selecting' && usage?.room_select_deadline_at) {
    return 'text-orange-500'
  }

  // 待选客服倒计时（橙色）
  if (supportCS && (sessStatus === 'room_locked' || sessStatus === 'staff_selecting') && usage?.room_locked_at) {
    if (!isUsageProjectServiceTimeAllowed(usage, new Date(nowTick.value))) {
      return 'text-gray-500'
    }
    return 'text-orange-500'
  }

  // 待开始服务倒计时（红色）
  if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    return 'text-red-500'
  }

  if (isQueueSession && sessStatus === 'timeout_waiting') {
    if (isTimeDrivenTimeoutWaitingExpired(usage, nowTick.value)) return 'text-red-500'
    return 'text-blue-500'
  }

  // 待自动下钟/结单倒计时（蓝色）
  if (sessStatus === 'auto_finishing') {
    return 'text-blue-500'
  }

  return 'text-blue-500'
}

const isOnsiteQueueUsage = (usage) => {
  return String(usage?.queue_kind || '').trim() === 'onsite' && Number(usage?.queue_no || 0) > 0
}

const getUsageQueueDisplayText = (usage) => {
  if (!isOnsiteQueueUsage(usage)) return ''
  const prefix = String(card.value?.merchant?.queue_prefix || '').trim()
  const no = Number(usage?.queue_no || 0)
  if (!no) return ''
  return `${prefix}${no}`
}

const getUsageQueueNoClass = (usage) => {
  if (!isOnsiteQueueUsage(usage)) return 'text-gray-500'
  if (usage?.queue_called_at) return 'text-green-600'
  return 'text-gray-500'
}

const getUsageQueueOperatorInfo = (usage) => {
  if (!isOnsiteQueueUsage(usage)) return ''
  const t = usage?.service_technician
  const account = String(t?.account || '').trim()
  const name = String(t?.name || '').trim()
  if (!account && !name) return ''
  if (account && name) return `${account}（${name}）`
  return account || name
}

const getExpectedCallAtMsForQueueNo = (queueNo) => {
  const n = Number(queueNo || 0)
  if (!n || n <= 1) return 0
  const prev = (usages.value || []).find(u => String(u?.queue_kind || '').trim() === 'onsite' && Number(u?.queue_no || 0) === n - 1)
  if (!prev) return 0
  const t1 = prev?.service_session_finished_at
  if (t1) {
    const ms = new Date(t1).getTime()
    return Number.isFinite(ms) ? ms : 0
  }
  const t2 = prev?.service_session_scheduled_finish_at
  if (t2) {
    const ms = new Date(t2).getTime()
    return Number.isFinite(ms) ? ms : 0
  }
  const t3 = prev?.finished_at
  if (t3) {
    const ms = new Date(t3).getTime()
    return Number.isFinite(ms) ? ms : 0
  }
  return 0
}

const getQueueCountdownMs = (usage) => {
  if (!isOnsiteQueueUsage(usage)) return 0
  const n = Number(usage?.queue_no || 0)
  if (!n) return 0

  // 已进入服务流程（服务中/待自动结束/待开始服务/待选房等）则不应再显示“预计叫号”倒计时
  const s = String(usage?.status || '').trim()
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  if (s === 'in_progress' || sessStatus) return 0

  // 已被叫号的不显示倒计时
  if (usage?.queue_called_at) return 0

  const expectMs = getExpectedCallAtMsForQueueNo(n)
  if (!expectMs) return 0

  const diff = expectMs - nowTick.value
  if (!Number.isFinite(diff) || diff <= 0) return 0
  if (diff > 10 * 60 * 1000) return 0
  return diff
}

const getUsageQueueCountdownColor = (usage) => {
  const diff = getQueueCountdownMs(usage)
  if (!diff) return 'text-gray-500'
  if (diff <= 0) return 'text-green-600'
  if (diff <= 900 * 1000) return 'text-red-500'
  return 'text-gray-500'
}

const getUsageQueueCountdownClass = (usage) => {
  const diff = getQueueCountdownMs(usage)
  if (!diff) return 'text-gray-500'
  if (diff <= 0) return 'text-green-600'
  if (diff <= 900 * 1000) return 'text-red-500'
  return 'text-gray-500'
}

const getUsageQueueCountdownText = (usage) => {
  const diff = getQueueCountdownMs(usage)
  if (!diff) return ''
  const totalSeconds = Math.floor(diff / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (diff <= 90 * 1000) {
    return `${minutes}分${seconds}秒后即将叫号`
  }
  return `${minutes}分${seconds}秒后预计叫号`
}

let usageQrPollTimer = null
let usageQrPollSessionId = ''

let autoAssignedQrCloseTimer = null
const scheduleAutoCloseUsageQrModal = () => {
  if (autoAssignedQrCloseTimer) {
    clearTimeout(autoAssignedQrCloseTimer)
    autoAssignedQrCloseTimer = null
  }
  autoAssignedQrCloseTimer = setTimeout(() => {
    autoAssignedQrCloseTimer = null
    closeUsageQrModal()
  }, 5000)
}

const stopUsageQrPoll = () => {
  if (usageQrPollTimer) {
    clearInterval(usageQrPollTimer)
    usageQrPollTimer = null
  }
  usageQrPollSessionId = ''
}

const trySwitchUsageQrToFinish = async () => {
  const sid = String(usageQrPollSessionId || '').trim()
  if (!sid) return
  await fetchUsages()
  const latest = (usages.value || []).find(u => String(u?.service_session_id || '') === sid)
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  const merchant = card.value?.merchant
  const isQueueMode = !merchant?.support_customer_service_mode && merchant?.support_queue && (merchant?.queue_mode === 'auto' || merchant?.queue_mode === 'manual')
  const isMultiQueueMode = merchant?.support_queue && merchant?.queue_mode === 'auto' && merchant?.support_multi_customer_service
  const sessStatus = normalizeSessionStatus(latest?.service_session_status)
  const precheckedAt = latest?.service_session_start_confirmed_at

  // 叫号模式：一旦进入 serving，说明扫码上号完成，自动关闭弹窗
  if (isQueueMode || isMultiQueueMode) {
    if (sessStatus === 'serving' || sessStatus === 'finished' || sessStatus === 'canceled') {
      stopUsageQrPoll()
      closeUsageQrModal()
      return
    }
    return
  }
  if (supportCS && sessStatus === 'canceled' && !precheckedAt && !latest?.service_technician) {
    stopUsageQrPoll()
    closeUsageQrModal()
    alert(getCardStartTimeoutPrompt())
    return
  }
  if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    const dl = getPrecheckDeadlineAtMs(latest)
    if (dl && Date.now() >= dl) {
      stopUsageQrPoll()
      closeUsageQrModal()
      alert(getCardStartTimeoutPrompt())
      return
    }
    return
  }
  stopUsageQrPoll()

  usagePrecheckDone.value = true
  usageQrDataUrl.value = ''
  setTimeout(() => {
    closeUsageQrModal()
  }, 3000)
}

const openUsageQrModal = async (usage) => {
  stopUsageQrPoll()
  qrMode.value = 'start'
  qrSessionId.value = ''
  usagePrecheckDone.value = false

  const merchant = card.value?.merchant
  const supportCS = Boolean(merchant?.support_customer_service)
  const sessID = usage?.service_session_id
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const precheckedAt = usage?.service_session_start_confirmed_at
  
  // 判断是否为叫号模式（未开启客服模式 + 开启叫号 + 自动叫号）
  const isQueueMode = !merchant?.support_customer_service_mode && merchant?.support_queue && (merchant?.queue_mode === 'auto' || merchant?.queue_mode === 'manual')
  const isMultiQueueMode = merchant?.support_queue && merchant?.queue_mode === 'auto' && merchant?.support_multi_customer_service
  
  // 叫号模式下的特殊处理：start_pending/delay_pending/timeout_waiting 都显示 SS 二维码，并轮询等待状态变化
  if ((isQueueMode || isMultiQueueMode) && sessID && (sessStatus === 'start_pending' || sessStatus === 'delay_pending' || sessStatus === 'timeout_waiting')) {
    // 多窗口叫号：delay_pending 但未分配技师时，仍在排队中，不弹出二维码
    if (isMultiQueueMode && sessStatus === 'delay_pending' && !usage?.service_technician) {
      return
    }
    qrMode.value = 'start'
    qrSessionId.value = String(sessID)
    selectedUsage.value = usage
    showUsageQrModal.value = true
    usageQrDataUrl.value = ''
    usagePrecheckDone.value = false
    usageQrPollSessionId = String(sessID)
    usageQrPollTimer = setInterval(() => {
      if (!showUsageQrModal.value || qrMode.value !== 'start') {
        stopUsageQrPoll()
        return
      }
      trySwitchUsageQrToFinish()
    }, FAST_POLL_INTERVAL_MS)
    try {
      usageQrDataUrl.value = await QRCode.toDataURL(`SS:${sessID}`, {
        margin: 1,
        scale: 8,
        errorCorrectionLevel: 'M'
      })
    } catch (_) {
      // ignore
    }
    return
  }

    // 会话已取消但 usage 仍在进行中：服务二维码失效（不自动跳转，改为长按记录进入重新选客服）
  if (supportCS && sessID && sessStatus === 'canceled' && !precheckedAt && !usage?.service_technician) {
    alert(getCardStartTimeoutLongPressPrompt())
    return
  }

  // 上钟超时后服务二维码失效（不自动跳转，改为长按记录进入重新选客服）
  if (supportCS && sessID && sessStatus === 'start_pending' && !precheckedAt) {
    const dl = getPrecheckDeadlineAtMs(usage)
    if (dl && Date.now() >= dl) {
      alert(getCardStartTimeoutLongPressPrompt())
      return
    }
  }

  // 仅当当前 usage 匹配 query.session_id 且确实处于待开始服务时，才显示服务二维码
  if (
    supportCS &&
    sessionIdFromQuery.value &&
    String(sessID || '') === String(sessionIdFromQuery.value) &&
    sessStatus === 'start_pending' &&
    !precheckedAt
  ) {
    qrMode.value = 'start'
    qrSessionId.value = String(sessID)
    selectedUsage.value = usage
    showUsageQrModal.value = true
    usageQrDataUrl.value = ''
    usagePrecheckDone.value = false
    usageQrPollSessionId = String(sessID)
    usageQrPollTimer = setInterval(() => {
      if (!showUsageQrModal.value || qrMode.value !== 'start') {
        stopUsageQrPoll()
        return
      }
      trySwitchUsageQrToFinish()
    }, FAST_POLL_INTERVAL_MS)
    try {
      usageQrDataUrl.value = await QRCode.toDataURL(precheckCode.value, {
        margin: 1,
        scale: 8,
        errorCorrectionLevel: 'M'
      })
    } catch (_) {
      // ignore
    }
    return
  }
  if (supportCS && sessID && sessStatus === 'start_pending' && !precheckedAt) {
    qrMode.value = 'start'
    qrSessionId.value = String(sessID)
    selectedUsage.value = usage
    showUsageQrModal.value = true
    usageQrDataUrl.value = ''
    usagePrecheckDone.value = false
    usageQrPollSessionId = String(sessID)
    usageQrPollTimer = setInterval(() => {
      if (!showUsageQrModal.value || qrMode.value !== 'start') {
        stopUsageQrPoll()
        return
      }
      trySwitchUsageQrToFinish()
    }, FAST_POLL_INTERVAL_MS)
    try {
      usageQrDataUrl.value = await QRCode.toDataURL(precheckCode.value, {
        margin: 1,
        scale: 8,
        errorCorrectionLevel: 'M'
      })
    } catch (_) {
      // ignore
    }
    return
  }

  return
}

const closeUsageQrModal = () => {
  stopUsageQrPoll()
  if (autoAssignedQrCloseTimer) {
    clearTimeout(autoAssignedQrCloseTimer)
    autoAssignedQrCloseTimer = null
  }
  showUsageQrModal.value = false
  selectedUsage.value = null
  usageQrDataUrl.value = ''
  qrMode.value = 'start'
  qrSessionId.value = ''
  usagePrecheckDone.value = false
}

const maybeOpenStartQrFromQuery = async () => {
  if (autoOpenStartQrDone.value) return
  if (route.query.openStartQr !== '1') return
  const sessionId = sessionIdFromQuery.value
  if (!sessionId) return

  const usage = (usages.value || []).find(u => {
    return String(u?.service_session_id || '') === sessionId &&
      normalizeSessionStatus(u?.service_session_status) === 'start_pending' &&
      !u?.service_session_start_confirmed_at
  })
  if (!usage) return

  autoOpenStartQrDone.value = true
  usageRecordsCollapsed.value = false
  await nextTick()
  await openUsageQrModal(usage)
}

// 记录“上钟超时后，已开始计时等待用户选技师”的记录：usageId -> enteredAtMs
const autoAssignCountdownMap = ref({})

const recordAutoAssignCountdownIfNeeded = (u) => {
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  if (!supportCS) return
  if (!isUsageStartTimeout(u)) return
  if (!u?.id) return
  if (!u?.staff_select_entered_at) return

  const enteredAtMs = new Date(u.staff_select_entered_at).getTime()
  if (Number.isNaN(enteredAtMs) || enteredAtMs <= 0) return
  const key = String(u.id)
  if (!autoAssignCountdownMap.value[key]) {
    autoAssignCountdownMap.value[key] = enteredAtMs
		startAutoAssignPollIfNeeded()
  }
}

const maybeAutoOpenPrecheckQrAfterAssigned = async () => {
  if (showUsageQrModal.value) return
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  if (!supportCS) return

  const list = usages.value || []
  const now = nowTick.value

  for (const u of list) {
    if (!u?.id) continue
    const key = String(u.id)
    const enteredAtMs = Number(autoAssignCountdownMap.value[key] || 0)
    if (!enteredAtMs) continue

    const deadline = enteredAtMs + getUsageAutoAssignDelayMinutes(u) * 60 * 1000
    // 未到5分钟：即使用户手动选了技师导致回到 start_pending，也不自动弹窗（保持原逻辑）
    if (now < deadline) continue

    const sessStatus = normalizeSessionStatus(u?.service_session_status)
    const precheckedAt = u?.service_session_start_confirmed_at
    if (sessStatus === 'start_pending' && !precheckedAt && Boolean(u?.service_technician) && Boolean(u?.service_session_id)) {
      // 认为是系统自动分配成功：自动弹出待上钟二维码，并在5秒后自动关闭
      delete autoAssignCountdownMap.value[key]
      await openUsageQrModal(u)
      if (showUsageQrModal.value) {
        scheduleAutoCloseUsageQrModal()
      }
      return
    }
  }
}

const clearUsageLongPress = () => {
  if (usageLongPressTimer) {
    clearTimeout(usageLongPressTimer)
    usageLongPressTimer = null
  }
}

const onUsageTouchStart = (e, usage) => {
  clearUsageLongPress()
  usageTouchMoved = false
  const t = e?.touches?.[0]
  usageTouchStartX = t?.clientX || 0
  usageTouchStartY = t?.clientY || 0

  usageLongPressTimer = setTimeout(() => {
    usageLongPressTimer = null
    if (usageTouchMoved) return
    ;(async () => {
      try {
        await fetchUsages()
      } catch (_) {
        // ignore
      }
      const latest = (usages.value || []).find(u => Number(u?.id) === Number(usage?.id)) || usage

      const merchant = card.value?.merchant
      const supportRoom = Boolean(merchant?.support_room)
      const supportCS = Boolean(merchant?.support_customer_service)
      const supportCSMode = Boolean(merchant?.support_customer_service_mode)
      const sessStatus = normalizeSessionStatus(latest?.service_session_status)
      const sessID = latest?.service_session_id
      const precheckedAt = latest?.service_session_start_confirmed_at
      
      // 判断是否为叫号模式（未开启客服模式 + 开启叫号 + 自动叫号）
      const isQueueMode = !merchant?.support_customer_service_mode && merchant?.support_queue && (merchant?.queue_mode === 'auto' || merchant?.queue_mode === 'manual')
      const isMultiQueueMode = merchant?.support_queue && merchant?.queue_mode === 'auto' && merchant?.support_multi_customer_service

      // 叫号模式下的特殊处理
      if (isQueueMode || isMultiQueueMode) {
        // 待叫号状态：不弹出任何内容（还在排队中）
        if (sessStatus === 'staff_selecting') {
          return
        }
        // 待扫码起单/待扫码上号状态：弹出二维码让客服扫码
        if ((sessStatus === 'start_pending' || sessStatus === 'delay_pending' || sessStatus === 'timeout_waiting') && sessID) {
          openUsageQrModal(latest)
          return
        }
        // 服务中/已完成等状态：不弹出二维码
        if (sessStatus === 'serving' || sessStatus === 'finished' || sessStatus === 'canceled') {
          return
        }
      }

      // 选客服冷却期：提示并不跳转
      const cooldownUntil = latest?.staff_select_cooldown_until
      if (cooldownUntil) {
        const dl = new Date(cooldownUntil).getTime()
        if (!Number.isNaN(dl) && Date.now() < dl) {
          const left = Math.max(0, dl - Date.now())
          const totalSeconds = Math.floor(left / 1000)
          const minutes = Math.floor(totalSeconds / 60)
          const seconds = totalSeconds % 60
          alert(`当前没有空闲客服，${minutes}分${seconds}秒后可再次选择客服`)
          return
        }
      }

      // 上钟超时：二维码应失效，直接进入重新选择客服
      if (supportCS && sessID) {
        // 会话取消且房间已释放：应重新选择房间
        if (supportRoom && sessStatus === 'canceled' && !precheckedAt && !latest?.service_room && !latest?.service_technician) {
          router.push({
            path: `/user/service-sessions/${sessID}`,
            query: { next_step: 'room_select' }
          })
          return
        }
        if (isUsageStartTimeout(latest)) {
          if (!ensureUsageProjectServiceTimeAllowed(latest)) return
          router.push({
            path: `/user/service-sessions/${sessID}`,
            query: { next_step: 'staff_select', usage_id: String(latest?.id || ''), can_revoke: latest?.can_revoke ? '1' : '0' }
          })
          return
        }
      }

      if (sessID) {
        if (supportCSMode && supportRoom && sessStatus === 'room_selecting') {
          router.push({ path: `/user/service-sessions/${sessID}`, query: { next_step: 'room_select' } })
          return
        }
        // 只有客服模式下才允许跳转到选择客服页面
        if (supportCSMode && (sessStatus === 'room_locked' || sessStatus === 'staff_selecting')) {
          if (!ensureUsageProjectServiceTimeAllowed(latest)) return
          router.push({
            path: `/user/service-sessions/${sessID}`,
            query: { next_step: 'staff_select', usage_id: String(latest?.id || ''), can_revoke: latest?.can_revoke ? '1' : '0' }
          })
          return
        }
      }

      openUsageQrModal(latest)
    })()
  }, 550)
}

const doRevokeUsage = async (usage) => {
  if (revokeLoading.value) return
  if (!usage?.can_revoke) return
  const usageId = String(usage?.id || '').trim()
  if (!usageId) return
  const ok = window.confirm('确认撤销该次核销？撤销后将返还次数，如需继续消费需重新核销。')
  if (!ok) return

  revokeLoading.value = true
  try {
    const res = await usageApi.revokeUsage(usageId)
    const remain = res?.data?.data?.remain_times
    await fetchCard()
    if (Number.isFinite(Number(remain))) {
      alert(`撤销成功，剩余次数：${remain}`)
    } else {
      alert('撤销成功')
    }
  } catch (e) {
    alert(e.response?.data?.error || '撤销失败')
  } finally {
    revokeLoading.value = false
  }
}

const onUsageTouchMove = (e) => {
  const t = e?.touches?.[0]
  const x = t?.clientX || 0
  const y = t?.clientY || 0
  if (Math.abs(x - usageTouchStartX) > 10 || Math.abs(y - usageTouchStartY) > 10) {
    usageTouchMoved = true
    clearUsageLongPress()
  }
}

const onUsageTouchEnd = () => {
  clearUsageLongPress()
}

const noticeAnchor = ref(null)
const appointmentAnchor = ref(null)
const cardSummarySection = ref(null)
const usagesAnchor = ref(null)
const usageRecordsSection = ref(null)
const bottomSpacerHeight = ref(0)

const showUserRescheduleModal = ref(false)
const userRescheduleDate = ref('')
const userRescheduleTime = ref('')
const userRescheduleSlots = ref([])
const userRescheduleError = ref('')
const userRescheduleLoading = ref(false)
const userRescheduleSubmitting = ref(false)
const userRescheduleReason = ref('')
const userRescheduleTechnicians = ref([])
const userRescheduleTechnicianId = ref(null)
const userRescheduleEligibility = ref(null)
const appointmentRescheduleRecommendationEnabled = ref(false)
const appointmentRescheduleRecommendationReason = ref('')
const appointmentRescheduleRecommendations = ref([])

const latestAppointmentRescheduleRequest = computed(() => {
  const list = Array.isArray(appointment.value?.reschedule_requests) ? appointment.value.reschedule_requests : []
  return list.find(item => item?.status === 'pending_user' || item?.status === 'pending_merchant') || null
})

const latestAppointmentCancelRequest = computed(() => {
  const list = Array.isArray(appointment.value?.cancel_requests) ? appointment.value.cancel_requests : []
  return [...list].reverse().find(item => item?.status === 'pending_user' || item?.status === 'pending_merchant' || item?.status === 'accepted' || item?.status === 'rejected') || null
})

const latestAppointmentRescheduleTechnicianText = computed(() => {
  const technician = latestAppointmentRescheduleRequest.value?.new_technician
  if (!technician) return ''
  const name = String(technician?.name || '').trim() || `客服${technician?.id || ''}`
  const account = String(technician?.account || '').trim()
  return account ? `${name} - ${account}` : name
})

const isUserRescheduleConfirmationPending = computed(() => latestAppointmentRescheduleRequest.value?.status === 'pending_user')
const isUserCancelConfirmationPending = computed(() => latestAppointmentCancelRequest.value?.status === 'pending_user')
const isMerchantReviewingUserCancelRequest = computed(() => latestAppointmentCancelRequest.value?.status === 'pending_merchant')
const canCancelUserRescheduleRequest = computed(() => {
  return String(latestAppointmentRescheduleRequest.value?.proposed_by_type || '').trim() === 'user'
})

const showUserRescheduleAction = computed(() => {
  if (!appointment.value) return false
  if (latestAppointmentRescheduleRequest.value) return false
  return appointment.value.status === 'confirmed'
})

const showCancelAppointmentAction = computed(() => {
  if (!appointment.value) return false
  if (isUserCancelConfirmationPending.value || isMerchantReviewingUserCancelRequest.value) return false
  return appointment.value.status === 'pending' || appointment.value.status === 'confirmed' || isAppointmentFailed.value
})

const canSubmitUserRebuttal = computed(() => {
  if (!appointment.value) return false
  return appointment.value.status === 'canceled' && !!String(appointment.value.merchant_cancel_reason || '').trim()
})

const userDisplayedRescheduleSlots = computed(() => {
  const list = userRescheduleSlots.value || []
  if (userRescheduleTechnicianId.value) {
    return list.filter(s => Array.isArray(s?.technician_ids) && s.technician_ids.includes(userRescheduleTechnicianId.value))
  }
  return list
})

const selectedUserRescheduleTechnicianText = computed(() => {
  const technicianId = Number(userRescheduleTechnicianId.value || 0)
  if (!technicianId) return ''
  const technician = (userRescheduleTechnicians.value || []).find(item => Number(item?.id || 0) === technicianId)
  if (!technician) return ''
  const name = String(technician?.name || '').trim() || `客服${technicianId}`
  const account = String(technician?.account || '').trim()
  return account ? `${name} - ${account}` : name
})

const userRescheduleDateMin = computed(() => {
  const dates = userRescheduleEligibility.value?.allowed_dates || []
  return dates[0] || ''
})

const userRescheduleDateMax = computed(() => {
  const dates = userRescheduleEligibility.value?.allowed_dates || []
  return dates.length > 0 ? dates[dates.length - 1] : ''
})

const userRescheduleDateHint = computed(() => {
  const eligibility = userRescheduleEligibility.value
  if (!eligibility?.allowed) return ''
  if (eligibility.rule_mode === 'same_day_only' && userRescheduleSlots.value.length === 0 && !userRescheduleLoading.value) {
    return '原预约日内暂无可改签时段'
  }
  if (eligibility.rule_mode === 'same_day_only') {
    return '当前规则仅允许改签到原预约日内的其他营业时段'
  }
  return ''
})
const userRescheduleComparisonHint = computed(() => {
  const eligibility = userRescheduleEligibility.value
  if (!eligibility?.allowed) return ''
  return '当前展示全部可改签时段，系统会优先推荐更优时段。'
})
const appointmentRescheduleRecommendationVisible = computed(() => {
  return appointmentRescheduleRecommendationEnabled.value
})

const userDisplayedRescheduleTechnicians = computed(() => {
  const list = userRescheduleTechnicians.value || []
  if (userRescheduleTime.value) {
    const slot = (userRescheduleSlots.value || []).find(s => s && s.time === userRescheduleTime.value)
    const candidates = Array.isArray(slot?.technician_candidates) ? slot.technician_candidates : []
    const byId = new Map(candidates.map(c => [Number(c.technician_id), c]))
    return list
      .filter(t => {
        const technicianId = Number(t?.id || 0)
        if (!byId.has(technicianId)) return false
        return true
      })
      .map(t => ({
        ...t,
        availability_state: byId.get(Number(t.id))?.availability_state || 'safe',
        predicted_delay_minutes: byId.get(Number(t.id))?.predicted_delay_minutes || 0
      }))
  }
  return list.map(t => ({ ...t, availability_state: 'safe', predicted_delay_minutes: 0 }))
})

const visibleUsages = computed(() => displayUsages.value.slice(0, visibleUsageCount.value))

const hasMoreUsages = computed(() => {
  return visibleUsageCount.value < displayUsages.value.length
})

const availableExtendProjects = computed(() => {
  const projects = Array.isArray(card.value?.projects) ? card.value.projects : []
  return projects.filter(project => Number(project?.id || 0) > 0 && Number(project?.duration || 0) >= 5)
})

const selectedExtendProject = computed(() => {
  const projectID = Number(selectedExtendProjectId.value || 0)
  if (!projectID) return null
  return availableExtendProjects.value.find(project => Number(project.id) === projectID) || null
})

const selectedExtendFailureReason = computed(() => {
  return String(selectedExtendUsage.value?.latest_extend_request?.reject_reason || '').trim()
})

const getUsageLatestExtendRequest = (usage) => usage?.latest_extend_request || null

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

const getUsageCurrentRemainingSeconds = (usage) => {
  const finishAtMs = getUsageServiceFinishAtMs(usage)
  if (!finishAtMs) return null
  const remain = Math.floor((finishAtMs - nowTick.value) / 1000)
  if (!Number.isFinite(remain)) return null
  return Math.max(0, remain)
}

const formatExtendRemainingSeconds = (seconds) => {
  const n = Number(seconds)
  if (!Number.isFinite(n) || n < 0) return ''
  return formatCountdownSeconds(Math.floor(n))
}

const getUsageApprovedExtendInfoLines = (usage) => {
  const req = getUsageLatestExtendRequest(usage)
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
    const currentRemain = getUsageCurrentRemainingSeconds(usage)
    if (currentRemain !== null) {
      after = currentRemain
      before = Math.max(0, currentRemain - minutes * 60)
    }
  }
  const beforeText = formatExtendRemainingSeconds(before)
  const afterText = formatExtendRemainingSeconds(after)
  const projectText = projectName ? `（${projectName} ${minutes}分钟）` : `（${minutes}分钟）`
  const firstLine = `已加钟${projectText}`
  if (!beforeText || !afterText) return [firstLine]
  return [firstLine, `加钟前剩余 ${beforeText}`, `加钟后剩余 ${afterText}`]
}

const shouldShowUsageExtendButton = (usage) => {
  if (!usage || isSyntheticAppointmentUsage(usage)) return false
  if (!usage?.service_session_id) return false
  if (String(usage?.status || '').trim() !== 'in_progress') return false
  if (normalizeSessionStatus(usage?.service_session_status) !== 'serving') return false
  return availableExtendProjects.value.length > 0
}

const getUsageExtendStatus = (usage) => String(getUsageLatestExtendRequest(usage)?.status || '').trim()

const getUsageExtendButtonText = (usage) => {
  const status = getUsageExtendStatus(usage)
  if (status === 'pending') return '加钟已申请'
  if (status === 'approved') return '已加钟'
  if (status === 'rejected') return '加钟失败'
  return '申请加钟'
}

const getUsageExtendButtonClass = (usage) => {
  const status = getUsageExtendStatus(usage)
  if (status === 'pending') return 'bg-orange-50 text-orange-600'
  if (status === 'approved') return 'bg-green-50 text-green-600'
  if (status === 'rejected') return 'bg-red-50 text-red-600'
  return 'bg-blue-500 text-white'
}

const isUsageExtendButtonDisabled = (usage) => {
  const status = getUsageExtendStatus(usage)
  return status === 'pending' || status === 'approved'
}

const handleUsageExtendClick = (usage) => {
  const status = getUsageExtendStatus(usage)
  selectedExtendUsage.value = usage
  if (status === 'rejected') {
    showExtendFailureModal.value = true
    return
  }
  openExtendRequestModal(usage)
}

const openExtendRequestModal = (usage) => {
  selectedExtendUsage.value = usage
  selectedExtendProjectId.value = availableExtendProjects.value[0]?.id || null
  showExtendRequestModal.value = true
}

const closeExtendRequestModal = () => {
  showExtendRequestModal.value = false
  selectedExtendProjectId.value = null
}

const closeExtendFailureModal = () => {
  showExtendFailureModal.value = false
}

const reapplyExtendRequest = () => {
  const usage = selectedExtendUsage.value
  showExtendFailureModal.value = false
  if (usage) openExtendRequestModal(usage)
}

const submitExtendRequest = async () => {
  const usage = selectedExtendUsage.value
  const sessionID = Number(usage?.service_session_id || 0)
  const projectID = Number(selectedExtendProjectId.value || 0)
  if (!sessionID || !projectID || extendRequestSubmitting.value) return

  extendRequestSubmitting.value = true
  try {
    await userServiceSessionApi.createExtendRequest(sessionID, { project_id: projectID })
    closeExtendRequestModal()
    await fetchUsages()
  } catch (e) {
    alert(e.response?.data?.error || '申请加钟失败')
  } finally {
    extendRequestSubmitting.value = false
  }
}

const getUsageOperatorInfo = (usage) => {
  if (isSyntheticAppointmentUsage(usage)) return ''

  // 结单前后都保留核销人员，服务人员单独展示，避免信息混用。
  if (usage.technician) {
    return `核销人员：${usage.technician.name || usage.technician.account || '技师'}`
  } else if (usage.merchant) {
    return `核销人员：${usage.merchant.name || '店铺'}`
  }

  return ''
}

const formatUsageStaffIdentity = (staff) => {
  if (!staff) return ''
  const account = String(staff?.account || '').trim()
  const name = String(staff?.name || '').trim()
  if (account && name) return `${account} - ${name}`
  return account || name || ''
}

const getUsageServiceStaffInfo = (usage) => {
  if (isSyntheticAppointmentUsage(usage)) return ''
  const staff = usage?.service_technician || null
  const identity = formatUsageStaffIdentity(staff)
  if (!identity) return ''
  return `服务人员：${identity}`
}

const getUsageRoomInfo = (usage) => {
  if (usage.service_room) {
    return `房间号：${usage.service_room.name || usage.service_room.code}`
  }
  return ''
}

const getUsageWindowInfo = (usage) => {
  // 仅开启叫号模式且技师指定了窗口号时显示
  if (!card.value?.merchant?.support_queue) return ''
  const tech = usage?.service_technician || usage?.technician
  if (!tech || !tech.window_no) return ''
  const term = card.value?.merchant?.queue_window_term || '窗口'
  return `${term}：${tech.window_no}`
}

const goBack = () => {
  router.push('/user/cards')
}

const toggleUsageRecordsCollapsed = () => {
  usageRecordsCollapsed.value = !usageRecordsCollapsed.value
}

const handleUsageHeaderClick = () => {
  if (!usageRecordsCollapsed.value) return
  usageRecordsCollapsed.value = false
}

const loadMoreUsages = () => {
  visibleUsageCount.value = Math.min(visibleUsageCount.value + 10, displayUsages.value.length)
}

const isSyntheticAppointmentUsage = (usage) => {
  return Boolean(usage?.is_synthetic_appointment_usage)
}

const getUsageAppointmentNumber = (usage) => {
  const appointmentID = Number(usage?.appointment_id || 0)
  if (appointmentID > 0) return appointmentID
  const appt = appointment.value
  if (!appt) return ''
  const currentAppointmentID = Number(appt.id || 0)
  const currentUsageID = Number(appt.usage_id || 0)
  const currentSessionID = Number(appt.service_session_id || 0)
  const usageID = Number(usage?.id || 0)
  const sessionID = Number(usage?.service_session_id || 0)
  if (currentAppointmentID <= 0) return ''
  if (currentUsageID > 0 && usageID === currentUsageID) return currentAppointmentID
  if (currentSessionID > 0 && sessionID === currentSessionID) return currentAppointmentID
  return ''
}

const openAppointmentSettlementModal = () => {
  if (!showAppointmentSettlementButton.value) return
  showAppointmentSettlementModal.value = true
}

const closeAppointmentSettlementModal = () => {
  showAppointmentSettlementModal.value = false
}

const shouldKeepUsageQrModalOpenForUsage = (usage) => {
  if (!usage) return false

  const merchant = card.value?.merchant
  const supportCS = Boolean(merchant?.support_customer_service)
  const sessStatus = normalizeSessionStatus(usage?.service_session_status)
  const precheckedAt = usage?.service_session_start_confirmed_at
  const sessID = usage?.service_session_id
  const { isQueueSession, isMultiQueueSession } = getQueueSessionMeta(usage, merchant)

  if (!sessID) return false

  if (isQueueSession) {
    if (sessStatus === 'start_pending') return true
    if (sessStatus === 'timeout_waiting') return true
    if (sessStatus === 'delay_pending') {
      if (isMultiQueueSession && !usage?.service_technician) return false
      return true
    }
    return false
  }

  if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    const deadlineMs = getPrecheckDeadlineAtMs(usage)
    if (deadlineMs && nowTick.value >= deadlineMs) return false
    return true
  }

  return false
}

const syncSelectedUsageAfterRefresh = () => {
  if (!selectedUsage.value) return

  const selectedId = Number(selectedUsage.value?.id || 0)
  const latest = (usages.value || []).find(u => Number(u?.id || 0) === selectedId)
  if (!latest) {
    if (showUsageQrModal.value) closeUsageQrModal()
    else selectedUsage.value = null
    return
  }

  selectedUsage.value = latest
  if (showUsageQrModal.value && !shouldKeepUsageQrModalOpenForUsage(latest)) {
    closeUsageQrModal()
  }
}

let fetchUsagesPromise = null

const fetchCard = async () => {
  try {
    const res = await cardApi.getCard(route.params.id)
    card.value = res.data.data
    
    if (card.value.merchant_id) {
      fetchNotices(card.value.merchant_id)
      await loadLiveServiceStatus(card.value.merchant_id)
      startLiveServicePolling()
    } else {
      stopLiveServicePolling()
      liveServiceStatus.value = null
      liveServiceError.value = ''
    }

    fetchUsages()
    fetchAppointment()
  } catch (err) {
    console.error('获取卡片详情失败:', err)
  }
}

const loadLiveServiceStatus = async (merchantId, options = {}) => {
  if (!merchantId) {
    liveServiceStatus.value = null
    return
  }

  const { silent = false } = options
  liveServiceLoading.value = true
  if (!silent) {
    liveServiceError.value = ''
  }

  try {
    const res = await shopApi.getLiveServiceStatus(merchantId)
    liveServiceStatus.value = res.data?.data || null
    liveServiceError.value = ''
  } catch (err) {
    console.error('加载门店实时状态失败:', err)
    liveServiceError.value = err.response?.data?.error || '门店实时状态获取失败'
    if (!silent) {
      liveServiceStatus.value = null
    }
  } finally {
    liveServiceLoading.value = false
  }
}

const stopLiveServicePolling = () => {
  if (liveServicePollTimer) {
    clearInterval(liveServicePollTimer)
    liveServicePollTimer = null
  }
}

const startLiveServicePolling = () => {
  stopLiveServicePolling()
  const merchantId = card.value?.merchant_id || card.value?.merchant?.id
  if (!merchantId) return

  liveServicePollTimer = setInterval(() => {
    if (document.hidden) return
    loadLiveServiceStatus(merchantId, { silent: true })
  }, LIVE_SERVICE_POLL_INTERVAL_MS)
}

const refreshLiveServiceStatus = async () => {
  const merchantId = card.value?.merchant_id || card.value?.merchant?.id
  if (!merchantId || liveServiceLoading.value) return
  await loadLiveServiceStatus(merchantId)
}

const fetchUsages = async () => {
  if (fetchUsagesPromise) return fetchUsagesPromise

  fetchUsagesPromise = (async () => {
    try {
      const res = await usageApi.getCardUsages(route.params.id)
      const allUsages = res.data.data || []
      usagesSnapshotAtMs.value = Date.now()

      usages.value = allUsages
      syncSelectedUsageAfterRefresh()
      clearInactiveUsageDeadlineState()

      try {
        for (const u of (usages.value || [])) {
          recordAutoAssignCountdownIfNeeded(u)
        }
      } catch (_) {
        // ignore
      }

      try {
        await maybeAutoOpenPrecheckQrAfterAssigned()
      } catch (_) {
        // ignore
      }

      startAutoAssignPollIfNeeded()
      startUsageLivePollIfNeeded()

      if (route.query.scrollToUsages === '1' && displayUsages.value.length > 0) {
        await scrollToUsages()
      }
      await maybeOpenStartQrFromQuery()
    } catch (err) {
      console.error('获取使用记录失败:', err)
    } finally {
      fetchUsagesPromise = null
    }
  })()

  return fetchUsagesPromise
}

const fetchNotices = async (merchantId) => {
  try {
    const res = await noticeApi.getMerchantNotices(merchantId, 5)
    notices.value = res.data.data || []
    
    // 如果需要滚动到通知区域
    if (route.query.scrollToNotice === '1' && notices.value.length > 0) {
      await scrollToNotice()
    }
  } catch (err) {
    console.error('获取通知失败:', err)
  }
}

const fetchAppointment = async () => {
  try {
    console.log('正在获取预约信息，卡片ID:', route.params.id)
    const res = await appointmentApi.getCardAppointment(route.params.id)
    console.log('预约信息响应:', res.data)
    const data = res.data.data

    if (data?.appointment) {
      appointment.value = data.appointment
      canArriveNow.value = Boolean(data.can_arrive_now)
      await loadAppointmentSettlementAndDelay()
      console.log('预约信息已设置:', appointment.value)
      // 启动倒计时
      startCountdownTimer()
      if (route.query.scrollToAppointment === '1') {
        await scrollToAppointment()
      }
      return
    }

    console.log('未找到预约信息')
    appointment.value = null
    appointmentSettlement.value = null
    appointmentDelayLedgers.value = []
    appointmentRescheduleRecommendationEnabled.value = false
    appointmentRescheduleRecommendationReason.value = ''
    appointmentRescheduleRecommendations.value = []
    canArriveNow.value = false
    stopCountdownTimer()
  } catch (err) {
    console.error('获取预约信息失败:', err)
    console.error('错误详情:', err.response?.data)
    canArriveNow.value = false
  }
}

const loadAppointmentSettlementAndDelay = async () => {
  const appointmentId = Number(appointment.value?.id || 0)
  if (!appointmentId) {
    appointmentSettlement.value = null
    appointmentDelayLedgers.value = []
    return
  }
  const cardId = Number(route.params.id || 0)
  const [settlementRes, delayRes] = await Promise.allSettled([
    appointmentApi.getUserSettlement(appointmentId),
    cardId ? appointmentApi.getUserCardDelayLedgers(cardId) : Promise.resolve({ data: { data: [] } })
  ])
  appointmentSettlement.value = settlementRes.status === 'fulfilled' ? (settlementRes.value.data?.data || null) : null
  appointmentDelayLedgers.value = delayRes.status === 'fulfilled' ? (delayRes.value.data?.data || []) : []
  const date = userRescheduleEligibility.value?.default_date || ''
  const recommendationRes = await Promise.allSettled([
    appointmentApi.getUserRescheduleRecommendations(appointmentId, date || undefined)
  ])
  if (recommendationRes[0].status === 'fulfilled') {
    const data = recommendationRes[0].value.data?.data || {}
    appointmentRescheduleRecommendationEnabled.value = !!data.enabled
    appointmentRescheduleRecommendationReason.value = String(data.reason || '').trim()
    appointmentRescheduleRecommendations.value = Array.isArray(data.recommendations) ? data.recommendations : []
  } else {
    appointmentRescheduleRecommendationEnabled.value = false
    appointmentRescheduleRecommendationReason.value = ''
    appointmentRescheduleRecommendations.value = []
  }
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

const isBalanceStyleCard = () => {
  const value = String(card.value?.card_type || '').trim()
  return value.includes('储值') || value.includes('余额') || value.includes('额度') || value.toLowerCase().includes('balance')
}

const getAppointmentSettlementStatusClass = (status) => {
  const value = String(status || '').trim()
  if (value === 'settled') return 'bg-emerald-50 text-emerald-700'
  if (value === 'refunded' || value === 'offset') return 'bg-sky-50 text-sky-700'
  if (value === 'frozen') return 'bg-amber-50 text-amber-700'
  if (value === 'transferred') return 'bg-violet-50 text-violet-700'
  return 'bg-gray-100 text-gray-600'
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
  if (value === 'merchant_timeout_failed') {
    return `客服${replaceTerms('起单', card.value?.merchant)}超时`
  }
  return map[value] || value
}

const getAppointmentFailureReasonText = (appt) => {
  const raw = String(appt?.failed_reason || appt?.disruption_reason || '').trim()
  if (!raw) return ''
  return getReasonText(raw)
}

const getAppointmentFailureLabel = (appt) => {
  const reason = String(appt?.failed_reason || appt?.disruption_reason || '').trim()
  if (reason === 'service_unclosed_cross_day' || reason === 'appointment_state_inconsistent') {
    return '异常结案'
  }
  return '预约失败'
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

const createUserForceMajeureRelief = async () => {
  if (!appointment.value) return
  const reason = window.prompt('请输入不可抗力原因，例如：停电、突发公共事故')
  if (reason == null) return
  const trimmedReason = String(reason || '').trim()
  if (!trimmedReason) {
    alert('不可抗力原因不能为空')
    return
  }
  const evidence = window.prompt('可补充举证说明（可选）') || ''
  try {
    await appointmentApi.createUserForceMajeureRelief(appointment.value.id, {
      reason: trimmedReason,
      evidence_note: String(evidence || '').trim()
    })
    alert('不可抗力申请已提交，待商户确认')
    await fetchAppointment()
  } catch (err) {
    alert(err.response?.data?.error || '提交不可抗力申请失败')
  }
}

const acceptForceMajeureRelief = async () => {
  const request = latestForceMajeureReliefRequest.value
  if (!request) return
  try {
    await appointmentApi.acceptForceMajeureRelief(request.id)
    alert('已确认不可抗力申请')
    await fetchAppointment()
  } catch (err) {
    alert(err.response?.data?.error || '确认不可抗力申请失败')
  }
}

const rejectForceMajeureRelief = async () => {
  const request = latestForceMajeureReliefRequest.value
  if (!request) return
  try {
    await appointmentApi.rejectForceMajeureRelief(request.id)
    alert('已拒绝不可抗力申请')
    await fetchAppointment()
  } catch (err) {
    alert(err.response?.data?.error || '拒绝不可抗力申请失败')
  }
}

const resetUserRescheduleForm = () => {
  userRescheduleEligibility.value = null
  userRescheduleDate.value = ''
  userRescheduleTime.value = ''
  userRescheduleSlots.value = []
  userRescheduleError.value = ''
  userRescheduleReason.value = ''
  userRescheduleTechnicians.value = []
  userRescheduleTechnicianId.value = null
}

const closeUserRescheduleModal = () => {
  showUserRescheduleModal.value = false
  userRescheduleLoading.value = false
  userRescheduleSubmitting.value = false
  resetUserRescheduleForm()
}

const buildUserAppointmentMinuteKey = (value) => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  const normalized = raw.replace('T', ' ').replace(/\.\d+$/, '').replace(/\//g, '-')
  const match = normalized.match(/(\d{4})-(\d{2})-(\d{2})\s+(\d{2}):(\d{2})/)
  if (match) {
    return `${match[1]}-${match[2]}-${match[3]} ${match[4]}:${match[5]}`
  }
  return normalized.slice(0, 16)
}

const loadUserRescheduleSlots = async (date, options = {}) => {
  const { alertOnError = true } = options
  if (!card.value?.merchant_id || !appointment.value) return
  userRescheduleLoading.value = true
  userRescheduleError.value = ''
  try {
    const res = await appointmentApi.getUserRescheduleSlots(appointment.value.id, date)
    userRescheduleEligibility.value = res.data?.data?.eligibility || userRescheduleEligibility.value
    const currentMinute = buildUserAppointmentMinuteKey(appointment.value?.appointment_time)
    const currentTechnicianId = Number(appointment.value?.technician_id || 0)
    const rawSlots = (res.data?.data?.time_slots || []).filter(slot => {
      const slotMinute = buildUserAppointmentMinuteKey(slot?.time)
      if (!currentMinute || slotMinute !== currentMinute) {
        return true
      }
      const ids = Array.isArray(slot?.technician_ids) ? slot.technician_ids.map(id => Number(id || 0)) : []
      return ids.some(id => id > 0 && id !== currentTechnicianId)
    })
    userRescheduleSlots.value = rawSlots
    userRescheduleTechnicians.value = res.data?.data?.technicians || []
    return true
  } catch (err) {
    userRescheduleSlots.value = []
    userRescheduleTechnicians.value = []
    userRescheduleError.value = err.response?.data?.error || '获取可改签时间失败'
    if (alertOnError) {
      alert(userRescheduleError.value)
    }
    return false
  } finally {
    userRescheduleLoading.value = false
  }
}

const openUserRescheduleModal = async () => {
  if (!appointment.value) return
  resetUserRescheduleForm()
  try {
    const eligibilityRes = await appointmentApi.getUserRescheduleEligibility(appointment.value.id)
    userRescheduleEligibility.value = eligibilityRes.data?.data || null
    userRescheduleDate.value = userRescheduleEligibility.value?.default_date || ''
    if (userRescheduleEligibility.value?.allowed && userRescheduleDate.value) {
      const loaded = await loadUserRescheduleSlots(userRescheduleDate.value)
      if (!loaded) {
        return
      }
    }
    showUserRescheduleModal.value = true
    if (appointment.value?.id) {
      try {
        const recRes = await appointmentApi.getUserRescheduleRecommendations(appointment.value.id, userRescheduleDate.value || undefined)
        const data = recRes.data?.data || {}
        appointmentRescheduleRecommendationEnabled.value = !!data.enabled
        appointmentRescheduleRecommendationReason.value = String(data.reason || '').trim()
        appointmentRescheduleRecommendations.value = Array.isArray(data.recommendations) ? data.recommendations : []
      } catch (_) {}
    }
  } catch (err) {
    alert(err.response?.data?.error || '获取改签资格失败')
  }
}

const toggleUserRescheduleTechnician = (id) => {
  const next = Number(id || 0)
  if (!next) return
  userRescheduleTechnicianId.value = userRescheduleTechnicianId.value === next ? null : next
  if (userRescheduleTime.value) {
    const slot = (userRescheduleSlots.value || []).find(s => s?.time === userRescheduleTime.value)
    const ids = Array.isArray(slot?.technician_ids) ? slot.technician_ids : []
    if (ids.length > 0 && !ids.includes(next)) {
      userRescheduleTime.value = ''
    }
  }
}

const selectUserRescheduleSlot = (slot) => {
  userRescheduleTime.value = slot?.time || ''
  if (userRescheduleTechnicianId.value) {
    const ids = Array.isArray(slot?.technician_ids) ? slot.technician_ids : []
    if (ids.length > 0 && !ids.includes(userRescheduleTechnicianId.value)) {
      userRescheduleTechnicianId.value = null
    }
  }
}

const submitUserRescheduleRequest = async () => {
  if (!appointment.value || !userRescheduleTime.value || userRescheduleSubmitting.value) return
  userRescheduleSubmitting.value = true
  try {
    const res = await appointmentApi.createUserRescheduleRequest(appointment.value.id, {
      appointment_time: userRescheduleTime.value,
      technician_id: userRescheduleTechnicianId.value ? Number(userRescheduleTechnicianId.value) : null,
      reason: String(userRescheduleReason.value || '').trim() || '用户申请改签'
    })
    const data = res.data?.data || {}
    closeUserRescheduleModal()
    await fetchAppointment()
    if (String(data?.mode || '').trim() === 'pending') {
      alert('改签提议已提交，等待对方确认')
    } else {
      alert('改签成功，新的预约已生效')
    }
  } catch (err) {
    alert(err.response?.data?.error || '提交改签申请失败')
  } finally {
    userRescheduleSubmitting.value = false
  }
}

const acceptUserRescheduleRequest = async () => {
  const req = latestAppointmentRescheduleRequest.value
  if (!appointment.value || !req) return
  try {
    await appointmentApi.acceptUserRescheduleRequest(appointment.value.id, req.id)
    await fetchAppointment()
    alert('你已确认改签，新的预约已生效')
  } catch (err) {
    alert(err.response?.data?.error || '确认改签失败')
  }
}

const rejectUserRescheduleRequest = async () => {
  const req = latestAppointmentRescheduleRequest.value
  if (!appointment.value || !req) return
  try {
    await appointmentApi.rejectUserRescheduleRequest(appointment.value.id, req.id)
    await fetchAppointment()
    alert('你已拒绝该改签提议')
  } catch (err) {
    alert(err.response?.data?.error || '拒绝改签失败')
  }
}

const acceptUserCancelRequest = async () => {
  const req = latestAppointmentCancelRequest.value
  if (!appointment.value || !req) return
  try {
    await appointmentApi.acceptUserCancelRequest(appointment.value.id, req.id)
    await fetchAppointment()
    alert('你已同意该取消申请，预约已取消')
  } catch (err) {
    alert(err.response?.data?.error || '确认取消失败')
  }
}

const rejectUserCancelRequest = async () => {
  const req = latestAppointmentCancelRequest.value
  if (!appointment.value || !req) return
  const note = window.prompt('请输入拒绝取消的说明', String(appointment.value?.user_rebuttal_note || '').trim())
  if (note == null) return
  const trimmedNote = String(note || '').trim()
  if (!trimmedNote) {
    alert('拒绝说明不能为空')
    return
  }
  try {
    await appointmentApi.rejectUserCancelRequest(appointment.value.id, req.id, { objection_note: trimmedNote })
    await fetchAppointment()
    alert('你已拒绝该取消申请')
  } catch (err) {
    alert(err.response?.data?.error || '拒绝取消失败')
  }
}

const cancelUserRescheduleRequest = async () => {
  const req = latestAppointmentRescheduleRequest.value
  if (!appointment.value || !req) return
  try {
    await appointmentApi.cancelUserRescheduleRequest(appointment.value.id, req.id)
    await fetchAppointment()
    alert('已撤销改签申请')
  } catch (err) {
    alert(err.response?.data?.error || '撤销改签失败')
  }
}


const isAppointmentFailed = computed(() => {
  if (!appointment.value || !appointment.value.appointment_time) return false
  if (appointment.value.status === 'failed') return true
  if (appointment.value.status !== 'pending' && appointment.value.status !== 'confirmed') return false
  const appointmentTimeMs = new Date(appointment.value.appointment_time).getTime()
  const nowMs = Date.now()
  return nowMs - appointmentTimeMs >= 30 * 60 * 1000
})

const cancelButtonDisabled = computed(() => {
  if (!appointment.value) return true
  if (isAppointmentFailed.value) return true
  return canceling.value || appointment.value.status === 'arrived' || appointment.value.status === 'completed' || appointment.value.status === 'canceled'
})

const cancelButtonText = computed(() => {
  if (isAppointmentFailed.value) {
    const label = getAppointmentFailureLabel(appointment.value)
    const reason = getAppointmentFailureReasonText(appointment.value)
    return reason ? `${label}：${reason}` : label
  }
  if (appointment.value?.cancel_deadline_at) {
    const cancelDeadlineAt = new Date(appointment.value.cancel_deadline_at).getTime()
    if (Number.isFinite(cancelDeadlineAt) && Date.now() > cancelDeadlineAt) {
      return canceling.value ? '提交中...' : '发起取消申请'
    }
  }
  return canceling.value ? '取消中...' : '取消预约'
})

const closeProjectModal = () => {
  showProjectModal.value = false
}

const closeVerifyCodeModal = () => {
  showVerifyCodeModal.value = false
  if (!verifyCode.value) {
    verifyCodeMode.value = 'verify'
  }
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

const getUsageProjectText = (usage) => {
  const pFromUsage = usage?.project
  if (pFromUsage && pFromUsage.name) {
    const duration = Number(pFromUsage.duration || 0)
    const txt = duration > 0 ? `${pFromUsage.name}（${duration}分钟）` : pFromUsage.name
    return txt ? `项目：${txt}` : ''
  }
  const pid = usage?.project_id
  if (!pid) return ''
  const p = (card.value.projects || []).find(p => Number(p.id) === Number(pid))
  if (!p) return ''
  const duration = Number(p.duration || 0)
  const txt = duration > 0 ? `${p.name}（${duration}分钟）` : p.name
  return txt ? `项目：${txt}` : ''
}

const doGenerateVerifyCode = async (projectId, options = {}) => {
  const payload = {}
  if (projectId) payload.project_id = projectId
  if (options?.appointmentId) payload.appointment_id = options.appointmentId
  const res = await cardApi.generateVerifyCode(route.params.id, payload)
  verifyCode.value = res.data.data.code
  const expireAt = new Date(res.data.data.expire_at * 1000)
  codeExpireTime.value = expireAt.toLocaleTimeString()
  verifyCodeMode.value = options?.appointmentId ? 'appointment_checkin' : 'verify'

  // 保存当前核销码对应的项目信息
  if (projectId) {
    const project = (card.value.projects || []).find(p => p.id === projectId)
    verifyCodeProject.value = project || null
  } else {
    verifyCodeProject.value = null
  }

  verifyQrDataUrl.value = await QRCode.toDataURL(verifyCode.value, {
    margin: 1,
    scale: 8,
    errorCorrectionLevel: 'M'
  })

  startVerifyStatusPoll()

  if (verifyExpireTimer) {
    clearTimeout(verifyExpireTimer)
    verifyExpireTimer = null
  }
  const delayMs = Math.max(0, expireAt.getTime() - Date.now())
  verifyExpireTimer = setTimeout(() => {
    verifyCode.value = ''
    codeExpireTime.value = ''
    verifyQrDataUrl.value = ''
    verifyCodeProject.value = null
    verifyCodeMode.value = 'verify'
    verifyExpireTimer = null

    stopVerifyStatusPoll()
  }, delayMs)
}

const confirmProjectAndGenerate = async () => {
  if (!selectedProjectId.value) return
  generating.value = true
  try {
    await doGenerateVerifyCode(Number(selectedProjectId.value))
    showProjectModal.value = false
    showVerifyCodeModal.value = true
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generating.value = false
  }
}

const generateCode = async () => {
  if (generating.value || card.value.remain_times <= 0) return

  const projects = card.value.projects || []
  if (projects.length > 1) {
    selectedProjectId.value = null
    showProjectModal.value = true
    return
  }

  generating.value = true
  try {
    const onlyProjectId = projects.length === 1 ? projects[0].id : null
    await doGenerateVerifyCode(onlyProjectId)
    showVerifyCodeModal.value = true
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generating.value = false
  }
}

watch(userRescheduleDate, async (nextDate, prevDate) => {
  if (!showUserRescheduleModal.value || !nextDate || nextDate === prevDate) return
  const allowedDates = userRescheduleEligibility.value?.allowed_dates || []
  if (allowedDates.length > 0 && !allowedDates.includes(nextDate)) {
    userRescheduleDate.value = userRescheduleEligibility.value?.default_date || allowedDates[0] || ''
    return
  }
  await loadUserRescheduleSlots(nextDate)
})

// 格式化时间显示
const formatTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  return `${hours}:${minutes}`
}

const getUserRescheduleSlotComparisonClass = (kind) => {
  if (kind === 'better') return 'text-green-600'
  if (kind === 'not_worse') return 'text-orange-500'
  return 'text-gray-400'
}

const maybeRedirectToUserRescheduleBeforeCancel = async () => {
  if (!appointment.value?.id) return false
  try {
    const [eligibilityRes, recommendationRes] = await Promise.allSettled([
      appointmentApi.getUserRescheduleEligibility(appointment.value.id),
      appointmentApi.getUserRescheduleRecommendations(appointment.value.id, userRescheduleDate.value || undefined)
    ])
    const eligibility = eligibilityRes.status === 'fulfilled' ? (eligibilityRes.value.data?.data || null) : null
    const recommendationData = recommendationRes.status === 'fulfilled' ? (recommendationRes.value.data?.data || {}) : {}
    const recommendations = Array.isArray(recommendationData.recommendations) ? recommendationData.recommendations : []
    if (!eligibility?.allowed && recommendations.length === 0) {
      return false
    }
    const message = recommendations.length > 0
      ? `系统当前已给出 ${recommendations.length} 个可参考改签时段。是否先查看改签建议，再决定是否取消？`
      : '当前预约仍可改签。是否先查看改签时段，再决定是否取消？'
    if (!window.confirm(message)) return false
    await openUserRescheduleModal()
    return true
  } catch (_) {
    return false
  }
}

const cancelAppointment = async () => {
  if (canceling.value) return

  if (isAppointmentFailed.value) return
  if (await maybeRedirectToUserRescheduleBeforeCancel()) return
  
  if (!confirm('确定要取消预约吗？')) return
  
  canceling.value = true
  try {
    const cancelDeadlineAt = appointment.value?.cancel_deadline_at ? new Date(appointment.value.cancel_deadline_at).getTime() : NaN
    const shouldUseCancelRequest = Number.isFinite(cancelDeadlineAt) ? Date.now() > cancelDeadlineAt : false
    if (shouldUseCancelRequest) {
      const reason = window.prompt('已超过直接取消时限，请填写取消申请原因')
      if (reason == null) return
      const trimmedReason = String(reason || '').trim()
      if (!trimmedReason) {
        alert('取消申请原因不能为空')
        return
      }
      await appointmentApi.createUserCancelRequest(appointment.value.id, { reason: trimmedReason })
      await fetchAppointment()
      alert('取消申请已提交，待商户确认')
      return
    }

    await appointmentApi.cancelAppointment(appointment.value.id)
    appointment.value = null
    stopCountdownTimer()
    alert('已取消预约')
  } catch (err) {
    if (isHandledAuthRedirectError(err)) return
    alert(err.response?.data?.error || '取消预约失败')
  } finally {
    canceling.value = false
  }
}

const submitUserRebuttal = async () => {
  if (!appointment.value?.id) return
  const initialValue = String(appointment.value?.user_rebuttal_note || '').trim()
  const note = window.prompt('请输入你的抗辩意见', initialValue)
  if (note == null) return
  const trimmedNote = String(note || '').trim()
  if (!trimmedNote) {
    alert('抗辩意见不能为空')
    return
  }
  try {
    await appointmentApi.updateUserRebuttal(appointment.value.id, { user_rebuttal_note: trimmedNote })
    await fetchAppointment()
    alert('抗辩意见已保存')
  } catch (err) {
    alert(err.response?.data?.error || '保存抗辩意见失败')
  }
}

const getAppointmentStatusClass = (appt) => {
  const status = String(appt?.status || '').trim()
  if (status === 'arrived' && isHistoricalArrivedAppointment(appt)) {
    return 'text-red-600'
  }
  if (status === 'failed') {
    return 'text-red-600'
  }
  const classes = {
    pending: 'text-primary',
    confirmed: 'text-primary',
    arrived: 'text-primary',
    completed: 'text-gray-600',
    canceled: 'text-gray-400'
  }
  return classes[status] || 'text-gray-500'
}

const getAppointmentStatusText = (appt) => {
  const status = String(appt?.status || '').trim()
  if (status === 'arrived' && isHistoricalArrivedAppointment(appt)) {
    return '超时待处理'
  }
  if (status === 'failed') {
    return getAppointmentFailureLabel(appt)
  }
  const texts = {
    pending: '待确认',
    confirmed: '待到店',
    arrived: '已到店待服务',
    completed: '已完成',
    canceled: '已取消',
    no_show: '已失约'
  }
  return texts[status] || status
}

const getWeekDay = (dateStr) => {
  if (!dateStr) return ''
  const weekDays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  const date = new Date(dateStr)
  return weekDays[date.getDay()]
}

// 计算倒计时（秒）
const calculateCountdown = () => {
  if (!appointment.value || !appointment.value.appointment_time) return 0
  const appointmentTimeMs = new Date(appointment.value.appointment_time).getTime()
  const nowMs = Date.now()
  return Math.floor((appointmentTimeMs - nowMs) / 1000)
}

const getAppointmentArrivalDeadlineMs = (targetAppointment = appointment.value) => {
  const appt = targetAppointment
  if (!appt) return 0
  if (appt.reserved_end_at) {
    const reservedEndMs = new Date(appt.reserved_end_at).getTime()
    if (Number.isFinite(reservedEndMs) && reservedEndMs > 0) {
      const minServiceMinutes = Math.max(0, Math.floor(Number(appt.late_arrival_min_service_minutes || 0)))
      return reservedEndMs - minServiceMinutes * 60 * 1000
    }
  }
  if (appt.appointment_time) {
    const appointmentTimeMs = new Date(appt.appointment_time).getTime()
    if (Number.isFinite(appointmentTimeMs) && appointmentTimeMs > 0) {
      return appointmentTimeMs + 15 * 60 * 1000
    }
  }
  return 0
}

// 更新倒计时
const updateCountdown = () => {
  countdown.value = calculateCountdown()
}

// 启动倒计时定时器
const startCountdownTimer = () => {
  stopCountdownTimer()
  updateCountdown()
  countdownTimer = setInterval(updateCountdown, 1000)
}

// 停止倒计时定时器
const stopCountdownTimer = () => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

// 判断预约是否已过
const isAppointmentPassed = () => {
  const deadlineMs = getAppointmentArrivalDeadlineMs()
  if (deadlineMs > 0) {
    return Date.now() > deadlineMs
  }
  return countdown.value < -60
}

// 获取预约时间的颜色类
const getAppointmentTimeClass = () => {
  if (countdown.value < -60) {
    return 'text-gray-400' // 超过1分钟，灰色
  } else if (countdown.value > 0 && countdown.value <= 300) {
    return 'text-red-500' // 5分钟内，红色
  } else {
    return 'text-gray-800' // 默认使用正文色
  }
}

// 获取倒计时文字颜色类
const getCountdownClass = () => {
  if (countdown.value > 0 && countdown.value <= 300) {
    return 'text-red-500' // 5分钟内，红色
  } else {
    return 'text-gray-600' // 默认中性色
  }
}

const formatCountdownSeconds = (totalSeconds) => {
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

// 获取倒计时文字
const getCountdownText = () => {
  const nowMs = Date.now()
  const appointmentTimeMs = appointment.value?.appointment_time ? new Date(appointment.value.appointment_time).getTime() : 0
  const arrivalDeadlineMs = getAppointmentArrivalDeadlineMs()
  if (appointmentTimeMs > 0 && nowMs > appointmentTimeMs && canArriveNow.value && arrivalDeadlineMs > nowMs) {
    const remainSeconds = Math.floor((arrivalDeadlineMs - nowMs) / 1000)
    if (remainSeconds <= 0) return '预约时间已到'
    return formatCountdownSeconds(remainSeconds)
  }
  if (countdown.value <= 0 && countdown.value > -60) {
    return '预约时间已到'
  }
  return formatCountdownSeconds(Math.abs(countdown.value))
}

const getAppointmentCountdownLabel = () => {
  const appt = appointment.value
  if (String(appt?.status || '').trim() !== 'confirmed') return ''
  const appointmentTimeMs = appt?.appointment_time ? new Date(appt.appointment_time).getTime() : 0
  if (appointmentTimeMs > 0 && Date.now() > appointmentTimeMs && canArriveNow.value) {
    return '最晚到店剩余'
  }
  return '距预约开始'
}

const shouldShowAppointmentCountdownLabel = computed(() => {
  return String(appointment.value?.status || '').trim() === 'confirmed'
})

const shouldShowAppointmentCountdownValue = computed(() => {
  return String(appointment.value?.status || '').trim() === 'confirmed' && !isAppointmentPassed()
})

const shouldShowAppointmentPassedText = computed(() => {
  return String(appointment.value?.status || '').trim() === 'confirmed' && isAppointmentPassed()
})

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

// 判断是否应该显示核销码区域
const shouldShowVerifyCode = () => {
  if (card.value?.locked) return false
  // 如果没有预约，显示核销码
  if (!appointment.value) {
    return true
  }
  
  // 如果有预约，判断条件
  // 预约到店核销统一以后端返回的 can_arrive_now 为准，
  // 这样能复用预约保护窗口与宽限时间判断，不再依赖前端本地倒计时硬编码。
  if (appointment.value.status === 'confirmed') {
    return canArriveNow.value
  }
  
  // 其他状态（pending, finished, canceled）不显示核销码
  return false
}

const showAppointmentArrivalVerifyButton = computed(() => {
  return Boolean(appointment.value && appointment.value.status === 'confirmed' && shouldShowVerifyCode())
})

const openAppointmentArrivalVerifyFlow = async () => {
  if (!showAppointmentArrivalVerifyButton.value || generating.value) return
  if (appointment.value?.project_id) {
    generating.value = true
    try {
      await doGenerateVerifyCode(Number(appointment.value.project_id), { appointmentId: Number(appointment.value.id) })
      showVerifyCodeModal.value = true
    } catch (err) {
      alert(err.response?.data?.error || '生成核销码失败')
    } finally {
      generating.value = false
    }
    return
  }
  await generateCode()
}

// 获取商家地址
const getMerchantAddress = () => {
  if (!card.value || !card.value.merchant) return ''
  
  const m = card.value.merchant
  const parts = []
  
  if (m.show_province && m.province) parts.push(m.province)
  if (m.city) parts.push(m.city)
  if (m.district) parts.push(m.district)
  if (m.address) parts.push(m.address)
  
  return parts.join('')
}

// 获取商家营业时间
const getMerchantBusinessHours = () => {
  if (!card.value || !card.value.merchant) return ''
  
  const m = card.value.merchant
  const hours = []
  
  // 全天营业
  if (m.all_day_start && m.all_day_end) {
    return `全天 ${m.all_day_start} - ${m.all_day_end}`
  }
  
  // 分时段营业
  if (m.morning_start && m.morning_end) {
    hours.push(`上午: ${m.morning_start} - ${m.morning_end}`)
  }
  if (m.afternoon_start && m.afternoon_end) {
    hours.push(`下午: ${m.afternoon_start} - ${m.afternoon_end}`)
  }
  if (m.evening_start && m.evening_end) {
    hours.push(`晚上: ${m.evening_start} - ${m.evening_end}`)
  }
  
  return hours.length > 0 ? hours.join('<br>') : ''
}

// 判断商户是否营业中
const isMerchantOpen = () => {
  if (typeof liveServiceStatus.value?.business_open === 'boolean') {
    return liveServiceStatus.value.business_open
  }
  if (!card.value || !card.value.merchant) return true
  // 默认为true，如果明确为false才显示打烊
  return card.value.merchant.is_open !== false
}

// 获取营业状态颜色
const getBusinessStatusColor = () => {
  return isMerchantOpen() ? 'text-green-500' : 'text-red-500'
}

const getLiveServiceStatusLevel = (status) => {
  return status?.status_level || 'default'
}

const formatLiveServiceCount = (value) => {
  if (!isMerchantOpen()) return '-'
  const count = Number(value || 0)
  return Number.isFinite(count) ? String(count) : '0'
}

const formatLiveServiceSummary = (status) => {
  if (!status) return ''
  if (!isMerchantOpen()) return '门店当前处于非营业状态'
  return status.summary_text || ''
}

const formatLiveServiceWait = (status) => {
  if (!status) return '读取中'
  if (status.estimate?.wait_text) return status.estimate.wait_text
  if (typeof status.estimate?.wait_minutes === 'number') return `${status.estimate.wait_minutes} 分钟`
  return '暂无法估算'
}

const formatLiveServiceRecommendation = (status) => {
  if (!status) return '正在根据门店当前服务进度估算'
  return status.recommendation_text || status.estimate?.recommended_arrival_text || '请以门店现场安排为准'
}

const shouldShowLiveServiceWaitLabel = (status) => {
  const waitMinutes = Number(status?.estimate?.wait_minutes)
  return Number.isFinite(waitMinutes) && waitMinutes >= 1
}

const getLiveServiceEstimateClass = (status) => {
  if (!isMerchantOpen()) return 'is-sample'
  const waitMinutes = Number(status?.estimate?.wait_minutes)
  if (Number.isFinite(waitMinutes) && waitMinutes > 0) return 'is-waiting'
  if (status?.status_level === 'busy') return 'is-waiting'
  return 'is-smooth'
}

const formatLiveQueueCurrentNo = (status) => {
  const no = status?.queue?.current_called_no || 0
  const prefix = status?.queue?.queue_prefix || ''
  return no > 0 ? `${prefix}${no}` : '暂无'
}

const formatLiveServiceGeneratedAt = (value) => {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  return `${hh}:${mm} 更新`
}

const waitForScrollLayout = async () => {
  await nextTick()
  await new Promise(resolve => requestAnimationFrame(() => resolve()))
  await new Promise(resolve => setTimeout(resolve, 100))
}

const getStickyHeaderHeight = () => {
  const header = document.querySelector('header')
  return header?.getBoundingClientRect().height || 0
}

const smoothScrollTo = (top) => {
  window.scrollTo({
    top: Math.max(0, top),
    behavior: 'smooth'
  })
}

const getMaxScrollTop = () => {
  const doc = document.documentElement
  const body = document.body
  const scrollHeight = Math.max(
    doc?.scrollHeight || 0,
    body?.scrollHeight || 0
  )
  const viewportHeight = window.innerHeight || doc?.clientHeight || 0
  return Math.max(0, scrollHeight - viewportHeight)
}

const ensureScrollableSpaceFor = async (targetScrollTop, bufferPx = 24) => {
  const requiredSpacer = Math.max(0, Math.ceil(targetScrollTop - getMaxScrollTop() + bufferPx))
  if (requiredSpacer <= bottomSpacerHeight.value) return
  bottomSpacerHeight.value = requiredSpacer
  await waitForScrollLayout()
}

const scrollElementToViewportTop = async (el, offsetPx = 0) => {
  await waitForScrollLayout()
  if (!el) return

  try {
    const rect = el.getBoundingClientRect()
    const targetScrollTop = rect.top + window.scrollY - offsetPx
    await ensureScrollableSpaceFor(targetScrollTop)
    smoothScrollTo(targetScrollTop)
  } catch (_) {
    try {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    } catch (_) {
      // ignore
    }
  }
}

const scrollToAfterElementBottom = async (el, gapPx = 2) => {
  await waitForScrollLayout()
  if (!el) return

  try {
    const rect = el.getBoundingClientRect()
    const targetScrollTop = rect.bottom + window.scrollY - getStickyHeaderHeight() + gapPx
    await ensureScrollableSpaceFor(targetScrollTop)
    smoothScrollTo(targetScrollTop)
  } catch (_) {
    try {
      el.scrollIntoView({ behavior: 'smooth', block: 'end' })
    } catch (_) {
      // ignore
    }
  }
}

const scrollToNotice = async () => {
  if (usageRecordsSection.value) {
    await scrollToAfterElementBottom(usageRecordsSection.value, 2)
    return
  }
  await scrollElementToViewportTop(noticeAnchor.value, 4)
}

const scrollToUsages = async () => {
  usageRecordsCollapsed.value = false
  await waitForScrollLayout()
  const previousSection = appointmentAnchor.value || cardSummarySection.value
  if (previousSection) {
    await scrollToAfterElementBottom(previousSection, 2)
    return
  }
  await scrollElementToViewportTop(usageRecordsSection.value || usagesAnchor.value, 12)
}

const scrollToAppointment = async () => {
  if (cardSummarySection.value) {
    await scrollToAfterElementBottom(cardSummarySection.value, 2)
    return
  }
  await scrollElementToViewportTop(appointmentAnchor.value, getStickyHeaderHeight())
}

watch(nowTick, () => {
  runUsageDeadlineMonitorTick()
})

onMounted(async () => {
  await fetchCard()
  startNowTickTimer()
  document.addEventListener('visibilitychange', handleVisibilityRefresh)
  // 如果有预约，启动倒计时
  if (appointment.value) {
    startCountdownTimer()
  }
})

onUnmounted(() => {
  stopAutoAssignPoll()
  stopUsageLivePoll()
  stopLiveServicePolling()
  stopNowTickTimer()
  stopCountdownTimer()
  stopVerifyStatusPoll()
  document.removeEventListener('visibilitychange', handleVisibilityRefresh)
  usageDeadlineTriggeredKeys.clear()
  usageDeadlineRetryState.clear()
  if (verifyExpireTimer) {
    clearTimeout(verifyExpireTimer)
    verifyExpireTimer = null
  }
  // 重置底部占位高度
  bottomSpacerHeight.value = 0
})
</script>

<style scoped>
.live-service-section {
  margin-left: -10px;
  margin-right: -10px;
}

.live-service-card {
  border: 1px solid #dbeafe;
  border-radius: 12px;
  background: linear-gradient(180deg, #f8fbff 0%, #ffffff 100%);
  padding: 12px;
}

.live-service-card.is-smooth {
  border-color: #bbf7d0;
  background: linear-gradient(180deg, #f0fdf4 0%, #ffffff 100%);
}

.live-service-card.is-busy {
  border-color: #fed7aa;
  background: linear-gradient(180deg, #fff7ed 0%, #ffffff 100%);
}

.live-service-card.is-paused,
.live-service-card.is-closed,
.live-service-card.is-unavailable,
.live-service-card.is-disabled {
  border-color: #e5e7eb;
  background: #f9fafb;
}

.live-service-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.live-service-title {
  color: #111827;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.5;
}

.live-service-summary {
  color: #6b7280;
  font-size: 13px;
  line-height: 1.6;
  margin-top: 2px;
}

.live-service-refresh {
  flex-shrink: 0;
  border: 0;
  border-radius: 8px;
  background: #eff6ff;
  color: #2563eb;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  padding: 7px 10px;
}

.live-service-refresh:disabled {
  cursor: default;
  opacity: 0.6;
}

.live-service-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.live-service-metrics > div {
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #eef2f7;
  padding: 9px 8px;
}

.live-service-metrics span,
.live-service-estimate span {
  display: block;
  color: #6b7280;
  font-size: 12px;
  line-height: 1.3;
}

.live-service-metrics strong {
  display: block;
  color: #111827;
  font-size: 20px;
  line-height: 1.1;
  margin-top: 5px;
}

.live-service-inline {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.live-service-inline span {
  border-radius: 8px;
  background: #eef2ff;
  color: #4b5563;
  font-size: 12px;
  padding: 5px 8px;
}

.live-service-estimate {
  display: flex;
  align-items: center;
  border-radius: 14px;
  background: #22c55e;
  color: #ffffff;
  margin-top: 10px;
  padding: 10px 12px;
}

.live-service-estimate.is-smooth {
  background: #22c55e;
}

.live-service-estimate.is-sample {
  background: #ef4444;
}

.live-service-estimate.is-waiting {
  background: #f59e0b;
}

.live-service-estimate strong {
  display: block;
  font-size: 20px;
  line-height: 1.1;
  margin-top: 5px;
}

.live-service-estimate p {
  display: inline-flex;
  align-items: center;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.18);
  color: #ffffff;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.4;
  margin: 10px 0 0;
  padding: 5px 10px;
  white-space: nowrap;
}

.live-service-footer {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
  color: #9ca3af;
  font-size: 12px;
  line-height: 1.5;
  margin-top: 10px;
}

.live-service-empty {
  color: #6b7280;
  font-size: 13px;
  line-height: 1.6;
  margin-top: 10px;
}
</style>
