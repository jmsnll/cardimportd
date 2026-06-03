<template>
  <div
    class="fixed bottom-5 right-5 z-[9999] w-80 pointer-events-none"
    aria-live="assertive"
    aria-atomic="true"
  >
    <Transition name="toast">
      <div
        v-if="visible"
        class="pointer-events-auto rounded-lg shadow-lg px-4 py-3 text-sm font-medium text-white"
        :class="type === 'success' ? 'bg-emerald-600' : 'bg-red-600'"
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
