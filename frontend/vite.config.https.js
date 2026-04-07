import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// 仅在需要HTTPS时使用此配置
export default defineConfig(({ mode }) => {
  const appTarget = process.env.VITE_APP_TARGET || 'user'
  const isMerchantApp = appTarget === 'merchant'
  const isAdminApp = appTarget === 'admin'
  
  let title
  if (isMerchantApp) {
    title = '卡包 - kabao.shop'
  } else if (isAdminApp) {
    title = '卡包管理 - admin.kabao.app'
  } else {
    title = '卡包 - kabao.app'
  }

  return {
    plugins: [
      vue(),
      {
        name: 'disable-pwa-in-dev',
        transformIndexHtml(html, { server }) {
          if (server && process.env.NODE_ENV === 'development') {
            const removed = html.replace('<link rel="manifest" href="/manifest.webmanifest" />', '')
            return removed.replace(/<title>[\s\S]*?<\/title>/, `<title>${title}</title>`)
          }
          return html.replace(/<title>[\s\S]*?<\/title>/, `<title>${title}</title>`)
        }
      }
    ],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src')
      }
    },
    server: {
      host: '0.0.0.0',
      port: isMerchantApp ? 3001 : (isAdminApp ? 3002 : 3000),
      // 仅在明确需要时启用HTTPS
      https: process.env.VITE_HTTPS === 'true' ? {
        key: './ssl/key.pem',
        cert: './ssl/cert.pem'
      } : false,
      headers: {
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        'Pragma': 'no-cache',
        'Expires': '0'
      },
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:8080',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        }
      }
    },
    build: {
      outDir: isMerchantApp ? 'dist-merchant' : (isAdminApp ? 'dist-admin' : 'dist-user'),
      rollupOptions: {
        output: {
          entryFileNames: 'assets/[name].[hash].js',
          chunkFileNames: 'assets/[name].[hash].js',
          assetFileNames: 'assets/[name].[hash].[ext]',
          manualChunks: {
            vendor: ['vue', 'vue-router', 'axios', 'qrcode'],
            pages: []
          }
        }
      },
      chunkSizeWarningLimit: 1000
    }
  }
})
