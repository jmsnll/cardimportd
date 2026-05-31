export type CardStatus = 'active' | 'pending'

export interface CardEntry {
  owner: string
  label?: string
  status: CardStatus
  first_seen?: string
  destination_template?: string
}

export interface PushoverConfig {
  app_token: string
  user_key: string
  events?: string[]
}

export interface NtfyConfig {
  url: string
  token?: string
  events?: string[]
}

export interface WebhookConfig {
  url: string
  secret?: string
  events?: string[]
}

export interface NotificationConfig {
  pushover?: PushoverConfig
  ntfy?: NtfyConfig
  webhook?: WebhookConfig
}

export interface Config {
  watch_paths: string[]
  import_root: string
  min_free_gb?: number
  mirror_root?: string
  cards: Record<string, CardEntry>
  file_extensions: string[]
  log_path?: string
  notifications?: NotificationConfig
  post_import_hook?: string
  write_manifest?: boolean
  destination_template?: string
}
