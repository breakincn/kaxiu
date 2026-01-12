<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white px-4 py-3 flex items-center gap-3 border-b sticky top-0 z-10">
      <button @click="goBack" class="p-1">
        <svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>
      <span class="font-medium text-gray-800">项目设置</span>
    </header>

    <div class="px-4 py-6">
      <div v-if="loading" class="text-gray-400 text-center py-10">加载中...</div>

      <div v-else class="space-y-4">
        <div class="bg-white rounded-xl shadow-sm overflow-hidden">
          <div class="px-4 py-4 border-b border-gray-100">
            <div class="text-gray-800 font-medium">项目列表</div>
          </div>
          
          <div class="px-4 py-4 space-y-4">
            <div v-for="(project, index) in form.projects" :key="index" class="border border-gray-200 rounded-lg p-4 space-y-3">
              <div class="flex items-center justify-between">
                <div class="text-sm font-medium text-gray-700">项目 {{ index + 1 }}</div>
                <button 
                  @click="removeProject(index)"
                  class="text-red-500 hover:text-red-700"
                  v-if="form.projects.length > 1"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                  </svg>
                </button>
              </div>
              
              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">项目名</div>
                <input
                  v-model="project.name"
                  type="text"
                  placeholder="如 A 项目(课型)"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
              
              <div>
                <div class="text-sm font-medium text-gray-700 mb-2">项目服务时长（分钟）</div>
                <input
                  v-model.number="project.duration"
                  type="number"
                  min="1"
                  placeholder="如 45"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
            
            <button
              @click="addProject"
              class="w-full py-2 border border-dashed border-gray-300 rounded-lg text-gray-600 hover:border-gray-400 hover:text-gray-800 transition-colors"
            >
              + 添加项目
            </button>
          </div>
        </div>

        <button
          @click="save"
          :disabled="saving"
          class="w-full bg-blue-500 text-white py-3 rounded-lg hover:bg-blue-600 font-medium disabled:bg-gray-300 disabled:cursor-not-allowed"
        >
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { merchantProjectApi } from '../../api'

const router = useRouter()

const loading = ref(true)
const saving = ref(false)

const removedProjectIds = ref([])

const form = ref({
  projects: [
    { id: null, name: 'A 项目(课型)', duration: 45 },
    { id: null, name: 'B 项目(课型)', duration: 60 }
  ]
})

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
    setTimeout(() => {
      if (router.currentRoute.value.path === '/merchant/project-settings') {
        router.push('/merchant/settings')
      }
    }, 80)
    return
  }
  router.push('/merchant/settings')
}

const load = async () => {
  loading.value = true
  try {
    const res = await merchantProjectApi.list()
    const list = res.data?.data || []
    removedProjectIds.value = []
    form.value = {
      projects: list.length > 0
        ? list.map(p => ({
          id: p.id,
          name: p.name,
          duration: p.duration
        }))
        : [
          { id: null, name: 'A 项目(课型)', duration: 45 },
          { id: null, name: 'B 项目(课型)', duration: 60 }
        ]
    }
  } catch (e) {
    console.error('加载项目设置失败', e)
    alert(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const addProject = () => {
  form.value.projects.push({ id: null, name: '', duration: 30 })
}

const removeProject = (index) => {
  const p = form.value.projects[index]
  if (p && p.id) {
    removedProjectIds.value.push(p.id)
  }
  form.value.projects.splice(index, 1)
}

const save = async () => {
  if (saving.value) return
  
  // 验证项目列表
  for (let i = 0; i < form.value.projects.length; i++) {
    const project = form.value.projects[i]
    if (!project.name || !project.name.trim()) {
      alert(`项目 ${i + 1} 的名称不能为空`)
      return
    }
    if (!project.duration || project.duration < 1) {
      alert(`项目 ${i + 1} 的服务时长必须大于等于1`)
      return
    }
  }

  saving.value = true
  try {
    // 先删除
    for (const id of removedProjectIds.value) {
      try {
        await merchantProjectApi.delete(id)
      } catch (e) {
        // ignore
      }
    }

    // 再新增/更新
    for (const p of form.value.projects) {
      const payload = {
        name: (p.name || '').trim(),
        duration: Number(p.duration || 0)
      }
      if (p.id) {
        await merchantProjectApi.update(p.id, payload)
      } else {
        await merchantProjectApi.create(payload)
      }
    }

    alert('保存成功')
    await load()
  } catch (e) {
    alert(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  load()
})
</script>
