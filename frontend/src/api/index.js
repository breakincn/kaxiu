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
const isLoginPagePath = (pathname) => {
  const p = String(pathname || '')
  return p === '/login' || isTechnicianLoginPath(p) || p.startsWith('/platform-admin/login')
}

const normalizeRequestPath = (config) => {
  const raw = String(config?.url || '').trim()
  if (!raw) return ''
  try {
    const u = raw.startsWith('http://') || raw.startsWith('https://')
      ? new URL(raw)
      : new URL(raw, window.location.origin)
    const path = String(u.pathname || '').trim()
    return path.startsWith('/api/') ? path.slice(4) : path
  } catch (_) {
    let path = raw.split('?')[0].split('#')[0].trim()
    if (path && !path.startsWith('/')) {
      path = `/${path}`
    }
    if (path.startsWith('/api/')) return path.slice(4)
    return path
  }
}

const isPublicAuthRequest = (config) => {
  const method = String(config?.method || 'get').toLowerCase()
  if (method !== 'post') return false
  const path = normalizeRequestPath(config)
  if (!path) return false
  if (path === '/user/login') return true
  if (path === '/user/register') return true
  if (path === '/user/sms/send') return true
  if (path === '/merchant/login') return true
  if (path === '/merchant/register') return true
  if (path === '/platform-admin/login') return true
  if (/^\/merchant\/s\/[^/]+\/login$/.test(path)) return true
  return false
}

const isCrossContextRequest = (config) => {
  const path = normalizeRequestPath(config)
  if (!path) return false
  const isMerchantReq = path.startsWith('/merchant/')
  const isUserReq = path.startsWith('/user/')
  const currentIsMerchant = isMerchantContextPath(window.location.pathname)
  if (isMerchantReq && !currentIsMerchant) return true
  if (isUserReq && currentIsMerchant) return true
  return false
}

const isMerchantContextPath = (pathname) => {
  if (isMerchantApp) return true
  return pathname.startsWith('/merchant') || isTechnicianLoginPath(pathname)
}

const isPlatformAdminPath = (pathname) => pathname.startsWith('/platform-admin')

// 请求拦截器 - 添加 token
api.interceptors.request.use(
  (config) => {
    // 匿名鉴权接口必须不携带任何历史 token，避免旧 token 污染登录链路
    if (isPublicAuthRequest(config)) {
      if (config.headers) {
        delete config.headers.Authorization
        delete config.headers['X-Platform-Admin-Token']
      }
      return config
    }

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

      // 匿名鉴权接口的401（例如登录密码错误）只向上抛出，不触发全局登出逻辑
      if (isPublicAuthRequest(error.config)) {
        console.log('public-auth 401，跳过全局登录态清理')
        return Promise.reject(error)
      }

      // 当前已在登录页时，401只交给页面自身处理，避免二次跳转循环
      if (isLoginPagePath(window.location.pathname)) {
        console.log('login-page 401，跳过全局重定向')
        return Promise.reject(error)
      }

      // 跨上下文请求（如用户页误打商户接口）不触发全局登出，避免误伤登录流程
      if (isCrossContextRequest(error.config)) {
        console.log('cross-context 401，跳过全局登录态清理')
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
        const staffSlug = getTechnicianShopSlug()
        clearMerchantPermissionKeys()
        clearMerchantAuth()
        import('../router').then(({ default: router }) => {
          if (isTechnicianLoginPath(pathname)) {
            router.replace(pathname)
          } else {
            if (active === 'staff') {
              if (staffSlug) {
                router.replace(`/s/${staffSlug}/login`)
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
  setRolePermissions: (roleId, data) => api.post(`/admin/service-roles/${roleId}/permissions`, data),

  getProfessionalBasePermissions: () => api.get('/admin/professional-base-permissions'),
  setProfessionalBasePermissions: (data) => api.post('/admin/professional-base-permissions', data),
  getSchedulerHealth: (params) => api.get('/admin/system/scheduler-health', { params })
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
  updateTechnicianAlias: (data) => api.put('/merchant/technician-alias', data),
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

  // 配置信息（用于前端看板/超时等配置展示）
  getConfig: () => api.get('/merchant/config'),
  getSchedulerHealth: (params) => api.get('/merchant/system/scheduler-health', { params }),

  // 技师（客服类型账号）自身
  getCurrentTechnician: () => api.get('/merchant/technician/me'),
  bindTechnicianPhone: (phone, code) => api.post('/merchant/technician/bind-phone', { phone, code }),
  changeTechnicianPassword: (oldPassword, newPassword) => api.post('/merchant/technician/password/reset', {
    old_password: oldPassword,
    new_password: newPassword
  }),

  // 技师账号管理
  getTechnicians: (roleKey) => {
    const rk = String(roleKey || '').trim()
    if (!rk) return api.get('/merchant/technicians')
    return api.get('/merchant/technicians', { params: { role: rk } })
  },
  createTechnician: (data) => api.post('/merchant/technicians', data),
  updateTechnician: (id, data) => api.put(`/merchant/technicians/${id}`, data),
  getTechnicianOriginalPassword: (id) => api.get(`/merchant/technicians/${id}/original-password`),
  resetTechnicianPassword: (id, newPassword) => api.post(`/merchant/technicians/${id}/reset-password`, {
    new_password: newPassword || ''
  }),
  deleteTechnician: (id) => api.delete(`/merchant/technicians/${id}`),

  // 专业客服岗位（商户自定义）
  getProfessionalRoles: () => api.get('/merchant/professional-roles'),
  createProfessionalRole: (data) => api.post('/merchant/professional-roles', data),

  // 运营客服岗位（平台默认 + 商户自定义）
  getOperationalRoles: () => api.get('/merchant/operational-roles'),
  createOperationalRole: (data) => api.post('/merchant/operational-roles', data),

  // 角色权限微调
  getRolePermissions: (roleKey) => api.get(`/merchant/role-permissions/${roleKey}`),
  setRolePermissions: (roleKey, data) => api.post(`/merchant/role-permissions/${roleKey}`, data),
  getRoleStartPendingSetting: (roleKey) => api.get(`/merchant/role-start-pending-settings/${encodeURIComponent(roleKey)}`),
  setRoleStartPendingSetting: (roleKey, seconds) => api.put(`/merchant/role-start-pending-settings/${encodeURIComponent(roleKey)}`, { start_pending_timeout_seconds: seconds }),

  // 岗位签到配置（按岗位独立控制是否需要签到）
  getRoleAttendanceConfigs: () => api.get('/merchant/role-attendance-configs'),
  setRoleAttendanceConfig: (roleKey, requireAttendance) => api.put(`/merchant/role-attendance-configs/${encodeURIComponent(roleKey)}`, { require_attendance: !!requireAttendance }),

  // 看板（Table）：房间/客服
  getTableRooms: () => api.get('/merchant/table/rooms'),
  getTableStaff: (type) => {
    const t = String(type || '').trim()
    if (!t) return api.get('/merchant/table/staff')
    return api.get('/merchant/table/staff', { params: { type: t } })
  }
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
  getCardProjects: (id) => api.get(`/user/cards/${id}/projects`),
  getUserCards: (userId, status) => api.get(`/user/users/${userId}/cards`, { params: { status } }),
  getVerifyCodeStatus: (code) => api.get(`/user/verify-codes/${encodeURIComponent(code)}/status`),
  getMerchantCards: (merchantId, params) => api.get(`/merchant/merchants/${merchantId}/cards`, { params }),
  getMerchantCard: (id) => api.get(`/merchant/cards/${id}`),
  createCard: (data) => api.post('/merchant/cards', data),
  updateCard: (id, data) => api.put(`/merchant/cards/${id}`, data),
  generateVerifyCode: (cardId, data) => api.post(`/user/cards/${cardId}/verify-code`, data),
  verifyCard: (code) => api.post('/merchant/verify', { code }),
  prepareVerify: (code) => api.post('/merchant/verify/prepare', { code }),
  commitVerify: (verifyToken, handCardNo, skipHandCard = false) => api.post('/merchant/verify/commit', {
    verify_token: verifyToken,
    hand_card_no: handCardNo || '',
    skip_hand_card: !!skipHandCard
  }),
  scanVerify: (code) => api.post('/merchant/verify/scan', { code }),
  getTodayVerify: (merchantId) => api.get(`/merchant/merchants/${merchantId}/today-verify`),
  bindUsageHandCard: (usageId, handCardNo) => api.put(`/merchant/usages/${usageId}/hand-card`, { hand_card_no: handCardNo }),
  queryHandCardForReturn: (handCardNo) => api.get('/merchant/hand-cards/return', { params: { hand_card_no: handCardNo } }),
  returnHandCard: (handCardNo) => api.post('/merchant/hand-cards/return', { hand_card_no: handCardNo }),
  unlockCard: (cardId, reason) => api.post(`/merchant/cards/${cardId}/unlock`, { reason })
}

export const usageApi = {
  getCardUsages: (cardId) => api.get(`/user/cards/${cardId}/usages`),
  revokeUsage: (usageId) => api.put(`/user/usages/${usageId}/revoke`),
  getMerchantUsages: (merchantId, params) => api.get(`/merchant/merchants/${merchantId}/usages`, { params })
}

export const userServiceSessionApi = {
  getSession: (id) => api.get(`/user/service-sessions/${id}`),
  listRooms: (id) => api.get(`/user/service-sessions/${id}/rooms`),
  resume: (id) => api.post(`/user/service-sessions/${id}/resume`),
  chooseRoom: (id, data) => api.post(`/user/service-sessions/${id}/room`, data),
  listTechnicians: (id) => api.get(`/user/service-sessions/${id}/technicians`),
  chooseTechnician: (id, data) => api.post(`/user/service-sessions/${id}/technician`, data),
  extend: (id, data) => api.post(`/user/service-sessions/${id}/extend`, data)
}

export const roomApi = {
  listRooms: () => api.get('/merchant/rooms'),
  createRoom: (data) => api.post('/merchant/rooms', data),
  updateRoom: (id, data) => api.put(`/merchant/rooms/${id}`, data),
  deleteRoom: (id) => api.delete(`/merchant/rooms/${id}`)
}

export const attendanceApi = {
  checkIn: (data) => api.post('/merchant/technician/checkin', data),
  checkOut: (data) => api.post('/merchant/technician/checkout', data),
  updateStatus: (data) => api.put('/merchant/technician/status', data),
  listAvailableTechnicians: () => api.get('/merchant/technicians/available'),
  getCurrentStatus: () => api.get('/merchant/technician/attendance'),
  listSchedulePublishings: (date) => api.get('/merchant/schedules/publishings', { params: { date } }),
  publishNextDaySchedule: () => api.post('/merchant/schedules/publish-next-day'),
  markScheduleLeave: (id) => api.post(`/merchant/schedules/${id}/leave`),
  getScheduleAffectedAppointments: (scheduleId) => api.get('/merchant/schedules/affected-appointments', { params: { schedule_id: scheduleId } })
}

export const queueApi = {
  // 叫号状态管理
  getCallingStatus: () => api.get('/merchant/queue/calling-status'),
  updateCallingStatus: (queuePaused) => api.put('/merchant/queue/calling-status', { queue_paused: queuePaused }),
  // 叫号信息（技师服务页：窗口/当前会话/叫号号数等）
  getCallInfo: () => api.get('/merchant/queue/call-info'),
  // 全店待叫号列表（当天现场队列）
  getPendingList: (params) => api.get('/merchant/queue/pending-list', { params }),
  // 全店“超时过号等待”列表
  getTimeoutWaitingList: (params) => api.get('/merchant/queue/timeout-waiting-list', { params }),
  // 触发下一个叫号（手动叫号模式）
  callNext: () => api.post('/merchant/queue/call-next'),
  // 技师点击“继续叫号”（手动叫号模式）：完成当前单 + 推进下一号
  continueCall: () => api.post('/merchant/queue/continue-call'),
  // 强制结束当前服务并继续叫号
  continueCallForce: () => api.post('/merchant/queue/continue-call-force'),
  // 扫码结单时触发下一个叫号
  finishCallNext: () => api.post('/merchant/queue/finish-call-next'),
  // 当前待上号单自动重分配到其他空闲客服
  reassignCurrentPending: (sessionId, reason) => api.post(`/merchant/queue/sessions/${sessionId}/reassign-current-pending`, { reason }),
  // 技师自己的叫号状态
  getTechnicianQueuePaused: () => api.get('/merchant/technician/queue-paused'),
  updateTechnicianQueuePaused: (queuePaused) => api.put('/merchant/technician/queue-paused', { queue_paused: queuePaused })
}

export const serviceSessionApi = {
  listSessions: (params) => api.get('/merchant/service-sessions', { params }),
  getSession: (id) => api.get(`/merchant/service-sessions/${id}`),
  chooseRoom: (id, data) => api.post(`/merchant/service-sessions/${id}/room`, data),
  chooseTechnician: (id, data) => api.post(`/merchant/service-sessions/${id}/technician`, data),
  extend: (id, data) => api.post(`/merchant/service-sessions/${id}/extend`, data),
  extendDuration: (id, data) => api.post(`/merchant/service-sessions/${id}/extend-duration`, data)
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
  getMerchantSettlement: (id) => api.get(`/merchant/appointments/${id}/settlement`),
  getUserSettlement: (id) => api.get(`/user/appointments/${id}/settlement`),
  getMerchantDelayLedgers: (id) => api.get(`/merchant/appointments/${id}/delay-ledgers`),
  getUserCardDelayLedgers: (cardId) => api.get(`/user/cards/${cardId}/delay-ledgers`),
  getMerchantCompensationSummary: (id) => api.get(`/merchant/appointments/${id}/compensation-summary`),
  createMerchantForceMajeureRelief: (id, data) => api.post(`/merchant/appointments/${id}/force-majeure-relief`, data),
  createUserForceMajeureRelief: (id, data) => api.post(`/user/appointments/${id}/force-majeure-relief`, data),
  acceptForceMajeureRelief: (requestId) => api.post(`/appointments/force-majeure-relief/${requestId}/accept`),
  rejectForceMajeureRelief: (requestId) => api.post(`/appointments/force-majeure-relief/${requestId}/reject`),
  getAvailableTimeSlots: (merchantId, date, projectId) => {
    const params = { date }
    if (projectId) {
      params.project_id = projectId
    }
    return api.get(`/merchant/merchants/${merchantId}/available-slots`, { params })
  },
  createAppointment: (data) => api.post('/user/appointments', data),
  confirmAppointment: (id) => api.put(`/merchant/appointments/${id}/confirm`),
  finishAppointment: (id) => api.put(`/merchant/appointments/${id}/finish`),
  updateResolution: (id, data) => api.put(`/merchant/appointments/${id}/resolution`, data),
  closeMerchantAppointmentException: (id, data) => api.post(`/merchant/appointments/${id}/exception-close`, data),
  reschedule: (id, data) => api.post(`/merchant/appointments/${id}/reschedule`, data),
  getMerchantRescheduleEligibility: (id) => api.get(`/merchant/appointments/${id}/reschedule-eligibility`),
  getMerchantRescheduleSlots: (id, date) => api.get(`/merchant/appointments/${id}/reschedule-slots`, { params: { date } }),
  createMerchantRescheduleRequest: (id, data) => api.post(`/merchant/appointments/${id}/reschedule-requests`, data),
  acceptMerchantRescheduleRequest: (appointmentId, requestId) => api.post(`/merchant/appointments/${appointmentId}/reschedule-requests/${requestId}/accept`),
  rejectMerchantRescheduleRequest: (appointmentId, requestId) => api.post(`/merchant/appointments/${appointmentId}/reschedule-requests/${requestId}/reject`),
  cancelMerchantRescheduleRequest: (appointmentId, requestId) => api.post(`/merchant/appointments/${appointmentId}/reschedule-requests/${requestId}/cancel`),
  getUserRescheduleEligibility: (id) => api.get(`/user/appointments/${id}/reschedule-eligibility`),
  getUserRescheduleSlots: (id, date) => api.get(`/user/appointments/${id}/reschedule-slots`, { params: { date } }),
  getUserRescheduleRecommendations: (id, date) => api.get(`/user/appointments/${id}/reschedule-recommendations`, { params: { date } }),
  createUserRescheduleRequest: (id, data) => api.post(`/user/appointments/${id}/reschedule-requests`, data),
  acceptUserRescheduleRequest: (appointmentId, requestId) => api.post(`/user/appointments/${appointmentId}/reschedule-requests/${requestId}/accept`),
  rejectUserRescheduleRequest: (appointmentId, requestId) => api.post(`/user/appointments/${appointmentId}/reschedule-requests/${requestId}/reject`),
  cancelUserRescheduleRequest: (appointmentId, requestId) => api.post(`/user/appointments/${appointmentId}/reschedule-requests/${requestId}/cancel`),
  createMerchantCancelRequest: (id, data) => api.post(`/merchant/appointments/${id}/cancel-requests`, data),
  acceptMerchantCancelRequest: (appointmentId, requestId) => api.post(`/merchant/appointments/${appointmentId}/cancel-requests/${requestId}/accept`),
  rejectMerchantCancelRequest: (appointmentId, requestId, data) => api.post(`/merchant/appointments/${appointmentId}/cancel-requests/${requestId}/reject`, data),
  createUserCancelRequest: (id, data) => api.post(`/user/appointments/${id}/cancel-requests`, data),
  acceptUserCancelRequest: (appointmentId, requestId) => api.post(`/user/appointments/${appointmentId}/cancel-requests/${requestId}/accept`),
  rejectUserCancelRequest: (appointmentId, requestId, data) => api.post(`/user/appointments/${appointmentId}/cancel-requests/${requestId}/reject`, data),
  updateUserRebuttal: (id, data) => api.post(`/user/appointments/${id}/rebuttal`, data),
  listCompensations: (id) => api.get(`/merchant/appointments/${id}/compensations`),
  createCompensation: (id, data) => api.post(`/merchant/appointments/${id}/compensations`, data),
  cancelAppointment: (id) => api.put(`/user/appointments/${id}/cancel`),
  cancelMerchantAppointment: (id, data) => api.put(`/merchant/appointments/${id}/cancel`, data),
  checkInAppointment: (id) => api.post(`/merchant/appointments/${id}/check-in`),
  getTechnicianMonthlyDisruptions: (id, month) => api.get(`/merchant/technicians/${id}/monthly-disruptions`, { params: { month } }),
  getMerchantAppointmentRepairOverview: (id) => api.get(`/merchant/appointments/${id}/repair-overview`)
}

 export const merchantProjectApi = {
   list: () => api.get('/merchant/projects'),
   create: (data) => api.post('/merchant/projects', data),
   update: (id, data) => api.put(`/merchant/projects/${id}`, data),
   delete: (id) => api.delete(`/merchant/projects/${id}`)
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
  getLiveServiceStatus: (id) => api.get(`/user/merchants/${id}/live-service-status`),

  // 技师端：通过店铺短链接登录
  technicianLogin: (slug, account, password) => api.post(`/merchant/s/${slug}/login`, { account, password }),
  
  // 用户端：直购流程
  createDirectPurchase: (data) => api.post('/user/direct-purchase', data),
  confirmDirectPurchase: (orderNo, data) => api.post(`/user/direct-purchase/${orderNo}/confirm`, data),
  getDirectPurchases: () => api.get('/user/direct-purchases'),
  
  // 商户营业状态
  toggleBusinessStatus: (data) => api.put('/merchant/business-status', data),
  
  // 配置信息
  getConfig: () => api.get('/merchant/config')
}

export default api
