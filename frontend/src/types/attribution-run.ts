import type { AttributionState } from './enums/attribution-state'

export interface BandContribution {
  band_hz: number
  predicted_db: number
  energy_share: number
}

export interface SourceContribution {
  source_profile_id: number
  source_code: string
  source_name: string
  coefficient: number
  contribution_pct: number
  overall_db: number
  bands: BandContribution[]
}

export interface AttributionEvidence {
  matrix_rows: number
  matrix_columns: number
  iterations: number
  converged: boolean
  condition_hint: number
  unreliable_bands: string[]
  warnings: string[]
  objective: number
  elapsed_millis: number
}

export interface AttributionRun {
  id: number
  run_code: string
  measurement_ids: number[]
  source_profile_ids: number[]
  algorithm_version: string
  input_hash: string
  input_snapshot: unknown
  normalized_bands: unknown
  contributions: SourceContribution[]
  evidence: AttributionEvidence
  residual_error: number
  attribution_state: AttributionState
  explanation: string
  started_at: string
  finished_at: string | null
  created_by: number
  reviewed_by: number | null
  review_note: string
  version: number
  created_at: string
  updated_at: string
}

export interface CreateAttributionRun {
  measurement_ids: number[]
  source_profile_ids: number[]
}
