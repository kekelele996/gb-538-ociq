import { defineStore } from 'pinia'
import { login as loginRequest } from '../api/auth-api'
import { tokenKey, userKey } from '../api/client'
import type { CurrentUser } from '../types/auth'

function storedUser(): CurrentUser | null {
  try {
    const raw = localStorage.getItem(userKey)
    return raw ? JSON.parse(raw) as CurrentUser : null
  } catch {
    localStorage.removeItem(userKey)
    return null
  }
}

export const useAuthStore = defineStore('auth', {
  state: () => ({ user: storedUser() as CurrentUser | null, busy: false }),
  actions: {
    async login(username: string, password: string) {
      this.busy = true
      try {
        const response = await loginRequest(username, password)
        localStorage.setItem(tokenKey, response.token)
        localStorage.setItem(userKey, JSON.stringify(response.user))
        this.user = response.user
      } finally {
        this.busy = false
      }
    },
    logout() {
      localStorage.removeItem(tokenKey)
      localStorage.removeItem(userKey)
      this.user = null
    },
  },
})
