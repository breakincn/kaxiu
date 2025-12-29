import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000
})

// 请求拦截器 - 添加 token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('userToken')
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
    if (error.response && error.response.status === 401) {
      // 未登录或 token 过期，跳转到登录页
      localStorage.removeItem('userToken')
      localStorage.removeItem('userId')
      localStorage.removeItem('userName')
      window.location.href = '/user/login'
    }
    return Promise.reject(error)
  }
)

export const authApi = {
  login: (phone, password) => api.post('/login', { phone, password }),
  getCurrentUser: () => api.get('/me')
}

export const userApi = {
  getUsers: () => api.get('/users'),
  getUser: (id) => api.get(`/users/${id}`),
  createUser: (data) => api.post('/users', data),
  getCurrentUser: () => api.get('/me')
}

export const merchantApi = {
  getMerchants: () => api.get('/merchants'),
  getMerchant: (id) => api.get(`/merchants/${id}`),
  createMerchant: (data) => api.post('/merchants', data),
  updateMerchant: (id, data) => api.put(`/merchants/${id}`, data),
  getQueueStatus: (id) => api.get(`/merchants/${id}/queue`)
}

export const cardApi = {
  getCards: () => api.get('/cards'),
  getCard: (id) => api.get(`/cards/${id}`),
  getUserCards: (userId, status) => api.get(`/users/${userId}/cards`, { params: { status } }),
  getMerchantCards: (merchantId) => api.get(`/merchants/${merchantId}/cards`),
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

export default api
