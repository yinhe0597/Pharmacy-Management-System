import http from './http'

// 报表
export function inventorySummary(params?: Record<string, unknown>) {
  return http.get('/reports/inventory-summary', { params })
}
export function expiryAnalysis() {
  return http.get('/reports/expiry-analysis')
}
export function specialDrugUsage(params?: Record<string, unknown>) {
  return http.get('/reports/special-drug-usage', { params })
}
export function dispensingWorkload(params?: Record<string, unknown>) {
  return http.get('/reports/dispensing-workload', { params })
}
export function splitStatistics(params?: Record<string, unknown>) {
  return http.get('/reports/split-statistics', { params })
}
export function patientCharges(params?: Record<string, unknown>) {
  return http.get('/reports/patient-charges', { params })
}

// 系统管理
export function listUsers(params?: Record<string, unknown>) {
  return http.get('/users', { params })
}
export function createUser(data: Record<string, unknown>) {
  return http.post('/users', data)
}
export function updateUser(id: number, data: Record<string, unknown>) {
  return http.put(`/users/${id}`, data)
}
export function deleteUser(id: number) {
  return http.delete(`/users/${id}`)
}
export function listOperationLogs(params?: Record<string, unknown>) {
  return http.get('/operation-logs', { params })
}
export function listSystemSettings() {
  return http.get('/system-settings')
}
export function updateSystemSetting(key: string, value: string) {
  return http.put(`/system-settings/${key}`, { value })
}
