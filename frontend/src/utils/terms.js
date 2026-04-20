import { getMerchantActiveAuth } from './auth'

const normalizeTerm = (value, fallback) => {
  const s = String(value || '').trim()
  return s || fallback
}

export const getStartTerm = (merchant) => {
  return normalizeTerm(merchant?.start_term, '起单')
}

export const getFinishTerm = (merchant) => {
  return normalizeTerm(merchant?.finish_term, '结单')
}

export const getStartTermFromStorage = () => {
  const active = getMerchantActiveAuth()
  const k = active === 'staff' ? 'technicianStartTerm' : 'merchantStartTerm'
  const storage = active === 'staff' ? sessionStorage : localStorage
  return normalizeTerm(storage.getItem(k), '起单')
}

export const getFinishTermFromStorage = () => {
  const active = getMerchantActiveAuth()
  const k = active === 'staff' ? 'technicianFinishTerm' : 'merchantFinishTerm'
  const storage = active === 'staff' ? sessionStorage : localStorage
  return normalizeTerm(storage.getItem(k), '结单')
}

export const isQueueModeMerchant = (merchant) => {
  if (!merchant) return false
  const queueMode = String(merchant?.queue_mode || '').trim()
  return !merchant?.support_customer_service_mode && !!merchant?.support_queue && (queueMode === 'auto' || queueMode === 'manual')
}

const getStartBaseTerm = (merchant) => (merchant ? getStartTerm(merchant) : getStartTermFromStorage())
const getFinishBaseTerm = (merchant) => (merchant ? getFinishTerm(merchant) : getFinishTermFromStorage())

export const getStartActionTerm = (merchant, options = {}) => {
  if (options.queueMode) return '叫号'
  return getStartBaseTerm(merchant)
}

export const getFinishActionTerm = (merchant, options = {}) => {
  if (options.queueMode) return '结号'
  return getFinishBaseTerm(merchant)
}

export const getPendingStartLabel = (merchant, options = {}) => `待${getStartActionTerm(merchant, options)}`

export const getPendingFinishLabel = (merchant, options = {}) => `待${getFinishActionTerm(merchant, options)}`

export const getAutoFinishLabel = (merchant, options = {}) => `待自动${getFinishActionTerm(merchant, options)}`

export const getServicePendingStartLabel = (merchant, options = {}) => `服务 ${getPendingStartLabel(merchant, options)}`

export const getServicePendingFinishLabel = (merchant, options = {}) => `服务 ${getPendingFinishLabel(merchant, options)}`

export const getStartCountdownLabel = (merchant, options = {}) => `${getPendingStartLabel(merchant, options)}倒计时`

export const getStartTimeoutLabel = (merchant, target = '客服', options = {}) => `${getStartActionTerm(merchant, options)}超时 重新选择${target}`

export const getScanStartLabel = (merchant) => {
  if (isQueueModeMerchant(merchant)) return '扫码上号'
  return `扫码${getStartBaseTerm(merchant)}`
}

export const getStartQrCodeLabel = (merchant) => {
  if (isQueueModeMerchant(merchant)) return '扫码上号二维码'
  return `${getStartBaseTerm(merchant)}二维码`
}

export const getStartCodePromptLabel = (merchant) => {
  if (isQueueModeMerchant(merchant)) return '请扫描上号码'
  return `请扫描${getStartBaseTerm(merchant)}码`
}

export const getStartSuccessLabel = (merchant, sessionId) => {
  if (isQueueModeMerchant(merchant)) return `上号成功！服务单#${sessionId ?? '-'}，已开始计时。`
  return `${getStartBaseTerm(merchant)}成功！服务单#${sessionId ?? '-'}，已开始计时。`
}

export const replaceTerms = (text, merchant) => {
  const t = String(text || '')
  const queueMode = isQueueModeMerchant(merchant)
  const start = queueMode ? '叫号' : getStartBaseTerm(merchant)
  const finish = queueMode ? '结号' : getFinishBaseTerm(merchant)
  return t
    .replaceAll('起单', start)
    .replaceAll('结单', finish)
}
