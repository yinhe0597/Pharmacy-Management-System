<template>
  <el-card>
    <el-tabs v-model="tab">
      <el-tab-pane label="用户管理" name="users">
        <div class="toolbar">
          <el-button type="success" @click="openUser">新建用户</el-button>
        </div>
        <el-table :data="users" border>
          <el-table-column prop="username" label="账号" width="140" />
          <el-table-column prop="name" label="姓名" width="120" />
          <el-table-column label="角色" width="130">
            <template #default="{ row }">{{ ROLE_LABELS[row.role as Role] ?? row.role }}</template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }"
              ><el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{
                row.status === 1 ? '启用' : '停用'
              }}</el-tag></template
            >
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="editUser(row)">编辑</el-button>
              <el-button link type="danger" @click="removeUser(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="操作日志" name="logs">
        <div class="toolbar">
          <el-input
            v-model="logKeyword"
            placeholder="关键字"
            clearable
            style="width: 200px"
            @keyup.enter="loadLogs"
          />
          <el-button type="primary" @click="loadLogs">查询</el-button>
        </div>
        <el-table :data="logs" border>
          <el-table-column prop="username" label="用户" width="120" />
          <el-table-column prop="action" label="动作" width="140" />
          <el-table-column prop="resource" label="资源" width="120" />
          <el-table-column prop="path" label="路径" min-width="160" />
          <el-table-column prop="created_at" label="时间" width="170" />
        </el-table>
        <el-pagination
          v-model:current-page="logPage"
          class="pager"
          layout="total, prev, pager, next"
          :total="logTotal"
          :page-size="20"
          @current-change="loadLogs"
        />
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="userVisible" :title="userForm.id ? '编辑用户' : '新建用户'" width="480px">
      <el-form label-width="90px">
        <el-form-item label="账号" required
          ><el-input v-model="userForm.username" :disabled="!!userForm.id"
        /></el-form-item>
        <el-form-item v-if="!userForm.id" label="密码" required
          ><el-input v-model="userForm.password"
        /></el-form-item>
        <el-form-item label="姓名" required><el-input v-model="userForm.name" /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="userForm.role">
            <el-option v-for="(label, key) in ROLE_LABELS" :key="key" :label="label" :value="key" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="userForm.status"
            ><el-option label="启用" :value="1" /><el-option label="停用" :value="0"
          /></el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userVisible = false">取消</el-button>
        <el-button type="primary" @click="saveUser">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listUsers, createUser, updateUser, deleteUser, listOperationLogs } from '@/api/reports'
import { ROLE_LABELS, type Role } from '@/types/business'

const tab = ref('users')
const users = ref<any[]>([])
const logs = ref<any[]>([])
const logTotal = ref(0)
const logPage = ref(1)
const logKeyword = ref('')
const userVisible = ref(false)
const userForm = reactive<Record<string, any>>({})

onMounted(async () => {
  loadUsers()
  loadLogs()
})
async function loadUsers() {
  users.value = (await listUsers({ page: 1, page_size: 200 }))?.list ?? []
}
async function loadLogs() {
  const res = await listOperationLogs({
    keyword: logKeyword.value,
    page: logPage.value,
    page_size: 20,
  })
  logs.value = res?.list ?? []
  logTotal.value = res?.total ?? 0
}
function openUser() {
  Object.keys(userForm).forEach((k) => delete userForm[k])
  userForm.status = 1
  userForm.role = 'doctor'
  userVisible.value = true
}
function editUser(row: any) {
  Object.assign(userForm, {
    id: row.id,
    username: row.username,
    name: row.name,
    role: row.role,
    status: row.status,
  })
  userVisible.value = true
}
async function saveUser() {
  if (userForm.id) await updateUser(userForm.id, userForm)
  else await createUser(userForm)
  ElMessage.success('已保存')
  userVisible.value = false
  loadUsers()
}
async function removeUser(row: any) {
  await deleteUser(row.id)
  ElMessage.success('已删除')
  loadUsers()
}
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
</style>
