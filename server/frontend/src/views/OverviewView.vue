<template>
  <div>
    <h2>概览</h2>
    <el-row :gutter="20">
      <el-col :span="8">
        <el-statistic title="在线客户端" :value="clientCount" />
      </el-col>
      <el-col :span="8">
        <el-statistic title="软件版本数" :value="softwareCount" />
      </el-col>
      <el-col :span="8">
        <el-statistic title="客户端版本数" :value="clientVersionCount" />
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listClients, listSoftware, listClientVersions } from '../api'

const clientCount = ref(0)
const softwareCount = ref(0)
const clientVersionCount = ref(0)

onMounted(async () => {
  try {
    const [clients, software, clientVersions] = await Promise.all([
      listClients(),
      listSoftware(),
      listClientVersions(),
    ])
    clientCount.value = clients.data.length
    softwareCount.value = software.data.length
    clientVersionCount.value = clientVersions.data.length
  } catch (e) {
    console.error(e)
  }
})
</script>
