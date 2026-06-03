<template>
  <Transition name="slide-up">
    <div
      v-if="activeImport && !minimized"
      class="fixed bottom-5 right-5 z-50 w-72 bg-white border border-slate-200 rounded-lg shadow-xl p-4"
      role="status"
      aria-live="polite"
      aria-label="Import progress"
    >
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-2">
          <span
            class="rounded-full px-2 py-0.5 text-xs font-medium"
            :class="{
              'bg-emerald-50 text-emerald-700': isCompleted,
              'bg-red-50 text-red-700': isFailed,
              'bg-blue-50 text-blue-700': isActive,
            }"
          >{{ statusLabel }}</span>
          <span class="text-sm font-medium text-slate-900">{{ activeImport.owner || 'Import' }}</span>
        </div>
        <button
          class="text-slate-400 hover:text-slate-600 text-lg leading-none"
          @click="minimized = true"
          aria-label="Minimize import progress"
        >×</button>
      </div>

      <div v-if="activeImport.total > 0" class="mb-3">
        <div class="w-full bg-slate-200 rounded-full h-1.5 mb-1.5">
          <div
            class="h-1.5 rounded-full transition-all"
            :class="{
              'bg-emerald-500': isCompleted,
              'bg-red-500': isFailed,
              'bg-blue-500': isActive,
            }"
            :style="{ width: progressPct + '%' }"
          ></div>
        </div>
        <p class="text-xs text-slate-500">
          {{ activeImport.imported }}/{{ activeImport.total }} files
          <template v-if="activeImport.bytes_copied"> · {{ formatBytes(activeImport.bytes_copied) }}</template>
        </p>
      </div>

      <p v-if="activeImport.failed > 0" class="text-xs text-red-600">
        {{ activeImport.failed }} file{{ activeImport.failed !== 1 ? 's' : '' }} failed
      </p>
      <p v-if="activeImport.error" class="text-xs text-red-600">{{ activeImport.error }}</p>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { ImportProgress } from '../composables/useEventStream'

const props = defineProps<{ activeImport: ImportProgress | null }>()

const minimized = ref(false)

watch(() => props.activeImport?.card_uuid, (newUuid, oldUuid) => {
  if (newUuid && newUuid !== oldUuid) {
    minimized.value = false  // new import — re-expand
  }
})

const isCompleted = computed(() => props.activeImport?.event === 'import_completed')
const isFailed = computed(() => props.activeImport?.event === 'import_failed')
const isActive = computed(() => !isCompleted.value && !isFailed.value)

const cardClass = computed(() => ({
  'is-success is-light': isCompleted.value,
  'is-danger is-light': isFailed.value,
  'is-info is-light': isActive.value,
}))

const tagClass = computed(() => ({
  'is-success': isCompleted.value,
  'is-danger': isFailed.value,
  'is-info': isActive.value,
}))

const progressClass = computed(() => ({
  'is-success': isCompleted.value,
  'is-danger': isFailed.value,
  'is-info': isActive.value,
}))

const statusLabel = computed(() => {
  if (isCompleted.value) return '✓ Done'
  if (isFailed.value) return '✗ Failed'
  return 'Importing…'
})

const progressPct = computed(() => {
  const { imported, total } = props.activeImport ?? { imported: 0, total: 0 }
  return total > 0 ? Math.round((imported / total) * 100) : 0
})

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}
</script>

<style scoped>
.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(1rem);
  opacity: 0;
}
</style>
