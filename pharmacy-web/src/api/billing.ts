import http from './http'

// 计费 / 诊疗项目
export function listChargeRecords(params?: Record<string, unknown>) {
  return http.get('/charge-records', { params })
}
export function createCharge(data: Record<string, unknown>) {
  return http.post('/charge-records', data)
}
export function chargePrescription(prescriptionId: number) {
  return http.post(`/charge-records/from-prescription/${prescriptionId}`)
}
export function voidCharge(id: number) {
  return http.post(`/charge-records/${id}/void`)
}
export function listClinicalServices(params?: Record<string, unknown>) {
  return http.get('/clinical-services', { params })
}
export function createClinicalService(data: Record<string, unknown>) {
  return http.post('/clinical-services', data)
}
export function setClinicalServiceStatus(id: number, status: number) {
  return http.patch(`/clinical-services/${id}/status`, { status })
}
export function updateClinicalService(id: number, data: Record<string, unknown>) {
  return http.put(`/clinical-services/${id}`, data)
}
export function deleteClinicalService(id: number) {
  return http.delete(`/clinical-services/${id}`)
}
