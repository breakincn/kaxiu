// 开发环境缓存清除工具
if (import.meta.env.DEV) {
  // 清除所有缓存
  const clearCache = async () => {
    if ('caches' in window) {
      const cacheNames = await caches.keys()
      await Promise.all(cacheNames.map(name => caches.delete(name)))
      console.log('开发环境：已清除所有缓存')
    }
    
    // 清除Service Worker注册
    if ('serviceWorker' in navigator) {
      const registrations = await navigator.serviceWorker.getRegistrations()
      await Promise.all(registrations.map(registration => registration.unregister()))
      console.log('开发环境：已注销所有Service Worker')
    }
  }
  
  // 页面加载时自动清除缓存
  window.addEventListener('load', clearCache)
  
  // 暴露到全局供手动调用
  window.clearDevCache = clearCache
}
