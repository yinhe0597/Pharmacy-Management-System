import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Role } from '@/types/business'
import { ROLE_LABELS } from '@/types/business'
import type { LoginUser } from '@/types/api'
import {
  getToken,
  setToken,
  clearToken,
  getStoredUser,
  setStoredUser,
  clearStoredUser,
} from '@/utils/auth'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(getToken())
  const user = ref<LoginUser | null>(null)

  // 从 localStorage 恢复
  const stored = getStoredUser()
  if (stored) {
    try {
      user.value = JSON.parse(stored)
    } catch {
      clearStoredUser()
    }
  }

  const role = computed<Role | ''>(() => (user.value?.role as Role) ?? '')
  const roleLabel = computed(() => (role.value ? ROLE_LABELS[role.value as Role] : ''))

  function setLogin(result: { token: string; user: LoginUser }): void {
    token.value = result.token
    user.value = result.user
    setToken(result.token)
    setStoredUser(JSON.stringify(result.user))
  }

  function clear(): void {
    token.value = ''
    user.value = null
    clearToken()
    clearStoredUser()
  }

  return { token, user, role, roleLabel, setLogin, clear }
})
