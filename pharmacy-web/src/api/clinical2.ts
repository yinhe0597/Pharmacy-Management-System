import http from './http'

// 二期就诊模块（docs/20 S2-S4）：就诊 / 病历 / 合并结算

// ---- 就诊 ----
export function listVisits(params?: Record<string, unknown>) {
  return http.get('/visits', { params })
}
export function getVisit(id: number) {
  return http.get(`/visits/${id}`)
}
export function registerVisit(data: Record<string, unknown>) {
  return http.post('/visits', data)
}
export function startVisit(id: number) {
  return http.post(`/visits/${id}/start`)
}
export function finishVisit(id: number) {
  return http.post(`/visits/${id}/finish`)
}
export function cancelVisit(id: number) {
  return http.post(`/visits/${id}/cancel`)
}

// ---- 病历 ----
export function getMedicalRecord(visitId: number) {
  return http.get(`/visits/${visitId}/medical-record`)
}
export function saveMedicalRecord(visitId: number, data: Record<string, unknown>) {
  return http.post(`/visits/${visitId}/medical-record`, data)
}

// ---- 合并结算 ----
export function listCharges(params?: Record<string, unknown>) {
  return http.get('/charges', { params })
}
export function getCharge(id: number) {
  return http.get(`/charges/${id}`)
}
export function createCharge(visitId: number, data: Record<string, unknown>) {
  return http.post(`/visits/${visitId}/charge`, data)
}
export function payCharge(id: number, data: Record<string, unknown>) {
  return http.post(`/charges/${id}/pay`, data)
}
export function refundCharge(id: number) {
  return http.post(`/charges/${id}/refund`)
}
