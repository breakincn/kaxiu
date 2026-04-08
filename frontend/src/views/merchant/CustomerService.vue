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
                  type="button"
                  class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                  @click="openCreateOperationalRole"
                >
                  添加岗位
                </button>
              </template>
              <template v-else>
                <button
                  type="button"
                  class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium"
                  @click="openCreateProfessionalRole"
                >
                  添加岗位
                </button>
              </template>
            </div>
          </div>
        </div>

        <div>
          <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>

          <div v-else>
            <div v-if="activeType === 'operational'" class="mt-4">
              <div class="mb-3">
                <label class="block text-gray-700 text-sm font-medium mb-2">新增客服（岗位）</label>
                <div class="flex items-center gap-2">
                  <select v-model="selectedOperationalRole" class="flex-1 px-4 py-3 border border-gray-200 rounded-lg bg-white">
                    <option v-for="r in operationalRoles" :key="r.key" :value="r.key">{{ r.name }}</option>
                  </select>
                  <button
                    type="button"
                    class="px-4 py-3 bg-primary text-white rounded-lg text-sm font-medium shrink-0"
                    :disabled="!selectedOperationalRole"
                    @click="openCreateOperationalStaff"
                  >
                    添加
                  </button>
                </div>
              </div>

              <div v-if="Object.keys(operationalTechsByRole).length === 0" class="text-center text-gray-400 py-10">暂无运营客服</div>

              <div v-else class="mt-4">
                <div v-for="(group, roleName) in operationalTechsByRole" :key="roleName" class="mb-6">
                  <div class="flex items-center justify-between mb-3">
                    <div class="flex items-center gap-2">
                      <div class="text-gray-800 font-medium">{{ roleName }}</div>
                      <span class="px-2 py-0.5 bg-blue-50 text-blue-600 rounded text-xs">{{ group.techs.length }}人</span>
                    </div>
                    <div class="flex items-center gap-2">
                      <div v-if="group.role" class="flex items-center gap-2">
                        <div class="text-gray-700 text-sm">开启签到</div>
                        <input
                          type="checkbox"
                          :checked="getRoleRequireAttendance(group.role.key)"
                          :disabled="attendanceConfigLoading || attendanceConfigSaving"
                          @change="(e) => onToggleRoleAttendance(group.role.key, e.target.checked)"
                        />
                      </div>
                      <button
                        v-if="group.role && group.role.allow_permission_adjust"
                        type="button"
                        class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                        @click="openPermissionAdjustOperationalByKey(group.role.key)"
                      >
                        权限微调
                      </button>
                      <button
                        v-if="group.role"
                        type="button"
                        class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                        @click="openStartPendingAdjust(group.role.key)"
                      >
                        参数设置
                      </button>
                    </div>
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
                        <button
                          v-if="t.can_view_original_password"
                          type="button"
                          class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                          @click="viewOriginalPassword(t)"
                        >
                          查看原始密码
                        </button>
                        <div class="flex-1"></div>
                        <button type="button" class="px-3 py-2 bg-red-50 text-red-600 rounded-lg text-sm font-medium" @click="removeTech(t)">删除</button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="mt-4">
              <div class="mb-3">
                <label class="block text-gray-700 text-sm font-medium mb-2">新增客服（岗位）</label>
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

              <div v-else class="mt-4">
                <div v-for="(group, roleName) in professionalTechsByRole" :key="roleName" class="mb-6">
                  <div class="flex items-center justify-between mb-3">
                    <div class="flex items-center gap-2">
                      <div class="text-gray-800 font-medium">{{ roleName }}</div>
                      <span class="px-2 py-0.5 bg-blue-50 text-blue-600 rounded text-xs">{{ group.techs.length }}人</span>
                    </div>
                    <div class="flex items-center gap-2">
                      <div v-if="group.role" class="flex items-center gap-2">
                        <div class="text-gray-700 text-sm">开启签到</div>
                        <input
                          type="checkbox"
                          :checked="getRoleRequireAttendance(group.role.key)"
                          :disabled="attendanceConfigLoading || attendanceConfigSaving"
                          @change="(e) => onToggleRoleAttendance(group.role.key, e.target.checked)"
                        />
                      </div>
                      <button
                        v-if="group.role && group.role.allow_permission_adjust"
                        type="button"
                        class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                        @click="openPermissionAdjustProfessional(group.role.key)"
                      >
                        权限微调
                      </button>
                      <button
                        v-if="group.role"
                        type="button"
                        class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                        @click="openStartPendingAdjust(group.role.key)"
                      >
                        参数设置
                      </button>
                    </div>
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
                          <div v-if="shouldShowWindowNo && t.window_no" class="text-gray-500 text-sm mt-1">{{ windowTerm }}：{{ t.window_no }}</div>
                          <div v-if="canConfigureAppointmentProjects(t)" class="text-gray-500 text-sm mt-1">{{ getAppointmentProjectSummary(t) }}</div>
                        </div>
                      </div>

                      <div class="mt-3 flex gap-2">
                        <button type="button" class="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium" @click="openEdit(t)">编辑</button>
                        <button
                          v-if="canConfigureAppointmentProjects(t)"
                          type="button"
                          class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                          @click="openProjectConfig(t)"
                        >
                          预约项目
                        </button>
                        <button
                          type="button"
                          class="px-3 py-2 rounded-lg text-sm font-medium"
                          :class="t.is_active ? 'bg-orange-50 text-orange-600' : 'bg-green-50 text-green-600'"
                          @click="toggleActive(t)"
                        >
                          {{ t.is_active ? '禁用' : '启用' }}
                        </button>
                        <button
                          v-if="t.can_view_original_password"
                          type="button"
                          class="px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm font-medium"
                          @click="viewOriginalPassword(t)"
                        >
                          查看原始密码
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

          <!-- 窗口号（仅开启叫号模式时显示，且仅对专业客服显示） -->
          <div v-if="shouldShowWindowNo" class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">{{ windowTerm }}（可选）</label>
            <input
              v-model="form.window_no"
              type="text"
              :placeholder="`如：1、A1等`"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
            <div class="text-gray-500 text-xs mt-2">用于叫号时显示服务{{ windowTerm }}</div>
          </div>

          <div v-if="!isEdit" class="text-gray-500 text-sm mb-5">
            系统将自动生成账号和 8 位纯数字默认密码，创建后可查看并复制原始密码
          </div>

          <div v-if="isEdit" class="mb-5">
            <label class="block text-gray-700 text-sm font-medium mb-2">重置密码</label>
            <button
              type="button"
              class="w-full py-3 border border-primary text-primary rounded-lg font-medium disabled:opacity-50"
              :disabled="saving"
              @click="openResetPasswordModal"
            >
              重置密码
            </button>
            <div class="text-gray-500 text-xs mt-2">可留空按系统规则自动生成新密码，或手动输入新密码</div>
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
          <div class="font-medium text-gray-800">{{ roleModalTitle }}</div>
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
              @input="onRoleNameInput"
            />
          </div>

          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">账号前缀</label>
            <input
              v-model="roleForm.account_prefix"
              type="text"
              placeholder="如：zj"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
              @input="onRolePrefixInput"
              @click="onRolePrefixClick"
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

    <div v-if="showPasswordModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closePasswordModal">
      <div class="bg-white rounded-2xl w-full max-w-md overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">{{ passwordModalTitle }}</div>
          <button type="button" class="text-gray-400" @click="closePasswordModal">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5">
          <div class="text-sm text-gray-600 mb-2">{{ passwordModalName }}</div>
          <div class="px-4 py-3 rounded-lg bg-gray-50 border border-gray-200 text-gray-800 font-mono break-all">{{ passwordModalPassword }}</div>
          <button type="button" class="w-full mt-4 py-3 bg-primary text-white rounded-lg font-medium" @click="copyPassword">
            复制密码
          </button>
        </div>
      </div>
    </div>

    <div v-if="showResetPasswordConfirm" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closeResetPasswordModal">
      <div class="bg-white rounded-2xl w-full max-w-md overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">重置密码</div>
          <button type="button" class="text-gray-400" @click="closeResetPasswordModal">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5">
          <div class="text-sm text-gray-600 mb-4">确认重置“{{ form.name || '该工作人员' }}”的密码？如不填写，系统将自动生成新密码。</div>
          <div class="mb-5">
            <label class="block text-gray-700 text-sm font-medium mb-2">新密码（可选）</label>
            <input
              v-model="resetPasswordForm.newPassword"
              type="text"
              placeholder="留空则自动生成"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
            <div class="text-gray-500 text-xs mt-2">手动输入时至少 8 位</div>
          </div>

          <div class="flex gap-3">
            <button
              type="button"
              class="flex-1 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium"
              :disabled="resettingPassword"
              @click="closeResetPasswordModal"
            >
              取消
            </button>
            <button
              type="button"
              class="flex-1 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
              :disabled="resettingPassword"
              @click="confirmResetPassword"
            >
              {{ resettingPassword ? '重置中...' : '确认重置' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showProjectConfigModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closeProjectConfig">
      <div class="bg-white rounded-2xl w-full max-w-lg overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">设置可预约项目</div>
          <button type="button" class="text-gray-400" @click="closeProjectConfig">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="px-5 py-5 max-h-[75vh] overflow-y-auto">
          <div class="text-sm text-gray-800 font-medium">{{ projectConfigTech?.name || '-' }}</div>
          <div class="text-xs text-gray-500 mt-1">未单独配置时，默认继承商户全部线上可预约项目。</div>

          <div class="mt-4 grid grid-cols-2 gap-2">
            <button
              type="button"
              class="px-3 py-2 rounded-lg text-sm font-medium border"
              :class="projectConfigBindingMode === 'merchant_default' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
              @click="projectConfigBindingMode = 'merchant_default'"
            >
              继承商户默认
            </button>
            <button
              type="button"
              class="px-3 py-2 rounded-lg text-sm font-medium border"
              :class="projectConfigBindingMode === 'custom' ? 'bg-primary text-white border-primary' : 'bg-white text-gray-700 border-gray-200'"
              @click="enableCustomProjectBinding"
            >
              单独指定项目
            </button>
          </div>

          <div class="mt-4 flex items-center justify-between text-xs text-gray-500">
            <div v-if="projectConfigBindingMode === 'merchant_default'">当前生效：全部线上可预约项目</div>
            <div v-else>当前已选 {{ projectConfigSelectedIds.length }} 项</div>
            <button
              v-if="projectConfigBindingMode === 'custom'"
              type="button"
              class="text-primary"
              @click="selectAllProjectConfig"
            >
              全选
            </button>
          </div>

          <div v-if="projectConfigLoading" class="text-center text-gray-400 py-10">加载中...</div>
          <div v-else-if="projectConfigProjects.length === 0" class="text-center text-gray-400 py-10">暂无可预约项目</div>
          <div v-else class="mt-4 space-y-3">
            <label
              v-for="project in projectConfigProjects"
              :key="project.id"
              class="flex items-start gap-3 px-3 py-3 border border-gray-200 rounded-lg"
              :class="projectConfigBindingMode === 'merchant_default' ? 'opacity-60' : ''"
            >
              <input
                v-model="projectConfigSelectedIds"
                type="checkbox"
                :value="project.id"
                class="mt-0.5 h-4 w-4"
                :disabled="projectConfigBindingMode !== 'custom'"
              />
              <div class="min-w-0">
                <div class="text-sm text-gray-800 font-medium">{{ project.name }}</div>
                <div class="text-xs text-gray-500 mt-1">时长 {{ project.duration }} 分钟</div>
              </div>
            </label>
          </div>
        </div>

        <div class="px-5 py-4 border-t flex gap-3">
          <button
            type="button"
            class="flex-1 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium"
            :disabled="projectConfigSaving"
            @click="closeProjectConfig"
          >
            取消
          </button>
          <button
            type="button"
            class="flex-1 py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
            :disabled="projectConfigSaving || projectConfigLoading"
            @click="saveProjectConfig"
          >
            {{ projectConfigSaving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

const router = useRouter()

const loading = ref(false)
const saving = ref(false)
const savingRole = ref(false)
const resettingPassword = ref(false)
const techs = ref([])

const attendanceConfigLoading = ref(false)
const attendanceConfigSaving = ref(false)
const roleAttendanceMap = ref({})

const merchant = ref({})

const operationalRolesData = ref([])
const professionalRolesData = ref([])

const activeType = ref('operational')

// 从本地存储恢复上次选择的客服类型
const restoreActiveType = () => {
  const saved = localStorage.getItem('customer_service_active_type')
  if (saved === 'operational' || saved === 'professional') {
    activeType.value = saved
  }
}

// 保存客服类型到本地存储
const saveActiveType = (type) => {
  localStorage.setItem('customer_service_active_type', type)
}

const operationalRoles = computed(() => {
  return (operationalRolesData.value || []).filter((r) => {
    if (!r) return false
    const k = String(r.key || '').trim()
    if (k === 'store_manager' || k === 'front_desk') return true
    return String(r.role_type || '').trim() === 'operational'
  })
})

const professionalRoles = computed(() => {
  const out = [...(professionalRolesData.value || [])].filter((r) => {
    if (!r) return false
    return String(r.role_type || '').trim() === 'professional'
  })
  out.sort((a, b) => {
    const sa = Number(a?.sort || 0)
    const sb = Number(b?.sort || 0)
    if (sa !== sb) return sa - sb
    return Number(a?.id || 0) - Number(b?.id || 0)
  })
  return out
})

// 是否显示窗口号（开启叫号模式且为专业客服）
const shouldShowWindowNo = computed(() => {
  return merchant.value?.support_queue && activeType.value === 'professional'
})

// 窗口自定义名词
const windowTerm = computed(() => {
  return merchant.value?.queue_window_term || '窗口'
})

const merchantSupportsAppointmentProjectBinding = computed(() => {
  return !!merchant.value?.support_appointment
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

const visibleTechsByRole = (roleKey) => {
  const list = techs.value || []
  return list.filter((t) => String(t?.service_role?.key || '') === String(roleKey))
}

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
    const k = String(t?.service_role?.key || '').trim()
    if (k === 'store_manager' || k === 'front_desk') return false
    const rt = String(t?.service_role?.role_type || '').trim()
    return rt === 'professional'
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

// 运营客服按岗位分组
const operationalTechsByRole = computed(() => {
  const list = techs.value || []
  const operationalTechs = list.filter((t) => {
    const k = String(t?.service_role?.key || '').trim()
    if (k === 'store_manager' || k === 'front_desk') return true
    const rt = String(t?.service_role?.role_type || '').trim()
    return rt === 'operational'
  })

  const grouped = {}
  operationalTechs.forEach((t) => {
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
  code: '',
  window_no: ''
})
const showPasswordModal = ref(false)
const passwordModalTitle = ref('查看原始密码')
const passwordModalName = ref('')
const passwordModalPassword = ref('')
const showResetPasswordConfirm = ref(false)
const resetPasswordForm = ref({
  newPassword: ''
})
const showProjectConfigModal = ref(false)
const projectConfigLoading = ref(false)
const projectConfigSaving = ref(false)
const projectConfigTech = ref(null)
const projectConfigProjects = ref([])
const projectConfigSelectedIds = ref([])
const projectConfigBindingMode = ref('merchant_default')

const showAddRole = ref(false)
const roleForm = ref({
  name: '',
  account_prefix: ''
})
const rolePrefixAutoMode = ref(true)
const rolePrefixLastClickAt = ref(0)

const roleModalTitle = computed(() => {
  return activeType.value === 'operational' ? '添加运营岗位' : '添加专业岗位'
})

const goBack = () => {
  router.back()
}

const openPermissionAdjustOperational = () => {
  if (!selectedOperationalRole.value) return
  router.push(`/merchant/role-permissions/${selectedOperationalRole.value}`)
}

const openPermissionAdjustOperationalByKey = (roleKey) => {
  if (!roleKey) return
  router.push(`/merchant/role-permissions/${roleKey}`)
}

const openPermissionAdjustProfessional = (roleKey) => {
  if (!roleKey) return
  router.push(`/merchant/role-permissions/${roleKey}`)
}

const openStartPendingAdjust = (roleKey) => {
  if (!roleKey) return
  const role = allRoles.value.find((r) => r?.key === roleKey)
  router.push({
    path: `/merchant/role-start-pending-settings/${roleKey}`,
    query: {
      ...(role?.name ? { role_name: role.name } : {}),
      ...(merchant.value?.start_term ? { start_term: merchant.value.start_term } : {})
    }
  })
}

const load = async () => {
  loading.value = true
  try {
    // 运营客服需要拉取所有店长和前台数据用于分组展示
    const res = await merchantApi.getTechnicians(activeType.value === 'operational' ? '' : '')
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
  form.value = { id: 0, name: '', window_no: '' }
  closeResetPasswordModal()
}

const openPasswordModal = ({ title, name, password }) => {
  passwordModalTitle.value = title || '查看原始密码'
  passwordModalName.value = name || ''
  passwordModalPassword.value = password || ''
  showPasswordModal.value = true
}

const closePasswordModal = () => {
  showPasswordModal.value = false
  passwordModalTitle.value = '查看原始密码'
  passwordModalName.value = ''
  passwordModalPassword.value = ''
}

const closeProjectConfig = () => {
  showProjectConfigModal.value = false
  projectConfigLoading.value = false
  projectConfigSaving.value = false
  projectConfigTech.value = null
  projectConfigProjects.value = []
  projectConfigSelectedIds.value = []
  projectConfigBindingMode.value = 'merchant_default'
}

const closeAllModals = () => {
  closePasswordModal()
  closeResetPasswordModal()
  closeProjectConfig()
  closeAdd()
  closeAddRole()
}

const copyPassword = async () => {
  if (!passwordModalPassword.value) return
  try {
    await navigator.clipboard.writeText(passwordModalPassword.value)
    closeAllModals()
    alert('已复制')
  } catch (e) {
    alert('复制失败')
  }
}

const openCreate = () => {
  isEdit.value = false
  form.value = { id: 0, name: '', window_no: '' }
  showAdd.value = true
}

const openCreateOperational = () => {
  openCreate()
}

const openCreateOperationalByKey = (roleKey) => {
  selectedOperationalRole.value = roleKey
  openCreate()
}

const openCreateProfessionalRole = () => {
  roleForm.value = { name: '', account_prefix: '' }
  rolePrefixAutoMode.value = true
  showAddRole.value = true
}

const openCreateOperationalRole = () => {
  roleForm.value = { name: '', account_prefix: '' }
  rolePrefixAutoMode.value = true
  showAddRole.value = true
}

const openCreateProfessionalStaff = () => {
  openCreate()
}

const openCreateOperationalStaff = () => {
  openCreate()
}

const openEdit = (t) => {
  if (!t || !t.id) return
  isEdit.value = true
  form.value = {
    id: t.id,
    name: t.name || '',
    code: t.code || '',
    window_no: t.window_no || ''
  }
  showAdd.value = true
}

const openResetPasswordModal = () => {
  if (!isEdit.value || !form.value.id) return
  resetPasswordForm.value = { newPassword: '' }
  showResetPasswordConfirm.value = true
}

const closeResetPasswordModal = () => {
  showResetPasswordConfirm.value = false
  resetPasswordForm.value = { newPassword: '' }
}

const confirmResetPassword = async () => {
  if (!form.value.id || resettingPassword.value) return
  resettingPassword.value = true
  try {
    const res = await merchantApi.resetTechnicianPassword(form.value.id, resetPasswordForm.value.newPassword)
    const pwd = res?.data?.data?.original_password || ''
    closeResetPasswordModal()
    await load()
    if (pwd) {
      openPasswordModal({
        title: '密码已重置',
        name: form.value.name,
        password: pwd
      })
    } else {
      alert('密码已重置')
    }
  } catch (e) {
    alert(e.response?.data?.error || '重置密码失败')
  } finally {
    resettingPassword.value = false
  }
}

const viewOriginalPassword = async (t) => {
  if (!t || !t.id) return
  try {
    const res = await merchantApi.getTechnicianOriginalPassword(t.id)
    const data = res?.data?.data || {}
    openPasswordModal({
      title: '查看原始密码',
      name: data.name || t.name || '',
      password: data.original_password || ''
    })
  } catch (e) {
    alert(e.response?.data?.error || '获取原始密码失败')
    await load()
  }
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
    if (activeType.value === 'operational' && !selectedOperationalRole.value) {
      alert('请选择岗位（称谓）')
      return
    }
  }

  saving.value = true
  try {
    if (isEdit.value) {
      const payload = { name: form.value.name }
      if (shouldShowWindowNo.value) {
        payload.window_no = form.value.window_no || ''
      }
      await merchantApi.updateTechnician(form.value.id, payload)
      alert('更新成功')
    } else {
      const res = await merchantApi.createTechnician({
        name: form.value.name,
        role: activeType.value === 'operational' ? selectedOperationalRole.value : selectedProfessionalRole.value
      })
      const pwd = res?.data?.data?.default_password
      if (pwd) {
        openPasswordModal({
          title: '创建成功',
          name: form.value.name,
          password: pwd
        })
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

const canConfigureAppointmentProjects = (t) => {
  if (!merchantSupportsAppointmentProjectBinding.value) return false
  return String(t?.service_role?.role_type || '').trim() !== 'operational'
}

const getAppointmentProjectSummary = (t) => {
  const mode = String(t?.appointment_project_binding_mode || 'merchant_default').trim()
  const projectIds = Array.isArray(t?.appointment_project_ids) ? t.appointment_project_ids : []
  if (mode !== 'custom') return '预约项目：全部线上项目'
  return `预约项目：已选 ${projectIds.length} 项`
}

const enableCustomProjectBinding = () => {
  projectConfigBindingMode.value = 'custom'
  if (projectConfigSelectedIds.value.length === 0) {
    projectConfigSelectedIds.value = projectConfigProjects.value.map((item) => item.id)
  }
}

const selectAllProjectConfig = () => {
  projectConfigSelectedIds.value = projectConfigProjects.value.map((item) => item.id)
}

const openProjectConfig = async (t) => {
  if (!t?.id || !canConfigureAppointmentProjects(t)) return
  showProjectConfigModal.value = true
  projectConfigLoading.value = true
  projectConfigTech.value = t
  projectConfigProjects.value = []
  projectConfigSelectedIds.value = []
  projectConfigBindingMode.value = 'merchant_default'
  try {
    const res = await merchantApi.getTechnicianAppointmentProjects(t.id)
    const data = res?.data?.data || {}
    const projects = Array.isArray(data.projects) ? data.projects : []
    const bindingMode = String(data.binding_mode || 'merchant_default').trim() || 'merchant_default'
    const projectIds = Array.isArray(data.project_ids)
      ? data.project_ids.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0)
      : []
    projectConfigProjects.value = projects
    projectConfigBindingMode.value = bindingMode
    projectConfigSelectedIds.value = bindingMode === 'custom'
      ? projectIds
      : projects.map((item) => item.id)
  } catch (e) {
    closeProjectConfig()
    alert(e.response?.data?.error || '加载客服可预约项目失败')
  } finally {
    projectConfigLoading.value = false
  }
}

const saveProjectConfig = async () => {
  if (!projectConfigTech.value?.id || projectConfigSaving.value) return
  projectConfigSaving.value = true
  try {
    const projectIds = projectConfigBindingMode.value === 'custom'
      ? [...new Set(projectConfigSelectedIds.value.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0))].sort((a, b) => a - b)
      : []
    await merchantApi.setTechnicianAppointmentProjects(projectConfigTech.value.id, { project_ids: projectIds })
    await load()
    closeProjectConfig()
    alert('保存成功')
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    projectConfigSaving.value = false
  }
}

const selectType = async (t) => {
  activeType.value = t
  saveActiveType(t)
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

const loadOperationalRoles = async () => {
  try {
    const res = await merchantApi.getOperationalRoles()
    operationalRolesData.value = res.data?.data || []
  } catch (e) {
    operationalRolesData.value = []
  }
}

const loadRoleAttendanceConfigs = async () => {
  attendanceConfigLoading.value = true
  try {
    const res = await merchantApi.getRoleAttendanceConfigs()
    const list = res.data?.data || []
    const m = {}
    list.forEach((it) => {
      if (!it || !it.service_role_key) return
      m[String(it.service_role_key)] = !!it.require_attendance
    })
    roleAttendanceMap.value = m
  } catch (e) {
    roleAttendanceMap.value = {}
  } finally {
    attendanceConfigLoading.value = false
  }
}

const getRoleRequireAttendance = (roleKey) => {
  const k = String(roleKey || '')
  if (!k) return true
  const v = roleAttendanceMap.value[k]
  if (typeof v === 'boolean') return v
  return true
}

const onToggleRoleAttendance = async (roleKey, checked) => {
  const k = String(roleKey || '').trim()
  if (!k) return
  if (attendanceConfigSaving.value) return
  attendanceConfigSaving.value = true
  try {
    await merchantApi.setRoleAttendanceConfig(k, !!checked)
    roleAttendanceMap.value = { ...roleAttendanceMap.value, [k]: !!checked }
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    attendanceConfigSaving.value = false
  }
}

const closeAddRole = () => {
  showAddRole.value = false
  roleForm.value = { name: '', account_prefix: '' }
  rolePrefixAutoMode.value = true
}

const zhInitialBoundaries = [
  { start: '阿', initial: 'a' },
  { start: '芭', initial: 'b' },
  { start: '擦', initial: 'c' },
  { start: '搭', initial: 'd' },
  { start: '蛾', initial: 'e' },
  { start: '发', initial: 'f' },
  { start: '噶', initial: 'g' },
  { start: '哈', initial: 'h' },
  { start: '击', initial: 'j' },
  { start: '喀', initial: 'k' },
  { start: '垃', initial: 'l' },
  { start: '妈', initial: 'm' },
  { start: '拿', initial: 'n' },
  { start: '哦', initial: 'o' },
  { start: '啪', initial: 'p' },
  { start: '期', initial: 'q' },
  { start: '然', initial: 'r' },
  { start: '撒', initial: 's' },
  { start: '塌', initial: 't' },
  { start: '挖', initial: 'w' },
  { start: '昔', initial: 'x' },
  { start: '压', initial: 'y' },
  { start: '匝', initial: 'z' }
]

const getChineseInitial = (char) => {
  if (!char || !/[\u4e00-\u9fff]/.test(char)) return ''
  for (let i = zhInitialBoundaries.length - 1; i >= 0; i -= 1) {
    const item = zhInitialBoundaries[i]
    if (char.localeCompare(item.start, 'zh-CN') >= 0) {
      return item.initial
    }
  }
  return ''
}

const generateRolePrefix = (name) => {
  const text = String(name || '').trim()
  if (!text) return ''

  const parts = text.match(/[A-Za-z]+|[\u4e00-\u9fff]/g) || []
  const initials = []

  parts.forEach((part) => {
    if (initials.length >= 5) return
    if (/^[A-Za-z]+$/.test(part)) {
      initials.push(part[0].toLowerCase())
      return
    }
    const initial = getChineseInitial(part)
    if (initial) initials.push(initial)
  })

  return initials.join('').slice(0, 5)
}

const onRoleNameInput = () => {
  if (!rolePrefixAutoMode.value) return
  roleForm.value.account_prefix = generateRolePrefix(roleForm.value.name)
}

const onRolePrefixInput = () => {
  const current = String(roleForm.value.account_prefix || '').trim().toLowerCase()
  roleForm.value.account_prefix = current
  rolePrefixAutoMode.value = false
}

const onRolePrefixDblClick = () => {
  roleForm.value.account_prefix = ''
  rolePrefixAutoMode.value = false
}

const onRolePrefixClick = () => {
  const now = Date.now()
  if (now - rolePrefixLastClickAt.value <= 300) {
    onRolePrefixDblClick()
    rolePrefixLastClickAt.value = 0
    return
  }
  rolePrefixLastClickAt.value = now
}

const submitRole = async () => {
  if (savingRole.value) return
  const name = String(roleForm.value.name || '').trim()
  const prefix = String(roleForm.value.account_prefix || '').trim()
  if (!name) {
    alert('请输入称谓')
    return
  }

	// 称谓重复校验：与“新增岗位（称谓）”下拉框一致
	const base = activeType.value === 'operational' ? (operationalRoles.value || []) : (professionalRoles.value || [])
	const exists = base.some((r) => String(r?.name || '').trim() === name)
	if (exists) {
		alert('该岗位称谓已经存在,请不要重复添加')
		return
	}

  if (!prefix) {
    alert('请输入账号前缀')
    return
  }
  savingRole.value = true
  try {
    const isOperational = activeType.value === 'operational'
    const res = isOperational
      ? await merchantApi.createOperationalRole({ name, account_prefix: prefix })
      : await merchantApi.createProfessionalRole({ name, account_prefix: prefix })
    const role = res?.data?.data
    if (isOperational) {
      await loadOperationalRoles()
    } else {
      await loadProfessionalRoles()
    }

    await loadRoleAttendanceConfigs()
    if (role && role.key) {
      if (isOperational) {
        selectedOperationalRole.value = String(role.key)
      } else {
        selectedProfessionalRole.value = String(role.key)
      }
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
  // 获取商户信息
  try {
    const res = await merchantApi.getCurrentMerchant()
    merchant.value = res.data?.data || {}
  } catch (e) {
    merchant.value = {}
  }

  await loadOperationalRoles()

  await loadProfessionalRoles()

  await loadRoleAttendanceConfigs()

  const opFirst = operationalRoles.value[0]
  if (opFirst && opFirst.key) {
    selectedOperationalRole.value = String(opFirst.key)
  }
  const proFirst = professionalRoles.value[0]
  if (proFirst && proFirst.key) {
    selectedProfessionalRole.value = String(proFirst.key)
  }

  // 恢复上次选择的客服类型，如果没有则根据数据情况设置默认
  restoreActiveType()
  if (!localStorage.getItem('customer_service_active_type')) {
    activeType.value = operationalRoles.value.length > 0 ? 'operational' : 'professional'
  }
  await load()
})

// 监听角色切换
watch([activeType, selectedProfessionalRole], () => {
  load()
})
</script>
