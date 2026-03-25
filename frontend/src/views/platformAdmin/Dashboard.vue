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
      <div class="bg-white rounded-xl shadow-sm p-4 border border-gray-100">
        <div class="flex items-start justify-between gap-3">
          <div>
            <div class="text-gray-800 font-medium">系统调度器健康状态</div>
            <div class="mt-2 text-sm text-gray-500">用于检查服务会话、预约、手牌锁卡调度器是否持续产生 tick，仅在平台端展示。</div>
          </div>
          <div class="px-2.5 py-1 rounded-full text-xs font-medium" :class="schedulerHealthBadgeClass">
            {{ schedulerHealthBadgeText }}
          </div>
        </div>

        <div v-if="schedulerHealthError" class="mt-3 text-sm text-red-500">{{ schedulerHealthError }}</div>
        <div v-else-if="!schedulerHealthLoaded" class="mt-3 text-sm text-gray-400">读取调度器状态中...</div>
        <div v-else class="mt-3 space-y-2">
          <div class="text-xs text-gray-400">{{ schedulerHealthSummary }}</div>
          <div
            v-for="item in schedulerHealthItems"
            :key="item.key"
            class="rounded-lg border px-3 py-3"
            :class="getSchedulerItemRowClass(item.status)"
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="text-sm font-medium text-gray-800">{{ item.label }}</div>
                <div class="text-xs text-gray-500 mt-1">{{ item.message }}</div>
              </div>
              <div class="text-xs font-medium" :class="getSchedulerItemTextClass(item.status)">
                {{ getSchedulerItemStatusText(item.status) }}
              </div>
            </div>
            <div class="mt-2 text-xs text-gray-500">
              最近 tick：{{ formatSchedulerTickAt(item.last_tick_at) }}
              <span v-if="item.last_tick_age_sec > 0"> · {{ formatSchedulerAge(item.last_tick_age_sec) }}</span>
            </div>
            <div class="mt-1 text-xs text-gray-400">
              来源：{{ item.source_service || '-' }}<span v-if="item.source_pid"> / PID {{ item.source_pid }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="bg-white rounded-xl shadow-sm p-4">
        <div class="flex items-center justify-between">
          <div class="text-gray-800 font-medium">运营角色（ServiceRole）</div>
          <button type="button" class="px-3 py-2 bg-indigo-50 text-indigo-600 rounded-lg text-sm font-medium" @click="openProfessionalBasePerms">
            配置基础权限
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
import { computed, onMounted, onUnmounted, ref } from 'vue'
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
const schedulerHealth = ref(null)
const schedulerHealthError = ref('')
const schedulerHealthLoaded = ref(false)
let schedulerHealthTimer = null

const schedulerHealthItems = computed(() => Array.isArray(schedulerHealth.value?.items) ? schedulerHealth.value.items : [])

const schedulerHealthBadgeText = computed(() => {
  const status = String(schedulerHealth.value?.overall_status || '').trim()
  if (!status) return '未读取'
  if (status === 'healthy') return '正常'
  if (status === 'degraded') return '异常'
  return '未知'
})

const schedulerHealthBadgeClass = computed(() => {
  const status = String(schedulerHealth.value?.overall_status || '').trim()
  if (status === 'healthy') return 'bg-green-50 text-green-700'
  if (status === 'degraded') return 'bg-red-50 text-red-600'
  return 'bg-gray-100 text-gray-500'
})

const schedulerHealthSummary = computed(() => {
  const generatedAt = schedulerHealth.value?.generated_at ? formatDateTime(schedulerHealth.value.generated_at) : '-'
  const threshold = Number(schedulerHealth.value?.stale_threshold_minutes || 0)
  const message = String(schedulerHealth.value?.overall_message || '').trim() || '未读取调度器状态'
  return `最近检查：${generatedAt} · 阈值：${threshold || '-'} 分钟 · ${message}`
})

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

const formatDateTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const year = String(date.getFullYear()).slice(-2)
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}`
}

const formatSchedulerTickAt = (value) => {
  if (!value) return '暂无'
  return formatDateTime(value)
}

const formatSchedulerAge = (seconds) => {
  const total = Number(seconds || 0)
  if (!Number.isFinite(total) || total <= 0) return '刚刚'
  const mins = Math.floor(total / 60)
  const secs = total % 60
  if (mins > 0) return `${mins}分${secs}秒前`
  return `${secs}秒前`
}

const getSchedulerItemStatusText = (status) => {
  const value = String(status || '').trim()
  if (value === 'healthy') return '正常'
  if (value === 'stale') return '超时'
  if (value === 'missing') return '缺失'
  return value || '未知'
}

const getSchedulerItemRowClass = (status) => {
  const value = String(status || '').trim()
  if (value === 'healthy') return 'border-green-100 bg-green-50/40'
  if (value === 'stale' || value === 'missing') return 'border-red-100 bg-red-50/40'
  return 'border-gray-100 bg-gray-50'
}

const getSchedulerItemTextClass = (status) => {
  const value = String(status || '').trim()
  if (value === 'healthy') return 'text-green-600'
  if (value === 'stale' || value === 'missing') return 'text-red-600'
  return 'text-gray-500'
}

const fetchSchedulerHealth = async (silent = false) => {
  if (!silent) {
    schedulerHealthError.value = ''
  }
  try {
    const res = await platformAdminApi.getSchedulerHealth({})
    schedulerHealth.value = res.data?.data || null
    schedulerHealthLoaded.value = true
    schedulerHealthError.value = ''
  } catch (e) {
    if (!silent) {
      schedulerHealthError.value = e.response?.data?.error || '读取调度器健康状态失败'
    }
    schedulerHealthLoaded.value = true
  }
}

const startSchedulerHealthTimer = () => {
  if (schedulerHealthTimer) return
  schedulerHealthTimer = setInterval(() => {
    fetchSchedulerHealth(true)
  }, 30000)
}

const stopSchedulerHealthTimer = () => {
  if (!schedulerHealthTimer) return
  clearInterval(schedulerHealthTimer)
  schedulerHealthTimer = null
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
    await Promise.all([loadRoles(), loadPerms(), fetchSchedulerHealth()])
    startSchedulerHealthTimer()
  } catch (e) {
    const msg = e.response?.data?.error || ''
    if (msg === '无权限' || msg === '平台管理员未配置') {
      logout()
      return
    }
  }
})

onUnmounted(() => {
  stopSchedulerHealthTimer()
})
</script>
