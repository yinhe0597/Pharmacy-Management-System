import http from './http'

// 供应商
export function listSuppliers(params?: Record<string, unknown>) {
  return http.get('/suppliers', { params })
}
export function createSupplier(data: Record<string, unknown>) {
  return http.post('/suppliers', data)
}
export function updateSupplier(id: number, data: Record<string, unknown>) {
  return http.put(`/suppliers/${id}`, data)
}
export function deleteSupplier(id: number) {
  return http.delete(`/suppliers/${id}`)
}
export function getSupplier(id: number) {
  return http.get(`/suppliers/${id}`)
}
// 药品-供应商供货关系
export function listDrugSuppliers(drugId: number) {
  return http.get(`/drugs/${drugId}/suppliers`)
}
export function bindDrugSupplier(drugId: number, data: Record<string, unknown>) {
  return http.post(`/drugs/${drugId}/suppliers`, data)
}
export function deleteDrugSupplier(id: number) {
  return http.delete(`/drug-suppliers/${id}`)
}
