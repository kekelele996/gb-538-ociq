import { api, json } from './client'
import type { LoginResponse } from '../types/auth'

export const login = (username: string, password: string) =>
  api<LoginResponse>('/auth/login', json('POST', { username, password }))
