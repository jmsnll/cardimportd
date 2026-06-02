import type { Config, CardEntry, CardStatus, HistoryEntry, StatusResponse, PreflightResult, User, DashboardResponse } from './types'

async function apiFetch<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((body as { error?: string }).error ?? 'HTTP ' + res.status)
  return body as T
}

export function getConfig(): Promise<Config> {
  return apiFetch<Config>('/api/config')
}

export function saveConfig(cfg: Config): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>('/api/config', {
    method: 'POST',
    body: JSON.stringify(cfg),
  })
}

export function getCards(): Promise<Record<string, CardEntry>> {
  return apiFetch<Record<string, CardEntry>>('/api/cards')
}

export function updateCard(uuid: string, owner: string, status: CardStatus): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>(`/api/cards/${uuid}`, {
    method: 'POST',
    body: JSON.stringify({ owner, status }),
  })
}

export function deleteCard(uuid: string): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>(`/api/cards/${uuid}`, {
    method: 'DELETE',
  })
}

export function testNotification(adapter?: string): Promise<{ ok: boolean }> {
  const qs = adapter ? `?adapter=${encodeURIComponent(adapter)}` : ''
  return apiFetch<{ ok: boolean }>(`/api/notify/test${qs}`, {
    method: 'POST',
  })
}

export function browseFSDir(path: string): Promise<{ path: string; entries: string[] }> {
  return apiFetch(`/api/fs?path=${encodeURIComponent(path)}`)
}

export function getHistory(limit = 50): Promise<HistoryEntry[]> {
  return apiFetch<HistoryEntry[]>(`/api/history?limit=${limit}`)
}

export function getStatus(): Promise<StatusResponse> {
  return apiFetch<StatusResponse>('/api/status')
}

export function getPreflight(uuid: string): Promise<PreflightResult> {
  return apiFetch<PreflightResult>(`/api/preflight/${encodeURIComponent(uuid)}`)
}

export function getUsers(): Promise<User[]> {
  return apiFetch<User[]>('/api/users')
}

export function createUser(user: User): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>('/api/users', { method: 'POST', body: JSON.stringify(user) })
}

export function updateUser(name: string, patch: Partial<User>): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>(`/api/users/${encodeURIComponent(name)}`, { method: 'PUT', body: JSON.stringify(patch) })
}

export function deleteUser(name: string): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>(`/api/users/${encodeURIComponent(name)}`, { method: 'DELETE' })
}

export function getDashboard(): Promise<DashboardResponse> {
  return apiFetch<DashboardResponse>('/api/dashboard')
}

export function triggerImport(uuid: string): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>(`/api/cards/${encodeURIComponent(uuid)}/import`, {
    method: 'POST',
  })
}
