self.addEventListener('install', (event) => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    self.clients.claim()
  )
})

self.addEventListener('fetch', (event) => {
  if (event.request.method !== 'GET') {
    return
  }

  const requestURL = new URL(event.request.url)
  if (requestURL.origin !== self.location.origin) {
    return
  }

  // 不拦截业务接口请求，让浏览器直接处理
  if (
    requestURL.pathname.startsWith('/api/')
    || requestURL.pathname.startsWith('/user/')
    || requestURL.pathname.startsWith('/merchant/')
    || requestURL.pathname.startsWith('/platform-admin/')
    || requestURL.pathname.startsWith('/admin/')
  ) {
    return
  }

  event.respondWith(fetch(event.request))
})
