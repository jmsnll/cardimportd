<template>
  <nav class="navbar is-brand" role="banner" aria-label="Card Importer">
    <div class="navbar-brand">
      <span class="navbar-item has-text-weight-semibold">Card Importer</span>
      <span class="navbar-item is-sub">cardimportd</span>
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
        :hidden="activeTab !== 'cards'"
      >
        <CardsView />
      </section>

      <section
        id="panel-settings"
        role="tabpanel"
        aria-labelledby="tab-settings"
        :hidden="activeTab !== 'settings'"
      >
        <SettingsView
          v-if="config"
          :config="config"
          @update:config="onConfigUpdated"
        />
      </section>

      <section
        id="panel-notifications"
        role="tabpanel"
        aria-labelledby="tab-notifications"
        :hidden="activeTab !== 'notifications'"
      >
        <NotificationsView
          v-if="config"
          :config="config"
          @update:config="onConfigUpdated"
        />
      </section>
    </div>
  </main>

  <Toast ref="toast" />
</template>

<script setup lang="ts">
import { ref, provide, onMounted } from 'vue'
import { getConfig } from './api'
import type { Config } from './types'
import Toast from './components/Toast.vue'
import CardsView from './views/CardsView.vue'
import SettingsView from './views/SettingsView.vue'
import NotificationsView from './views/NotificationsView.vue'

type Tab = 'cards' | 'settings' | 'notifications'
const tabs: Tab[] = ['cards', 'settings', 'notifications']

const activeTab = ref<Tab>('cards')
const config = ref<Config | null>(null)
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
    showToast('Failed to load config: ' + (err as Error).message, 'error')
  }
})
</script>
