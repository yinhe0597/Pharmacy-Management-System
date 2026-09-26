import http, { downloadFile, saveBlob } from './http'

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
// 二期 S7：就诊量/收入构成/诊断分布
export function visitVolume(params?: Record<string, unknown>) {
  return http.get('/reports/visit-volume', { params })
}
export function revenueBreakdown(params?: Record<string, unknown>) {
  return http.get('/reports/revenue-breakdown', { params })
}
export function diagnosisDistribution(params?: Record<string, unknown>) {
  return http.get('/reports/diagnosis-distribution', { params })
}
// 报表导出 CSV（name 取值见后端 Export* 常量）
export async function exportReport(params: Record<string, unknown>) {
  const { blob, filename } = await downloadFile('/reports/export', params)
  saveBlob(blob, filename)
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
