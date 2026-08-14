import axios, { type AxiosInstance, type AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiEnvelope } from '@/types/api'
import { getToken, clearToken } from '@/utils/auth'

const http: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

// 请求拦截：注入 token
http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截：解包信封 { code, message, data }
http.interceptors.response.use(
  (response: AxiosResponse<ApiEnvelope>) => {
    const body = response.data
    if (body.code === 0) {
      return body.data as never
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
    // 未认证：清 token 跳登录
    clearToken()
    ElMessage.error('登录已失效，请重新登录')
    window.location.href = '/login'
    return
  }
  ElMessage.error(message || '操作失败')
}

export default http
