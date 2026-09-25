import type { SourceProfile } from '../types/source-profile'

/** 同一设备的全部版本按版本号升序排列，形成版本链。 */
export function lineageOf(items: SourceProfile[], sourceCode: string): SourceProfile[] {
  return items.filter((item) => item.source_code === sourceCode).sort((a, b) => a.version - b.version)
}

/** 版本链中当前启用（进入归因候选）的版本，至多一个。 */
export function activeOf(lineage: SourceProfile[]): SourceProfile | null {
  return lineage.find((item) => item.profile_state === 'active') ?? null
}
