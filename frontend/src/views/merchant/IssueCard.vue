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

        <div v-if="searchError" class="mt-3 text-sm text-gray-700">
          {{ searchError }}
        </div>

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

        <div v-else-if="searched" class="mt-4 text-center text-gray-400">
          未找到用户
        </div>
      </div>

      <div class="bg-white rounded-xl p-4 shadow-sm" v-if="selectedUser">
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
              <option
                v-for="tpl in templates"
                :key="tpl.id"
                :value="tpl.id"
              >
                {{ tpl.name }}（¥{{ (tpl.price / 100).toFixed(2) }}）
              </option>
            </select>
          </div>

          <div>
            <label class="block text-gray-700 text-sm font-medium mb-2">卡号</label>
            <input
              v-model="cardForm.card_no"
              type="text"
              disabled
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary bg-gray-50 text-gray-500"
            />
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
            @click="submit"
            :disabled="submitting || !canSubmit"
            class="w-full mt-2 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
          >
            {{ submitting ? '提交中...' : '确认发卡' }}
          </button>

          <div v-if="submitError" class="text-sm text-gray-700">
            {{ submitError }}
          </div>
          <div v-if="submitSuccess" class="text-sm text-primary">
            {{ submitSuccess }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { cardApi, merchantApi, shopApi } from '../../api'

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

const cardForm = ref({
  template_id: 0,
  card_no: '',
  start_date: '',
  end_date: ''
})

const templates = ref([])
const selectedTemplate = computed(() => {
  const id = Number(cardForm.value.template_id || 0)
  if (!id) return null
  return (templates.value || []).find(t => Number(t.id) === id) || null
})

const loadTemplates = async () => {
  try {
    const res = await shopApi.getCardTemplates()
    const list = res.data.data || []
    templates.value = list.filter(t => t && (t.card_type === 'times' || t.card_type === 'lesson'))
  } catch (e) {
    templates.value = []
  }
}

const loadNextCardNo = async () => {
  try {
    const res = await merchantApi.getNextCardNo()
    cardForm.value.card_no = res?.data?.data?.card_no || ''
  } catch (_) {
    cardForm.value.card_no = ''
  }
}

const calcEndDate = (startDateStr, validDays) => {
  const base = startDateStr ? new Date(startDateStr) : new Date()
  const end = new Date(base)
  const days = Number(validDays || 0)
  if (days > 0) {
    end.setDate(end.getDate() + days)
  } else {
    end.setFullYear(end.getFullYear() + 100)
  }
  return end.toISOString().split('T')[0]
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

const canSubmit = computed(() => {
  return Boolean(
    selectedUser.value &&
      selectedUser.value.id &&
      selectedTemplate.value &&
      cardForm.value.end_date
  )
})

const goBack = () => {
  router.back()
}

const ensureMerchantLogin = () => {
  const storedMerchantId = localStorage.getItem('merchantId')
  const storedToken = localStorage.getItem('merchantToken')

  if (!storedMerchantId || !storedToken) {
    router.replace('/merchant/login')
    return false
  }

  const parsedMerchantId = Number.parseInt(storedMerchantId, 10)
  if (Number.isNaN(parsedMerchantId) || parsedMerchantId <= 0) {
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
    if (!tpl) {
      submitError.value = '请选择卡片模板'
      return
    }

    const startDate = cardForm.value.start_date || new Date().toISOString().split('T')[0]
    const endDate = calcEndDate(startDate, tpl.valid_days)

    const payload = {
      user_id: selectedUser.value.id,
      card_type: tpl.name,
      total_times: Number(tpl.total_times) || 0,
      recharge_amount: Math.round((Number(tpl.recharge_amount) || 0) / 100),
      start_date: startDate,
      end_date: endDate
    }

    const res = await cardApi.createCard(payload)
    const card = res.data.data
    alert(`发卡成功：卡号 ${card?.card_no || ''}`)
    await loadNextCardNo()
    // 跳转到卡片管理页
    router.push('/merchant?tab=cards')
  } catch (err) {
    submitError.value = err.response?.data?.error || '发卡失败'
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  ensureMerchantLogin()
  loadTemplates()
  loadNextCardNo()
  // 设置默认开始日期为今天
  const today = new Date().toISOString().split('T')[0]
  cardForm.value.start_date = today
})
</script>
