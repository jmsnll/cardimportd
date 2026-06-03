<template>
  <div>
    <div v-if="loading" class="text-sm text-slate-500 py-8 text-center">Loading…</div>
    <div v-else-if="error" class="rounded bg-red-50 border border-red-200 text-sm text-red-700 p-3" role="alert">{{ error }}</div>
    <div v-else class="grid grid-cols-3 gap-5">

      <!-- Main column (2/3) -->
      <div class="col-span-2 space-y-5">

        <!-- Mounted cards -->
        <div class="bg-white border border-slate-200 rounded-lg p-5">
          <h2 class="text-sm font-semibold text-slate-900 mb-4">Currently mounted</h2>
          <p v-if="data!.mounted_cards.length === 0" class="text-sm text-slate-500 py-3">No cards detected. Insert a card reader to begin.</p>
          <div v-else>
            <div
              v-for="card in data!.mounted_cards"
              :key="card.uuid"
              class="flex items-center justify-between py-2.5 border-b border-slate-100 last:border-0"
            >
              <div>
                <p class="text-sm font-medium text-slate-900">{{ cardLabel(card.uuid) }}</p>
                <p class="text-xs text-slate-500">{{ card.mount_point }} · {{ card.fstype }}</p>
              </div>
              <div class="flex items-center gap-2">
                <span
                  v-if="cardStatus(card.uuid) === 'pending'"
                  class="rounded-full px-2 py-0.5 text-xs font-medium bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20"
                >Pending</span>
                <span
                  v-else
                  class="rounded-full px-2 py-0.5 text-xs font-medium bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-600/20"
                >Active</span>
                <button
                  v-if="cardStatus(card.uuid) === 'active'"
                  class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed"
                  :class="{ 'opacity-60 cursor-wait': importing.has(card.uuid) }"
                  :disabled="importing.has(card.uuid) || !!data!.active_import"
                  @click="runImport(card.uuid)"
                >Import now</button>
              </div>
            </div>
          </div>

          <div v-if="data!.active_import" class="bg-blue-50 border border-blue-200 rounded text-sm text-blue-800 p-3 mt-3">
            <span class="font-semibold">Importing:</span>
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

        <!-- Pending cards -->
        <div v-if="Object.keys(data!.pending_cards).length > 0" class="bg-white border border-slate-200 rounded-lg p-5">
          <h2 class="text-sm font-semibold text-slate-900 mb-4">Pending cards</h2>
          <p class="text-sm text-slate-500 mb-3">These cards need to be assigned an owner before imports will run.</p>
          <ul class="space-y-1.5">
            <li v-for="(entry, uuid) in data!.pending_cards" :key="uuid" class="flex items-center gap-2">
              <span class="rounded-full px-2 py-0.5 text-xs font-medium bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20">pending</span>
              <span class="font-mono text-xs text-slate-600">{{ uuid }}</span>
              <span v-if="entry.first_seen" class="text-xs text-slate-500">first seen {{ formatDate(entry.first_seen) }}</span>
            </li>
          </ul>
          <p class="text-sm text-slate-500 mt-3">Go to the <strong class="text-slate-700">Cards</strong> tab to activate.</p>
        </div>

      </div>

      <!-- Sidebar (1/3) -->
      <div class="space-y-5">

        <!-- Storage -->
        <div class="bg-white border border-slate-200 rounded-lg p-5">
          <h2 class="text-sm font-semibold text-slate-900 mb-4">Storage</h2>
          <div v-if="!data!.storage" class="text-sm text-slate-500">Storage stats unavailable.</div>
          <div v-else>
            <div class="w-full bg-slate-200 rounded-full h-1.5 mb-2">
              <div
                class="bg-slate-600 h-1.5 rounded-full"
                :style="{ width: (Math.round((data!.storage.total_bytes - data!.storage.available_bytes) / data!.storage.total_bytes * 100)) + '%' }"
              ></div>
            </div>
            <p class="text-xs text-slate-500">
              {{ formatGB(data!.storage.total_bytes - data!.storage.available_bytes) }} used of
              {{ formatGB(data!.storage.total_bytes) }} total
              ({{ formatGB(data!.storage.free_bytes) }} free)
            </p>
          </div>
        </div>

        <!-- Recent imports -->
        <div class="bg-white border border-slate-200 rounded-lg p-5">
          <h2 class="text-sm font-semibold text-slate-900 mb-4">Recent imports</h2>
          <p v-if="data!.recent_history.length === 0" class="text-sm text-slate-500">No imports yet.</p>
          <div v-else class="overflow-x-auto">
            <table class="w-full">
              <thead>
                <tr>
                  <th class="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wide bg-slate-50">Owner</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wide bg-slate-50">Date</th>
                  <th class="px-4 py-2 text-right text-xs font-medium text-slate-500 uppercase tracking-wide bg-slate-50">Files</th>
                  <th class="px-4 py-2 text-right text-xs font-medium text-slate-500 uppercase tracking-wide bg-slate-50">Size</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(entry, i) in data!.recent_history" :key="i" class="hover:bg-slate-50 border-t border-slate-100">
                  <td class="px-4 py-2.5 text-sm text-slate-700">{{ entry.owner }}</td>
                  <td class="px-4 py-2.5 text-xs text-slate-500">{{ formatDate(entry.completed_at) }}</td>
                  <td class="px-4 py-2.5 text-sm text-slate-700 text-right">{{ entry.imported }}</td>
                  <td class="px-4 py-2.5 text-sm text-slate-700 text-right">{{ formatBytes(entry.bytes_copied) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
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
