<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航 -->
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">我的昵称</span>
    </header>

    <!-- 昵称设置表单 -->
    <div class="px-4 py-6">
      <div class="bg-white rounded-xl shadow-sm p-6">
        <div class="mb-6">
          <label class="block text-sm font-medium text-gray-700 mb-2">
            昵称
          </label>
          <input
            v-model="nickname"
            type="text"
            placeholder="请输入昵称"
            @input="handleNicknameInput"
            class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            maxlength="7"
          />
          <p class="mt-2 text-sm text-gray-500">
            昵称最多 7 个字，且不能包含空格
          </p>
        </div>

        <button
          @click="handleSave"
          :disabled="!canSave"
          class="w-full bg-purple-500 text-white py-3 rounded-lg font-medium hover:bg-purple-600 transition-colors disabled:bg-gray-300 disabled:cursor-not-allowed"
        >
          {{ loading ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../../api'

const router = useRouter()
const nickname = ref('')
const loading = ref(false)
const MAX_NICKNAME_LENGTH = 7

const normalizeNickname = (value) => {
  const noSpaces = String(value || '').replace(/\s+/g, '')
  return Array.from(noSpaces).slice(0, MAX_NICKNAME_LENGTH).join('')
}

const canSave = computed(() => {
  return !loading.value && !!nickname.value.trim()
})

const goBack = () => {
  router.back()
}

const handleNicknameInput = () => {
  nickname.value = normalizeNickname(nickname.value)
}

// 获取当前用户信息
const loadUserInfo = async () => {
  try {
    if (!localStorage.getItem('userToken')) {
      router.push('/login')
      return
    }

    const response = await authApi.getCurrentUser()
    const currentNickname =
      response.data?.data?.nickname ||
      response.data?.data?.username ||
      localStorage.getItem('userName') ||
      ''
    nickname.value = normalizeNickname(currentNickname)
  } catch (error) {
    console.error('获取用户信息失败:', error)
    if (error.response?.status === 401) {
      localStorage.removeItem('userToken')
      router.push('/login')
    }
  }
}

const handleSave = async () => {
  const trimmedNickname = normalizeNickname(nickname.value)
  if (!trimmedNickname) {
    alert('请输入昵称')
    return
  }
  if (/\s/.test(String(nickname.value || ''))) {
    alert('昵称不能包含空格')
    nickname.value = trimmedNickname
    return
  }
  if (Array.from(trimmedNickname).length > MAX_NICKNAME_LENGTH) {
    alert('昵称最多 7 个字')
    nickname.value = normalizeNickname(trimmedNickname)
    return
  }

  loading.value = true
  try {
    if (!localStorage.getItem('userToken')) {
      alert('请先登录')
      router.push('/login')
      return
    }

    const response = await authApi.updateNickname(trimmedNickname)

    if (response.data.data) {
      // 更新localStorage中的用户昵称
      localStorage.setItem('userName', trimmedNickname)
      alert('昵称保存成功')
      router.back()
    }
  } catch (error) {
    console.error('保存昵称失败:', error)
    if (error.response?.status === 401) {
      alert('登录已过期，请重新登录')
      localStorage.removeItem('userToken')
      router.push('/login')
    } else {
      alert(error.response?.data?.error || '保存昵称失败，请稍后重试')
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadUserInfo()
})
</script>
