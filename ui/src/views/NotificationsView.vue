<template>
  <div class="surface">
    <div class="panel-heading">Pushover</div>

    <div class="note">
      Pushover sends push notifications when imports complete or fail.
      Create an application at <strong>pushover.net</strong> to get an app token.
    </div>

    <div class="form-group">
      <label for="n-app-token">App token</label>
      <input
        id="n-app-token"
        type="text"
        v-model="appToken"
        placeholder="aXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
      />
    </div>

    <div class="form-group">
      <label for="n-user-key">User key</label>
      <input
        id="n-user-key"
        type="text"
        v-model="userKey"
        placeholder="uXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
      />
    </div>

    <div class="form-actions">
      <button class="btn btn-primary" @click="save">Save</button>
      <button class="btn btn-secondary" @click="sendTest">Send test</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, inject } from 'vue'
import { saveConfig, testNotification } from '../api'
import type { Config } from '../types'

const props = defineProps<{ config: Config }>()
const emit = defineEmits<{ (e: 'update:config', cfg: Config): void }>()

const showToast = inject<(msg: string, type?: 'success' | 'error') => void>('showToast')!

const appToken = ref(props.config.notifications?.pushover?.app_token ?? '')
const userKey = ref(props.config.notifications?.pushover?.user_key ?? '')

watch(() => props.config, (cfg) => {
  appToken.value = cfg.notifications?.pushover?.app_token ?? ''
  userKey.value = cfg.notifications?.pushover?.user_key ?? ''
})

async function save() {
  const updated: Config = {
    ...props.config,
    notifications: {
      ...props.config.notifications,
      pushover: {
        app_token: appToken.value.trim(),
        user_key: userKey.value.trim(),
      },
    },
  }
  try {
    await saveConfig(updated)
    emit('update:config', updated)
    showToast('Notifications saved')
  } catch (err) {
    showToast((err as Error).message, 'error')
  }
}

async function sendTest() {
  try {
    await testNotification()
    showToast('Test notification sent')
  } catch (err) {
    showToast((err as Error).message, 'error')
  }
}
</script>
