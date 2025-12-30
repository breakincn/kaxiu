/**
 * 格式化日期时间为 YY-MM-DD HH:mm 格式
 * @param {string|Date} dateStr - 日期字符串或日期对象
 * @returns {string} 格式化后的日期时间字符串，例如：24-12-31 14:30
 */
export function formatDateTime(dateStr) {
  if (!dateStr) return ''
  
  const date = new Date(dateStr)
  
  // 检查日期是否有效
  if (isNaN(date.getTime())) return dateStr
  
  const year = date.getFullYear().toString().slice(-2) // 取后两位年份
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  
  return `${year}-${month}-${day} ${hours}:${minutes}`
}

/**
 * 格式化日期为 YY-MM-DD 格式
 * @param {string|Date} dateStr - 日期字符串或日期对象
 * @returns {string} 格式化后的日期字符串，例如：24-12-31
 */
export function formatDate(dateStr) {
  if (!dateStr) return ''
  
  const date = new Date(dateStr)
  
  // 检查日期是否有效
  if (isNaN(date.getTime())) return dateStr
  
  const year = date.getFullYear().toString().slice(-2) // 取后两位年份
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')
  
  return `${year}-${month}-${day}`
}
