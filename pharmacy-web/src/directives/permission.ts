import type { Directive } from 'vue'
import { useUserStore } from '@/stores/user'
import { hasPermission, type Permission, type Role } from '@/types/business'

// v-permission="'drug:write'" 按钮级权限控制（docs/16 §4.3）
export const permission: Directive<HTMLElement, Permission> = {
  mounted(el, binding) {
    const store = useUserStore()
    const role = store.role as Role
    const required = binding.value
    if (!role || (required && !hasPermission(role, required))) {
      el.parentNode?.removeChild(el)
    }
  },
}
