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
              次数：{{ selectedTemplate.total_times }}
            </div>
            <div v-else class="text-sm text-gray-700">额度：¥{{ (selectedTemplate.recharge_amount / 100).toFixed(2) }}</div>
            <div class="text-sm text-gray-700">售价：¥{{ (selectedTemplate.price / 100).toFixed(2) }}</div>
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
              <div class="mt-1 text-xs leading-4 text-gray-500">未输入手机号时，可直接生成推广活动链接</div>
            </div>

            <div v-if="promotionEnabled" class="space-y-3">
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
              <div v-if="promotionLink" class="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-gray-700">
                <div class="font-medium text-green-700 mb-1">推广链接已生成</div>
                <div class="break-all">{{ promotionLink }}</div>
                <button @click="copyPromotionLink" class="mt-2 text-primary text-sm">复制链接</button>
              </div>
            </div>
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
let promotionSaveTimer = null
let isFillingPromotionForm = false
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
}

const fillPromotionForm = (campaign) => {
  if (!campaign || !selectedTemplate.value) {
    resetPromotionForm()
    return
  }
  const isBalance = selectedTemplate.value.card_type === 'balance'
  isFillingPromotionForm = true
  promotionForm.value = {
    title: campaign.title || '',
    reward_value: isBalance ? ((campaign.reward_recharge_amount || 0) / 100) : (campaign.reward_total_times || ''),
    reward_card_quantity: campaign.reward_card_quantity || '',
    reward_threshold: campaign.reward_threshold || '',
    promo_price_yuan: campaign.promo_price ? (campaign.promo_price / 100) : '',
    promo_quantity: campaign.promo_quantity || '',
    promo_ends_at: campaign.promo_ends_at ? formatDateTimeLocal(campaign.promo_ends_at) : ''
  }
  const payload = buildPromotionPayload()
  lastSavedPromotionPayload.value = getPromotionPayloadSignature(payload)
  promotionLink.value = `${window.location.origin}${campaign.share_path || `/promo/${campaign.slug}`}`
  setTimeout(() => {
    isFillingPromotionForm = false
  }, 0)
}

const formatDateTimeLocal = (value) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const offset = date.getTimezoneOffset()
  const local = new Date(date.getTime() - offset * 60000)
  return local.toISOString().slice(0, 16)
}

const loadCurrentCampaign = async () => {
  if (!selectedTemplate.value) return
  try {
    const res = await shopApi.listPromotionCampaigns({ template_id: selectedTemplate.value.id })
    const list = res.data.data || []
    const latest = list.find(item => item.status === 'active') || list[0] || null
    currentCampaign.value = latest
    fillPromotionForm(latest)
  } catch (_) {
    currentCampaign.value = null
    promotionLink.value = ''
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
    const res = currentCampaign.value
      ? await shopApi.updatePromotionCampaign(currentCampaign.value.id, payload)
      : await shopApi.createPromotionCampaign(payload)
    currentCampaign.value = res.data.data
    lastSavedPromotionPayload.value = payloadSignature
    fillPromotionForm(currentCampaign.value)
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
    if (isFillingPromotionForm) return
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
