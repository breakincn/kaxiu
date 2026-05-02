<template>
  <div class="min-h-screen bg-[#f6efe4] text-gray-800">
    <div v-if="loading" class="min-h-screen flex items-center justify-center">加载中...</div>
    <div v-else-if="errorMessage" class="min-h-screen flex items-center justify-center">{{ errorMessage }}</div>
    <div v-else-if="!landing" class="min-h-screen flex items-center justify-center">推广页不存在</div>
    <div v-else class="max-w-3xl mx-auto px-4 py-6 space-y-4">
      <div class="rounded-[28px] overflow-hidden bg-gradient-to-br from-[#ff8a34] via-[#ffb24a] to-[#ffd16d] text-white p-6 shadow-[0_20px_60px_rgba(255,138,52,0.25)]">
        <div class="text-xs uppercase tracking-[0.35em] text-white/80">Kabao For Merchant</div>
        <h1 class="mt-3 text-3xl font-black leading-tight">卡包商户入驻</h1>
        <p class="mt-3 text-sm leading-6 text-white/90">{{ landing.subtitle }}</p>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <div
          v-for="item in landing.highlights || []"
          :key="item.title"
          class="rounded-[24px] bg-white p-5 shadow-sm border border-[#f1e2cf]"
        >
          <div class="text-sm font-bold text-[#ff7b23]">{{ item.title }}</div>
          <div class="mt-2 text-sm leading-6 text-gray-600">{{ item.description }}</div>
        </div>
      </div>

      <div class="rounded-[28px] bg-white p-5 shadow-sm border border-[#f1e2cf] space-y-4">
        <div>
          <div class="text-lg font-bold">开始使用卡包</div>
          <div class="mt-1 text-sm text-gray-500">注册商户后自动绑定推广关系，后续付费将按约定进入分成台账。</div>
        </div>

        <div class="rounded-2xl bg-[#fff8ef] border border-[#f5e0bf] px-4 py-3">
          <div class="text-xs text-gray-500">推广码</div>
          <div class="mt-1 text-base font-bold tracking-[0.15em] text-[#ff7b23]">{{ landing.promotion_code }}</div>
        </div>

        <div class="grid gap-3 md:grid-cols-[1fr_auto] items-center">
          <div class="min-w-0">
            <div class="text-xs text-gray-500">商户注册入口</div>
            <div class="mt-1 break-all text-sm text-gray-700">{{ landing.register_url }}</div>
          </div>
          <button
            type="button"
            class="rounded-2xl bg-[#ff7b23] px-5 py-3 text-sm font-medium text-white"
            @click="goRegister"
          >
            去注册商户
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { authApi } from '../../api'

const route = useRoute()
const loading = ref(true)
const landing = ref(null)
const errorMessage = ref('')

const resolveLandingErrorMessage = (err) => {
  const status = err?.response?.status
  const message = String(err?.response?.data?.error || '').trim()
  if (status === 404) {
    return message || '推广页不存在'
  }
  return message || '推广页加载失败'
}

const loadLanding = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await authApi.getMerchantReferralLanding(route.params.refCode)
    landing.value = res.data?.data || null
  } catch (err) {
    landing.value = null
    errorMessage.value = resolveLandingErrorMessage(err)
    alert(errorMessage.value)
  } finally {
    loading.value = false
  }
}

const goRegister = () => {
  if (!landing.value?.register_url) return
  window.location.href = landing.value.register_url
}

onMounted(loadLanding)
</script>
