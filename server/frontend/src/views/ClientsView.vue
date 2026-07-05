<template>
  <div>
    <h2>客户端管理</h2>
    <el-table
      :data="clients"
      v-loading="loading"
      border
      style="margin-top: 20px"
      :default-sort="{ prop: 'id', order: 'ascending' }"
      row-key="id"
      @expand-change="handleExpand"
    >
      <el-table-column type="selection" width="55" />
      <el-table-column type="expand" width="45">
        <template #default="{ row }">
          <div style="padding: 10px 20px">
            <el-descriptions :column="3" border size="small" title="客户端详情">
              <el-descriptions-item label="客户端版本">{{ row.client_version || '-' }}</el-descriptions-item>
              <el-descriptions-item label="运行时长">{{ row.process_runtime || 0 }} 秒</el-descriptions-item>
              <el-descriptions-item label="IP">{{ row.ip || '-' }}</el-descriptions-item>
              <el-descriptions-item label="系统版本">{{ row.os_version || '-' }}</el-descriptions-item>
              <el-descriptions-item label="内存">{{ row.memory || '-' }}</el-descriptions-item>
              <el-descriptions-item label="CPU">{{ row.cpu || '-' }}</el-descriptions-item>
            </el-descriptions>

            <h4 style="margin-top: 16px; margin-bottom: 8px">最近命令记录</h4>
            <el-table :data="rowCommands[row.id] || []" border size="small" v-loading="rowLoading[row.id]">
              <el-table-column prop="command_type" label="命令" width="120" />
              <el-table-column label="状态" width="120">
                <template #default="{ row: cmd }">
                  <el-tag :type="commandStatusType(cmd.status)" size="small">
                    {{ commandStatusText(cmd.status) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="progress" label="进度" width="100">
                <template #default="{ row: cmd }">
                  <span v-if="cmd.status === 'downloading'">{{ cmd.progress }}%</span>
                  <span v-else>-</span>
                </template>
              </el-table-column>
              <el-table-column prop="message" label="消息" show-overflow-tooltip />
              <el-table-column prop="updated_at" label="更新时间" width="160">
                <template #default="{ row: cmd }">
                  {{ formatTime(cmd.updated_at) }}
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="id" label="ID" width="70" sortable />
      <el-table-column prop="name" label="名称" width="140" />
      <el-table-column label="运行状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.is_running ? 'success' : 'info'" size="small">
            {{ row.is_running ? '运行中' : '已停止' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="在线状态" width="160">
        <template #default="{ row }">
          <el-tag :type="isOnline(row) ? 'success' : 'danger'" size="small">
            {{ isOnline(row) ? '在线 ' + onlineDuration(row) : '离线 ' + offlineDuration(row) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="software_version" label="软件版本" width="120" />
      <el-table-column label="命令状态" width="180">
        <template #default="{ row }">
          <div v-if="row.latest_command">
            <el-tag :type="commandStatusType(row.latest_command.status)" size="small">
              {{ commandStatusText(row.latest_command.status) }}
            </el-tag>
            <el-progress
              v-if="row.latest_command.status === 'downloading'"
              :percentage="row.latest_command.progress"
              :stroke-width="4"
              :show-text="false"
              style="margin-top: 4px"
            />
          </div>
          <span v-else class="text-muted">-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-dropdown trigger="click" @command="(cmd: string) => handleCommand(row, cmd)">
            <el-button size="small" type="primary">
              操作<el-icon class="el-icon--right"><arrow-down /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="update-software">更新软件</el-dropdown-item>
                <el-dropdown-item command="update-resource">更新资源包</el-dropdown-item>
                <el-dropdown-item command="rename">改名</el-dropdown-item>
                <el-dropdown-item command="start">启动</el-dropdown-item>
                <el-dropdown-item command="stop">停止</el-dropdown-item>
                <el-dropdown-item command="restart">重启</el-dropdown-item>
                <el-dropdown-item divided command="delete">删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
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

    <el-dialog v-model="renameDialogVisible" title="修改客户端名称" width="400px">
      <el-input v-model="editingName" placeholder="请输入新名称" />
      <template #footer>
        <el-button @click="renameDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmRename" :loading="renameLoading">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown } from '@element-plus/icons-vue'
import { listClients, clientAction, listSoftware, updateClientName, deleteClient, listClientCommands } from '../api'

const loading = ref(false)
const clients = ref([] as any[])
const rowCommands = ref<Record<number, any[]>>({})
const rowLoading = ref<Record<number, boolean>>({})

const updateDialogVisible = ref(false)
const updateLoading = ref(false)
const versionsLoading = ref(false)
const softwareVersions = ref([] as any[])
const selectedVersion = ref('')
const currentClient = ref<any>(null)

const renameDialogVisible = ref(false)
const renameLoading = ref(false)
const editingName = ref('')
const editingClientId = ref(0)

const ONLINE_THRESHOLD_SECONDS = 35

function isOnline(row: any) {
  if (!row.last_seen) return false
  const last = new Date(row.last_seen).getTime()
  return Date.now() - last < ONLINE_THRESHOLD_SECONDS * 1000
}

function offlineDuration(row: any) {
  if (!row.last_seen) return '未知'
  const last = new Date(row.last_seen).getTime()
  const diff = Math.floor((Date.now() - last) / 1000)
  if (diff < 60) return `${diff}秒`
  if (diff < 3600) return `${Math.floor(diff / 60)}分${diff % 60}秒`
  const hours = Math.floor(diff / 3600)
  const mins = Math.floor((diff % 3600) / 60)
  return `${hours}小时${mins}分`
}

function onlineDuration(row: any) {
  const since = row.online_since ? new Date(row.online_since).getTime() : 0
  if (!since) return ''
  const diff = Math.floor((Date.now() - since) / 1000)
  if (diff < 60) return `${diff}秒`
  if (diff < 3600) return `${Math.floor(diff / 60)}分${diff % 60}秒`
  const hours = Math.floor(diff / 3600)
  const mins = Math.floor((diff % 3600) / 60)
  return `${hours}小时${mins}分`
}

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return date.toLocaleString()
}

function commandStatusType(status: string) {
  switch (status) {
    case 'completed':
      return 'success'
    case 'failed':
      return 'danger'
    case 'downloading':
      return 'primary'
    case 'received':
    case 'sent':
    case 'pending':
      return 'warning'
    default:
      return 'info'
  }
}

function commandStatusText(status: string) {
  switch (status) {
    case 'pending':
      return '待下发'
    case 'sent':
      return '已下发'
    case 'received':
      return '已收到'
    case 'downloading':
      return '下载中'
    case 'completed':
      return '已完成'
    case 'failed':
      return '失败'
    default:
      return status || '未知'
  }
}

async function load() {
  loading.value = true
  try {
    const res = await listClients()
    clients.value = (res.data || []).sort((a: any, b: any) => a.id - b.id)
  } finally {
    loading.value = false
  }
}

async function handleExpand(row: any, expandedRows: any[]) {
  const expanded = expandedRows.some((r: any) => r.id === row.id)
  if (!expanded) return
  rowLoading.value[row.id] = true
  try {
    const res = await listClientCommands(row.id)
    rowCommands.value[row.id] = res.data || []
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '获取命令记录失败')
  } finally {
    rowLoading.value[row.id] = false
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

function openRename(row: any) {
  editingClientId.value = row.id
  editingName.value = row.name
  renameDialogVisible.value = true
}

async function confirmRename() {
  if (!editingClientId.value || !editingName.value) return
  renameLoading.value = true
  try {
    await updateClientName(editingClientId.value, editingName.value)
    ElMessage.success('名称已修改')
    renameDialogVisible.value = false
    load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '修改失败')
  } finally {
    renameLoading.value = false
  }
}

async function remove(row: any) {
  try {
    await ElMessageBox.confirm('确定删除该客户端记录吗？', '提示', { type: 'warning' })
    await deleteClient(row.id)
    ElMessage.success('删除成功')
    load()
  } catch {
    // cancelled
  }
}

async function handleCommand(row: any, cmd: string) {
  switch (cmd) {
    case 'update-software':
      openUpdateSoftware(row)
      break
    case 'rename':
      openRename(row)
      break
    case 'delete':
      await remove(row)
      break
    default:
      await action(row, cmd)
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

<style scoped>
.text-muted {
  color: #909399;
}
</style>
