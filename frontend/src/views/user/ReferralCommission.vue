<template>
  <div class="min-h-screen bg-[#f5f6f4] text-[#202124]">
    <div class="mx-auto max-w-5xl px-4 pb-8 pt-4">
      <div class="mb-4 flex items-center justify-between">
        <button
          class="inline-flex h-10 w-10 items-center justify-center rounded-full bg-white text-lg text-gray-700 shadow-sm ring-1 ring-gray-100 active:scale-95"
          aria-label="返回"
          @click="goBack"
        >
          ←
        </button>
        <div class="rounded-full bg-white px-3 py-1.5 text-xs font-medium text-gray-500 shadow-sm ring-1 ring-gray-100">
          推广码：{{ overview.profile?.promotion_code || '未生成' }}
        </div>
      </div>

      <section class="overflow-hidden rounded-[28px] bg-[linear-gradient(180deg,#ff8415_0px,#ff8115_40px,#ff851a_80px,#ff8720_100px,#ff8d2c_120px,#ff9030_140px,#ff953a_160px,#ffa04c_180px,#ffae62_198px,#ffc185_216px,#ffe0bf_236px,#fff1e5_258px,#fffcfa_292px,#ffffff_324px)] text-white shadow-sm">
        <div class="grid gap-0 md:grid-cols-[1.1fr_0.9fr]">
          <div class="p-6 md:p-8">
            <div class="inline-flex rounded-full bg-white/20 px-3 py-1 text-xs font-medium text-white">商户推广收益</div>
            <h1 class="mt-4 text-3xl font-black leading-tight md:text-4xl">推广分成</h1>
            <p class="mt-3 max-w-xl text-sm leading-6 text-white/72">邀请商户通过你的专属链接入驻，商户首次付费起两年内的付费可按 50% 计入分成。</p>

            <div class="mt-6 grid gap-3 sm:grid-cols-3">
              <div class="rounded-[22px] bg-[#ffd9bd]/50 p-4">
                <div class="text-xs font-medium text-white/92">可申请提现</div>
                <div class="mt-2 text-2xl font-black text-white">{{ formatMoney(overview.summary.claimable_amount) }}</div>
              </div>
              <div class="rounded-[22px] bg-[#ffd9bd]/50 p-4">
                <div class="text-xs font-medium text-white/92">累计分成</div>
                <div class="mt-2 text-2xl font-black text-white">{{ formatMoney(overview.summary.total_commission_amount) }}</div>
              </div>
              <div class="rounded-[22px] bg-[#ffd9bd]/50 p-4">
                <div class="text-xs font-medium text-white/92">已推广商户</div>
                <div class="mt-2 text-2xl font-black text-white">{{ overview.summary.merchant_count || 0 }}</div>
              </div>
            </div>
          </div>

          <div class="bg-[#fff7ed] p-5 text-gray-900 md:p-6">
            <div class="rounded-[24px] bg-white p-4 shadow-sm">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <div class="text-xs font-medium text-gray-500">我的推广链接</div>
                  <div class="mt-1 text-lg font-black text-[#ff7b23]">分享给商户入驻</div>
                </div>
                <div class="flex h-11 w-11 items-center justify-center rounded-2xl bg-[#fff0df] text-xl text-[#ff7b23]">↗</div>
              </div>
              <div class="mt-4 rounded-2xl bg-gray-50 p-3">
                <div class="break-all text-sm leading-5 text-gray-700">{{ resolvedShareLink || '生成后展示专属推广链接' }}</div>
              </div>
              <div class="mt-4 grid grid-cols-2 gap-3">
                <button class="rounded-2xl bg-[#ff7b23] px-4 py-3 text-sm font-bold text-white shadow-sm active:scale-[0.98]" @click="generateShareLink">
                  复制链接
                </button>
                <button class="rounded-2xl bg-[#34a853] px-4 py-3 text-sm font-bold text-white shadow-sm active:scale-[0.98]" @click="generatePoster">
                  生成图片
                </button>
              </div>
            </div>

            <div class="mt-4 grid grid-cols-2 gap-3 text-sm">
              <div class="rounded-2xl bg-white p-4 shadow-sm">
                <div class="text-xs text-gray-500">已申请/已打款</div>
                <div class="mt-2 font-black">{{ formatMoney((overview.summary.applied_amount || 0) + (overview.summary.paid_amount || 0)) }}</div>
              </div>
              <div class="rounded-2xl bg-white p-4 shadow-sm">
                <div class="text-xs text-gray-500">分成窗口</div>
                <div class="mt-2 font-black">首次付费后 2 年</div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="mt-5 grid gap-4 md:grid-cols-[0.82fr_1.18fr]">
        <div class="space-y-4">
          <div class="rounded-[24px] bg-white p-5 shadow-sm ring-1 ring-gray-100">
            <div class="flex items-center justify-between">
              <div class="text-base font-black">收益流程</div>
              <span class="rounded-full bg-[#fff4e8] px-3 py-1 text-xs font-medium text-[#ff7b23]">50%</span>
            </div>
            <div class="mt-5 space-y-4">
              <div class="flex gap-3">
                <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#ff7b23] text-sm font-bold text-white">1</div>
                <div>
                  <div class="font-bold">分享链接</div>
                  <div class="mt-1 text-sm text-gray-500">商户从专属链接注册后归属到你的推广关系。</div>
                </div>
              </div>
              <div class="flex gap-3">
                <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#ff7b23] text-sm font-bold text-white">2</div>
                <div>
                  <div class="font-bold">商户付费</div>
                  <div class="mt-1 text-sm text-gray-500">首次付费开始计算两年分成窗口。</div>
                </div>
              </div>
              <div class="flex gap-3">
                <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#ff7b23] text-sm font-bold text-white">3</div>
                <div>
                  <div class="font-bold">申请提现</div>
                  <div class="mt-1 text-sm text-gray-500">可提现流水可单笔申请，也可按商户合并申请。</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="rounded-[24px] bg-white p-5 shadow-sm ring-1 ring-gray-100">
          <div class="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <div class="text-lg font-black">已推广商户</div>
              <div class="text-sm text-gray-500">商户、付费、分成和提现状态集中展示。</div>
            </div>
            <div class="text-sm font-medium text-gray-500">{{ merchants.length }} 个商户</div>
          </div>

          <div v-if="loading" class="py-12 text-center text-gray-500">加载中...</div>
          <div v-else-if="merchants.length === 0" class="mt-5 rounded-2xl bg-gray-50 py-12 text-center text-gray-500">还没有已推广商户</div>

          <div v-for="merchant in merchants" :key="merchant.referral_id" class="mt-4 rounded-[22px] border border-gray-100 bg-[#fbfaf8] p-4">
            <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <div class="text-lg font-black">{{ merchant.merchant_name }}</div>
                  <span v-if="claimableLedgers(merchant).length > 0" class="rounded-full bg-[#fff4e8] px-2.5 py-1 text-xs font-medium text-[#ff7b23]">
                    {{ claimableLedgers(merchant).length }} 笔可提现
                  </span>
                </div>
                <div class="mt-2 grid gap-1 text-sm text-gray-500">
                  <div>注册时间：{{ formatDateTime(merchant.registered_at) || '未记录' }}</div>
                  <div>首次付费：{{ formatDateTime(merchant.first_paid_at) || '未发生' }}</div>
                  <div>分成截止：{{ formatDateTime(merchant.commission_expires_at) || '首次付费后生成' }}</div>
                </div>
              </div>

              <div class="grid grid-cols-2 gap-2 text-sm sm:grid-cols-4 lg:min-w-[420px]">
                <div class="rounded-2xl bg-white p-3">
                  <div class="text-xs text-gray-500">累计付费</div>
                  <div class="mt-1 font-black">{{ formatMoney(merchant.total_paid_amount) }}</div>
                </div>
                <div class="rounded-2xl bg-white p-3">
                  <div class="text-xs text-gray-500">累计分成</div>
                  <div class="mt-1 font-black">{{ formatMoney(merchant.total_commission_amount) }}</div>
                </div>
                <div class="rounded-2xl bg-white p-3">
                  <div class="text-xs text-gray-500">可申请</div>
                  <div class="mt-1 font-black text-[#ff7b23]">{{ formatMoney(merchant.claimable_amount) }}</div>
                </div>
                <div class="rounded-2xl bg-white p-3">
                  <div class="text-xs text-gray-500">已处理</div>
                  <div class="mt-1 font-black">{{ formatMoney((merchant.applied_amount || 0) + (merchant.paid_amount || 0)) }}</div>
                </div>
              </div>
            </div>

            <div class="mt-4 space-y-2">
              <div v-for="ledger in merchant.payment_ledgers || []" :key="ledger.id" class="rounded-2xl bg-white px-4 py-3">
                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div class="grid gap-x-6 gap-y-1 text-sm text-gray-600 sm:grid-cols-2">
                    <div>商户付费：<span class="font-bold text-gray-900">{{ formatMoney(ledger.paid_amount) }}</span></div>
                    <div>分成：<span class="font-bold text-[#ff7b23]">{{ formatMoney(ledger.commission_amount) }}</span> / {{ formatRate(ledger.commission_rate_bp) }}</div>
                    <div>付款时间：{{ formatDateTime(ledger.paid_at) }}</div>
                    <div>申请截止：{{ formatDateTime(ledger.claim_deadline) }}</div>
                  </div>
                  <div class="flex shrink-0 items-center gap-3">
                    <span :class="statusClass(ledger.withdrawal_status)" class="rounded-full px-3 py-1 text-xs font-medium">
                      {{ statusText(ledger.withdrawal_status) }}
                    </span>
                    <button
                      v-if="ledger.withdrawal_status === 'claimable'"
                      class="rounded-xl bg-[#ff7b23] px-3 py-2 text-xs font-bold text-white active:scale-95"
                      @click="openWithdrawModal([ledger])"
                    >
                      申请提现
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <button
              v-if="claimableLedgers(merchant).length > 1"
              class="mt-4 w-full rounded-2xl border border-[#ffb77b] bg-[#fff4e8] px-4 py-3 text-sm font-bold text-[#ff7b23] active:scale-[0.99]"
              @click="openWithdrawModal(claimableLedgers(merchant))"
            >
              合并申请该商户全部可提现分成
            </button>
          </div>
        </div>
      </section>
    </div>

    <div v-if="posterVisible" class="fixed inset-0 z-40 bg-black/60 p-4 overflow-y-auto" @click.self="posterVisible = false">
      <div class="mx-auto max-w-md rounded-[30px] bg-white p-5">
        <div class="flex items-center justify-between">
          <div class="text-lg font-bold">海报预览</div>
          <button class="text-gray-400" @click="posterVisible = false">关闭</button>
        </div>
        <img v-if="posterDataUrl" :src="posterDataUrl" alt="推广海报" class="mt-4 w-full rounded-3xl border border-[#f2e5d2]" />
        <div class="mt-4 flex gap-3">
          <button class="flex-1 rounded-2xl border border-gray-200 px-4 py-3 text-sm" @click="posterVisible = false">关闭</button>
          <button class="flex-1 rounded-2xl bg-[#ff7b23] px-4 py-3 text-sm text-white" @click="downloadPoster">下载图片</button>
        </div>
      </div>
    </div>

    <div v-if="withdrawVisible" class="fixed inset-0 z-40 bg-black/50 p-4 overflow-y-auto" @click.self="withdrawVisible = false">
      <div class="mx-auto max-w-md rounded-[30px] bg-white p-5">
        <div class="text-lg font-bold">申请分成提现</div>
        <div class="mt-2 text-sm text-gray-500">本次申请 {{ withdrawLedgers.length }} 笔，合计 {{ formatMoney(withdrawAmount) }}</div>

        <div class="mt-4 space-y-3">
          <input v-model="withdrawForm.payee_name" type="text" placeholder="收款人姓名" class="w-full rounded-2xl border border-gray-200 px-4 py-3 outline-none" />
          <input v-model="withdrawForm.payee_account" type="text" placeholder="微信号 / 支付宝账号 / 银行账号" class="w-full rounded-2xl border border-gray-200 px-4 py-3 outline-none" />
          <select v-model="withdrawForm.payee_channel" class="w-full rounded-2xl border border-gray-200 px-4 py-3 outline-none">
            <option value="wechat">微信</option>
            <option value="alipay">支付宝</option>
            <option value="bank">银行卡</option>
          </select>
        </div>

        <div class="mt-4 flex gap-3">
          <button class="flex-1 rounded-2xl border border-gray-200 px-4 py-3 text-sm" @click="withdrawVisible = false">取消</button>
          <button class="flex-1 rounded-2xl bg-[#ff7b23] px-4 py-3 text-sm text-white" @click="submitWithdrawal">提交申请</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import QRCode from 'qrcode'
import { referralCommissionApi } from '../../api'
import heroStoreIllustration from '../../assets/merchant-referral/hero-store-illustration-wide.png'
import saleLeftIcon from '../../assets/merchant-referral/icon-feature-sale-left.png'
import saleRightIcon from '../../assets/merchant-referral/icon-feature-sale-right.png'
import verifyLeftIcon from '../../assets/merchant-referral/icon-feature-verify-left.png'
import verifyRightIcon from '../../assets/merchant-referral/icon-feature-verify-right.png'
import bookingLeftIcon from '../../assets/merchant-referral/icon-feature-booking-left.png'
import bookingRightIcon from '../../assets/merchant-referral/icon-feature-booking-right.png'

const router = useRouter()
const loading = ref(true)
const overview = ref({ profile: null, summary: {} })
const merchants = ref([])
const posterVisible = ref(false)
const posterDataUrl = ref('')
const posterBlob = ref(null)
const posterPayload = ref(null)
const withdrawVisible = ref(false)
const withdrawLedgers = ref([])
const withdrawForm = ref({
  payee_name: '',
  payee_account: '',
  payee_channel: 'wechat'
})
const posterFeatureAssets = [
  { leftIcon: saleLeftIcon, rightIcon: saleRightIcon },
  { leftIcon: verifyLeftIcon, rightIcon: verifyRightIcon },
  { leftIcon: bookingLeftIcon, rightIcon: bookingRightIcon }
]

const withdrawAmount = computed(() => withdrawLedgers.value.reduce((sum, item) => sum + (item.commission_amount || 0), 0))
const buildReferralShareLink = (sharePath, fallbackURL = '') => {
  const path = String(sharePath || '').trim()
  const fallback = String(fallbackURL || '').trim()
  if (!path) return fallback
  if (typeof window === 'undefined') return fallback || path
  return `${window.location.origin}${path}`
}
const isPrivateOrLocalHost = (hostname) => {
  const host = String(hostname || '').trim().toLowerCase()
  if (!host) return false
  if (host === 'localhost' || host === '127.0.0.1' || host === '::1') return true
  if (/^10\.\d+\.\d+\.\d+$/.test(host)) return true
  if (/^192\.168\.\d+\.\d+$/.test(host)) return true
  if (/^172\.(1[6-9]|2\d|3[0-1])\.\d+\.\d+$/.test(host)) return true
  return false
}
const resolvePosterRegisterLink = (payload) => {
  const rawURL = String(payload?.register_url || '').trim()
  const refCode = String(payload?.promotion_code || '').trim()
  if (!rawURL) return ''
  if (typeof window === 'undefined') return rawURL
  const { protocol, hostname, origin } = window.location
  if ((protocol !== 'https:' && protocol !== 'http:') || !isPrivateOrLocalHost(hostname) || !refCode) {
    return rawURL
  }
  return `${origin}/merchant/${refCode}/login`
}
const resolvedShareLink = computed(() => buildReferralShareLink(overview.value.profile?.share_path, overview.value.profile?.share_link))

const formatMoney = (amount) => `¥${((Number(amount) || 0) / 100).toFixed(2)}`
const formatRate = (bp) => `${((Number(bp) || 0) / 100).toFixed(0)}%`
const formatDateTime = (value) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('zh-CN')
}
const statusText = (status) => ({
  claimable: '可申请',
  applied: '已申请',
  approved: '审核通过',
  paid: '已打款',
  expired: '已过期',
  rejected: '已驳回'
}[status] || status)
const statusClass = (status) => ({
  claimable: 'bg-[#fff4e8] text-[#ff7b23]',
  applied: 'bg-[#eef5ff] text-[#2b6fff]',
  approved: 'bg-[#eefbf1] text-[#1f8f48]',
  paid: 'bg-[#edf7f1] text-[#18794e]',
  expired: 'bg-gray-100 text-gray-500',
  rejected: 'bg-red-50 text-red-500'
}[status] || 'bg-gray-100 text-gray-500')

const claimableLedgers = (merchant) => (merchant.payment_ledgers || []).filter((item) => item.withdrawal_status === 'claimable')

const goBack = () => {
  router.push('/user/cards')
}

const loadData = async () => {
  loading.value = true
  try {
    const [overviewRes, merchantRes] = await Promise.all([
      referralCommissionApi.getOverview(),
      referralCommissionApi.getMerchants()
    ])
    overview.value = overviewRes.data?.data || { profile: null, summary: {} }
    merchants.value = merchantRes.data?.data || []
  } catch (err) {
    alert(err.response?.data?.error || '加载推广分成失败')
  } finally {
    loading.value = false
  }
}

const generateShareLink = async () => {
  try {
    const res = await referralCommissionApi.createShareLink()
    const payload = res.data?.data || {}
    const shareLink = buildReferralShareLink(payload.share_path, payload.share_link)
    if (!shareLink) throw new Error('share link missing')
    await navigator.clipboard.writeText(shareLink)
    overview.value.profile = {
      ...(overview.value.profile || {}),
      ...payload
    }
    alert('推广链接已复制')
  } catch (err) {
    alert(err.response?.data?.error || '生成推广链接失败')
  }
}

const canvasToBlob = (canvas) => new Promise((resolve, reject) => {
  canvas.toBlob((blob) => {
    if (blob) resolve(blob)
    else reject(new Error('生成图片失败'))
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
      if (!navigator.canShare || navigator.canShare({ files: [file] })) {
        await navigator.share({ files: [file], title: '推广海报' })
        alert('图片已生成，请在系统面板中保存到相册')
        return true
      }
    } catch (_) {
      // 用户取消或系统不支持文件分享时，回退为下载
    }
  }
  downloadBlobImage(blob, filename)
  alert('图片已生成并开始下载')
  return false
}

const generatePoster = async () => {
  try {
    const res = await referralCommissionApi.getPoster()
    posterPayload.value = res.data?.data || null
    const registerLink = resolvePosterRegisterLink(posterPayload.value)
    if (!registerLink) throw new Error('poster payload missing')
    const width = 430
    const scale = 2
    const shellPadding = 16
    const heroHeight = 246
    const featureCardHeight = 144
    const featureGap = 16
    const featureTop = heroHeight - 40
    const registerTop = featureTop + featureCardHeight * 3 + featureGap * 2 + 16
    const registerHeight = 280
    const height = registerTop + registerHeight + 18
    const canvas = document.createElement('canvas')
    const heroSubtitle = String(posterPayload.value?.subtitle || '').trim()
    canvas.width = width * scale
    canvas.height = height * scale
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('canvas unsupported')
    ctx.scale(scale, scale)

    const qrDataUrl = await QRCode.toDataURL(registerLink, {
      width: 220,
      margin: 1,
      color: { dark: '#111111', light: '#FFFFFF' }
    })
    const [heroImage, qrImage, ...featureImages] = await Promise.all([
      loadImage(heroStoreIllustration),
      loadImage(qrDataUrl),
      ...posterFeatureAssets.flatMap((item) => [loadImage(item.leftIcon), loadImage(item.rightIcon)])
    ])
    const features = (posterPayload.value?.highlights || []).slice(0, 3).map((item, index) => ({
      ...item,
      leftImage: featureImages[index * 2],
      rightImage: featureImages[index * 2 + 1]
    }))

    const pageGradient = ctx.createLinearGradient(0, 0, 0, 324)
    pageGradient.addColorStop(0, '#ff8415')
    pageGradient.addColorStop(0.25, '#ff851a')
    pageGradient.addColorStop(0.43, '#ff953a')
    pageGradient.addColorStop(0.61, '#ffc185')
    pageGradient.addColorStop(0.8, '#fff1e5')
    pageGradient.addColorStop(1, '#ffffff')
    ctx.fillStyle = pageGradient
    ctx.fillRect(0, 0, width, height)
    const glow = ctx.createRadialGradient(width * 0.86, 34, 20, width * 0.86, 34, 190)
    glow.addColorStop(0, 'rgba(255, 210, 164, 0.3)')
    glow.addColorStop(0.5, 'rgba(255, 178, 108, 0.14)')
    glow.addColorStop(1, 'rgba(255, 178, 108, 0)')
    ctx.fillStyle = glow
    ctx.fillRect(0, 0, width, 210)

    ctx.save()
    ctx.beginPath()
    ctx.rect(0, 0, width, heroHeight)
    ctx.clip()
    drawImageContain(ctx, heroImage, 210, 52, 218, 136)
    ctx.restore()

    ctx.fillStyle = 'rgba(255,255,255,0.92)'
    ctx.font = '12px sans-serif'
    ctx.letterSpacing = '4px'
    ctx.fillText('KABAO FOR MERCHANT', 24, 44)
    ctx.letterSpacing = '0px'
    ctx.fillStyle = '#ffffff'
    ctx.font = '900 35px sans-serif'
    ctx.fillText('卡包商户入驻', 24, 98)
    ctx.font = '700 16px sans-serif'
    wrapText(ctx, heroSubtitle, 24, 154, 250, 30)

    features.forEach((item, index) => {
      const currentTop = featureTop + index * (featureCardHeight + featureGap)
      const cardX = shellPadding
      const cardWidth = width - shellPadding * 2
      ctx.save()
      ctx.shadowColor = 'rgba(245, 147, 39, 0.07)'
      ctx.shadowBlur = 24
      ctx.shadowOffsetY = 12
      ctx.fillStyle = '#ffffff'
      roundRect(ctx, cardX, currentTop, cardWidth, featureCardHeight, 28)
      ctx.fill()
      ctx.restore()
      ctx.strokeStyle = 'rgba(246, 233, 216, 0.9)'
      ctx.lineWidth = 1
      roundRect(ctx, cardX, currentTop, cardWidth, featureCardHeight, 28)
      ctx.stroke()

      drawImageContain(ctx, item.leftImage, cardX + 22, currentTop + 29, 86, 86)
      drawImageContain(ctx, item.rightImage, cardX + cardWidth - 80, currentTop + 41, 62, 62)
      ctx.fillStyle = '#ff6f19'
      ctx.font = '800 19px sans-serif'
      wrapText(ctx, item.title || '', cardX + 122, currentTop + 40, 188, 24)
      ctx.fillStyle = '#4b5563'
      ctx.font = '14px sans-serif'
      wrapText(ctx, item.description || '', cardX + 122, currentTop + 78, 176, 26)
    })

    const panelX = shellPadding
    const panelWidth = width - shellPadding * 2
    ctx.fillStyle = '#ffffff'
    roundRect(ctx, panelX, registerTop, panelWidth, registerHeight, 28)
    ctx.fill()
    ctx.strokeStyle = 'rgba(246, 233, 216, 0.9)'
    roundRect(ctx, panelX, registerTop, panelWidth, registerHeight, 28)
    ctx.stroke()

    const panelInnerX = panelX + 18
    const titleY = registerTop + 36
    const accentGradient = ctx.createLinearGradient(panelInnerX, registerTop + 20, panelInnerX, registerTop + 38)
    accentGradient.addColorStop(0, '#ff6c12')
    accentGradient.addColorStop(1, '#ffb05e')
    ctx.fillStyle = accentGradient
    roundRect(ctx, panelInnerX, registerTop + 23, 4, 18, 2)
    ctx.fill()
    ctx.fillStyle = '#111827'
    ctx.font = '900 20px sans-serif'
    ctx.fillText('开始使用卡包', panelInnerX + 14, titleY)
    ctx.fillStyle = '#4b5563'
    ctx.font = '14px sans-serif'
    wrapText(ctx, '完成注册后即可开始使用售卡、核销、预约等商户功能', panelInnerX, registerTop + 64, panelWidth - 36, 25)

    const qrSize = 132
    const qrTop = registerTop + 94
    ctx.drawImage(qrImage, (width - qrSize) / 2, qrTop, qrSize, qrSize)
    ctx.fillStyle = '#7a7a7a'
    ctx.font = '12px sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText('微信等识别二维码可直接打开注册页', width / 2, qrTop + 160)
    ctx.textAlign = 'left'

    posterDataUrl.value = canvas.toDataURL('image/png')
    posterBlob.value = await canvasToBlob(canvas)
    posterVisible.value = true
  } catch (err) {
    alert(err.response?.data?.error || err.message || '生成推广图片失败')
  }
}

const loadImage = (src) => new Promise((resolve, reject) => {
  const img = new Image()
  img.onload = () => resolve(img)
  img.onerror = reject
  img.src = src
})

const drawImageContain = (ctx, image, x, y, width, height) => {
  if (!image) return
  const ratio = Math.min(width / image.naturalWidth, height / image.naturalHeight)
  const drawWidth = image.naturalWidth * ratio
  const drawHeight = image.naturalHeight * ratio
  ctx.drawImage(image, x + (width - drawWidth) / 2, y + (height - drawHeight) / 2, drawWidth, drawHeight)
}

const drawCopyIcon = (ctx, x, y) => {
  ctx.save()
  ctx.strokeStyle = '#6b7280'
  ctx.lineWidth = 1.7
  roundRect(ctx, x + 5, y + 3, 11, 13, 2.5)
  ctx.stroke()
  ctx.beginPath()
  ctx.moveTo(x, y + 11.5)
  ctx.lineTo(x, y + 1.8)
  ctx.quadraticCurveTo(x, y, x + 1.8, y)
  ctx.lineTo(x + 10.5, y)
  ctx.stroke()
  ctx.restore()
}

const roundRect = (ctx, x, y, width, height, radius) => {
  const safeRadius = Math.min(Math.max(Number(radius) || 0, 0), Math.abs(width) / 2, Math.abs(height) / 2)
  ctx.beginPath()
  ctx.moveTo(x + safeRadius, y)
  ctx.arcTo(x + width, y, x + width, y + height, safeRadius)
  ctx.arcTo(x + width, y + height, x, y + height, safeRadius)
  ctx.arcTo(x, y + height, x, y, safeRadius)
  ctx.arcTo(x, y, x + width, y, safeRadius)
  ctx.closePath()
}

const wrapText = (ctx, text, x, y, maxWidth, lineHeight) => {
  const chars = String(text || '').split('')
  let line = ''
  let currentY = y
  for (const ch of chars) {
    const testLine = line + ch
    if (ctx.measureText(testLine).width > maxWidth && line) {
      ctx.fillText(line, x, currentY)
      line = ch
      currentY += lineHeight
    } else {
      line = testLine
    }
  }
  if (line) ctx.fillText(line, x, currentY)
  return currentY
}

const countWrappedLines = (ctx, text, maxWidth) => {
  const chars = String(text || '').split('')
  let line = ''
  let lines = 0
  for (const ch of chars) {
    const testLine = line + ch
    if (ctx.measureText(testLine).width > maxWidth && line) {
      lines += 1
      line = ch
    } else {
      line = testLine
    }
  }
  return line ? lines + 1 : Math.max(lines, 1)
}

const estimateWrappedTextBottom = (text, maxWidth, lineHeight, font, startY = 0) => {
  const measureCanvas = document.createElement('canvas')
  const measureCtx = measureCanvas.getContext('2d')
  if (!measureCtx) return startY
  measureCtx.font = font
  const lines = countWrappedLines(measureCtx, text, maxWidth)
  return startY + Math.max(lines - 1, 0) * lineHeight
}

const downloadPoster = async () => {
  if (!posterDataUrl.value) return
  const filename = `merchant_referral_poster_${overview.value.profile?.promotion_code || Date.now()}.png`
  if (posterBlob.value) {
    await savePosterBlob(posterBlob.value, filename)
    return
  }

  const resp = await fetch(posterDataUrl.value)
  const blob = await resp.blob()
  posterBlob.value = blob
  await savePosterBlob(blob, filename)
}

const openWithdrawModal = (ledgers) => {
  withdrawLedgers.value = ledgers
  withdrawVisible.value = true
}

const submitWithdrawal = async () => {
  try {
    await referralCommissionApi.createWithdrawal({
      payment_ledger_ids: withdrawLedgers.value.map((item) => item.id),
      ...withdrawForm.value
    })
    alert('提现申请已提交')
    withdrawVisible.value = false
    withdrawLedgers.value = []
    withdrawForm.value = {
      payee_name: '',
      payee_account: '',
      payee_channel: 'wechat'
    }
    await loadData()
  } catch (err) {
    alert(err.response?.data?.error || '提交提现申请失败')
  }
}

onMounted(loadData)
</script>
