<template>
  <div class="merchant-referral-page">
    <div v-if="loading" class="merchant-referral-state">加载中...</div>
    <div v-else-if="errorMessage" class="merchant-referral-state merchant-referral-state-error">{{ errorMessage }}</div>
    <div v-else-if="!landing" class="merchant-referral-state merchant-referral-state-error">推广页不存在</div>
    <div v-else class="merchant-referral-shell">
      <section class="hero-banner">
        <div class="hero-copy">
          <div class="hero-kicker">KABAO FOR MERCHANT</div>
          <h1 class="hero-title">卡包商户入驻</h1>
          <p class="hero-subtitle">{{ landing.subtitle }}</p>
        </div>
        <div class="hero-illustration">
          <img :src="heroStoreIllustration" alt="" />
        </div>
      </section>

      <section class="feature-list">
        <article
          v-for="item in featureItems"
          :key="item.title"
          class="feature-card"
        >
          <img class="feature-left-icon" :src="item.leftIcon" :alt="`${item.title}图标`" />
          <div class="feature-copy">
            <h2 class="feature-title">{{ item.title }}</h2>
            <p class="feature-description">{{ item.description }}</p>
          </div>
          <img class="feature-right-icon" :src="item.rightIcon" :alt="`${item.title}趋势图标`" />
        </article>
      </section>

      <section class="register-panel">
        <div class="register-heading-row">
          <span class="register-accent"></span>
          <div>
            <h2 class="register-title">开始使用卡包</h2>
            <p class="register-description">完成注册后即可开始使用售卡、核销、预约等商户功能</p>
          </div>
        </div>

        <div class="register-link-label">商户注册入口</div>
        <div class="register-link-box">
          <div class="register-link-text">{{ registerUrl }}</div>
          <button type="button" class="register-copy-button" @click="copyRegisterUrl">
            <svg viewBox="0 0 24 24" fill="none" class="register-copy-icon">
              <rect x="8.5" y="7.5" width="11" height="13" rx="2.5" stroke="currentColor" stroke-width="1.7" />
              <path d="M5 15.5V5.8C5 4.81 5.81 4 6.8 4h8.7" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </button>
        </div>

        <button type="button" class="register-submit" @click="goRegister">
          去注册商户
        </button>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { authApi } from '../../api'
import heroStoreIllustration from '../../assets/merchant-referral/hero-store-illustration-wide.png'
import saleLeftIcon from '../../assets/merchant-referral/icon-feature-sale-left.png'
import saleRightIcon from '../../assets/merchant-referral/icon-feature-sale-right.png'
import verifyLeftIcon from '../../assets/merchant-referral/icon-feature-verify-left.png'
import verifyRightIcon from '../../assets/merchant-referral/icon-feature-verify-right.png'
import bookingLeftIcon from '../../assets/merchant-referral/icon-feature-booking-left.png'
import bookingRightIcon from '../../assets/merchant-referral/icon-feature-booking-right.png'

const route = useRoute()
const loading = ref(true)
const landing = ref(null)
const errorMessage = ref('')

const featureAssets = [
  { leftIcon: saleLeftIcon, rightIcon: saleRightIcon },
  { leftIcon: verifyLeftIcon, rightIcon: verifyRightIcon },
  { leftIcon: bookingLeftIcon, rightIcon: bookingRightIcon }
]

const isPrivateOrLocalHost = (hostname) => {
  const host = String(hostname || '').trim().toLowerCase()
  if (!host) return false
  if (host === 'localhost' || host === '127.0.0.1' || host === '::1') return true
  if (/^10\.\d+\.\d+\.\d+$/.test(host)) return true
  if (/^192\.168\.\d+\.\d+\.\d+$/.test(host)) return true
  if (/^172\.(1[6-9]|2\d|3[0-1])\.\d+\.\d+$/.test(host)) return true
  return false
}

const registerUrl = computed(() => {
  const rawURL = String(landing.value?.register_url || '').trim()
  const refCode = String(route.params.refCode || '').trim()
  if (!rawURL) return ''
  if (typeof window === 'undefined') return rawURL
  const { protocol, hostname, origin } = window.location
  if ((protocol !== 'https:' && protocol !== 'http:') || !isPrivateOrLocalHost(hostname) || !refCode) {
    return rawURL
  }
  return `${origin}/merchant/${refCode}/login`
})

const featureItems = computed(() => {
  const source = Array.isArray(landing.value?.highlights) ? landing.value.highlights : []
  return source.slice(0, 3).map((item, index) => ({
    ...item,
    ...(featureAssets[index] || featureAssets[featureAssets.length - 1])
  }))
})

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

const copyRegisterUrl = async () => {
  if (!registerUrl.value) return
  try {
    await navigator.clipboard.writeText(registerUrl.value)
    alert('注册链接已复制')
  } catch (_) {
    alert('复制失败，请手动复制链接')
  }
}

const goRegister = () => {
  if (!registerUrl.value) return
  window.location.href = registerUrl.value
}

onMounted(loadLanding)
</script>

<style scoped>
.merchant-referral-page {
  min-height: 100vh;
  background:
    radial-gradient(circle at 84% 6%, rgba(255, 247, 236, 0.54) 0%, rgba(255, 225, 192, 0.2) 13%, transparent 28%),
    linear-gradient(
      180deg,
      #ff7b1b 0,
      #ff8a28 92px,
      #ff9d43 164px,
      #ffbb72 224px,
      #ffd9ba 272px,
      #fff0e4 318px,
      #fff8f1 356px,
      #fffdfa 394px
    );
}

.merchant-referral-state {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  color: #6b7280;
  font-size: 14px;
}

.merchant-referral-state-error {
  text-align: center;
}

.merchant-referral-shell {
  max-width: 440px;
  margin: 0 auto;
  padding-bottom: 18px;
}

.hero-banner {
  position: relative;
  min-height: 246px;
  overflow: hidden;
  padding: 34px 24px 30px;
  background: transparent;
}

.hero-copy {
  position: relative;
  z-index: 2;
  max-width: 238px;
}

.hero-kicker {
  color: rgba(255, 255, 255, 0.92);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.34em;
}

.hero-title {
  margin: 22px 0 0;
  color: #fff;
  font-size: 35px;
  line-height: 1.1;
  font-weight: 900;
  letter-spacing: -0.03em;
}

.hero-subtitle {
  margin: 20px 0 0;
  color: rgba(255, 255, 255, 0.98);
  font-size: 16px;
  line-height: 1.9;
  font-weight: 700;
}

.hero-illustration {
  position: absolute;
  right: -6px;
  top: 8px;
  width: 176px;
  height: 132px;
}

.hero-illustration img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.feature-list {
  margin: 10px 16px 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.feature-card {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 144px;
  padding: 20px 18px 20px 16px;
  border-radius: 28px;
  background: #fff;
  border: 1px solid rgba(246, 233, 216, 0.9);
  box-shadow:
    0 18px 36px rgba(245, 147, 39, 0.07),
    inset 0 1px 0 rgba(255, 255, 255, 0.96);
}

.feature-left-icon {
  width: 86px;
  height: 86px;
  flex: 0 0 86px;
  object-fit: contain;
}

.feature-copy {
  min-width: 0;
  flex: 1;
}

.feature-title {
  margin: 0;
  color: #ff6f19;
  font-size: 19px;
  line-height: 1.35;
  font-weight: 800;
}

.feature-description {
  margin: 10px 0 0;
  color: #4b5563;
  font-size: 14px;
  line-height: 1.85;
}

.feature-right-icon {
  width: 62px;
  height: 62px;
  flex: 0 0 62px;
  object-fit: contain;
}

.register-panel {
  margin: 16px;
  padding: 20px 18px 22px;
  border-radius: 28px;
  background: #fff;
  border: 1px solid rgba(246, 233, 216, 0.9);
  box-shadow:
    0 18px 36px rgba(245, 147, 39, 0.07),
    inset 0 1px 0 rgba(255, 255, 255, 0.96);
}

.register-heading-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.register-accent {
  flex: 0 0 4px;
  width: 4px;
  height: 18px;
  margin-top: 4px;
  border-radius: 999px;
  background: linear-gradient(180deg, #ff6c12 0%, #ffb05e 100%);
}

.register-title {
  margin: 0;
  color: #111827;
  font-size: 20px;
  line-height: 1.3;
  font-weight: 900;
}

.register-description {
  margin: 7px 0 0;
  color: #4b5563;
  font-size: 14px;
  line-height: 1.8;
}

.register-link-label {
  margin-top: 16px;
  color: #6b7280;
  font-size: 13px;
}

.register-link-box {
  margin-top: 8px;
  min-height: 50px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px 0 14px;
  border-radius: 14px;
  border: 1px solid #ececec;
  background: #f8f8f8;
}

.register-link-text {
  min-width: 0;
  flex: 1;
  color: #4b5563;
  font-size: 12px;
  line-height: 1.5;
  word-break: break-all;
}

.register-copy-button {
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: #6b7280;
  cursor: pointer;
}

.register-copy-button:active,
.register-submit:active {
  transform: scale(0.985);
}

.register-copy-icon {
  width: 18px;
  height: 18px;
}

.register-submit {
  width: 100%;
  height: 54px;
  margin-top: 16px;
  border: none;
  border-radius: 14px;
  background: linear-gradient(135deg, #ff7a18 0%, #ff6500 100%);
  box-shadow: 0 14px 28px rgba(255, 122, 24, 0.18);
  color: #fff;
  font-size: 18px;
  font-weight: 900;
  cursor: pointer;
}
</style>
