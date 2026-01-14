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
            type="button"
            class="px-3 py-2 rounded-lg text-sm font-medium border"
            :class="activeType === 'operational' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectType('operational')"
          >
            运营客服
          </button>
          <button
            type="button"
            class="px-3 py-2 rounded-lg text-sm font-medium border"
            :class="activeType === 'professional' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
            @click="selectType('professional')"
          >
            专业客服
          </button>
        </div>

        <div v-if="allRoles.length === 0" class="mt-3 text-sm text-gray-500">暂无可用客服类型</div>

        <div class="mt-4 border-t border-gray-100 pt-4">
          <div class="flex items-center justify-between">
            <div class="text-gray-800 font-medium">
              <span v-if="activeType === 'operational'">运营客服账号</span>
              <span v-else>专业客服账号</span>
            </div>
            <div class="flex items-center gap-2">
              <template v-if="activeType === 'operational'">
                <button
                  v-if="selectedOperationalRoleObj && selectedOperationalRoleObj.allow_permission_adjust"
                  type="button"
                  class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                  @click="openPermissionAdjustOperational"
                >
                  权限微调
                </button>
                <button
                  type="button"
                  class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                  @click="openCreateOperational"
                >
                  添加{{ selectedOperationalRoleObj?.name || '运营客服' }}
                </button>
              </template>
              <template v-else>
                <button
                  type="button"
                  class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                  @click="openCreateProfessionalRole"
                >
                  添加客服
                </button>
              </template>
            </div>
          </div>
        </div>

        <div>
          <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>

          <div v-else>
            <div v-if="activeType === 'operational'" class="mt-4">
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="r in operationalRoles"
                  :key="r.key"
                  type="button"
                  class="px-3 py-2 rounded-lg text-sm font-medium border"
                  :class="selectedOperationalRole === r.key ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
                  @click="selectOperationalRole(r.key)"
                >
                  {{ r.name }}
                </button>
              </div>

              <div v-if="visibleTechs.length === 0" class="text-center text-gray-400 py-10">暂无{{ selectedOperationalRoleObj?.name || '运营客服' }}</div>

              <div v-else class="mt-4 space-y-3">
                <div v-for="t in visibleTechs" :key="t.id" class="border border-gray-100 rounded-xl p-4">
                  <div class="flex items-start justify-between gap-3">
                    <div>
                      <div class="flex items-center gap-2">
                        <div class="text-gray-800 font-medium">{{ t.name }}</div>
                        <span class="px-2 py-0.5 rounded text-xs" :class="t.is_active ? 'bg-green-50 text-green-600' : 'bg-gray-100 text-gray-500'">
                          {{ t.is_active ? '启用' : '禁用' }}
                        </span>
                      </div>
                      <div class="text-gray-500 text-sm mt-1">编号：{{ t.code }}　账号：{{ selectedOperationalRoleObj?.name }}: {{ t.account }}</div>
                    </div>
                    <div class="text-gray-400 text-xs">ID: {{ t.id }}</div>
                  </div>

                  <div class="mt-3 flex gap-2">
                    <button type="button" class="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium" @click="openEdit(t)">编辑</button>
                    <button
                      type="button"
                      class="px-3 py-2 rounded-lg text-sm font-medium"
                      :class="t.is_active ? 'bg-orange-50 text-orange-600' : 'bg-green-50 text-green-600'"
                      @click="toggleActive(t)"
                    >
                      {{ t.is_active ? '禁用' : '启用' }}
                    </button>
                    <div class="flex-1"></div>
                    <button type="button" class="px-3 py-2 bg-red-50 text-red-600 rounded-lg text-sm font-medium" @click="removeTech(t)">删除</button>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="mt-4">
              <div class="mb-3">
                <label class="block text-gray-700 text-sm font-medium mb-2">新增岗位（称谓）</label>
                <div class="flex items-center gap-2">
                  <select v-model="selectedProfessionalRole" class="flex-1 px-4 py-3 border border-gray-200 rounded-lg bg-white">
                    <option v-for="r in professionalRoles" :key="r.key" :value="r.key">{{ r.name }}</option>
                  </select>
                  <button
                    type="button"
                    class="px-4 py-3 bg-primary text-white rounded-lg text-sm font-medium shrink-0"
                    :disabled="!selectedProfessionalRole"
                    @click="openCreateProfessionalStaff"
                  >
                    添加
                  </button>
                </div>
              </div>

              <div v-if="Object.keys(professionalTechsByRole).length === 0" class="text-center text-gray-400 py-10">暂无专业客服</div>

              <div v-else class="mt-4 space-y-6">
                <div v-for="(group, roleName) in professionalTechsByRole" :key="roleName" class="bg-gray-50 rounded-xl p-4">
                  <div class="flex items-center justify-between mb-3">
                    <div class="flex items-center gap-2">
                      <div class="text-gray-800 font-medium">{{ roleName }}</div>
                      <span class="px-2 py-0.5 bg-blue-50 text-blue-600 rounded text-xs">{{ group.techs.length }}人</span>
                    </div>
                    <button
                      v-if="group.role && group.role.allow_permission_adjust"
                      type="button"
                      class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                      @click="openPermissionAdjustProfessional(group.role.key)"
                    >
                      权限微调
                    </button>
                  </div>
                  
                  <div class="space-y-3">
                    <div v-for="t in group.techs" :key="t.id" class="bg-white border border-gray-100 rounded-xl p-4">
                      <div class="flex items-start justify-between gap-3">
                        <div>
                          <div class="flex items-center gap-2">
                            <div class="text-gray-800 font-medium">{{ t.name }}</div>
                            <span class="px-2 py-0.5 rounded text-xs" :class="t.is_active ? 'bg-green-50 text-green-600' : 'bg-gray-100 text-gray-500'">
                              {{ t.is_active ? '启用' : '禁用' }}
                            </span>
                          </div>
                          <div class="text-gray-500 text-sm mt-1">编号：{{ t.code }}　账号：{{ t.account }}</div>
                        </div>
                        <div class="text-gray-400 text-xs">ID: {{ t.id }}</div>
                      </div>

                      <div class="mt-3 flex gap-2">
                        <button type="button" class="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium" @click="openEdit(t)">编辑</button>
                        <button
                          type="button"
                          class="px-3 py-2 rounded-lg text-sm font-medium"
                          :class="t.is_active ? 'bg-orange-50 text-orange-600' : 'bg-green-50 text-green-600'"
                          @click="toggleActive(t)"
                        >
                          {{ t.is_active ? '禁用' : '启用' }}
                        </button>
                        <div class="flex-1"></div>
                        <button type="button" class="px-3 py-2 bg-red-50 text-red-600 rounded-lg text-sm font-medium" @click="removeTech(t)">删除</button>
                      </div>
                    </div>
                  </div>
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
          <div class="font-medium text-gray-800">{{ staffModalTitle }}</div>
          <button type="button" class="text-gray-400" @click="closeAdd">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5">
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">{{ staffNameLabel }}</label>
            <input
              v-model="form.name"
              type="text"
              :placeholder="staffNamePlaceholder"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <div v-if="!isEdit" class="text-gray-500 text-sm mb-5">
            系统将自动生成账号（前缀+4位编号自增），默认密码为账号+123
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

    <div v-if="showAddRole" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closeAddRole">
      <div class="bg-white rounded-2xl w-full max-w-md overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">添加专业客服</div>
          <button type="button" class="text-gray-400" @click="closeAddRole">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5">
          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">称谓</label>
            <input
              v-model="roleForm.name"
              type="text"
              placeholder="如：助教"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">账号前缀</label>
            <input
              v-model="roleForm.account_prefix"
              type="text"
              placeholder="如：zj"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
            <div class="text-gray-500 text-sm mt-2">最多5个英文字母，如：zj / js</div>
          </div>

          <button
            type="button"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
            :disabled="savingRole"
            @click="submitRole"
          >
            {{ savingRole ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi, platformApi } from '../../api'

const router = useRouter()

const loading = ref(false)
const saving = ref(false)
const savingRole = ref(false)
const techs = ref([])

const operationalRolesData = ref([])
const professionalRolesData = ref([])

const activeType = ref('operational')

const operationalRoles = computed(() => {
  return operationalRolesData.value.filter((r) => r && (r.key === 'store_manager' || r.key === 'front_desk'))
})

const professionalRoles = computed(() => {
  return professionalRolesData.value
})

const allRoles = computed(() => {
  return [...(operationalRolesData.value || []), ...(professionalRolesData.value || [])]
})

const selectedOperationalRole = ref('store_manager')
const selectedProfessionalRole = ref('')

const selectedOperationalRoleObj = computed(() => {
  const k = selectedOperationalRole.value
  return operationalRoles.value.find((r) => r && r.key === k) || null
})

const selectedProfessionalRoleObj = computed(() => {
  const k = selectedProfessionalRole.value
  return professionalRoles.value.find((r) => r && r.key === k) || null
})

const staffModalTitle = computed(() => {
  if (isEdit.value) return '编辑工作人员'
  if (activeType.value === 'professional') {
    return `添加${selectedProfessionalRoleObj.value?.name || '专业客服'}`
  }
  return `添加${selectedOperationalRoleObj.value?.name || '运营客服'}`
})

const staffNameLabel = computed(() => {
  if (activeType.value === 'professional') return '称谓/昵称'
  return '姓名'
})

const staffNamePlaceholder = computed(() => {
  if (activeType.value === 'professional') return '如：老师1'
  return '如：朱迪亚'
})

const visibleTechs = computed(() => {
  const list = techs.value || []
  if (activeType.value === 'operational') {
    const k = selectedOperationalRole.value
    return list.filter((t) => String(t?.service_role?.key || '') === String(k))
  }
  return list.filter((t) => {
    const k = String(t?.service_role?.key || '')
    return k !== 'store_manager' && k !== 'front_desk'
  })
})

// 专业客服按岗位分组
const professionalTechsByRole = computed(() => {
  const list = techs.value || []
  const professionalTechs = list.filter((t) => {
    const k = String(t?.service_role?.key || '')
    return k !== 'store_manager' && k !== 'front_desk'
  })
  
  const grouped = {}
  professionalTechs.forEach(t => {
    const roleName = t.service_role?.name || '未知岗位'
    if (!grouped[roleName]) {
      grouped[roleName] = {
        role: t.service_role,
        techs: []
      }
    }
    grouped[roleName].techs.push(t)
  })
  
  return grouped
})

const showAdd = ref(false)
const isEdit = ref(false)
const form = ref({
  id: 0,
  name: '',
  code: ''
})

const showAddRole = ref(false)
const roleForm = ref({
  name: '',
  account_prefix: ''
})

const goBack = () => {
  router.back()
}

const openPermissionAdjustOperational = () => {
  if (!selectedOperationalRole.value) return
  router.push(`/merchant/role-permissions/${selectedOperationalRole.value}`)
}

const openPermissionAdjustProfessional = (roleKey) => {
  if (!roleKey) return
  router.push(`/merchant/role-permissions/${roleKey}`)
}

const load = async () => {
  loading.value = true
  try {
    const res = await merchantApi.getTechnicians(activeType.value === 'operational' ? selectedOperationalRole.value : '')
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
  form.value = { id: 0, name: '' }
}

const openCreate = () => {
  isEdit.value = false
  form.value = { id: 0, name: '' }
  showAdd.value = true
}

const openCreateOperational = () => {
  openCreate()
}

const openCreateProfessionalRole = () => {
  roleForm.value = { name: '', account_prefix: '' }
  showAddRole.value = true
}

const openCreateProfessionalStaff = () => {
  openCreate()
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
    alert('请输入姓名')
    return
  }

  if (!isEdit.value) {
    if (activeType.value === 'professional' && !selectedProfessionalRole.value) {
      alert('请选择岗位（称谓）')
      return
    }
  }

  saving.value = true
  try {
    if (isEdit.value) {
      await merchantApi.updateTechnician(form.value.id, { name: form.value.name })
      alert('更新成功')
    } else {
      const res = await merchantApi.createTechnician({
        name: form.value.name,
        role: activeType.value === 'operational' ? selectedOperationalRole.value : selectedProfessionalRole.value
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
  if (!confirm('确定要删除该工作人员吗？')) return
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

const selectType = async (t) => {
  activeType.value = t
  closeAdd()
  await load()
}

const selectOperationalRole = async (key) => {
  selectedOperationalRole.value = key
  closeAdd()
  await load()
}

const loadProfessionalRoles = async () => {
  try {
    const res = await merchantApi.getProfessionalRoles()
    professionalRolesData.value = res.data?.data || []
  } catch (e) {
    professionalRolesData.value = []
  }
}

const closeAddRole = () => {
  showAddRole.value = false
  roleForm.value = { name: '', account_prefix: '' }
}

const submitRole = async () => {
  if (savingRole.value) return
  const name = String(roleForm.value.name || '').trim()
  const prefix = String(roleForm.value.account_prefix || '').trim()
  if (!name) {
    alert('请输入称谓')
    return
  }
  if (!prefix) {
    alert('请输入账号前缀')
    return
  }
  savingRole.value = true
  try {
    const res = await merchantApi.createProfessionalRole({ name, account_prefix: prefix })
    const role = res?.data?.data
    await loadProfessionalRoles()
    if (role && role.key) {
      selectedProfessionalRole.value = String(role.key)
    }
    closeAddRole()
    alert('创建成功')
  } catch (e) {
    alert(e.response?.data?.error || '创建失败')
  } finally {
    savingRole.value = false
  }
}

onMounted(async () => {
  try {
    const res = await platformApi.getServiceRoles()
    operationalRolesData.value = res.data?.data || []
  } catch (e) {
    operationalRolesData.value = []
  }

  await loadProfessionalRoles()

  const opFirst = operationalRoles.value[0]
  if (opFirst && opFirst.key) {
    selectedOperationalRole.value = String(opFirst.key)
  }
  const proFirst = professionalRoles.value[0]
  if (proFirst && proFirst.key) {
    selectedProfessionalRole.value = String(proFirst.key)
  }

  activeType.value = operationalRoles.value.length > 0 ? 'operational' : 'professional'
  await load()
})

// 监听角色切换
watch([activeType, selectedOperationalRole, selectedProfessionalRole], () => {
  load()
})
</script>
