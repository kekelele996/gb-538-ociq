import { api, json } from './client'
import type { AttributionRun, CreateAttributionRun } from '../types/attribution-run'

export const attributionRunApi = {
  list: () => api<AttributionRun[]>('/attribution-runs'),
  get: (id: number) => api<AttributionRun>(`/attribution-runs/${id}`),
  create: (value: CreateAttributionRun, idempotencyKey: string) =>
    api<AttributionRun>('/attribution-runs', json('POST', value, { 'Idempotency-Key': idempotencyKey })),
  review: (id: number, version: number, note: string) =>
    api<AttributionRun>(`/attribution-runs/${id}/review`, json('POST', { version, note })),
  confirm: (id: number, version: number) =>
    api<AttributionRun>(`/attribution-runs/${id}/confirm`, json('POST', { version })),
  void: (id: number, version: number) =>
    api<AttributionRun>(`/attribution-runs/${id}/void`, json('POST', { version })),
}
