<template>
  <div>
    <h2>客户端管理</h2>
    <el-table :data="clients" v-loading="loading" border style="margin-top: 20px">
      <el-table-column type="selection" width="55" />
      <el-table-column prop="name" label="名称" width="140" />
      <el-table-column prop="client_version" label="客户端版本" width="120" />
      <el-table-column prop="software_version" label="软件版本" width="120" />
      <el-table-column label="运行状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.is_running ? 'success' : 'info'">{{ row.is_running ? '运行中' : '已停止' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="process_runtime" label="运行时长(秒)" width="120" />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="os_version" label="系统版本" />
      <el-table-column prop="memory" label="内存" width="140" />
      <el-table-column prop="cpu" label="CPU" />
      <el-table-column label="操作" width="320" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="openUpdateSoftware(row)">更新软件</el-button>
          <el-button size="small" @click="action(row, 'start')">启动</el-button>
          <el-button size="small" @click="action(row, 'stop')">停止</el-button>
          <el-button size="small" @click="action(row, 'restart')">重启</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="updateDialogVisible" title="更新软件" width="400px">
      <el-form label-position="top">
        <el-form-item label="选择版本（留空则更新到最新版本）">
          <el-select
            v-model="selectedVersion"
            placeholder="请选择软件版本"
            clearable
            style="width: 100%"
            v-loading="versionsLoading"
          >
            <el-option
              v-for="item in softwareVersions"
              :key="item.id"
              :label="`${item.name} (${item.version})${item.is_latest ? ' [最新]' : ''}`"
              :value="item.version"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="updateDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmUpdateSoftware" :loading="updateLoading">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listClients, clientAction, listSoftware } from '../api'

const loading = ref(false)
const clients = ref([] as any[])

const updateDialogVisible = ref(false)
const updateLoading = ref(false)
const versionsLoading = ref(false)
const softwareVersions = ref([] as any[])
const selectedVersion = ref('')
const currentClient = ref<any>(null)

async function load() {
  loading.value = true
  try {
    const res = await listClients()
    clients.value = res.data
  } finally {
    loading.value = false
  }
}

async function openUpdateSoftware(row: any) {
  currentClient.value = row
  selectedVersion.value = ''
  updateDialogVisible.value = true
  versionsLoading.value = true
  try {
    const res = await listSoftware()
    softwareVersions.value = res.data
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '获取软件版本列表失败')
  } finally {
    versionsLoading.value = false
  }
}

async function confirmUpdateSoftware() {
  if (!currentClient.value) return
  updateLoading.value = true
  try {
    await clientAction(
      currentClient.value.id,
      'update-software',
      selectedVersion.value || undefined
    )
    ElMessage.success('命令已下发')
    updateDialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '下发失败')
  } finally {
    updateLoading.value = false
  }
}

async function action(row: any, act: string) {
  try {
    await clientAction(row.id, act)
    ElMessage.success('命令已下发')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '下发失败')
  }
}

onMounted(() => {
  load()
  setInterval(load, 5000)
})
</script>
