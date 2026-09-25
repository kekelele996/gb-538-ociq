import { api, json } from './client'
import type { MeasurementState } from '../types/enums/measurement-quality'
import type { CreateNoiseMeasurement, NoiseMeasurement } from '../types/noise-measurement'

export const noiseMeasurementApi = {
  list: () => api<NoiseMeasurement[]>('/noise-measurements'),
  get: (id: number) => api<NoiseMeasurement>(`/noise-measurements/${id}`),
  create: (value: CreateNoiseMeasurement) => api<NoiseMeasurement>('/noise-measurements', json('POST', value)),
  transition: (id: number, toState: MeasurementState, version: number) =>
    api<NoiseMeasurement>(`/noise-measurements/${id}/transition`, json('POST', { to_state: toState, version })),
}
