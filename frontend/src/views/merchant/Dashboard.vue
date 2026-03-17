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
    <div v-if="visibleStatsCount > 0" class="px-4 pt-3 pb-3 grid gap-3" :class="{
      'grid-cols-1': visibleStatsCount === 1,
      'grid-cols-2': visibleStatsCount === 2,
      'grid-cols-3': visibleStatsCount === 3
    }">
      <button
        v-if="canDirectSaleManage && merchant.support_direct_sale"
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
        v-if="canVerify"
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-gray-100"
        @click="selectTab('verify')"
      >
        <div class="text-gray-600 text-sm mb-1">今日核销</div>
        <div class="text-3xl font-bold" :class="todayVerifyCount > 0 ? 'text-secondary' : 'text-gray-400'">{{ todayVerifyCount }}</div>
        <div class="text-gray-500 text-sm">次</div>
      </button>
    </div>

    <!-- Tab 切换 -->
    <div class="px-4 flex gap-2 border-b bg-white">
      <button
        v-if="showVerifyTab"
        @click="selectTab('verify')"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'verify'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        扫码核销
      </button>
      <button
        v-if="showAppointmentTab"
        @click="selectTab('appointment')"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'appointment'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        预约
      </button>
      <button
        v-if="showFinishTab"
        @click="selectTab('finish')"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
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
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'notice'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        通知
      </button>
      <button
        v-if="showCardsTab"
        @click="selectTab('cards')"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'cards'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        {{ canVerify ? '卡片' : '售卡' }}
      </button>

      <button
        v-if="showTableTab"
        @click="selectTab('table')"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'table'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        看板
      </button>

      <button
        v-if="showServiceTab"
        @click="selectTab('service')"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'service'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        服务
      </button>
    </div>

    <!-- 看板 -->
    <div v-if="currentTab === 'table' && showTableTab" class="py-2">
      <Table :embedded="true" />
    </div>

    <!-- 预约 -->
    <div v-if="currentTab === 'appointment' && showAppointmentTab" class="px-4 py-4 space-y-4">
      <div v-if="appointmentGroups.length > 0" class="space-y-4">
        <div v-for="group in appointmentGroups" :key="group.key" class="space-y-4">
          <div v-if="group.title" class="px-1 text-sm font-medium text-gray-500">{{ group.title }}</div>
          <div v-for="appt in group.items" :key="appt.id" class="bg-white rounded-xl p-4 shadow-sm">
          <div class="flex justify-between items-start">
            <div>
              <div class="font-medium text-gray-800">{{ appt.user?.nickname || appt.user_id }} <span class="ml-2 text-gray-500 text-sm font-normal">{{ formatAppointmentTechnicianDisplay(appt) }}</span></div>
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
                :class="appt.status === 'arrived' ? 'bg-amber-50 text-amber-700 border border-amber-100' : 'bg-primary-light text-primary border border-primary/10'"
              >
                {{ getAppointmentRiskHint(appt) }}
              </div>
              <div v-if="appt.resolution_note" class="mt-2 rounded-lg px-3 py-2 text-sm bg-gray-50 text-gray-700 border border-gray-100">
                处理备注: {{ appt.resolution_note }}
              </div>
            </div>
            <span :class="getStatusBadgeClass(appt)">
              {{ getStatusText(appt) }}
            </span>
          </div>

          <div class="flex gap-2 mt-3">
            <template v-if="canOperateAppointment(appt)">
              <button
                v-if="appt.status === 'pending' && !isPendingExpired(appt)"
                @click="confirmAppointment(appt.id)"
                class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
              >
                确认预约
              </button>
              <button
                v-if="appt.status === 'pending' && !isPendingExpired(appt)"
                @click="cancelAppointment(appt.id)"
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
              <div v-if="appt.status === 'confirmed'" class="flex-1 py-2 text-primary text-sm font-medium text-center">
                待到店核销
              </div>
              <button
                v-if="appt.status === 'confirmed'"
                @click="cancelAppointment(appt.id)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                取消
              </button>
              <button
                v-if="appt.status === 'arrived'"
                @click="keepWaitingForAppointment(appt)"
                class="flex-1 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm"
              >
                继续等待原客服
              </button>
              <button
                v-if="appt.status === 'arrived' && appt.service_session_id"
                @click="reassignAppointmentService(appt)"
                class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
              >
                改派其他客服
              </button>
              <button
                v-if="['confirmed', 'arrived', 'completed', 'no_show', 'failed'].includes(appt.status)"
                @click="saveAppointmentResolution(appt)"
                class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
              >
                改签/补偿
              </button>
            </template>
          </div>
        </div>
        </div>
      </div>
      <div v-else class="text-center py-12 text-gray-400">
        暂无预约
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
              <div class="text-gray-500 text-sm mt-1">项目：{{ usage.project?.name || '-' }}</div>
              <div class="text-gray-500 text-sm mt-1">状态：{{ getUsageServiceStatusText(usage) }}</div>
              <div v-if="usage.service_technician?.name" class="text-gray-500 text-sm mt-1">
                当前{{ replaceTerms('客服', merchant) }}：{{ usage.service_technician.name }}
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
                {{ getVerifyOperatorInfo(usage).split(' / ')[0] }}
              </div>
              <div v-if="getVerifyOperatorInfo(usage).includes(' / ')" class="text-gray-500 text-xs mt-1">
                {{ getVerifyOperatorInfo(usage).split(' / ')[1] }}
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
              v-else
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
            v-else
            @click="goScanStart"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium"
          >
            {{ getMerchantScanStartLabel() }}
          </button>
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
      <div class="bg-white rounded-xl p-4 shadow-sm">
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
                v-if="getRoomManageServiceCountdownSeconds() !== null"
                :class="['text-sm mt-1 font-medium', getRemainingSecondsClass(getRoomManageServiceCountdownSeconds())]"
              >
                服务倒计时：{{ formatServiceCountdownSeconds(getRoomManageServiceCountdownSeconds()) }}
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
          <div v-for="usage in todayStartUsages" :key="usage.id" class="flex justify-between items-start py-3 border-b last:border-0">
            <div class="flex-1">
              <div class="text-gray-800 font-medium">{{ usage.card?.user?.nickname || '用户' }}</div>
              <div class="text-gray-500 text-sm mt-1">单号：{{ getUsageTrackingNumber(usage) }}</div>
              <div class="text-gray-500 text-sm mt-1">卡号：{{ usage.card?.card_no || '-' }}</div>
              <div class="text-gray-500 text-sm mt-1">项目：{{ formatProjectNameWithDuration(usage.project) }}</div>
              <div class="text-gray-500 text-sm mt-1">状态：{{ getUsageServiceStatusText(usage) }}</div>
              <div
                v-if="getUsageServiceRemainingSeconds(usage) !== null"
                :class="['text-sm mt-1 font-medium', getRemainingSecondsClass(getUsageServiceRemainingSeconds(usage))]"
              >
                服务剩余：{{ formatRemainingSeconds(getUsageServiceRemainingSeconds(usage)) }}
              </div>
              <div class="text-gray-400 text-sm mt-1">{{ formatDateTime(usage.used_at) }}</div>
            </div>
            <div class="text-right">
              <div class="text-sm">
                核销次数：<span class="text-gray-700">{{ getUsageCountDisplayText(usage).totalTimes }}</span> / <span :class="getUsageCountColorClass(usage)">{{ getUsageCountDisplayText(usage).usedTimes }}</span> 次
              </div>
            </div>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
          {{ getMerchantTodayNoStartRecordLabel() }}
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
import { ensureMerchantPermissionsLoaded, merchantApi, appointmentApi, shopApi, attendanceApi, serviceSessionApi, usageApi, noticeApi, cardApi, queueApi } from '../../api'
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

// 统计卡片显示个数
const visibleStatsCount = computed(() => {
  let count = 0
  if (canDirectSaleManage.value && merchant.value.support_direct_sale) count++
  if (showAppointmentSummaryCard.value) count++
  if (canVerify.value) count++
  return count
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

const showAppointmentTab = computed(() => {
  return !!merchant.value?.support_appointment && (canAppointmentView.value || canAppointmentManage.value)
})

const showAppointmentSummaryCard = computed(() => {
  return showAppointmentTab.value && appointmentSummaryCount.value > 0
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
  } else {
    return 'appointment'
  }
}

const getFirstVisibleTab = () => {
  if (showAppointmentTab.value) return 'appointment'
  if (showVerifyTab.value) return 'verify'
  if (showFinishTab.value) return 'finish'
  if (showNoticeTab.value) return 'notice'
  if (showCardsTab.value) return 'cards'
  if (showTableTab.value) return 'table'
  if (showServiceTab.value) return 'service'
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
const unassignedAppointments = computed(() => {
  if (!isTechnicianAuth()) return []
  return (appointments.value || []).filter(a => !a?.technician_id)
})
const assignedAppointments = computed(() => {
  if (!isTechnicianAuth()) return appointments.value || []
  const currentTechnicianId = getTechnicianId()
  if (!currentTechnicianId) return []
  return (appointments.value || []).filter(a => Number(a?.technician_id) === Number(currentTechnicianId))
})
const appointmentGroups = computed(() => {
  const list = appointments.value || []
  if (!isTechnicianAuth()) {
    const active = list.filter(a => !['completed', 'no_show', 'failed'].includes(a?.status))
    const settled = list.filter(a => ['completed', 'no_show', 'failed'].includes(a?.status))
    const groups = []
    if (active.length > 0) groups.push({ key: 'active', title: '进行中', items: active })
    if (settled.length > 0) groups.push({ key: 'settled', title: '已结束', items: settled })
    return groups
  }

  const groups = []
  const myActive = assignedAppointments.value.filter(a => !['completed', 'no_show', 'failed'].includes(a?.status))
  const mySettled = assignedAppointments.value.filter(a => ['completed', 'no_show', 'failed'].includes(a?.status))
  if (unassignedAppointments.value.length > 0) {
    groups.push({ key: 'unassigned', title: '待分配', items: unassignedAppointments.value })
  }
  if (myActive.length > 0) {
    groups.push({
      key: 'assigned',
      title: unassignedAppointments.value.length > 0 ? '我的预约' : '',
      items: myActive
    })
  }
  if (mySettled.length > 0) {
    groups.push({ key: 'assigned-settled', title: '已结束', items: mySettled })
  }
  return groups
})
const appointmentSummaryCount = computed(() => {
  if (!showAppointmentTab.value) return 0
  return (appointments.value || []).filter(a => ['pending', 'confirmed', 'arrived'].includes(a?.status)).length
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
    router.replace({ path: '/merchant', query: { tab: tabParam === 'queue' ? 'appointment' : (tabParam || 'appointment') } })
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
  const sess = serviceSessions.value.find(s => s.technician_id === techId && ['start_pending', 'delay_pending', 'serving', 'auto_finishing'].includes(normalizeSessionStatus(s.status)))
  if (!sess) {
    // 没有活跃会话，返回服务器中的签到状态 idle/paused
    return serverAttendanceStatus.value
  }
  if (normalizeSessionStatus(sess.status) === 'start_pending' && !sess.start_confirmed_at) return 'service_pending_presettlement'
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

const getUsageTrackingNumber = (usage) => {
  if (!usage?.id) return ''
  return String(usage.id).padStart(9, '0')
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
  const totalTimes = card.total_times || 0
  const currentRemainTimes = card.remain_times || 0
  
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
  
  // 其他状态：蓝色
  return 'text-blue-600'
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
    todayVerifyCount.value = res.data.data.today_verify_count || 0
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
    return
  }
  try {
    const res = await appointmentApi.getMerchantAppointments(merchantId.value)
    appointments.value = (res.data.data || [])
      .filter(a => a.status !== 'canceled')
      .sort((a, b) => new Date(a?.appointment_time || 0).getTime() - new Date(b?.appointment_time || 0).getTime())
  } catch (err) {
    console.error('获取预约列表失败:', err)
  }
}

const fetchTodayUsages = async () => {
  try {
    const res = await usageApi.getMerchantUsages(merchantId.value)
    const today = new Date().toISOString().split('T')[0]
    todayUsages.value = (res.data.data || []).filter(u => u.used_at && u.used_at.startsWith(today))
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
    const today = new Date().toISOString().split('T')[0]
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
    const res = await usageApi.getMerchantUsages(merchantId.value)
    const today = new Date().toISOString().split('T')[0]
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

const cancelAppointment = async (id) => {
  if (!confirm('确定要取消这个预约吗？此操作不可撤销。')) {
    return
  }
  
  try {
    await appointmentApi.cancelMerchantAppointment(id)
    fetchAppointments()
    fetchQueueStatus()
    alert('预约已取消')
  } catch (err) {
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
      if (data.session_wait_state === 'appointment_waiting' && Number(data.predicted_wait_minutes || 0) > 0) {
        extraMessages.push(`预约客户已到店，预计等待 ${data.predicted_wait_minutes} 分钟`)
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
    return 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }

  if (appt.status === 'confirmed' && isWriteOffExpired(appt)) {
    return 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }

  const classes = {
    pending: 'px-2 py-1 rounded text-xs font-medium bg-primary-light text-primary',
    confirmed: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-700',
    arrived: 'px-2 py-1 rounded text-xs font-medium bg-amber-50 text-amber-700',
    completed: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-700',
    failed: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500',
    no_show: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500',
    canceled: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
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

  const appointmentTime = new Date(appt.appointment_time).getTime()
  const serviceMinutes = getAppointmentServiceMinutes(appt)
  const deadlineMs = appointmentTime + (serviceMinutes + 30) * 60 * 1000
  return currentTime.value > deadlineMs
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

const getAppointmentCardTypeDisplay = (appt) => {
  return String(appt?.card?.card_type || '').trim()
}

const getAppointmentCardNoDisplay = (appt) => {
  return String(appt?.card?.card_no || '').trim()
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
  const predictedWait = Number(appt?.predicted_wait_minutes || 0)
  if (appt?.status === 'confirmed' && predictedWait > 0) {
    return `该预约属于风险可约，若前序服务压单，预计到店等待 ${predictedWait} 分钟`
  }
  if (appt?.status === 'arrived' && predictedWait > 0) {
    return `前序服务未结束，预计还需等待 ${predictedWait} 分钟，系统已按预约优先等待处理`
  }
  if (appt?.status === 'arrived' && appt?.service_session_id) {
    return '客户已到店，服务会话已绑定到本次预约'
  }
  return ''
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

const saveAppointmentResolution = async (appt) => {
  const initialValue = String(appt?.resolution_note || '').trim()
  const note = window.prompt('请输入改签/补偿/人工处理备注', initialValue)
  if (note === null) return
  const nextNote = String(note || '').trim()
  if (!nextNote) {
    alert('处理备注不能为空')
    return
  }
  try {
    await appointmentApi.updateResolution(appt.id, { resolution_note: nextNote })
    alert('处理备注已保存')
    await fetchAppointments()
  } catch (err) {
    alert(err.response?.data?.error || '保存处理备注失败')
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
})

watch(currentTab, (tab) => {
  const normalizedTab = tab === 'start' ? 'service' : tab
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
  // 倒计时：appointment/verify/service 需要每秒刷新 currentTime
  if (tab === 'appointment' || tab === 'verify' || tab === 'service') {
    startCountdownTimer()
  } else {
    stopCountdownTimer()
  }

  if (tab === 'appointment') {
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
    if (savedTab && ['queue', 'verify', 'appointment', 'start', 'finish', 'notice', 'cards', 'table', 'service'].includes(savedTab)) {
      const normalizedSavedTab = savedTab === 'queue' ? 'appointment' : (savedTab === 'start' ? 'service' : savedTab)
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
  if (tabParam && ['queue', 'verify', 'appointment', 'start', 'finish', 'notice', 'cards', 'table', 'service'].includes(tabParam)) {
    selectTab(tabParam === 'queue' ? 'appointment' : (tabParam === 'start' ? 'service' : tabParam))
  }

  // 检查错误参数，显示错误弹窗
  const errorParam = route.query.error
  if (errorParam) {
    showErrorModalWithMessage(String(errorParam))
    // 清除URL中的错误参数，避免刷新时重复显示
    router.replace({ path: '/merchant', query: { tab: tabParam === 'queue' ? 'appointment' : (tabParam || 'appointment') } })
  }

  const userCodeParam = route.query.user_code
  routeUserCode.value = userCodeParam ? String(userCodeParam) : ''
  scanUserCodeActive.value = !!userCodeParam && String(route.query.from_scan || '') === '1'
  
  const storedMerchantId = getMerchantId()
  if (!storedMerchantId) {
    console.log('No merchantId found, redirecting to login')
    router.replace('/login')
    return
  }

  const parsedMerchantId = Number.parseInt(storedMerchantId, 10)
  if (Number.isNaN(parsedMerchantId) || parsedMerchantId <= 0) {
    console.log('Invalid merchantId:', storedMerchantId, 'redirecting to login')
    router.replace('/login')
    return
  }

  console.log('Valid merchantId:', parsedMerchantId, 'loading data...')
  merchantId.value = parsedMerchantId
  await fetchMerchant()
  console.log('Merchant loaded:', merchant.value)

  await fetchCurrentTechnicianMe()
  nextTick(() => {
    openHandCardModalFromPendingVerify()
  })

  // 根据权限选择默认Tab
  if (!tabParam) {
    // 如果已经从 localStorage 恢复了 tab，并且该 tab 有权限显示，则保持不变
    const restoredTab = currentTab.value
      if (restoredTab && ['verify', 'appointment', 'start', 'finish', 'notice', 'cards', 'table', 'service'].includes(restoredTab)) {
        // 检查恢复的 tab 是否有权限显示
      const canShowRestoredTab = 
        (restoredTab === 'appointment' && showAppointmentTab.value) ||
        (restoredTab === 'verify' && showVerifyTab.value) ||
        (restoredTab === 'finish' && showFinishTab.value) ||
        (restoredTab === 'notice' && showNoticeTab.value) ||
        (restoredTab === 'cards' && showCardsTab.value) ||
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
      finish: showFinishTab.value,
      notice: showNoticeTab.value,
      cards: showCardsTab.value,
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
  loadCardTemplates() // 加载卡片模板
  
  // 如果是技师登录，获取当前签到状态
  if (isTechnicianAuth()) {
    await fetchCurrentAttendanceStatus()
  }

  // 获取叫号状态（有权限时）
  await fetchQueueCallingStatus()
  
  // 根据最终的 currentTab 加载对应的数据
  if (currentTab.value === 'appointment') {
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
    appointment_waiting: '预约优先等待',
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
    const remain = getUsageServiceRemainingSeconds(usage)
    if (remain === null || remain === undefined) continue
    const normalizedRemain = Number(remain)
    if (!Number.isFinite(normalizedRemain)) continue
    items.push({ key: `start_usage:serving:${usageId}`, remain: normalizedRemain })
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
