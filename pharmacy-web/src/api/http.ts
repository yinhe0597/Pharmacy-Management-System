import axios, { type AxiosInstance, type AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiEnvelope } from '@/types/api'
import { getToken, clearToken } from '@/utils/auth'

const instance: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
})

instance.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

instance.interceptors.response.use(
  (response): any => {
    const body = response.data as ApiEnvelope
    if (body.code === 0) {
      return body.data
    }
    handleError(body.code, body.message)
    return Promise.reject(new Error(body.message))
  },
  (error) => {
    if (axios.isCancel(error)) {
      return Promise.reject(error)
    }
    ElMessage.error(error?.message ?? '网络异常，请重试')
    return Promise.reject(error)
  },
)

function handleError(code: number, message: string): void {
  if (code === 9002) {
    clearToken()
    ElMessage.error('登录已失效，请重新登录')
    window.location.href = '/login'
    return
  }
  ElMessage.error(message || '操作失败')
}

// 请求取消池：路由切换时统一 abort，避免旧请求覆盖新页面数据（docs/19）
const pending = new Map<string, AbortController>()

function keyOf(url: string, config?: AxiosRequestConfig): string {
  return `${url}${JSON.stringify(config?.params ?? '')}`
}

function attachCancel(url: string, config?: AxiosRequestConfig): AxiosRequestConfig {
  const controller = new AbortController()
  const key = keyOf(url, config)
  const prev = pending.get(key)
  if (prev) prev.abort()
  pending.set(key, controller)
  return { ...config, signal: controller.signal }
}

function finish(url: string, config?: AxiosRequestConfig): void {
  pending.delete(keyOf(url, config))
}

export function cancelAllRequests(): void {
  pending.forEach((c) => c.abort())
  pending.clear()
}

// 统一请求封装：响应已解包为 data 字段（返回 Promise<T>，默认 any）
const request = {
  get<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return instance
      .get(url, attachCancel(url, config))
      .finally(() => finish(url, config)) as unknown as Promise<T>
  },
  post<T = any>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return instance
      .post(url, data, attachCancel(url, config))
      .finally(() => finish(url, config)) as unknown as Promise<T>
  },
  put<T = any>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return instance
      .put(url, data, attachCancel(url, config))
      .finally(() => finish(url, config)) as unknown as Promise<T>
  },
  patch<T = any>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return instance
      .patch(url, data, attachCancel(url, config))
      .finally(() => finish(url, config)) as unknown as Promise<T>
  },
  delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return instance
      .delete(url, attachCancel(url, config))
      .finally(() => finish(url, config)) as unknown as Promise<T>
  },
}

export default request
