<template>
  <div>
    <!-- Pushover -->
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

        <div class="field">
          <label class="label">Events</label>
          <TagInput v-model="pushoverEvents" />
          <p class="help">Leave empty to notify on all events, or specify: <code>import_completed</code> <code>import_failed</code> <code>import_started</code></p>
        </div>
      </fieldset>

      <div class="field is-grouped mt-5">
        <div class="control">
          <button class="button is-link" @click="save">Save</button>
        </div>
        <div class="control">
          <button class="button is-light" @click="sendTest('pushover')">Send test</button>
        </div>
      </div>
    </div>

    <!-- ntfy -->
    <div class="box mt-5">
      <h2 class="title is-5 mb-4">ntfy</h2>

      <article class="message is-info mb-5">
        <div class="message-body">
          ntfy is an open-source pub/sub notification service. Use <strong>ntfy.sh</strong> or self-host.
          The URL must include the full topic path, e.g. <code>https://ntfy.sh/mycards</code>.
        </div>
      </article>

      <fieldset>
        <legend class="is-sr-only">ntfy settings</legend>

        <div class="field">
          <label class="label" for="n-ntfy-url">Topic URL</label>
          <div class="control">
            <input
              id="n-ntfy-url"
              class="input"
              type="url"
              v-model="ntfyUrl"
              placeholder="https://ntfy.sh/mycards"
            />
          </div>
        </div>

        <div class="field">
          <label class="label" for="n-ntfy-token">Token (optional)</label>
          <div class="control">
            <input
              id="n-ntfy-token"
              class="input"
              type="password"
              v-model="ntfyToken"
              autocomplete="off"
              spellcheck="false"
            />
          </div>
        </div>

        <div class="field">
          <label class="label">Events</label>
          <TagInput v-model="ntfyEvents" />
          <p class="help">Leave empty to notify on all events, or specify: <code>import_completed</code> <code>import_failed</code> <code>import_started</code></p>
        </div>
      </fieldset>

      <div class="field is-grouped mt-5">
        <div class="control">
          <button class="button is-link" @click="save">Save</button>
        </div>
        <div class="control">
          <button class="button is-light" @click="sendTest('ntfy')">Send test</button>
        </div>
      </div>
    </div>

    <!-- Webhook -->
    <div class="box mt-5">
      <h2 class="title is-5 mb-4">Webhook</h2>

      <article class="message is-info mb-5">
        <div class="message-body">
          Send a JSON POST request to any URL when import events occur.
          If a secret is provided, requests include an <code>X-Cardimportd-Signature</code> header (HMAC-SHA256).
        </div>
      </article>

      <fieldset>
        <legend class="is-sr-only">Webhook settings</legend>

        <div class="field">
          <label class="label" for="n-wh-url">Webhook URL</label>
          <div class="control">
            <input
              id="n-wh-url"
              class="input"
              type="url"
              v-model="webhookUrl"
              placeholder="https://example.com/hooks/cardimport"
            />
          </div>
        </div>

        <div class="field">
          <label class="label" for="n-wh-secret">Secret (optional)</label>
          <div class="control">
            <input
              id="n-wh-secret"
              class="input"
              type="password"
              v-model="webhookSecret"
              autocomplete="off"
              spellcheck="false"
            />
          </div>
        </div>

        <div class="field">
          <label class="label">Events</label>
          <TagInput v-model="webhookEvents" />
          <p class="help">Leave empty to notify on all events, or specify: <code>import_completed</code> <code>import_failed</code> <code>import_started</code></p>
        </div>
      </fieldset>

      <div class="field is-grouped mt-5">
        <div class="control">
          <button class="button is-link" @click="save">Save</button>
        </div>
        <div class="control">
          <button class="button is-light" @click="sendTest('webhook')">Send test</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, inject } from 'vue'
import { saveConfig, testNotification } from '../api'
import type { Config } from '../types'
import TagInput from '../components/TagInput.vue'

const props = defineProps<{ config: Config }>()
const emit = defineEmits<{ (e: 'update:config', cfg: Config): void }>()

const showToast = inject<(msg: string, type?: 'success' | 'error') => void>('showToast')!

const appToken = ref(props.config.notifications?.pushover?.app_token ?? '')
const userKey = ref(props.config.notifications?.pushover?.user_key ?? '')
const pushoverEvents = ref<string[]>(props.config.notifications?.pushover?.events ?? [])

const ntfyUrl = ref(props.config.notifications?.ntfy?.url ?? '')
const ntfyToken = ref(props.config.notifications?.ntfy?.token ?? '')
const ntfyEvents = ref<string[]>(props.config.notifications?.ntfy?.events ?? [])

const webhookUrl = ref(props.config.notifications?.webhook?.url ?? '')
const webhookSecret = ref(props.config.notifications?.webhook?.secret ?? '')
const webhookEvents = ref<string[]>(props.config.notifications?.webhook?.events ?? [])

watch(() => props.config, (cfg) => {
  appToken.value = cfg.notifications?.pushover?.app_token ?? ''
  userKey.value = cfg.notifications?.pushover?.user_key ?? ''
  pushoverEvents.value = cfg.notifications?.pushover?.events ?? []
  ntfyUrl.value = cfg.notifications?.ntfy?.url ?? ''
  ntfyToken.value = cfg.notifications?.ntfy?.token ?? ''
  ntfyEvents.value = cfg.notifications?.ntfy?.events ?? []
  webhookUrl.value = cfg.notifications?.webhook?.url ?? ''
  webhookSecret.value = cfg.notifications?.webhook?.secret ?? ''
  webhookEvents.value = cfg.notifications?.webhook?.events ?? []
})

async function save() {
  const updated: Config = {
    ...props.config,
    notifications: {
      pushover: appToken.value.trim() ? {
        app_token: appToken.value.trim(),
        user_key: userKey.value.trim(),
        events: pushoverEvents.value.length ? pushoverEvents.value : undefined,
      } : undefined,
      ntfy: ntfyUrl.value.trim() ? {
        url: ntfyUrl.value.trim(),
        token: ntfyToken.value.trim() || undefined,
        events: ntfyEvents.value.length ? ntfyEvents.value : undefined,
      } : undefined,
      webhook: webhookUrl.value.trim() ? {
        url: webhookUrl.value.trim(),
        secret: webhookSecret.value.trim() || undefined,
        events: webhookEvents.value.length ? webhookEvents.value : undefined,
      } : undefined,
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

async function sendTest(adapter: string) {
  try {
    await testNotification(adapter)
    showToast('Test notification sent')
  } catch (err) {
    showToast((err as Error).message, 'error')
  }
}
</script>
