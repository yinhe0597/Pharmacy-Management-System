// 统一响应信封与分页（docs/03 §1）

export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface PageQuery {
  page?: number
  page_size?: number
}

// 登录响应
export interface LoginUser {
  id: number
  username?: string
  name: string
  role: string
}

export interface LoginResult {
  token: string
  user: LoginUser
}
