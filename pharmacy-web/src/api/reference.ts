import http from './http'

// 参考数据（ICD-10/集采/医保/耗材/非医保 + 目录匹配）
export function listDiagnosisCodes(params?: Record<string, unknown>) {
  return http.get('/reference/diagnosis-codes', { params })
}
export function listVBPDrugs(params?: Record<string, unknown>) {
  return http.get('/reference/vbp-drugs', { params })
}
export function listNHSADrugs(params?: Record<string, unknown>) {
  return http.get('/reference/nhsa-drugs', { params })
}
export function listConsumables(params?: Record<string, unknown>) {
  return http.get('/reference/consumables', { params })
}
export function listNonInsuranceDrugs(params?: Record<string, unknown>) {
  return http.get('/reference/non-insurance-drugs', { params })
}
export function matchDrug(name: string) {
  return http.get('/reference/drug-match', { params: { name } })
}
