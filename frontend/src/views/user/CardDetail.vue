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
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <svg :class="isMerchantOpen() ? 'text-green-500' : 'text-red-500'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
              <span class="font-medium text-gray-800">营业时间</span>
            </div>
            <span v-if="!isMerchantOpen()" class="bg-red-500 text-white text-sm font-medium px-3 py-1 rounded">打烊</span>
          </div>
          <div class="text-sm leading-relaxed text-gray-500" v-html="getMerchantBusinessHours()"></div>
        </div>
      </div>
    </div>

    <!-- 预约状态展示：仅保留查看，不再在详情页提供入口操作 -->
    <div v-if="card.merchant?.support_appointment && !card?.locked && appointment" class="px-4 mt-4">
      <div ref="appointmentAnchor" class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center gap-2 mb-3">
          <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <span class="font-medium text-gray-800">预约排队</span>
        </div>

        <div class="space-y-3">
          <div class="flex justify-between items-center">
            <div>
              <span class="text-gray-500">我的预约</span>
              <span class="ml-2 text-gray-600 text-sm">
                {{ appointment.technician ? ((appointment.technician.service_role?.name || '客服') + '：' + appointment.technician.name) : '待分配' }}
              </span>
            </div>
            <span :class="getAppointmentStatusClass(appointment.status)">
              {{ getAppointmentStatusText(appointment.status) }}
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
              <div v-if="appointment.status === 'confirmed'" class="text-sm text-gray-600 mb-1">距待开始</div>
              <div v-if="!isAppointmentPassed()" :class="getCountdownClass()" class="text-sm font-medium">
                {{ getCountdownText() }}
              </div>
              <div v-else class="text-sm text-gray-400">
                预约已过
              </div>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-4 pt-3 mt-3 border-t border-gray-100">
            <div>
              <div class="text-gray-400 text-xs">前面排队</div>
              <div class="text-2xl font-bold text-gray-800">{{ queueBefore }}<span class="text-sm font-normal">人</span></div>
            </div>
            <div>
              <div class="text-gray-400 text-xs">预计等待</div>
              <div class="text-2xl font-bold text-gray-800">{{ estimatedMinutes }}<span class="text-sm font-normal">分钟</span></div>
            </div>
          </div>
          <p class="text-xs text-gray-400">* 排队进度由商户服务确认后即时更新</p>

          <button
            @click="cancelAppointment"
            :disabled="cancelButtonDisabled"
            class="w-full py-2.5 border-2 border-red-400 text-red-500 font-medium rounded-lg hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors mt-3"
          >
            {{ cancelButtonText }}
          </button>
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
        <div v-if="!usageRecordsCollapsed && usages.length > 0">
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
              <div class="text-gray-400 text-sm mt-0.5">
                单号：{{ getUsageTrackingNumber(usage) }}
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
              <div v-if="getUsageRoomInfo(usage)" class="text-gray-400 text-sm mt-0.5">
                {{ getUsageRoomInfo(usage) }}
              </div>
              <div v-if="getUsageWindowInfo(usage)" class="text-gray-400 text-sm mt-0.5">
                {{ getUsageWindowInfo(usage) }}
              </div>
              <div v-if="getUsageProjectText(usage)" class="text-gray-400 text-sm mt-0.5">
                {{ getUsageProjectText(usage) }}
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
            </div>
            <div v-if="getUsageOperatorInfo(usage)" class="col-span-2 flex items-center justify-between text-gray-400 text-sm mt-0.5">
              <span v-if="getUsageOperatorInfo(usage)">{{ getUsageOperatorInfo(usage) }}</span>
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

    <!-- 预约时间选择弹窗 -->
    <div v-if="showModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="closeModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg max-h-[80vh] overflow-hidden flex flex-col">
        <!-- 弹窗头部 -->
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between flex-shrink-0">
          <h3 class="font-medium text-lg">选择预约时间</h3>
          <button @click="closeModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <!-- 可滚动内容区域 -->
        <div class="overflow-y-auto flex-1">
          <!-- 项目选择（先选项目，再选时间） -->
          <div class="px-5 py-3 border-b">
            <div class="text-sm font-medium text-gray-700 mb-2">选择项目</div>
            <div v-if="!card.projects || card.projects.length === 0" class="text-gray-400 text-sm">暂无可选项目</div>
            <div v-else class="space-y-2">
              <label v-for="p in card.projects" :key="p.id" class="flex items-center gap-3">
                <input type="radio" name="appt_project" :value="p.id" v-model="selectedAppointmentProjectId" />
                <div class="flex-1">
                  <div class="text-gray-800">{{ p.name }}</div>
                  <div v-if="p.duration" class="text-gray-400 text-xs">时长 {{ p.duration }} 分钟</div>
                </div>
              </label>
            </div>
          </div>

          <!-- 专业客服（可选）：仅在商户开启客服且有启用客服时展示；与时间段双向联动 -->
          <div v-if="availableTechnicians.length > 0" class="px-5 py-3 border-b">
            <div class="text-sm font-medium text-gray-700 mb-2">选择专业客服</div>
            <div class="flex flex-wrap gap-2">
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
          </div>

          <!-- 时间段列表 -->
          <div class="px-5 py-4">
            <div v-if="loadingSlots" class="text-center py-8 text-gray-400">
              加载中...
            </div>
            <div v-else-if="timeSlotError" class="text-center py-8 text-gray-400">
              {{ timeSlotError }}
            </div>
            <div v-else-if="timeSlots.length === 0" class="text-center py-8 text-gray-400">
              请先选择预约项目
            </div>
            <div v-else-if="displayedTimeSlots.length === 0" class="text-center py-8 text-gray-400">
              当前所选专业客服无可用时间段
            </div>
            <div v-else class="grid grid-cols-2 gap-3">
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
                <div>{{ formatTime(slot.time) }}</div>
              </button>
            </div>
          </div>
        </div>

        <!-- 弹窗底部 -->
        <div class="px-5 py-4 border-t flex-shrink-0 bg-white">
          <button
            @click="confirmAppointment"
            :disabled="!selectedAppointmentProjectId || !selectedTimeSlot || appointing"
            class="w-full py-3 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {{ appointing ? '预约中...' : '确认预约' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { cardApi, usageApi, noticeApi, appointmentApi } from '../../api'
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
const queueBefore = ref(0)
const estimatedMinutes = ref(0)
const countdown = ref(0)
const usageRecordsCollapsed = ref(false)
const visibleUsageCount = ref(10)
let countdownTimer = null


const verifyCode = ref('')
const codeExpireTime = ref('')
const generating = ref(false)
const verifyQrDataUrl = ref('')
const verifyCodeProject = ref(null) // 当前核销码对应的项目
let verifyExpireTimer = null

let verifyStatusPollTimer = null
const verifyStatusChecking = ref(false)
const hasJumpedToRoomSelect = ref(false)
const showProjectModal = ref(false)
const selectedProjectId = ref(null)
const appointing = ref(false)
const canceling = ref(false)

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

const getStartPendingTimeoutMs = (usage) => {
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
  const remainSeconds = getUsageSessionStartPendingRemainingSeconds(usage)
  if (remainSeconds > 0) {
    const snapshotAt = Number(usagesSnapshotAtMs.value || 0)
    const elapsed = snapshotAt > 0 ? nowMs - snapshotAt : 0
    const remain = remainSeconds * 1000 - (Number.isFinite(elapsed) ? elapsed : 0)
    if (remain > 0) return remain
    return 0
  }

  const baseMs = getUsageSessionUpdatedAtMs(usage)
  if (!baseMs) return 0
  const timeoutSeconds = Number(usage?.service_session_start_pending_timeout_seconds || 0) > 0
    ? Number(usage?.service_session_start_pending_timeout_seconds)
    : 180
  const deadlineMs = baseMs + timeoutSeconds * 1000
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

const getUsageStaffSelectAutoAssignDeadlineAtMs = (usage) => {
  const baseRaw = usage?.staff_select_entered_at || usage?.room_locked_at
  if (!baseRaw) return 0
  const baseMs = new Date(baseRaw).getTime()
  if (!Number.isFinite(baseMs) || baseMs <= 0) return 0
  return baseMs + 5 * 60 * 1000
}

const getUsageStartTimeoutAutoAssignDeadlineAtMs = (usage) => {
  const raw = usage?.staff_select_entered_at
  if (!raw) return 0
  const enteredAtMs = new Date(raw).getTime()
  if (!Number.isFinite(enteredAtMs) || enteredAtMs <= 0) return 0
  return enteredAtMs + 5 * 60 * 1000
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
  if (!usage?.id) return ''
  return String(usage.id).padStart(9, '0')
}

const getUsageSequence = (index) => {
  if (!card.value || !usages.value || !usages.value[index]) return 0
  if (usages.value[index].status === 'failed') return '-'
  
  let seq = (card.value.total_times || 0) - (card.value.remain_times || 0)
  for (let i = 0; i < index; i++) {
    const u = usages.value[i]
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
  if (!shouldLivePollUsages()) return
  fetchUsages()
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

    // 自动分配客服倒计时（5分钟）
    // 优先使用 staff_select_entered_at（用户进入选择客服页的时间），否则使用 room_locked_at（锁房时间）
    const baseMs = usage?.staff_select_entered_at
      ? new Date(usage.staff_select_entered_at).getTime()
      : usage?.room_locked_at
      ? new Date(usage.room_locked_at).getTime()
      : 0
    if (!baseMs || Number.isNaN(baseMs)) return ''
    const deadline = baseMs + 5 * 60 * 1000
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

  // 上钟超时后重新选择客服：如果已经开始计时（staff_select_entered_at）则展示5分钟自动分配倒计时
  if (supportCS && isUsageStartTimeout(usage)) {
    const enteredAtMs = usage?.staff_select_entered_at ? new Date(usage.staff_select_entered_at).getTime() : 0
    if (!enteredAtMs || Number.isNaN(enteredAtMs)) return ''
    const deadline = enteredAtMs + 5 * 60 * 1000
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

    const deadline = enteredAtMs + 5 * 60 * 1000
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

// 预约弹窗相关
const showModal = ref(false)
const selectedDate = ref('')
const selectedTimeSlot = ref('')
const timeSlots = ref([])
const timeSlotError = ref('')
const loadingSlots = ref(false)

const selectedAppointmentProjectId = ref(null)

const availableTechnicians = ref([])
const selectedTechnicianId = ref(null)

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

const visibleUsages = computed(() => {
  return (usages.value || []).slice(0, visibleUsageCount.value)
})

const hasMoreUsages = computed(() => {
  return visibleUsageCount.value < (usages.value?.length || 0)
})

const getUsageOperatorInfo = (usage) => {
  // 如果已结单，只显示服务人员
  if (usage.status === 'success' && usage.finished_at) {
    if (usage.technician) {
      return `服务人员：${usage.technician.name || usage.technician.account || '技师'}`
    }
    // 结单完成且没有技师信息（商户老板操作），不显示任何信息
    return ''
  }
  
  // 结单前，显示核销人员信息
  if (usage.technician) {
    // 技师操作：显示技师姓名或账号
    return `核销人员：${usage.technician.name || usage.technician.account || '技师'}`
  } else if (usage.merchant) {
    // 商户老板操作：显示店名
    return `核销人员：${usage.merchant.name || '店铺'}`
  }
  
  return ''
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
  visibleUsageCount.value = Math.min(visibleUsageCount.value + 10, usages.value.length)
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
    }

    fetchUsages()
    fetchAppointment()
  } catch (err) {
    console.error('获取卡片详情失败:', err)
  }
}

const fetchUsages = async () => {
  if (fetchUsagesPromise) return fetchUsagesPromise

  fetchUsagesPromise = (async () => {
    try {
      const res = await usageApi.getCardUsages(route.params.id)
      const allUsages = res.data.data || []
      usagesSnapshotAtMs.value = Date.now()

      // 过滤掉超过12小时的失败记录
      const now = Date.now()
      usages.value = allUsages.filter(u => {
        if (u.status !== 'failed') return true
        if (!u.used_at) return true
        const usedAtMs = new Date(u.used_at).getTime()
        const diffHours = (now - usedAtMs) / (1000 * 60 * 60)
        return diffHours <= 12
      })
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

      if (route.query.scrollToUsages === '1' && usages.value.length > 0) {
        await scrollToUsages()
      }
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
      queueBefore.value = data.queue_before || 0
      estimatedMinutes.value = data.estimated_minutes || 0
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
    queueBefore.value = 0
    estimatedMinutes.value = 0
    stopCountdownTimer()
  } catch (err) {
    console.error('获取预约信息失败:', err)
    console.error('错误详情:', err.response?.data)
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
  return canceling.value || appointment.value.status === 'finished' || appointment.value.status === 'canceled'
})

const cancelButtonText = computed(() => {
  if (isAppointmentFailed.value) return appointment.value?.failed_reason ? `预约失败：${appointment.value.failed_reason}` : '预约失败'
  return canceling.value ? '取消中...' : '取消预约'
})

const closeProjectModal = () => {
  showProjectModal.value = false
}

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

const doGenerateVerifyCode = async (projectId) => {
  const payload = projectId ? { project_id: projectId } : undefined
  const res = await cardApi.generateVerifyCode(route.params.id, payload)
  verifyCode.value = res.data.data.code
  const expireAt = new Date(res.data.data.expire_at * 1000)
  codeExpireTime.value = expireAt.toLocaleTimeString()

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
  } catch (err) {
    alert(err.response?.data?.error || '生成核销码失败')
  } finally {
    generating.value = false
  }
}

// 显示预约弹窗
const showAppointmentModal = async () => {
  if (!card.value || !card.value.merchant_id) {
    alert('卡片信息加载中，请稍后再试')
    return
  }
  
  showModal.value = true
  selectedTechnicianId.value = null
  selectedDate.value = getTomorrowDate()
  selectedAppointmentProjectId.value = null
  selectedTimeSlot.value = ''
  timeSlots.value = []
  timeSlotError.value = ''
  availableTechnicians.value = []
}

// 关闭弹窗
const closeModal = () => {
  showModal.value = false
  selectedTechnicianId.value = null
  selectedDate.value = ''
  selectedAppointmentProjectId.value = null
  selectedTimeSlot.value = ''
  timeSlots.value = []
  timeSlotError.value = ''
  availableTechnicians.value = []
}

const onAppointmentProjectChange = async () => {
  selectedTimeSlot.value = ''
  timeSlots.value = []
  timeSlotError.value = ''
  if (!selectedAppointmentProjectId.value) return
  await loadTimeSlots(selectedDate.value)
}

watch(selectedAppointmentProjectId, () => {
  onAppointmentProjectChange()
})

// 获取明天日期
const getTomorrowDate = () => {
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  return tomorrow.toISOString().slice(0, 10)
}

// 加载可用时间段
const loadTimeSlots = async (date) => {
  if (!card.value.merchant_id) {
    console.error('商户ID不存在')
    return
  }
  if (!selectedAppointmentProjectId.value) {
    timeSlots.value = []
    timeSlotError.value = ''
    return
  }
  
  loadingSlots.value = true
  timeSlotError.value = ''
  try {
    console.log('正在获取时间段，商户ID:', card.value.merchant_id, '日期:', date)
    const res = await appointmentApi.getAvailableTimeSlots(card.value.merchant_id, date, selectedAppointmentProjectId.value)
    console.log('获取时间段响应:', res.data)
    timeSlots.value = res.data.data.time_slots || []

    availableTechnicians.value = res.data.data.technicians || []

  } catch (err) {
    console.error('获取可用时间段失败:', err)
    console.error('错误详情:', err.response?.data)
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
  // 若当前已选时间段不支持该客服，则清空时间段
  if (selectedTimeSlot.value) {
    const slot = (timeSlots.value || []).find(s => s && s.time === selectedTimeSlot.value)
    const ids = Array.isArray(slot?.technician_ids) ? slot.technician_ids : []
    if (ids.length > 0 && !ids.includes(next)) {
      selectedTimeSlot.value = ''
    }
  }
}

// 选择时间段
const selectTimeSlot = (slot) => {
  selectedTimeSlot.value = slot.time
  // 若尚未选择客服，则联动上方客服列表（通过 displayedTechnicians 计算属性实现）
  if (selectedTechnicianId.value) {
    const ids = Array.isArray(slot?.technician_ids) ? slot.technician_ids : []
    if (ids.length > 0 && !ids.includes(selectedTechnicianId.value)) {
      selectedTechnicianId.value = null
    }
  }
}

// 格式化时间显示
const formatTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  return `${hours}:${minutes}`
}

// 确认预约
const confirmAppointment = async () => {
  if (!selectedTimeSlot.value || appointing.value) return

  if (!selectedAppointmentProjectId.value) {
    alert('请选择项目')
    return
  }

  if ((availableTechnicians.value || []).length > 0 && !selectedTechnicianId.value) {
    const ok = window.confirm('你未选择客服，系统稍后将自动分配客服')
    if (!ok) return
  }

  appointing.value = true
  try {
    const userId = localStorage.getItem('userId')
    if (!userId) {
      alert('请先登录')
      router.push('/login')
      return
    }
    
    await appointmentApi.createAppointment({
      card_id: Number(route.params.id),
      merchant_id: card.value.merchant_id,
      user_id: parseInt(userId),
      project_id: Number(selectedAppointmentProjectId.value),
      technician_id: selectedTechnicianId.value ? Number(selectedTechnicianId.value) : null,
      appointment_time: selectedTimeSlot.value
    })
    
    closeModal()
    await fetchAppointment()
    alert('预约成功！')
  } catch (err) {
    alert(err.response?.data?.error || '预约失败')
  } finally {
    appointing.value = false
  }
}

const cancelAppointment = async () => {
  if (canceling.value) return

  if (isAppointmentFailed.value) return
  
  if (!confirm('确定要取消预约吗？')) return
  
  canceling.value = true
  try {
    await appointmentApi.cancelAppointment(appointment.value.id)
    
    appointment.value = null
    queueBefore.value = 0
    estimatedMinutes.value = 0
    stopCountdownTimer()
    
    alert('已取消预约')
  } catch (err) {
    alert(err.response?.data?.error || '取消预约失败')
  } finally {
    canceling.value = false
  }
}

const getAppointmentStatusClass = (status) => {
  const classes = {
    pending: 'text-primary',
    confirmed: 'text-primary',
    finished: 'text-gray-600',
    canceled: 'text-gray-400'
  }
  return classes[status] || 'text-gray-500'
}

const getAppointmentStatusText = (status) => {
  const texts = {
    pending: '待确认',
    confirmed: '排队中',
    finished: '已完成',
    canceled: '已取消'
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

// 判断预约是否已过（超过预约时间1分钟）
const isAppointmentPassed = () => {
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

// 获取倒计时文字
const getCountdownText = () => {
  if (countdown.value <= 0 && countdown.value > -60) {
    return '预约时间已到'
  }
  
  const totalSeconds = Math.abs(countdown.value)
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
  // 1. 预约状态必须是已确认(confirmed)
  // 2. 当前时间距离预约时间小于等于5分钟（即倒计时 <= 300秒 且 > -60秒）
  if (appointment.value.status === 'confirmed') {
    // countdown.value > 0 表示还没到预约时间
    // countdown.value <= 300 表示距离预约时间小于等于5分钟
    // countdown.value > -60 表示还没有超过预约时间1分钟
    return countdown.value <= 300 && countdown.value > -60
  }
  
  // 其他状态（pending, finished, canceled）不显示核销码
  return false
}

// 获取商家地址
const getMerchantAddress = () => {
  if (!card.value || !card.value.merchant) return ''
  
  const m = card.value.merchant
  const parts = []
  
  if (m.province) parts.push(m.province)
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
    return `全天营业: ${m.all_day_start} - ${m.all_day_end}`
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
  if (!card.value || !card.value.merchant) return true
  // 默认为true，如果明确为false才显示打烊
  return card.value.merchant.is_open !== false
}

// 获取营业状态颜色
const getBusinessStatusColor = () => {
  return isMerchantOpen() ? 'text-green-500' : 'text-red-500'
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
