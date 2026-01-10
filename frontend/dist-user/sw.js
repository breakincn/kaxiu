self.addEventListener('install', (event) => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    self.clients.claim()
  )
})

self.addEventListener('fetch', (event) => {
  // 不拦截API请求，让浏览器直接处理
  if (event.request.url.includes('/api/')) {
    return
  }
  event.respondWith(fetch(event.request))
})
