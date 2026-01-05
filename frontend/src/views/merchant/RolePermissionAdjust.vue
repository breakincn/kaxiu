<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">权限微调</span>
      <div class="flex-1"></div>
      <button type="button" class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium disabled:opacity-50" :disabled="saving" @click="save">
        {{ saving ? '保存中...' : '保存' }}
      </button>
    </header>

    <div class="px-4 py-4">
      <div class="bg-white rounded-xl shadow-sm p-4">
        <div class="text-gray-800 font-medium">{{ role?.name || '-' }}</div>
        <div class="text-gray-500 text-sm mt-1">key：{{ role?.key || '-' }}</div>
      </div>

      <div class="bg-white rounded-xl shadow-sm p-4 mt-4">
        <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>
        <div v-else>
          <div v-if="items.length === 0" class="text-center text-gray-400 py-10">暂无权限项</div>
          <div v-else class="space-y-2">
            <div v-for="it in items" :key="it.permission.id" class="flex items-center justify-between border border-gray-100 rounded-xl px-4 py-3">
              <div class="pr-3">
                <div class="text-gray-800 text-sm font-medium">{{ it.permission.name }}</div>
                <div class="text-gray-500 text-xs mt-0.5">{{ it.permission.key }}</div>
              </div>
              <label class="flex items-center gap-2 text-sm text-gray-700 shrink-0">
                <input type="checkbox" v-model="it.override_allowed" />
                <span>{{ it.override_allowed ? '允许' : '禁止' }}</span>
              </label>
            </div>
          </div>
        </div>
      </div>

      <div class="text-gray-400 text-xs mt-3 px-1">
        默认权限由平台配置；此处仅保存“覆盖值”。
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { merchantApi } from '../../api'

const router = useRouter()
const route = useRoute()

const roleKey = ref(String(route.params.roleKey || ''))

const loading = ref(false)
const saving = ref(false)

const role = ref(null)
const items = ref([])

const goBack = () => {
  router.back()
}

const load = async () => {
  loading.value = true
  try {
    const res = await merchantApi.getRolePermissions(roleKey.value)
    role.value = res.data?.data?.role || null
    const raw = res.data?.data?.items || []
    items.value = raw.map((it) => ({
      permission: it.permission,
      default_allowed: !!it.default_allowed,
      override_allowed: typeof it.override_allowed === 'boolean' ? it.override_allowed : it.effective_allowed
    }))
  } catch (e) {
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (saving.value) return
  saving.value = true
  try {
    const payload = {
      items: items.value.map((it) => ({ permission_key: it.permission.key, allowed: !!it.override_allowed }))
    }
    await merchantApi.setRolePermissions(roleKey.value, payload)
    alert('保存成功')
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
