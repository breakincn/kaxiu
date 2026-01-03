const CACHE_NAME = 'kabao-static-v1'

self.addEventListener('install', (event) => {
  self.skipWaiting()
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll([
        '/',
        '/index.html',
        '/manifest.webmanifest',
        '/favicon.svg'
      ])
    })
  )
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    Promise.all([
      self.clients.claim(),
      caches.keys().then((keys) =>
        Promise.all(keys.map((k) => (k === CACHE_NAME ? null : caches.delete(k))))
      )
    ])
  )
})

self.addEventListener('fetch', (event) => {
  const req = event.request
  if (req.method !== 'GET') return

  event.respondWith(
    caches.match(req).then((cached) => {
      if (cached) return cached
      return fetch(req)
        .then((resp) => {
          const url = new URL(req.url)
          if (url.origin === self.location.origin && resp && resp.status === 200) {
            const copy = resp.clone()
            caches.open(CACHE_NAME).then((cache) => cache.put(req, copy))
          }
          return resp
        })
        .catch(() => cached)
    })
  )
})
