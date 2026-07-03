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
          <el-button size="small" type="primary" @click="action(row, 'update-software')">更新软件</el-button>
          <el-button size="small" @click="action(row, 'start')">启动</el-button>
          <el-button size="small" @click="action(row, 'stop')">停止</el-button>
          <el-button size="small" @click="action(row, 'restart')">重启</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listClients, clientAction } from '../api'

const loading = ref(false)
const clients = ref([] as any[])

async function load() {
  loading.value = true
  try {
    const res = await listClients()
    clients.value = res.data
  } finally {
    loading.value = false
  }
}

async function action(row: any, act: string) {
  let version = undefined
  if (act === 'update-software') {
    const input = await ElMessageBox.prompt('留空则更新到最新版本，或输入指定版本号', '更新软件', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputValue: '',
    }).catch(() => null)
    if (input === null) return
    version = input.value
  }
  try {
    await clientAction(row.id, act, version || undefined)
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
