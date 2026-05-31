export type CardStatus = 'active' | 'pending'

export interface CardEntry {
  owner: string
  status: CardStatus
  first_seen?: string
}

export interface PushoverConfig {
  app_token: string
  user_key: string
}

export interface NotificationConfig {
  pushover?: PushoverConfig
}

export interface Config {
  watch_paths: string[]
  import_root: string
  cards: Record<string, CardEntry>
  file_extensions: string[]
  log_path?: string
  notifications?: NotificationConfig
}
