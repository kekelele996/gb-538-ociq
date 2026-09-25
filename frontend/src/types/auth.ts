export type UserRole = 'admin' | 'acoustic_engineer' | 'data_analyst' | 'reviewer' | 'auditor'

export interface CurrentUser {
  id: number
  username: string
  display_name: string
  role: UserRole
}

export interface LoginResponse {
  token: string
  expires_at: string
  user: CurrentUser
}
