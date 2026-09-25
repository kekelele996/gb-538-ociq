import type { Spectrum } from './common'

export interface MonitoringPoint {
  id: number
  point_code: string
  name: string
  x_m: number
  y_m: number
  height_m: number
  area_type: 'boundary' | 'workshop' | 'office' | 'residential'
  background_profile: Spectrum
  owner_team: string
  point_state: 'active' | 'inactive'
  version: number
  measurement_count: number
  latest_quality: string
  latest_measurement_time: string | null
  created_at: string
  updated_at: string
}

export interface CreateMonitoringPoint {
  point_code: string
  name: string
  x_m: number
  y_m: number
  height_m: number
  area_type: MonitoringPoint['area_type']
  background_profile: Spectrum
  owner_team: string
}
