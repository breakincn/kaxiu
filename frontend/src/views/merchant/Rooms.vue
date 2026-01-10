<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">房间管理</span>
    </header>

    <div class="px-4 py-6 space-y-4">
      <div v-if="loading" class="text-gray-400 text-center py-10">加载中...</div>

      <div v-else class="space-y-4">
        <div class="bg-white rounded-xl shadow-sm p-4 space-y-3">
          <div class="text-sm font-medium text-gray-700">{{ editingId ? '编辑房间' : '新增房间' }}</div>

          <input
            v-model="form.name"
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm"
            placeholder="房间编号/名称"
          />

          <label class="flex items-center justify-between text-sm text-gray-700">
            <span>启用</span>
            <input type="checkbox" v-model="form.is_active" />
          </label>

          <div class="flex gap-2">
            <button
              class="flex-1 px-4 py-2 rounded-lg text-sm font-medium"
              :class="saving ? 'bg-gray-100 text-gray-400' : 'bg-primary text-white'"
              :disabled="saving"
              @click="save"
            >
              {{ saving ? '保存中...' : (editingId ? '保存' : '新增') }}
            </button>

            <button
              v-if="editingId"
              class="px-4 py-2 rounded-lg text-sm font-medium bg-gray-100 text-gray-700"
              @click="resetForm"
              type="button"
            >
              取消
            </button>
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
            <div class="text-gray-800 font-medium">房间列表</div>
            <button
              class="px-3 py-2 rounded-lg text-sm font-medium"
              :class="refreshing ? 'bg-gray-100 text-gray-400' : 'bg-gray-900 text-white'"
              :disabled="refreshing"
              @click="load"
            >
              刷新
            </button>
          </div>

          <div v-if="rooms.length === 0" class="text-gray-500 text-sm p-4">暂无房间</div>

          <div v-else class="divide-y divide-gray-100">
            <div v-for="r in rooms" :key="r.id" class="px-4 py-4 flex items-start justify-between gap-3">
              <div>
                <div class="text-gray-800 font-medium">{{ r.name }}</div>
                <div class="text-gray-500 text-xs mt-1">{{ r.is_active ? '启用' : '停用' }}</div>
              </div>

              <div class="flex items-center gap-2">
                <button class="px-3 py-2 rounded-lg text-sm bg-gray-100 text-gray-700" @click="edit(r)">编辑</button>
                <button class="px-3 py-2 rounded-lg text-sm bg-red-50 text-red-600" @click="remove(r)">删除</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { roomApi } from '../../api'

const router = useRouter()

const loading = ref(true)
const refreshing = ref(false)
const saving = ref(false)

const rooms = ref([])

const editingId = ref(null)
const form = ref({
  name: '',
  is_active: true
})

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
    setTimeout(() => {
      if (router.currentRoute.value.path === '/merchant/rooms') {
        router.push('/merchant/settings')
      }
    }, 80)
    return
  }
  router.push('/merchant/settings')
}

const resetForm = () => {
  editingId.value = null
  form.value = { name: '', is_active: true }
}

const load = async () => {
  if (refreshing.value) return
  refreshing.value = true
  try {
    const res = await roomApi.listRooms()
    rooms.value = res.data?.data || []
  } catch (e) {
    rooms.value = []
    alert(e.response?.data?.error || '加载失败')
  } finally {
    refreshing.value = false
  }
}

const save = async () => {
  if (saving.value) return
  const name = (form.value.name || '').trim()
  if (!name) {
    alert('请输入房间名称')
    return
  }

  saving.value = true
  try {
    if (editingId.value) {
      await roomApi.updateRoom(editingId.value, { name, is_active: !!form.value.is_active })
    } else {
      await roomApi.createRoom({ name, is_active: !!form.value.is_active })
    }
    resetForm()
    await load()
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    saving.value = false
  }
}

const edit = (r) => {
  editingId.value = r.id
  form.value = {
    name: r.name || '',
    is_active: !!r.is_active
  }
}

const remove = async (r) => {
  if (!r?.id) return
  if (!confirm('确定要删除该房间吗？')) return

  try {
    await roomApi.deleteRoom(r.id)
    await load()
  } catch (e) {
    alert(e.response?.data?.error || '删除失败')
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await load()
  } finally {
    loading.value = false
  }
})
</script>
