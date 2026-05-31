import { ref, onUnmounted } from 'vue'

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
}

export function useEventStream() {
  const activeImport = ref<ImportProgress | null>(null)
  let es: EventSource | null = null
  let retryTimer: ReturnType<typeof setTimeout> | null = null
  let retryDelay = 1000

  function connect() {
    if (es) {
      es.close()
    }
    es = new EventSource('/api/events')

    es.onmessage = (e) => {
      try {
        const evt = JSON.parse(e.data) as ImportProgress
        if (evt.event === 'import_completed' || evt.event === 'import_failed') {
          activeImport.value = evt
          setTimeout(() => {
            if (activeImport.value?.card_uuid === evt.card_uuid) {
              activeImport.value = null
            }
          }, 4000)
        } else {
          activeImport.value = evt
        }
        retryDelay = 1000
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

  return { activeImport }
}
