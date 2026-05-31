<template>
  <div class="tag-input">
    <div class="field is-grouped is-grouped-multiline">
      <div v-for="(tag, index) in modelValue" :key="index" class="control">
        <div class="tags has-addons">
          <span class="tag is-link">{{ tag }}</span>
          <button
            class="tag is-delete"
            type="button"
            :aria-label="`Remove ${tag}`"
            @click="removeTag(index)"
          ></button>
        </div>
      </div>
    </div>
    <div class="field has-addons">
      <div class="control is-expanded">
        <input
          ref="inputEl"
          v-model="inputValue"
          class="input is-small"
          type="text"
          aria-label="Add a new value"
          placeholder="Type and press Enter to add"
          @keydown.enter.prevent="addTag"
          @keydown.backspace="onBackspace"
        />
      </div>
      <div class="control">
        <button
          class="button is-small is-link"
          type="button"
          @click="addTag"
        >Add</button>
      </div>
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

<style scoped>
.tag-input .field.is-grouped.is-grouped-multiline {
  margin-bottom: 0.5rem;
}

.tag-input .field.is-grouped.is-grouped-multiline:empty {
  display: none;
}
</style>
