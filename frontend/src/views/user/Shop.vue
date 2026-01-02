<template>
  <div class="shop-page">
    <!-- 加载中 -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>加载中...</p>
    </div>

    <!-- 商户不存在 -->
    <div v-else-if="!shopInfo" class="error-state">
      <div class="error-icon">😕</div>
      <h2>店铺不存在</h2>
      <p>请检查链接是否正确</p>
    </div>

    <!-- 商户信息 -->
    <template v-else>
      <!-- 商户头部 -->
      <div class="shop-header">
        <div class="merchant-avatar">{{ shopInfo.merchant.name.charAt(0) }}</div>
        <div class="merchant-info">
          <h1 class="merchant-name">{{ shopInfo.merchant.name }}</h1>
          <p class="merchant-type">{{ shopInfo.merchant.type }}</p>
        </div>
      </div>

      <!-- 在售卡片列表 -->
      <div class="card-section">
        <h2 class="section-title">在售卡片</h2>
        
        <div v-if="shopInfo.card_templates.length === 0" class="empty-cards">
          <p>暂无在售卡片</p>
        </div>
        
        <div v-else class="card-list">
          <div 
            v-for="card in shopInfo.card_templates" 
            :key="card.id" 
            class="card-item"
            @click="selectCard(card)"
          >
            <div class="card-content">
              <div class="card-name">{{ card.name }}</div>
              <div class="card-meta">
                <span class="card-type">{{ getCardTypeLabel(card.card_type) }}</span>
                <span v-if="card.card_type !== 'balance'" class="card-times">{{ card.total_times }}次</span>
                <span v-else class="card-amount">充{{ (card.recharge_amount / 100).toFixed(0) }}元</span>
              </div>
              <div v-if="card.description" class="card-desc">{{ card.description }}</div>
              <div class="card-validity">
                {{ card.valid_days > 0 ? `${card.valid_days}天有效` : '永久有效' }}
              </div>
            </div>
            <div class="card-price">
              <span class="price-label">¥</span>
              <span class="price-value">{{ (card.price / 100).toFixed(2) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 购买弹窗 -->
      <div v-if="showPurchaseModal" class="modal-overlay" @click.self="closePurchaseModal">
        <div class="modal purchase-modal">
          <div class="modal-header">
            <h3>确认购买</h3>
            <button class="close-btn" @click="closePurchaseModal">×</button>
          </div>
          
          <div class="modal-body">
            <div class="purchase-card-info">
              <div class="purchase-card-name">{{ selectedCard.name }}</div>
              <div class="purchase-card-meta">
                <span>{{ getCardTypeLabel(selectedCard.card_type) }}</span>
                <span v-if="selectedCard.card_type !== 'balance'">· {{ selectedCard.total_times }}次</span>
                <span v-else>· 充{{ (selectedCard.recharge_amount / 100).toFixed(0) }}元</span>
                <span>· {{ selectedCard.valid_days > 0 ? `${selectedCard.valid_days}天有效` : '永久有效' }}</span>
              </div>
              <div class="purchase-price">
                <span class="price-label">¥</span>
                <span class="price-value">{{ (selectedCard.price / 100).toFixed(2) }}</span>
              </div>
            </div>
            
            <!-- 未登录提示 -->
            <div v-if="!isLoggedIn" class="login-prompt">
              <p>请先登录后再购买</p>
              <button class="login-btn" @click="goLogin">去登录</button>
            </div>
            
            <!-- 选择支付方式 -->
            <div v-else class="payment-methods">
              <h4>选择支付方式</h4>
              <div class="payment-options">
                <div 
                  v-if="shopInfo.payment_config.has_alipay"
                  class="payment-option"
                  :class="{ selected: paymentMethod === 'alipay' }"
                  @click="handlePaymentMethodClick('alipay')"
                >
                  <div class="payment-icon alipay">支</div>
                  <span>支付宝</span>
                </div>
                <div 
                  v-if="shopInfo.payment_config.has_wechat"
                  class="payment-option"
                  :class="{ selected: paymentMethod === 'wechat' }"
                  @click="handlePaymentMethodClick('wechat')"
                >
                  <div class="payment-icon wechat">微</div>
                  <span>微信支付</span>
                </div>
              </div>

              <div v-if="!shopInfo.payment_config.has_alipay || !shopInfo.payment_config.has_wechat" style="margin-bottom: 16px;">
                <button
                  v-if="!shopInfo.payment_config.has_alipay"
                  class="purchase-btn"
                  type="button"
                  @click="launchScanApp('alipay')"
                  style="margin-bottom: 10px;"
                >
                  支付宝扫码支付
                </button>
                <button
                  v-if="!shopInfo.payment_config.has_wechat"
                  class="purchase-btn"
                  type="button"
                  @click="launchScanApp('wechat')"
                >
                  微信扫码支付
                </button>
              </div>
              
              <div class="purchase-tip">
                <p>💡 付款将直接转给商户，卡包不参与收款</p>
              </div>
              
              <button 
                class="purchase-btn" 
                @click="createPurchase"
                :disabled="!paymentMethod || purchasing"
              >
                {{ purchasing ? '处理中...' : '立即购买' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 绑定手机号弹窗 -->
      <div v-if="showBindPhoneModal" class="modal-overlay" @click.self="closeBindPhoneModal">
        <div class="modal purchase-modal">
          <div class="modal-header">
            <h3>绑定手机号</h3>
            <button class="close-btn" @click="closeBindPhoneModal">×</button>
          </div>

          <div class="modal-body">
            <div class="mb-4">
              <label class="block text-gray-700 text-sm font-medium mb-2">手机号</label>
              <input
                v-model="bindPhoneForm.phone"
                type="tel"
                placeholder="请输入手机号"
                class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              />
            </div>

            <div class="mb-4">
              <label class="block text-gray-700 text-sm font-medium mb-2">验证码</label>
              <div class="flex gap-2">
                <input
                  v-model="bindPhoneForm.code"
                  type="text"
                  placeholder="请输入验证码"
                  class="flex-1 px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
                />
                <button
                  type="button"
                  :disabled="sendingBindCode || bindCountdown > 0"
                  class="px-3 py-3 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-200 transition-colors disabled:opacity-50 whitespace-nowrap shrink-0"
                  @click="sendBindCode"
                >
                  {{ bindCountdown > 0 ? `${bindCountdown}s` : (sendingBindCode ? '发送中...' : '发送验证码') }}
                </button>
              </div>
            </div>

            <button
              class="purchase-btn"
              @click="submitBindPhone"
              :disabled="bindingPhone"
            >
              {{ bindingPhone ? '处理中...' : '绑定并继续' }}
            </button>
          </div>
        </div>
      </div>

      <!-- 支付中弹窗 -->
      <div v-if="showPaymentModal" class="modal-overlay" @click.self="cancelPayment">
        <div class="modal payment-modal">
          <div class="modal-header">
            <h3 class="payment-title">{{ paymentTitle }}</h3>
            <button class="close-btn" @click="cancelPayment">×</button>
          </div>
          
          <div class="modal-body">
            <div class="payment-info">
              <div v-if="paymentUrl" class="payment-qrcode">
                <div v-if="isImageUrl(paymentUrl)" style="width: 100%;">
                  <div style="text-align: center; color: #666; font-size: 14px; margin-bottom: 10px;" @click="launchScanApp(paymentMethod)">点击支付码 直接扫码支付</div>
                  <img :src="paymentUrl" alt="收款码" @click="launchScanApp(paymentMethod)" />
                </div>
              </div>
              <button
                v-if="paymentMethod && !paymentUrl"
                class="purchase-btn"
                type="button"
                @click="launchScanApp(paymentMethod)"
              >
                {{ paymentMethod === 'alipay' ? '打开支付宝扫一扫' : '打开微信扫一扫' }}
              </button>
              
              <div class="payment-amount">
                <span>支付金额：</span>
                <span class="amount">¥{{ (currentOrder?.price / 100).toFixed(2) }}</span>
              </div>
              
              <button class="save-payment-btn" @click="savePayment" :disabled="saveButtonDisabled">
                保存支付码至手机付款
              </button>

              <div v-if="showPaymentGuide" class="payment-guide" :class="{ highlighted: guideHighlighted }" @click="openPaymentApp">
                <div class="payment-guide-icon">📱</div>
                <div class="payment-guide-text">
<!--                  打开{{ paymentMethod === 'alipay' ? '支付宝' : '微信' }}扫一扫,点击相册,选择支付码;确认输入付款¥{{ (currentOrder?.price / 100).toFixed(2) }}元-->
<!--                  在{{ paymentMethod === 'alipay' ? '支付宝' : '微信' }}中：点击"相册" → 选择刚保存的支付码 → 确认支付¥{{ (currentOrder?.price / 100).toFixed(2) }}-->
                  打开{{ paymentMethod === 'alipay' ? '支付宝' : '微信' }}扫一扫 → 点击相册 → 选择支付码;输入付款金额¥{{ (currentOrder?.price / 100).toFixed(2) }}元
                </div>
              </div>
            </div>
            
            <div v-if="showPaymentActions" class="payment-actions">
              <button class="cancel-btn" @click="cancelPayment">取消</button>
              <button class="confirm-btn" @click="confirmPayment" :disabled="confirming">
                {{ confirming ? '提交中...' : '已完成付款' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 成功弹窗 -->
      <div v-if="showSuccessModal" class="modal-overlay">
        <div class="modal success-modal">
          <div class="success-icon">✓</div>
          <h3>已提交付款</h3>
          <p>等待商户确认后，卡片将自动加入您的卡包</p>
          <button class="view-btn" @click="goToCards">查看我的卡包</button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authApi, shopApi, smsApi, userApi } from '../../api/index.js'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const shopInfo = ref(null)
const selectedCard = ref(null)
const showPurchaseModal = ref(false)
const showPaymentModal = ref(false)
const showSuccessModal = ref(false)

const currentUser = ref(null)
const showBindPhoneModal = ref(false)
const bindingPhone = ref(false)
const sendingBindCode = ref(false)

const bindPhoneForm = ref({
  phone: '',
  code: ''
})

function hasPaymentQRCode(method) {
  const cfg = shopInfo.value?.payment_config
  if (!cfg) return false
  if (method === 'alipay') return !!cfg.has_alipay
  if (method === 'wechat') return !!cfg.has_wechat
  return false
}

function launchScanApp(method) {
  if (!method) return
  paymentMethod.value = method
  try {
    if (method === 'alipay') {
      window.location.href = 'alipayqr://platformapi/startapp?saId=10000007'
    } else {
      window.location.href = 'weixin://'
    }
  } catch (e) {
  }
}

const bindCountdown = ref(0)
let bindTimer = null

const pendingAfterBind = ref(null)
const paymentMethod = ref('')
const paymentUrl = ref('')
const currentOrder = ref(null)
const purchasing = ref(false)
const confirming = ref(false)

const showPaymentGuide = ref(true)
const showPaymentActions = ref(false)
const saveButtonDisabled = ref(false)
const guideHighlighted = ref(false)

let paymentActionsTimer = null

onBeforeUnmount(() => {
  if (paymentActionsTimer) {
    clearTimeout(paymentActionsTimer)
    paymentActionsTimer = null
  }

	if (bindTimer) {
		clearInterval(bindTimer)
		bindTimer = null
	}
})

const isLoggedIn = computed(() => {
  return !!localStorage.getItem('userToken')
})

const paymentTitle = computed(() => {
  const card = selectedCard.value
  if (!card) return '完成付款'

  if (card.card_type === 'balance') {
    return `${card.name} 充${(card.recharge_amount / 100).toFixed(0)}元`
  }

  const unit = card.card_type === 'lesson' ? '课时' : '次'
  return `${card.name} ${card.total_times}${unit}`
})

onMounted(() => {
  loadShopInfo()
	if (isLoggedIn.value) {
		fetchCurrentUser()
	}
})

async function fetchCurrentUser() {
	try {
		const res = await authApi.getCurrentUser()
		currentUser.value = res.data.data
	} catch (e) {
		currentUser.value = null
	}
}

function hasBoundPhone(user) {
	return !!(user && user.phone)
}

function closeBindPhoneModal() {
	showBindPhoneModal.value = false
	bindPhoneForm.value = { phone: '', code: '' }
	bindCountdown.value = 0
	if (bindTimer) {
		clearInterval(bindTimer)
		bindTimer = null
	}
	pendingAfterBind.value = null
}

function startBindCountdown() {
	bindCountdown.value = 60
	bindTimer = setInterval(() => {
		bindCountdown.value -= 1
		if (bindCountdown.value <= 0) {
			clearInterval(bindTimer)
			bindTimer = null
		}
	}, 1000)
}

async function sendBindCode() {
	if (!bindPhoneForm.value.phone) {
		alert('请先输入手机号')
		return
	}

	sendingBindCode.value = true
	try {
		const res = await smsApi.send(bindPhoneForm.value.phone, 'user_bind_phone')
		const debugCode = res.data?.data?.debug_code
		if (debugCode) {
			alert(`验证码已发送（开发模式）：${debugCode}`)
		} else {
			alert('验证码已发送')
		}
		startBindCountdown()
	} catch (err) {
		alert(err.response?.data?.error || '发送失败，请重试')
	} finally {
		sendingBindCode.value = false
	}
}

async function submitBindPhone() {
	if (!bindPhoneForm.value.phone) {
		alert('请输入手机号')
		return
	}
	if (!bindPhoneForm.value.code) {
		alert('请输入验证码')
		return
	}

	bindingPhone.value = true
	try {
		const res = await userApi.bindPhone(bindPhoneForm.value.phone, bindPhoneForm.value.code)
		currentUser.value = res.data.data
		showBindPhoneModal.value = false
		const fn = pendingAfterBind.value
		pendingAfterBind.value = null
		bindPhoneForm.value = { phone: '', code: '' }
		if (fn) {
			await fn()
		}
	} catch (err) {
		alert(err.response?.data?.error || '绑定失败，请重试')
	} finally {
		bindingPhone.value = false
	}
}

async function ensurePhoneBound(continuation) {
	if (!isLoggedIn.value) return true
	await fetchCurrentUser()
	if (hasBoundPhone(currentUser.value)) return true
	showBindPhoneModal.value = true
	pendingAfterBind.value = continuation || null
	return false
}

async function loadShopInfo() {
  loading.value = true
  try {
    const slug = route.params.slug
    const id = route.params.id
    
    let res
    if (slug) {
      res = await shopApi.getShopInfo(slug)
    } else if (id) {
      res = await shopApi.getShopInfoByID(id)
    } else {
      shopInfo.value = null
      return
    }
    
    shopInfo.value = res.data.data
  } catch (e) {
    console.error('加载店铺信息失败', e)
    shopInfo.value = null
  } finally {
    loading.value = false
  }
}

function getCardTypeLabel(type) {
  const labels = { times: '次数卡', lesson: '课时卡', balance: '充值卡' }
  return labels[type] || type
}

function selectCard(card) {
  selectedCard.value = card
  paymentMethod.value = getDefaultPaymentMethod()
  showPurchaseModal.value = true
}

function handlePaymentMethodClick(method) {
  if (paymentMethod.value === method) {
    // 如果点击的是已选中的支付方式，直接触发购买
    createPurchase()
  } else {
    // 否则只是切换支付方式
    paymentMethod.value = method
  }
}

function closePurchaseModal() {
  showPurchaseModal.value = false
  selectedCard.value = null
}

function goLogin() {
  // 保存当前页面用于登录后返回
  localStorage.setItem('redirectAfterLogin', route.fullPath)
  router.push('/user/login')
}

async function createPurchase() {
  if (!paymentMethod.value) {
    paymentMethod.value = getDefaultPaymentMethod()
  }

  if (!paymentMethod.value) {
    alert('商户未配置收款方式')
    return
  }

	const ok = await ensurePhoneBound(createPurchase)
	if (!ok) {
		return
	}
  
  purchasing.value = true
  try {
    const res = await shopApi.createDirectPurchase({
      card_template_id: selectedCard.value.id,
      payment_method: paymentMethod.value
    })
    
    currentOrder.value = res.data.data
    paymentUrl.value = res.data.data.payment_url

    showPaymentGuide.value = true
    showPaymentActions.value = false
    if (paymentActionsTimer) {
      clearTimeout(paymentActionsTimer)
      paymentActionsTimer = null
    }
    
    showPurchaseModal.value = false
    showPaymentModal.value = true
  } catch (e) {
    alert(e.response?.data?.error || '创建订单失败')
  } finally {
    purchasing.value = false
  }
}

function isImageUrl(url) {
  if (!url) return false
  return url.match(/\.(jpg|jpeg|png|gif|webp)$/i) || url.includes('qr') || url.includes('code')
}

function getDefaultPaymentMethod() {
  const cfg = shopInfo.value?.payment_config
  if (!cfg) return ''
  if (cfg.default_method === 'alipay' && cfg.has_alipay) return 'alipay'
  if (cfg.default_method === 'wechat' && cfg.has_wechat) return 'wechat'
  if (cfg.has_alipay) return 'alipay'
  if (cfg.has_wechat) return 'wechat'
  return ''
}

async function savePayment() {
  if (!paymentUrl.value) return

  try {
    const resp = await fetch(paymentUrl.value)
    const blob = await resp.blob()

    const extByType = {
      'image/png': 'png',
      'image/jpeg': 'jpg',
      'image/webp': 'webp',
    }
    const ext = extByType[blob.type] || 'jpg'
    const filename = `payment_qrcode_${Date.now()}.${ext}`

    // 只尝试系统分享，不触发下载
    if (navigator.share && window.File) {
      try {
        const file = new File([blob], filename, { type: blob.type || 'image/jpeg' })
        await navigator.share({ files: [file], title: '收款码' })
      } catch (e) {
        // 分享失败也不进行下载
        console.log('分享失败或取消')
      }
    } else {
      // 不支持分享API的情况，也不进行下载
      console.log('当前浏览器不支持分享功能')
    }
  } catch (e) {
    // fetch 失败也不进行任何操作
    console.log('获取图片失败')
  }

  // 保存后按钮变灰，提示条高亮可点击
  saveButtonDisabled.value = true
  guideHighlighted.value = true
  
  // 不再使用30秒定时器自动显示底部按钮
  // 用户需要点击提示条来显示底部按钮
}

function openPaymentApp() {
  if (!guideHighlighted.value) return
  
  const isAlipay = paymentMethod.value === 'alipay'
  
  // 尝试调起对应的支付应用
  try {
    if (isAlipay) {
      // 尝试调起支付宝
      window.location.href = 'alipayqr://platformapi/startapp?saId=10000007'
    } else {
      // 尝试调起微信
      window.location.href = 'weixin://'
    }
  } catch (e) {
    // 调起失败，显示底部按钮
  }
  
  // 15秒后隐藏提示条并显示底部按钮
  if (paymentActionsTimer) {
    clearTimeout(paymentActionsTimer)
    paymentActionsTimer = null
  }
  paymentActionsTimer = setTimeout(() => {
    showPaymentGuide.value = false
    showPaymentActions.value = true
    paymentActionsTimer = null
  }, 15000)
}

function cancelPayment() {
  showPaymentModal.value = false
  currentOrder.value = null
  paymentUrl.value = ''

  showPaymentGuide.value = true
  showPaymentActions.value = false
  saveButtonDisabled.value = false
  guideHighlighted.value = false
  if (paymentActionsTimer) {
    clearTimeout(paymentActionsTimer)
    paymentActionsTimer = null
  }
}

async function confirmPayment() {
  if (!currentOrder.value) return

  if (!currentOrder.value.order_no) {
    alert('订单信息缺失，请重新发起购买')
    return
  }
  if (!currentOrder.value.card_template_id) {
    alert('卡片信息缺失，请重新发起购买')
    return
  }
  if (!currentOrder.value.payment_method) {
    alert('支付方式缺失，请重新选择支付方式')
    return
  }
  
  confirming.value = true
  try {
    await shopApi.confirmDirectPurchase(currentOrder.value.order_no, {
      card_template_id: currentOrder.value.card_template_id,
      payment_method: currentOrder.value.payment_method
    })
    showPaymentModal.value = false
    showSuccessModal.value = true

    showPaymentGuide.value = true
    showPaymentActions.value = false
    saveButtonDisabled.value = false
    guideHighlighted.value = false
    if (paymentActionsTimer) {
      clearTimeout(paymentActionsTimer)
      paymentActionsTimer = null
    }
  } catch (e) {
    const rawMsg = e?.response?.data?.error || e?.message || ''
    const friendlyMsg = rawMsg && String(rawMsg).toLowerCase().includes('eof')
      ? '提交失败：请求参数异常，请重新发起购买'
      : (rawMsg || '提交失败')
    alert(friendlyMsg)
  } finally {
    confirming.value = false
  }
}

function goToCards() {
  router.push('/user/cards')
}
</script>

<style scoped>
.shop-page {
  min-height: 100vh;
  background: linear-gradient(180deg, #1890ff 0%, #1890ff 180px, #f5f5f5 180px);
}

.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  color: #fff;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-state {
  color: #333;
  background: #f5f5f5;
}

.error-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.shop-header {
  display: flex;
  align-items: center;
  padding: 24px 20px 40px;
  color: #fff;
}

.merchant-avatar {
  width: 64px;
  height: 64px;
  background: rgba(255,255,255,0.2);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 600;
  margin-right: 16px;
}

.merchant-name {
  font-size: 22px;
  font-weight: 600;
  margin: 0 0 4px;
}

.merchant-type {
  font-size: 14px;
  opacity: 0.9;
  margin: 0;
}

.card-section {
  margin: -20px 16px 0;
  padding-bottom: 20px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 12px;
  color: #333;
}

.empty-cards {
  padding: 40px;
  text-align: center;
  background: #fff;
  border-radius: 12px;
  color: #999;
}

.card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.card-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  cursor: pointer;
  transition: transform 0.2s;
}

.card-item:active {
  transform: scale(0.98);
}

.card-name {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 6px;
}

.card-meta {
  display: flex;
  gap: 8px;
  font-size: 13px;
  color: #666;
  margin-bottom: 4px;
}

.card-type {
  color: #1890ff;
}

.card-desc {
  font-size: 12px;
  color: #999;
  margin-bottom: 4px;
}

.card-validity {
  font-size: 12px;
  color: #999;
}

.card-price {
  text-align: right;
}

.price-label {
  font-size: 14px;
  color: #f50;
}

.price-value {
  font-size: 24px;
  font-weight: 600;
  color: #f50;
}

/* 弹窗 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: flex-end;
  justify-content: center;
  z-index: 1000;
}

.modal {
  width: 100%;
  max-height: 80vh;
  background: #fff;
  border-radius: 20px 20px 0 0;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #eee;
}

.modal-header h3 {
  font-size: 18px;
  margin: 0;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: none;
  font-size: 24px;
  color: #999;
  cursor: pointer;
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
}

/* 购买弹窗 */
.purchase-card-info {
  padding: 16px;
  background: #f9f9f9;
  border-radius: 12px;
  margin-bottom: 20px;
}

.purchase-card-name {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 8px;
}

.purchase-card-meta {
  font-size: 14px;
  color: #666;
  margin-bottom: 12px;
}

.purchase-price .price-label {
  font-size: 16px;
}

.purchase-price .price-value {
  font-size: 28px;
}

.login-prompt {
  text-align: center;
  padding: 20px;
}

.login-prompt p {
  color: #666;
  margin-bottom: 16px;
}

.login-btn {
  padding: 12px 40px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
}

.payment-methods h4 {
  font-size: 14px;
  color: #666;
  margin-bottom: 12px;
}

.payment-options {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.payment-option {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  border: 2px solid #eee;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.payment-option.selected {
  border-color: #1890ff;
  background: #e6f7ff;
}

.payment-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 600;
  color: #fff;
  margin-bottom: 8px;
}

.payment-icon.alipay {
  background: #1677ff;
}

.payment-icon.wechat {
  background: #07c160;
}

.purchase-tip {
  padding: 12px;
  background: #fffbe6;
  border-radius: 8px;
  margin-bottom: 20px;
}

.purchase-tip p {
  margin: 0;
  font-size: 13px;
  color: #ad8b00;
}

.purchase-btn {
  width: 100%;
  padding: 14px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
}

.purchase-btn:disabled {
  background: #ccc;
}

/* 支付弹窗 */
.payment-modal .modal-header {
  justify-content: space-between;
  align-items: center;
}

.payment-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
  text-align: center;
  flex: 1;
}

.payment-modal .close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: none;
  font-size: 24px;
  color: #999;
  cursor: pointer;
  flex-shrink: 0;
}

.payment-info {
  text-align: center;
}

.payment-qrcode {
  display: flex;
  justify-content: center;
  margin-bottom: 16px;
}

.payment-qrcode img {
  max-width: 200px;
  border-radius: 8px;
}

.payment-link {
  padding: 20px;
}

.pay-link-btn {
  display: inline-block;
  padding: 12px 32px;
  background: #1890ff;
  color: #fff;
  text-decoration: none;
  border-radius: 8px;
  margin-top: 12px;
}

.payment-amount {
  font-size: 16px;
  color: #333;
  margin-bottom: 20px;
}

.payment-amount .amount {
  font-size: 24px;
  font-weight: 600;
  color: #f50;
}

.payment-guide {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  background: #f2f4f7;
  border-radius: 10px;
  margin-bottom: 8px;
  text-align: left;
  transition: all 0.3s ease;
}

.payment-guide.highlighted {
  background: linear-gradient(135deg, #fff8e1 0%, #ffe082 100%);
  border: 2px dashed #ffa726;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(255, 167, 38, 0.2);
  transform: translateY(-2px);
  animation: pulse-animation 2s ease-in-out infinite;
}

.payment-guide.highlighted .payment-guide-text {
  color: #e65100;
  font-weight: 500;
}

.payment-guide.highlighted .payment-guide-icon {
  color: #e65100;
}

.payment-guide-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex: 0 0 auto;
}

.payment-guide-text {
  color: #333;
  font-size: 14px;
  line-height: 1.4;
}

@keyframes pulse-animation {
  0%, 100% {
    transform: scale(1);
    box-shadow: 0 2px 8px rgba(255, 167, 38, 0.2);
  }
  50% {
    transform: scale(1.02);
    box-shadow: 0 4px 16px rgba(255, 167, 38, 0.4);
  }
}

.save-payment-btn {
  width: 100%;
  padding: 16px;
  background: linear-gradient(135deg, #fff8e1 0%, #ffe082 100%);
  color: #e65100;
  border: 2px dashed #ffa726;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  margin-bottom: 20px;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(255, 167, 38, 0.2);
}

.save-payment-btn:not(:disabled) {
  animation: pulse-animation 2s ease-in-out infinite;
}

.save-payment-btn:disabled {
  background: linear-gradient(135deg, #f5f5f5 0%, #e0e0e0 100%);
  color: #999;
  border: none;
  cursor: not-allowed;
  transform: none;
  animation: none;
  box-shadow: none;
}

.payment-actions {
  display: flex;
  gap: 12px;
  margin-top: 20px;
}

.cancel-btn {
  flex: 1;
  padding: 14px;
  background: #f5f5f5;
  color: #666;
  border: 1px solid #ddd;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
}

.confirm-btn {
  flex: 2;
  padding: 14px;
  background: #52c41a;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
}

.confirm-btn:disabled {
  background: #ccc;
}

/* 成功弹窗 */
.success-modal {
  max-width: 320px;
  margin: auto;
  border-radius: 20px;
  padding: 40px 20px;
  text-align: center;
}

.success-icon {
  width: 64px;
  height: 64px;
  background: #52c41a;
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  margin: 0 auto 16px;
}

.success-modal h3 {
  font-size: 20px;
  margin-bottom: 8px;
}

.success-modal p {
  color: #666;
  margin-bottom: 24px;
}

.view-btn {
  width: 100%;
  padding: 14px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
}
</style>
