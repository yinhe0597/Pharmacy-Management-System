<template>
  <div>
    <el-card>
      <h3>👋 欢迎，{{ store.user?.name }}</h3>
      <p>
        当前角色：<el-tag>{{ store.roleLabel }}</el-tag>
      </p>
      <p class="tip">后端服务已就绪（Swagger：http://localhost:8080/swagger/index.html）。</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from '@/stores/user'
const store = useUserStore()
</script>

<style scoped>
.tip {
  color: #909399;
  font-size: 13px;
}
</style>
