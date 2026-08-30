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
export function updateInteraction(id: number, data: Record<string, unknown>) {
  return http.put(`/interactions/${id}`, data)
}
export function deleteInteraction(id: number) {
  return http.delete(`/interactions/${id}`)
}

// 药品成分
export function listDrugIngredients(drugId: number) {
  return http.get(`/drugs/${drugId}/ingredients`)
}
export function addDrugIngredient(drugId: number, data: Record<string, unknown>) {
  return http.post(`/drugs/${drugId}/ingredients`, data)
}
export function deleteDrugIngredient(id: number) {
  return http.delete(`/drug-ingredients/${id}`)
}

// 成分相互作用规则
export function listIngredientInteractions(params?: Record<string, unknown>) {
  return http.get('/ingredient-interactions', { params })
}
export function createIngredientInteraction(data: Record<string, unknown>) {
  return http.post('/ingredient-interactions', data)
}
export function updateIngredientInteraction(id: number, data: Record<string, unknown>) {
  return http.put(`/ingredient-interactions/${id}`, data)
}
export function deleteIngredientInteraction(id: number) {
  return http.delete(`/ingredient-interactions/${id}`)
}

// 药理分类相互作用规则
export function listClassInteractions(params?: Record<string, unknown>) {
  return http.get('/class-interactions', { params })
}
export function createClassInteraction(data: Record<string, unknown>) {
  return http.post('/class-interactions', data)
}
export function updateClassInteraction(id: number, data: Record<string, unknown>) {
  return http.put(`/class-interactions/${id}`, data)
}
export function deleteClassInteraction(id: number) {
  return http.delete(`/class-interactions/${id}`)
}

// 交互标签规则
export function listTagInteractions(params?: Record<string, unknown>) {
  return http.get('/tag-interactions', { params })
}
export function createTagInteraction(data: Record<string, unknown>) {
  return http.post('/tag-interactions', data)
}
export function updateTagInteraction(id: number, data: Record<string, unknown>) {
  return http.put(`/tag-interactions/${id}`, data)
}
export function deleteTagInteraction(id: number) {
  return http.delete(`/tag-interactions/${id}`)
}
