<template>
  <div class="box">
    <h2 class="title is-5 mb-4">Pushover</h2>

    <article class="message is-info mb-5">
      <div class="message-body">
        Pushover sends push notifications when imports complete or fail.
        Create an application at <strong>pushover.net</strong> to obtain an app token.
      </div>
    </article>

    <fieldset>
      <legend class="is-sr-only">Pushover credentials</legend>

      <div class="field">
        <label class="label" for="n-app-token">App token</label>
        <div class="control">
          <input
            id="n-app-token"
            class="input"
            type="password"
            v-model="appToken"
            placeholder="aXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
            autocomplete="off"
            spellcheck="false"
          />
        </div>
      </div>

      <div class="field">
        <label class="label" for="n-user-key">User key</label>
        <div class="control">
          <input
            id="n-user-key"
            class="input"
            type="password"
            v-model="userKey"
            placeholder="uXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
            autocomplete="off"
            spellcheck="false"
          />
        </div>
      </div>
    </fieldset>

    <div class="field is-grouped mt-5">
      <div class="control">
        <button class="button is-link" @click="save">Save</button>
      </div>
      <div class="control">
        <button class="button is-light" @click="sendTest">Send test notification</button>
      </div>
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
