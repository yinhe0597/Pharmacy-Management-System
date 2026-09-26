// 后端实体类型（与 internal/model 对齐，仅列出前端渲染/编辑所需字段）

export interface Drug {
  id: number
  code: string
  item_type: string
  generic_name: string
  brand_name?: string
  dosage_form?: string
  specification?: string
  manufacturer?: string
  approval_number?: string
  category_id?: number | null
  insurance_class?: string
  vbp_batch?: number
  base_unit?: string
  split_unit?: string
  pack_size?: number
  is_split_allowed?: boolean
  retail_price?: number
  purchase_price?: number
  split_retail_price?: number
  split_purchase_price?: number
  antibiotic_level?: number
  special_control_type?: number
  psychotropic_level?: number
  max_single_dose?: number
  max_daily_dose?: number
  expiry_warning_days?: number
  active_ingredient?: string
  atc_code?: string
  pharmacological_group?: string
  pregnancy_category?: string
  age_min_years?: number | null
  age_max_years?: number | null
  interaction_tags?: string
  lactation_safe?: boolean | null
  contraindication_notes?: string
  py_code?: string
  status: number
  is_frozen?: boolean
}

export interface Category {
  id: number
  code: string
  name: string
  parent_id: number
  sort_order?: number
  status: number
}

export interface Supplier {
  id: number
  code: string
  name: string
  contact_person?: string
  phone?: string
  address?: string
  status?: number
}

export interface Patient {
  id: number
  card_no: string
  name: string
  gender?: string
  age?: string
  phone?: string
  is_lactating?: boolean
}

export interface PatientAllergy {
  id: number
  patient_id: number
  drug_name: string
  reaction?: string
  severity: number
}

export interface MedicationRecord {
  drug_name: string
  dosage: string
  begin_date: string
}

export interface PurchaseOrder {
  id: number
  order_no: string
  supplier_id: number
  supplier_name?: string
  status: string
  total_amount?: number
  created_at: string
  items?: PurchaseOrderItem[]
}

export interface PurchaseOrderItem {
  id: number
  drug_id: number
  drug_name?: string
  quantity: number
  unit_price: number
  received_quantity?: number
}

export interface Inventory {
  id: number
  drug_id: number
  drug_name?: string
  location_id: number
  location_name?: string
  batch_no: string
  expiry_date: string
  quantity: number
  reserved_quantity: number
  is_split: boolean
  unit_price: number
  status: number
}

export interface RequisitionOrder {
  id: number
  requisition_no: string
  location_id: number
  purpose: string
  reason?: string
  operator_name?: string
  status: string
  created_at: string
}

export interface RequisitionOrderItem {
  id: number
  requisition_order_id: number
  drug_id: number
  drug_name?: string
  batch_no?: string
  is_split: boolean
  quantity: number
  unit_price: number
}

export interface Prescription {
  id: number
  prescription_no: string
  patient_id: number | null
  patient_name: string
  patient_gender?: string
  patient_age?: string
  patient_card_no?: string
  is_pregnant?: boolean
  is_lactating?: boolean
  diagnosis_code?: string
  diagnosis?: string
  department?: string
  doctor_name?: string
  prescription_type: number
  special_control_type?: number
  source?: string
  status: string
  total_amount: number
  auditor_name?: string
  checker_name?: string
  created_at: string
}

export interface PrescriptionItem {
  id: number
  prescription_id: number
  line_no: number
  drug_id: number
  drug_name: string
  specification?: string
  pack_size?: number
  is_split_allowed?: boolean
  is_split?: boolean
  quantity: number
  unit_price: number
  retail_price?: number
  amount: number
  usage_text?: string
  frequency?: string
  single_dose?: number
  total_daily_dose?: number
  days?: number
  dispensed_quantity?: number
  returned_quantity?: number
  status: string
}

export interface PrescriptionDetail extends Prescription {
  items: PrescriptionItem[]
  dispense_records: unknown[]
  audit_logs: AuditLog[]
  patient?: Patient
  allergies?: PatientAllergy[]
}

export interface AuditLog {
  id: number
  action: string
  from_status?: string
  to_status?: string
  operator_name?: string
  remarks?: string
  created_at: string
}

export interface ChargeRecord {
  id: number
  patient_id: number
  patient_name: string
  item_type: string
  item_name: string
  quantity: number
  unit_price: number
  amount: number
  voided: boolean
  ref_type?: string
  ref_id?: number
  created_at: string
}

export interface ClinicalService {
  id: number
  code: string
  name: string
  category?: string
  unit_price: number
  unit: string
  status: number
}

export interface ReferenceMatch {
  name: string
  insurance_class: string
  vbp_batch: number
}
