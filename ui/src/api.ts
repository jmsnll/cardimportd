import type { Config, CardEntry, CardStatus } from './types'

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

export function testNotification(): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>('/api/notify/test', {
    method: 'POST',
  })
}

export function browseFSDir(path: string): Promise<{ path: string; entries: string[] }> {
  return apiFetch(`/api/fs?path=${encodeURIComponent(path)}`)
}
