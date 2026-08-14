<template>
  <div>
    <div class="toolbar">
      <el-button type="success" @click="open">新增</el-button>
    </div>
    <el-table :data="list" border>
      <el-table-column
        v-for="c in columns"
        :key="c"
        :prop="c"
        :label="LABELS[c] ?? c"
        min-width="120"
      />
    </el-table>
    <el-pagination
      v-model:current-page="page"
      class="pager"
      layout="total, prev, pager, next"
      :total="total"
      :page-size="20"
      @current-change="load"
    />

    <el-dialog v-model="visible" title="新增" width="480px">
      <el-form label-width="90px">
        <el-form-item v-for="f in fields" :key="f" :label="LABELS[f] ?? f">
          <el-input v-model="form[f]" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  api: (params: Record<string, unknown>) => Promise<any>
  create: (data: Record<string, unknown>) => Promise<any>
  columns: string[]
  fields: string[]
}>()

const LABELS: Record<string, string> = {
  patient_name: '患者',
  question: '咨询内容',
  answer: '药师答复',
  drug_name: '药品',
  reaction: '反应',
  severity: '程度',
  content: '指导内容',
  created_at: '时间',
}

const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const visible = ref(false)
const form = reactive<Record<string, string>>({})

onMounted(load)
async function load() {
  const res = await props.api({ page: page.value, page_size: 20 })
  list.value = res?.list ?? []
  total.value = res?.total ?? 0
}
function open() {
  props.fields.forEach((f) => (form[f] = ''))
  visible.value = true
}
async function save() {
  await props.create({ ...form })
  ElMessage.success('已保存')
  visible.value = false
  load()
}
</script>

<style scoped>
.toolbar {
  margin-bottom: 12px;
}
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
</style>
