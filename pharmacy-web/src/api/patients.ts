import http from './http'

// 患者 / 过敏史 / 用药史
export function listPatients(params?: Record<string, unknown>) {
  return http.get('/patients', { params })
}
export function getPatient(id: number) {
  return http.get(`/patients/${id}`)
}
export function createPatient(data: Record<string, unknown>) {
  return http.post('/patients', data)
}
export function updatePatient(id: number, data: Record<string, unknown>) {
  return http.put(`/patients/${id}`, data)
}
export function listAllergies(patientId: number) {
  return http.get(`/patients/${patientId}/allergies`)
}
export function addAllergy(patientId: number, data: Record<string, unknown>) {
  return http.post(`/patients/${patientId}/allergies`, data)
}
export function deleteAllergy(id: number) {
  return http.delete(`/patient-allergies/${id}`)
}
export function medicationHistory(patientId: number) {
  return http.get(`/patients/${patientId}/medication-history`)
}
