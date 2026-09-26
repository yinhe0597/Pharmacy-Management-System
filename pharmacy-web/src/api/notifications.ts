import http from './http'

// 站内通知
export function listNotifications(params?: Record<string, unknown>) {
  return http.get('/notifications', { params })
}
export function unreadCount() {
  return http.get('/notifications/unread-count')
}
export function markRead(id: number) {
  return http.put(`/notifications/${id}/read`)
}
export function markAllRead() {
  return http.put('/notifications/read-all')
}
