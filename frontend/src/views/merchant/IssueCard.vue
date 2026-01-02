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
            <label class="block text-gray-700 text-sm font-medium mb-2">卡类型</label>
            <input
              v-model="cardForm.card_type"
              type="text"
              placeholder="如：洗剪吹10次卡"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-gray-700 text-sm font-medium mb-2">总次数</label>
              <input
                v-model.number="cardForm.total_times"
                type="number"
                min="1"
                placeholder="如：10"
                class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              />
            </div>
            <div>
              <label class="block text-gray-700 text-sm font-medium mb-2">充值金额（元）</label>
              <input
                v-model.number="cardForm.recharge_amount"
                type="number"
                min="0"
                placeholder="可选"
                class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              />
            </div>
          </div>

          <div>
            <label class="block text-gray-700 text-sm font-medium mb-2">卡号（可选）</label>
            <input
              v-model="cardForm.card_no"
              type="text"
              placeholder="不填则自动生成"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
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
                class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
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
import { cardApi, merchantApi } from '../../api'

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
  card_type: '',
  total_times: 10,
  recharge_amount: 0,
  card_no: '',
  start_date: '',
  end_date: ''
})

// 监听开始日期变化，自动设置结束日期为两年后
watch(() => cardForm.value.start_date, (newStartDate) => {
  if (newStartDate) {
    const startDate = new Date(newStartDate)
    const endDate = new Date(startDate)
    endDate.setFullYear(endDate.getFullYear() + 2)
    cardForm.value.end_date = endDate.toISOString().split('T')[0]
  }
})

const canSubmit = computed(() => {
  return Boolean(
    selectedUser.value &&
      selectedUser.value.id &&
      cardForm.value.card_type &&
      cardForm.value.total_times &&
      Number(cardForm.value.total_times) > 0 &&
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
    const payload = {
      user_id: selectedUser.value.id,
      card_type: cardForm.value.card_type,
      total_times: Number(cardForm.value.total_times),
      recharge_amount: Number(cardForm.value.recharge_amount) || 0,
      card_no: cardForm.value.card_no || undefined,
      start_date: cardForm.value.start_date || undefined,
      end_date: cardForm.value.end_date
    }

    const res = await cardApi.createCard(payload)
    const card = res.data.data
    alert(`发卡成功：卡号 ${card?.card_no || ''}`)
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
  // 设置默认开始日期为今天
  const today = new Date().toISOString().split('T')[0]
  cardForm.value.start_date = today
  // 结束日期会通过watch自动设置为两年后
})
</script>
