import type { SourceProfile } from '../types/source-profile'

/** 同一声源编号的全部版本，按版本号升序，用于页面上的版本关系链。 */
export const siblingVersions = (items: SourceProfile[], sourceCode: string) =>
  items.filter((item) => item.source_code === sourceCode).sort((a, b) => a.version - b.version)

/** 同一声源编号当前启用中的版本（可排除某个版本自身）；没有时返回 null。 */
export const currentActive = (items: SourceProfile[], sourceCode: string, excludeId?: number) =>
  siblingVersions(items, sourceCode).find((item) => item.profile_state === 'active' && item.id !== excludeId) ?? null

/** 启用/切回时向服务端声明的当前启用版本 ID；认为没有启用版本时为 null。 */
export const expectedActiveId = (items: SourceProfile[], target: SourceProfile) =>
  currentActive(items, target.source_code, target.id)?.id ?? null

/** 替代了指定版本的那个版本（“被 V3 替代”标注）；未被替代时返回 null。 */
export const supersededBy = (items: SourceProfile[], item: SourceProfile) =>
  item.superseded_by_id == null ? null : items.find((candidate) => candidate.id === item.superseded_by_id) ?? null
