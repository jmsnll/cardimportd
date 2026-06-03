<template>
  <div ref="rootEl" class="relative" @keydown.esc="closeDropdown">
    <label class="block text-sm font-medium text-slate-700 mb-1">{{ label }}</label>
    <div class="flex gap-2">
      <input
        class="flex-1 rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
        type="text"
        :placeholder="placeholder"
        :value="modelValue"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      />
      <button
        type="button"
        class="inline-flex items-center rounded px-3 py-2 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 shrink-0"
        @click="openBrowser"
      >Browse</button>
    </div>
    <p v-if="help" class="text-xs text-slate-500 mt-1">{{ help }}</p>

    <!-- Dropdown -->
    <div v-if="open" class="absolute z-50 top-full left-0 right-0 mt-1 bg-white border border-slate-200 rounded-lg shadow-lg overflow-hidden">
      <!-- Breadcrumb row -->
      <div class="flex items-center gap-1 px-3 py-2 border-b border-slate-100 bg-slate-50 flex-wrap">
        <span class="text-xs text-slate-500">Current:</span>
        <button
          type="button"
          class="font-mono text-xs text-slate-700 hover:text-slate-900 truncate max-w-xs"
          :title="browsePath"
          @click="selectPath"
        >{{ browsePath || '/' }}</button>
        <button
          v-if="browsePath && browsePath !== '/'"
          type="button"
          class="text-xs text-slate-500 hover:text-slate-700 ml-1"
          title="Go to parent directory"
          @click="navigateUp"
        >↑ Up</button>
      </div>

      <!-- Content area (max height, scrollable) -->
      <div class="max-h-60 overflow-y-auto">
        <div v-if="fetchError" class="px-3 py-2 text-xs text-red-600">{{ fetchError }}</div>
        <div v-else-if="loading" class="px-3 py-2 text-xs text-slate-500">Loading…</div>
        <div v-else-if="entries.length === 0" class="px-3 py-2 text-xs text-slate-500">No subdirectories found.</div>
        <div v-else role="listbox" :aria-label="`Subdirectories of ${browsePath}`">
          <a
            v-for="entry in entries"
            :key="entry"
            class="block px-3 py-2 text-sm text-slate-700 hover:bg-slate-50 cursor-pointer"
            role="option"
            href="#"
            @click.prevent="navigateInto(entry)"
          >{{ entry }}</a>
        </div>
      </div>

      <!-- Footer actions -->
      <div class="flex gap-2 px-3 py-2 border-t border-slate-100 bg-slate-50">
        <button
          type="button"
          class="inline-flex items-center rounded px-2.5 py-1 text-xs font-medium bg-slate-900 text-white hover:bg-slate-700"
          @click="selectPath"
        >Select this directory</button>
        <button
          type="button"
          class="inline-flex items-center rounded px-2.5 py-1 text-xs font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50"
          @click="closeDropdown"
        >Cancel</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { browseFSDir } from '../api'

const props = defineProps<{
  modelValue: string
  label: string
  placeholder?: string
  help?: string
}>()

const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

const open = ref(false)
const browsePath = ref('')
const entries = ref<string[]>([])
const loading = ref(false)
const fetchError = ref('')

async function fetchDir(path: string) {
  loading.value = true
  fetchError.value = ''
  entries.value = []
  try {
    const result = await browseFSDir(path || '/')
    browsePath.value = result.path
    entries.value = result.entries
  } catch (err) {
    fetchError.value = (err as Error).message || 'Failed to list directory.'
  } finally {
    loading.value = false
  }
}

async function openBrowser() {
  browsePath.value = props.modelValue || '/'
  open.value = true
  await fetchDir(props.modelValue || '/')
}

async function navigateInto(entry: string) {
  const base = browsePath.value.endsWith('/') ? browsePath.value : browsePath.value + '/'
  await fetchDir(base + entry)
}

async function navigateUp() {
  const parts = browsePath.value.replace(/\/$/, '').split('/')
  parts.pop()
  const parent = parts.join('/') || '/'
  await fetchDir(parent)
}

function selectPath() {
  emit('update:modelValue', browsePath.value)
  closeDropdown()
}

function closeDropdown() {
  open.value = false
  fetchError.value = ''
  entries.value = []
}

const rootEl = ref<HTMLElement | null>(null)

function handleOutsideClick(event: MouseEvent) {
  if (!open.value) return
  if (rootEl.value && !rootEl.value.contains(event.target as Node)) {
    closeDropdown()
  }
}

onMounted(() => {
  document.addEventListener('click', handleOutsideClick, true)
})

onUnmounted(() => {
  document.removeEventListener('click', handleOutsideClick, true)
})
</script>
