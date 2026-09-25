import { describe, expect, it } from 'vitest'
import { currentActive, expectedActiveId, siblingVersions, supersededBy } from './source-versions'
import type { SourceProfile } from '../types/source-profile'

function profile(id: number, code: string, version: number, state: SourceProfile['profile_state'], supersededById: number | null = null): SourceProfile {
  return {
    id, source_code: code, name: `${code} device`, x_m: 0, y_m: 0, height_m: 1,
    reference_distance_m: 1, octave_power: {}, directivity: {}, operating_factor: 1,
    profile_state: state, version, lock_version: 1, superseded_by_id: supersededById,
    created_by: 1, created_at: '2026-09-25T00:00:00Z', updated_at: '2026-09-25T00:00:00Z',
  }
}

const items = [
  profile(3, 'SRC-A', 3, 'draft'),
  profile(1, 'SRC-A', 1, 'retired', 2),
  profile(2, 'SRC-A', 2, 'active'),
  profile(4, 'SRC-B', 1, 'draft'),
]

describe('source version lineage helpers', () => {
  it('lists sibling versions of one device in ascending order', () => {
    expect(siblingVersions(items, 'SRC-A').map((item) => item.version)).toEqual([1, 2, 3])
    expect(siblingVersions(items, 'SRC-B').map((item) => item.version)).toEqual([1])
  })

  it('finds the single active version excluding the target itself', () => {
    expect(currentActive(items, 'SRC-A')?.id).toBe(2)
    expect(currentActive(items, 'SRC-A', 2)).toBeNull()
    expect(currentActive(items, 'SRC-B')).toBeNull()
  })

  it('declares the expected active version for an enable request', () => {
    expect(expectedActiveId(items, items[0])).toBe(2)
    expect(expectedActiveId(items, items[2])).toBeNull()
    expect(expectedActiveId(items, items[3])).toBeNull()
  })

  it('resolves which version replaced a retired one', () => {
    expect(supersededBy(items, items[1])?.version).toBe(2)
    expect(supersededBy(items, items[2])).toBeNull()
  })
})
