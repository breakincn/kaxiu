<template>
  <div v-if="visible" class="fixed inset-0 z-50 flex items-end justify-center bg-black/40" @click.self="dismiss">
    <div class="w-full max-w-lg rounded-t-2xl bg-white px-5 pt-5 pb-4">
      <div class="flex items-start justify-between gap-3">
        <div>
          <div class="text-gray-900 font-medium">为获得最佳体验，请安装到主屏幕后使用</div>
          <div class="text-gray-500 text-sm mt-1">安装后以应用模式打开，可减少浏览器顶部提示条对扫码的遮挡。</div>
        </div>
        <button type="button" class="p-1 text-gray-500" @click="dismiss">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <button
        type="button"
        class="mt-4 w-full rounded-xl bg-gray-50 border border-gray-100 px-4 py-3 text-left"
        @click="expanded = !expanded"
      >
        <div class="flex items-center justify-between">
          <div class="text-gray-800 font-medium">查看安装步骤</div>
          <svg class="w-5 h-5 text-gray-500" :class="expanded ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
          </svg>
        </div>
        <div v-if="expanded" class="mt-3 space-y-4 text-sm">
          <div>
            <div class="text-gray-700 font-medium">Android Chrome</div>
            <div class="mt-2 space-y-1 text-gray-600">
              <div>1. 右上角菜单（⋮）</div>
              <div>2. 选择“安装应用 / 添加到主屏幕”</div>
              <div>3. 从桌面图标打开，再进入扫码页</div>
            </div>
          </div>

          <div>
            <div class="text-gray-700 font-medium">iPhone Safari</div>
            <div class="mt-2 space-y-1 text-gray-600">
              <div>1. 点击底部分享按钮（□↑）</div>
              <div>2. 选择“添加到主屏幕”</div>
              <div>3. 从桌面“卡包”图标打开，再进入扫码页</div>
            </div>
          </div>
        </div>
      </button>

      <div class="mt-4 grid grid-cols-2 gap-3">
        <button type="button" class="rounded-xl bg-gray-100 text-gray-700 py-3 font-medium" @click="dismiss">
          我知道了
        </button>
        <button type="button" class="rounded-xl bg-primary text-white py-3 font-medium" @click="dismissForDay">
          24小时内不再提示
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
  pageKey: {
    type: String,
    required: true
  }
})

const visible = ref(false)
const expanded = ref(false)

const isStandalone = () => {
  try {
    const mm = window.matchMedia && window.matchMedia('(display-mode: standalone)')
    if (mm && mm.matches) return true
  } catch (_) {
    // ignore
  }

  try {
    if (typeof navigator !== 'undefined' && 'standalone' in navigator) {
      return Boolean(navigator.standalone)
    }
  } catch (_) {
    // ignore
  }

  return false
}

const getDismissKey = () => `pwa_install_guide_dismiss_until:${props.pageKey}`

const shouldShow = () => {
  if (isStandalone()) return false
  try {
    const until = parseInt(localStorage.getItem(getDismissKey()) || '0')
    if (until && Date.now() < until) return false
  } catch (_) {
    // ignore
  }
  return true
}

const dismiss = () => {
  visible.value = false
}

const dismissForDay = () => {
  try {
    localStorage.setItem(getDismissKey(), String(Date.now() + 24 * 60 * 60 * 1000))
  } catch (_) {
    // ignore
  }
  visible.value = false
}

onMounted(() => {
  visible.value = shouldShow()
})
</script>
