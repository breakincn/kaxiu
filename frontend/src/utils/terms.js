import { getMerchantActiveAuth } from './auth'

export const getStartTerm = (merchant) => {
  const v = merchant?.start_term
  const s = String(v || '').trim()
  return s || '起单'
}

export const getFinishTerm = (merchant) => {
  const v = merchant?.finish_term
  const s = String(v || '').trim()
  return s || '结单'
}

export const getStartTermFromStorage = () => {
  const active = getMerchantActiveAuth()
  const k = active === 'staff' ? 'technicianStartTerm' : 'merchantStartTerm'
  const s = String(localStorage.getItem(k) || '').trim()
  return s || '起单'
}

export const getFinishTermFromStorage = () => {
  const active = getMerchantActiveAuth()
  const k = active === 'staff' ? 'technicianFinishTerm' : 'merchantFinishTerm'
  const s = String(localStorage.getItem(k) || '').trim()
  return s || '结单'
}

export const isQueueModeMerchant = (merchant) => {
  if (!merchant) return false
  const queueMode = String(merchant?.queue_mode || '').trim()
  return !merchant?.support_customer_service_mode && !!merchant?.support_queue && (queueMode === 'auto' || queueMode === 'manual')
}

export const getStartActionTerm = (merchant, options = {}) => {
  if (options.queueMode) return '上号'
  return merchant ? getStartTerm(merchant) : getStartTermFromStorage()
}

export const getFinishActionTerm = (merchant, options = {}) => {
  if (options.queueMode) return '下号'
  return merchant ? getFinishTerm(merchant) : getFinishTermFromStorage()
}

export const getPendingStartLabel = (merchant, options = {}) => `待${getStartActionTerm(merchant, options)}`

export const getPendingFinishLabel = (merchant, options = {}) => `待${getFinishActionTerm(merchant, options)}`

export const getAutoFinishLabel = (merchant, options = {}) => `待自动${getFinishActionTerm(merchant, options)}`

export const getServicePendingStartLabel = (merchant, options = {}) => `服务 ${getPendingStartLabel(merchant, options)}`

export const getServicePendingFinishLabel = (merchant, options = {}) => `服务 ${getPendingFinishLabel(merchant, options)}`

export const getStartCountdownLabel = (merchant, options = {}) => `${getPendingStartLabel(merchant, options)}倒计时`

export const getStartTimeoutLabel = (merchant, target = '客服', options = {}) => `${getStartActionTerm(merchant, options)}超时 重新选择${target}`

export const replaceTerms = (text, merchant) => {
  const t = String(text || '')
  const start = merchant ? getStartTerm(merchant) : getStartTermFromStorage()
  const finish = merchant ? getFinishTerm(merchant) : getFinishTermFromStorage()
  return t
    .replaceAll('起单', start)
    .replaceAll('结单', finish)
}
