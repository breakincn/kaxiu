import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000
})

// 请求拦截器 - 添加 token
api.interceptors.request.use(
  (config) => {
    // 根据当前路径判断是用户还是商户
    const isMerchant = window.location.pathname.startsWith('/merchant')
    const token = isMerchant 
      ? localStorage.getItem('merchantToken')
      : localStorage.getItem('userToken')
    
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器 - 处理未登录错误
api.interceptors.response.use(
  (response) => response,
  (error) => {
    console.log('API响应错误:', error.response?.status, error.config?.url)
    if (error.response && error.response.status === 401) {
      console.log('收到401错误，清除localStorage并跳转登录页')
      // 根据当前路径判断跳转到哪个登录页
      const isMerchant = window.location.pathname.startsWith('/merchant')
      
      if (isMerchant) {
        // 商户端，清空商户登录信息
        localStorage.removeItem('merchantToken')
        localStorage.removeItem('merchantId')
        localStorage.removeItem('merchantName')
        localStorage.removeItem('merchantPhone')
        import('../router').then(({ default: router }) => {
          router.replace('/merchant/login')
        })
      } else {
        // 用户端，清空用户登录信息
        localStorage.removeItem('userToken')
        localStorage.removeItem('userId')
        localStorage.removeItem('userName')
        import('../router').then(({ default: router }) => {
          router.replace('/user/login')
        })
      }
    }
    return Promise.reject(error)
  }
)

export const authApi = {
  login: (username, password) => api.post('/login', { username, password }),
  register: (data) => api.post('/user/register', data),
  getCurrentUser: () => api.get('/me')
}

export const smsApi = {
  send: (phone, purpose) => api.post('/sms/send', { phone, purpose })
}

export const userApi = {
  getUsers: () => api.get('/users'),
  getUser: (id) => api.get(`/users/${id}`),
  createUser: (data) => api.post('/users', data),
  bindPhone: (phone, code) => api.post('/user/bind-phone', { phone, code }),
  getCurrentUser: () => api.get('/me')
}

export const merchantApi = {
  register: (data) => api.post('/merchant/register', data),
  login: (phone, password) => api.post('/merchant/login', { phone, password }),
  getMerchants: () => api.get('/merchants'),
  getMerchant: (id) => api.get(`/merchants/${id}`),
  createMerchant: (data) => api.post('/merchants', data),
  updateMerchant: (id, data) => api.put(`/merchants/${id}`, data),
  getQueueStatus: (id) => api.get(`/merchants/${id}/queue`),
  searchUsersByPhone: (phone) => api.get('/merchant/users/search', { params: { phone } }),
  bindPhone: (phone, code) => api.post('/merchant/bind-phone', { phone, code }),
  getCurrentMerchant: () => api.get('/merchant/me')
}

export const cardApi = {
  getCards: () => api.get('/cards'),
  getCard: (id) => api.get(`/cards/${id}`),
  getUserCards: (userId, status) => api.get(`/users/${userId}/cards`, { params: { status } }),
  getMerchantCards: (merchantId, params) => api.get(`/merchants/${merchantId}/cards`, { params }),
  createCard: (data) => api.post('/cards', data),
  updateCard: (id, data) => api.put(`/cards/${id}`, data),
  generateVerifyCode: (cardId) => api.post(`/cards/${cardId}/verify-code`),
  verifyCard: (code) => api.post('/verify', { code }),
  getTodayVerify: (merchantId) => api.get(`/merchants/${merchantId}/today-verify`)
}

export const usageApi = {
  getCardUsages: (cardId) => api.get(`/cards/${cardId}/usages`),
  getMerchantUsages: (merchantId) => api.get(`/merchants/${merchantId}/usages`)
}

export const noticeApi = {
  getMerchantNotices: (merchantId, limit) => api.get(`/merchants/${merchantId}/notices`, { params: { limit } }),
  createNotice: (data) => api.post('/notices', data),
  deleteNotice: (id) => api.delete(`/notices/${id}`),
  togglePinNotice: (id) => api.put(`/notices/${id}/pin`)
}

export const appointmentApi = {
  getMerchantAppointments: (merchantId, status) => api.get(`/merchants/${merchantId}/appointments`, { params: { status } }),
  getUserAppointments: (userId) => api.get(`/users/${userId}/appointments`),
  getCardAppointment: (cardId) => api.get(`/cards/${cardId}/appointment`),
  getAvailableTimeSlots: (merchantId, date) => api.get(`/merchants/${merchantId}/available-slots`, { params: { date } }),
  createAppointment: (data) => api.post('/appointments', data),
  confirmAppointment: (id) => api.put(`/appointments/${id}/confirm`),
  finishAppointment: (id) => api.put(`/appointments/${id}/finish`),
  cancelAppointment: (id) => api.put(`/appointments/${id}/cancel`)
}

// ==================== Shop 模块（商户收款二维码 + 卡包直购） ====================
export const shopApi = {
  // 商户端：收款配置
  getPaymentConfig: () => api.get('/merchant/payment-config'),
  savePaymentConfig: (data) => api.post('/merchant/payment-config', data),
  uploadPaymentQRCode: (formData) => api.post('/merchant/payment-qrcode/upload', formData),
  
  // 商户端：卡片模板管理
  getCardTemplates: () => api.get('/merchant/card-templates'),
  createCardTemplate: (data) => api.post('/merchant/card-templates', data),
  updateCardTemplate: (id, data) => api.put(`/merchant/card-templates/${id}`, data),
  deleteCardTemplate: (id) => api.delete(`/merchant/card-templates/${id}`),
  
  // 商户端：店铺短链接
  getShopSlug: () => api.get('/merchant/shop-slug'),
  saveShopSlug: (slug) => api.post('/merchant/shop-slug', { slug }),
  
  // 商户端：直购订单
  getMerchantDirectPurchases: () => api.get('/merchant/direct-purchases'),
  confirmMerchantDirectPurchase: (orderNo) => api.post(`/merchant/direct-purchases/${orderNo}/confirm`),
  
  // 公开接口：店铺信息
  getShopInfo: (slug) => api.get(`/shop/${slug}`),
  getShopInfoByID: (id) => api.get(`/shop/id/${id}`),
  
  // 用户端：直购流程
  createDirectPurchase: (data) => api.post('/direct-purchase', data),
  confirmDirectPurchase: (orderNo, data) => api.post(`/direct-purchase/${orderNo}/confirm`, data),
  getDirectPurchases: () => api.get('/direct-purchases')
}

export default api
