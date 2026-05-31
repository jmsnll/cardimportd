<template>
  <div class="toast-overlay" aria-live="assertive" aria-atomic="true">
    <Transition name="toast">
      <div
        v-if="visible"
        class="notification mb-0"
        :class="type === 'success' ? 'is-success' : 'is-danger'"
        role="alert"
      >{{ message }}</div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const visible = ref(false)
const message = ref('')
const type = ref<'success' | 'error'>('success')
let timer: ReturnType<typeof setTimeout> | null = null

function show(msg: string, t: 'success' | 'error' = 'success') {
  message.value = msg
  type.value = t
  visible.value = true
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => {
    visible.value = false
  }, 3000)
}

defineExpose({ show })
</script>
