<template>
  <el-card>
    <el-tabs v-model="tab" @tab-change="reload">
      <el-tab-pane label="药品配伍禁忌" name="drug" />
      <el-tab-pane label="成分交互" name="ingredient" />
      <el-tab-pane label="分类交互" name="class" />
      <el-tab-pane label="标签交互" name="tag" />
      <el-tab-pane label="药品成分" name="ingredientlib" />
    </el-tabs>

    <!-- 药品配伍禁忌 -->
    <template v-if="tab === 'drug'">
      <div class="toolbar">
        <el-button v-permission="'drug:write'" type="success" @click="openDrugRule"
          >新增配伍禁忌</el-button
        >
      </div>
      <el-table v-loading="loading" :data="drugRules" border>
        <el-table-column prop="drug_a_id" label="药品A ID" width="100" />
        <el-table-column prop="drug_b_id" label="药品B ID" width="100" />
        <el-table-column label="级别" width="90">
          <template #default="{ row }">
            <el-tag :type="LEVEL_TAG[row.level]" size="small">{{ LEVEL_LABEL[row.level] }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="mechanism" label="机制" min-width="120" />
        <el-table-column prop="description" label="说明" min-width="160" />
        <el-table-column prop="evidence_level" label="证据等级" width="100" />
        <el-table-column v-if="canWrite" label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDrugRule(row)">编辑</el-button>
            <el-button link type="danger" @click="removeDrugRule(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-model:current-page="drugPage"
        class="pager"
        layout="total, prev, pager, next"
        :total="drugTotal"
        :page-size="20"
        @current-change="loadDrugRules"
      />
    </template>

    <!-- 成分 / 分类 / 标签交互（同构） -->
    <template v-else-if="tab !== 'ingredientlib'">
      <div class="toolbar">
        <el-button v-permission="'drug:write'" type="success" @click="openRuleTab"
          >新增{{ RULE_TITLE[tab] }}</el-button
        >
      </div>
      <el-table v-loading="loading" :data="ruleRows" border>
        <el-table-column :prop="tab === 'ingredient' ? 'ingredient_a' : tab === 'class' ? 'class_a' : 'tag_a'" :label="`${RULE_KEY[tab]}A`" min-width="120" />
        <el-table-column :prop="tab === 'ingredient' ? 'ingredient_b' : tab === 'class' ? 'class_b' : 'tag_b'" :label="`${RULE_KEY[tab]}B`" min-width="120" />
        <el-table-column label="级别" width="90">
          <template #default="{ row }">
            <el-tag :type="LEVEL_TAG[row.level]" size="small">{{ LEVEL_LABEL[row.level] }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="mechanism" label="机制" min-width="120" />
        <el-table-column prop="description" label="说明" min-width="160" />
        <el-table-column prop="evidence_level" label="证据等级" width="100" />
        <el-table-column v-if="canWrite" label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openRuleTab(row)">编辑</el-button>
            <el-button link type="danger" @click="removeRule(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-model:current-page="rulePage"
        class="pager"
        layout="total, prev, pager, next"
        :total="ruleTotal"
        :page-size="20"
        @current-change="loadRules"
      />
    </template>

    <!-- 药品成分 -->
    <template v-else>
      <div class="toolbar">
        <DrugPicker v-model="ingredientDrugId" style="width: 320px" placeholder="选择药品查看/维护成分" />
      </div>
      <el-table v-if="ingredientDrugId" v-loading="loading" :data="ingredients" border>
        <el-table-column prop="ingredient_name" label="成分名" min-width="160" />
        <el-table-column prop="strength" label="含量/强度" width="140" />
        <el-table-column v-if="canWrite" label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" @click="removeIngredient(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-else description="请先选择药品" />
      <div v-if="ingredientDrugId && canWrite" class="toolbar" style="margin-top: 12px">
        <el-input v-model="ingredientForm.ingredient_name" placeholder="成分名" style="width: 200px" />
        <el-input v-model="ingredientForm.strength" placeholder="含量（可选）" style="width: 160px" />
        <el-button type="success" @click="addIngredient">新增成分</el-button>
      </div>
    </template>
  </el-card>

  <!-- 配伍禁忌（药品对药品）弹窗 -->
  <el-dialog v-model="drugRuleVisible" :title="drugRuleForm.id ? '编辑配伍禁忌' : '新增配伍禁忌'" width="560px">
    <el-form label-width="100px">
      <el-form-item label="药品A" required>
        <DrugPicker v-model="drugRuleForm.drug_a_id" />
      </el-form-item>
      <el-form-item label="药品B" required>
        <DrugPicker v-model="drugRuleForm.drug_b_id" />
      </el-form-item>
      <el-form-item label="级别" required>
        <el-radio-group v-model="drugRuleForm.level">
          <el-radio :value="1">禁忌</el-radio>
          <el-radio :value="2">慎用</el-radio>
          <el-radio :value="3">注意</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="机制"><el-input v-model="drugRuleForm.mechanism" /></el-form-item>
      <el-form-item label="说明"><el-input v-model="drugRuleForm.description" type="textarea" /></el-form-item>
      <el-form-item label="证据等级"><el-input v-model="drugRuleForm.evidence_level" placeholder="A/B/C" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="drugRuleVisible = false">取消</el-button>
      <el-button type="primary" @click="saveDrugRule">保存</el-button>
    </template>
  </el-dialog>

  <!-- 成分/分类/标签规则弹窗 -->
  <el-dialog v-model="ruleVisible" :title="(ruleForm.id ? '编辑' : '新增') + RULE_TITLE[tab]" width="560px">
    <el-form label-width="100px">
      <el-form-item :label="`${RULE_KEY[tab]}A`" required>
        <el-input v-model="ruleForm.a" />
      </el-form-item>
      <el-form-item :label="`${RULE_KEY[tab]}B`" required>
        <el-input v-model="ruleForm.b" />
      </el-form-item>
      <el-form-item label="级别" required>
        <el-radio-group v-model="ruleForm.level">
          <el-radio :value="1">禁忌</el-radio>
          <el-radio :value="2">慎用</el-radio>
          <el-radio :value="3">注意</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="机制"><el-input v-model="ruleForm.mechanism" /></el-form-item>
      <el-form-item label="说明"><el-input v-model="ruleForm.description" type="textarea" /></el-form-item>
      <el-form-item label="证据等级"><el-input v-model="ruleForm.evidence_level" placeholder="A/B/C" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="ruleVisible = false">取消</el-button>
      <el-button type="primary" @click="saveRule">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listInteractions,
  createInteraction,
  updateInteraction,
  deleteInteraction,
  listIngredientInteractions,
  createIngredientInteraction,
  updateIngredientInteraction,
  deleteIngredientInteraction,
  listClassInteractions,
  createClassInteraction,
  updateClassInteraction,
  deleteClassInteraction,
  listTagInteractions,
  createTagInteraction,
  updateTagInteraction,
  deleteTagInteraction,
  listDrugIngredients,
  addDrugIngredient,
  deleteDrugIngredient,
} from '@/api/drugs'
import { hasPermission } from '@/types/business'
import { useUserStore } from '@/stores/user'
import DrugPicker from '@/components/DrugPicker.vue'

const LEVEL_LABEL: Record<number, string> = { 1: '禁忌', 2: '慎用', 3: '注意' }
const LEVEL_TAG: Record<number, string> = { 1: 'danger', 2: 'warning', 3: 'info' }
const RULE_TITLE: Record<string, string> = {
  ingredient: '成分交互',
  class: '分类交互',
  tag: '标签交互',
}
const RULE_KEY: Record<string, string> = { ingredient: '成分', class: '分类', tag: '标签' }

const tab = ref('drug')
const loading = ref(false)
const userStore = useUserStore()
const canWrite = computed(() => userStore.role !== '' && hasPermission(userStore.role as never, 'drug:write'))

const interactionApi = { update: updateInteraction, remove: deleteInteraction }
async function removeDrugRule(row: any) {
  await ElMessageBox.confirm('确认删除该配伍禁忌？', '提示', { type: 'warning' })
  await interactionApi.remove(row.id)
  ElMessage.success('已删除')
  loadDrugRules()
}
const drugRules = ref<any[]>([])
const drugTotal = ref(0)
const drugPage = ref(1)
const drugRuleVisible = ref(false)
const drugRuleForm = reactive<Record<string, any>>({})

const ruleRows = ref<any[]>([])
const ruleTotal = ref(0)
const rulePage = ref(1)
const ruleVisible = ref(false)
const ruleForm = reactive<Record<string, any>>({})

const ingredientDrugId = ref<number | null>(null)
const ingredients = ref<any[]>([])
const ingredientForm = reactive({ ingredient_name: '', strength: '' })

onMounted(loadDrugRules)
watch(tab, (t) => {
  if (t === 'ingredient' || t === 'class' || t === 'tag') loadRules()
})
function reload() {
  if (tab.value === 'drug') loadDrugRules()
  else if (tab.value === 'ingredientlib' && ingredientDrugId.value) loadIngredients()
  else loadRules()
}

// ---- 药品配伍禁忌 ----
async function loadDrugRules() {
  loading.value = true
  try {
    const res = await listInteractions({ page: drugPage.value, page_size: 20 })
    drugRules.value = res?.list ?? []
    drugTotal.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
function openDrugRule(row?: any) {
  Object.assign(drugRuleForm, {
    id: row?.id ?? 0,
    drug_a_id: row?.drug_a_id ?? null,
    drug_b_id: row?.drug_b_id ?? null,
    level: row?.level ?? 1,
    mechanism: row?.mechanism ?? '',
    description: row?.description ?? '',
    evidence_level: row?.evidence_level ?? '',
  })
  drugRuleVisible.value = true
}
async function saveDrugRule() {
  if (!drugRuleForm.drug_a_id || !drugRuleForm.drug_b_id) {
    ElMessage.warning('请选择药品 A / B')
    return
  }
  const payload = { ...drugRuleForm }
  if (payload.id) await updateInteraction(payload.id, payload)
  else {
    delete payload.id
    await createInteraction(payload)
  }
  ElMessage.success('已保存')
  drugRuleVisible.value = false
  loadDrugRules()
}

// ---- 成分/分类/标签规则 ----
function apiFor(t: string) {
  switch (t) {
    case 'ingredient':
      return { list: listIngredientInteractions, create: createIngredientInteraction, update: updateIngredientInteraction, remove: deleteIngredientInteraction }
    case 'class':
      return { list: listClassInteractions, create: createClassInteraction, update: updateClassInteraction, remove: deleteClassInteraction }
    default:
      return { list: listTagInteractions, create: createTagInteraction, update: updateTagInteraction, remove: deleteTagInteraction }
  }
}
async function loadRules() {
  loading.value = true
  try {
    const res = await apiFor(tab.value).list({ page: rulePage.value, page_size: 20 })
    ruleRows.value = res?.list ?? []
    ruleTotal.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
function openRuleTab(row?: any) {
  const a = tab.value === 'ingredient' ? 'ingredient_a' : tab.value === 'class' ? 'class_a' : 'tag_a'
  const b = tab.value === 'ingredient' ? 'ingredient_b' : tab.value === 'class' ? 'class_b' : 'tag_b'
  Object.assign(ruleForm, {
    id: row?.id ?? 0,
    a: row?.[a] ?? '',
    b: row?.[b] ?? '',
    level: row?.level ?? 1,
    mechanism: row?.mechanism ?? '',
    description: row?.description ?? '',
    evidence_level: row?.evidence_level ?? '',
  })
  ruleVisible.value = true
}
async function saveRule() {
  if (!ruleForm.a || !ruleForm.b) {
    ElMessage.warning('请填写 A / B 两侧内容')
    return
  }
  const t = tab.value
  const keyA = t === 'ingredient' ? 'ingredient_a' : t === 'class' ? 'class_a' : 'tag_a'
  const keyB = t === 'ingredient' ? 'ingredient_b' : t === 'class' ? 'class_b' : 'tag_b'
  const payload: Record<string, unknown> = {
    [keyA]: ruleForm.a,
    [keyB]: ruleForm.b,
    level: ruleForm.level,
    mechanism: ruleForm.mechanism,
    description: ruleForm.description,
    evidence_level: ruleForm.evidence_level,
  }
  const api = apiFor(t)
  if (ruleForm.id) await api.update(ruleForm.id, payload)
  else await api.create(payload)
  ElMessage.success('已保存')
  ruleVisible.value = false
  loadRules()
}
async function removeRule(row: any) {
  await ElMessageBox.confirm('确认删除该交互规则？', '提示', { type: 'warning' })
  await apiFor(tab.value).remove(row.id)
  ElMessage.success('已删除')
  loadRules()
}

// ---- 药品成分 ----
async function loadIngredients() {
  if (!ingredientDrugId.value) return
  loading.value = true
  try {
    ingredients.value = (await listDrugIngredients(ingredientDrugId.value)) ?? []
  } finally {
    loading.value = false
  }
}
watch(ingredientDrugId, loadIngredients)
async function addIngredient() {
  if (!ingredientForm.ingredient_name) {
    ElMessage.warning('请填写成分名')
    return
  }
  await addDrugIngredient(ingredientDrugId.value!, { ...ingredientForm })
  ElMessage.success('已新增')
  ingredientForm.ingredient_name = ''
  ingredientForm.strength = ''
  loadIngredients()
}
async function removeIngredient(row: any) {
  await ElMessageBox.confirm('确认删除该成分？', '提示', { type: 'warning' })
  await deleteDrugIngredient(row.id)
  ElMessage.success('已删除')
  loadIngredients()
}
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  align-items: center;
}
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
</style>
