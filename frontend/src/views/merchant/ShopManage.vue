<template>
  <div class="shop-manage">
    <!-- 顶部导航 -->
    <div class="header">
      <button class="back-btn" @click="$router.push('/merchant')">
        <span class="icon">←</span>
      </button>
      <h1>售卡管理</h1>
      <div class="placeholder"></div>
    </div>

    <!-- Tab 切换 -->
    <div class="tabs">
      <div 
        class="tab" 
        :class="{ active: activeTab === 'templates' }"
        @click="activeTab = 'templates'"
      >在售卡片</div>
      <div 
        class="tab" 
        :class="{ active: activeTab === 'payment' }"
        @click="activeTab = 'payment'"
      >收款配置</div>
      <div 
        class="tab" 
        :class="{ active: activeTab === 'qrcode' }"
        @click="activeTab = 'qrcode'"
      >售卡二维码</div>
      <div 
        class="tab" 
        :class="{ active: activeTab === 'orders' }"
        @click="activeTab = 'orders'"
      >直购订单</div>
    </div>

    <!-- 在售卡片列表 -->
    <div v-if="activeTab === 'templates'" class="tab-content">
      <div class="section-header">
        <h2>在售卡片模板</h2>
        <button class="add-btn" @click="showTemplateModal = true">+ 添加</button>
      </div>
      
      <div v-if="templates.length === 0" class="empty-state">
        <p>暂无在售卡片，点击上方"添加"创建</p>
      </div>
      
      <div v-else class="template-list">
        <div 
          v-for="tpl in templates" 
          :key="tpl.id" 
          class="template-item"
        >
          <div
            class="template-card"
            :class="{ inactive: !tpl.is_active }"
          >
            <div class="template-info">
              <div class="template-name">{{ tpl.name }}</div>
              <div class="template-meta">
                <span class="type-tag">{{ getCardTypeLabel(tpl.card_type) }}</span>
                <span class="price">¥{{ (tpl.price / 100).toFixed(2) }}</span>
              </div>
              <div class="template-detail">
                <span v-if="tpl.card_type !== 'balance'">{{ tpl.total_times }}次</span>
                <span v-else>充值{{ (tpl.recharge_amount / 100).toFixed(0) }}元</span>
                <span v-if="tpl.valid_days > 0">· {{ tpl.valid_days }}天有效</span>
                <span v-else>· 永久有效</span>
              </div>
            </div>
            <div class="template-actions">
              <button class="action-btn" @click="editTemplate(tpl)">编辑</button>
              <button
                class="action-btn"
                :class="tpl.is_active ? 'danger' : 'success'"
                @click="toggleTemplateStatus(tpl)"
              >
                {{ tpl.is_active ? '下架' : '上架' }}
              </button>
            </div>
          </div>

        </div>
      </div>
    </div>

    <!-- 收款配置 -->
    <div v-if="activeTab === 'payment'" class="tab-content">
      <div class="payment-form">
        <div class="form-section">
          <h3>支付宝收款</h3>
          <div class="form-group">
            <label>收款码图片</label>
            <input type="file" accept="image/*" @change="(e) => onUploadQRCode(e, 'alipay')" />
            <div v-if="paymentConfig.alipay_qr_code" class="qr-preview">
              <img :src="paymentConfig.alipay_qr_code" alt="支付宝收款码" />
            </div>
          </div>
          <button
            v-if="canSetDefault && paymentConfig.alipay_qr_code"
            type="button"
            class="action-btn"
            :class="paymentConfig.default_method === 'alipay' ? 'success' : ''"
            @click="paymentConfig.default_method = 'alipay'"
          >
            {{ paymentConfig.default_method === 'alipay' ? '默认收款码' : '设为默认' }}
          </button>
        </div>
        
        <div class="form-section">
          <h3>微信收款</h3>
          <div class="form-group">
            <label>收款码图片</label>
            <input type="file" accept="image/*" @change="(e) => onUploadQRCode(e, 'wechat')" />
            <div v-if="paymentConfig.wechat_qr_code" class="qr-preview">
              <img :src="paymentConfig.wechat_qr_code" alt="微信收款码" />
            </div>
          </div>
          <button
            v-if="canSetDefault && paymentConfig.wechat_qr_code"
            type="button"
            class="action-btn"
            :class="paymentConfig.default_method === 'wechat' ? 'success' : ''"
            @click="paymentConfig.default_method = 'wechat'"
          >
            {{ paymentConfig.default_method === 'wechat' ? '默认收款码' : '设为默认' }}
          </button>
        </div>
        
        <div class="form-tip">
          <p>💡 提示：资金将直接进入您的支付宝/微信账户，卡包不参与收款，如不配置收款码则只能到店现场扫码支付</p>
        </div>
        
        <button class="save-btn" @click="savePaymentConfig" :disabled="saving">
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>
    </div>

    <!-- 售卡二维码 -->
    <div v-if="activeTab === 'qrcode'" class="tab-content">
      <div class="qrcode-section">
        <div class="form-group">
          <label>店铺短链接</label>
          <div class="slug-input">
            <span class="prefix">kabao.me/shop/</span>
            <input 
              v-model="shopSlug" 
              type="text" 
              placeholder="yourshop"
              @input="slugChanged = true"
            />
          </div>
          <p class="slug-tip">只能包含字母、数字、下划线和连字符，2-30个字符</p>
        </div>
        
        <button 
          v-if="slugChanged" 
          class="save-btn" 
          @click="saveShopSlug"
          :disabled="saving"
        >
          {{ saving ? '保存中...' : '保存短链接' }}
        </button>
        
        <div v-if="shopSlug && !slugChanged" class="qrcode-preview">
          <h3>您的售卡二维码</h3>
          <div class="qrcode-box">
            <img :src="qrcodeUrl" alt="售卡二维码" v-if="qrcodeUrl" />
            <div v-else class="qrcode-placeholder">
              <p>二维码生成中...</p>
            </div>
          </div>
          <p class="shop-url">{{ shopFullUrl }}</p>
          <div class="qrcode-actions">
            <button class="action-btn" @click="copyShopUrl">复制链接</button>
            <button class="action-btn" @click="downloadQrcode">下载二维码</button>
          </div>
        </div>
        
        <div class="qrcode-tips">
          <h4>使用场景</h4>
          <ul>
            <li>📍 前台/收银台张贴</li>
            <li>📱 朋友圈/群分享</li>
            <li>🎴 印在海报/名片上</li>
          </ul>
        </div>
      </div>
    </div>

    <!-- 直购订单 -->
    <div v-if="activeTab === 'orders'" class="tab-content">
      <div v-if="orders.length === 0" class="empty-state">
        <p>暂无直购订单</p>
      </div>
      
      <div v-else class="order-list">
        <div v-for="order in orders" :key="order.id" class="order-card" :class="{ 'paid-order': order.status === 'paid' }">
          <div class="order-header">
            <span class="order-no">{{ order.order_no }}</span>
            <button
              v-if="order.status === 'paid'"
              type="button"
              class="order-status confirm-order"
              :class="order.status"
              @click="confirmMerchantOrder(order)"
              :disabled="confirmingMap[order.order_no]"
            >
              {{ confirmingMap[order.order_no] ? '确认中...' : '确认订单' }}
            </button>
            <span v-else class="order-status" :class="order.status">
              {{ getOrderStatusLabel(order.status) }}
            </span>
          </div>
          <div class="order-info">
            <div class="order-user">用户：{{ order.user?.nickname || order.user?.phone }}</div>
            <div class="order-card-name">{{ order.card_template?.name }}</div>
            <div class="order-price">¥{{ (order.price / 100).toFixed(2) }}</div>
          </div>
          <div class="order-footer">
            <span class="order-time">{{ formatTime(order.created_at) }}</span>
            <span v-if="order.status === 'paid'" class="paid-elapsed">
              已付款 {{ formatElapsed(order.paid_at) }}
            </span>
            <span class="payment-method">{{ order.payment_method === 'alipay' ? '支付宝' : '微信' }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑卡片模板弹窗 -->
    <div v-if="showTemplateModal" class="modal-overlay" @click.self="closeTemplateModal">
      <div class="modal">
        <div class="modal-header">
          <h3>{{ editingTemplate ? '编辑卡片模板' : '添加卡片模板' }}</h3>
          <button class="close-btn" @click="closeTemplateModal">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>卡片名称 *</label>
            <input v-model="templateForm.name" type="text" placeholder="如：洗车10次卡" />
          </div>
          <div class="form-group">
            <label>卡片类型 *</label>
            <select v-model="templateForm.card_type">
              <option value="times">次数卡</option>
              <option value="lesson">课时卡</option>
              <option value="balance">充值卡</option>
            </select>
          </div>
          <div class="form-group">
            <label>售价（元）*</label>
            <input v-model.number="templateForm.priceYuan" type="number" min="0.01" step="0.01" placeholder="如：100" />
          </div>
          <div class="form-group" v-if="templateForm.card_type !== 'balance'">
            <label>总次数 *</label>
            <input v-model.number="templateForm.total_times" type="number" min="1" placeholder="如：10" />
          </div>
          <div class="form-group" v-if="templateForm.card_type === 'balance'">
            <label>充值金额（元）*</label>
            <input v-model.number="templateForm.rechargeAmountYuan" type="number" min="1" placeholder="如：100" />
          </div>
          <div class="form-group">
            <label>有效期（天）</label>
            <input v-model.number="templateForm.valid_days" type="number" min="0" placeholder="0表示永久有效" />
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea v-model="templateForm.description" placeholder="卡片描述（可选）" rows="2"></textarea>
          </div>
        </div>
        <div class="modal-footer">
          <button class="cancel-btn" @click="closeTemplateModal">取消</button>
          <button class="confirm-btn" @click="saveTemplate" :disabled="saving">
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { shopApi } from '../../api/index.js'

const route = useRoute()

const activeTab = ref('templates')
const loading = ref(false)
const saving = ref(false)

// 卡片模板
const templates = ref([])
const showTemplateModal = ref(false)
const editingTemplate = ref(null)
const templateForm = ref({
  name: '',
  card_type: 'times',
  priceYuan: '',
  total_times: '',
  rechargeAmountYuan: '',
  valid_days: 0,
  description: ''
})

// 收款配置
const paymentConfig = ref({
  alipay_qr_code: '',
  wechat_qr_code: '',
  default_method: ''
})

const canSetDefault = computed(() => {
  return !!paymentConfig.value.alipay_qr_code && !!paymentConfig.value.wechat_qr_code
})

// 店铺短链接
const shopSlug = ref('')
const slugChanged = ref(false)

// 直购订单
const orders = ref([])

const confirmingMap = ref({})
const nowTick = ref(Date.now())
let nowTimer = null
let ordersPollTimer = null

// 二维码URL
const shopFullUrl = computed(() => {
  if (!shopSlug.value) return ''
  return `${window.location.origin}/shop/${shopSlug.value}`
})

const qrcodeUrl = computed(() => {
  if (!shopSlug.value || slugChanged.value) return ''
  return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(shopFullUrl.value)}`
})

onMounted(() => {
  const tabParam = route.query.tab
  if (tabParam && ['templates', 'payment', 'qrcode', 'orders'].includes(tabParam)) {
    activeTab.value = tabParam
  }

  loadTemplates()
  loadPaymentConfig()
  loadShopSlug()
  loadOrders()

  nowTimer = setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)

  ordersPollTimer = setInterval(() => {
    loadOrders()
  }, 5000)
})

onBeforeUnmount(() => {
  if (nowTimer) {
    clearInterval(nowTimer)
    nowTimer = null
  }
  if (ordersPollTimer) {
    clearInterval(ordersPollTimer)
    ordersPollTimer = null
  }
})

async function loadTemplates() {
  try {
    const res = await shopApi.getCardTemplates()
    templates.value = res.data.data || []
  } catch (e) {
    console.error('加载卡片模板失败', e)
  }
}

async function loadPaymentConfig() {
  try {
    const res = await shopApi.getPaymentConfig()
    if (res.data.data) {
      paymentConfig.value = {
        alipay_qr_code: res.data.data.alipay_qr_code || '',
        wechat_qr_code: res.data.data.wechat_qr_code || '',
        default_method: res.data.data.default_method || ''
      }
    }
  } catch (e) {
    console.error('加载收款配置失败', e)
  }
}

async function onUploadQRCode(e, method) {
  const file = e?.target?.files?.[0]
  if (!file) return
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await shopApi.uploadPaymentQRCode(fd)
    const url = res?.data?.data?.url || ''
    if (!url) {
      alert('上传失败')
      return
    }
    if (method === 'alipay') {
      paymentConfig.value.alipay_qr_code = url
    } else {
      paymentConfig.value.wechat_qr_code = url
    }
    if (!paymentConfig.value.default_method) {
      paymentConfig.value.default_method = method
    }
  } catch (err) {
    alert(err.response?.data?.error || '上传失败')
  } finally {
    if (e?.target) e.target.value = ''
  }
}

async function loadShopSlug() {
  try {
    const res = await shopApi.getShopSlug()
    if (res.data.data) {
      shopSlug.value = res.data.data.slug
    }
  } catch (e) {
    console.error('加载店铺短链接失败', e)
  }
}

async function loadOrders() {
  try {
    const res = await shopApi.getMerchantDirectPurchases()
    orders.value = res.data.data || []
  } catch (e) {
    console.error('加载直购订单失败', e)
  }
}

function getCardTypeLabel(type) {
  const labels = { times: '次数卡', lesson: '课时卡', balance: '充值卡' }
  return labels[type] || type
}

function getOrderStatusLabel(status) {
  const labels = { pending: '待支付', paid: '待确认', confirmed: '已完成', canceled: '已取消' }
  return labels[status] || status
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}

function formatElapsed(fromTime) {
  if (!fromTime) return ''
  const fromTs = new Date(fromTime).getTime()
  if (!fromTs) return ''
  const diff = Math.max(0, Math.floor((nowTick.value - fromTs) / 1000))
  const h = Math.floor(diff / 3600)
  const m = Math.floor((diff % 3600) / 60)
  const s = diff % 60
  if (h > 0) return `${h}小时${m}分${s}秒`
  if (m > 0) return `${m}分${s}秒`
  return `${s}秒`
}

async function confirmMerchantOrder(order) {
  if (!order?.order_no) return
  if (!confirm('确认已收到该笔付款，并为用户发卡吗？')) return
  confirmingMap.value = { ...confirmingMap.value, [order.order_no]: true }
  try {
    await shopApi.confirmMerchantDirectPurchase(order.order_no)
    await loadOrders()
  } catch (e) {
    alert(e.response?.data?.error || '确认失败')
  } finally {
    confirmingMap.value = { ...confirmingMap.value, [order.order_no]: false }
  }
}

function editTemplate(tpl) {
  editingTemplate.value = tpl
  templateForm.value = {
    name: tpl.name,
    card_type: tpl.card_type,
    priceYuan: tpl.price / 100,
    total_times: tpl.total_times,
    rechargeAmountYuan: tpl.recharge_amount / 100,
    valid_days: tpl.valid_days,
    description: tpl.description
  }
  showTemplateModal.value = true
}

function closeTemplateModal() {
  showTemplateModal.value = false
  editingTemplate.value = null
  templateForm.value = {
    name: '',
    card_type: 'times',
    priceYuan: '',
    total_times: '',
    rechargeAmountYuan: '',
    valid_days: 0,
    description: ''
  }
}

async function saveTemplate() {
  const form = templateForm.value
  if (!form.name || !form.priceYuan) {
    alert('请填写必填项')
    return
  }
  
  const data = {
    name: form.name,
    card_type: form.card_type,
    price: Math.round(form.priceYuan * 100),
    total_times: form.total_times || 0,
    recharge_amount: Math.round((form.rechargeAmountYuan || 0) * 100),
    valid_days: form.valid_days || 0,
    description: form.description || ''
  }
  
  saving.value = true
  try {
    if (editingTemplate.value) {
      await shopApi.updateCardTemplate(editingTemplate.value.id, data)
    } else {
      await shopApi.createCardTemplate(data)
    }
    closeTemplateModal()
    loadTemplates()
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleTemplateStatus(tpl) {
  try {
    await shopApi.updateCardTemplate(tpl.id, { is_active: !tpl.is_active })
    loadTemplates()
  } catch (e) {
    alert('操作失败')
  }
}

async function savePaymentConfig() {
  const config = paymentConfig.value
  if (!config.alipay_qr_code && !config.wechat_qr_code) {
    alert('请至少配置一种收款方式')
    return
  }
  
  saving.value = true
  try {
    await shopApi.savePaymentConfig(config)
    alert('保存成功')
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function saveShopSlug() {
  if (!shopSlug.value) {
    alert('请输入店铺短链接')
    return
  }
  
  saving.value = true
  try {
    await shopApi.saveShopSlug(shopSlug.value)
    slugChanged.value = false
    alert('保存成功')
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

function copyShopUrl() {
  navigator.clipboard.writeText(shopFullUrl.value)
  alert('链接已复制')
}

function downloadQrcode() {
  const link = document.createElement('a')
  link.href = qrcodeUrl.value
  link.download = `shop-qrcode-${shopSlug.value}.png`
  link.click()
}
</script>

<style scoped>
.shop-manage {
  min-height: 100vh;
  background: var(--kb-surface-muted);
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: var(--kb-surface);
  border-bottom: 1px solid var(--kb-border);
}

.back-btn {
  width: 36px;
  height: 36px;
  border: none;
  background: var(--kb-surface-muted);
  border-radius: 8px;
  font-size: 18px;
  cursor: pointer;
}

.header h1 {
  font-size: 18px;
  font-weight: 600;
  color: var(--kb-text);
}

.placeholder {
  width: 36px;
}

.tabs {
  display: flex;
  background: var(--kb-surface);
  border-bottom: 1px solid var(--kb-border);
  overflow-x: auto;
}

.tab {
  flex: 1;
  min-width: 80px;
  padding: 12px 8px;
  text-align: center;
  font-size: 14px;
  color: var(--kb-text-muted);
  cursor: pointer;
  white-space: nowrap;
}

.tab.active {
  color: var(--kb-primary);
  border-bottom: 2px solid var(--kb-primary);
}

.tab-content {
  padding: 16px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-header h2 {
  font-size: 16px;
  font-weight: 600;
  color: var(--kb-text);
}

.add-btn {
  padding: 8px 16px;
  background: var(--kb-primary);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--kb-text-muted);
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.template-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.template-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--kb-surface);
  border-radius: 12px;
  box-shadow: var(--kb-shadow);
}

.template-card.inactive {
  opacity: 0.6;
}

.template-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--kb-text);
}

.template-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.type-tag {
  padding: 2px 8px;
  background: var(--kb-primary-soft);
  color: var(--kb-primary-dark);
  border-radius: 4px;
  font-size: 12px;
}

.price {
  font-size: 16px;
  font-weight: 600;
  color: var(--kb-text);
}

.template-detail {
  font-size: 12px;
  color: var(--kb-text-muted);
}

.template-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  padding: 6px 12px;
  border: 1px solid var(--kb-border);
  background: var(--kb-surface);
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  color: var(--kb-text);
}

.action-btn.danger {
  color: var(--kb-primary-dark);
  border-color: var(--kb-primary);
}

.action-btn.success {
  color: var(--kb-primary-dark);
  border-color: var(--kb-primary);
}

/* 收款配置 */
.payment-form {
  background: var(--kb-surface);
  border-radius: 12px;
  padding: 20px;
}

.form-section {
  margin-bottom: 24px;
}

.form-section h3 {
  font-size: 16px;
  margin-bottom: 12px;
  color: var(--kb-text);
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  color: var(--kb-text-muted);
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--kb-border);
  border-radius: 8px;
  font-size: 14px;
  box-sizing: border-box;
  background: var(--kb-surface);
  color: var(--kb-text);
}

.form-tip {
  padding: 12px;
  background: var(--kb-warning-soft);
  border-radius: 8px;
  margin-bottom: 20px;
}

.form-tip p {
  margin: 0;
  font-size: 13px;
  color: var(--kb-text-muted);
}

.save-btn {
  width: 100%;
  padding: 12px;
  background: var(--kb-primary);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
}

.save-btn:disabled {
  background: var(--kb-border);
}

/* 二维码 */
.qrcode-section {
  background: var(--kb-surface);
  border-radius: 12px;
  padding: 20px;
}

.slug-input {
  display: flex;
  align-items: center;
  border: 1px solid var(--kb-border);
  border-radius: 8px;
  overflow: hidden;
}

.slug-input .prefix {
  padding: 10px 12px;
  background: var(--kb-surface-muted);
  color: var(--kb-text-muted);
  font-size: 14px;
}

.slug-input input {
  flex: 1;
  padding: 10px 12px;
  border: none;
  font-size: 14px;
}

.slug-tip {
  margin-top: 6px;
  font-size: 12px;
  color: var(--kb-text-muted);
}

.qrcode-preview {
  margin-top: 24px;
  text-align: center;
}

.qrcode-preview h3 {
  font-size: 16px;
  margin-bottom: 16px;
  color: var(--kb-text);
}

.qrcode-box {
  display: inline-block;
  padding: 16px;
  background: var(--kb-surface);
  border: 1px solid var(--kb-border);
  border-radius: 12px;
}

.qrcode-box img {
  width: 200px;
  height: 200px;
}

.shop-url {
  margin-top: 12px;
  font-size: 14px;
  color: var(--kb-text-muted);
}

.qrcode-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  margin-top: 16px;
}

.qrcode-tips {
  margin-top: 24px;
  padding: 16px;
  background: var(--kb-surface-muted);
  border-radius: 8px;
}

.qrcode-tips h4 {
  font-size: 14px;
  margin-bottom: 8px;
  color: var(--kb-text);
}

.qrcode-tips ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.qrcode-tips li {
  font-size: 13px;
  color: var(--kb-text-muted);
  margin-bottom: 4px;
}

/* 订单列表 */
.order-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.order-card {
  padding: 16px;
  background: var(--kb-surface);
  border-radius: 12px;
  box-shadow: var(--kb-shadow);
}

.order-card.paid-order {
  border: 1px solid var(--kb-primary);
  background: var(--kb-primary-soft);
}

.order-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.order-no {
  font-size: 12px;
  color: var(--kb-text-muted);
}

.order-status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
}

.order-status.pending {
  background: var(--kb-surface-muted);
  color: var(--kb-text-muted);
}

.order-status.paid {
  background: var(--kb-primary-soft);
  color: var(--kb-primary-dark);
}

.order-status.confirmed {
  background: var(--kb-accent-soft);
  color: var(--kb-accent);
}

.order-status.canceled {
  background: var(--kb-surface-muted);
  color: var(--kb-text-muted);
}

.order-status.confirm-order {
  border: none;
  cursor: pointer;
}

.order-status.paid.confirm-order {
  color: var(--kb-primary-dark);
  border: 1px solid var(--kb-primary);
}

.order-status.confirm-order:not(:disabled) {
  animation: pulse-animation 2s ease-in-out infinite;
}

.order-status.confirm-order:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.paid-elapsed {
  color: var(--kb-primary-dark);
}

@keyframes pulse-animation {
  0%, 100% {
    transform: scale(1);
    box-shadow: 0 0 0 rgba(255, 107, 53, 0);
  }
  50% {
    transform: scale(1.02);
    box-shadow: 0 4px 16px rgba(255, 107, 53, 0.22);
  }
}

.order-info {
  margin-bottom: 8px;
}

.order-user {
  font-size: 14px;
  color: var(--kb-text);
}

.order-card-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--kb-text);
}

.order-price {
  font-size: 16px;
  font-weight: 600;
  color: var(--kb-text);
}

.order-footer {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--kb-text-muted);
}

.paid-orders {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.paid-order-notice {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 12px;
  background: var(--kb-warning-soft);
  border: 1px solid var(--kb-border);
  border-radius: 10px;
}

.notice-left {
  min-width: 0;
}

.notice-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--kb-text);
  margin-bottom: 4px;
}

.notice-sub {
  font-size: 12px;
  color: var(--kb-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.notice-action {
  flex: none;
  padding: 8px 12px;
  border: none;
  background: var(--kb-primary);
  color: #fff;
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
}

.notice-action:disabled {
  background: var(--kb-border);
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
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  width: 90%;
  max-width: 400px;
  max-height: 80vh;
  background: var(--kb-surface);
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--kb-border);
}

.modal-header h3 {
  font-size: 16px;
  margin: 0;
  color: var(--kb-text);
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: none;
  font-size: 24px;
  color: var(--kb-text-muted);
  cursor: pointer;
}

.modal-body {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
}

.modal-footer {
  display: flex;
  gap: 12px;
  padding: 16px;
  border-top: 1px solid var(--kb-border);
}

.cancel-btn {
  flex: 1;
  padding: 12px;
  background: var(--kb-surface-muted);
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  color: var(--kb-text);
}

.confirm-btn {
  flex: 1;
  padding: 12px;
  background: var(--kb-primary);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
}

.confirm-btn:disabled {
  background: var(--kb-border);
}
</style>
