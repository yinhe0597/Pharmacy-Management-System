<template>
  <el-card>
    <el-tabs v-model="tab">
      <el-tab-pane label="库存列表" name="stock">
        <div class="toolbar">
          <el-input
            v-model="query.keyword"
            placeholder="药品名称"
            clearable
            style="width: 220px"
            @keyup.enter="loadStock"
          />
          <el-button type="primary" @click="loadStock">查询</el-button>
          <el-button v-permission="'inventory:write'" type="success" @click="openStockIn"
            >其他入库</el-button
          >
        </div>
        <el-table v-loading="loading" :data="stock" border>
          <el-table-column prop="drug_name" label="药品" min-width="140" />
          <el-table-column prop="batch_no" label="批号" width="120" />
          <el-table-column prop="location_name" label="库房" width="100" />
          <el-table-column prop="expiry_date" label="效期" width="110" />
          <el-table-column label="口径" width="70">
            <template #default="{ row }">{{ row.is_split ? '拆零' : '整盒' }}</template>
          </el-table-column>
          <el-table-column prop="quantity" label="数量" width="80" />
          <el-table-column prop="reserved_quantity" label="已预占" width="80" />
          <el-table-column v-if="canWrite" label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openTransfer(row)">调拨</el-button>
              <el-button link type="warning" @click="openAdjust(row)">调整</el-button>
              <el-button v-if="!row.is_split" link type="success" @click="openSplitUnits(row)"
                >按片拆零</el-button
              >
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          v-model:current-page="query.page"
          class="pager"
          layout="total, prev, pager, next"
          :total="stockTotal"
          :page-size="20"
          @current-change="loadStock"
        />
      </el-tab-pane>

      <el-tab-pane label="领用/补发登记单" name="requisition">
        <div class="toolbar">
          <el-button v-permission="'inventory:write'" type="success" @click="openRequisition"
            >新建补发登记单</el-button
          >
        </div>
        <el-table v-loading="loading" :data="orders" border>
          <el-table-column prop="requisition_no" label="单号" width="150" />
          <el-table-column prop="purpose" label="目的" width="100">
            <template #default="{ row }">{{ PURPOSE[row.purpose] ?? row.purpose }}</template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="140" />
          <el-table-column prop="operator_name" label="操作人" width="100" />
          <el-table-column prop="created_at" label="时间" width="170" />
        </el-table>
        <el-pagination
          v-model:current-page="orderPage"
          class="pager"
          layout="total, prev, pager, next"
          :total="orderTotal"
          :page-size="20"
          @current-change="loadOrders"
        />
      </el-tab-pane>

      <el-tab-pane label="盘点管理" name="stocktake">
        <div class="toolbar">
          <el-button v-permission="'inventory:write'" type="success" @click="openStocktake"
            >新建盘点单</el-button
          >
          <span class="hint"
            >流程：新建（冻结库存变动）→ 开始盘点 → 录实盘 → 差异调整入账 → 归档完成</span
          >
        </div>
        <el-table v-loading="loading" :data="stocktakes" border>
          <el-table-column prop="stocktake_no" label="盘点单号" width="160" />
          <el-table-column prop="location_id" label="库房" width="100">
            <template #default="{ row }">{{ locationName(row.location_id) }}</template>
          </el-table-column>
          <el-table-column label="类型" width="90">
            <template #default="{ row }">{{ row.type === 1 ? '周期' : '动态' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="ST_TAKE_TAG[row.status] ?? 'info'" size="small">{{
                ST_TAKE_LABEL[row.status] ?? row.status
              }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="started_at" label="开始时间" width="170" />
          <el-table-column prop="completed_at" label="完成时间" width="170" />
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openStocktakeDetail(row)"
                >详情/录实盘</el-button
              >
              <el-button
                v-if="row.status === 'draft'"
                v-permission="'inventory:write'"
                link
                type="success"
                @click="doStartStocktake(row)"
                >开始</el-button
              >
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          v-model:current-page="stTakePage"
          class="pager"
          layout="total, prev, pager, next"
          :total="stTakeTotal"
          :page-size="20"
          @current-change="loadStocktakes"
        />
      </el-tab-pane>

      <el-tab-pane label="拆零操作单" name="splitorders">
        <el-table v-loading="loading" :data="splitOrders" border>
          <el-table-column prop="id" label="单号" width="80" />
          <el-table-column prop="drug_id" label="药品ID" width="90" />
          <el-table-column prop="batch_no" label="批号" width="120" />
          <el-table-column prop="boxes" label="拆盒数" width="90" />
          <el-table-column prop="units_in" label="入片数" width="90" />
          <el-table-column prop="damaged" label="破损" width="80" />
          <el-table-column prop="reviewer_name" label="复核人" width="100" />
          <el-table-column prop="created_at" label="时间" width="170" />
        </el-table>
        <el-pagination
          v-model:current-page="splitPage"
          class="pager"
          layout="total, prev, pager, next"
          :total="splitTotal"
          :page-size="20"
          @current-change="loadSplitOrders"
        />
      </el-tab-pane>

      <el-tab-pane label="库存流水" name="transactions">
        <div class="toolbar">
          <el-select
            v-model="txnQuery.txn_type"
            clearable
            placeholder="全部类型"
            style="width: 150px"
          >
            <el-option v-for="(label, key) in TXN_TYPES" :key="key" :label="label" :value="key" />
          </el-select>
          <el-select
            v-model="txnQuery.location_id"
            clearable
            placeholder="全部库房"
            style="width: 150px"
          >
            <el-option v-for="l in locations" :key="l.id" :label="l.name" :value="l.id" />
          </el-select>
          <el-button type="primary" @click="loadTransactions">查询</el-button>
        </div>
        <el-table v-loading="loading" :data="transactions" border>
          <el-table-column prop="transaction_no" label="流水号" width="150" />
          <el-table-column prop="drug_id" label="药品ID" width="80" />
          <el-table-column prop="batch_no" label="批号" width="110" />
          <el-table-column prop="txn_type" label="类型" width="100">
            <template #default="{ row }">{{ TXN_TYPES[row.txn_type] ?? row.txn_type }}</template>
          </el-table-column>
          <el-table-column prop="quantity" label="数量" width="80" />
          <el-table-column label="变前/变后" width="110">
            <template #default="{ row }"
              >{{ row.before_quantity }} / {{ row.after_quantity }}</template
            >
          </el-table-column>
          <el-table-column prop="remarks" label="备注" min-width="120" />
          <el-table-column prop="operator_name" label="操作人" width="100" />
          <el-table-column prop="created_at" label="时间" width="170" />
        </el-table>
        <el-pagination
          v-model:current-page="txnQuery.page"
          class="pager"
          layout="total, prev, pager, next"
          :total="txnTotal"
          :page-size="20"
          @current-change="loadTransactions"
        />
      </el-tab-pane>

      <el-tab-pane label="库存预警" name="alerts">
        <div class="toolbar">
          <el-radio-group v-model="alertKind" @change="loadAlerts">
            <el-radio-button value="below">库存下限</el-radio-button>
            <el-radio-button value="expiry">效期预警</el-radio-button>
          </el-radio-group>
        </div>
        <el-table v-if="alertKind === 'below'" v-loading="loading" :data="alerts" border>
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="drug_id" label="药品ID" width="80" />
          <el-table-column prop="message" label="预警信息" min-width="200" />
          <el-table-column prop="quantity" label="当前库存" width="90" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="created_at" label="时间" width="170" />
          <el-table-column v-if="canWrite" label="处置" width="150" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="resolve(row, 'resolved')">已处理</el-button>
              <el-button link type="info" @click="resolve(row, 'ignored')">忽略</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-table v-else v-loading="loading" :data="expiry" border>
          <el-table-column prop="drug_id" label="药品ID" width="90" />
          <el-table-column prop="drug_name" label="药品" min-width="140" />
          <el-table-column prop="batch_no" label="批号" width="120" />
          <el-table-column prop="expiry_date" label="到期日" width="120" />
          <el-table-column prop="quantity" label="库存" width="90" />
          <el-table-column prop="days_left" label="剩余天数" width="100" />
          <el-table-column prop="location_name" label="库房" width="110" />
        </el-table>
        <el-pagination
          v-model:current-page="alertPage"
          class="pager"
          layout="total, prev, pager, next"
          :total="alertTotal"
          :page-size="20"
          @current-change="loadAlerts"
        />
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="reqVisible" title="新建补发登记单" width="640px">
      <el-form label-width="90px">
        <el-form-item label="库房" required>
          <el-select v-model="reqForm.location_id" placeholder="选择库房" style="width: 220px">
            <el-option v-for="l in locations" :key="l.id" :label="l.name" :value="l.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="目的">
          <el-select v-model="reqForm.purpose">
            <el-option label="补发" value="supplement" />
            <el-option label="临床领用" value="clinical" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="原因"><el-input v-model="reqForm.reason" /></el-form-item>
        <el-form-item label="明细">
          <div v-for="(it, idx) in reqForm.items" :key="idx" class="item-row">
            <DrugPicker v-model="it.drug_id" style="flex: 1" />
            <el-input-number v-model="it.quantity" :min="1" placeholder="数量(LDU)" />
            <el-button link type="danger" @click="reqForm.items.splice(idx, 1)">删</el-button>
          </div>
          <el-button link type="primary" @click="reqForm.items.push({ drug_id: null, quantity: 1 })"
            >+ 添加明细</el-button
          >
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reqVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRequisition">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="transferVisible" title="库存调拨" width="560px">
      <el-form label-width="90px">
        <el-form-item label="药品批次">
          <span>{{ transferForm.drug_name }}（{{ transferForm.batch_no }}）</span>
        </el-form-item>
        <el-form-item label="转出库房" required>
          <el-select v-model="transferForm.from_location_id" disabled style="width: 220px">
            <el-option v-for="l in locations" :key="l.id" :label="l.name" :value="l.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="转入库房" required>
          <el-select
            v-model="transferForm.to_location_id"
            placeholder="选择库房"
            style="width: 220px"
          >
            <el-option v-for="l in locations" :key="l.id" :label="l.name" :value="l.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量" required>
          <el-input-number v-model="transferForm.quantity" :min="1" />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="transferForm.remarks" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="transferVisible = false">取消</el-button>
        <el-button type="primary" @click="saveTransfer">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="adjustVisible" title="库存调整（报损/盘盈）" width="520px">
      <el-form label-width="90px">
        <el-form-item label="药品批次">
          <span
            >{{ adjustForm.drug_name }}（{{ adjustForm.batch_no }}，当前
            {{ adjustForm.current }}）</span
          >
        </el-form-item>
        <el-form-item label="调整数量" required>
          <el-input-number v-model="adjustForm.quantity" />
          <div class="hint">正数 = 盘盈入库，负数 = 报损/盘亏出库</div>
        </el-form-item>
        <el-form-item label="原因" required><el-input v-model="adjustForm.reason" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adjustVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAdjust">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="splitUnitsVisible" title="按片拆零（整盒→片）" width="560px">
      <el-form label-width="110px">
        <el-form-item label="药品批次">
          <span
            >{{ splitUnitsForm.drug_name }}（{{ splitUnitsForm.batch_no }}，库存
            {{ splitUnitsForm.quantity }}）</span
          >
        </el-form-item>
        <el-form-item label="拆出盒数" required>
          <el-input-number v-model="splitUnitsForm.boxes" :min="1" />
        </el-form-item>
        <el-form-item label="实际入片数" required>
          <el-input-number v-model="splitUnitsForm.units" :min="0" />
          <div class="hint">账目平齐：入片数 + 破损 = 拆盒数 × 包装含量</div>
        </el-form-item>
        <el-form-item label="破损片数">
          <el-input-number v-model="splitUnitsForm.damaged" :min="0" />
        </el-form-item>
        <el-form-item label="复核人" required>
          <el-select
            v-model="splitUnitsForm.reviewer_id"
            filterable
            placeholder="选择复核人（须 ≠ 操作人）"
          >
            <el-option
              v-for="u in users"
              :key="u.id"
              :label="`${u.name}（${u.username}）`"
              :value="u.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="splitUnitsVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSplitUnits">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="stockInVisible" title="其他入库" width="720px">
      <el-form label-width="90px">
        <el-form-item label="明细">
          <div v-for="(it, idx) in stockInForm.entries" :key="idx" class="item-row">
            <DrugPicker v-model="it.drug_id" style="flex: 1" />
            <el-select v-model="it.location_id" placeholder="库房" style="width: 130px">
              <el-option v-for="l in locations" :key="l.id" :label="l.name" :value="l.id" />
            </el-select>
            <el-input v-model="it.batch_no" placeholder="批号" style="width: 110px" />
            <el-input v-model="it.expiry_date" placeholder="效期 YYYY-MM-DD" style="width: 140px" />
            <el-input-number
              v-model="it.quantity"
              :min="1"
              placeholder="数量"
              style="width: 110px"
            />
            <el-button link type="danger" @click="stockInForm.entries.splice(idx, 1)">删</el-button>
          </div>
          <el-button link type="primary" @click="stockInForm.entries.push(newStockInEntry())"
            >+ 添加明细</el-button
          >
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="stockInVisible = false">取消</el-button>
        <el-button type="primary" @click="saveStockIn">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="stTakeCreateVisible" title="新建盘点单" width="440px">
      <el-form label-width="90px">
        <el-form-item label="库房" required>
          <el-select v-model="stTakeForm.location_id" placeholder="选择库房">
            <el-option v-for="l in locations" :key="l.id" :label="l.name" :value="l.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型" required>
          <el-radio-group v-model="stTakeForm.type">
            <el-radio :value="1">周期盘点</el-radio>
            <el-radio :value="2">动态盘点</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="stTakeCreateVisible = false">取消</el-button>
        <el-button type="primary" @click="saveStocktake">创建并冻结库存</el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="stTakeDetailVisible"
      :title="`盘点单 ${stTakeDetail?.stocktake_no ?? ''}`"
      size="720px"
    >
      <template v-if="stTakeDetail">
        <el-descriptions :column="3" border size="small" style="margin-bottom: 12px">
          <el-descriptions-item label="库房">{{
            locationName(stTakeDetail.location_id)
          }}</el-descriptions-item>
          <el-descriptions-item label="类型">{{
            stTakeDetail.type === 1 ? '周期' : '动态'
          }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{
            ST_TAKE_LABEL[stTakeDetail.status] ?? stTakeDetail.status
          }}</el-descriptions-item>
        </el-descriptions>
        <div class="toolbar">
          <el-button
            v-if="stTakeDetail.status === 'counting' && canWrite"
            type="primary"
            size="small"
            @click="saveCounted"
            >保存实盘录入</el-button
          >
          <el-button size="small" @click="printStocktake">打印盘点单</el-button>
          <el-button
            v-if="stTakeDetail.status === 'counting' && canWrite"
            type="warning"
            size="small"
            @click="doAdjustStocktake"
            >确认差异调整</el-button
          >
          <el-button
            v-if="stTakeDetail.status === 'adjusted' && canWrite"
            type="success"
            size="small"
            @click="doCompleteStocktake"
            >归档完成</el-button
          >
        </div>
        <el-table :data="stTakeDetail.items ?? []" border size="small">
          <el-table-column prop="drug_name" label="药品" min-width="130" />
          <el-table-column prop="batch_no" label="批号" width="100" />
          <el-table-column label="口径" width="60">
            <template #default="{ row }">{{ row.is_split ? '拆零' : '整盒' }}</template>
          </el-table-column>
          <el-table-column prop="book_quantity" label="账面" width="70" />
          <el-table-column label="实盘" width="120">
            <template #default="{ row }">
              <el-input-number
                v-if="stTakeDetail.status === 'counting' && canWrite"
                v-model="counted[row.id]"
                :min="0"
                size="small"
              />
              <span v-else>{{ row.counted_quantity ?? '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="差异" width="70">
            <template #default="{ row }">{{
              row.counted_quantity != null ? row.counted_quantity - row.book_quantity : '—'
            }}</template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>
  </el-card>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { printDocument, escapeHtml } from '@/utils/print'
import {
  listInventory,
  listLocations,
  listRequisitionOrders,
  createRequisitionOrder,
  transfer,
  adjust,
  stockIn,
  listTransactions,
  listAlerts,
  resolveAlert,
  listExpiryWarnings,
  splitUnits,
  listSplitOrders,
  createStocktake,
  listStocktakes,
  getStocktake,
  startStocktake,
  enterCounted,
  completeStocktake,
  adjustStocktake,
} from '@/api/inventory'
import { listUsers } from '@/api/reports'
import { hasPermission } from '@/types/business'
import { useUserStore } from '@/stores/user'
import DrugPicker from '@/components/DrugPicker.vue'

const PURPOSE: Record<string, string> = { supplement: '补发', clinical: '临床领用', other: '其他' }
const TXN_TYPES: Record<string, string> = {
  purchase_in: '采购入库',
  stock_in: '其他入库',
  requisition: '领用',
  dispense: '发药',
  dispense_return: '退药回补',
  stocktake_adjust: '盘点调整',
  transfer_out: '调拨转出',
  transfer_in: '调拨转入',
  split_out: '拆盒',
  split_in: '拆零入片',
  waste: '报损',
}
const ST_TAKE_LABEL: Record<string, string> = {
  draft: '待盘点',
  counting: '盘点中',
  completed: '已完成',
  adjusted: '已调整',
}
const ST_TAKE_TAG: Record<string, string> = {
  draft: 'info',
  counting: 'warning',
  completed: 'primary',
  adjusted: 'success',
}

const tab = ref('stock')
const stock = ref<any[]>([])
const stockTotal = ref(0)
const orders = ref<any[]>([])
const orderTotal = ref(0)
const orderPage = ref(1)
const stocktakes = ref<any[]>([])
const stTakeTotal = ref(0)
const stTakePage = ref(1)
const splitOrders = ref<any[]>([])
const splitTotal = ref(0)
const splitPage = ref(1)
const transactions = ref<any[]>([])
const txnTotal = ref(0)
const alerts = ref<any[]>([])
const expiry = ref<any[]>([])
const alertTotal = ref(0)
const alertPage = ref(1)
const alertKind = ref<'below' | 'expiry'>('below')
const loading = ref(false)
const locations = ref<any[]>([])
const users = ref<any[]>([])
const userStore = useUserStore()
const canWrite = computed(
  () => userStore.role !== '' && hasPermission(userStore.role as never, 'inventory:write'),
)
const query = reactive({ keyword: '', page: 1, page_size: 20 })
const txnQuery = reactive({ txn_type: '', location_id: undefined as number | undefined, page: 1 })

const reqVisible = ref(false)
const reqForm = reactive<{
  location_id: number | undefined
  purpose: string
  reason: string
  items: any[]
}>({
  location_id: undefined,
  purpose: 'supplement',
  reason: '',
  items: [{ drug_id: null, quantity: 1 }],
})

const transferVisible = ref(false)
const transferForm = reactive<{
  inventory_id: number
  drug_name: string
  batch_no: string
  from_location_id: number | undefined
  to_location_id: number | undefined
  quantity: number
  remarks: string
}>({
  inventory_id: 0,
  drug_name: '',
  batch_no: '',
  from_location_id: undefined,
  to_location_id: undefined,
  quantity: 1,
  remarks: '',
})

const adjustVisible = ref(false)
const adjustForm = reactive<{
  inventory_id: number
  drug_name: string
  batch_no: string
  current: number
  quantity: number
  reason: string
}>({ inventory_id: 0, drug_name: '', batch_no: '', current: 0, quantity: 0, reason: '' })

const splitUnitsVisible = ref(false)
const splitUnitsForm = reactive<{
  inventory_id: number
  drug_name: string
  batch_no: string
  quantity: number
  boxes: number
  units: number
  damaged: number
  reviewer_id: number | undefined
}>({
  inventory_id: 0,
  drug_name: '',
  batch_no: '',
  quantity: 0,
  boxes: 1,
  units: 0,
  damaged: 0,
  reviewer_id: undefined,
})

const stockInVisible = ref(false)
const stockInForm = reactive<{ entries: any[] }>({ entries: [] })

const stTakeCreateVisible = ref(false)
const stTakeForm = reactive<{ location_id: number | undefined; type: number }>({
  location_id: undefined,
  type: 1,
})
const stTakeDetailVisible = ref(false)
const stTakeDetail = ref<any>(null)
const counted = reactive<Record<number, number>>({})

function locationName(id: number) {
  return locations.value.find((l) => l.id === id)?.name ?? `库房${id}`
}

function newStockInEntry() {
  return { drug_id: null, location_id: undefined, batch_no: '', expiry_date: '', quantity: 1 }
}

onMounted(() => {
  loadLocations()
  loadUsers()
  loadStock()
  loadOrders()
  loadStocktakes()
  loadSplitOrders()
})

async function loadLocations() {
  locations.value = (await listLocations()) ?? []
}
async function loadUsers() {
  try {
    const res = await listUsers({ page: 1, page_size: 200 })
    users.value = res?.list ?? []
  } catch {
    users.value = [] // 无 user:admin 权限时忽略（复核人仅药房主管可见完整列表）
  }
}

async function loadStock() {
  loading.value = true
  try {
    const res = await listInventory(query)
    stock.value = res?.list ?? []
    stockTotal.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
async function loadOrders() {
  loading.value = true
  try {
    const res = await listRequisitionOrders({ page: orderPage.value, page_size: 20 })
    orders.value = res?.list ?? []
    orderTotal.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
async function loadStocktakes() {
  const res = await listStocktakes({ page: stTakePage.value, page_size: 20 })
  stocktakes.value = res?.list ?? []
  stTakeTotal.value = res?.total ?? 0
}
async function loadSplitOrders() {
  const res = await listSplitOrders({ page: splitPage.value, page_size: 20 })
  splitOrders.value = res?.list ?? []
  splitTotal.value = res?.total ?? 0
}
async function loadTransactions() {
  loading.value = true
  try {
    const res = await listTransactions({ ...txnQuery, page_size: 20 })
    transactions.value = res?.list ?? []
    txnTotal.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
async function loadAlerts() {
  loading.value = true
  try {
    const res =
      alertKind.value === 'below'
        ? await listAlerts({ page: alertPage.value, page_size: 20 })
        : await listExpiryWarnings({ page: alertPage.value, page_size: 20 })
    const rows = res?.list ?? res ?? []
    if (alertKind.value === 'below') alerts.value = rows
    else expiry.value = rows
    alertTotal.value = res?.total ?? rows.length ?? 0
  } finally {
    loading.value = false
  }
}
async function resolve(row: any, action: string) {
  await resolveAlert(row.id, action)
  ElMessage.success('已处置')
  loadAlerts()
}

function openRequisition() {
  reqForm.location_id = undefined
  reqForm.purpose = 'supplement'
  reqForm.reason = ''
  reqForm.items = [{ drug_id: null, quantity: 1 }]
  reqVisible.value = true
}
async function saveRequisition() {
  if (!reqForm.location_id) {
    ElMessage.warning('请选择库房')
    return
  }
  await createRequisitionOrder(reqForm)
  ElMessage.success('已登记')
  reqVisible.value = false
  loadOrders()
}

function openTransfer(row: any) {
  transferForm.inventory_id = row.id
  transferForm.drug_name = row.drug_name
  transferForm.batch_no = row.batch_no
  transferForm.from_location_id = row.location_id
  transferForm.to_location_id = undefined
  transferForm.quantity = 1
  transferForm.remarks = ''
  transferVisible.value = true
}
async function saveTransfer() {
  if (!transferForm.to_location_id) {
    ElMessage.warning('请选择转入库房')
    return
  }
  await transfer({
    from_location_id: transferForm.from_location_id,
    to_location_id: transferForm.to_location_id,
    items: [{ inventory_id: transferForm.inventory_id, quantity: transferForm.quantity }],
    remarks: transferForm.remarks,
  })
  ElMessage.success('已调拨')
  transferVisible.value = false
  loadStock()
}

function openAdjust(row: any) {
  adjustForm.inventory_id = row.id
  adjustForm.drug_name = row.drug_name
  adjustForm.batch_no = row.batch_no
  adjustForm.current = row.quantity
  adjustForm.quantity = 0
  adjustForm.reason = ''
  adjustVisible.value = true
}
async function saveAdjust() {
  if (!adjustForm.quantity || !adjustForm.reason) {
    ElMessage.warning('请填写调整数量与原因')
    return
  }
  await adjust({
    inventory_id: adjustForm.inventory_id,
    quantity: adjustForm.quantity,
    reason: adjustForm.reason,
  })
  ElMessage.success('已调整')
  adjustVisible.value = false
  loadStock()
}

function openSplitUnits(row: any) {
  splitUnitsForm.inventory_id = row.id
  splitUnitsForm.drug_name = row.drug_name
  splitUnitsForm.batch_no = row.batch_no
  splitUnitsForm.quantity = row.quantity
  splitUnitsForm.boxes = 1
  splitUnitsForm.units = 0
  splitUnitsForm.damaged = 0
  splitUnitsForm.reviewer_id = undefined
  splitUnitsVisible.value = true
}
async function saveSplitUnits() {
  if (!splitUnitsForm.reviewer_id) {
    ElMessage.warning('麻精药品拆零须双人复核，请选择复核人')
    return
  }
  const reviewer = users.value.find((u) => u.id === splitUnitsForm.reviewer_id)
  await splitUnits({
    inventory_id: splitUnitsForm.inventory_id,
    boxes: splitUnitsForm.boxes,
    units: splitUnitsForm.units,
    damaged: splitUnitsForm.damaged,
    reviewer_id: splitUnitsForm.reviewer_id,
    reviewer_name: reviewer?.name ?? '',
  })
  ElMessage.success('已拆零')
  splitUnitsVisible.value = false
  loadStock()
  loadSplitOrders()
}

function openStockIn() {
  stockInForm.entries = [newStockInEntry()]
  stockInVisible.value = true
}
async function saveStockIn() {
  const entries = stockInForm.entries
  if (!entries.length) {
    ElMessage.warning('请添加入库明细')
    return
  }
  for (const e of entries) {
    if (!e.drug_id || !e.location_id || !e.batch_no || !e.expiry_date || !e.quantity) {
      ElMessage.warning('请完整填写每条明细（药品/库房/批号/效期/数量）')
      return
    }
  }
  await stockIn({
    entries: entries.map((e) => ({
      drug_id: e.drug_id,
      location_id: e.location_id,
      batch_no: e.batch_no,
      expiry_date: `${e.expiry_date}T00:00:00+08:00`,
      is_split: false,
      quantity: e.quantity,
      unit_price: 0,
    })),
  })
  ElMessage.success('已入库')
  stockInVisible.value = false
  loadStock()
}

// ---- 盘点 ----
function openStocktake() {
  stTakeForm.location_id = undefined
  stTakeForm.type = 1
  stTakeCreateVisible.value = true
}
async function saveStocktake() {
  if (!stTakeForm.location_id) {
    ElMessage.warning('请选择库房')
    return
  }
  await createStocktake({ location_id: stTakeForm.location_id, type: stTakeForm.type })
  ElMessage.success('盘点单已创建（期间该库房禁止库存变动）')
  stTakeCreateVisible.value = false
  loadStocktakes()
}
async function doStartStocktake(row: any) {
  await startStocktake(row.id)
  ElMessage.success('盘点已开始')
  loadStocktakes()
}
async function openStocktakeDetail(row: any) {
  const res = await getStocktake(row.id)
  stTakeDetail.value = res
  Object.keys(counted).forEach((k) => delete counted[Number(k)])
  for (const it of res?.items ?? []) {
    if (it.counted_quantity != null) counted[it.id] = it.counted_quantity
  }
  stTakeDetailVisible.value = true
}
async function saveCounted() {
  const items = Object.entries(counted).map(([itemId, qty]) => ({
    item_id: Number(itemId),
    counted_quantity: qty,
  }))
  if (!items.length) {
    ElMessage.warning('请先录入实盘数量')
    return
  }
  await enterCounted(stTakeDetail.value.id, { items })
  ElMessage.success('实盘已保存')
  openStocktakeDetail(stTakeDetail.value)
}
async function doCompleteStocktake() {
  await completeStocktake(stTakeDetail.value.id)
  ElMessage.success('盘点单已归档')
  openStocktakeDetail(stTakeDetail.value)
  loadStocktakes()
}
function printStocktake() {
  const d = stTakeDetail.value
  if (!d) return
  const rows = (d.items ?? [])
    .map(
      (it: any, i: number) =>
        `<tr><td>${i + 1}</td><td>${escapeHtml(it.drug_name)}</td>` +
        `<td>${escapeHtml(it.batch_no)}</td><td>${it.is_split ? '拆零' : '整盒'}</td>` +
        `<td>${escapeHtml(it.book_quantity)}</td>` +
        `<td>${escapeHtml(it.counted_quantity ?? '')}</td>` +
        `<td>${it.counted_quantity != null ? it.counted_quantity - it.book_quantity : ''}</td></tr>`,
    )
    .join('')
  printDocument(
    `盘点单 ${d.stocktake_no ?? ''}`,
    `<h2>盘点单</h2>
<div class="meta"><span>单号：${escapeHtml(d.stocktake_no)}</span>` +
      `<span>库房：${escapeHtml(locationName(d.location_id))}</span>` +
      `<span>类型：${d.type === 1 ? '周期' : '动态'}</span>` +
      `<span>状态：${escapeHtml(ST_TAKE_LABEL[d.status] ?? d.status)}</span></div>
<table><thead><tr><th>#</th><th>药品</th><th>批号</th><th>口径</th><th>账面</th><th>实盘</th><th>差异</th></tr></thead>
<tbody>${rows}</tbody></table>
<div class="footer"><span>打印时间：${new Date().toLocaleString('zh-CN')}</span>` +
      `<span>盘点人签字：________</span><span>复核人签字：________</span></div>`,
  )
}
async function doAdjustStocktake() {
  await adjustStocktake(stTakeDetail.value.id)
  ElMessage.success('差异已调整入账，可归档完成')
  openStocktakeDetail(stTakeDetail.value)
  loadStock()
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
.item-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
