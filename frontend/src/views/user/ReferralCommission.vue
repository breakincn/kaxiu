<template>
  <div class="min-h-screen bg-[#f7f2ea] text-gray-800">
    <div class="max-w-5xl mx-auto px-4 py-6 space-y-4">
      <div class="rounded-[30px] overflow-hidden bg-[radial-gradient(circle_at_top_left,_#ffd6ad,_transparent_32%),linear-gradient(135deg,#fff6ea,#f3ece2)] border border-[#ead9c3] p-5 shadow-sm">
        <div class="flex items-start justify-between gap-4">
          <div>
            <div class="text-xs uppercase tracking-[0.35em] text-[#c67a2a]">Referral Commission</div>
            <h1 class="mt-2 text-2xl font-black">推广分成</h1>
            <p class="mt-2 text-sm leading-6 text-gray-600">邀请商户通过你的专属链接入驻，商户首次付费起两年内的付费可按 50% 计入分成。</p>
          </div>
          <button
            class="inline-flex shrink-0 items-center gap-1.5 self-start whitespace-nowrap rounded-full border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-700"
            @click="goBack"
          >
            <span aria-hidden="true">←</span>
            返回
          </button>
        </div>

        <div v-if="overview.profile" class="mt-5 grid gap-3 md:grid-cols-[1fr_auto_auto] items-center rounded-3xl bg-white/80 p-4">
          <div class="min-w-0">
            <div class="text-xs text-gray-500">我的推广链接</div>
            <div class="mt-1 break-all text-sm text-[#ff7b23]">{{ resolvedShareLink }}</div>
            <div class="mt-2 text-xs text-gray-500">推广码：{{ overview.profile.promotion_code }}</div>
          </div>
          <button class="rounded-2xl bg-[#ff7b23] px-4 py-3 text-sm font-medium text-white" @click="generateShareLink">
            生成推广链接
          </button>
          <button class="rounded-2xl border border-[#ffb77b] bg-[#fff4e8] px-4 py-3 text-sm font-medium text-[#ff7b23]" @click="generatePoster">
            生成推广图片
          </button>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-4">
        <div class="rounded-3xl bg-white p-5 shadow-sm border border-[#ead9c3]">
          <div class="text-xs text-gray-500">已推广商户</div>
          <div class="mt-2 text-3xl font-black">{{ overview.summary.merchant_count || 0 }}</div>
        </div>
        <div class="rounded-3xl bg-white p-5 shadow-sm border border-[#ead9c3]">
          <div class="text-xs text-gray-500">累计分成</div>
          <div class="mt-2 text-3xl font-black">{{ formatMoney(overview.summary.total_commission_amount) }}</div>
        </div>
        <div class="rounded-3xl bg-white p-5 shadow-sm border border-[#ead9c3]">
          <div class="text-xs text-gray-500">可申请提现</div>
          <div class="mt-2 text-3xl font-black text-[#ff7b23]">{{ formatMoney(overview.summary.claimable_amount) }}</div>
        </div>
        <div class="rounded-3xl bg-white p-5 shadow-sm border border-[#ead9c3]">
          <div class="text-xs text-gray-500">已申请/已打款</div>
          <div class="mt-2 text-3xl font-black">{{ formatMoney((overview.summary.applied_amount || 0) + (overview.summary.paid_amount || 0)) }}</div>
        </div>
      </div>

      <div class="rounded-3xl bg-white p-5 shadow-sm border border-[#ead9c3] space-y-4">
        <div class="flex items-center justify-between gap-3">
          <div>
            <div class="text-lg font-bold">已推广列表</div>
            <div class="text-sm text-gray-500">展示已通过你的推广链接注册的商户，以及分成窗口内的付费台账。</div>
          </div>
        </div>

        <div v-if="loading" class="py-10 text-center text-gray-500">加载中...</div>
        <div v-else-if="merchants.length === 0" class="py-10 text-center text-gray-500">还没有已推广商户</div>

        <div v-for="merchant in merchants" :key="merchant.referral_id" class="rounded-[26px] border border-[#f0e0ca] bg-[#fffdfa] p-4 space-y-3">
          <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
            <div>
              <div class="text-lg font-bold">{{ merchant.merchant_name }}</div>
              <div class="mt-1 text-sm text-gray-500">注册时间：{{ formatDateTime(merchant.registered_at) || '未记录' }}</div>
              <div class="mt-1 text-sm text-gray-500">首次付费：{{ formatDateTime(merchant.first_paid_at) || '未发生' }}</div>
              <div class="mt-1 text-sm text-gray-500">分成截止：{{ formatDateTime(merchant.commission_expires_at) || '首次付费后生成' }}</div>
            </div>
            <div class="grid grid-cols-2 gap-3 text-sm md:min-w-[280px]">
              <div class="rounded-2xl bg-white px-3 py-2 border border-[#f2e5d2]">
                <div class="text-xs text-gray-500">累计付费</div>
                <div class="mt-1 font-bold">{{ formatMoney(merchant.total_paid_amount) }}</div>
              </div>
              <div class="rounded-2xl bg-white px-3 py-2 border border-[#f2e5d2]">
                <div class="text-xs text-gray-500">累计分成</div>
                <div class="mt-1 font-bold">{{ formatMoney(merchant.total_commission_amount) }}</div>
              </div>
              <div class="rounded-2xl bg-white px-3 py-2 border border-[#f2e5d2]">
                <div class="text-xs text-gray-500">可申请</div>
                <div class="mt-1 font-bold text-[#ff7b23]">{{ formatMoney(merchant.claimable_amount) }}</div>
              </div>
              <div class="rounded-2xl bg-white px-3 py-2 border border-[#f2e5d2]">
                <div class="text-xs text-gray-500">已申请/已打款</div>
                <div class="mt-1 font-bold">{{ formatMoney((merchant.applied_amount || 0) + (merchant.paid_amount || 0)) }}</div>
              </div>
            </div>
          </div>

          <div class="space-y-2">
            <div v-for="ledger in merchant.payment_ledgers || []" :key="ledger.id" class="rounded-2xl bg-white px-4 py-3 border border-[#f2e5d2]">
              <div class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                <div class="text-sm text-gray-600">
                  <div>商户付费：{{ formatMoney(ledger.paid_amount) }}</div>
                  <div>分成：{{ formatMoney(ledger.commission_amount) }} / {{ formatRate(ledger.commission_rate_bp) }}</div>
                  <div>付款时间：{{ formatDateTime(ledger.paid_at) }}</div>
                  <div>申请截止：{{ formatDateTime(ledger.claim_deadline) }}</div>
                </div>
                <div class="flex items-center gap-3">
                  <span :class="statusClass(ledger.withdrawal_status)" class="rounded-full px-3 py-1 text-xs font-medium">
                    {{ statusText(ledger.withdrawal_status) }}
                  </span>
                  <button
                    v-if="ledger.withdrawal_status === 'claimable'"
                    class="rounded-xl bg-[#ff7b23] px-3 py-2 text-xs font-medium text-white"
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
            class="rounded-xl border border-[#ffb77b] bg-[#fff4e8] px-4 py-2 text-sm font-medium text-[#ff7b23]"
            @click="openWithdrawModal(claimableLedgers(merchant))"
          >
            合并申请该商户全部可提现分成
          </button>
        </div>
      </div>
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

const router = useRouter()
const loading = ref(true)
const overview = ref({ profile: null, summary: {} })
const merchants = ref([])
const posterVisible = ref(false)
const posterDataUrl = ref('')
const posterPayload = ref(null)
const withdrawVisible = ref(false)
const withdrawLedgers = ref([])
const withdrawForm = ref({
  payee_name: '',
  payee_account: '',
  payee_channel: 'wechat'
})

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

const generatePoster = async () => {
  try {
    const res = await referralCommissionApi.getPoster()
    posterPayload.value = res.data?.data || null
    const registerLink = resolvePosterRegisterLink(posterPayload.value)
    if (!registerLink) throw new Error('poster payload missing')
    const width = 430
    const scale = 2
    const pagePadding = 18
    const cardWidth = width - pagePadding * 2
    const highlightGap = 14
    const highlightHeight = 86
    const canvas = document.createElement('canvas')
    const measureCanvas = document.createElement('canvas')
    const measureCtx = measureCanvas.getContext('2d')
    if (!measureCtx) throw new Error('canvas unsupported')
    const heroSubtitle = String(posterPayload.value?.subtitle || '').trim()
    measureCtx.font = '14px sans-serif'
    const heroSubtitleLines = countWrappedLines(measureCtx, heroSubtitle, width - 76)
    measureCtx.font = '13px sans-serif'
    const registerUrlLines = countWrappedLines(measureCtx, registerLink, cardWidth - 52)
    const registerInfoHeight = Math.max(90, 44 + registerUrlLines * 20)
    const heroHeight = Math.max(148, 120 + Math.max(heroSubtitleLines - 1, 0) * 24)
    const highlightsHeight = ((posterPayload.value?.highlights || []).length * highlightHeight) + (Math.max((posterPayload.value?.highlights || []).length - 1, 0) * highlightGap)
    const startCardTop = pagePadding + heroHeight + pagePadding + highlightsHeight + pagePadding
    const startDescriptionY = startCardTop + 62
    const startDescriptionBottom = estimateWrappedTextBottom('完成注册后即可开始使用售卡、核销、预约等商户功能', width - 88, 20, '13px sans-serif', startDescriptionY)
    const registerInfoTop = startDescriptionBottom + 18
    const registerTextY = registerInfoTop + 52
    const registerTextBottom = estimateWrappedTextBottom(registerLink, width - 104, 20, '13px sans-serif', registerTextY)
    const buttonTop = Math.max(registerInfoTop + registerInfoHeight + 18, registerTextBottom + 24)
    const qrTop = buttonTop + 70
    const startCardHeight = qrTop + 174 - startCardTop + 28
    const height = startCardTop + startCardHeight + pagePadding
    canvas.width = width * scale
    canvas.height = height * scale
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('canvas unsupported')
    ctx.scale(scale, scale)
    ctx.fillStyle = '#f7f2ea'
    ctx.fillRect(0, 0, width, height)

    ctx.fillStyle = '#ffffff'
    roundRect(ctx, pagePadding, pagePadding, cardWidth, heroHeight, 28)
    ctx.fill()
    const gradient = ctx.createLinearGradient(pagePadding, pagePadding, width - pagePadding, pagePadding + heroHeight)
    gradient.addColorStop(0, '#ff8a34')
    gradient.addColorStop(1, '#ffd16d')
    ctx.fillStyle = gradient
    roundRect(ctx, pagePadding, pagePadding, cardWidth, heroHeight, 28)
    ctx.fill()

    ctx.fillStyle = 'rgba(255,255,255,0.85)'
    ctx.font = '12px sans-serif'
    ctx.fillText('KABAO FOR MERCHANT', 38, 44)
    ctx.fillStyle = '#ffffff'
    ctx.font = 'bold 28px sans-serif'
    wrapText(ctx, '卡包商户入驻', 38, 84, width - 76, 38)
    ctx.font = '14px sans-serif'
    wrapText(ctx, heroSubtitle, 38, 124, width - 76, 24)

    let cardTop = pagePadding + heroHeight + pagePadding
    ;(posterPayload.value.highlights || []).forEach((item, index) => {
      ctx.fillStyle = '#ffffff'
      const currentTop = cardTop + index * (highlightHeight + highlightGap)
      roundRect(ctx, 24, currentTop, width - 48, highlightHeight, 22)
      ctx.fill()
      ctx.fillStyle = '#ff7b23'
      ctx.font = 'bold 16px sans-serif'
      ctx.fillText(item.title || '', 44, currentTop + 34)
      ctx.fillStyle = '#5f5f5f'
      ctx.font = '13px sans-serif'
      wrapText(ctx, item.description || '', 44, currentTop + 60, width - 88, 20)
    })

    ctx.fillStyle = '#ffffff'
    roundRect(ctx, 24, startCardTop, width - 48, startCardHeight, 24)
    ctx.fill()
    ctx.fillStyle = '#111827'
    ctx.font = 'bold 18px sans-serif'
    ctx.fillText('开始使用卡包', 44, startCardTop + 34)
    ctx.fillStyle = '#6b7280'
    ctx.font = '13px sans-serif'
    wrapText(ctx, '完成注册后即可开始使用售卡、核销、预约等商户功能', 44, startDescriptionY, width - 88, 20)

    ctx.fillStyle = '#f8fafc'
    roundRect(ctx, 36, registerInfoTop, width - 72, registerInfoHeight, 18)
    ctx.fill()
    ctx.fillStyle = '#6b7280'
    ctx.font = '12px sans-serif'
    ctx.fillText('商户注册入口', 52, registerInfoTop + 24)
    ctx.fillStyle = '#374151'
    ctx.font = '13px sans-serif'
    wrapText(ctx, registerLink, 52, registerTextY, width - 104, 20)

    ctx.fillStyle = '#ff7b23'
    roundRect(ctx, 36, buttonTop, width - 72, 48, 18)
    ctx.fill()
    ctx.fillStyle = '#ffffff'
    ctx.font = 'bold 16px sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText('去注册商户', width / 2, buttonTop + 30)

    const qrDataUrl = await QRCode.toDataURL(registerLink, {
      width: 220,
      margin: 1,
      color: { dark: '#111111', light: '#FFFFFF' }
    })
    const qrImage = await loadImage(qrDataUrl)
    ctx.drawImage(qrImage, (width - 150) / 2, qrTop, 150, 150)
    ctx.fillStyle = '#7a7a7a'
    ctx.font = '13px sans-serif'
    ctx.fillText('微信等识别二维码可直接打开注册页', width / 2, qrTop + 174)
    ctx.textAlign = 'left'

    posterDataUrl.value = canvas.toDataURL('image/png')
    await canvasToBlob(canvas)
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

const roundRect = (ctx, x, y, width, height, radius) => {
  ctx.beginPath()
  ctx.moveTo(x + radius, y)
  ctx.arcTo(x + width, y, x + width, y + height, radius)
  ctx.arcTo(x + width, y + height, x, y + height, radius)
  ctx.arcTo(x, y + height, x, y, radius)
  ctx.arcTo(x, y, x + width, y, radius)
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

const downloadPoster = () => {
  if (!posterDataUrl.value) return
  const link = document.createElement('a')
  link.href = posterDataUrl.value
  link.download = `merchant_referral_poster_${overview.value.profile?.promotion_code || Date.now()}.png`
  link.click()
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
