import { defineStore } from 'pinia'
import { noiseMeasurementApi } from '../api/noise-measurement-api'
import { errorMessage } from '../api/client'
import type { MeasurementState } from '../types/enums/measurement-quality'
import type { CreateNoiseMeasurement, NoiseMeasurement } from '../types/noise-measurement'

export const useNoiseMeasurementStore = defineStore('noise-measurements', {
  state: () => ({ items: [] as NoiseMeasurement[], loading: false, error: '' }),
  actions: {
    async load() {
      this.loading = true; this.error = ''
      try { this.items = await noiseMeasurementApi.list() } catch (error) { this.error = errorMessage(error) } finally { this.loading = false }
    },
    async create(value: CreateNoiseMeasurement) { const created = await noiseMeasurementApi.create(value); await this.load(); return created },
    async transition(item: NoiseMeasurement, toState: MeasurementState) {
      const updated = await noiseMeasurementApi.transition(item.id, toState, item.version); await this.load(); return updated
    },
  },
})
