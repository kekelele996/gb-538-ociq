import { api, query } from './client'
import type { AuditLog } from '../types/audit'

export const auditApi = {
  list: (filters: { entity_type?: string; request_id?: string; actor?: string; limit?: number } = {}) =>
    api<AuditLog[]>(`/audit-logs${query(filters)}`),
}
