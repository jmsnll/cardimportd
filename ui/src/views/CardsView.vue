<template>
  <div>
    <article v-if="error" class="message is-danger" role="alert">
      <div class="message-body">Failed to load cards: {{ error }}</div>
    </article>

    <div v-else-if="loading" class="has-text-centered py-6">
      <p class="has-text-grey">Loading cards…</p>
    </div>

    <div v-else-if="Object.keys(cards).length === 0" class="has-text-centered py-6">
      <p class="is-size-1 mb-3" aria-hidden="true">💾</p>
      <p class="is-size-5 has-text-weight-semibold mb-2">No cards registered</p>
      <p class="has-text-grey">
        Insert a card — the daemon will create a pending entry. Refresh to see it.
      </p>
    </div>

    <div v-else class="box p-0">
      <table class="table is-fullwidth is-hoverable is-striped mb-0" aria-label="Registered cards">
        <thead>
          <tr>
            <th scope="col">UUID</th>
            <th scope="col">Owner</th>
            <th scope="col">Status</th>
            <th scope="col">First Seen</th>
            <th scope="col"><span class="is-sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(entry, uuid) in cards" :key="uuid">
            <td class="uuid-cell">{{ uuid }}</td>

            <td>
              <template v-if="editingUuid === uuid">
                <div class="control">
                  <input
                    v-model="editOwner"
                    class="input is-small"
                    type="text"
                    placeholder="Owner name"
                    :aria-label="`Owner name for card ${uuid}`"
                    :ref="(el) => { if (el) editInputs[String(uuid)] = el as HTMLInputElement }"
                  />
                </div>
                <div class="control mt-1">
                  <input
                    v-model="editLabel"
                    class="input is-small"
                    type="text"
                    placeholder="Label (optional)"
                    :aria-label="`Label for card ${uuid}`"
                  />
                </div>
                <div class="control mt-1">
                  <input
                    v-model="editDestTemplate"
                    class="input is-small"
                    type="text"
                    placeholder="Destination template (optional)"
                    :aria-label="`Destination template for card ${uuid}`"
                  />
                </div>
              </template>
              <template v-else>
                <span v-if="!entry.owner" class="has-text-grey-light is-italic">unset</span>
                <span v-else>{{ entry.owner }}</span>
                <br v-if="entry.label" />
                <small v-if="entry.label" class="has-text-grey">{{ entry.label }}</small>
              </template>
            </td>

            <td>
              <span
                class="tag"
                :class="entry.status === 'active' ? 'is-success' : 'is-warning'"
              >{{ entry.status }}</span>
            </td>

            <td>{{ entry.first_seen ? new Date(entry.first_seen).toLocaleString() : '—' }}</td>

            <td>
              <div class="buttons are-small is-right">
                <template v-if="editingUuid === uuid">
                  <button class="button is-success is-small" @click="saveCard(String(uuid), 'active')">Activate</button>
                  <button class="button is-light is-small" @click="saveCard(String(uuid), 'pending')">Keep Pending</button>
                  <button class="button is-light is-small" @click="cancelEdit">Cancel</button>
                </template>
                <template v-else>
                  <button
                    class="button is-light is-small"
                    @click="startEdit(String(uuid), entry.owner, entry.label ?? '', entry.destination_template ?? '')"
                    :aria-label="`Edit card ${uuid}`"
                  >Edit</button>
                  <button
                    class="button is-danger is-light is-small"
                    @click="removeCard(String(uuid))"
                    :aria-label="`Remove card ${uuid}`"
                  >Remove</button>
                </template>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, inject, nextTick } from 'vue'
import { getCards, deleteCard } from '../api'
import type { CardEntry, CardStatus } from '../types'

const cards = ref<Record<string, CardEntry>>({})
const error = ref<string | null>(null)
const loading = ref(true)
const editingUuid = ref<string | null>(null)
const editOwner = ref('')
const editLabel = ref('')
const editDestTemplate = ref('')
const editInputs: Record<string, HTMLInputElement> = {}

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

onMounted(load)
</script>
