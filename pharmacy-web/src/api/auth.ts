import http from './http'
import type { LoginResult, LoginUser } from '@/types/api'

export function login(username: string, password: string) {
  return http.post<LoginResult>('/auth/login', { username, password })
}

export function logout() {
  return http.post('/auth/logout')
}

export function profile() {
  return http.get<LoginUser>('/auth/profile')
}

export function changePassword(oldPassword: string, newPassword: string) {
  return http.put('/auth/password', { old_password: oldPassword, new_password: newPassword })
}
