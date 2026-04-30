<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <span class="font-medium text-gray-800">发卡 / 开卡</span>
    </header>

    <div class="px-4 py-4 space-y-4">
      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="font-medium text-gray-800 mb-3">通过手机号搜索用户</div>
        <div class="flex gap-2">
          <input
            v-model="phoneQuery"
            type="tel"
            placeholder="输入手机号（支持模糊搜索）"
            class="flex-1 px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
          />
          <button
            @click="searchUsers"
            :disabled="!phoneQuery || searching"
            class="px-4 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ searching ? '搜索中...' : '搜索' }}
          </button>
        </div>

        <div v-if="searchError" class="mt-3 text-sm text-gray-700">{{ searchError }}</div>

        <div v-if="users.length > 0" class="mt-4 space-y-2">
          <button
            v-for="u in users"
            :key="u.id"
            @click="selectUser(u)"
            class="w-full text-left px-4 py-3 border border-gray-200 rounded-lg hover:border-primary"
          >
            <div class="flex items-center justify-between">
              <div>
                <div class="text-gray-800 font-medium">{{ u.nickname || '未命名用户' }}</div>
                <div class="text-gray-500 text-sm">手机号：{{ u.phone }}</div>
              </div>
              <div class="text-primary text-sm font-medium">选择</div>
            </div>
          </button>
        </div>

        <div v-else-if="searched" class="mt-4 text-center text-gray-400">未找到用户</div>
      </div>

      <div v-if="selectedUser" class="bg-white rounded-xl p-4 shadow-sm">
        <div class="flex items-start justify-between">
          <div>
            <div class="font-medium text-gray-800">已选择用户</div>
            <div class="text-gray-700 mt-2">{{ selectedUser.nickname || '未命名用户' }}</div>
            <div class="text-gray-500 text-sm">手机号：{{ selectedUser.phone }}</div>
            <div class="text-gray-500 text-sm">用户ID：{{ selectedUser.id }}</div>
          </div>
          <button @click="clearSelectedUser" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <div class="bg-white rounded-xl p-4 shadow-sm">
        <div class="font-medium text-gray-800 mb-4">卡信息</div>

        <div class="space-y-3">
          <div>
            <label class="block text-gray-700 text-sm font-medium mb-2">选择卡片</label>
            <select
              v-model.number="cardForm.template_id"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            >
              <option :value="0">请选择售卡模板</option>
              <option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">
                {{ tpl.name }}（¥{{ (tpl.price / 100).toFixed(2) }}）
              </option>
            </select>
          </div>

          <div v-if="selectedTemplate" class="px-4 py-3 border border-gray-100 rounded-lg bg-gray-50">
            <div class="text-sm text-gray-700">类型：{{ getCardTypeLabel(selectedTemplate.card_type) }}</div>
            <div v-if="selectedTemplate.card_type !== 'balance'" class="text-sm text-gray-700">
              次数：{{ promotionEnabled ? (promotionForm.reward_value || 0) : selectedTemplate.total_times }}
            </div>
            <div v-else class="text-sm text-gray-700">
              额度：¥{{ ((promotionEnabled ? Number(promotionForm.reward_value || 0) : selectedTemplate.recharge_amount / 100) || 0).toFixed(2) }}
            </div>
            <div v-if="!promotionEnabled" class="text-sm text-gray-700">售价：¥{{ (selectedTemplate.price / 100).toFixed(2) }}</div>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-gray-700 text-sm font-medium mb-2">开始日期（可选）</label>
              <input
                v-model="cardForm.start_date"
                type="date"
                class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              />
            </div>
            <div>
              <label class="block text-gray-700 text-sm font-medium mb-2">结束日期</label>
              <input
                v-model="cardForm.end_date"
                type="date"
                disabled
                class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary bg-gray-50 text-gray-500"
              />
            </div>
          </div>

          <div v-if="canShowPromotionToggle" class="rounded-2xl border border-orange-200 bg-orange-50 p-4 space-y-3">
            <div class="rounded-2xl bg-white px-4 py-3 shadow-sm">
              <div class="flex items-center justify-between gap-3">
                <div class="text-[17px] font-medium leading-6 text-gray-900">推广卡</div>
                <button
                  type="button"
                  role="switch"
                  :aria-checked="promotionEnabled"
                  @click="togglePromotionEnabled"
                  class="relative inline-flex h-[30px] w-[56px] shrink-0 items-center rounded-full transition-colors duration-200 focus:outline-none"
                  :class="promotionEnabled ? 'bg-[#34c759]' : 'bg-[#d1d1d6]'"
                >
                  <span
                    class="inline-block h-[26px] w-[26px] rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,0.22)] transition-transform duration-200"
                    :class="promotionEnabled ? 'translate-x-[28px]' : 'translate-x-[2px]'"
                  />
                </button>
              </div>
              <div class="mt-1 text-xs leading-4 text-gray-500">未指定用户手机号时，可打开生成推广卡</div>
            </div>

            <div v-if="promotionEnabled" class="space-y-3">
              <div
                v-if="currentCampaign && promotionLink"
                class="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-gray-700"
              >
                <div class="font-medium text-green-700 truncate">
                  {{ currentCampaign.title || '推广卡活动' }}
                </div>
                <div class="mt-1 truncate text-gray-700">{{ promotionLink }}</div>
                <div class="mt-3 flex flex-wrap gap-3">
                  <button @click="copyPromotionLink" class="text-primary text-sm">复制链接</button>
                  <button
                    @click="handlePosterAction"
                    :disabled="generatingPoster"
                    class="text-primary text-sm disabled:opacity-50"
                  >
                    {{ generatingPoster ? '生成中...' : posterActionLabel }}
                  </button>
                </div>
              </div>

              <div>
                <label class="block text-gray-700 text-sm font-medium mb-2">推广（促销）标题</label>
                <input
                  v-model="promotionForm.title"
                  type="text"
                  maxlength="120"
                  placeholder="可选，最多120个中文字"
                  class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-gray-700 text-sm font-medium mb-2">{{ rewardValueLabel }}</label>
                  <input
                    v-model.number="promotionForm.reward_value"
                    type="number"
                    min="1"
                    :placeholder="rewardValuePlaceholder"
                    class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                  />
                </div>
                <div>
                  <label class="block text-gray-700 text-sm font-medium mb-2">推广数量</label>
                  <input
                    v-model.number="promotionForm.reward_threshold"
                    type="number"
                    min="1"
                    placeholder="达标门槛"
                    class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                  />
                </div>
              </div>

              <div>
                <label class="block text-gray-700 text-sm font-medium mb-2">推广发卡数量</label>
                <input
                  v-model.number="promotionForm.reward_card_quantity"
                  type="number"
                  min="1"
                  placeholder="奖励卡总库存"
                  class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-gray-700 text-sm font-medium mb-2">促销发卡数量</label>
                  <input
                    v-model.number="promotionForm.promo_quantity"
                    type="number"
                    min="0"
                    placeholder="促销资格库存"
                    class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                  />
                </div>
                <div>
                  <label class="block text-gray-700 text-sm font-medium mb-2">项目促销价格（元）</label>
                  <input
                    v-model.number="promotionForm.promo_price_yuan"
                    type="number"
                    min="0"
                    step="0.01"
                    placeholder="留空则不做促销价"
                    class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                  />
                </div>
              </div>

              <div>
                <label class="block text-gray-700 text-sm font-medium mb-2">促销截止日期</label>
                <input
                  v-model="promotionForm.promo_ends_at"
                  type="datetime-local"
                  class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
              </div>

              <div v-if="promotionError" class="text-sm text-red-500">{{ promotionError }}</div>
              <div v-else-if="promotionSaving" class="text-sm text-gray-500">正在更新推广链接...</div>
            </div>
          </div>

          <button
            v-if="!promotionEnabled"
            @click="submit"
            :disabled="submitting || !canSubmit"
            class="w-full mt-2 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ submitting ? '提交中...' : '确认发卡' }}
          </button>

          <div v-if="submitError" class="text-sm text-gray-700">{{ submitError }}</div>
          <div v-if="submitSuccess" class="text-sm text-primary">{{ submitSuccess }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import QRCode from 'qrcode'
import { cardApi, merchantApi, shopApi } from '../../api'
import { getMerchantId, getMerchantToken, hasMerchantPermission } from '../../utils/auth'

const router = useRouter()

const phoneQuery = ref('')
const searching = ref(false)
const searched = ref(false)
const searchError = ref('')
const users = ref([])
const selectedUser = ref(null)

const submitting = ref(false)
const submitError = ref('')
const submitSuccess = ref('')

const promotionEnabled = ref(false)
const promotionSaving = ref(false)
const promotionError = ref('')
const promotionLink = ref('')
const currentCampaign = ref(null)
const generatingPoster = ref(false)
const editingCampaignId = ref(0)
const savedPosterDataUrl = ref('')
let promotionSaveTimer = null
const lastSavedPromotionPayload = ref('')

const cardForm = ref({
  template_id: 0,
  start_date: '',
  end_date: ''
})

const promotionForm = ref({
  title: '',
  reward_value: '',
  reward_card_quantity: '',
  reward_threshold: '',
  promo_price_yuan: '',
  promo_quantity: '',
  promo_ends_at: ''
})

const templates = ref([])
const selectedTemplate = computed(() => {
  const id = Number(cardForm.value.template_id || 0)
  return (templates.value || []).find(t => Number(t.id) === id) || null
})

const canShowPromotionToggle = computed(() => Boolean(selectedTemplate.value && !phoneQuery.value.trim()))
const rewardValueLabel = computed(() => {
  return selectedTemplate.value?.card_type === 'balance' ? '推广卡额度（元）' : '推广卡次数'
})
const rewardValuePlaceholder = computed(() => {
  return selectedTemplate.value?.card_type === 'balance' ? '填写奖励额度' : '填写奖励次数'
})
const posterActionLabel = computed(() => savedPosterDataUrl.value ? '打开图片' : '生成图片')

const loadTemplates = async () => {
  try {
    const res = await shopApi.getCardTemplates()
    const list = res.data.data || []
    templates.value = list.filter(t => t && (t.card_type === 'times' || t.card_type === 'lesson' || t.card_type === 'balance'))
  } catch (_) {
    templates.value = []
  }
}

const calcEndDate = (startDateStr, validDays) => {
  const base = startDateStr ? new Date(startDateStr) : new Date()
  const end = new Date(base)
  const days = Number(validDays || 0)
  if (days > 0) {
    end.setDate(end.getDate() + days)
  } else {
    end.setFullYear(end.getFullYear() + 20)
  }
  return end.toISOString().split('T')[0]
}

const getCardTypeLabel = (type) => {
  const labels = { times: '次数卡', lesson: '课时卡', balance: '充值卡' }
  return labels[type] || type
}

watch(
  [() => cardForm.value.start_date, () => selectedTemplate.value],
  ([newStartDate, tpl]) => {
    if (!tpl) {
      cardForm.value.end_date = ''
      return
    }
    cardForm.value.end_date = calcEndDate(newStartDate, tpl.valid_days)
  }
)

watch(
  () => phoneQuery.value,
  (value) => {
    if (value.trim()) {
      promotionEnabled.value = false
    }
  }
)

watch(
  () => cardForm.value.template_id,
  async () => {
    promotionEnabled.value = false
    promotionLink.value = ''
    currentCampaign.value = null
    editingCampaignId.value = 0
    savedPosterDataUrl.value = ''
    lastSavedPromotionPayload.value = ''
    resetPromotionForm()
    if (selectedTemplate.value) {
      await loadCurrentCampaign()
    }
  }
)

const canSubmit = computed(() => {
  return Boolean(selectedUser.value && selectedUser.value.id && selectedTemplate.value && cardForm.value.end_date)
})

const goBack = () => router.back()

const ensureMerchantLogin = () => {
  const storedMerchantId = getMerchantId()
  const storedToken = getMerchantToken()
  if (!storedMerchantId || !storedToken) {
    router.replace('/merchant/login')
    return false
  }
  return true
}

const searchUsers = async () => {
  if (!ensureMerchantLogin()) return
  if (!phoneQuery.value || searching.value) return
  searching.value = true
  searched.value = false
  searchError.value = ''
  users.value = []
  try {
    const res = await merchantApi.searchUsersByPhone(phoneQuery.value)
    users.value = res.data.data || []
    searched.value = true
  } catch (err) {
    searchError.value = err.response?.data?.error || '搜索失败'
    searched.value = true
  } finally {
    searching.value = false
  }
}

const selectUser = (u) => {
  selectedUser.value = u
  submitSuccess.value = ''
  submitError.value = ''
}

const clearSelectedUser = () => {
  selectedUser.value = null
  submitSuccess.value = ''
  submitError.value = ''
}

const submit = async () => {
  if (!ensureMerchantLogin()) return
  if (!canSubmit.value || submitting.value) return
  submitting.value = true
  submitError.value = ''
  submitSuccess.value = ''
  try {
    const tpl = selectedTemplate.value
    const startDate = cardForm.value.start_date || new Date().toISOString().split('T')[0]
    const endDate = calcEndDate(startDate, tpl.valid_days)
    const payload = {
      user_id: selectedUser.value.id,
      card_type: tpl.name,
      total_times: tpl.card_type === 'balance' ? 0 : (Number(tpl.total_times) || 0),
      recharge_amount: tpl.card_type === 'balance' ? Math.round((Number(tpl.recharge_amount) || 0) / 100) : 0,
      start_date: startDate,
      end_date: endDate,
      project_ids: Array.isArray(tpl.project_ids) ? tpl.project_ids : []
    }
    await cardApi.createCard(payload)
    router.push('/merchant?tab=cards')
  } catch (err) {
    submitError.value = err.response?.data?.error || '发卡失败'
  } finally {
    submitting.value = false
  }
}

const resetPromotionForm = () => {
  promotionForm.value = {
    title: '',
    reward_value: '',
    reward_card_quantity: '',
    reward_threshold: '',
    promo_price_yuan: '',
    promo_quantity: '',
    promo_ends_at: ''
  }
}

const buildPromotionPayload = () => {
  if (!selectedTemplate.value) return null
  const isBalance = selectedTemplate.value.card_type === 'balance'
  return {
    card_template_id: selectedTemplate.value.id,
    title: promotionForm.value.title || '',
    reward_total_times: isBalance ? 0 : Number(promotionForm.value.reward_value || 0),
    reward_recharge_amount: isBalance ? Math.round(Number(promotionForm.value.reward_value || 0) * 100) : 0,
    reward_card_quantity: Number(promotionForm.value.reward_card_quantity || 0),
    reward_threshold: Number(promotionForm.value.reward_threshold || 0),
    promo_price: promotionForm.value.promo_price_yuan === '' ? 0 : Math.round(Number(promotionForm.value.promo_price_yuan || 0) * 100),
    promo_quantity: Number(promotionForm.value.promo_quantity || 0),
    promo_ends_at: promotionForm.value.promo_ends_at || '',
    status: 'active'
  }
}

const getPromotionPayloadSignature = (payload) => {
  if (!payload) return ''
  return JSON.stringify(payload)
}

const canAutoSavePromotion = computed(() => {
  const payload = buildPromotionPayload()
  if (!promotionEnabled.value || !payload) return false
  if (payload.reward_card_quantity <= 0 || payload.reward_threshold <= 0) return false
  if (selectedTemplate.value?.card_type === 'balance') {
    if (payload.reward_recharge_amount <= 0) return false
  } else if (payload.reward_total_times <= 0) {
    return false
  }
  if (payload.promo_price > 0 && (payload.promo_quantity <= 0 || !payload.promo_ends_at)) return false
  return true
})

const schedulePromotionSave = () => {
  if (promotionSaveTimer) {
    clearTimeout(promotionSaveTimer)
    promotionSaveTimer = null
  }
  if (!canAutoSavePromotion.value) return
  const payload = buildPromotionPayload()
  const nextSignature = getPromotionPayloadSignature(payload)
  if (nextSignature && nextSignature === lastSavedPromotionPayload.value) return
  promotionSaveTimer = setTimeout(() => {
    savePromotionCampaign()
  }, 450)
}

const togglePromotionEnabled = () => {
  promotionEnabled.value = !promotionEnabled.value
  if (promotionEnabled.value) {
    editingCampaignId.value = 0
    lastSavedPromotionPayload.value = ''
    resetPromotionForm()
  }
}

const formatDateTimeLocal = (value) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const offset = date.getTimezoneOffset()
  const local = new Date(date.getTime() - offset * 60000)
  return local.toISOString().slice(0, 16)
}

const isCampaignCurrentlyValid = (campaign) => {
  if (!campaign || campaign.status !== 'active') return false
  if (!campaign.promo_ends_at) return true
  const endAt = new Date(campaign.promo_ends_at)
  if (Number.isNaN(endAt.getTime())) return true
  return endAt.getTime() > Date.now()
}

const getPosterStorageKey = (campaign) => {
  const id = Number(campaign?.id || 0)
  if (!id) return ''
  return `merchant_promotion_poster_${id}`
}

const loadSavedPosterData = (campaign) => {
  const key = getPosterStorageKey(campaign)
  if (!key) {
    savedPosterDataUrl.value = ''
    return
  }
  savedPosterDataUrl.value = localStorage.getItem(key) || ''
}

const loadPromotionCampaignDetail = async (campaignId) => {
  const id = Number(campaignId || 0)
  if (!id) return null
  const res = await shopApi.getPromotionCampaign(id)
  return res?.data?.data || null
}

const loadCurrentCampaign = async () => {
  if (!selectedTemplate.value) return
  try {
    const res = await shopApi.listPromotionCampaigns({ template_id: selectedTemplate.value.id })
    const list = res.data.data || []
    const latest = list.find(item => isCampaignCurrentlyValid(item)) || null
    const detail = latest?.id ? await loadPromotionCampaignDetail(latest.id) : null
    currentCampaign.value = detail
    promotionLink.value = detail ? `${window.location.origin}${detail.share_path || `/promo/${detail.slug}`}` : ''
    loadSavedPosterData(detail)
    editingCampaignId.value = 0
    lastSavedPromotionPayload.value = ''
    resetPromotionForm()
  } catch (_) {
    currentCampaign.value = null
    promotionLink.value = ''
    editingCampaignId.value = 0
    savedPosterDataUrl.value = ''
    lastSavedPromotionPayload.value = ''
    resetPromotionForm()
  }
}

const savePromotionCampaign = async () => {
  if (!selectedTemplate.value || promotionSaving.value || !canAutoSavePromotion.value) return
  promotionSaving.value = true
  promotionError.value = ''
  try {
    const payload = buildPromotionPayload()
    const payloadSignature = getPromotionPayloadSignature(payload)
    const res = editingCampaignId.value
      ? await shopApi.updatePromotionCampaign(editingCampaignId.value, payload)
      : await shopApi.createPromotionCampaign(payload)
    currentCampaign.value = res.data.data
    editingCampaignId.value = Number(res.data.data?.id || 0)
    lastSavedPromotionPayload.value = payloadSignature
    promotionLink.value = `${window.location.origin}${currentCampaign.value?.share_path || `/promo/${currentCampaign.value?.slug}`}`
    loadSavedPosterData(currentCampaign.value)
    promotionEnabled.value = true
  } catch (err) {
    promotionError.value = err.response?.data?.error || '生成推广链接失败'
  } finally {
    promotionSaving.value = false
  }
}

watch(
  () => promotionEnabled.value,
  (enabled) => {
    if (!enabled && promotionSaveTimer) {
      clearTimeout(promotionSaveTimer)
      promotionSaveTimer = null
      return
    }
    if (enabled) {
      schedulePromotionSave()
    }
  }
)

watch(
  () => ({
    title: promotionForm.value.title,
    reward_value: promotionForm.value.reward_value,
    reward_card_quantity: promotionForm.value.reward_card_quantity,
    reward_threshold: promotionForm.value.reward_threshold,
    promo_price_yuan: promotionForm.value.promo_price_yuan,
    promo_quantity: promotionForm.value.promo_quantity,
    promo_ends_at: promotionForm.value.promo_ends_at,
    template_id: selectedTemplate.value?.id || 0,
    enabled: promotionEnabled.value
  }),
  () => {
    if (!promotionEnabled.value) return
    schedulePromotionSave()
  },
  { deep: true }
)

const copyPromotionLink = async () => {
  if (!promotionLink.value) return
  try {
    await navigator.clipboard.writeText(promotionLink.value)
    alert('已复制推广链接')
  } catch (_) {
    alert('复制失败，请手动复制')
  }
}

const openSavedPoster = () => {
  if (!savedPosterDataUrl.value) return
  const popup = window.open(savedPosterDataUrl.value, '_blank')
  if (!popup) {
    const link = document.createElement('a')
    link.href = savedPosterDataUrl.value
    link.target = '_blank'
    link.rel = 'noopener noreferrer'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }
}

const handlePosterAction = async () => {
  if (savedPosterDataUrl.value) {
    openSavedPoster()
    return
  }
  await generatePromotionPoster()
}

const formatPosterDateTime = (value) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).replace(/\//g, '/')
}

const wrapPosterText = (ctx, text, maxWidth) => {
  const value = String(text || '').trim()
  if (!value) return []
  const chars = Array.from(value)
  const lines = []
  let current = ''
  for (const char of chars) {
    const candidate = `${current}${char}`
    if (ctx.measureText(candidate).width <= maxWidth || !current) {
      current = candidate
      continue
    }
    lines.push(current)
    current = char
  }
  if (current) lines.push(current)
  return lines
}

const drawRoundedRect = (ctx, x, y, width, height, radius, fillStyle, strokeStyle = '') => {
  const r = Math.min(radius, width / 2, height / 2)
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + width, y, x + width, y + height, r)
  ctx.arcTo(x + width, y + height, x, y + height, r)
  ctx.arcTo(x, y + height, x, y, r)
  ctx.arcTo(x, y, x + width, y, r)
  ctx.closePath()
  if (fillStyle) {
    ctx.fillStyle = fillStyle
    ctx.fill()
  }
  if (strokeStyle) {
    ctx.strokeStyle = strokeStyle
    ctx.stroke()
  }
}

const canvasToBlob = (canvas) => new Promise((resolve, reject) => {
  canvas.toBlob((blob) => {
    if (blob) {
      resolve(blob)
      return
    }
    reject(new Error('blob create failed'))
  }, 'image/png')
})

const downloadBlobImage = (blob, filename) => {
  const objectUrl = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = objectUrl
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  setTimeout(() => URL.revokeObjectURL(objectUrl), 1000)
}

const savePosterBlob = async (blob, filename) => {
  if (navigator.share && window.File) {
    try {
      const file = new File([blob], filename, { type: blob.type || 'image/png' })
      await navigator.share({ files: [file], title: '推广卡活动海报' })
      alert('图片已生成，请在系统面板中保存到相册')
      return
    } catch (_) {
      // 用户取消或系统不支持文件分享时，回退为下载
    }
  }
  downloadBlobImage(blob, filename)
  alert('图片已生成并开始下载')
}

const generatePromotionPoster = async () => {
  if (!currentCampaign.value || !promotionLink.value || generatingPoster.value) return

  generatingPoster.value = true
  try {
    const campaign = currentCampaign.value?.id
      ? (await loadPromotionCampaignDetail(currentCampaign.value.id)) || currentCampaign.value
      : currentCampaign.value
    if (!campaign?.card_template || !campaign?.merchant) {
      throw new Error('campaign detail missing')
    }
    currentCampaign.value = campaign

    const cardTemplate = campaign.card_template || {}
    const merchant = campaign.merchant || {}
    const cardType = getCardTypeLabel(cardTemplate.card_type)
    const showPromoPrice = Boolean(campaign.promo_active && Number(campaign.promo_price || 0) > 0)
    const viewportWidth = typeof window !== 'undefined'
      ? Number(window.innerWidth || document.documentElement?.clientWidth || 0)
      : 0
    const posterWidth = Math.min(Math.max(viewportWidth || 390, 375), 430)
    const scale = 3
    const pagePadding = 16
    const sectionGap = 16
    const sectionWidth = posterWidth - pagePadding * 2
    const sectionPadding = 16
    const measureCanvas = document.createElement('canvas')
    const measureCtx = measureCanvas.getContext('2d')
    if (!measureCtx) throw new Error('canvas unsupported')
    measureCtx.font = '400 14px sans-serif'
    const descriptionLines = wrapPosterText(measureCtx, cardTemplate.description || '', sectionWidth - sectionPadding * 2).slice(0, 2)
    const shareDesc = `邀请新用户注册成功 +1，首次付款成功再 +1，达到 ${Number(campaign.reward_threshold || 0)} 后可在3天内领取奖励卡。`
    const shareDescLines = wrapPosterText(measureCtx, shareDesc, sectionWidth - sectionPadding * 2).slice(0, 3)
    measureCtx.font = '700 23px sans-serif'
    const cardNameLines = wrapPosterText(measureCtx, cardTemplate.name || '', sectionWidth - sectionPadding * 2).slice(0, 2)
    const topSectionY = pagePadding
    const cardNameLineHeight = 30
    const cardNameFirstBaseline = topSectionY + 122
    const cardNameLastBaseline = cardNameFirstBaseline + Math.max(cardNameLines.length - 1, 0) * cardNameLineHeight
    const merchantBaseline = cardNameLastBaseline + 34
    const priceRowY = merchantBaseline + 50
    let topContentBottom = priceRowY
    let promoMetaY = 0
    if (showPromoPrice && Number(campaign.promo_remaining || 0) > 0 && campaign.promo_ends_at) {
      promoMetaY = priceRowY + 24
      topContentBottom = promoMetaY
    }
    const baseInfoY = topContentBottom + (showPromoPrice ? 28 : 30)
    topContentBottom = baseInfoY
    if (descriptionLines.length > 0) {
      const descriptionFirstBaseline = baseInfoY + 30
      const descriptionLastBaseline = descriptionFirstBaseline + Math.max(descriptionLines.length - 1, 0) * 20
      topContentBottom = descriptionLastBaseline
    }
    const topSectionHeight = topContentBottom - topSectionY + 28

    const purchaseSectionY = topSectionY + topSectionHeight + sectionGap
    const optionCardY = purchaseSectionY + 62
    const optionCardHeight = 78
    const optionCardBottom = optionCardY + optionCardHeight
    let purchaseContentBottom = optionCardBottom
    let promoButtonY = 0
    let storeButtonY = 0
    if (showPromoPrice) {
      promoButtonY = optionCardBottom + 18
      const promoButtonBottom = promoButtonY + 66
      storeButtonY = promoButtonBottom + 18
      purchaseContentBottom = storeButtonY + 66
    }
    const purchaseSectionHeight = purchaseContentBottom - purchaseSectionY + sectionPadding

    const shareSectionY = purchaseSectionY + purchaseSectionHeight + sectionGap
    const shareDescFirstBaseline = shareSectionY + 74
    const shareDescLastBaseline = shareDescFirstBaseline + Math.max(shareDescLines.length - 1, 0) * 22
    const shareButtonY = shareDescLastBaseline + 24
    const shareSectionHeight = shareButtonY + 60 - shareSectionY + sectionPadding

    const qrSectionY = shareSectionY + shareSectionHeight + sectionGap
    const qrCardHeight = 212
    const qrCardY = qrSectionY + 76
    const qrSectionHeight = qrCardY + qrCardHeight - qrSectionY + sectionPadding
    const posterHeight = pagePadding + topSectionHeight + sectionGap + purchaseSectionHeight + sectionGap + shareSectionHeight + sectionGap + qrSectionHeight + pagePadding
    const canvas = document.createElement('canvas')
    canvas.width = posterWidth * scale
    canvas.height = posterHeight * scale
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('canvas unsupported')
    ctx.scale(scale, scale)

    ctx.fillStyle = '#f7f4ef'
    ctx.fillRect(0, 0, posterWidth, posterHeight)
    drawRoundedRect(ctx, pagePadding, topSectionY, sectionWidth, topSectionHeight, 26, '#ffffff')
    ctx.fillStyle = '#fb923c'
    ctx.font = '500 16px sans-serif'
    ctx.fillText('推广卡活动', pagePadding + sectionPadding, topSectionY + 28)

    const titleBarX = pagePadding + sectionPadding
    const titleBarY = topSectionY + 42
    const titleBarWidth = sectionWidth - sectionPadding * 2
    const titleBarHeight = 54
    const gradient = ctx.createLinearGradient(titleBarX, titleBarY, titleBarX + titleBarWidth, titleBarY)
    gradient.addColorStop(0, '#f97316')
    gradient.addColorStop(1, '#fbbf24')
    drawRoundedRect(ctx, titleBarX, titleBarY, titleBarWidth, titleBarHeight, 20, gradient)
    const posterTitle = String(campaign.title || '推广卡活动').trim()
    const posterBang = '!'
    ctx.font = '700 22px sans-serif'
    const titleWidth = ctx.measureText(posterTitle).width
    ctx.font = '900 28px sans-serif'
    const bangWidth = ctx.measureText(posterBang).width
    const titleGroupGap = 2
    const titleGroupWidth = titleWidth + titleGroupGap + bangWidth
    const titleGroupStartX = titleBarX + (titleBarWidth - titleGroupWidth) / 2
    ctx.fillStyle = '#ffffff'
    ctx.font = '700 22px sans-serif'
    ctx.textAlign = 'start'
    ctx.textBaseline = 'middle'
    ctx.fillText(posterTitle, titleGroupStartX, titleBarY + titleBarHeight / 2)
    ctx.save()
    ctx.fillStyle = '#ff5a36'
    ctx.font = '900 28px sans-serif'
    ctx.translate(titleGroupStartX + titleWidth + titleGroupGap, titleBarY + titleBarHeight / 2 - 2)
    ctx.rotate(12 * Math.PI / 180)
    ctx.fillText(posterBang, 0, 0)
    ctx.restore()
    ctx.textAlign = 'start'
    ctx.textBaseline = 'alphabetic'

    ctx.fillStyle = '#1f2937'
    ctx.font = '700 23px sans-serif'
    cardNameLines.forEach((line, index) => {
      ctx.fillText(line, pagePadding + sectionPadding, cardNameFirstBaseline + index * cardNameLineHeight)
    })
    ctx.fillStyle = '#6b7280'
    ctx.font = '400 15px sans-serif'
    ctx.fillText(`${merchant.name || ''} · ${cardType}`, pagePadding + sectionPadding, merchantBaseline)

    const originalPrice = Number(cardTemplate.price || 0)
    const promoPrice = Number(showPromoPrice ? campaign.promo_price || 0 : cardTemplate.price || 0)
    let priceStartX = pagePadding + sectionPadding
    if (showPromoPrice && originalPrice > 0) {
      const originalPriceText = `¥${(originalPrice / 100).toFixed(2)}`
      ctx.fillStyle = '#9ca3af'
      ctx.font = '400 15px sans-serif'
      ctx.fillText(originalPriceText, priceStartX, priceRowY)
      const strikeWidth = ctx.measureText(originalPriceText).width
      ctx.strokeStyle = '#9ca3af'
      ctx.lineWidth = 1.5
      ctx.beginPath()
      ctx.moveTo(priceStartX, priceRowY - 7)
      ctx.lineTo(priceStartX + strikeWidth, priceRowY - 7)
      ctx.stroke()
      priceStartX += strikeWidth + 18
    }
    ctx.fillStyle = '#f97316'
    ctx.font = '700 29px sans-serif'
    ctx.fillText(`¥${(promoPrice / 100).toFixed(2)}`, priceStartX, priceRowY)

    if (promoMetaY > 0) {
      ctx.fillStyle = '#f97316'
      ctx.font = '500 14px sans-serif'
      ctx.fillText(`促销剩余 ${campaign.promo_remaining} 份，截止 ${formatPosterDateTime(campaign.promo_ends_at)}`, pagePadding + sectionPadding, promoMetaY)
    }

    const baseInfo = cardTemplate.card_type === 'balance'
      ? `原卡额度：¥${(Number(cardTemplate.recharge_amount || 0) / 100).toFixed(2)}`
      : `原卡次数：${Number(cardTemplate.total_times || 0)}`
    const validity = Number(cardTemplate.valid_days || 0) > 0 ? `${cardTemplate.valid_days}天有效` : '长期有效'
    ctx.fillStyle = '#374151'
    ctx.font = '400 16px sans-serif'
    ctx.fillText(`${baseInfo}   ${validity}`, pagePadding + sectionPadding, baseInfoY)
    if (descriptionLines.length > 0) {
      ctx.fillStyle = '#6b7280'
      ctx.font = '400 14px sans-serif'
      descriptionLines.forEach((line, index) => {
        ctx.fillText(line, pagePadding + sectionPadding, baseInfoY + 30 + index * 20)
      })
    }

    drawRoundedRect(ctx, pagePadding, purchaseSectionY, sectionWidth, purchaseSectionHeight, 26, '#ffffff')
    ctx.fillStyle = '#1f2937'
    ctx.font = '700 20px sans-serif'
    ctx.fillText('购买方式', pagePadding + sectionPadding, purchaseSectionY + 38)

    const optionGap = 12
    const optionWidth = (sectionWidth - sectionPadding * 2 - optionGap) / 2
    drawRoundedRect(ctx, pagePadding + sectionPadding, optionCardY, optionWidth, optionCardHeight, 18, '#ffffff', '#d9deea')
    drawRoundedRect(ctx, pagePadding + sectionPadding + optionWidth + optionGap, optionCardY, optionWidth, optionCardHeight, 18, '#ffffff', '#d9deea')
    ctx.fillStyle = '#1f2937'
    ctx.font = '600 16px sans-serif'
    ctx.fillText('原价支付宝购买', pagePadding + sectionPadding + 14, optionCardY + 31)
    ctx.fillText('原价微信购买', pagePadding + sectionPadding + optionWidth + optionGap + 14, optionCardY + 31)
    ctx.fillStyle = '#6b7280'
    ctx.font = '400 12px sans-serif'
    ctx.fillText('始终可用', pagePadding + sectionPadding + 14, optionCardY + 53)
    ctx.fillText('始终可用', pagePadding + sectionPadding + optionWidth + optionGap + 14, optionCardY + 53)

    if (showPromoPrice) {
      drawRoundedRect(ctx, pagePadding + sectionPadding, promoButtonY, sectionWidth - sectionPadding * 2, 66, 18, '#ff6d00')
      ctx.fillStyle = '#ffffff'
      ctx.font = '700 18px sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.fillText('促销价线上购买', posterWidth / 2, promoButtonY + 33)
      ctx.textAlign = 'start'
      ctx.textBaseline = 'alphabetic'

      drawRoundedRect(ctx, pagePadding + sectionPadding, storeButtonY, sectionWidth - sectionPadding * 2, 66, 18, '#ffffff', '#f59e0b')
      ctx.fillStyle = '#ea580c'
      ctx.font = '700 17px sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.fillText('领取活动后到店付款', posterWidth / 2, storeButtonY + 33)
      ctx.textAlign = 'start'
      ctx.textBaseline = 'alphabetic'
    }

    drawRoundedRect(ctx, pagePadding, shareSectionY, sectionWidth, shareSectionHeight, 26, '#ffffff')
    ctx.fillStyle = '#1f2937'
    ctx.font = '700 20px sans-serif'
    ctx.fillText('转发领卡', pagePadding + sectionPadding, shareSectionY + 38)
    ctx.fillStyle = '#6b7280'
    ctx.font = '400 14px sans-serif'
    shareDescLines.forEach((line, index) => {
      ctx.fillText(line, pagePadding + sectionPadding, shareDescFirstBaseline + index * 22)
    })
    drawRoundedRect(ctx, pagePadding + sectionPadding, shareButtonY, sectionWidth - sectionPadding * 2, 60, 18, '#16a34a')
    ctx.fillStyle = '#ffffff'
    ctx.font = '700 18px sans-serif'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText('转发领卡', posterWidth / 2, shareButtonY + 30)
    ctx.textAlign = 'start'
    ctx.textBaseline = 'alphabetic'

    drawRoundedRect(ctx, pagePadding, qrSectionY, sectionWidth, qrSectionHeight, 26, '#ffffff')
    ctx.fillStyle = '#1f2937'
    ctx.font = '700 18px sans-serif'
    ctx.fillText('活动页面二维码', pagePadding + sectionPadding, qrSectionY + 34)
    ctx.fillStyle = '#6b7280'
    ctx.font = '400 13px sans-serif'
    ctx.fillText('微信识别二维码打开当前推广卡活动页', pagePadding + sectionPadding, qrSectionY + 58)

    const qrCodeSize = 148
    const qrDataUrl = await QRCode.toDataURL(promotionLink.value, {
      width: qrCodeSize,
      margin: 1,
      color: {
        dark: '#111827',
        light: '#ffffff'
      }
    })
    const qrImage = await new Promise((resolve, reject) => {
      const img = new Image()
      img.onload = () => resolve(img)
      img.onerror = reject
      img.src = qrDataUrl
    })

    const qrCardWidth = 210
    const qrCardX = (posterWidth - qrCardWidth) / 2
    drawRoundedRect(ctx, qrCardX, qrCardY, qrCardWidth, qrCardHeight, 22, '#f9fafb')
    ctx.drawImage(qrImage, (posterWidth - qrCodeSize) / 2, qrCardY + 16, qrCodeSize, qrCodeSize)
    ctx.fillStyle = '#374151'
    ctx.font = '500 12px sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText('微信识别二维码打开活动页', posterWidth / 2, qrCardY + 188)
    ctx.textAlign = 'start'

    const dataUrl = canvas.toDataURL('image/png')
    const blob = await canvasToBlob(canvas)
    const filename = `promotion_poster_${campaign.slug || Date.now()}.png`
    const storageKey = getPosterStorageKey(campaign)
    if (storageKey) {
      localStorage.setItem(storageKey, dataUrl)
      savedPosterDataUrl.value = dataUrl
    }
    await savePosterBlob(blob, filename)
  } catch (err) {
    alert('生成图片失败')
  } finally {
    generatingPoster.value = false
  }
}

onMounted(async () => {
  ensureMerchantLogin()
  if (!hasMerchantPermission('merchant.card.issue')) {
    alert('您没有发卡权限，请联系管理员开通')
    goBack()
    return
  }
  await loadTemplates()
  cardForm.value.start_date = new Date().toISOString().split('T')[0]
})
</script>
