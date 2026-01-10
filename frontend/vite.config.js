import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

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
      // 在开发环境禁用PWA相关功能
      {
        name: 'disable-pwa-in-dev',
        transformIndexHtml(html, { server }) {
          if (server && process.env.NODE_ENV === 'development') {
            // 移除manifest引用以禁用PWA
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
      headers: {
        // 开发环境禁用缓存
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        'Pragma': 'no-cache',
        'Expires': '0'
      },
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:8080',
          changeOrigin: true
        }
      }
    },
    build: {
      outDir: isMerchantApp ? 'dist-merchant' : (isAdminApp ? 'dist-admin' : 'dist-user'),
      // 减少代码分割
      rollupOptions: {
        output: {
          entryFileNames: 'assets/[name].[hash].js',
          chunkFileNames: 'assets/[name].[hash].js',
          assetFileNames: 'assets/[name].[hash].[ext]',
          // 手动合并 chunks，减少文件数量
          manualChunks: {
            // 将所有第三方库合并到一个文件
            vendor: ['vue', 'vue-router', 'axios', 'qrcode'],
            // 将所有页面组件合并到一个文件
            pages: []
          }
        }
      },
      // 设置最小 chunk 大小阈值，避免生成过小的文件
      chunkSizeWarningLimit: 1000
    }
  }
})
