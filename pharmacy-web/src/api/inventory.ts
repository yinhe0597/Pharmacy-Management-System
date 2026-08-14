import http from './http'

// 库存 / 领用补发登记单
export function listInventory(params?: Record<string, unknown>) {
  return http.get('/inventory', { params })
}
export function listLocations() {
  return http.get('/inventory/locations')
}
export function transfer(data: Record<string, unknown>) {
  return http.post('/inventory/transfer', data)
}
export function split(data: Record<string, unknown>) {
  return http.post('/inventory/split', data)
}
export function splitUnits(data: Record<string, unknown>) {
  return http.post('/inventory/split-units', data)
}
export function adjust(data: Record<string, unknown>) {
  return http.post('/inventory/adjust', data)
}
export function stockIn(data: Record<string, unknown>) {
  return http.post('/inventory/stock-in', data)
}
export function requisition(data: Record<string, unknown>) {
  return http.post('/inventory/requisition', data)
}
export function listRequisitionOrders(params?: Record<string, unknown>) {
  return http.get('/inventory/requisition-orders', { params })
}
export function getRequisitionOrder(id: number) {
  return http.get(`/inventory/requisition-orders/${id}`)
}
export function createRequisitionOrder(data: Record<string, unknown>) {
  return http.post('/inventory/requisition-orders', data)
}
export function listSplitOrders(params?: Record<string, unknown>) {
  return http.get('/inventory/split-orders', { params })
}
export function listTransactions(params?: Record<string, unknown>) {
  return http.get('/inventory/transactions', { params })
}
export function listAlerts(params?: Record<string, unknown>) {
  return http.get('/inventory/stock-warnings', { params })
}
export function resolveAlert(id: number, action: string) {
  return http.post(`/inventory/alerts/${id}/resolve`, { action })
}
