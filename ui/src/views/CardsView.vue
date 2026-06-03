<template>
  <div>
    <!-- Error state -->
    <div v-if="error" class="rounded bg-red-50 border border-red-200 text-sm text-red-700 p-3 mb-4" role="alert">
      Failed to load cards: {{ error }}
    </div>

    <!-- Loading state -->
    <div v-else-if="loading" class="text-sm text-slate-500 py-8 text-center">Loading cards…</div>

    <template v-else>
      <!-- Header row -->
      <div class="flex items-center justify-between mb-4">
        <p class="text-sm text-slate-500">{{ Object.keys(cards).length }} card(s) registered</p>
        <button
          class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-40"
          @click="load"
          :disabled="loading"
        >Refresh</button>
      </div>

      <!-- Empty state -->
      <div v-if="Object.keys(cards).length === 0" class="text-center py-12">
        <p class="text-sm font-medium text-slate-900 mb-1">No cards registered</p>
        <p class="text-sm text-slate-500">Insert a card — the daemon will create a pending entry. Refresh to see it.</p>
      </div>

      <!-- Cards table -->
      <div v-else class="bg-white border border-slate-200 rounded-lg overflow-hidden">
        <table class="w-full text-sm" aria-label="Registered cards">
          <thead>
            <tr class="bg-slate-50">
              <th scope="col" class="px-4 py-2.5 text-left text-xs font-medium text-slate-500 uppercase tracking-wide">UUID</th>
              <th scope="col" class="px-4 py-2.5 text-left text-xs font-medium text-slate-500 uppercase tracking-wide">Owner</th>
              <th scope="col" class="px-4 py-2.5 text-left text-xs font-medium text-slate-500 uppercase tracking-wide">Status</th>
              <th scope="col" class="px-4 py-2.5 text-left text-xs font-medium text-slate-500 uppercase tracking-wide">First Seen</th>
              <th scope="col" class="px-4 py-2.5 text-left text-xs font-medium text-slate-500 uppercase tracking-wide">Last Import</th>
              <th scope="col" class="px-4 py-2.5 text-left text-xs font-medium text-slate-500 uppercase tracking-wide"><span class="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="(entry, uuid) in cards" :key="uuid" class="hover:bg-slate-50">
              <!-- UUID -->
              <td class="px-4 py-3 font-mono text-xs text-slate-600">{{ uuid }}</td>

              <!-- Owner (normal / edit) -->
              <td class="px-4 py-3 text-slate-700">
                <template v-if="editingUuid === uuid">
                  <select
                    v-if="users.length > 0"
                    v-model="editOwner"
                    class="block w-full rounded border border-slate-300 text-sm px-2 py-1.5 text-slate-900 focus:outline-none focus:ring-2 focus:ring-slate-500 mb-1.5"
                    :aria-label="`Owner for card ${uuid}`"
                  >
                    <option value="">— select owner —</option>
                    <option v-for="u in users" :key="u.name" :value="u.name">{{ u.name }}</option>
                  </select>
                  <input
                    v-else
                    v-model="editOwner"
                    class="block w-full rounded border border-slate-300 text-sm px-2 py-1.5 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 mb-1.5"
                    type="text"
                    placeholder="Owner name"
                    :aria-label="`Owner name for card ${uuid}`"
                    :ref="(el) => { if (el) editInputs[String(uuid)] = el as HTMLInputElement }"
                  />
                  <input
                    v-model="editLabel"
                    class="block w-full rounded border border-slate-300 text-sm px-2 py-1.5 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 mb-1.5"
                    type="text"
                    placeholder="Label (optional)"
                    :aria-label="`Label for card ${uuid}`"
                  />
                  <input
                    v-model="editDestTemplate"
                    class="block w-full rounded border border-slate-300 text-sm px-2 py-1.5 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 mb-1.5"
                    type="text"
                    placeholder="Destination template (optional)"
                    :aria-label="`Destination template for card ${uuid}`"
                  />
                </template>
                <template v-else>
                  <span v-if="!entry.owner" class="italic text-slate-400">unset</span>
                  <span v-else>{{ entry.owner }}</span>
                  <br v-if="entry.label" />
                  <small v-if="entry.label" class="text-slate-500">{{ entry.label }}</small>
                </template>
              </td>

              <!-- Status badges -->
              <td class="px-4 py-3 text-slate-700">
                <span
                  v-if="entry.status === 'active'"
                  class="rounded-full px-2 py-0.5 text-xs font-medium bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-600/20"
                >{{ entry.status }}</span>
                <span
                  v-else
                  class="rounded-full px-2 py-0.5 text-xs font-medium bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20"
                >{{ entry.status }}</span>
                <span
                  v-if="mountedUuids.has(String(uuid))"
                  class="ml-1.5 rounded-full px-2 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-600/20"
                  title="Currently inserted"
                >Mounted</span>
              </td>

              <!-- First Seen -->
              <td class="px-4 py-3 text-slate-700">{{ entry.first_seen ? new Date(entry.first_seen).toLocaleString() : '—' }}</td>

              <!-- Last Import -->
              <td class="px-4 py-3 text-slate-700">
                {{ lastImport[String(uuid)] ? new Date(lastImport[String(uuid)]).toLocaleString() : '—' }}
              </td>

              <!-- Actions -->
              <td class="px-4 py-3 text-right">
                <div class="flex items-center justify-end gap-2">
                  <template v-if="editingUuid === uuid">
                    <button
                      class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700 disabled:opacity-40"
                      @click="saveCard(String(uuid), 'active')"
                    >Activate</button>
                    <button
                      class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-amber-50 text-amber-700 border border-amber-200 hover:bg-amber-100"
                      @click="saveCard(String(uuid), 'pending')"
                    >Keep Pending</button>
                    <button
                      class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-40"
                      @click="cancelEdit"
                    >Cancel</button>
                  </template>
                  <template v-else>
                    <button
                      v-if="entry.status === 'pending'"
                      class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-amber-50 text-amber-700 border border-amber-200 hover:bg-amber-100"
                      @click="activatingUuid = String(uuid)"
                      :aria-label="`Register card ${uuid}`"
                    >Register</button>
                    <button
                      v-if="mountedUuids.has(String(uuid))"
                      class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-40"
                      @click="runPreflight(String(uuid))"
                      :disabled="preflightLoading.has(String(uuid))"
                    >{{ preflightLoading.has(String(uuid)) ? 'Scanning…' : 'Preflight' }}</button>
                    <button
                      class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-40"
                      @click="startEdit(String(uuid), entry.owner, entry.label ?? '', entry.destination_template ?? '')"
                      :aria-label="`Edit card ${uuid}`"
                    >Edit</button>
                    <button
                      class="inline-flex items-center rounded px-2 py-1 text-xs font-medium text-red-600 hover:text-red-700"
                      @click="removeCard(String(uuid))"
                      :aria-label="`Remove card ${uuid}`"
                    >Remove</button>
                  </template>
                </div>
                <p v-if="preflightResults[String(uuid)]" class="text-xs text-slate-500 mt-1 text-right">
                  {{ preflightResults[String(uuid)].total_on_card }} files on card
                  · {{ preflightResults[String(uuid)].to_import }} to import
                </p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <ActivateCardModal
      v-if="activatingUuid"
      :uuid="activatingUuid"
      :users="users"
      @confirm="activateCard"
      @cancel="activatingUuid = null"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, inject, nextTick } from 'vue'
import { getCards, deleteCard } from '../api'
import type { CardEntry, CardStatus } from '../types'
import ActivateCardModal from '../components/ActivateCardModal.vue'

const cards = ref<Record<string, CardEntry>>({})
const error = ref<string | null>(null)
const loading = ref(true)
const editingUuid = ref<string | null>(null)
const editOwner = ref('')
const editLabel = ref('')
const editDestTemplate = ref('')
const editInputs: Record<string, HTMLInputElement> = {}

const users = ref<Array<{ name: string }>>([])
const activatingUuid = ref<string | null>(null)

const mountedUuids = ref<Set<string>>(new Set())
const lastImport = ref<Record<string, string>>({})
const preflightLoading = ref<Set<string>>(new Set())
const preflightResults = ref<Record<string, { total_on_card: number; to_import: number }>>({})

const showToast = inject<(msg: string, type?: 'success' | 'error') => void>('showToast')!

async function load() {
  error.value = null
  loading.value = true
  try {
    cards.value = await getCards()
  } catch (err) {
    error.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

async function loadUsers() {
  try {
    const res = await fetch('/api/users')
    if (res.ok) users.value = await res.json()
  } catch { /* non-fatal */ }
}

async function loadStatus() {
  try {
    const res = await fetch('/api/status')
    if (!res.ok) return
    const data = await res.json()
    mountedUuids.value = new Set((data.mounted_cards ?? []).map((m: { uuid: string }) => m.uuid))
  } catch {
    // non-fatal
  }
}

async function loadHistory() {
  try {
    const res = await fetch('/api/history?limit=200')
    if (!res.ok) return
    const entries: Array<{ uuid: string; completed_at: string }> = await res.json()
    const map: Record<string, string> = {}
    for (const e of entries) {
      if (!map[e.uuid]) map[e.uuid] = e.completed_at  // entries are newest-first
    }
    lastImport.value = map
  } catch {
    // non-fatal
  }
}

async function runPreflight(uuid: string) {
  if (preflightLoading.value.has(uuid)) return  // debounce
  preflightLoading.value = new Set([...preflightLoading.value, uuid])
  try {
    const res = await fetch(`/api/preflight/${encodeURIComponent(uuid)}`)
    if (!res.ok) { showToast('Preflight failed', 'error'); return }
    const data = await res.json()
    preflightResults.value = { ...preflightResults.value, [uuid]: data }
  } catch (err) {
    showToast((err as Error).message, 'error')
  } finally {
    preflightLoading.value = new Set([...preflightLoading.value].filter(u => u !== uuid))
  }
}

function onCardsRefresh() {
  void load()
  void loadStatus()
}

function startEdit(uuid: string, owner: string, label: string, destTemplate: string) {
  editingUuid.value = uuid
  editOwner.value = owner ?? ''
  editLabel.value = label ?? ''
  editDestTemplate.value = destTemplate ?? ''
  nextTick(() => {
    editInputs[uuid]?.focus()
  })
}

function cancelEdit() {
  editingUuid.value = null
  editOwner.value = ''
  editLabel.value = ''
  editDestTemplate.value = ''
}

async function activateCard(payload: { owner: string; label: string; status: 'active' | 'pending' }) {
  if (!activatingUuid.value) return
  const uuid = activatingUuid.value
  activatingUuid.value = null
  try {
    const res = await fetch(`/api/cards/${uuid}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ owner: payload.owner, label: payload.label, status: payload.status }),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error((body as { error?: string }).error ?? 'HTTP ' + res.status)
    showToast(payload.status === 'active' ? 'Card activated' : 'Card saved')
    await load()
  } catch (err) {
    showToast((err as Error).message, 'error')
  }
}

async function saveCard(uuid: string, status: CardStatus) {
  try {
    const res = await fetch(`/api/cards/${uuid}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        owner: editOwner.value.trim(),
        label: editLabel.value.trim(),
        destination_template: editDestTemplate.value.trim(),
        status,
      }),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error((body as { error?: string }).error ?? 'HTTP ' + res.status)
    showToast('Card updated')
    editingUuid.value = null
    await load()
  } catch (err) {
    showToast((err as Error).message, 'error')
  }
}

async function removeCard(uuid: string) {
  if (!confirm(`Remove card ${uuid}? This only removes the registration.`)) return
  try {
    await deleteCard(uuid)
    showToast('Card removed')
    await load()
  } catch (err) {
    showToast((err as Error).message, 'error')
  }
}

onMounted(() => {
  window.addEventListener('cards:refresh', onCardsRefresh)
  void load()
  void loadUsers()
  void loadStatus()
  void loadHistory()
})

onUnmounted(() => {
  window.removeEventListener('cards:refresh', onCardsRefresh)
})
</script>
