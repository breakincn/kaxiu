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

export const replaceTerms = (text, merchant) => {
  const t = String(text || '')
  const start = merchant ? getStartTerm(merchant) : getStartTermFromStorage()
  const finish = merchant ? getFinishTerm(merchant) : getFinishTermFromStorage()
  return t
    .replaceAll('起单', start)
    .replaceAll('结单', finish)
}
