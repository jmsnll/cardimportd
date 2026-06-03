<template>
  <div>
    <div v-if="modelValue.length" class="flex flex-wrap gap-1.5 mb-2">
      <span
        v-for="(tag, index) in modelValue"
        :key="index"
        class="inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium bg-slate-100 text-slate-700 border border-slate-200"
      >
        {{ tag }}
        <button
          type="button"
          class="text-slate-400 hover:text-slate-600 leading-none"
          :aria-label="`Remove ${tag}`"
          @click="removeTag(index)"
        >×</button>
      </span>
    </div>
    <div class="flex gap-2">
      <input
        ref="inputEl"
        v-model="inputValue"
        class="flex-1 rounded border border-slate-300 text-sm px-3 py-1.5 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
        type="text"
        aria-label="Add a new value"
        placeholder="Type and press Enter to add"
        @keydown.enter.prevent="addTag"
        @keydown.backspace="onBackspace"
      />
      <button
        type="button"
        class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50"
        @click="addTag"
      >Add</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ (e: 'update:modelValue', value: string[]): void }>()

const inputValue = ref('')
const inputEl = ref<HTMLInputElement | null>(null)

function addTag() {
  const trimmed = inputValue.value.trim()
  if (!trimmed) return
  emit('update:modelValue', [...props.modelValue, trimmed])
  inputValue.value = ''
}

function removeTag(index: number) {
  const updated = [...props.modelValue]
  updated.splice(index, 1)
  emit('update:modelValue', updated)
}

function onBackspace() {
  if (inputValue.value === '' && props.modelValue.length > 0) {
    removeTag(props.modelValue.length - 1)
  }
}
</script>
