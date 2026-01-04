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
        <div class="flex items-center justify-between">
          <div class="text-gray-800 font-medium">技师账号</div>
          <button
            type="button"
            class="px-3 py-2 bg-primary text-white rounded-lg text-sm font-medium"
            @click="showAdd = true"
          >
            添加技师
          </button>
        </div>

        <div v-if="loading" class="text-center text-gray-400 py-10">加载中...</div>

        <div v-else>
          <div v-if="techs.length === 0" class="text-center text-gray-400 py-10">暂无技师</div>

          <div v-else class="mt-4 space-y-3">
            <div
              v-for="t in techs"
              :key="t.id"
              class="border border-gray-100 rounded-xl p-4"
            >
              <div class="flex items-center justify-between">
                <div>
                  <div class="text-gray-800 font-medium">{{ t.name }}</div>
                  <div class="text-gray-500 text-sm mt-1">编号：{{ t.code }}　账号：{{ t.account }}</div>
                </div>
                <div class="text-gray-400 text-xs">ID: {{ t.id }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showAdd" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center px-4 z-50" @click.self="closeAdd">
      <div class="bg-white rounded-2xl w-full max-w-md overflow-hidden">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <div class="font-medium text-gray-800">添加技师</div>
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

          <div class="mb-4">
            <label class="block text-gray-700 text-sm font-medium mb-2">技师编号</label>
            <input
              v-model="form.code"
              type="text"
              placeholder="如：0001"
              class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:border-primary"
            />
          </div>

          <div class="text-gray-500 text-sm mb-5">
            默认账号：<span class="text-gray-800 font-medium">js{{ form.code || 'xxxx' }}</span>
            <br />
            默认密码：<span class="text-gray-800 font-medium">{{ (form.code || 'xxxx') + '12345' }}</span>
          </div>

          <button
            type="button"
            class="w-full py-3 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
            :disabled="saving"
            @click="createTech"
          >
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api'

const router = useRouter()

const loading = ref(false)
const saving = ref(false)
const techs = ref([])

const showAdd = ref(false)
const form = ref({
  name: '',
  code: ''
})

const goBack = () => {
  router.back()
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
  form.value = { name: '', code: '' }
}

const createTech = async () => {
  if (saving.value) return
  if (!form.value.name) {
    alert('请输入技师姓名')
    return
  }
  if (!form.value.code) {
    alert('请输入技师编号')
    return
  }

  saving.value = true
  try {
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
    closeAdd()
    await load()
  } catch (e) {
    alert(e.response?.data?.error || '创建失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
