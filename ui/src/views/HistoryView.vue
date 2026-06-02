<template>
  <div>
    <div class="level mb-4">
      <div class="level-left">
        <div class="level-item">
          <h2 class="title is-5 mb-0">Import History</h2>
        </div>
      </div>
      <div class="level-right">
        <div class="level-item">
          <div class="select is-small">
            <select v-model="ownerFilter" aria-label="Filter by owner">
              <option value="">All owners</option>
              <option v-for="owner in uniqueOwners" :key="owner" :value="owner">{{ owner }}</option>
            </select>
          </div>
        </div>
        <div class="level-item ml-2">
          <button class="button is-small" @click="load" :disabled="loading">Refresh</button>
        </div>
      </div>
    </div>

    <article v-if="error" class="message is-danger" role="alert">
      <div class="message-body">Failed to load history: {{ error }}</div>
    </article>

    <div v-else-if="loading" class="has-text-centered py-6">
      <p class="has-text-grey">Loading history…</p>
    </div>

    <div v-else-if="filtered.length === 0" class="has-text-centered py-6">
      <p class="is-size-1 mb-3" aria-hidden="true">📋</p>
      <p class="is-size-5 has-text-weight-semibold mb-2">No imports yet</p>
      <p class="has-text-grey">History will appear here after the first import completes.</p>
    </div>

    <div v-else class="box p-0">
      <table class="table is-fullwidth is-hoverable is-striped mb-0" aria-label="Import history">
        <thead>
          <tr>
            <th scope="col">Date</th>
            <th scope="col">Owner</th>
            <th scope="col">Card</th>
            <th scope="col" class="has-text-right">Imported</th>
            <th scope="col" class="has-text-right">Skipped</th>
            <th scope="col" class="has-text-right">Failed</th>
            <th scope="col" class="has-text-right">Size</th>
            <th scope="col" class="has-text-right">Duration</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="entry in filtered" :key="entry.started_at + entry.uuid">
            <td>{{ formatDate(entry.started_at) }}</td>
            <td>{{ entry.owner || '—' }}</td>
            <td class="uuid-cell">{{ entry.uuid }}</td>
            <td class="has-text-right">
              <span class="tag is-success is-light">{{ entry.imported }}</span>
            </td>
            <td class="has-text-right">
              <span class="tag is-light">{{ entry.skipped }}</span>
            </td>
            <td class="has-text-right">
              <span :class="['tag', entry.failed > 0 ? 'is-danger is-light' : 'is-light']">{{ entry.failed }}</span>
            </td>
            <td class="has-text-right has-text-grey is-size-7">{{ formatBytes(entry.bytes_copied) }}</td>
            <td class="has-text-right has-text-grey is-size-7">{{ formatDuration(entry.started_at, entry.completed_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="!loading && history.length >= limit" class="mt-4 has-text-centered">
      <button class="button is-small is-light" @click="loadMore">Load more</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getHistory } from '../api'
import type { HistoryEntry } from '../types'

const history = ref<HistoryEntry[]>([])
const error = ref<string | null>(null)
const loading = ref(true)
const ownerFilter = ref('')
const limit = ref(50)

const filtered = computed(() =>
  ownerFilter.value
    ? history.value.filter(e => e.owner === ownerFilter.value)
    : history.value
)

const uniqueOwners = computed(() =>
  [...new Set(history.value.map(e => e.owner).filter(Boolean))].sort()
)

async function load() {
  error.value = null
  loading.value = true
  try {
    history.value = await getHistory(limit.value)
  } catch (err) {
    error.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  limit.value += 50
  await load()
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString()
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

function formatDuration(start: string, end: string): string {
  const ms = new Date(end).getTime() - new Date(start).getTime()
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  const mins = Math.floor(ms / 60000)
  const secs = Math.round((ms % 60000) / 1000)
  return `${mins}m ${secs}s`
}

onMounted(load)
</script>
