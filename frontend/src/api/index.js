import axios from 'axios'

import {
  clearMerchantAuth,
  clearMerchantPermissionKeys,
  getMerchantActiveAuth,
  getMerchantToken,
  getTechnicianShopSlug,
  setMerchantPermissionKeys
} from '../utils/auth'

const defaultApiBaseURL = import.meta.env.DEV ? '/api' : 'https://api.kabao.app'
const apiBaseURL = import.meta.env.VITE_API_BASE_URL || defaultApiBaseURL

const api = axios.create({
  baseURL: apiBaseURL,
  timeout: 10000
})

const host = typeof window !== 'undefined' ? window.location.host : ''
const isMerchantApp = host === 'kabao.shop' || host.endsWith('.kabao.shop')

const isTechnicianLoginPath = (pathname) => /^\/s\/[^/]+\/login$/.test(pathname)

const isMerchantContextPath = (pathname) => {
  if (isMerchantApp) return true
  return pathname.startsWith('/merchant') || isTechnicianLoginPath(pathname)
}

const isPlatformAdminPath = (pathname) => pathname.startsWith('/platform-admin')

// 请求拦截器 - 添加 token
api.interceptors.request.use(
  (config) => {
    // 根据当前路径判断是用户还是商户
    const isMerchant = isMerchantContextPath(window.location.pathname)

    if (isPlatformAdminPath(window.location.pathname)) {
      const t = localStorage.getItem('platformAdminToken')
      if (t) {
        config.headers['X-Platform-Admin-Token'] = t
      }
      return config
    }

    const token = isMerchant ? getMerchantToken() : localStorage.getItem('userToken')

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
      console.log('收到401错误，检查错误类型和请求上下文')
      
      // 检查是否是主动登录请求的失败
      const isLoginRequest = error.config?.url?.includes('/login')
      const isPostMethod = error.config?.method?.toLowerCase() === 'post'
      
      // 如果是主动登录请求失败，不清空现有登录态，但清除临时状态
      if (isLoginRequest && isPostMethod) {
        console.log('主动登录请求失败，不清空现有登录态')
        // 清除可能的临时状态，避免状态污染
        sessionStorage.removeItem('merchantActiveAuth')
        sessionStorage.removeItem('technicianShopSlug')
        return Promise.reject(error)
      }
      
      // 其他 401 错误（如 token 过期、无效 token 等）需要清空登录态
      console.log('会话过期或无效token，清除登录态并跳转登录页')
      // 根据当前路径判断跳转到哪个登录页
      const pathname = window.location.pathname
      const isMerchant = isMerchantContextPath(pathname)
      
      if (isMerchant) {
        // 商户端：仅清空“当前激活身份”的登录信息，避免覆盖其它账号
        const active = getMerchantActiveAuth()
        clearMerchantPermissionKeys()
        clearMerchantAuth()
        import('../router').then(({ default: router }) => {
          if (isTechnicianLoginPath(pathname)) {
            router.replace(pathname)
          } else {
            if (active === 'technician') {
              const slug = getTechnicianShopSlug()
              if (slug) {
                router.replace(`/s/${slug}/login`)
              } else {
                router.replace('/login')
              }
            } else {
              router.replace('/login')
            }
          }
        })
      } else {
        // 用户端，清空用户登录信息
        localStorage.removeItem('userToken')
        localStorage.removeItem('userId')
        localStorage.removeItem('userName')
        import('../router').then(({ default: router }) => {
          router.replace('/login')
        })
      }
    }
    return Promise.reject(error)
  }
)

export const authApi = {
  login: (username, password) => api.post('/user/login', { username, password }),
  register: (data) => api.post('/user/register', data),
  getCurrentUser: () => api.get('/user/me')
}

export const smsApi = {
  sendCode: (phone, type) => api.post('/user/sms/send', { phone, type }),
}

export const platformApi = {
  getServiceRoles: () => api.get('/platform/service-roles')
}

export const platformAdminApi = {
  listServiceRoles: () => api.get('/admin/service-roles'),
  createServiceRole: (data) => api.post('/admin/service-roles', data),
  updateServiceRole: (id, data) => api.put(`/admin/service-roles/${id}`, data),
  deleteServiceRole: (id) => api.delete(`/admin/service-roles/${id}`),

  listPermissions: () => api.get('/admin/permissions'),
  createPermission: (data) => api.post('/admin/permissions', data),
  updatePermission: (id, data) => api.put(`/admin/permissions/${id}`, data),
  deletePermission: (id) => api.delete(`/admin/permissions/${id}`),

  getRolePermissions: (roleId) => api.get(`/admin/service-roles/${roleId}/permissions`),
  setRolePermissions: (roleId, data) => api.post(`/admin/service-roles/${roleId}/permissions`, data)
}

export const userApi = {
  getUsers: () => api.get('/user/users'),
  getUser: (id) => api.get(`/user/users/${id}`),
  createUser: (data) => api.post('/user/users', data),
  bindPhone: (phone, code) => api.post('/user/bind-phone', { phone, code }),
  getUserCode: () => api.get('/user/code'),
  getCurrentUser: () => api.get('/user/me')
}

export const merchantApi = {
  register: (data) => api.post('/merchant/register', data),
  login: (phone, password) => api.post('/merchant/login', { phone, password }),
  getMerchants: () => api.get('/merchant/merchants'),
  getMerchant: (id) => api.get(`/merchant/merchants/${id}`),
  createMerchant: (data) => api.post('/merchant/merchants', data),
  updateMerchant: (id, data) => api.put(`/merchant/merchants/${id}`, data),
  getQueueStatus: (id) => api.get(`/merchant/merchants/${id}/queue`),
  searchUsersByPhone: (phone) => api.get('/merchant/users/search', { params: { phone } }),
  bindPhone: (phone, code, password) => api.post('/merchant/bind-phone', { phone, code, password }),
  getCurrentMerchant: () => api.get('/merchant/me'),
  updateCurrentMerchantServices: (data) => api.put('/merchant/services', data),
  updateMerchantInfo: (data) => api.put('/merchant/info', data),
  getNextCardNo: () => api.get('/merchant/next-card-no'),
  toggleBusinessStatus: (data) => api.put('/merchant/business-status', data),

  // 当前账号权限
  getMyPermissions: () => api.get('/merchant/permissions'),

  // 技师（客服类型账号）自身
  getCurrentTechnician: () => api.get('/merchant/technician/me'),
  bindTechnicianPhone: (phone, code) => api.post('/merchant/technician/bind-phone', { phone, code }),

  // 技师账号管理
  getTechnicians: (roleKey) => api.get('/merchant/technicians', { params: { role: roleKey } }),
  createTechnician: (data) => api.post('/merchant/technicians', data),
  updateTechnician: (id, data) => api.put(`/merchant/technicians/${id}`, data),
  deleteTechnician: (id) => api.delete(`/merchant/technicians/${id}`),

  // 角色权限微调
  getRolePermissions: (roleKey) => api.get(`/merchant/role-permissions/${roleKey}`),
  setRolePermissions: (roleKey, data) => api.post(`/merchant/role-permissions/${roleKey}`, data)
}

export const ensureMerchantPermissionsLoaded = async () => {
  try {
    const res = await merchantApi.getMyPermissions()
    const keys = res.data?.data?.permission_keys || []
    setMerchantPermissionKeys(keys)
    return keys
  } catch (e) {
    return []
  }
}

export const cardApi = {
  getCards: () => api.get('/user/cards'),
  getCard: (id) => api.get(`/user/cards/${id}`),
  getUserCards: (userId, status) => api.get(`/user/users/${userId}/cards`, { params: { status } }),
  getMerchantCards: (merchantId, params) => api.get(`/merchant/merchants/${merchantId}/cards`, { params }),
  getMerchantCard: (id) => api.get(`/merchant/cards/${id}`),
  createCard: (data) => api.post('/merchant/cards', data),
  updateCard: (id, data) => api.put(`/merchant/cards/${id}`, data),
  generateVerifyCode: (cardId) => api.post(`/user/cards/${cardId}/verify-code`),
  verifyCard: (code) => api.post('/merchant/verify', { code }),
  scanVerify: (code) => api.post('/merchant/verify/scan', { code }),
  finishVerify: (code) => api.post('/merchant/verify/finish', { code }),
  getTodayVerify: (merchantId) => api.get(`/merchant/merchants/${merchantId}/today-verify`)
}

export const usageApi = {
  getCardUsages: (cardId) => api.get(`/user/cards/${cardId}/usages`),
  getMerchantUsages: (merchantId) => api.get(`/merchant/merchants/${merchantId}/usages`)
}

export const noticeApi = {
  getMerchantNotices: (merchantId, limit) => api.get(`/merchant/merchants/${merchantId}/notices`, { params: { limit } }),
  createNotice: (data) => api.post('/merchant/notices', data),
  deleteNotice: (id) => api.delete(`/merchant/notices/${id}`),
  togglePinNotice: (id) => api.put(`/merchant/notices/${id}/pin`)
}

export const appointmentApi = {
  getMerchantAppointments: (merchantId, status) => api.get(`/merchant/merchants/${merchantId}/appointments`, { params: { status } }),
  getMerchantTechnicians: (merchantId) => api.get(`/merchant/merchants/${merchantId}/technicians`),
  getUserAppointments: (userId) => api.get(`/user/users/${userId}/appointments`),
  getCardAppointment: (cardId) => api.get(`/user/cards/${cardId}/appointment`),
  getAvailableTimeSlots: (merchantId, date) => api.get(`/merchant/merchants/${merchantId}/available-slots`, { params: { date } }),
  createAppointment: (data) => api.post('/user/appointments', data),
  confirmAppointment: (id) => api.put(`/merchant/appointments/${id}/confirm`),
  finishAppointment: (id) => api.put(`/merchant/appointments/${id}/finish`),
  cancelAppointment: (id) => api.put(`/user/appointments/${id}/cancel`)
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
  getShopInfo: (slug) => api.get(`/user/s/${slug}`),
  getShopInfoByID: (id) => api.get(`/user/s/id/${id}`),

  // 技师端：通过店铺短链接登录
  technicianLogin: (slug, account, password) => api.post(`/merchant/s/${slug}/login`, { account, password }),
  
  // 用户端：直购流程
  createDirectPurchase: (data) => api.post('/user/direct-purchase', data),
  confirmDirectPurchase: (orderNo, data) => api.post(`/user/direct-purchase/${orderNo}/confirm`, data),
  getDirectPurchases: () => api.get('/user/direct-purchases'),
  
  // 商户营业状态
  toggleBusinessStatus: (data) => api.put('/merchant/business-status', data)
}

export default api
