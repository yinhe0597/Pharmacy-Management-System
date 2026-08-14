<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <div class="login-header">
          <h2>💊 药房管理系统</h2>
          <p>药房进销存与处方调配</p>
        </div>
      </template>
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top" @keyup.enter="onSubmit">
        <el-form-item label="账号" prop="username">
          <el-input v-model="form.username" placeholder="请输入账号" :prefix-icon="User" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" :prefix-icon="Lock" />
        </el-form-item>
        <el-button type="primary" class="login-btn" :loading="loading" @click="onSubmit">登 录</el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, type FormInstance } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { login } from '@/api/auth'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const store = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function onSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      const result = await login(form.username, form.password)
      store.setLogin(result)
      ElMessage.success(`欢迎，${result.user.name}`)
      const redirect = (route.query.redirect as string) || '/'
      router.push(redirect)
    } catch {
      // 错误提示已在拦截器处理
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #e0f2f1 0%, #bbdefb 100%);
}
.login-card {
  width: 380px;
}
.login-header {
  text-align: center;
}
.login-header h2 {
  margin: 0 0 4px;
  color: #2c3e50;
}
.login-header p {
  margin: 0;
  color: #909399;
  font-size: 13px;
}
.login-btn {
  width: 100%;
}
</style>
