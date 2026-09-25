import { defineStore } from 'pinia'
import { monitoringPointApi } from '../api/monitoring-point-api'
import { errorMessage } from '../api/client'
import type { CreateMonitoringPoint, MonitoringPoint } from '../types/monitoring-point'

export const useMonitoringPointStore = defineStore('monitoring-points', {
  state: () => ({ items: [] as MonitoringPoint[], loading: false, error: '' }),
  actions: {
    async load() {
      this.loading = true; this.error = ''
      try { this.items = await monitoringPointApi.list() } catch (error) { this.error = errorMessage(error) } finally { this.loading = false }
    },
    async create(value: CreateMonitoringPoint) { const created = await monitoringPointApi.create(value); await this.load(); return created },
    async deactivate(item: MonitoringPoint) { const updated = await monitoringPointApi.deactivate(item.id, item.version); await this.load(); return updated },
  },
})
