<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">设置客服</span>
      <div class="flex-1"></div>
    </header>

    <div class="px-4 py-4">
      <div class="bg-white rounded-xl shadow-sm p-4">
        <div class="text-gray-800 font-medium">客服类型</div>

        <div class="mt-3 flex flex-wrap gap-2">
          <button
            v-for="r in roles"
            :key="r.key"
            type="button"
            class="px-3 py-2 rounded-lg text-sm font-medium border"
            :class="activeRole === r.key ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectRole(r.key)"
          >
            {{ r.name }}
          </button>
        </div>

        <div v-if="roles.length === 0" class="mt-3 text-sm text-gray-500">
          暂无可用客服类型
        </div>

        <div class="mt-4 border-t border-gray-100 pt-4">
          <div v-if="activeRole === 'technician'" class="flex items-center justify-between">
            <div class="text-gray-800 font-medium">技师账号</div>
            <div class="flex items-center gap-2">
              <button
                v-if="activeRoleObj && activeRoleObj.allow_permission_adjust"
                type="button"
                class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                @click="openPermissionAdjust"
              >
                权限微调
              </button>
              <button
                type="button"
                class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                @click="openCreate"
              >
                添加技师
              </button>
            </div>
          </div>

          <div v-else class="flex items-center justify-between">
            <div class="text-gray-500 text-sm">该客服类型功能开发中</div>
            <button
              v-if="activeRoleObj && activeRoleObj.allow_permission_adjust"
              type="button"
              class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
              @click="openPermissionAdjust"
            >
              权限微调
            </button>
          </div>
        </div>

        <div v-if="activeRole === 'technician'">
          <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>

          <div v-else>
            <div v-if="techs.length === 0" class="text-center text-gray-400 py-10">暂无技师</div>

            <div v-else class="mt-4 space-y-3">
              <div
                v-for="t in techs"
                :key="t.id"
                class="border border-gray-100 rounded-xl p-4"
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <div class="flex items-center gap-2">
                      <div class="text-gray-800 font-medium">{{ t.name }}</div>
                      <span
                        class="px-2 py-0.5 rounded text-xs"
                        :class="t.is_active ? 'bg-green-50 text-green-600' : 'bg-gray-100 text-gray-500'"
                      >
                        {{ t.is_active ? '启用' : '禁用' }}
                      </span>
                    </div>
                    <div class="text-gray-500 text-sm mt-1">编号：{{ t.code }}　账号：{{ t.account }}</div>
                  </div>
                  <div class="text-gray-400 text-xs">ID: {{ t.id }}</div>
                </div>

                <div class="mt-3 flex gap-2">
                  <button
                    type="button"
                    class="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium"
                    @click="openEdit(t)"
                  >
                    编辑
                  </button>
                  <button
                    type="button"
                    class="px-3 py-2 rounded-lg text-sm font-medium"
                    :class="t.is_active ? 'bg-orange-50 text-orange-600' : 'bg-green-50 text-green-600'"
                    @click="toggleActive(t)"
                  >
                    {{ t.is_active ? '禁用' : '启用' }}
                  </button>
                  <div class="flex-1"></div>
                  <button
                    type="button"
                    class="px-3 py-2 bg-red-50 text-red-600 rounded-lg text-sm font-medium"
                    @click="removeTech(t)"
                  >
                    删除
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showAdd" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closeAdd">
      <div class="bg-white rounded-2xl w-full max-w-md overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">{{ isEdit ? '编辑技师' : '添加技师' }}</div>
          <button type="button" class="text-gray-400" @click="closeAdd">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5">
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">技师姓名</label>
            <input
              v-model="form.name"
              type="text"
              placeholder="如：技师1"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <div v-if="!isEdit" class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">技师编号</label>
            <input
              v-model="form.code"
              type="text"
              placeholder="如：0001"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <div v-if="!isEdit" class="text-gray-500 text-sm mb-5">
            默认账号：<span class="text-gray-800 font-medium">js{{ form.code || 'xxxx' }}</span>
            <br />
            默认密码：<span class="text-gray-800 font-medium">{{ (form.code || 'xxxx') + '12345' }}</span>
          </div>

          <button
            type="button"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
            :disabled="saving"
            @click="submit"
          >
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi, platformApi } from '../../api'

const router = useRouter()

const loading = ref(false)
const saving = ref(false)
const techs = ref([])

const roles = ref([])

const activeRole = ref('')

const activeRoleObj = computed(() => {
  const k = activeRole.value
  if (!k) return null
  return roles.value.find((r) => r && r.key === k) || null
})

const showAdd = ref(false)
const isEdit = ref(false)
const form = ref({
  id: 0,
  name: '',
  code: ''
})

const goBack = () => {
  router.back()
}

const openPermissionAdjust = () => {
  if (!activeRole.value) return
  router.push(`/merchant/role-permissions/${activeRole.value}`)
}

const load = async () => {
  loading.value = true
  try {
    const res = await merchantApi.getTechnicians()
    techs.value = res.data.data || []
  } catch (e) {
    techs.value = []
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const closeAdd = () => {
  showAdd.value = false
  isEdit.value = false
  form.value = { id: 0, name: '', code: '' }
}

const openCreate = () => {
  isEdit.value = false
  form.value = { id: 0, name: '', code: '' }
  showAdd.value = true
}

const openEdit = (t) => {
  if (!t || !t.id) return
  isEdit.value = true
  form.value = {
    id: t.id,
    name: t.name || '',
    code: t.code || ''
  }
  showAdd.value = true
}

const submit = async () => {
  if (saving.value) return
  if (!form.value.name) {
    alert('请输入技师姓名')
    return
  }
  if (!isEdit.value && !form.value.code) {
    alert('请输入技师编号')
    return
  }

  saving.value = true
  try {
    if (isEdit.value) {
      await merchantApi.updateTechnician(form.value.id, { name: form.value.name })
      alert('更新成功')
    } else {
      const res = await merchantApi.createTechnician({
        name: form.value.name,
        code: form.value.code
      })
      const pwd = res?.data?.data?.default_password
      if (pwd) {
        alert(`创建成功！默认密码：${pwd}`)
      } else {
        alert('创建成功')
      }
    }
    closeAdd()
    await load()
  } catch (e) {
    alert(e.response?.data?.error || (isEdit.value ? '更新失败' : '创建失败'))
  } finally {
    saving.value = false
  }
}

const toggleActive = async (t) => {
  if (!t || !t.id) return
  if (saving.value) return
  const next = !t.is_active
  saving.value = true
  try {
    await merchantApi.updateTechnician(t.id, { is_active: next })
    await load()
  } catch (e) {
    alert(e.response?.data?.error || '操作失败')
  } finally {
    saving.value = false
  }
}

const removeTech = async (t) => {
  if (!t || !t.id) return
  if (!confirm('确定要删除该技师吗？')) return
  if (saving.value) return
  saving.value = true
  try {
    await merchantApi.deleteTechnician(t.id)
    await load()
  } catch (e) {
    alert(e.response?.data?.error || '删除失败')
  } finally {
    saving.value = false
  }
}

const selectRole = async (key) => {
  activeRole.value = key
  closeAdd()
  if (key === 'technician') {
    await load()
  }
}

onMounted(async () => {
  try {
    const res = await platformApi.getServiceRoles()
    roles.value = res.data?.data || []
  } catch (e) {
    roles.value = []
  }

  const hasTechnician = roles.value.some((r) => r && r.key === 'technician')
  if (hasTechnician) {
    activeRole.value = 'technician'
    await load()
    return
  }

  const first = roles.value.find((r) => r && r.key)
  activeRole.value = first ? String(first.key) : ''
})
</script>
