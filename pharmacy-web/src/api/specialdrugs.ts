import http from './http'

// 特殊药品（麻精处方/空安瓿/专账）
export function listSpecialPrescriptions(params?: Record<string, unknown>) {
  return http.get('/special-drugs/prescriptions', { params })
}
export function listAmpouleReturns(params?: Record<string, unknown>) {
  return http.get('/special-drugs/ampoule-returns', { params })
}
export function createAmpouleReturn(data: Record<string, unknown>) {
  return http.post('/special-drugs/ampoule-returns', data)
}
export function verifyAmpouleReturn(id: number) {
  return http.post(`/special-drugs/ampoule-returns/${id}/verify`)
}
export function listLedgers(params?: Record<string, unknown>) {
  return http.get('/special-drugs/ledgers', { params })
}
// 发药专册登记（手工补录）
export function registerDispense(data: Record<string, unknown>) {
  return http.post('/special-drugs/dispense-register', data)
}
