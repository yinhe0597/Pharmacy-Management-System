<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="logo">💊 药房管理系统</div>
      <el-menu :default-active="activePath" router background-color="#001529" text-color="#rgba(255,255,255,0.7)" active-text-color="#fff">
        <el-menu-item v-for="item in visibleMenu" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="header-title">{{ currentTitle }}</div>
        <el-dropdown @command="onCommand">
          <span class="user-trigger">
            <el-icon><User /></el-icon>
            {{ store.user?.name }}（{{ store.roleLabel }}）
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, ArrowDown } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { MENU } from '@/router/menu'
import { hasPermission, type Permission, type Role } from '@/types/business'

const route = useRoute()
const router = useRouter()
const store = useUserStore()

const activePath = computed(() => route.path)
const currentTitle = computed(() => (route.meta.title as string) ?? '')
const visibleMenu = computed(() => {
  const role = store.role as Role
  return MENU.filter((m) => !m.permission || (role && hasPermission(role, m.permission as Permission)))
})

async function onCommand(cmd: string) {
  if (cmd === 'logout') {
    store.clear()
    ElMessage.success('已退出登录')
    router.push('/login')
  }
}
</script>

<style scoped>
.layout {
  height: 100vh;
}
.aside {
  background: #001529;
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
}
.aside :deep(.el-menu) {
  border-right: none;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #e4e7ed;
  background: #fff;
}
.header-title {
  font-size: 16px;
  font-weight: 600;
}
.user-trigger {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  color: #303133;
}
.main {
  background: #f5f7fa;
}
</style>
