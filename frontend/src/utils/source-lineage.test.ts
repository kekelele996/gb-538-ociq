import { describe, expect, it } from 'vitest'
import { activeOf, lineageOf } from './source-lineage'
import type { SourceProfile } from '../types/source-profile'

function profile(id: number, sourceCode: string, version: number, state: SourceProfile['profile_state']): SourceProfile {
  return {
    id, source_code: sourceCode, name: `Source ${sourceCode}`,
    x_m: 0, y_m: 0, height_m: 1, reference_distance_m: 1,
    octave_power: {}, directivity: {}, operating_factor: 0.8,
    profile_state: state, version, lock_version: 1, created_by: 1,
    created_at: '2026-09-25T00:00:00Z', updated_at: '2026-09-25T00:00:00Z',
  }
}

describe('source version lineage', () => {
  const items = [
    profile(3, 'SRC-A', 3, 'draft'),
    profile(1, 'SRC-A', 1, 'retired'),
    profile(2, 'SRC-A', 2, 'active'),
    profile(4, 'SRC-B', 1, 'active'),
  ]

  it('groups only same-device versions ordered by version', () => {
    const lineage = lineageOf(items, 'SRC-A')
    expect(lineage.map((item) => item.version)).toEqual([1, 2, 3])
  })

  it('finds at most one active candidate per device', () => {
    expect(activeOf(lineageOf(items, 'SRC-A'))?.version).toBe(2)
    expect(activeOf(lineageOf(items, 'SRC-B'))?.version).toBe(1)
  })

  it('returns null when no version is active', () => {
    const retiredOnly = [profile(5, 'SRC-C', 1, 'retired'), profile(6, 'SRC-C', 2, 'draft')]
    expect(activeOf(lineageOf(retiredOnly, 'SRC-C'))).toBeNull()
  })
})
