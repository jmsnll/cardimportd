<template>
  <div>
    <div v-if="loading" class="has-text-grey py-4">Loading…</div>
    <div v-else-if="error" class="notification is-danger is-light">{{ error }}</div>
    <div v-else class="columns">
      <div class="column is-two-thirds">

        <div class="box">
          <h2 class="title is-5 mb-3">Currently mounted</h2>
          <div v-if="data!.mounted_cards.length === 0" class="has-text-grey is-size-7 py-2">
            No cards detected. Insert a card reader to begin.
          </div>
          <div v-else>
            <div
              v-for="card in data!.mounted_cards"
              :key="card.uuid"
              class="level is-mobile mb-3 pb-3"
              style="border-bottom: 1px solid #f0f0f0"
            >
              <div class="level-left">
                <div class="level-item">
                  <div>
                    <p class="has-text-weight-semibold is-size-7">{{ cardLabel(card.uuid) }}</p>
                    <p class="is-size-7 has-text-grey">{{ card.mount_point }} · {{ card.fstype }}</p>
                  </div>
                </div>
              </div>
              <div class="level-right">
                <div class="level-item">
                  <span v-if="cardStatus(card.uuid) === 'pending'" class="tag is-warning is-light mr-2">Pending</span>
                  <span v-else class="tag is-success is-light mr-2">Active</span>
                  <button
                    v-if="cardStatus(card.uuid) === 'active'"
                    class="button is-small is-link"
                    :class="{ 'is-loading': importing.has(card.uuid) }"
                    :disabled="importing.has(card.uuid) || !!data!.active_import"
                    @click="runImport(card.uuid)"
                  >Import now</button>
                </div>
              </div>
            </div>
          </div>

          <div v-if="data!.active_import" class="notification is-info is-light mt-3 mb-0 py-2 px-3">
            <span class="has-text-weight-semibold">Importing:</span>
            {{ data!.active_import.owner }} —
            <template v-if="data!.active_import.total > 0">
              {{ data!.active_import.imported }}/{{ data!.active_import.total }} files
            </template>
            <template v-else>
              {{ data!.active_import.imported }} files
            </template>
            ({{ formatBytes(data!.active_import.bytes_copied) }})
          </div>
        </div>

        <div v-if="Object.keys(data!.pending_cards).length > 0" class="box">
          <h2 class="title is-5 mb-2">Pending cards</h2>
          <p class="is-size-7 has-text-grey mb-3">These cards need to be assigned an owner before imports will run.</p>
          <ul>
            <li v-for="(entry, uuid) in data!.pending_cards" :key="uuid" class="is-size-7 mb-1">
              <span class="tag is-warning is-light mr-1">pending</span>
              <span class="has-text-weight-semibold">{{ uuid }}</span>
              <span v-if="entry.first_seen" class="has-text-grey ml-2">first seen {{ formatDate(entry.first_seen) }}</span>
            </li>
          </ul>
          <p class="is-size-7 mt-2 has-text-grey">→ Go to the <strong>Cards</strong> tab to activate.</p>
        </div>
      </div>

      <div class="column">
        <div class="box">
          <h2 class="title is-5 mb-3">Storage</h2>
          <div v-if="!data!.storage" class="has-text-grey is-size-7">Storage stats unavailable.</div>
          <div v-else>
            <progress
              class="progress is-info mb-1"
              :value="data!.storage.total_bytes - data!.storage.available_bytes"
              :max="data!.storage.total_bytes"
            ></progress>
            <p class="is-size-7 has-text-grey">
              {{ formatGB(data!.storage.total_bytes - data!.storage.available_bytes) }} used of
              {{ formatGB(data!.storage.total_bytes) }} total
              ({{ formatGB(data!.storage.free_bytes) }} free)
            </p>
          </div>
        </div>

        <div class="box">
          <h2 class="title is-5 mb-3">Recent imports</h2>
          <div v-if="data!.recent_history.length === 0" class="has-text-grey is-size-7">No imports yet.</div>
          <table v-else class="table is-narrow is-fullwidth is-size-7">
            <thead>
              <tr>
                <th>Owner</th>
                <th>Date</th>
                <th class="has-text-right">Files</th>
                <th class="has-text-right">Size</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(entry, i) in data!.recent_history" :key="i">
                <td>{{ entry.owner }}</td>
                <td>{{ formatDate(entry.completed_at) }}</td>
                <td class="has-text-right">{{ entry.imported }}</td>
                <td class="has-text-right">{{ formatBytes(entry.bytes_copied) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, inject } from 'vue'
import { getDashboard, triggerImport } from '../api'
import type { DashboardResponse } from '../types'

const showToast = inject<(msg: string, type?: 'success' | 'error') => void>('showToast')!

const data = ref<DashboardResponse | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const importing = ref<Set<string>>(new Set())

async function load() {
  try {
    data.value = await getDashboard()
    error.value = null
  } catch (err) {
    error.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

function cardLabel(uuid: string): string {
  const entry = data.value?.cards[uuid]
  if (entry?.label) return entry.label
  if (entry?.owner) return entry.owner
  return uuid.slice(0, 8) + '…'
}

function cardStatus(uuid: string): 'active' | 'pending' {
  return data.value?.cards[uuid]?.status ?? 'pending'
}

async function runImport(uuid: string) {
  importing.value = new Set([...importing.value, uuid])
  try {
    await triggerImport(uuid)
    showToast('Import started')
    await load()
  } catch (err) {
    showToast((err as Error).message, 'error')
  } finally {
    const next = new Set(importing.value)
    next.delete(uuid)
    importing.value = next
  }
}

function formatBytes(bytes: number): string {
  if (bytes >= 1e9) return (bytes / 1e9).toFixed(1) + ' GB'
  if (bytes >= 1e6) return (bytes / 1e6).toFixed(1) + ' MB'
  if (bytes >= 1e3) return (bytes / 1e3).toFixed(1) + ' KB'
  return bytes + ' B'
}

function formatGB(bytes: number): string {
  return (bytes / 1e9).toFixed(1) + ' GB'
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function onCardsRefresh() { load() }

onMounted(() => {
  load()
  window.addEventListener('cards:refresh', onCardsRefresh)
})

onUnmounted(() => {
  window.removeEventListener('cards:refresh', onCardsRefresh)
})
</script>
