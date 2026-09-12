import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearStoredUser,
  clearToken,
  getStoredUser,
  getToken,
  setStoredUser,
  setToken,
} from './auth'

function installLocalStorageMock(): void {
  const store = new Map<string, string>()
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    writable: true,
    value: {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => store.set(key, value),
      removeItem: (key: string) => {
        store.delete(key)
      },
      clear: () => store.clear(),
    },
  })
}

beforeEach(() => {
  installLocalStorageMock()
})

describe('auth utils', () => {
  it('未设置时 getToken 返回空字符串', () => {
    expect(getToken()).toBe('')
  })

  it('setToken 后 getToken 可回读', () => {
    setToken('token-123')
    expect(getToken()).toBe('token-123')
  })

  it('clearToken 清除 token', () => {
    setToken('token-123')
    clearToken()
    expect(getToken()).toBe('')
  })

  it('用户 JSON 存取与清除', () => {
    expect(getStoredUser()).toBe('')
    setStoredUser('{"name":"admin"}')
    expect(getStoredUser()).toBe('{"name":"admin"}')
    clearStoredUser()
    expect(getStoredUser()).toBe('')
  })
})
