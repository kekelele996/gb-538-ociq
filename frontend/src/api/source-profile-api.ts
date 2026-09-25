import { api, json } from './client'
import type { CreateSourceProfile, SourceProfile } from '../types/source-profile'

export const sourceProfileApi = {
  list: () => api<SourceProfile[]>('/source-profiles'),
  get: (id: number) => api<SourceProfile>(`/source-profiles/${id}`),
  create: (value: CreateSourceProfile) => api<SourceProfile>('/source-profiles', json('POST', value)),
  transition: (id: number, toState: 'active' | 'retired', lockVersion: number) =>
    api<SourceProfile>(`/source-profiles/${id}/transition`, json('POST', { to_state: toState, lock_version: lockVersion })),
}
