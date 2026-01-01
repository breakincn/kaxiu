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
                  @click="paymentMethod = 'alipay'"
                >
                  <div class="payment-icon alipay">支</div>
                  <span>支付宝</span>
                </div>
                <div 
                  v-if="shopInfo.payment_config.has_wechat"
                  class="payment-option"
                  :class="{ selected: paymentMethod === 'wechat' }"
                  @click="paymentMethod = 'wechat'"
                >
                  <div class="payment-icon wechat">微</div>
                  <span>微信支付</span>
                </div>
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

      <!-- 支付中弹窗 -->
      <div v-if="showPaymentModal" class="modal-overlay">
        <div class="modal payment-modal">
          <div class="modal-header">
            <h3>完成付款</h3>
          </div>
          
          <div class="modal-body">
            <div class="payment-info">
              <p class="payment-hint">请使用{{ paymentMethod === 'alipay' ? '支付宝' : '微信' }}扫描下方二维码完成付款</p>
              
              <div class="payment-qrcode">
                <img :src="paymentUrl" alt="收款码" v-if="paymentUrl && isImageUrl(paymentUrl)" />
                <div v-else-if="paymentUrl" class="payment-link">
                  <p>请点击下方链接完成付款</p>
                  <button type="button" class="pay-link-btn" @click="openPaymentLink">去付款</button>
                </div>
              </div>
              
              <div class="payment-amount">
                <span>支付金额：</span>
                <span class="amount">¥{{ (currentOrder?.price / 100).toFixed(2) }}</span>
              </div>
            </div>
            
            <div class="payment-actions">
              <button class="cancel-btn" @click="cancelPayment">取消</button>
              <button class="confirm-btn" @click="confirmPayment" :disabled="confirming">
                {{ confirming ? '开卡中...' : '已完成付款' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 成功弹窗 -->
      <div v-if="showSuccessModal" class="modal-overlay">
        <div class="modal success-modal">
          <div class="success-icon">✓</div>
          <h3>开卡成功</h3>
          <p>卡片已加入您的卡包</p>
          <button class="view-btn" @click="goToCards">查看我的卡包</button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { shopApi } from '../../api/index.js'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const shopInfo = ref(null)
const selectedCard = ref(null)
const showPurchaseModal = ref(false)
const showPaymentModal = ref(false)
const showSuccessModal = ref(false)
const paymentMethod = ref('')
const paymentUrl = ref('')
const currentOrder = ref(null)
const purchasing = ref(false)
const confirming = ref(false)

const isLoggedIn = computed(() => {
  return !!localStorage.getItem('userToken')
})

onMounted(() => {
  loadShopInfo()
})

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
  
  purchasing.value = true
  try {
    const res = await shopApi.createDirectPurchase({
      card_template_id: selectedCard.value.id,
      payment_method: paymentMethod.value
    })
    
    currentOrder.value = res.data.data
    paymentUrl.value = res.data.data.payment_url
    
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

function isNavigableLink(url) {
  if (!url) return false
  const u = String(url).trim()
  return u.startsWith('https://') || u.startsWith('http://') || u.startsWith('weixin://') || u.startsWith('wxp://') || u.startsWith('alipay://')
}

function openPaymentLink() {
  if (!paymentUrl.value) return
  if (!isNavigableLink(paymentUrl.value)) {
    alert('付款链接格式不正确，请让商户配置正确的收款链接，或使用收款码图片。')
    return
  }
  window.location.assign(paymentUrl.value)
}

function getDefaultPaymentMethod() {
  const cfg = shopInfo.value?.payment_config
  if (!cfg) return ''
  if (cfg.has_alipay) return 'alipay'
  if (cfg.has_wechat) return 'wechat'
  return ''
}

function cancelPayment() {
  showPaymentModal.value = false
  currentOrder.value = null
  paymentUrl.value = ''
}

async function confirmPayment() {
  if (!currentOrder.value) return
  
  confirming.value = true
  try {
    await shopApi.confirmDirectPurchase(currentOrder.value.order_no)
    showPaymentModal.value = false
    showSuccessModal.value = true
  } catch (e) {
    alert(e.response?.data?.error || '开卡失败')
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
.payment-info {
  text-align: center;
}

.payment-hint {
  color: #666;
  margin-bottom: 16px;
}

.payment-qrcode {
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
}

.payment-amount .amount {
  font-size: 24px;
  font-weight: 600;
  color: #f50;
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
  border: none;
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
