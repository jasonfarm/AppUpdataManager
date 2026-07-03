<template>
  <div>
    <h2>资源包管理</h2>
    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :span="12">
        <el-upload
          :http-request="handleUpload"
          :show-file-list="false"
          accept=".zip"
          action="#"
        >
          <el-button type="primary">上传新资源包</el-button>
        </el-upload>
      </el-col>
    </el-row>

    <el-table :data="resourceList" v-loading="loading" border>
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
import {
  listResourcePackages,
  uploadResourcePackage,
  deleteResourcePackage,
  setLatestResourcePackage,
  updateResourcePackageName,
} from '../api'

const loading = ref(false)
const resourceList = ref([] as any[])
const nameDialogVisible = ref(false)
const editingName = ref('')
const editingId = ref(0)

async function load() {
  loading.value = true
  try {
    const res = await listResourcePackages()
    resourceList.value = res.data
  } finally {
    loading.value = false
  }
}

async function handleUpload(options: any) {
  const file = options.file
  const version = file.name.replace(/\.zip$/i, '')
  const name = version
  const formData = new FormData()
  formData.append('file', file)
  formData.append('name', name)
  formData.append('version', version)
  try {
    await uploadResourcePackage(formData)
    ElMessage.success('上传成功')
    load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '上传失败')
  }
}

async function setLatest(row: any) {
  try {
    await setLatestResourcePackage(row.id)
    ElMessage.success('已设为最新资源包')
    load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  }
}

async function remove(row: any) {
  try {
    await ElMessageBox.confirm('确定删除该资源包吗？', '提示', { type: 'warning' })
    await deleteResourcePackage(row.id)
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
    await updateResourcePackageName(editingId.value, editingName.value)
    ElMessage.success('修改成功')
    nameDialogVisible.value = false
    load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '修改失败')
  }
}

onMounted(load)
</script>
