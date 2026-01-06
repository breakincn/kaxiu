import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  const appTarget = process.env.VITE_APP_TARGET || 'user'
  const isMerchantApp = appTarget === 'merchant'
  const title = isMerchantApp ? '卡包 - kabao.shop' : '卡包 - kabao.app'

  return {
    plugins: [
      vue(),
      // 在开发环境禁用PWA相关功能
      {
        name: 'disable-pwa-in-dev',
        transformIndexHtml(html, { server }) {
          if (server && !process.env.NODE_ENV?.includes('prod')) {
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
      port: isMerchantApp ? 3001 : 3000,
      headers: {
        // 开发环境禁用缓存
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        'Pragma': 'no-cache',
        'Expires': '0'
      },
      proxy: {
        '/api': {
          target: 'http://10.0.0.20:8080',
          changeOrigin: true
        }
      }
    },
    build: {
      outDir: isMerchantApp ? 'dist-merchant' : 'dist-user',
      // 生产构建时确保文件名包含hash以破坏缓存
      rollupOptions: {
        output: {
          entryFileNames: 'assets/[name].[hash].js',
          chunkFileNames: 'assets/[name].[hash].js',
          assetFileNames: 'assets/[name].[hash].[ext]'
        }
      }
    }
  }
})
