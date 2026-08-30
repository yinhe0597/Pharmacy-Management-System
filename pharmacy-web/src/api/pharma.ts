import http from './http'

// 药学服务（咨询/不良反应/用药指导）
export function listConsultations(params?: Record<string, unknown>) {
  return http.get('/consultations', { params })
}
export function createConsultation(data: Record<string, unknown>) {
  return http.post('/consultations', data)
}
export function updateConsultation(id: number, data: Record<string, unknown>) {
  return http.put(`/consultations/${id}`, data)
}
export function deleteConsultation(id: number) {
  return http.delete(`/consultations/${id}`)
}
export function listAdverseReactions(params?: Record<string, unknown>) {
  return http.get('/adverse-reactions', { params })
}
export function createAdverseReaction(data: Record<string, unknown>) {
  return http.post('/adverse-reactions', data)
}
export function updateAdverseReaction(id: number, data: Record<string, unknown>) {
  return http.put(`/adverse-reactions/${id}`, data)
}
export function deleteAdverseReaction(id: number) {
  return http.delete(`/adverse-reactions/${id}`)
}
export function listGuidances(params?: Record<string, unknown>) {
  return http.get('/medication-guidances', { params })
}
export function createGuidance(data: Record<string, unknown>) {
  return http.post('/medication-guidances', data)
}
export function updateGuidance(id: number, data: Record<string, unknown>) {
  return http.put(`/medication-guidances/${id}`, data)
}
export function deleteGuidance(id: number) {
  return http.delete(`/medication-guidances/${id}`)
}
