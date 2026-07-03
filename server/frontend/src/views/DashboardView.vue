<template>
  <el-container class="dashboard">
    <el-aside width="220px" class="sidebar">
      <div class="logo">AppUpdateManager</div>
      <el-menu :default-active="$route.path" router class="menu" background-color="#304156" text-color="#bfcbd9" active-text-color="#409EFF">
        <el-menu-item index="/">
          <span>概览</span>
        </el-menu-item>
        <el-menu-item index="/software">
          <span>软件版本</span>
        </el-menu-item>
        <el-menu-item index="/resource-packages">
          <span>资源包</span>
        </el-menu-item>
        <el-menu-item index="/clients">
          <span>客户端管理</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span>当前用户: {{ auth.username }}</span>
        <el-button type="danger" size="small" @click="logout">退出</el-button>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.dashboard {
  min-height: 100vh;
}
.sidebar {
  background-color: #304156;
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  border-bottom: 1px solid #1f2d3d;
}
.menu {
  border-right: none;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #fff;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}
</style>
