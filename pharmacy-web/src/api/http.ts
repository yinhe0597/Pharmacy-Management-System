import axios, { type AxiosInstance, type AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiEnvelope } from '@/types/api'
import { getToken, clearToken } from '@/utils/auth'

const instance: AxiosInstance = axios.create({
  baseURL: '/api/v1',
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

// 统一请求封装：响应已解包为 data 字段（返回 Promise<T>，默认 any）
const request = {
  get<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return instance.get(url, config) as unknown as Promise<T>
  },
  post<T = any>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return instance.post(url, data, config) as unknown as Promise<T>
  },
  put<T = any>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return instance.put(url, data, config) as unknown as Promise<T>
  },
  patch<T = any>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return instance.patch(url, data, config) as unknown as Promise<T>
  },
  delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return instance.delete(url, config) as unknown as Promise<T>
  },
}

export default request
