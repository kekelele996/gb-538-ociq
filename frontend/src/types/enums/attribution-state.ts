export type AttributionState = 'queued' | 'calculating' | 'completed' | 'failed' | 'reviewed' | 'confirmed' | 'voided'

export const attributionStateLabels: Record<AttributionState, string> = {
  queued: '排队中', calculating: '计算中', completed: '待复核', failed: '失败',
  reviewed: '已复核', confirmed: '已确认', voided: '已作废',
}
