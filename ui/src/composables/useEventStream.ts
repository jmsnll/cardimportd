import { ref, onUnmounted } from 'vue'
import { getStatus } from '../api'
import type { MountedCard } from '../types'

export interface ImportProgress {
  event: string
  owner: string
  card_uuid: string
  total: number
  imported: number
  skipped: number
  failed: number
  bytes_copied: number
  error?: string
  mount_point?: string
  fstype?: string
}

export function useEventStream() {
  const activeImport = ref<ImportProgress | null>(null)
  const mountedCards = ref<MountedCard[]>([])
  let es: EventSource | null = null
  let retryTimer: ReturnType<typeof setTimeout> | null = null
  let retryDelay = 1000

  async function onConnect() {
    try {
      const status = await getStatus()
      mountedCards.value = status.mounted_cards ?? []
      if (status.active_import) {
        activeImport.value = status.active_import
      }
    } catch {
      // non-fatal — SSE events will fill in state
    }
  }

  function connect() {
    if (es) es.close()
    es = new EventSource('/api/events')

    es.onopen = () => {
      retryDelay = 1000
      void onConnect()
    }

    es.onmessage = (e) => {
      try {
        const evt = JSON.parse(e.data) as ImportProgress
        if (evt.event === 'card_detected' || evt.event === 'card_removed') {
          window.dispatchEvent(new CustomEvent('cards:refresh'))
          void onConnect() // resync mounted cards
          return
        }
        if (evt.event === 'import_completed' || evt.event === 'import_failed') {
          activeImport.value = evt
          setTimeout(() => {
            if (activeImport.value?.card_uuid === evt.card_uuid) {
              activeImport.value = null
            }
          }, 5000)
        } else {
          activeImport.value = evt
        }
      } catch {
        // ignore malformed events
      }
    }

    es.onerror = () => {
      es?.close()
      es = null
      retryTimer = setTimeout(() => {
        retryDelay = Math.min(retryDelay * 2, 30000)
        connect()
      }, retryDelay)
    }
  }

  connect()

  onUnmounted(() => {
    es?.close()
    if (retryTimer) clearTimeout(retryTimer)
  })

  return { activeImport, mountedCards }
}
