export const formatDate = (value: string | null | undefined) => value
  ? new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).format(new Date(value))
  : '—'

export const fixed = (value: number | null | undefined, places = 1) => Number.isFinite(value) ? Number(value).toFixed(places) : '—'

export const shortHash = (value: string | null | undefined) => value ? `${value.slice(0, 10)}…${value.slice(-6)}` : '—'
