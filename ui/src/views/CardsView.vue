<template>
  <div>
    <div v-if="error" class="note">Failed to load cards: {{ error }}</div>

    <div v-else-if="Object.keys(cards).length === 0" class="empty-state">
      <span class="empty-state-icon">&#128190;</span>
      <div class="empty-state-title">No cards registered</div>
      <div class="empty-state-body">
        Insert a card &mdash; the daemon will create a pending entry. Refresh to see it.
      </div>
    </div>

    <div v-else class="surface">
      <table class="card-table">
        <thead>
          <tr>
            <th>UUID</th>
            <th>Owner</th>
            <th>Status</th>
            <th>First Seen</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(entry, uuid) in cards" :key="uuid">
            <td class="uuid-cell">{{ uuid }}</td>

            <td>
              <template v-if="editingUuid === uuid">
                <input
                  v-model="editOwner"
                  class="inline-edit-input"
                  placeholder="Owner name"
                  ref="editInput"
                />
              </template>
              <template v-else>
                <em v-if="!entry.owner" style="color:#aaa">unset</em>
                <span v-else>{{ entry.owner }}</span>
              </template>
            </td>

            <td>
              <span class="badge" :class="`badge-${entry.status}`">{{ entry.status }}</span>
            </td>

            <td>{{ entry.first_seen ? new Date(entry.first_seen).toLocaleString() : '—' }}</td>

            <td class="actions-cell">
              <template v-if="editingUuid === uuid">
                <button class="btn btn-success btn-sm" @click="saveCard(String(uuid), 'active')">Activate</button>
                <button class="btn btn-secondary btn-sm" @click="saveCard(String(uuid), 'pending')">Keep Pending</button>
                <button class="btn btn-secondary btn-sm" @click="cancelEdit">Cancel</button>
              </template>
              <template v-else>
                <button class="btn btn-secondary btn-sm" @click="startEdit(String(uuid), entry.owner)">Edit</button>
                <button class="btn btn-danger btn-sm" @click="removeCard(String(uuid))">Remove</button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, inject, nextTick } from 'vue'
import type { Ref } from 'vue'
import { getCards, updateCard, deleteCard } from '../api'
import type { CardEntry, CardStatus } from '../types'

const cards = ref<Record<string, CardEntry>>({})
const error = ref<string | null>(null)
const editingUuid = ref<string | null>(null)
const editOwner = ref('')
const editInput = ref<HTMLInputElement | null>(null)

const showToast = inject<(msg: string, type?: 'success' | 'error') => void>('showToast')!

async function load() {
  error.value = null
  try {
    cards.value = await getCards()
  } catch (err) {
    error.value = (err as Error).message
  }
}

function startEdit(uuid: string, owner: string) {
  editingUuid.value = uuid
  editOwner.value = owner ?? ''
  nextTick(() => {
    if (editInput.value) editInput.value.focus()
  })
}

function cancelEdit() {
  editingUuid.value = null
  editOwner.value = ''
}

async function saveCard(uuid: string, status: CardStatus) {
  try {
    await updateCard(uuid, editOwner.value.trim(), status)
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
