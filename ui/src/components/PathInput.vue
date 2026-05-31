<template>
  <div ref="rootEl" class="field path-input-wrapper" @keydown.esc="closeDropdown">
    <label class="label">{{ label }}</label>
    <div class="field has-addons">
      <div class="control is-expanded">
        <input
          class="input"
          type="text"
          :placeholder="placeholder"
          :value="modelValue"
          @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        />
      </div>
      <div class="control">
        <button
          class="button is-light is-small"
          type="button"
          @click="openBrowser"
        >Browse</button>
      </div>
    </div>
    <p v-if="help" class="help">{{ help }}</p>

    <div v-if="open" class="dropdown is-active" style="width: 100%">
      <div class="dropdown-menu" style="width: 100%; min-width: 100%">
        <div class="dropdown-content">

          <!-- Breadcrumb / current path row -->
          <div class="dropdown-item path-breadcrumb">
            <span class="has-text-grey is-size-7">Current: </span>
            <button
              class="button is-ghost is-small path-breadcrumb-btn"
              type="button"
              :title="browsePath"
              @click="selectPath"
            >{{ browsePath || '/' }}</button>
            <button
              v-if="browsePath && browsePath !== '/'"
              class="button is-ghost is-small"
              type="button"
              title="Go to parent directory"
              @click="navigateUp"
            >&#8593; Up</button>
          </div>

          <hr class="dropdown-divider" />

          <!-- Error state -->
          <div v-if="fetchError" class="dropdown-item">
            <span class="has-text-danger is-size-7">{{ fetchError }}</span>
          </div>

          <!-- Loading state -->
          <div v-else-if="loading" class="dropdown-item">
            <span class="has-text-grey is-size-7">Loading…</span>
          </div>

          <!-- Empty state -->
          <div v-else-if="entries.length === 0" class="dropdown-item">
            <span class="has-text-grey is-size-7">No subdirectories found.</span>
          </div>

          <!-- Directory list -->
          <div
            v-else
            role="listbox"
            :aria-label="`Subdirectories of ${browsePath}`"
          >
            <a
              v-for="entry in entries"
              :key="entry"
              class="dropdown-item"
              role="option"
              href="#"
              @click.prevent="navigateInto(entry)"
            >{{ entry }}</a>
          </div>

          <hr class="dropdown-divider" />

          <!-- Select / cancel actions -->
          <div class="dropdown-item path-actions">
            <button
              class="button is-link is-small"
              type="button"
              @click="selectPath"
            >Select this directory</button>
            <button
              class="button is-light is-small"
              type="button"
              @click="closeDropdown"
            >Cancel</button>
          </div>
        </div>
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

<style scoped>
.path-input-wrapper {
  position: relative;
}

.path-input-wrapper .dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 100;
}

.path-input-wrapper .dropdown-menu {
  display: block;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.path-input-wrapper .dropdown-content {
  max-height: 280px;
  overflow-y: auto;
}

.path-breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  flex-wrap: wrap;
}

.path-breadcrumb-btn {
  font-family: "SF Mono", "Consolas", "Menlo", monospace;
  font-size: 12px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  height: auto;
  padding: 0 0.25rem;
}

.path-actions {
  display: flex;
  gap: 0.5rem;
}
</style>
