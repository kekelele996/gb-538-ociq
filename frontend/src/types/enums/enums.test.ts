import { describe, expect, it } from 'vitest'
import { attributionStateLabels } from './attribution-state'
import { measurementQualityLabels, measurementStateLabels, measurementTransitions } from './measurement-quality'

describe('shared domain enums', () => {
  it('keeps every required measurement quality and state label', () => {
    expect(Object.keys(measurementQualityLabels)).toEqual(['valid', 'contaminated', 'clipped', 'missing'])
    expect(Object.keys(measurementStateLabels)).toEqual(['captured', 'validated', 'normalized', 'ready', 'rejected', 'superseded'])
    expect(measurementTransitions.captured).not.toContain('ready')
  })

  it('keeps all immutable attribution lifecycle labels', () => {
    expect(Object.keys(attributionStateLabels)).toEqual(['queued', 'calculating', 'completed', 'failed', 'reviewed', 'confirmed', 'voided'])
  })
})
