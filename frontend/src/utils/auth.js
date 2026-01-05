export const getMerchantActiveAuth = () => {
  const v = sessionStorage.getItem('merchantActiveAuth')
  return v === 'technician' ? 'technician' : 'merchant'
}

export const setMerchantActiveAuth = (type) => {
  sessionStorage.setItem('merchantActiveAuth', type === 'technician' ? 'technician' : 'merchant')
}

export const getMerchantToken = () => {
  const active = getMerchantActiveAuth()
  return active === 'technician'
    ? localStorage.getItem('technicianToken')
    : localStorage.getItem('merchantToken')
}

export const getMerchantId = () => {
  const active = getMerchantActiveAuth()
  return active === 'technician'
    ? localStorage.getItem('technicianMerchantId')
    : localStorage.getItem('merchantId')
}

export const getTechnicianShopSlug = () => {
  return sessionStorage.getItem('technicianShopSlug') || ''
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

  if (active === 'technician') {
    localStorage.removeItem('technicianToken')
    localStorage.removeItem('technicianMerchantId')
    localStorage.removeItem('technicianMerchantName')
    localStorage.removeItem('technicianMerchantPhone')

    sessionStorage.removeItem('technicianId')
    sessionStorage.removeItem('technicianName')
    sessionStorage.removeItem('technicianCode')
    sessionStorage.removeItem('technicianAccount')
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
    const raw = sessionStorage.getItem('merchantPermissionKeys')
    if (!raw) return []
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? arr : []
  } catch (e) {
    return []
  }
}

export const setMerchantPermissionKeys = (keys) => {
  const arr = Array.isArray(keys) ? keys : []
  sessionStorage.setItem('merchantPermissionKeys', JSON.stringify(arr))
}

export const clearMerchantPermissionKeys = () => {
  sessionStorage.removeItem('merchantPermissionKeys')
}

export const hasMerchantPermission = (key) => {
  const k = String(key || '').trim()
  if (!k) return false
  const keys = getMerchantPermissionKeys()
  if (keys.includes('*')) return true
  return keys.includes(k)
}
