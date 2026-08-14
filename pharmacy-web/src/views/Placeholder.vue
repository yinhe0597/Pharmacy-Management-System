<template>
  <el-empty :description="`「${title}」模块开发中（P1-P5 阶段交付）`" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
const route = useRoute()
const title = computed(() => (route.meta.title as string) ?? '本')
</script>
