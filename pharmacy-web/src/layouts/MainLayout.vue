<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="logo">💊 药房管理系统</div>
      <el-menu
        :default-active="activePath"
        router
        background-color="#001529"
        text-color="rgba(255,255,255,0.7)"
        active-text-color="#fff"
      >
        <el-menu-item v-for="item in visibleMenu" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="header-title">{{ currentTitle }}</div>
        <div class="header-right">
          <el-dropdown trigger="click" @visible-change="onNotifOpen">
            <el-badge :value="unread" :hidden="unread === 0" class="bell">
              <el-icon :size="20"><Bell /></el-icon>
            </el-badge>
            <template #dropdown>
              <div class="notif-panel">
                <div class="notif-head">
                  <b>站内通知</b>
                  <span>
                    <el-button link type="primary" size="small" @click="toggleUnread">{{
                      onlyUnread ? '看全部' : '仅看未读'
                    }}</el-button>
                    <el-button link type="primary" size="small" @click="readAll"
                      >全部已读</el-button
                    >
                  </span>
                </div>
                <div v-if="!notifs.length" class="notif-empty">暂无通知</div>
                <div
                  v-for="n in notifs"
                  :key="n.id"
                  class="notif-item"
                  :class="{ unread: !n.is_read }"
                  @click="readOne(n)"
                >
                  <div class="notif-title">
                    <el-tag :type="levelTag(n.level)" size="small">{{
                      levelLabel(n.level)
                    }}</el-tag>
                    {{ n.title }}
                  </div>
                  <div class="notif-content">{{ n.content }}</div>
                  <div class="notif-time">{{ formatTime(n.created_at) }}</div>
                </div>
              </div>
            </template>
          </el-dropdown>
          <el-dropdown @command="onCommand">
            <span class="user-trigger">
              <el-icon><User /></el-icon>
              {{ store.user?.name }}（{{ store.roleLabel }}）
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">个人中心</el-dropdown-item>
                <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, ArrowDown, Bell } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { MENU } from '@/router/menu'
import { hasPermission, type Permission, type Role } from '@/types/business'
import { listNotifications, unreadCount, markRead, markAllRead } from '@/api/notifications'

const route = useRoute()
const router = useRouter()
const store = useUserStore()

const activePath = computed(() => route.path)
const currentTitle = computed(() => (route.meta.title as string) ?? '')
const visibleMenu = computed(() => {
  const role = store.role as Role
  return MENU.filter(
    (m) => !m.permission || (role && hasPermission(role, m.permission as Permission)),
  )
})

async function onCommand(cmd: string) {
  if (cmd === 'profile') {
    router.push('/profile')
  } else if (cmd === 'logout') {
    store.clear()
    ElMessage.success('已退出登录')
    router.push('/login')
  }
}

// 站内通知：未读轮询 + 下拉已读（预警闭环触达）
const unread = ref(0)
const notifs = ref<any[]>([])
const onlyUnread = ref(false)
let timer: number | undefined

async function refreshUnread() {
  if (!store.user) return
  try {
    const r = await unreadCount()
    unread.value = Number(r?.unread ?? 0)
  } catch {
    /* 静默：避免 token 失效时刷屏 */
  }
}
async function loadNotifs() {
  const r = await listNotifications({
    page: 1,
    page_size: 10,
    unread: onlyUnread.value ? 'true' : undefined,
  })
  notifs.value = r?.list ?? []
}
async function onNotifOpen(open: boolean) {
  if (open) {
    await loadNotifs()
    await refreshUnread()
  }
}
async function toggleUnread() {
  onlyUnread.value = !onlyUnread.value
  await loadNotifs()
}
async function readOne(n: any) {
  if (!n.is_read) {
    await markRead(n.id)
    n.is_read = true
    unread.value = Math.max(0, unread.value - 1)
  }
}
async function readAll() {
  await markAllRead()
  await loadNotifs()
  await refreshUnread()
}
function levelTag(level: string): string {
  return level === 'critical' ? 'danger' : level === 'warning' ? 'warning' : 'info'
}
function levelLabel(level: string): string {
  return level === 'critical' ? '紧急' : level === 'warning' ? '预警' : '通知'
}
function formatTime(s: string): string {
  return String(s ?? '')
    .slice(0, 16)
    .replace('T', ' ')
}
onMounted(() => {
  refreshUnread()
  timer = window.setInterval(refreshUnread, 60000)
})
onUnmounted(() => {
  if (timer !== undefined) window.clearInterval(timer)
})
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
.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}
.bell {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: #606266;
}
.notif-panel {
  width: 340px;
  max-height: 420px;
  overflow-y: auto;
  padding: 8px 12px;
}
.notif-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0 8px;
}
.notif-empty {
  color: #909399;
  text-align: center;
  padding: 24px 0;
}
.notif-item {
  padding: 8px;
  border-top: 1px solid #ebeef5;
  cursor: pointer;
}
.notif-item.unread {
  background: #f0f7ff;
}
.notif-title {
  font-weight: 600;
  display: flex;
  gap: 6px;
  align-items: center;
}
.notif-content {
  color: #606266;
  font-size: 13px;
  margin-top: 4px;
}
.notif-time {
  color: #909399;
  font-size: 12px;
  margin-top: 4px;
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
