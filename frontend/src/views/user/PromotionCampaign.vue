<template>
  <div class="min-h-screen bg-[#f7f4ef] text-gray-800">
    <div v-if="loading" class="min-h-screen flex items-center justify-center">加载中...</div>
    <div v-else-if="!campaign" class="min-h-screen flex items-center justify-center">活动不存在</div>
    <div v-else class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <div class="rounded-3xl bg-white shadow-sm p-5 space-y-3">
        <div class="text-xs uppercase tracking-[0.3em] text-orange-400">推广卡活动</div>
        <div v-if="campaign.title" class="rounded-2xl bg-gradient-to-r from-orange-500 to-amber-400 text-white p-4 text-xl font-semibold flex items-center justify-center gap-0.5 text-center">
          <span>{{ campaign.title }}</span>
          <span
            class="inline-flex items-start justify-center w-6 h-6 text-[#ff3b30] text-[28px] leading-none font-black rotate-12 -translate-y-[2px] -ml-[4px]"
            aria-hidden="true"
          >!</span>
        </div>
        <div class="text-2xl font-semibold">{{ campaign.card_template.name }}</div>
        <div class="text-sm text-gray-500">
          {{ campaign.merchant.name }} · {{ getCardTypeLabel(campaign.card_template.card_type) }}
        </div>
        <div class="flex items-end gap-3">
          <div v-if="showPromoPrice" class="text-sm text-gray-400 line-through">
            ¥{{ (campaign.card_template.price / 100).toFixed(2) }}
          </div>
          <div class="text-3xl font-bold text-orange-500">
            ¥{{ ((showPromoPrice ? campaign.promo_price : campaign.card_template.price) / 100).toFixed(2) }}
          </div>
        </div>
        <div v-if="showPromoPrice" class="text-sm text-orange-500">
          促销剩余 {{ campaign.promo_remaining }} 份，截止 {{ formatDateTime(campaign.promo_ends_at) }}
        </div>
        <div class="text-sm text-gray-600">
          <span v-if="campaign.card_template.card_type === 'balance'">原卡额度：¥{{ (campaign.card_template.recharge_amount / 100).toFixed(2) }}</span>
          <span v-else>原卡次数：{{ campaign.card_template.total_times }}</span>
          <span class="ml-3">{{ campaign.card_template.valid_days > 0 ? `${campaign.card_template.valid_days}天有效` : '长期有效' }}</span>
        </div>
        <div v-if="campaign.card_template.description" class="text-sm text-gray-500">
          {{ campaign.card_template.description }}
        </div>
      </div>

      <div class="rounded-3xl bg-white shadow-sm p-5 space-y-3">
        <div class="font-semibold">购买方式</div>
        <div class="grid grid-cols-2 gap-3">
          <button class="rounded-2xl border border-gray-200 px-4 py-3 text-left" @click="createOriginalPurchase('alipay')">
            <div class="font-medium">原价支付宝购买</div>
            <div class="text-xs text-gray-500">始终可用</div>
          </button>
          <button class="rounded-2xl border border-gray-200 px-4 py-3 text-left" @click="createOriginalPurchase('wechat')">
            <div class="font-medium">原价微信购买</div>
            <div class="text-xs text-gray-500">始终可用</div>
          </button>
        </div>

        <template v-if="showPromoPrice">
          <button class="w-full rounded-2xl bg-orange-500 text-white px-4 py-3 font-medium" @click="createPromoPurchase">
            促销价线上购买
          </button>
          <button
            v-if="campaign.can_claim_promo || hasActiveClaim"
            class="w-full rounded-2xl border border-orange-300 text-orange-500 px-4 py-3 font-medium"
            @click="claimOrCreateStoreOrder"
          >
            {{ hasActiveClaim ? '到店付款开卡' : '领取活动后到店付款' }}
          </button>
        </template>

        <div v-if="campaign.my_claim" class="text-sm text-gray-500">
          当前活动资格：{{ formatClaimStatus(campaign.my_claim) }}
        </div>
      </div>

      <div v-if="campaign.can_share || campaign.my_referral" class="rounded-3xl bg-white shadow-sm p-5 space-y-3">
        <div class="font-semibold">转发领卡</div>
        <div class="text-sm text-gray-500">
          邀请新用户注册成功 +1，首次付款成功再 +1，达到 {{ campaign.reward_threshold }} 后可在3天内领取奖励卡。
        </div>
        <button
          v-if="campaign.can_share"
          class="w-full rounded-2xl bg-green-600 text-white px-4 py-3 font-medium"
          @click="handleShareAction"
        >
          转发领卡
        </button>
        <div v-if="campaign.my_referral" class="rounded-2xl bg-gray-50 p-4 text-sm space-y-1">
          <div>注册数：{{ campaign.my_referral.register_count }}</div>
          <div>付款数：{{ campaign.my_referral.paid_count }}</div>
          <div>累计进度：{{ campaign.my_referral.progress_count }}</div>
          <div v-if="campaign.my_referral.reward_status === 'claimable'" class="text-orange-500">
            奖励待领取，截止 {{ formatDateTime(campaign.my_referral.reward_expires_at) }}
          </div>
          <div v-if="shareLink" class="break-all text-primary">{{ shareLink }}</div>
          <button
            v-if="campaign.my_referral.reward_status === 'claimable'"
            class="mt-2 rounded-xl bg-orange-500 text-white px-4 py-2"
            @click="claimReward"
          >
            领取奖励卡
          </button>
        </div>
      </div>

      <div v-if="paymentModalVisible" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-30" @click.self="closePaymentModal">
        <div class="w-full max-w-sm rounded-3xl bg-white p-5 space-y-4">
          <div class="text-lg font-semibold">完成付款</div>
          <div class="text-sm text-gray-500">支付金额：¥{{ ((currentOrder?.price || 0) / 100).toFixed(2) }}</div>
          <img v-if="paymentUrl" :src="paymentUrl" alt="收款码" class="w-full rounded-2xl border border-gray-200" />
          <div class="grid grid-cols-2 gap-3">
            <button class="rounded-2xl border border-gray-200 px-4 py-3" @click="closePaymentModal">取消</button>
            <button class="rounded-2xl bg-orange-500 text-white px-4 py-3" @click="confirmPayment">
              已完成付款
            </button>
          </div>
        </div>
      </div>

      <div v-if="loginModalVisible" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-30" @click.self="loginModalVisible = false">
        <div class="w-full max-w-sm rounded-3xl bg-white p-5 space-y-4">
          <div class="text-lg font-semibold">请先注册或登录</div>
          <div class="text-sm text-gray-500">登录后即可转发领卡、领取活动或购买卡片。</div>
          <div class="grid grid-cols-2 gap-3">
            <button class="rounded-2xl border border-gray-200 px-4 py-3" @click="goRegister">去注册</button>
            <button class="rounded-2xl bg-gray-900 text-white px-4 py-3" @click="goLogin">去登录</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { shopApi } from '../../api'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const campaign = ref(null)
const currentOrder = ref(null)
const paymentUrl = ref('')
const paymentModalVisible = ref(false)
const loginModalVisible = ref(false)
const shareLink = ref('')

const isLoggedIn = computed(() => Boolean(localStorage.getItem('userToken')))
const hasActiveClaim = computed(() => campaign.value?.my_claim?.status === 'active')
const showPromoPrice = computed(() => Boolean(campaign.value?.promo_active && campaign.value?.promo_price > 0))

const loadCampaign = async () => {
  loading.value = true
  try {
    const res = await shopApi.getPublicPromotionCampaign(route.params.slug, route.params.refCode)
    campaign.value = res.data.data
  } catch (err) {
    campaign.value = null
    alert(err.response?.data?.error || '活动加载失败')
  } finally {
    loading.value = false
  }
}

const ensureUserAction = () => {
  if (isLoggedIn.value) return true
  localStorage.setItem('redirectAfterLogin', route.fullPath)
  localStorage.setItem('promotionReferralContext', JSON.stringify({
    campaignSlug: route.params.slug,
    referralCode: String(route.params.refCode || '').trim()
  }))
  loginModalVisible.value = true
  return false
}

const getCardTypeLabel = (type) => {
  const labels = { times: '次数卡', lesson: '课时卡', balance: '充值卡' }
  return labels[type] || type
}

const formatDateTime = (value) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('zh-CN')
}

const formatClaimStatus = (claim) => {
  if (!claim) return ''
  if (claim.status === 'active') return `已领取，24小时内有效，截止 ${formatDateTime(claim.expires_at)}`
  if (claim.status === 'paid') return '已付款，等待商户确认开卡'
  if (claim.status === 'confirmed') return '已完成活动购买'
  if (claim.status === 'expired') return '已过期'
  return claim.status
}

const createOriginalPurchase = async (method) => {
  if (!ensureUserAction()) return
  try {
    const res = await shopApi.createDirectPurchase({
      card_template_id: campaign.value.card_template_id,
      campaign_id: campaign.value.id,
      referral_code: route.params.refCode || '',
      use_promo: false,
      payment_method: method
    })
    currentOrder.value = res.data.data
    paymentUrl.value = res.data.data.payment_url || ''
    paymentModalVisible.value = true
  } catch (err) {
    alert(err.response?.data?.error || '创建订单失败')
  }
}

const createPromoPurchase = async () => {
  if (!ensureUserAction()) return
  try {
    const payload = {
      card_template_id: campaign.value.card_template_id,
      campaign_id: campaign.value.id,
      referral_code: route.params.refCode || '',
      use_promo: true,
      payment_method: campaign.value.payment_config.default_method || (campaign.value.payment_config.has_alipay ? 'alipay' : 'wechat')
    }
    if (campaign.value.my_claim?.id) {
      payload.claim_id = campaign.value.my_claim.id
    }
    const res = await shopApi.createDirectPurchase(payload)
    currentOrder.value = res.data.data
    paymentUrl.value = res.data.data.payment_url || ''
    paymentModalVisible.value = true
    await loadCampaign()
  } catch (err) {
    alert(err.response?.data?.error || '创建促销订单失败')
  }
}

const claimOrCreateStoreOrder = async () => {
  if (!ensureUserAction()) return
  try {
    let claimId = campaign.value.my_claim?.id
    if (!claimId) {
      const claimRes = await shopApi.claimPromotionCampaign(route.params.slug, route.params.refCode)
      claimId = claimRes.data.data.id
    }
    await shopApi.createDirectPurchase({
      card_template_id: campaign.value.card_template_id,
      campaign_id: campaign.value.id,
      claim_id: claimId,
      referral_code: route.params.refCode || '',
      use_promo: true,
      payment_method: 'store'
    })
    alert('已参与活动，请到店付款后由商户确认开卡')
    await loadCampaign()
  } catch (err) {
    alert(err.response?.data?.error || '领取活动失败')
  }
}

const confirmPayment = async () => {
  if (!currentOrder.value?.order_no) return
  try {
    await shopApi.confirmDirectPurchase(currentOrder.value.order_no, {})
    paymentModalVisible.value = false
    currentOrder.value = null
    paymentUrl.value = ''
    alert('已提交付款，等待商户确认开卡')
    await loadCampaign()
  } catch (err) {
    alert(err.response?.data?.error || '提交付款失败')
  }
}

const closePaymentModal = () => {
  paymentModalVisible.value = false
  currentOrder.value = null
  paymentUrl.value = ''
}

const handleShareAction = async () => {
  if (!ensureUserAction()) return
  try {
    const res = await shopApi.getPromotionRefLink(route.params.slug, route.params.refCode)
    shareLink.value = `${window.location.origin}${res.data.data.share_path}`
    await navigator.clipboard.writeText(shareLink.value)
    alert('专属推广链接已复制')
    await loadCampaign()
  } catch (err) {
    alert(err.response?.data?.error || '生成推广链接失败')
  }
}

const claimReward = async () => {
  try {
    await shopApi.claimPromotionReward(route.params.slug)
    alert('奖励卡领取成功')
    await loadCampaign()
  } catch (err) {
    alert(err.response?.data?.error || '领取奖励失败')
  }
}

const goLogin = () => {
  router.push('/login')
}

const goRegister = () => {
  router.push('/user/register')
}

onMounted(loadCampaign)
</script>
