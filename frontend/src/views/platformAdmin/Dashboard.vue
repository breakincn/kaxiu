<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <span class="font-medium text-gray-800">平台后台</span>
      <div class="flex-1"></div>
      <button type="button" class="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium" @click="logout">
        退出
      </button>
    </header>

    <div class="px-4 py-4 space-y-4">
      <div class="bg-white rounded-xl shadow-sm p-4">
        <div class="flex items-center justify-between">
          <div class="text-gray-800 font-medium">固定角色（ServiceRole）</div>
          <button type="button" class="px-3 py-2 bg-indigo-50 text-indigo-600 rounded-lg text-sm font-medium" @click="openProfessionalBasePerms">
            配置专业客服基础默认权限
          </button>
        </div>
        <div class="mt-2 text-sm text-gray-500">平台后台仅保留固定角色默认权限配置，不再新增、编辑或删除客服角色。</div>

        <div v-if="loadingRoles" class="text-center text-gray-400 py-10">加载中...</div>
        <div v-else>
          <div v-if="operationalRoles.length === 0" class="text-center text-gray-400 py-10">暂无固定角色</div>
          <div v-else class="mt-4 space-y-3">
            <div v-for="r in operationalRoles" :key="r.id" class="border border-gray-100 rounded-xl p-4">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <div class="flex items-center gap-2">
                    <div class="text-gray-800 font-medium">{{ r.name }}</div>
                    <span class="px-2 py-0.5 rounded text-xs" :class="r.is_active ? 'bg-green-50 text-green-600' : 'bg-gray-100 text-gray-500'">
                      {{ r.is_active ? '启用' : '禁用' }}
                    </span>
                    <span v-if="r.allow_permission_adjust" class="px-2 py-0.5 rounded text-xs bg-blue-50 text-blue-600">可微调</span>
                  </div>
                  <div class="text-gray-500 text-sm mt-1">key：{{ r.key }}　sort：{{ r.sort }}</div>
                  <div v-if="r.description" class="text-gray-400 text-sm mt-1">{{ r.description }}</div>
                </div>
                <div class="text-gray-400 text-xs">ID: {{ r.id }}</div>
              </div>
              <div class="mt-3 flex gap-2">
                <button type="button" class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium" @click="openRolePerms(r)">
                  配置默认权限
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="bg-white rounded-xl shadow-sm p-4">
        <div class="flex items-center justify-between">
          <div class="text-gray-800 font-medium">权限枚举（Permission）</div>
          <button type="button" class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium" @click="openCreatePerm">
            新增
          </button>
        </div>

        <div v-if="loadingPerms" class="text-center text-gray-400 py-10">加载中...</div>
        <div v-else>
          <div v-if="perms.length === 0" class="text-center text-gray-400 py-10">暂无权限</div>
          <div v-else class="mt-4 space-y-3">
            <div v-for="p in perms" :key="p.id" class="border border-gray-100 rounded-xl p-4">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <div class="text-gray-800 font-medium">{{ p.name }}</div>
                  <div class="text-gray-500 text-sm mt-1">key：{{ p.key }}　group：{{ p.group || '-' }}　sort：{{ p.sort }}</div>
                  <div v-if="p.description" class="text-gray-400 text-sm mt-1">{{ p.description }}</div>
                </div>
                <div class="text-gray-400 text-xs">ID: {{ p.id }}</div>
              </div>
              <div class="mt-3 flex gap-2">
                <button type="button" class="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium" @click="openEditPerm(p)">编辑</button>
                <div class="flex-1"></div>
                <button type="button" class="px-3 py-2 bg-red-50 text-red-600 rounded-lg text-sm font-medium" @click="deletePerm(p)">删除</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 权限弹窗 -->
    <div v-if="showPermModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closePermModal">
      <div class="bg-white rounded-2xl w-full max-w-md overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">{{ permForm.id ? '编辑权限' : '新增权限' }}</div>
          <button type="button" class="text-gray-400" @click="closePermModal">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="px-5 py-5">
          <div class="mb-4" v-if="!permForm.id">
            <label class="block text-gray-700 text-sm font-medium mb-2">key</label>
            <input v-model="permForm.key" type="text" placeholder="如 merchant.card.verify" class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary" />
          </div>
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">name</label>
            <input v-model="permForm.name" type="text" placeholder="如 核销" class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary" />
          </div>
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">group</label>
            <input v-model="permForm.group" type="text" placeholder="如 卡片" class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary" />
          </div>
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">description</label>
            <input v-model="permForm.description" type="text" placeholder="描述" class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary" />
          </div>
          <div class="mb-5">
            <label class="block text-gray-700 text-sm font-medium mb-2">sort</label>
            <input v-model.number="permForm.sort" type="number" class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary" />
          </div>
          <button type="button" class="w-full py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50" :disabled="saving" @click="savePerm">
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 角色默认权限弹窗 -->
    <div v-if="showRolePermModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closeRolePermModal">
      <div class="bg-white rounded-2xl w-full max-w-2xl overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">配置默认权限：{{ rolePermRole?.name }}</div>
          <button type="button" class="text-gray-400" @click="closeRolePermModal">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="px-5 py-5">
          <div v-if="loadingRolePerms" class="text-center text-gray-400 py-10">加载中...</div>
          <div v-else>
            <div v-if="rolePermItems.length === 0" class="text-center text-gray-400 py-10">暂无权限</div>
            <div v-else class="max-h-[60vh] overflow-y-auto border border-gray-100 rounded-xl">
              <div v-for="it in rolePermItems" :key="it.permission.id" class="flex items-center justify-between px-4 py-3 border-b border-gray-100">
                <div>
                  <div class="text-gray-800 text-sm font-medium">{{ it.permission.name }}</div>
                  <div class="text-gray-500 text-xs">{{ it.permission.key }}</div>
                </div>
                <label class="flex items-center gap-2 text-sm text-gray-700">
                  <input type="checkbox" v-model="it.allowed" :disabled="!!it.is_base" />允许
                </label>
              </div>
            </div>
            <button type="button" class="mt-4 w-full py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50" :disabled="saving" @click="saveRolePerms">
              {{ saving ? '保存中...' : '保存' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 专业客服基础默认权限弹窗 -->
    <div v-if="showProfessionalBasePermModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closeProfessionalBasePermModal">
      <div class="bg-white rounded-2xl w-full max-w-2xl overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">配置专业客服基础默认权限</div>
          <button type="button" class="text-gray-400" @click="closeProfessionalBasePermModal">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div class="px-5 py-5">
          <div v-if="loadingProfessionalBasePerms" class="text-center text-gray-400 py-10">加载中...</div>
          <div v-else>
            <div v-if="professionalBasePermItems.length === 0" class="text-center text-gray-400 py-10">暂无权限</div>
            <div v-else class="max-h-[60vh] overflow-y-auto border border-gray-100 rounded-xl">
              <div v-for="it in professionalBasePermItems" :key="it.permission.id" class="flex items-center justify-between px-4 py-3 border-b border-gray-100">
                <div>
                  <div class="text-gray-800 text-sm font-medium">{{ it.permission.name }}</div>
                  <div class="text-gray-500 text-xs">{{ it.permission.key }}</div>
                </div>
                <label class="flex items-center gap-2 text-sm text-gray-700">
                  <input type="checkbox" v-model="it.allowed" />允许
                </label>
              </div>
            </div>
            <button type="button" class="mt-4 w-full py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50" :disabled="saving" @click="saveProfessionalBasePerms">
              {{ saving ? '保存中...' : '保存' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { platformAdminApi } from '../../api'

const router = useRouter()

const roles = ref([])
const perms = ref([])

const loadingRoles = ref(false)
const loadingPerms = ref(false)
const saving = ref(false)

const showPermModal = ref(false)
const permForm = ref({ id: 0, key: '', name: '', group: '', description: '', sort: 0 })

const showRolePermModal = ref(false)
const rolePermRole = ref(null)
const rolePermItems = ref([])
const loadingRolePerms = ref(false)

const operationalRoles = ref([])

const showProfessionalBasePermModal = ref(false)
const professionalBasePermItems = ref([])
const loadingProfessionalBasePerms = ref(false)

const logout = () => {
  localStorage.removeItem('platformAdminToken')
  router.replace('/platform-admin/login')
}

const ensureToken = () => {
  const t = localStorage.getItem('platformAdminToken')
  if (!t) {
    router.replace('/platform-admin/login')
    return false
  }
  return true
}

const loadRoles = async () => {
  loadingRoles.value = true
  try {
    const res = await platformAdminApi.listServiceRoles()
    roles.value = res.data?.data || []
    operationalRoles.value = (roles.value || []).filter((r) => (r.role_type || '').trim() === 'operational')
  } finally {
    loadingRoles.value = false
  }
}

const openProfessionalBasePerms = async () => {
  showProfessionalBasePermModal.value = true
  loadingProfessionalBasePerms.value = true
  try {
    const res = await platformAdminApi.getProfessionalBasePermissions()
    professionalBasePermItems.value = res.data?.data?.items || []
  } catch (e) {
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loadingProfessionalBasePerms.value = false
  }
}

const closeProfessionalBasePermModal = () => {
  showProfessionalBasePermModal.value = false
}

const saveProfessionalBasePerms = async () => {
  if (saving.value) return
  saving.value = true
  try {
    await platformAdminApi.setProfessionalBasePermissions({
      items: (professionalBasePermItems.value || []).map((it) => ({
        permission_key: it.permission?.key,
        allowed: !!it.allowed
      }))
    })
    closeProfessionalBasePermModal()
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

const loadPerms = async () => {
  loadingPerms.value = true
  try {
    const res = await platformAdminApi.listPermissions()
    perms.value = res.data?.data || []
  } finally {
    loadingPerms.value = false
  }
}

const openCreatePerm = () => {
  permForm.value = { id: 0, key: '', name: '', group: '', description: '', sort: 0 }
  showPermModal.value = true
}

const openEditPerm = (p) => {
  permForm.value = { id: p.id, key: p.key, name: p.name, group: p.group || '', description: p.description || '', sort: p.sort || 0 }
  showPermModal.value = true
}

const closePermModal = () => {
  showPermModal.value = false
}

const savePerm = async () => {
  if (saving.value) return
  if (!permForm.value.name) {
    alert('请输入 name')
    return
  }
  if (!permForm.value.id && !permForm.value.key) {
    alert('请输入 key')
    return
  }
  saving.value = true
  try {
    if (permForm.value.id) {
      await platformAdminApi.updatePermission(permForm.value.id, {
        name: permForm.value.name,
        group: permForm.value.group,
        description: permForm.value.description,
        sort: permForm.value.sort
      })
    } else {
      await platformAdminApi.createPermission({
        key: permForm.value.key,
        name: permForm.value.name,
        group: permForm.value.group,
        description: permForm.value.description,
        sort: permForm.value.sort
      })
    }
    closePermModal()
    await loadPerms()
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

const deletePerm = async (p) => {
  if (!confirm('确定要删除该权限吗？')) return
  if (saving.value) return
  saving.value = true
  try {
    await platformAdminApi.deletePermission(p.id)
    await loadPerms()
  } catch (e) {
    alert(e.response?.data?.error || '删除失败')
  } finally {
    saving.value = false
  }
}

const openRolePerms = async (r) => {
  showRolePermModal.value = true
  rolePermRole.value = r
  rolePermItems.value = []
  loadingRolePerms.value = true
  try {
    const res = await platformAdminApi.getRolePermissions(r.id)
    const items = res.data?.data?.items || []
    for (const it of items) {
      if (it && it.is_base) {
        it.allowed = true
      }
    }
    rolePermItems.value = items
  } catch (e) {
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loadingRolePerms.value = false
  }
}

const closeRolePermModal = () => {
  showRolePermModal.value = false
  rolePermRole.value = null
  rolePermItems.value = []
}

const saveRolePerms = async () => {
  if (!rolePermRole.value) return
  if (saving.value) return
  saving.value = true
  try {
    const items = rolePermItems.value.map((it) => ({
      permission_key: it.permission.key,
      allowed: it && it.is_base ? true : !!it.allowed
    }))
    await platformAdminApi.setRolePermissions(rolePermRole.value.id, { items })
    closeRolePermModal()
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  if (!ensureToken()) return
  try {
    await Promise.all([loadRoles(), loadPerms()])
  } catch (e) {
    const msg = e.response?.data?.error || ''
    if (msg === '无权限' || msg === '平台管理员未配置') {
      logout()
      return
    }
  }
})
</script>
