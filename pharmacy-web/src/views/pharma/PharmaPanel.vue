<template>
  <div>
    <div class="toolbar">
      <el-button v-permission="'inventory:write'" type="success" @click="open">新增</el-button>
    </div>
    <el-table :data="list" border>
      <el-table-column
        v-for="c in columns"
        :key="c"
        :prop="c"
        :label="LABELS[c] ?? c"
        min-width="120"
      />
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button v-permission="'inventory:write'" link type="primary" @click="open(row)"
            >编辑</el-button
          >
          <el-button v-permission="'inventory:write'" link type="danger" @click="remove(row)"
            >删除</el-button
          >
        </template>
      </el-table-column>
    </el-table>
    <el-pagination
      v-model:current-page="page"
      class="pager"
      layout="total, prev, pager, next"
      :total="total"
      :page-size="20"
      @current-change="load"
    />

    <el-dialog v-model="visible" :title="form.id ? '编辑' : '新增'" width="480px">
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
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps<{
  api: (params: Record<string, unknown>) => Promise<any>
  create: (data: Record<string, unknown>) => Promise<any>
  update?: (id: number, data: Record<string, unknown>) => Promise<any>
  remove?: (id: number) => Promise<any>
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
const form = reactive<Record<string, any>>({})

onMounted(load)
async function load() {
  const res = await props.api({ page: page.value, page_size: 20 })
  list.value = res?.list ?? []
  total.value = res?.total ?? 0
}
function open(row?: any) {
  props.fields.forEach((f) => (form[f] = row?.[f] ?? ''))
  form.id = row?.id ?? 0
  visible.value = true
}
async function save() {
  if (form.id && props.update) {
    const { id, ...payload } = form
    await props.update(id, payload)
  } else {
    const { id, ...payload } = form
    await props.create(payload)
  }
  ElMessage.success('已保存')
  visible.value = false
  load()
}
async function remove(row: any) {
  if (!props.remove) return
  await ElMessageBox.confirm('确认删除该记录？', '提示', { type: 'warning' })
  await props.remove(row.id)
  ElMessage.success('已删除')
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
