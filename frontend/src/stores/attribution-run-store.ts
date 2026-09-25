import { defineStore } from 'pinia'
import { attributionRunApi } from '../api/attribution-run-api'
import { errorMessage } from '../api/client'
import type { AttributionRun, CreateAttributionRun } from '../types/attribution-run'

export const useAttributionRunStore = defineStore('attribution-runs', {
  state: () => ({ items: [] as AttributionRun[], selected: null as AttributionRun | null, loading: false, error: '' }),
  actions: {
    async load() {
      this.loading = true; this.error = ''
      try { this.items = await attributionRunApi.list() } catch (error) { this.error = errorMessage(error) } finally { this.loading = false }
    },
    async select(id: number) { this.selected = await attributionRunApi.get(id); return this.selected },
    async create(value: CreateAttributionRun) {
      const key = crypto.randomUUID(); const created = await attributionRunApi.create(value, key); await this.load(); this.selected = created; return created
    },
    async review(item: AttributionRun, note: string) { const updated = await attributionRunApi.review(item.id, item.version, note); await this.load(); this.selected = updated; return updated },
    async confirm(item: AttributionRun) { const updated = await attributionRunApi.confirm(item.id, item.version); await this.load(); this.selected = updated; return updated },
    async void(item: AttributionRun) { const updated = await attributionRunApi.void(item.id, item.version); await this.load(); this.selected = updated; return updated },
  },
})
