<template>
  <el-row :gutter="16">
    <el-col :span="12">
      <el-card header="我的资料">
        <el-descriptions v-loading="loading" :column="1" border>
          <el-descriptions-item label="用户名">{{ me?.username }}</el-descriptions-item>
          <el-descriptions-item label="姓名">{{ me?.name }}</el-descriptions-item>
          <el-descriptions-item label="角色">
            <el-tag size="small">{{ store.roleLabel }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="电话">{{ me?.phone ?? '—' }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="me?.status === 1 ? 'success' : 'danger'" size="small">{{
              me?.status === 1 ? '启用' : '停用'
            }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>
    </el-col>
    <el-col :span="12">
      <el-card header="修改密码">
        <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="90px">
          <el-form-item label="原密码" prop="old_password">
            <el-input v-model="pwdForm.old_password" type="password" show-password />
          </el-form-item>
          <el-form-item label="新密码" prop="new_password">
            <el-input v-model="pwdForm.new_password" type="password" show-password />
            <div class="hint">至少 8 位，含字母与数字</div>
          </el-form-item>
          <el-form-item label="确认密码" prop="confirm">
            <el-input v-model="pwdForm.confirm" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="saving" @click="savePassword">保存新密码</el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { profile, changePassword } from '@/api/auth'
import { useUserStore } from '@/stores/user'

const store = useUserStore()
const loading = ref(false)
const saving = ref(false)
const me = ref<any>(null)
const pwdFormRef = ref<FormInstance>()
const pwdForm = reactive({ old_password: '', new_password: '', confirm: '' })

const pwdRules: FormRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '至少 8 位', trigger: 'blur' },
    {
      validator: (_r, v: string, cb) => {
        if (v && !/[A-Za-z]/.test(v)) cb(new Error('须包含字母'))
        else if (v && !/\d/.test(v)) cb(new Error('须包含数字'))
        else cb()
      },
      trigger: 'blur',
    },
  ],
  confirm: [
    {
      validator: (_r, v: string, cb) => {
        if (v !== pwdForm.new_password) cb(new Error('两次输入不一致'))
        else cb()
      },
      trigger: 'blur',
    },
  ],
}

onMounted(load)
async function load() {
  loading.value = true
  try {
    me.value = await profile()
  } finally {
    loading.value = false
  }
}
async function savePassword() {
  await pwdFormRef.value?.validate()
  saving.value = true
  try {
    await changePassword(pwdForm.old_password, pwdForm.new_password)
    ElMessage.success('密码已修改')
    pwdForm.old_password = ''
    pwdForm.new_password = ''
    pwdForm.confirm = ''
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
