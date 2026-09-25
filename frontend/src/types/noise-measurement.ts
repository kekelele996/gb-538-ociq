import type { Spectrum } from './common'
import type { MeasurementQuality, MeasurementState } from './enums/measurement-quality'

export interface NoiseMeasurement {
  id: number
  monitoring_point_id: number
  point_code: string
  point_name: string
  measured_at: string
  duration_s: number
  octave_bands: Spectrum
  normalized_bands?: Spectrum
  overall_dba: number
  background_dba: number
  weather_note: string
  source_checksum: string
  measurement_quality: MeasurementQuality
  quality_reason: string
  measurement_state: MeasurementState
  imported_by: number
  version: number
  created_at: string
  updated_at: string
}

export interface CreateNoiseMeasurement {
  monitoring_point_id: number
  measured_at: string
  duration_s: number
  octave_bands: Spectrum
  overall_dba: number
  background_dba: number
  weather_note: string
  quality_reason: string
}
