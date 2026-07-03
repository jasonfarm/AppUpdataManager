<template>
  <div>
    <h2>软件版本管理</h2>
    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :span="12">
        <el-upload
          :http-request="handleUpload"
          :show-file-list="false"
          accept=".exe"
          action="#"
        >
          <el-button type="primary">上传新版本</el-button>
        </el-upload>
      </el-col>
    </el-row>

    <el-table :data="softwareList" v-loading="loading" border>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="version" label="版本" width="120" />
      <el-table-column prop="filename" label="文件名" />
      <el-table-column label="最新" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.is_latest" type="success">是</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240">
        <template #default="{ row }">
          <el-button size="small" @click="setLatest(row)">设为最新</el-button>
          <el-button size="small" @click="editName(row)">改名</el-button>
          <el-button size="small" type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="nameDialogVisible" title="修改名称" width="400px">
      <el-input v-model="editingName" placeholder="请输入新名称" />
      <template #footer>
        <el-button @click="nameDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveName">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listSoftware, uploadSoftware, deleteSoftware, setLatestSoftware, updateSoftwareName } from '../api'

const loading = ref(false)
const softwareList = ref([] as any[])
const nameDialogVisible = ref(false)
const editingName = ref('')
const editingId = ref(0)

async function load() {
  loading.value = true
  try {
    const res = await listSoftware()
    softwareList.value = res.data
  } finally {
    loading.value = false
  }
}

async function handleUpload(options: any) {
  const file = options.file
  const version = file.name.replace(/\.exe$/i, '')
  const name = version
  const formData = new FormData()
  formData.append('file', file)
  formData.append('name', name)
  formData.append('version', version)
  try {
    await uploadSoftware(formData)
    ElMessage.success('上传成功')
    load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '上传失败')
  }
}

async function setLatest(row: any) {
  try {
    await setLatestSoftware(row.id)
    ElMessage.success('已设为最新版本')
    load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  }
}

async function remove(row: any) {
  try {
    await ElMessageBox.confirm('确定删除该版本吗？', '提示', { type: 'warning' })
    await deleteSoftware(row.id)
    ElMessage.success('删除成功')
    load()
  } catch {
    // cancelled
  }
}

function editName(row: any) {
  editingId.value = row.id
  editingName.value = row.name
  nameDialogVisible.value = true
}

async function saveName() {
  try {
    await updateSoftwareName(editingId.value, editingName.value)
    ElMessage.success('修改成功')
    nameDialogVisible.value = false
    load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '修改失败')
  }
}

onMounted(load)
</script>
