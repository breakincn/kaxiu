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
