const TOKEN_KEY = 'yf_token'
const USER_KEY = 'yf_user'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? ''
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function getStoredUser(): string {
  return localStorage.getItem(USER_KEY) ?? ''
}

export function setStoredUser(userJson: string): void {
  localStorage.setItem(USER_KEY, userJson)
}

export function clearStoredUser(): void {
  localStorage.removeItem(USER_KEY)
}
