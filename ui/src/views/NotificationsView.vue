<template>
  <div>
    <!-- Pushover -->
    <div class="bg-white border border-slate-200 rounded-lg p-6 mb-4">
      <h2 class="text-sm font-semibold text-slate-900 mb-1">Pushover</h2>
      <p class="text-xs text-slate-500 mb-5">Push notifications to your devices when imports complete or fail.</p>

      <div class="rounded bg-blue-50 border border-blue-200 text-sm text-blue-800 p-3 mb-5">
        Pushover sends push notifications when imports complete or fail.
        Create an application at <strong>pushover.net</strong> to obtain an app token.
      </div>

      <fieldset>
        <legend class="sr-only">Pushover credentials</legend>

        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-1" for="n-app-token">App token</label>
          <input
            id="n-app-token"
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
            type="password"
            v-model="appToken"
            placeholder="aXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
            autocomplete="off"
            spellcheck="false"
          />
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-1" for="n-user-key">User key</label>
          <input
            id="n-user-key"
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
            type="password"
            v-model="userKey"
            placeholder="uXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
            autocomplete="off"
            spellcheck="false"
          />
        </div>

        <div class="mb-0">
          <label class="block text-sm font-medium text-slate-700 mb-1">Events</label>
          <TagInput v-model="pushoverEvents" />
          <p class="text-xs text-slate-500 mt-1">Leave empty to notify on all events, or specify: <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_completed</code> <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_failed</code> <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_started</code></p>
        </div>
      </fieldset>

      <div class="flex gap-2 mt-5">
        <button class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700" @click="save">Save</button>
        <button class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50" @click="sendTest('pushover')">Send test</button>
      </div>
    </div>

    <!-- ntfy -->
    <div class="bg-white border border-slate-200 rounded-lg p-6 mb-4">
      <h2 class="text-sm font-semibold text-slate-900 mb-1">ntfy</h2>
      <p class="text-xs text-slate-500 mb-5">Open-source pub/sub notifications via ntfy.sh or a self-hosted instance.</p>

      <div class="rounded bg-blue-50 border border-blue-200 text-sm text-blue-800 p-3 mb-5">
        ntfy is an open-source pub/sub notification service. Use <strong>ntfy.sh</strong> or self-host.
        The URL must include the full topic path, e.g. <code class="font-mono text-xs bg-blue-100 px-1 py-0.5 rounded">https://ntfy.sh/mycards</code>.
      </div>

      <fieldset>
        <legend class="sr-only">ntfy settings</legend>

        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-1" for="n-ntfy-url">Topic URL</label>
          <input
            id="n-ntfy-url"
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
            type="url"
            v-model="ntfyUrl"
            placeholder="https://ntfy.sh/mycards"
          />
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-1" for="n-ntfy-token">Token (optional)</label>
          <input
            id="n-ntfy-token"
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
            type="password"
            v-model="ntfyToken"
            autocomplete="off"
            spellcheck="false"
          />
        </div>

        <div class="mb-0">
          <label class="block text-sm font-medium text-slate-700 mb-1">Events</label>
          <TagInput v-model="ntfyEvents" />
          <p class="text-xs text-slate-500 mt-1">Leave empty to notify on all events, or specify: <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_completed</code> <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_failed</code> <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_started</code></p>
        </div>
      </fieldset>

      <div class="flex gap-2 mt-5">
        <button class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700" @click="save">Save</button>
        <button class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50" @click="sendTest('ntfy')">Send test</button>
      </div>
    </div>

    <!-- Webhook -->
    <div class="bg-white border border-slate-200 rounded-lg p-6 mb-4">
      <h2 class="text-sm font-semibold text-slate-900 mb-1">Webhook</h2>
      <p class="text-xs text-slate-500 mb-5">Send a JSON POST to any URL when import events occur.</p>

      <div class="rounded bg-blue-50 border border-blue-200 text-sm text-blue-800 p-3 mb-5">
        Send a JSON POST request to any URL when import events occur.
        If a secret is provided, requests include an <code class="font-mono text-xs bg-blue-100 px-1 py-0.5 rounded">X-Cardimportd-Signature</code> header (HMAC-SHA256).
      </div>

      <fieldset>
        <legend class="sr-only">Webhook settings</legend>

        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-1" for="n-wh-url">Webhook URL</label>
          <input
            id="n-wh-url"
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
            type="url"
            v-model="webhookUrl"
            placeholder="https://example.com/hooks/cardimport"
          />
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-1" for="n-wh-secret">Secret (optional)</label>
          <input
            id="n-wh-secret"
            class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
            type="password"
            v-model="webhookSecret"
            autocomplete="off"
            spellcheck="false"
          />
        </div>

        <div class="mb-0">
          <label class="block text-sm font-medium text-slate-700 mb-1">Events</label>
          <TagInput v-model="webhookEvents" />
          <p class="text-xs text-slate-500 mt-1">Leave empty to notify on all events, or specify: <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_completed</code> <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_failed</code> <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">import_started</code></p>
        </div>
      </fieldset>

      <div class="flex gap-2 mt-5">
        <button class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700" @click="save">Save</button>
        <button class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50" @click="sendTest('webhook')">Send test</button>
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
