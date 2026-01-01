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
          <p>💡 提示：资金将直接进入您的支付宝/微信账户，卡包不参与收款</p>
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
        <div v-for="order in orders" :key="order.id" class="order-card">
          <div class="order-header">
            <span class="order-no">{{ order.order_no }}</span>
            <span class="order-status" :class="order.status">
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
import { ref, onMounted, computed } from 'vue'
import { shopApi } from '../../api/index.js'

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
  loadTemplates()
  loadPaymentConfig()
  loadShopSlug()
  loadOrders()
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
  const labels = { pending: '待支付', confirmed: '已完成', canceled: '已取消' }
  return labels[status] || status
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
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
  background: #f5f5f5;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: #fff;
  border-bottom: 1px solid #eee;
}

.back-btn {
  width: 36px;
  height: 36px;
  border: none;
  background: #f5f5f5;
  border-radius: 8px;
  font-size: 18px;
  cursor: pointer;
}

.header h1 {
  font-size: 18px;
  font-weight: 600;
}

.placeholder {
  width: 36px;
}

.tabs {
  display: flex;
  background: #fff;
  border-bottom: 1px solid #eee;
  overflow-x: auto;
}

.tab {
  flex: 1;
  min-width: 80px;
  padding: 12px 8px;
  text-align: center;
  font-size: 14px;
  color: #666;
  cursor: pointer;
  white-space: nowrap;
}

.tab.active {
  color: #1890ff;
  border-bottom: 2px solid #1890ff;
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
}

.add-btn {
  padding: 8px 16px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: #999;
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.template-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.05);
}

.template-card.inactive {
  opacity: 0.6;
}

.template-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 6px;
}

.template-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.type-tag {
  padding: 2px 8px;
  background: #e6f7ff;
  color: #1890ff;
  border-radius: 4px;
  font-size: 12px;
}

.price {
  font-size: 16px;
  font-weight: 600;
  color: #f50;
}

.template-detail {
  font-size: 12px;
  color: #999;
}

.template-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  padding: 6px 12px;
  border: 1px solid #ddd;
  background: #fff;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
}

.action-btn.danger {
  color: #f50;
  border-color: #f50;
}

.action-btn.success {
  color: #52c41a;
  border-color: #52c41a;
}

/* 收款配置 */
.payment-form {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
}

.form-section {
  margin-bottom: 24px;
}

.form-section h3 {
  font-size: 16px;
  margin-bottom: 12px;
  color: #333;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  color: #666;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #ddd;
  border-radius: 8px;
  font-size: 14px;
  box-sizing: border-box;
}

.form-tip {
  padding: 12px;
  background: #fffbe6;
  border-radius: 8px;
  margin-bottom: 20px;
}

.form-tip p {
  margin: 0;
  font-size: 13px;
  color: #ad8b00;
}

.save-btn {
  width: 100%;
  padding: 12px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
}

.save-btn:disabled {
  background: #ccc;
}

/* 二维码 */
.qrcode-section {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
}

.slug-input {
  display: flex;
  align-items: center;
  border: 1px solid #ddd;
  border-radius: 8px;
  overflow: hidden;
}

.slug-input .prefix {
  padding: 10px 12px;
  background: #f5f5f5;
  color: #999;
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
  color: #999;
}

.qrcode-preview {
  margin-top: 24px;
  text-align: center;
}

.qrcode-preview h3 {
  font-size: 16px;
  margin-bottom: 16px;
}

.qrcode-box {
  display: inline-block;
  padding: 16px;
  background: #fff;
  border: 1px solid #eee;
  border-radius: 12px;
}

.qrcode-box img {
  width: 200px;
  height: 200px;
}

.shop-url {
  margin-top: 12px;
  font-size: 14px;
  color: #666;
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
  background: #f5f5f5;
  border-radius: 8px;
}

.qrcode-tips h4 {
  font-size: 14px;
  margin-bottom: 8px;
}

.qrcode-tips ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.qrcode-tips li {
  font-size: 13px;
  color: #666;
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
  background: #fff;
  border-radius: 12px;
}

.order-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.order-no {
  font-size: 12px;
  color: #999;
}

.order-status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
}

.order-status.pending {
  background: #fff7e6;
  color: #fa8c16;
}

.order-status.confirmed {
  background: #f6ffed;
  color: #52c41a;
}

.order-status.canceled {
  background: #fff1f0;
  color: #f5222d;
}

.order-info {
  margin-bottom: 8px;
}

.order-user {
  font-size: 14px;
  color: #333;
}

.order-card-name {
  font-size: 14px;
  font-weight: 500;
}

.order-price {
  font-size: 16px;
  font-weight: 600;
  color: #f50;
}

.order-footer {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #999;
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
  background: #fff;
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
  border-bottom: 1px solid #eee;
}

.modal-header h3 {
  font-size: 16px;
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
  flex: 1;
  padding: 16px;
  overflow-y: auto;
}

.modal-footer {
  display: flex;
  gap: 12px;
  padding: 16px;
  border-top: 1px solid #eee;
}

.cancel-btn {
  flex: 1;
  padding: 12px;
  background: #f5f5f5;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
}

.confirm-btn {
  flex: 1;
  padding: 12px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
}

.confirm-btn:disabled {
  background: #ccc;
}
</style>
