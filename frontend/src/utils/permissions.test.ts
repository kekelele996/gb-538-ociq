import { describe, expect, it } from 'vitest'
import { can } from './permissions'

describe('role capabilities', () => {
  it('keeps source editing away from analysts and auditors', () => {
    expect(can('acoustic_engineer', 'sourceWrite')).toBe(true)
    expect(can('data_analyst', 'sourceWrite')).toBe(false)
    expect(can('auditor', 'sourceWrite')).toBe(false)
  })

  it('requires a reviewer or admin for confirmation', () => {
    expect(can('reviewer', 'confirm')).toBe(true)
    expect(can('admin', 'confirm')).toBe(true)
    expect(can('acoustic_engineer', 'confirm')).toBe(false)
  })
})
