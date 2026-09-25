import { defineStore } from 'pinia'
import { sourceProfileApi } from '../api/source-profile-api'
import { errorMessage } from '../api/client'
import type { CreateSourceProfile, SourceProfile } from '../types/source-profile'

export const useSourceProfileStore = defineStore('source-profiles', {
  state: () => ({ items: [] as SourceProfile[], loading: false, error: '' }),
  actions: {
    async load() {
      this.loading = true; this.error = ''
      try { this.items = await sourceProfileApi.list() } catch (error) { this.error = errorMessage(error) } finally { this.loading = false }
    },
    async create(value: CreateSourceProfile) { const created = await sourceProfileApi.create(value); await this.load(); return created },
    async transition(item: SourceProfile, toState: 'active' | 'retired') {
      const updated = await sourceProfileApi.transition(item.id, toState, item.lock_version); await this.load(); return updated
    },
  },
})
