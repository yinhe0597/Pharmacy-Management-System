import http from './http'

// å¤„æ–¹
export interface PrescriptionItemInput {
  drug_id: number
  quantity: number
  is_split?: boolean
  usage_text?: string
  frequency?: string
  route?: string // ç»™è¯é€”å¾„ï¼ˆoral/external/iv/im/iv_drip/inhale/otherï¼‰
  batch_group?: string // åˆ†æ‰¹ç»„ï¼ˆå£æœç»„/è¾“æ¶²ç»„1 ç­‰ï¼‰
  single_dose?: number
  total_daily_dose?: number
  days?: number
}

export interface PrescriptionInput {
  patient_id?: number
  patient_name: string
  patient_gender?: string
  patient_age?: string
  patient_card_no?: string
  diagnosis_code?: string
  diagnosis?: string
  department?: string
  doctor_name?: string
  prescription_type?: number
  is_pregnant?: boolean
  is_lactating?: boolean
  remarks?: string
  items: PrescriptionItemInput[]
}

export function listPrescriptions(params?: Record<string, unknown>) {
  return http.get('/prescriptions', { params })
}
export function getPrescription(id: number) {
  return http.get(`/prescriptions/${id}`)
}
export function createPrescription(data: PrescriptionInput) {
  return http.post('/prescriptions', data)
}
export function updatePrescription(id: number, data: PrescriptionInput) {
  return http.put(`/prescriptions/${id}`, data)
}
export function submitPrescription(id: number) {
  return http.post(`/prescriptions/${id}/submit`)
}
export function verifyOrder(id: number, remarks?: string) {
  return http.post(`/prescriptions/${id}/verify-order`, { remarks })
}
export function reviewPrescription(id: number, action: string, remarks?: string) {
  return http.post(`/prescriptions/${id}/review`, { action, remarks })
}
export function dispensePrescription(id: number) {
  return http.post(`/prescriptions/${id}/dispense`)
}
export function confirmDispense(id: number) {
  return http.post(`/prescriptions/${id}/confirm-dispense`)
}
export function returnPrescription(
  id: number,
  items: { item_id: number; return_quantity: number }[],
) {
  return http.post(`/prescriptions/${id}/return`, { items })
}
export function cancelPrescription(id: number) {
  return http.post(`/prescriptions/${id}/cancel`)
}

// Éó¼ÆÈÕÖ¾£¨Á÷×ªÈ«¼ÇÂ¼£©
export function getAuditLog(id: number) {
  return http.get(`/prescriptions/${id}/audit-log`)
}
