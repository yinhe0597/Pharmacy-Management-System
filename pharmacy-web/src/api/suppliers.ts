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
