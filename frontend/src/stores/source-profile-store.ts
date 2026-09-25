import { defineStore } from 'pinia'
import { sourceProfileApi } from '../api/source-profile-api'
import { errorMessage } from '../api/client'
import { expectedActiveId } from '../utils/source-versions'
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
      const expected = toState === 'active' ? expectedActiveId(this.items, item) : null
      try {
        const updated = await sourceProfileApi.transition(item.id, toState, item.lock_version, expected)
        await this.load()
        return updated
      } catch (error) {
        // 冲突（409）时先刷新到最新状态，让操作者看到对方已完成的变更。
        await this.load()
        throw error
      }
    },
  },
})
