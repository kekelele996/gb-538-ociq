export const OCTAVE_BANDS = [63, 125, 250, 500, 1000, 2000, 4000, 8000] as const
export type OctaveBand = typeof OCTAVE_BANDS[number]
export type Spectrum = Record<string, number>

export interface ApiEnvelope<T> {
  code: string
  message: string
  data: T
  request_id: string
}

export interface ApiErrorBody {
  code: string
  message: string
  request_id: string
}
