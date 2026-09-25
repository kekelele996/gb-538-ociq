import type { ApiEnvelope, ApiErrorBody } from '../types/common'

export const tokenKey = 'noisetrace_access_token'
export const userKey = 'noisetrace_current_user'

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly requestId: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  const token = localStorage.getItem(tokenKey)
  if (options.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const response = await fetch(`/api/v1${path}`, { ...options, headers })
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | ApiErrorBody | null
  if (!response.ok) {
    const error = payload as ApiErrorBody | null
    if (response.status === 401) {
      localStorage.removeItem(tokenKey)
      localStorage.removeItem(userKey)
      window.dispatchEvent(new Event('noisetrace-auth-expired'))
    }
    throw new ApiError(
      response.status,
      error?.code ?? 'http_error',
      error?.message ?? `请求失败（HTTP ${response.status}）`,
      error?.request_id ?? response.headers.get('X-Request-ID') ?? '',
    )
  }
  return (payload as ApiEnvelope<T>).data
}

export const api = <T>(path: string, options?: RequestInit) => request<T>(path, options)

export const json = (method: string, body?: unknown, headers?: HeadersInit): RequestInit => ({
  method,
  headers,
  body: body === undefined ? undefined : JSON.stringify(body),
})

export const errorMessage = (error: unknown) => error instanceof ApiError
  ? [error.message, error.requestId && `Request ID: ${error.requestId}`].filter(Boolean).join(' · ')
  : error instanceof Error ? error.message : '请求未完成，请稍后重试。'

export function query(params: Record<string, string | number | undefined>) {
  const values = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') values.set(key, String(value))
  })
  return values.size ? `?${values.toString()}` : ''
}
