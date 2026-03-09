import { ref } from 'vue'

const merchantPermissionVersion = ref(0)

export const getMerchantActiveAuth = () => {
  const v = sessionStorage.getItem('merchantActiveAuth')
  return v === 'staff' ? 'staff' : 'merchant'
}

export const setMerchantActiveAuth = (type) => {
  sessionStorage.setItem('merchantActiveAuth', type === 'staff' ? 'staff' : 'merchant')
}

export const getMerchantToken = () => {
  const active = getMerchantActiveAuth()
  return active === 'staff'
    ? localStorage.getItem('technicianToken')
    : localStorage.getItem('merchantToken')
}

export const getMerchantId = () => {
  const active = getMerchantActiveAuth()
  return active === 'staff'
    ? localStorage.getItem('technicianMerchantId')
    : localStorage.getItem('merchantId')
}

const getMerchantPermissionKeysStorageKey = () => {
  const active = getMerchantActiveAuth()
  const merchantId = getMerchantId() || ''
  const staffId = active === 'staff' ? (sessionStorage.getItem('technicianId') || '') : ''
  return `merchantPermissionKeys:${active}:${merchantId}:${staffId}`
}

export const getTechnicianShopSlug = () => {
  return sessionStorage.getItem('technicianShopSlug') || ''
}

export const getTechnicianPasswordNeedReset = () => {
  return sessionStorage.getItem('technicianPasswordNeedReset') === '1'
}

export const setTechnicianPasswordNeedReset = (needReset) => {
  if (needReset) {
    sessionStorage.setItem('technicianPasswordNeedReset', '1')
    return
  }
  sessionStorage.removeItem('technicianPasswordNeedReset')
}

export const setTechnicianShopSlug = (slug) => {
  if (slug) {
    sessionStorage.setItem('technicianShopSlug', String(slug))
  } else {
    sessionStorage.removeItem('technicianShopSlug')
  }
}

export const clearMerchantAuth = () => {
  const active = getMerchantActiveAuth()

  if (active === 'staff') {
    localStorage.removeItem('technicianToken')
    localStorage.removeItem('technicianMerchantId')
    localStorage.removeItem('technicianMerchantName')
    localStorage.removeItem('technicianMerchantPhone')

    sessionStorage.removeItem('technicianId')
    sessionStorage.removeItem('technicianName')
    sessionStorage.removeItem('technicianCode')
    sessionStorage.removeItem('technicianAccount')
    sessionStorage.removeItem('technicianPasswordNeedReset')
    sessionStorage.removeItem('technicianShopSlug')
    sessionStorage.removeItem('merchantActiveAuth')
    return
  }

  localStorage.removeItem('merchantToken')
  localStorage.removeItem('merchantId')
  localStorage.removeItem('merchantName')
  localStorage.removeItem('merchantPhone')
  sessionStorage.removeItem('merchantActiveAuth')
}

export const getMerchantPermissionKeys = () => {
  try {
    const raw = sessionStorage.getItem(getMerchantPermissionKeysStorageKey()) || sessionStorage.getItem('merchantPermissionKeys')
    if (!raw) return []
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? arr : []
  } catch (e) {
    return []
  }
}

export const setMerchantPermissionKeys = (keys) => {
  const arr = Array.isArray(keys) ? keys : []
  sessionStorage.setItem(getMerchantPermissionKeysStorageKey(), JSON.stringify(arr))
  sessionStorage.removeItem('merchantPermissionKeys')
  merchantPermissionVersion.value++
}

export const clearMerchantPermissionKeys = () => {
  sessionStorage.removeItem(getMerchantPermissionKeysStorageKey())
  sessionStorage.removeItem('merchantPermissionKeys')
  merchantPermissionVersion.value++
}

export const getTechnicianId = () => {
  const raw = sessionStorage.getItem('technicianId')
  if (!raw) return null
  const n = parseInt(String(raw), 10)
  return Number.isFinite(n) && n > 0 ? n : null
}

export const isTechnicianAuth = () => {
  return getMerchantActiveAuth() === 'staff'
}

export const hasMerchantPermission = (key) => {
  void merchantPermissionVersion.value
  const k = String(key || '').trim()
  if (!k) return false
  const keys = getMerchantPermissionKeys()
  if (keys.includes('*')) return true
  return keys.includes(k)
}
