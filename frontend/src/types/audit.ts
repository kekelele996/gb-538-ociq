export interface AuditLog {
  id: number
  request_id: string
  actor_id: number
  actor_name: string
  action: string
  entity_type: string
  entity_id: number
  before: unknown
  after: unknown
  metadata: unknown
  created_at: string
}
