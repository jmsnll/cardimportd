<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
    role="dialog"
    aria-modal="true"
    :aria-label="`Register card ${uuid}`"
  >
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md">
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-slate-200">
        <h2 class="text-sm font-semibold text-slate-900">Register Card</h2>
        <button
          class="text-slate-400 hover:text-slate-600 text-xl leading-none"
          aria-label="close"
          @click="$emit('cancel')"
        >×</button>
      </div>

      <!-- Body -->
      <div class="px-6 py-5 space-y-4">
        <p class="text-xs text-slate-500">UUID: <code class="font-mono bg-slate-100 px-1 py-0.5 rounded">{{ uuid }}</code></p>

        <div>
          <label class="block text-sm font-medium text-slate-700 mb-1">Owner</label>
          <select
            v-if="users.length > 0"
            v-model="owner"
            aria-label="Select owner"
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 focus:outline-none focus:ring-2 focus:ring-slate-500 cursor-pointer"
          >
            <option value="">— select a person —</option>
            <option v-for="u in users" :key="u.name" :value="u.name">{{ u.name }}</option>
          </select>
          <input
            v-else
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500"
            type="text"
            placeholder="Owner name"
            v-model="owner"
          />
          <p v-if="users.length > 0" class="text-xs text-slate-500 mt-1">Add people in Settings → People to populate this list.</p>
        </div>

        <div>
          <label class="block text-sm font-medium text-slate-700 mb-1">
            Label <span class="font-normal text-slate-400">(optional)</span>
          </label>
          <input
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500"
            type="text"
            placeholder="e.g. X-T5 Main Card"
            v-model="label"
          />
        </div>
      </div>

      <!-- Footer -->
      <div class="flex gap-2 px-6 py-4 border-t border-slate-200 bg-slate-50 rounded-b-lg">
        <button
          class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700 disabled:opacity-40"
          @click="confirm"
          :disabled="!owner.trim()"
        >Activate &amp; Import</button>
        <button
          class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-amber-50 text-amber-700 border border-amber-200 hover:bg-amber-100"
          @click="confirmPending"
        >Keep Pending</button>
        <button
          class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50"
          @click="$emit('cancel')"
        >Cancel</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ uuid: string; users: Array<{ name: string }> }>()
const emit = defineEmits<{
  confirm: [payload: { owner: string; label: string; status: 'active' | 'pending' }]
  cancel: []
}>()

const owner = ref('')
const label = ref('')

function confirm() {
  if (!owner.value.trim()) return
  emit('confirm', { owner: owner.value.trim(), label: label.value.trim(), status: 'active' })
}

function confirmPending() {
  emit('confirm', { owner: owner.value.trim(), label: label.value.trim(), status: 'pending' })
}
</script>
