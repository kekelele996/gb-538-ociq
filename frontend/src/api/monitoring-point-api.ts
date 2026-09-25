import { api, json } from './client'
import type { CreateMonitoringPoint, MonitoringPoint } from '../types/monitoring-point'

export const monitoringPointApi = {
  list: () => api<MonitoringPoint[]>('/monitoring-points'),
  get: (id: number) => api<MonitoringPoint>(`/monitoring-points/${id}`),
  create: (value: CreateMonitoringPoint) => api<MonitoringPoint>('/monitoring-points', json('POST', value)),
  update: (id: number, value: Omit<CreateMonitoringPoint, 'point_code'> & { version: number }) =>
    api<MonitoringPoint>(`/monitoring-points/${id}`, json('PUT', value)),
  deactivate: (id: number, version: number) =>
    api<MonitoringPoint>(`/monitoring-points/${id}/deactivate`, json('POST', { version })),
}
