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

    <!-- 营业状态按钮 -->
    <div class="px-4 pt-4">
      <button
        v-if="canBusinessStatusUpdate"
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
    <div class="px-4 py-4 grid gap-3" :class="{
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
        v-if="showQueueTab"
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-gray-100"
        @click="currentTab = 'queue'"
      >
        <div class="text-gray-600 text-sm mb-1">待处理预约</div>
        <div class="text-3xl font-bold" :class="pendingAppointments > 0 ? 'text-orange-500' : 'text-gray-400'">{{ pendingAppointments }}</div>
        <div class="text-gray-500 text-sm">人</div>
      </button>
      <button
        v-if="canVerify"
        type="button"
        class="bg-white rounded-xl p-4 text-left border border-gray-100"
        @click="currentTab = 'verify'"
      >
        <div class="text-gray-600 text-sm mb-1">今日核销</div>
        <div class="text-3xl font-bold" :class="todayVerifyCount > 0 ? 'text-secondary' : 'text-gray-400'">{{ todayVerifyCount }}</div>
        <div class="text-gray-500 text-sm">次</div>
      </button>
    </div>

    <!-- Tab 切换 -->
    <div class="px-4 flex gap-2 border-b bg-white">
      <button
        v-if="showQueueTab"
        @click="currentTab = 'queue'"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'queue'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        排队
      </button>
      <button
        v-if="showVerifyTab"
        @click="currentTab = 'verify'"
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
        v-if="showFinishTab"
        @click="currentTab = 'finish'"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'finish'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        扫码结单
      </button>
      <button
        v-if="showNoticeTab"
        @click="currentTab = 'notice'"
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
        @click="currentTab = 'cards'"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          currentTab === 'cards'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500'
        ]"
      >
        卡片
      </button>
    </div>

    <!-- 排队管理 -->
    <div v-if="currentTab === 'queue' && showQueueTab" class="px-4 py-4 space-y-4">
      <div v-for="appt in appointments" :key="appt.id" class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex justify-between items-start mb-2">
          <div>
            <div class="font-medium text-gray-800">用户 ID: {{ appt.user?.nickname || appt.user_id }}</div>
            <div class="text-gray-500 text-sm">预约时间: {{ formatDateTime(appt.appointment_time) }}</div>
            <!-- 待确认预约的倒计时 -->
            <div v-if="appt.status === 'pending' && getPendingCountdown(appt) !== null" :class="getPendingCountdownClass(appt)">
              {{ getPendingCountdownDisplay(appt) }}
            </div>
            <!-- 已确认预约的倒计时 -->
            <div v-if="appt.status === 'confirmed' && getAppointmentCountdown(appt) !== null && !isServiceTimeExpired(appt)" :class="getCountdownClass(appt)">
              {{ getCountdownDisplay(appt) }}
            </div>
          </div>
          <span :class="getStatusBadgeClass(appt)">
            {{ getStatusText(appt) }}
          </span>
        </div>
        
        <div class="flex gap-2 mt-3">
          <button
            v-if="appt.status === 'pending' && !isPendingExpired(appt)"
            @click="confirmAppointment(appt.id)"
            class="flex-1 py-2 bg-primary text-white rounded-lg text-sm font-medium"
          >
            确认预约
          </button>
          <button
            v-if="appt.status === 'pending' && isPendingExpired(appt)"
            disabled
            class="flex-1 py-2 bg-gray-100 text-gray-400 rounded-lg text-sm font-medium cursor-not-allowed"
          >
            未确认预约
          </button>
          <button
            v-if="shouldShowFinishButton(appt) && !isWriteOffExpired(appt)"
            @click="finishAppointment(appt.id)"
            class="flex-1 py-2 bg-gray-900 text-white rounded-lg text-sm font-medium"
          >
            完成服务 (扣次)
          </button>
          <button
            v-if="appt.status === 'confirmed' && isWriteOffExpired(appt)"
            disabled
            class="flex-1 py-2 bg-gray-100 text-gray-400 rounded-lg text-sm font-medium cursor-not-allowed"
          >
            未核销
          </button>
          <button
            v-if="appt.status !== 'finished' && appt.status !== 'canceled'"
            @click="cancelAppointment(appt.id)"
            class="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm"
          >
            取消
          </button>
        </div>
      </div>

      <div v-if="appointments.length === 0" class="text-center py-12 text-gray-400">
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
          <div v-for="usage in todayUsages" :key="usage.id" class="flex justify-between items-center py-2 border-b last:border-0">
            <div>
              <div class="text-gray-800">{{ usage.card?.user?.nickname || '用户' }}</div>
              <div class="text-gray-400 text-sm">{{ formatDateTime(usage.used_at) }}</div>
            </div>
            <span class="text-gray-700 text-sm">核销 {{ usage.used_times }} 次</span>
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
          扫码结单
        </button>
      </div>

      <!-- 今日结单记录 -->
      <div class="bg-white rounded-xl p-4 shadow-sm mt-4">
        <h3 class="font-medium text-gray-800 mb-4">今日结单记录</h3>
        <div v-if="todayUsages.length > 0" class="space-y-3">
          <div v-for="usage in todayUsages" :key="usage.id" class="flex justify-between items-center py-2 border-b last:border-0">
            <div>
              <div class="text-gray-800">{{ usage.card?.user?.nickname || '用户' }}</div>
              <div class="text-gray-400 text-sm">{{ formatDateTime(usage.used_at) }}</div>
            </div>
            <span class="text-gray-700 text-sm">结单 {{ usage.used_times }} 次</span>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
          今日暂无结单
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
      <div v-if="cardsError" class="bg-gray-50 border border-gray-100 text-gray-700 rounded-lg p-3 text-sm mb-4">
        {{ cardsError }}
      </div>

      <div v-if="routeUserCode" ref="userCodeAnchor" class="bg-white rounded-xl p-4 shadow-sm mb-4 flex items-center justify-between">
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
            <option value="">全部卡片类型</option>
            <option
              v-for="tpl in cardTemplates"
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

      <div v-if="currentDisplay === 'cards' && cardsLoading" class="text-center py-12 text-gray-400">
        加载中...
      </div>

      <div v-else-if="currentDisplay === 'cards'">
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
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, onActivated, watch, nextTick, computed } from 'vue'
import { useRouter, useRoute, onBeforeRouteLeave } from 'vue-router'
import { ensureMerchantPermissionsLoaded, merchantApi, cardApi, appointmentApi, noticeApi, usageApi, shopApi } from '../../api'
import { formatDateTime, formatDate } from '../../utils/dateFormat'
import QRCode from 'qrcode'

import { getMerchantId, hasMerchantPermission } from '../../utils/auth'

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

const canBusinessStatusUpdate = computed(() => hasMerchantPermission('merchant.business_status.update'))
const canDirectSaleManage = computed(() => hasMerchantPermission('merchant.direct_sale.manage'))
const canCardIssue = computed(() => hasMerchantPermission('merchant.card.issue'))
const canFinishVerify = computed(() => hasMerchantPermission('merchant.card.finish'))
const canVerify = computed(() => hasMerchantPermission('merchant.card.verify'))
const canNoticeManage = computed(() => hasMerchantPermission('merchant.notice.manage'))
const canAppointmentView = computed(() => hasMerchantPermission('merchant.appointment.view'))
const canAppointmentManage = computed(() => hasMerchantPermission('merchant.appointment.manage'))

// 统计卡片显示个数
const visibleStatsCount = computed(() => {
  let count = 0
  if (canDirectSaleManage.value && merchant.value.support_direct_sale) count++
  if (showQueueTab.value) count++
  if (canVerify.value) count++
  return count
})

// Tab 显示控制
const showQueueTab = computed(() => {
  // 有预约权限或管理预约权限，并且商户支持预约
  const hasPermission = canAppointmentView.value || canAppointmentManage.value
  const merchantSupportsAppointment = merchant.value && merchant.value.support_appointment === true
  console.log('showQueueTab:', { hasPermission, merchantSupportsAppointment, merchant: merchant.value })
  return merchantSupportsAppointment && hasPermission
})

const showVerifyTab = computed(() => {
  // 有核销权限
  console.log('showVerifyTab:', canVerify.value)
  return canVerify.value
})

const showFinishTab = computed(() => {
  // 有结单权限
  console.log('showFinishTab:', canFinishVerify.value)
  // 若同时拥有核销+结单权限，则只显示扫码核销页（结单走扫码智能逻辑）
  return canFinishVerify.value && !canVerify.value
})

const showNoticeTab = computed(() => {
  // 有通知管理权限
  return canNoticeManage.value
})

const canCardSell = computed(() => hasMerchantPermission('merchant.card.sell'))

const showCardsTab = computed(() => {
  return canVerify.value || canCardSell.value
})
const currentTab = ref('queue')
const routeUserCode = ref('')
const userCodeAnchor = ref(null)

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
  // 如果是技师登录，显示技师账号
  if (isTechnicianAuth()) {
    const technicianAccount = sessionStorage.getItem('technicianAccount')
    if (technicianAccount) return `技师: ${technicianAccount}`
    const technicianCode = sessionStorage.getItem('technicianCode')
    if (technicianCode) return `技师: ${technicianCode}`
    return '技师'
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
const todayUsages = ref([])
const notices = ref([])
const currentTime = ref(Date.now())
let countdownTimer = null

const verifyCodeInput = ref('')
const verifying = ref(false)
const verifyResult = ref(null)
const showVerifyInput = ref(false)

const noticeForm = ref({
  title: '',
  content: ''
})

const currentView = ref('cards') // 'cards' | 'sellTemplates'
const issuedCards = ref([])
const sellTemplates = ref([])
const displayMode = ref('auto') // 'auto' | 'cards' | 'sellTemplates'
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
  // 如果手动指定了显示模式，优先使用
  if (displayMode.value !== 'auto') {
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
    await fetchIssuedCards()
  }
)

const cardTemplates = ref([])

const showBusinessStatusModal = ref(false)

const getCardTypeLabel = (type) => {
  const labels = { times: '次数卡', lesson: '课时卡', balance: '充值卡' }
  return labels[type] || type
}

const goScanVerify = () => {
  router.push('/merchant/scan-verify')
}

const goScanFinish = () => {
  router.push('/merchant/scan-verify?mode=finish')
}

const onTopScanClick = () => {
  if (Date.now() < suppressTopScanClickUntil.value) return
  // 顶部扫码入口也按同样规则：
  // - 只有结单权限：进入结单模式（只结单，不核销）
  // - 同时有核销+结单：进入智能模式（优先核销，满足条件才结单）
  // - 只有核销：进入核销模式
  if (!canVerify.value && canFinishVerify.value) {
    goScanFinish()
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

const fetchMerchant = async () => {
  try {
    const res = await merchantApi.getMerchant(merchantId.value)
    merchant.value = res.data.data
  } catch (err) {
    console.error('获取商户信息失败:', err)
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
  await fetchIssuedCards()
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
  if (!merchant.value.support_direct_sale) {
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
    appointments.value = (res.data.data || []).filter(a => a.status !== 'finished' && a.status !== 'canceled')
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

const finishAppointment = async (id) => {
  try {
    await appointmentApi.finishAppointment(id)
    fetchAppointments()
    fetchQueueStatus()
  } catch (err) {
    alert(err.response?.data?.error || '完成失败')
  }
}

const cancelAppointment = async (id) => {
  if (!confirm('确定要取消这个预约吗？此操作不可撤销。')) {
    return
  }
  
  try {
    await appointmentApi.cancelAppointment(id)
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
    verifyResult.value = {
      success: true,
      message: `核销成功！剩余次数: ${res.data.data.remain_times}`
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
    finished: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-700',
    canceled: 'px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-500'
  }
  return classes[appt.status] || ''
}

const getStatusText = (appt) => {
  if (!appt) return ''

  if (appt.status === 'pending' && isPendingExpired(appt)) {
    return '过期未确认'
  }

  if (appt.status === 'confirmed' && isWriteOffExpired(appt)) {
    return '已过服务时间'
  }

  const texts = {
    pending: '待确认',
    confirmed: '排队中',
    finished: '已完成',
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
  let serviceMinutes = merchant.value.avg_service_minutes
  if (!serviceMinutes || serviceMinutes <= 0) serviceMinutes = 30
  const deadlineMs = appointmentTime + (serviceMinutes + 30) * 60 * 1000
  return currentTime.value > deadlineMs
}

// 判断是否已过服务时间（不显示倒计时）
const isServiceTimeExpired = (appt) => {
  if (!appt || appt.status !== 'confirmed' || !appt.appointment_time) return false

  const appointmentTime = new Date(appt.appointment_time).getTime()
  let serviceMinutes = merchant.value.avg_service_minutes
  if (!serviceMinutes || serviceMinutes <= 0) serviceMinutes = 30
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
  if (countdown === null) return 'text-primary text-sm font-medium mt-1'

  // 统一色相：临近用主色，其余用灰
  if (countdown <= 600) {
    return 'text-primary text-sm font-medium mt-1'
  }
  return 'text-gray-500 text-sm font-medium mt-1'
}

// 判断是否应该显示完成服务按钮
const shouldShowFinishButton = (appt) => {
  if (appt.status !== 'confirmed') return false
  if (!appt.appointment_time) return false
  
  const appointmentTime = new Date(appt.appointment_time).getTime()
  const now = currentTime.value
  const elapsed = now - appointmentTime // 已过的时间（毫秒）
  
  // 需要过了预约时间 + 服务时长 - 1分钟 才显示按钮
  let serviceMinutes = merchant.value.avg_service_minutes
  if (!serviceMinutes || serviceMinutes <= 0) serviceMinutes = 30
  const requiredTime = (serviceMinutes - 1) * 60 * 1000
  
  return elapsed >= requiredTime
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

watch(currentTab, (tab) => {
  if (tab !== 'cards' && scanUserCodeActive.value) {
    scanUserCodeActive.value = false
    routeUserCode.value = ''
  }
  if (tab === 'queue') {
    fetchAppointments()
    startCountdownTimer()
  } else {
    stopCountdownTimer()
    if (tab === 'verify') {
      // 重置为默认状态
      showVerifyInput.value = false
      verifyCodeInput.value = ''
      verifyResult.value = null
      fetchTodayUsages()
    } else if (tab === 'finish') {
      // 结单Tab也显示今日核销记录
      fetchTodayUsages()
    } else if (tab === 'cards') {
      // 重置显示模式为自动，让computed决定显示什么
      displayMode.value = 'auto'
      // 如果默认显示售卡模板，则加载售卡模板数据
      if (currentDisplay.value === 'sellTemplates') {
        loadSellTemplates()
      } else {
        fetchIssuedCards()
      }
    } else if (tab === 'notice') {
      fetchNotices()
    }
  }
})

watch(
  () => route.query.user_code,
  async (v) => {
    if (v) {
      routeUserCode.value = String(v)
      scanUserCodeActive.value = String(route.query.from_scan || '') === '1'
      if (currentTab.value === 'cards') {
        await fetchIssuedCards()
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
        await fetchIssuedCards()
      }
    }
  }
)

onMounted(async () => {
  console.log('Merchant Dashboard mounted')
  console.log('localStorage merchantId:', localStorage.getItem('merchantId'))

  // 等待权限加载完成
  await ensureMerchantPermissionsLoaded()
  console.log('Permissions loaded, checking permissions:', {
    canAppointmentView: canAppointmentView.value,
    canAppointmentManage: canAppointmentManage.value,
    canVerify: canVerify.value,
    canFinishVerify: canFinishVerify.value
  })
  
  // 检查查询参数，自动切换到指定Tab
  const tabParam = route.query.tab
  if (tabParam && ['queue', 'verify', 'finish', 'notice', 'cards'].includes(tabParam)) {
    currentTab.value = tabParam
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
  
  // 根据权限选择默认Tab
  if (!tabParam) {
    if (showQueueTab.value) {
      currentTab.value = 'queue'
    } else if (showVerifyTab.value) {
      currentTab.value = 'verify'
    } else if (showFinishTab.value) {
      currentTab.value = 'finish'
    } else if (showNoticeTab.value) {
      currentTab.value = 'notice'
    } else {
      currentTab.value = showCardsTab.value ? 'cards' : 'queue'
    }
  } else {
    // 如果指定了tab但没有权限，则切换到默认tab
    if (currentTab.value === 'queue' && !showQueueTab.value) {
      if (showVerifyTab.value) {
        currentTab.value = 'verify'
      } else if (showFinishTab.value) {
        currentTab.value = 'finish'
      } else if (showNoticeTab.value) {
        currentTab.value = 'notice'
      } else {
        currentTab.value = showCardsTab.value ? 'cards' : 'queue'
      }
    } else if (currentTab.value === 'verify' && !showVerifyTab.value) {
      if (showQueueTab.value) {
        currentTab.value = 'queue'
      } else if (showFinishTab.value) {
        currentTab.value = 'finish'
      } else if (showNoticeTab.value) {
        currentTab.value = 'notice'
      } else {
        currentTab.value = showCardsTab.value ? 'cards' : 'queue'
      }
    } else if (currentTab.value === 'finish' && !showFinishTab.value) {
      if (showQueueTab.value) {
        currentTab.value = 'queue'
      } else if (showVerifyTab.value) {
        currentTab.value = 'verify'
      } else if (showNoticeTab.value) {
        currentTab.value = 'notice'
      } else {
        currentTab.value = showCardsTab.value ? 'cards' : 'queue'
      }
    } else if (currentTab.value === 'notice' && !showNoticeTab.value) {
      if (showQueueTab.value) {
        currentTab.value = 'queue'
      } else if (showVerifyTab.value) {
        currentTab.value = 'verify'
      } else if (showFinishTab.value) {
        currentTab.value = 'finish'
      } else {
        currentTab.value = showCardsTab.value ? 'cards' : 'queue'
      }
    } else if (currentTab.value === 'cards' && !showCardsTab.value) {
      if (showQueueTab.value) {
        currentTab.value = 'queue'
      } else if (showVerifyTab.value) {
        currentTab.value = 'verify'
      } else if (showFinishTab.value) {
        currentTab.value = 'finish'
      } else if (showNoticeTab.value) {
        currentTab.value = 'notice'
      } else {
        currentTab.value = 'queue'
      }
    }
  }

  fetchQueueStatus()
  fetchPendingDirectPurchases()
  fetchAppointments()
  loadCardTemplates() // 加载卡片模板
  if (merchant.value.support_appointment) {
    startCountdownTimer()
  }

  if (currentTab.value === 'cards' && scanUserCodeActive.value && routeUserCode.value) {
    await fetchIssuedCards()
    await scrollToUserCodeHint()
    await cleanupScanQuery()
  }
})

onBeforeRouteLeave(() => {
  scanUserCodeActive.value = false
  routeUserCode.value = ''
})

onUnmounted(() => {
  stopCountdownTimer()
  scanUserCodeActive.value = false
  routeUserCode.value = ''
})

onActivated(() => {
  if (!merchantId.value) return
  fetchMerchant()
})
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
