import http from './http'

// 药品/分类/配伍/交互规则（主数据）
export function listDrugs(params?: Record<string, unknown>) {
  return http.get('/drugs', { params })
}
export function getDrug(id: number) {
  return http.get(`/drugs/${id}`)
}
export function createDrug(data: Record<string, unknown>) {
  return http.post('/drugs', data)
}
export function updateDrug(id: number, data: Record<string, unknown>) {
  return http.put(`/drugs/${id}`, data)
}
export function deleteDrug(id: number) {
  return http.delete(`/drugs/${id}`)
}
export function setDrugStatus(id: number, status: number) {
  return http.patch(`/drugs/${id}/status`, { status })
}
export function listCategories() {
  return http.get('/categories')
}
export function createCategory(data: Record<string, unknown>) {
  return http.post('/categories', data)
}
export function updateCategory(id: number, data: Record<string, unknown>) {
  return http.put(`/categories/${id}`, data)
}
export function deleteCategory(id: number) {
  return http.delete(`/categories/${id}`)
}
export function listInteractions(params?: Record<string, unknown>) {
  return http.get('/interactions', { params })
}
export function createInteraction(data: Record<string, unknown>) {
  return http.post('/interactions', data)
}
