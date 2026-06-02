<template>
  <Transition name="slide-up">
    <div
      v-if="activeImport && !minimized"
      class="import-progress-card notification"
      :class="cardClass"
      role="status"
      aria-live="polite"
      aria-label="Import progress"
    >
      <button class="delete" @click="minimized = true" aria-label="Minimize import progress"></button>
      <div class="is-flex is-align-items-center mb-2">
        <span class="tag mr-2" :class="tagClass">{{ statusLabel }}</span>
        <strong class="is-size-7">{{ activeImport.owner || 'Import' }}</strong>
      </div>
      <div v-if="activeImport.total > 0" class="mb-2">
        <progress
          class="progress is-small mb-1"
          :class="progressClass"
          :value="activeImport.imported"
          :max="activeImport.total"
        >{{ progressPct }}%</progress>
        <p class="is-size-7 has-text-grey">
          {{ activeImport.imported }}/{{ activeImport.total }} files
          <template v-if="activeImport.bytes_copied"> · {{ formatBytes(activeImport.bytes_copied) }}</template>
        </p>
      </div>
      <p v-if="activeImport.failed > 0" class="is-size-7 has-text-danger">
        {{ activeImport.failed }} file{{ activeImport.failed !== 1 ? 's' : '' }} failed
      </p>
      <p v-if="activeImport.error" class="is-size-7 has-text-danger">{{ activeImport.error }}</p>
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
.import-progress-card {
  position: fixed;
  bottom: 1.5rem;
  right: 1.5rem;
  width: 280px;
  z-index: 100;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

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
