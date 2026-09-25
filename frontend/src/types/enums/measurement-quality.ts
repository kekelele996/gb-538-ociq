export type MeasurementQuality = 'valid' | 'contaminated' | 'clipped' | 'missing'

export const measurementQualityLabels: Record<MeasurementQuality, string> = {
  valid: '有效',
  contaminated: '背景污染',
  clipped: '削顶',
  missing: '频带缺失',
}

export type MeasurementState = 'captured' | 'validated' | 'normalized' | 'ready' | 'rejected' | 'superseded'

export const measurementStateLabels: Record<MeasurementState, string> = {
  captured: '已采集', validated: '已校验', normalized: '已扣背景', ready: '可归因',
  rejected: '已拒绝', superseded: '已替代',
}

export const measurementTransitions: Partial<Record<MeasurementState, MeasurementState[]>> = {
  captured: ['validated', 'rejected'], validated: ['normalized', 'rejected'],
  normalized: ['ready', 'rejected'], ready: ['superseded'],
}
