import http from './http'

// 采购
export function purchaseSuggestions() {
  return http.get('/purchase/suggestions')
}
export function listPurchaseOrders(params?: Record<string, unknown>) {
  return http.get('/purchase-orders', { params })
}
export function getPurchaseOrder(id: number) {
  return http.get(`/purchase-orders/${id}`)
}
export function createPurchaseOrder(data: Record<string, unknown>) {
  return http.post('/purchase-orders', data)
}
export function submitPurchaseOrder(id: number) {
  return http.post(`/purchase-orders/${id}/submit`)
}
export function cancelPurchaseOrder(id: number) {
  return http.post(`/purchase-orders/${id}/cancel`)
}
export function receivePurchaseOrder(id: number, data: Record<string, unknown>) {
  return http.post(`/purchase-orders/${id}/receive`, data)
}
export function completeReceipt(id: number) {
  return http.post(`/purchase-receipts/${id}/complete`)
}

// 收货单
export function listReceipts(params?: Record<string, unknown>) {
  return http.get('/purchase-receipts', { params })
}
export function getReceipt(id: number) {
  return http.get(`/purchase-receipts/${id}`)
}
