import type { Spectrum } from './common'

export interface SourceProfile {
  id: number
  source_code: string
  name: string
  x_m: number
  y_m: number
  height_m: number
  reference_distance_m: number
  octave_power: Spectrum
  directivity: Spectrum
  operating_factor: number
  profile_state: 'draft' | 'active' | 'retired'
  version: number
  lock_version: number
  created_by: number
  created_at: string
  updated_at: string
}

export interface CreateSourceProfile {
  source_code: string
  name: string
  x_m: number
  y_m: number
  height_m: number
  reference_distance_m: number
  octave_power: Spectrum
  directivity: Spectrum
  operating_factor: number
}
