<template>
  <nav class="navbar is-brand" role="banner" aria-label="Card Importer">
    <div class="navbar-brand">
      <span class="navbar-item has-text-weight-semibold">Card Importer</span>
      <span class="navbar-item is-sub">cardimportd</span>
    </div>
    <div class="navbar-end" v-if="activeImport" aria-live="polite" aria-label="Import progress">
      <div class="navbar-item">
        <span class="tag is-light mr-2" v-if="activeImport.event === 'import_completed'">✓ Done</span>
        <span class="tag is-warning mr-2" v-else-if="activeImport.event === 'import_failed'">✗ Failed</span>
        <span class="tag is-info mr-2" v-else>Importing…</span>
        <span class="is-size-7 has-text-white">
          <template v-if="activeImport.owner">{{ activeImport.owner }} · </template>
          {{ activeImport.imported }}/{{ activeImport.total }} files
        </span>
      </div>
    </div>
  </nav>

  <div class="tabs is-brand mb-0" role="tablist" aria-label="Main navigation" @keydown="onTabKeydown">
    <ul>
      <li :class="{ 'is-active': activeTab === 'cards' }">
        <a
          id="tab-cards"
          role="tab"
          :aria-selected="activeTab === 'cards'"
          :tabindex="activeTab === 'cards' ? 0 : -1"
          href="#panel-cards"
          @click.prevent="activeTab = 'cards'"
        >Cards</a>
      </li>
      <li :class="{ 'is-active': activeTab === 'history' }">
        <a
          id="tab-history"
          role="tab"
          :aria-selected="activeTab === 'history'"
          :tabindex="activeTab === 'history' ? 0 : -1"
          href="#panel-history"
          @click.prevent="activeTab = 'history'"
        >History</a>
      </li>
      <li :class="{ 'is-active': activeTab === 'settings' }">
        <a
          id="tab-settings"
          role="tab"
          :aria-selected="activeTab === 'settings'"
          :tabindex="activeTab === 'settings' ? 0 : -1"
          href="#panel-settings"
          @click.prevent="activeTab = 'settings'"
        >Settings</a>
      </li>
      <li :class="{ 'is-active': activeTab === 'notifications' }">
        <a
          id="tab-notifications"
          role="tab"
          :aria-selected="activeTab === 'notifications'"
          :tabindex="activeTab === 'notifications' ? 0 : -1"
          href="#panel-notifications"
          @click.prevent="activeTab = 'notifications'"
        >Notifications</a>
      </li>
    </ul>
  </div>

  <main class="section pt-5">
    <div class="container">
      <section
        id="panel-cards"
        role="tabpanel"
        aria-labelledby="tab-cards"
        v-show="activeTab === 'cards'"
      >
        <CardsView />
      </section>

      <section
        id="panel-history"
        role="tabpanel"
        aria-labelledby="tab-history"
        v-show="activeTab === 'history'"
      >
        <HistoryView />
      </section>

      <section
        id="panel-settings"
        role="tabpanel"
        aria-labelledby="tab-settings"
        v-show="activeTab === 'settings'"
      >
        <article v-if="configError" class="message is-danger" role="alert">
          <div class="message-body">Failed to load config: {{ configError }}</div>
        </article>
        <SettingsView
          v-else-if="config"
          :config="config"
          @update:config="onConfigUpdated"
        />
      </section>

      <section
        id="panel-notifications"
        role="tabpanel"
        aria-labelledby="tab-notifications"
        v-show="activeTab === 'notifications'"
      >
        <article v-if="configError" class="message is-danger" role="alert">
          <div class="message-body">Failed to load config: {{ configError }}</div>
        </article>
        <NotificationsView
          v-else-if="config"
          :config="config"
          @update:config="onConfigUpdated"
        />
      </section>
    </div>
  </main>

  <ImportProgress :active-import="activeImport" />
  <Toast ref="toast" />
</template>

<script setup lang="ts">
import { ref, provide, onMounted } from 'vue'
import { useEventStream } from './composables/useEventStream'
import { getConfig } from './api'
import type { Config } from './types'
import Toast from './components/Toast.vue'
import ImportProgress from './components/ImportProgress.vue'
import CardsView from './views/CardsView.vue'
import HistoryView from './views/HistoryView.vue'
import SettingsView from './views/SettingsView.vue'
import NotificationsView from './views/NotificationsView.vue'

const { activeImport, mountedCards } = useEventStream()

type Tab = 'cards' | 'history' | 'settings' | 'notifications'
const tabs: Tab[] = ['cards', 'history', 'settings', 'notifications']

const activeTab = ref<Tab>('cards')
const config = ref<Config | null>(null)
const configError = ref<string | null>(null)
const toast = ref<InstanceType<typeof Toast> | null>(null)

function showToast(msg: string, type: 'success' | 'error' = 'success') {
  toast.value?.show(msg, type)
}

provide('showToast', showToast)

function onConfigUpdated(cfg: Config) {
  config.value = cfg
}

function onTabKeydown(e: KeyboardEvent) {
  const idx = tabs.indexOf(activeTab.value)
  if (e.key === 'ArrowRight') {
    activeTab.value = tabs[(idx + 1) % tabs.length]
    focusTab(activeTab.value)
  } else if (e.key === 'ArrowLeft') {
    activeTab.value = tabs[(idx - 1 + tabs.length) % tabs.length]
    focusTab(activeTab.value)
  } else if (e.key === 'Home') {
    activeTab.value = tabs[0]
    focusTab(activeTab.value)
  } else if (e.key === 'End') {
    activeTab.value = tabs[tabs.length - 1]
    focusTab(activeTab.value)
  }
}

function focusTab(tab: Tab) {
  const el = document.getElementById(`tab-${tab}`)
  el?.focus()
}

onMounted(async () => {
  try {
    config.value = await getConfig()
  } catch (err) {
    configError.value = (err as Error).message
    showToast('Failed to load config: ' + (err as Error).message, 'error')
  }
})
</script>
