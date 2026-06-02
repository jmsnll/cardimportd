<template>
  <div class="modal is-active" role="dialog" aria-modal="true" :aria-label="`Register card ${uuid}`">
    <div class="modal-background" @click="$emit('cancel')"></div>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">Register Card</p>
        <button class="delete" aria-label="close" @click="$emit('cancel')"></button>
      </header>
      <section class="modal-card-body">
        <p class="has-text-grey is-size-7 mb-4">UUID: <code>{{ uuid }}</code></p>

        <div class="field">
          <label class="label">Owner</label>
          <div class="control">
            <div v-if="users.length > 0" class="select is-fullwidth">
              <select v-model="owner" aria-label="Select owner">
                <option value="">— select a person —</option>
                <option v-for="u in users" :key="u.name" :value="u.name">{{ u.name }}</option>
              </select>
            </div>
            <input v-else class="input" type="text" placeholder="Owner name" v-model="owner" />
          </div>
          <p v-if="users.length > 0" class="help">Add people in Settings → People to populate this list.</p>
        </div>

        <div class="field">
          <label class="label">Label <span class="has-text-grey has-text-weight-normal">(optional)</span></label>
          <div class="control">
            <input class="input" type="text" placeholder="e.g. X-T5 Main Card" v-model="label" />
          </div>
        </div>
      </section>
      <footer class="modal-card-foot">
        <div class="buttons">
          <button
            class="button is-success"
            @click="confirm"
            :disabled="!owner.trim()"
          >Activate &amp; Import</button>
          <button class="button is-warning is-light" @click="confirmPending">Keep Pending</button>
          <button class="button is-light" @click="$emit('cancel')">Cancel</button>
        </div>
      </footer>
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
