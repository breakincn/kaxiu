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

    <!-- 卡片详情 -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
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
      </div>
    </div>

    <!-- 营业时间 -->
    <div v-if="getMerchantBusinessHours()" class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <svg :class="isMerchantOpen() ? 'text-green-500' : 'text-red-500'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <span class="font-medium">营业时间</span>
          </div>
          <span v-if="!isMerchantOpen()" class="bg-red-500 text-white text-sm font-medium px-3 py-1 rounded">打烊</span>
        </div>
        <div class="text-sm leading-relaxed text-gray-500" v-html="getMerchantBusinessHours()"></div>
      </div>
    </div>

    <!-- 预约排队区域（如果支持且不在冷却中） -->
    <div v-if="card.merchant?.support_appointment && !isInCooldown" class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center gap-2 mb-3">
          <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <span class="font-medium text-gray-800">预约排队</span>
        </div>
        
        <div v-if="appointment" class="space-y-3">
          <div class="flex justify-between items-center">
            <span class="text-gray-500">我的预约</span>
            <span :class="getAppointmentStatusClass(appointment.status)">
              {{ getAppointmentStatusText(appointment.status) }}
            </span>
          </div>
          <div class="flex justify-between items-center">
            <div :class="getAppointmentTimeClass()" class="font-medium text-lg">
              {{ formatDateTime(appointment.appointment_time) }}
            </div>
            <div class="text-right">
              <div v-if="appointment.status === 'confirmed'" class="text-sm text-gray-600 mb-1">排队中</div>
              <div v-if="!isAppointmentPassed()" :class="getCountdownClass()" class="text-sm font-medium">
                {{ getCountdownText() }}
              </div>
              <div v-else class="text-sm text-gray-400">
                预约已过
              </div>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-4 pt-3 mt-3 border-t border-gray-200">
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
          
          <!-- 取消预约按钮 -->
          <button
            @click="cancelAppointment"
            :disabled="cancelButtonDisabled"
            class="w-full py-2.5 border-2 border-red-400 text-red-500 font-medium rounded-lg hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors mt-3"
          >
            {{ cancelButtonText }}
          </button>
        </div>
        
        <div v-else class="text-center">
          <button
            @click="showAppointmentModal"
            :disabled="appointing || isInCooldown"
            class="w-full py-3 border-2 border-primary text-primary font-medium rounded-lg hover:bg-primary-light disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {{ isInCooldown ? cooldownButtonText : '我要预约' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 核销码区域 -->
    <div v-if="shouldShowVerifyCode()" class="px-4 mt-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="text-center text-gray-600 mb-3">到店出示核销码</div>
        <button
          @click="generateCode"
          :disabled="generating || card.remain_times <= 0"
          class="w-full py-3 border-2 border-primary text-primary font-medium rounded-lg hover:bg-primary-light disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {{ generating ? '生成中...' : (verifyCode ? verifyCode : '生成核销码') }}
        </button>

			<div v-if="verifyQrDataUrl" class="mt-4 flex justify-center">
				<img :src="verifyQrDataUrl" alt="核销二维码" class="w-48 h-48" />
			</div>
			
			<!-- 当前核销次数 -->
			<div v-if="card && verifyQrDataUrl" class="mt-3 text-center text-sm text-gray-700 font-medium">
				第{{ card.total_times - card.remain_times + 1 }}次核销
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

    <!-- 选择项目弹窗（多项目时生成核销码前选择） -->
    <div v-if="showProjectModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeProjectModal">
      <div class="bg-white rounded-xl w-[90%] max-w-sm overflow-hidden">
        <div class="px-4 py-3 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">请选择项目</div>
          <button class="text-gray-400" @click="closeProjectModal">×</button>
        </div>
        <div class="p-4 max-h-[60vh] overflow-y-auto">
          <div v-if="!card.projects || card.projects.length === 0" class="text-center text-gray-400 py-6">暂无可选项目</div>
          <label v-for="p in card.projects" :key="p.id" class="flex items-center gap-3 py-2">
            <input type="radio" name="project" :value="p.id" v-model="selectedProjectId" />
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

    <!-- 使用记录 -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-5 shadow-sm border border-gray-200">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
            </svg>
            <span class="font-medium text-gray-800">使用记录</span>
          </div>
          <span class="text-gray-600 text-sm">
            总数{{ card.total_times }}次/剩余{{ card.remain_times }}次
          </span>
        </div>
        <div v-if="usages.length > 0" class="space-y-2">
          <div
            v-for="usage in usages"
            :key="usage.id"
            class="grid grid-cols-[1fr_auto] items-start p-3 bg-white rounded-lg shadow-sm"
            @touchstart="(e) => onUsageTouchStart(e, usage)"
            @touchmove="onUsageTouchMove"
            @touchend="onUsageTouchEnd"
            @touchcancel="onUsageTouchEnd"
            @contextmenu.prevent
            style="-webkit-touch-callout: none;"
          >
            <div class="min-w-0">
              <div class="text-gray-800">核销次数: {{ usage.used_times }}</div>
              <div class="text-gray-400 text-sm mt-0.5">
                单号：{{ getUsageTrackingNumber(usage) }}
              </div>
              <div class="flex items-center gap-2">
                <span class="text-gray-500 text-sm">{{ getWeekDay(usage.used_at) }}</span>
                <span class="text-gray-400 text-sm">{{ formatDateTime(usage.used_at) }}</span>
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
              <div v-if="getUsageProjectText(usage)" class="text-gray-400 text-sm mt-0.5">
                {{ getUsageProjectText(usage) }}
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
                v-if="isUsageStartTimeout(usage) && usage?.can_revoke"
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

            <div v-if="getUsageOperatorInfo(usage) || getUsageRoomInfo(usage)" class="col-span-2 flex items-center justify-between text-gray-400 text-sm mt-0.5">
              <span v-if="getUsageOperatorInfo(usage)">{{ getUsageOperatorInfo(usage) }}</span>
              <span v-if="getUsageRoomInfo(usage)" class="text-gray-600 text-xs whitespace-nowrap">{{ getUsageRoomInfo(usage) }}</span>
            </div>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-4">
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
    <div v-if="shouldShowBottomSpacer" :style="{ height: getBottomSpacerHeight() }"></div>

    <!-- 预约时间选择弹窗 -->
    <div v-if="showModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="closeModal">
      <div class="bg-white rounded-2xl w-11/12 max-w-lg max-h-[80vh] overflow-hidden">
        <!-- 弹窗头部 -->
        <div class="bg-primary text-white px-5 py-4 flex items-center justify-between">
          <h3 class="font-medium text-lg">选择预约时间</h3>
          <button @click="closeModal" class="text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-3 border-b">
          <div class="flex gap-2">
            <button
              type="button"
              @click="appointmentMode = 'time'"
              :class="appointmentMode === 'time' ? 'bg-primary text-white' : 'bg-gray-100 text-gray-700'"
              class="flex-1 py-2 px-4 rounded-lg font-medium transition-colors"
            >
              按时间段
            </button>
            <button
              type="button"
              @click="appointmentMode = 'technician'"
              :disabled="technicians.length === 0"
              :class="appointmentMode === 'technician' ? 'bg-primary text-white' : 'bg-gray-100 text-gray-700'"
              class="flex-1 py-2 px-4 rounded-lg font-medium transition-colors disabled:opacity-50"
            >
              选技师
            </button>
          </div>

          <div v-if="appointmentMode === 'technician'" class="mt-3">
            <div class="text-sm font-medium text-gray-700 mb-2">选择技师</div>
            <div v-if="loadingTechnicians" class="text-gray-400 text-sm">加载中...</div>
            <div v-else-if="technicians.length === 0" class="text-gray-400 text-sm">暂无技师</div>
            <div v-else class="grid grid-cols-2 gap-2">
              <button
                v-for="t in technicians"
                :key="t.id"
                type="button"
                @click="selectedTechnicianId = t.id"
                :class="selectedTechnicianId === t.id ? 'bg-primary text-white' : 'bg-white border-2 border-gray-200 text-gray-700 hover:border-primary'"
                class="py-2 px-3 rounded-lg font-medium transition-all text-sm"
              >
                {{ t.name }}
              </button>
            </div>
          </div>
        </div>

        <!-- 日期选择 -->
        <div class="px-5 py-3 border-b">
          <div class="flex gap-2">
            <button
              type="button"
              class="flex-1 py-2 px-4 rounded-lg font-medium transition-colors bg-primary text-white"
            >
              明天
            </button>
          </div>
        </div>

        <!-- 时间段列表 -->
        <div class="px-5 py-4 overflow-y-auto" style="max-height: 400px;">
          <div v-if="loadingSlots" class="text-center py-8 text-gray-400">
            加载中...
          </div>
          <div v-else-if="timeSlots.length === 0" class="text-center py-8 text-gray-400">
            明日无可用时间段
          </div>
          <div v-else class="grid grid-cols-2 gap-3">
            <button
              v-for="slot in timeSlots"
              :key="slot.time"
              @click="selectTimeSlot(slot)"
              :disabled="!slot.available"
              :class="{
                'bg-primary text-white': selectedTimeSlot === slot.time && slot.available,
                'bg-gray-100 text-gray-400 cursor-not-allowed': !slot.available,
                'bg-white border-2 border-gray-200 text-gray-700 hover:border-primary': slot.available && selectedTimeSlot !== slot.time
              }"
              class="py-3 px-4 rounded-lg font-medium transition-all"
            >
              <div>{{ formatTime(slot.time) }}</div>
              <div v-if="!slot.available" class="text-xs mt-1">已被预约</div>
            </button>
          </div>
        </div>

        <!-- 弹窗底部 -->
        <div class="px-5 py-4 border-t">
          <button
            @click="confirmAppointment"
            :disabled="!selectedTimeSlot || appointing || (appointmentMode === 'technician' && !selectedTechnicianId)"
            class="w-full py-3 bg-primary text-white font-medium rounded-lg hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {{ appointing ? '预纤中...' : '确认预约' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { cardApi, usageApi, noticeApi, appointmentApi } from '../../api'
import { formatDateTime, formatDate } from '../../utils/dateFormat'
import QRCode from 'qrcode'

import { replaceTerms } from '../../utils/terms'

const router = useRouter()
const route = useRoute()

const card = ref({})
const usages = ref([])
const notices = ref([])
const appointment = ref(null)
const queueBefore = ref(0)
const estimatedMinutes = ref(0)
const countdown = ref(0)
let countdownTimer = null

const cooldownUntil = ref(null)

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
  return replaceTerms('起单二维码', card.value?.merchant)
})

const usageQrAlt = computed(() => {
  return replaceTerms('起单二维码', card.value?.merchant)
})

let usageLongPressTimer = null
let usageTouchStartX = 0
let usageTouchStartY = 0
let usageTouchMoved = false

const revokeLoading = ref(false)

const isUsageStartTimeout = (usage) => {
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  if (!supportCS) return false
  const supportRoom = Boolean(card.value?.merchant?.support_room)
  const sessStatus = String(usage?.service_session_status || '').trim()
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

const getUsageStatusText = (usage) => {
  const s = String(usage?.status || '').trim()
  if (s === 'in_progress') {
    const supportCS = Boolean(card.value?.merchant?.support_customer_service)
    const supportRoom = Boolean(card.value?.merchant?.support_room)
    const sessStatus = String(usage?.service_session_status || '').trim()
    const precheckedAt = usage?.service_session_start_confirmed_at
    const now = nowTick.value
    if (sessStatus === 'finished') return '完成'
    if (supportRoom && sessStatus === 'room_selecting') return '待选房间'
    if (supportCS && sessStatus === 'room_locked') return '待选客服'
    // 会话已取消但 usage 仍在进行中：
    // - 若房间/客服均已释放：视为“超时未选择客服”，需重新选择房间
    // - 否则：视为上钟超时，需重新选择客服
    if (supportCS && sessStatus === 'canceled' && !precheckedAt) {
      if (supportRoom && !usage?.service_room && !usage?.service_technician) {
        return '服务超时重新选择房间'
      }
      // 已经选定/自动分配了客服：应回到待起单
      if (usage?.service_technician) {
        return replaceTerms('待起单', card.value?.merchant)
      }
      return '上钟超时 重新选择客服'
    }
    // 上钟超时统一优先判断（避免兜底到待结单）
    if (supportCS) {
      const cnt = Number(usage?.start_timeout_count || 0)
      if (cnt > 0 && sessStatus === 'staff_selecting' && !usage?.service_technician) {
        return '上钟超时 重新选择客服'
      }
    }
    if (supportCS && sessStatus === 'staff_selecting') {
      // 已经选定/自动分配了客服：应回到待起单
      if (usage?.service_technician) return replaceTerms('待起单', card.value?.merchant)
      return '待选客服'
    }
    if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
      const dl = getPrecheckDeadlineAtMs(usage)
      if (dl && now < dl) return replaceTerms('待起单', card.value?.merchant)
      if (dl && now >= dl) return '上钟超时 重新选择客服'
    }
    // 未上钟成功（未确认起单）时，永远不要进入“待下钟/待结单”兜底
    if (supportCS && !precheckedAt) {
      // start_pending 且已超时：应立刻显示“上钟超时 重新选择客服”（无需刷新页面）
      if (sessStatus === 'start_pending') {
        const dl = getPrecheckDeadlineAtMs(usage)
        if (dl && now >= dl) return '上钟超时 重新选择客服'
      }
      return replaceTerms('待起单', card.value?.merchant)
    }
    return replaceTerms('待结单', card.value?.merchant)
  }
  if (s === 'success') return '完成'
  if (s === 'failed') return '失败'
  return s || ''
}

const getUsageStatusClass = (usage) => {
  const s = String(usage?.status || '').trim()
  if (s === 'in_progress') {
    const supportCS = Boolean(card.value?.merchant?.support_customer_service)
    const supportRoom = Boolean(card.value?.merchant?.support_room)
    const sessStatus = String(usage?.service_session_status || '').trim()
    const precheckedAt = usage?.service_session_start_confirmed_at
    const now = nowTick.value
    if (sessStatus === 'finished') return 'text-gray-600'
    if (supportRoom && sessStatus === 'room_selecting') return 'text-orange-500'
    if (supportCS && sessStatus === 'room_locked') return 'text-orange-500'
    // 会话已取消但 usage 仍在进行中：视为上钟超时
    if (supportCS && sessStatus === 'canceled' && !precheckedAt) {
      // 已经选定/自动分配了客服：不再视为上钟超时
      if (usage?.service_technician) {
        return 'text-red-500'
      }
      return 'text-red-500'
    }
    // 上钟超时统一优先判断（避免兜底到待结单样式）
    if (supportCS) {
      const cnt = Number(usage?.start_timeout_count || 0)
      if (cnt > 0 && sessStatus === 'staff_selecting') {
        return 'text-red-500'
      }
    }
    if (supportCS && sessStatus === 'staff_selecting') {
      return 'text-orange-500'
    }
    if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
      const dl = getPrecheckDeadlineAtMs(usage)
      if (dl && now < dl) return 'text-red-500'
    }
    if (supportCS && sessStatus === 'start_pending' && !precheckedAt && getPrecheckDeadlineAtMs(usage) && now >= getPrecheckDeadlineAtMs(usage)) {
      return 'text-red-500'
    }
    // 未上钟成功（未确认起单）时，永远不要进入“待下钟/待结单”蓝色兜底
    if (supportCS && !precheckedAt) {
      return 'text-red-500'
    }
    return 'text-blue-500'
  }
  if (s === 'success') return ''
  if (s === 'failed') return 'text-red-500'
  return ''
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
  if (!card.value?.merchant?.support_room) return

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
  if (!card.value?.merchant?.support_room) return

  await checkVerifyStatusAndMaybeJump()
  if (hasJumpedToRoomSelect.value) return

  verifyStatusPollTimer = setInterval(() => {
    // 核销码被清空/过期后停止轮询
    if (!verifyCode.value) {
      stopVerifyStatusPoll()
      return
    }
    checkVerifyStatusAndMaybeJump()
  }, 1200)
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

const getUsageServiceStartAtMs = (usage) => {
  const confirmedAtMs = getUsageSessionStartConfirmedAtMs(usage)
  if (confirmedAtMs) return confirmedAtMs

  const startedAtMs = getUsageSessionStartedAtMs(usage)
  if (startedAtMs) return startedAtMs

  // 若未扫码起单，则按后端调度逻辑推算：updated_at + start_pending_timeout_seconds + 60s
  const sessUpdatedAtMs = getUsageSessionUpdatedAtMs(usage)
  const startPendingTimeoutMs = getStartPendingTimeoutMs(usage)
  if (sessUpdatedAtMs && startPendingTimeoutMs) return sessUpdatedAtMs + startPendingTimeoutMs + 60 * 1000

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

const getUsageServiceStartAtText = (usage) => {
  if (String(usage?.status || '').trim() !== 'in_progress') return ''
  const ms = getUsageServiceStartAtMs(usage)
  if (!ms) return ''
  return formatDateTime(new Date(ms))
}

const shouldShowServiceStartTime = (usage) => {
  if (String(usage?.status || '').trim() !== 'in_progress') return false
  
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  const sessStatus = String(usage?.service_session_status || '').trim()
  const precheckedAt = usage?.service_session_start_confirmed_at
  
  // 如果支持客服且处于待起单状态且未确认起单，则不显示服务开始时间
  // 因为此时显示的是预估时间，不是实际开始时间
  if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    return false
  }

  // 会话已取消但 usage 仍在进行中：不显示预估开始时间
  if (supportCS && sessStatus === 'canceled' && !precheckedAt) {
    return false
  }
  
  return true
}

const shouldShowServiceRemainTime = (usage) => {
  if (String(usage?.status || '').trim() !== 'in_progress') return false
  
  const precheckedAt = usage?.service_session_start_confirmed_at
  
  // 仅在已扫码起单确认后才显示剩余时间（避免未上钟前显示预估时间）
  if (!precheckedAt) return false
  return true
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
  }, 2000)
}

const getUsageStatusCountdownText = (usage) => {
  const s = String(usage?.status || '').trim()
  if (s !== 'in_progress') return ''

  const supportRoom = Boolean(card.value?.merchant?.support_room)
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  const sessStatus = String(usage?.service_session_status || '').trim()
  const precheckedAt = usage?.service_session_start_confirmed_at
  const now = nowTick.value

  // 待选房间倒计时（90秒）
  if (supportRoom && sessStatus === 'room_selecting' && usage?.room_select_deadline_at) {
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

  // 待起单倒计时
  if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    const dl = getPrecheckDeadlineAtMs(usage)
    if (dl) {
      const diff = dl - now
      if (diff > 0) {
        const totalSeconds = Math.floor(diff / 1000)
        const minutes = Math.floor(totalSeconds / 60)
        const seconds = totalSeconds % 60
        return `${minutes}分${seconds}秒后重新选择客服`
      }
      // 超时后不显示倒计时
      return ''
    }
  }

  return ''
}

const getUsageStatusCountdownClass = (usage) => {
  const s = String(usage?.status || '').trim()
  if (s !== 'in_progress') return 'text-gray-400'

  const supportRoom = Boolean(card.value?.merchant?.support_room)
  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  const sessStatus = String(usage?.service_session_status || '').trim()
  const precheckedAt = usage?.service_session_start_confirmed_at

  // 待选房间倒计时（橙色）
  if (supportRoom && sessStatus === 'room_selecting' && usage?.room_select_deadline_at) {
    return 'text-orange-500'
  }

  // 待选客服倒计时（橙色）
  if (supportCS && (sessStatus === 'room_locked' || sessStatus === 'staff_selecting') && usage?.room_locked_at) {
    return 'text-orange-500'
  }

  // 待起单倒计时（红色）
  if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    return 'text-red-500'
  }

  return 'text-blue-500'
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
  const sessStatus = String(latest?.service_session_status || '').trim()
  const precheckedAt = latest?.service_session_start_confirmed_at
  if (supportCS && sessStatus === 'canceled' && !precheckedAt && !latest?.service_technician) {
    stopUsageQrPoll()
    closeUsageQrModal()
    alert('上钟超时，请重新选择客服')
    return
  }
  if (supportCS && sessStatus === 'start_pending' && !precheckedAt) {
    const dl = getPrecheckDeadlineAtMs(latest)
    if (dl && Date.now() >= dl) {
      stopUsageQrPoll()
      closeUsageQrModal()
      alert('上钟超时，请重新选择客服')
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

  const supportCS = Boolean(card.value?.merchant?.support_customer_service)
  const sessID = usage?.service_session_id
  const sessStatus = String(usage?.service_session_status || '').trim()
  const precheckedAt = usage?.service_session_start_confirmed_at

    // 会话已取消但 usage 仍在进行中：起单二维码失效（不自动跳转，改为长按记录进入重新选客服）
  if (supportCS && sessID && sessStatus === 'canceled' && !precheckedAt && !usage?.service_technician) {
    alert('上钟超时，请长按该记录重新选择客服')
    return
  }

  // 上钟超时后起单二维码失效（不自动跳转，改为长按记录进入重新选客服）
  if (supportCS && sessID && sessStatus === 'start_pending' && !precheckedAt) {
    const dl = getPrecheckDeadlineAtMs(usage)
    if (dl && Date.now() >= dl) {
      alert('上钟超时，请长按该记录重新选择客服')
      return
    }
  }

  // 仅当当前 usage 匹配 query.session_id 且确实处于待起单时，才显示起单二维码
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
    }, 2000)
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
    }, 2000)
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

    const sessStatus = String(u?.service_session_status || '').trim()
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

      const supportRoom = Boolean(card.value?.merchant?.support_room)
      const supportCS = Boolean(card.value?.merchant?.support_customer_service)
      const sessStatus = String(latest?.service_session_status || '').trim()
      const sessID = latest?.service_session_id
      const precheckedAt = latest?.service_session_start_confirmed_at

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
        if (supportRoom && sessStatus === 'room_selecting') {
          router.push({ path: `/user/service-sessions/${sessID}`, query: { next_step: 'room_select' } })
          return
        }
			if (supportCS && (sessStatus === 'room_locked' || sessStatus === 'staff_selecting')) {
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
const shouldShowBottomSpacer = ref(false)

// 预约弹窗相关
const showModal = ref(false)
const appointmentMode = ref('time')
const selectedDate = ref('')
const selectedTimeSlot = ref('')
const timeSlots = ref([])
const loadingSlots = ref(false)

const technicians = ref([])
const loadingTechnicians = ref(false)
const selectedTechnicianId = ref(null)

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
  // 只在已完成状态显示房间号
  if (usage.status === 'success' && usage.finished_at && usage.service_room) {
    return `房间: ${usage.service_room.name || usage.service_room.code}`
  }
  return ''
}

const goBack = () => {
  router.push('/user/cards')
}

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
  try {
    const res = await usageApi.getCardUsages(route.params.id)
    usages.value = res.data.data || []

		// 记录“已开始计时等待用户选技师”的 usage
		try {
			for (const u of (usages.value || [])) {
				recordAutoAssignCountdownIfNeeded(u)
			}
		} catch (_) {
			// ignore
		}

		// 每次刷新使用记录后，尝试检测“自动分配成功”并自动弹出待上钟二维码（5秒自动关闭）
		try {
			await maybeAutoOpenPrecheckQrAfterAssigned()
		} catch (_) {
			// ignore
		}

		// 可能存在计时任务，确保轮询启动；若已无计时任务则停止
		startAutoAssignPollIfNeeded()
  } catch (err) {
    console.error('获取使用记录失败:', err)
  }
}

const fetchNotices = async (merchantId) => {
  try {
    const res = await noticeApi.getMerchantNotices(merchantId, 5)
    notices.value = res.data.data || []
    
    // 如果需要滚动到通知区域
    if (route.query.scrollToNotice === '1' && notices.value.length > 0) {
      shouldShowBottomSpacer.value = true
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
    cooldownUntil.value = data?.cooldown_until || null

    if (data?.appointment) {
      appointment.value = data.appointment
      queueBefore.value = data.queue_before || 0
      estimatedMinutes.value = data.estimated_minutes || 0
      console.log('预约信息已设置:', appointment.value)
      // 启动倒计时
      startCountdownTimer()
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

const getCooldownRemainingSeconds = () => {
  if (!cooldownUntil.value) return 0
  const until = new Date(cooldownUntil.value).getTime()
  const now = Date.now()
  return Math.max(0, Math.floor((until - now) / 1000))
}

const isInCooldown = computed(() => {
  return getCooldownRemainingSeconds() > 0
})

const cooldownButtonText = computed(() => {
  const seconds = getCooldownRemainingSeconds()
  if (seconds <= 0) return '我要预约'
  const minutes = Math.ceil(seconds / 60)
  return `冷却中（约${minutes}分钟后可预约）`
})

const isAppointmentFailed = computed(() => {
  if (!appointment.value || !appointment.value.appointment_time) return false
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
  if (isAppointmentFailed.value) return '预约失败'
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
  appointmentMode.value = 'time'
  selectedTechnicianId.value = null
  selectedDate.value = getTomorrowDate()
  selectedTimeSlot.value = ''
  await loadTechnicians(card.value.merchant_id)
  await loadTimeSlots(selectedDate.value)
}

// 关闭弹窗
const closeModal = () => {
  showModal.value = false
  appointmentMode.value = 'time'
  selectedTechnicianId.value = null
  selectedDate.value = ''
  selectedTimeSlot.value = ''
  timeSlots.value = []
}

const loadTechnicians = async (merchantId) => {
  loadingTechnicians.value = true
  try {
    const res = await appointmentApi.getMerchantTechnicians(merchantId)
    technicians.value = res.data.data || []
  } catch (_) {
    technicians.value = []
  } finally {
    loadingTechnicians.value = false
  }
}

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
  
  loadingSlots.value = true
  try {
    console.log('正在获取时间段，商户ID:', card.value.merchant_id, '日期:', date)
    const res = await appointmentApi.getAvailableTimeSlots(card.value.merchant_id, date)
    console.log('获取时间段响应:', res.data)
    timeSlots.value = res.data.data.time_slots || []

    if (appointmentMode.value === 'technician' && !selectedTimeSlot.value) {
      const first = (timeSlots.value || []).find(s => s && s.available)
      if (first) selectedTimeSlot.value = first.time
    }
  } catch (err) {
    console.error('获取可用时间段失败:', err)
    console.error('错误详情:', err.response?.data)
    alert(`获取可用时间段失败: ${err.response?.data?.error || err.message}`)
  } finally {
    loadingSlots.value = false
  }
}

// 选择时间段
const selectTimeSlot = (slot) => {
  if (!slot.available) return
  selectedTimeSlot.value = slot.time
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

  if (appointmentMode.value === 'technician' && !selectedTechnicianId.value) {
    alert('请选择技师')
    return
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
      merchant_id: card.value.merchant_id,
      user_id: parseInt(userId),
      technician_id: appointmentMode.value === 'technician' ? selectedTechnicianId.value : null,
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

// 判断是否应该显示核销码区域
const shouldShowVerifyCode = () => {
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

const getBottomSpacerHeight = () => {
  // 当有通知时，添加底部占位高度，确保可以滚动到通知区域
  const windowHeight = window.innerHeight || 800
  const estimatedContentHeight = 700 // 估算页面内容高度
  const minSpacerHeight = Math.max(windowHeight - estimatedContentHeight, 300)
  return `${minSpacerHeight}px`
}

const scrollToNotice = async () => {
  // 等待DOM更新，包括动态占位元素的渲染
  await nextTick()
  // 再次等待，确保占位元素高度计算完成
  await new Promise(resolve => setTimeout(resolve, 100))
  
  const el = noticeAnchor.value
  if (!el) return
  try {
    // 获取元素的位置信息
    const rect = el.getBoundingClientRect()
    // 计算目标滚动位置：元素顶部 + 当前滚动位置 - 4px偏移
    const targetScrollTop = rect.top + window.scrollY - 4

    // 平滑滚动到目标位置
    window.scrollTo({
      top: targetScrollTop,
      behavior: 'smooth'
    })
  } catch (_) {
    // 降级方案：使用 scrollIntoView
    try {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    } catch (_) {
      // ignore
    }
  }
}

onMounted(async () => {
  await fetchCard()
  startNowTickTimer()
  // 如果有预约，启动倒计时
  if (appointment.value) {
    startCountdownTimer()
  }
})

onUnmounted(() => {
  stopAutoAssignPoll()
  stopNowTickTimer()
  stopCountdownTimer()
  stopVerifyStatusPoll()
  if (verifyExpireTimer) {
    clearTimeout(verifyExpireTimer)
    verifyExpireTimer = null
  }
  // 重置底部占位状态
  shouldShowBottomSpacer.value = false
})
</script>
