<template>
  <el-card>
    <el-tabs v-model="tab">
      <!-- 用户管理：新建/编辑各类人员账号，编辑时可重置密码 -->
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
          <el-table-column prop="phone" label="电话" width="130" />
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

      <!-- 操作日志：查看所有人日志，支持用户/动作/时间筛选 -->
      <el-tab-pane label="操作日志" name="logs">
        <div class="toolbar">
          <el-input
            v-model="logUser"
            placeholder="用户ID"
            clearable
            style="width: 100px"
            @keyup.enter="loadLogs"
          />
          <el-select v-model="logAction" placeholder="动作" clearable style="width: 130px">
            <el-option v-for="a in LOG_ACTIONS" :key="a" :label="a" :value="a" />
          </el-select>
          <el-date-picker
            v-model="logRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            start-placeholder="开始"
            end-placeholder="结束"
          />
          <el-input
            v-model="logKeyword"
            placeholder="关键字"
            clearable
            style="width: 160px"
            @keyup.enter="loadLogs"
          />
          <el-button type="primary" @click="loadLogs">查询</el-button>
        </div>
        <el-table :data="logs" border>
          <el-table-column prop="username" label="用户" width="110" />
          <el-table-column prop="user_role" label="角色" width="110" />
          <el-table-column prop="action" label="动作" width="120" />
          <el-table-column prop="resource" label="资源" width="100" />
          <el-table-column prop="path" label="路径" min-width="150" />
          <el-table-column prop="detail" label="详情" min-width="140" show-overflow-tooltip />
          <el-table-column prop="ip" label="IP" width="120" />
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

      <!-- 库房管理：药房库房台账（新建/编辑） -->
      <el-tab-pane label="库房管理" name="locations">
        <div class="toolbar">
          <el-button type="success" @click="openLocation">新建库房</el-button>
        </div>
        <el-table :data="locations" border>
          <el-table-column prop="code" label="编码" width="120" />
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column label="类型" width="110">
            <template #default="{ row }">{{ LOCATION_TYPES[row.type] ?? row.type }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"
              ><el-tag :type="row.is_active ? 'success' : 'info'" size="small">{{
                row.is_active ? '启用' : '停用'
              }}</el-tag></template
            >
          </el-table-column>
          <el-table-column label="操作" width="90" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openLocation(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 系统设置：自定义默认诊费（合并结算自动带入） -->
      <el-tab-pane label="系统设置" name="settings">
        <div class="toolbar">
          <el-button type="primary" @click="saveSettings">保存诊费配置</el-button>
        </div>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="默认挂号费（分）">
            <el-input-number v-model="feeRegistration" :min="0" :step="100" />
          </el-descriptions-item>
          <el-descriptions-item label="默认诊查费（分）">
            <el-input-number v-model="feeConsultation" :min="0" :step="100" />
          </el-descriptions-item>
          <el-descriptions-item label="说明">
            结算就诊费用时，若就诊尚无对应费用且金额 &gt; 0，将自动把挂号费/诊查费带入合并结算单。
          </el-descriptions-item>
        </el-descriptions>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="userVisible" :title="userForm.id ? '编辑用户' : '新建用户'" width="480px">
      <el-form ref="userFormRef" :model="userForm" :rules="userRules" label-width="90px">
        <el-form-item label="账号" prop="username"
          ><el-input v-model="userForm.username" :disabled="!!userForm.id"
        /></el-form-item>
        <el-form-item :label="userForm.id ? '重置密码' : '密码'" prop="password">
          <el-input
            v-model="userForm.password"
            type="password"
            show-password
            :placeholder="userForm.id ? '留空则不修改' : '初始密码'"
          />
        </el-form-item>
        <el-form-item label="姓名" prop="name"><el-input v-model="userForm.name" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="userForm.phone" /></el-form-item>
        <el-form-item label="角色" prop="role">
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
    <el-dialog
      v-model="locationVisible"
      :title="locationForm.id ? '编辑库房' : '新建库房'"
      width="440px"
    >
      <el-form label-width="80px">
        <el-form-item label="编码" required
          ><el-input v-model="locationForm.code" :disabled="!!locationForm.id"
        /></el-form-item>
        <el-form-item label="名称" required><el-input v-model="locationForm.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="locationForm.type">
            <el-option
              v-for="(label, key) in LOCATION_TYPES"
              :key="key"
              :label="label"
              :value="Number(key)"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="启用"><el-switch v-model="locationForm.is_active" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="locationVisible = false">取消</el-button>
        <el-button type="primary" @click="saveLocation">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import {
  listUsers,
  createUser,
  updateUser,
  deleteUser,
  listOperationLogs,
  listSystemSettings,
  updateSystemSetting,
} from '@/api/reports'
import { ROLE_LABELS, type Role } from '@/types/business'
import { listLocations, createLocation, updateLocation } from '@/api/inventory'

const LOCATION_TYPES: Record<number, string> = {
  1: '药库',
  2: '药房',
  3: '科室',
  4: '其他',
}

const tab = ref('users')
const users = ref<any[]>([])
const logs = ref<any[]>([])
const logTotal = ref(0)
const logPage = ref(1)
const logUser = ref('')
const logAction = ref('')
const logKeyword = ref('')
const logRange = ref<[string, string] | null>(null)
const LOG_ACTIONS = ['login', 'create', 'update', 'delete', 'change_password', 'logout']

const userVisible = ref(false)
const userForm = reactive<Record<string, any>>({})
const userFormRef = ref<FormInstance>()
const userRules: FormRules = {
  username: [
    { required: true, message: '请输入账号', trigger: 'blur' },
    { min: 3, message: '至少 3 位', trigger: 'blur' },
  ],
  password: [
    {
      validator: (_r, v: string, cb) => {
        if (!userForm.id && !v) cb(new Error('请输入初始密码'))
        else if (v && v.length < 8) cb(new Error('至少 8 位'))
        else if (v && !(/[A-Za-z]/.test(v) && /\d/.test(v))) cb(new Error('须同时包含字母与数字'))
        else cb()
      },
      trigger: 'blur',
    },
  ],
  name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
}

const feeRegistration = ref(0)
const feeConsultation = ref(0)
const locations = ref<any[]>([])
const locationVisible = ref(false)
const locationForm = reactive<Record<string, any>>({ code: '', name: '', type: 2, is_active: true })

onMounted(async () => {
  loadUsers()
  loadLogs()
  loadSettings()
  loadLocations()
})
async function loadLocations() {
  locations.value = (await listLocations()) ?? []
}
function openLocation(row?: any) {
  Object.assign(locationForm, {
    id: row?.id ?? 0,
    code: row?.code ?? '',
    name: row?.name ?? '',
    type: row?.type ?? 2,
    is_active: row?.is_active ?? true,
  })
  locationVisible.value = true
}
async function saveLocation() {
  if (!locationForm.code || !locationForm.name) {
    ElMessage.warning('请填写编码与名称')
    return
  }
  if (locationForm.id) await updateLocation(locationForm.id, locationForm)
  else await createLocation(locationForm)
  ElMessage.success('已保存')
  locationVisible.value = false
  loadLocations()
}
async function loadUsers() {
  users.value = (await listUsers({ page: 1, page_size: 200 }))?.list ?? []
}
async function loadLogs() {
  const params: Record<string, unknown> = { page: logPage.value, page_size: 20 }
  if (logUser.value) params.user_id = logUser.value
  if (logAction.value) params.action = logAction.value
  if (logKeyword.value) params.keyword = logKeyword.value
  if (logRange.value) {
    params.start = `${logRange.value[0]}T00:00:00+08:00`
    params.end = `${logRange.value[1]}T23:59:59+08:00`
  }
  const res = await listOperationLogs(params)
  logs.value = res?.list ?? []
  logTotal.value = res?.total ?? 0
}
async function loadSettings() {
  const list = (await listSystemSettings()) ?? []
  for (const s of list) {
    if (s.key === 'default_registration_fee') feeRegistration.value = Number(s.value) || 0
    if (s.key === 'default_consultation_fee') feeConsultation.value = Number(s.value) || 0
  }
}
async function saveSettings() {
  await updateSystemSetting('default_registration_fee', String(feeRegistration.value))
  await updateSystemSetting('default_consultation_fee', String(feeConsultation.value))
  ElMessage.success('诊费配置已保存')
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
    phone: row.phone,
    role: row.role,
    status: row.status,
  })
  userVisible.value = true
}
async function saveUser() {
  await userFormRef.value?.validate()
  const payload: Record<string, any> = {
    name: userForm.name,
    role: userForm.role,
    phone: userForm.phone ?? '',
    status: userForm.status,
  }
  if (userForm.password) payload.password = userForm.password
  if (userForm.id) await updateUser(userForm.id, payload)
  else await createUser({ username: userForm.username, password: userForm.password, ...payload })
  ElMessage.success('已保存')
  userVisible.value = false
  loadUsers()
}
async function removeUser(row: any) {
  await ElMessageBox.confirm(`确认删除用户 ${row.username}？`, '删除')
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
  flex-wrap: wrap;
}
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
</style>
