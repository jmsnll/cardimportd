<template>
  <header>
    <span class="header-title">Card Importer</span>
    <span class="header-sub">cardimportd</span>
  </header>

  <nav class="tab-bar">
    <button
      class="tab-btn"
      :class="{ active: activeTab === 'cards' }"
      @click="activeTab = 'cards'"
    >Cards</button>
    <button
      class="tab-btn"
      :class="{ active: activeTab === 'settings' }"
      @click="activeTab = 'settings'"
    >Settings</button>
    <button
      class="tab-btn"
      :class="{ active: activeTab === 'notifications' }"
      @click="activeTab = 'notifications'"
    >Notifications</button>
  </nav>

  <main>
    <div v-show="activeTab === 'cards'">
      <CardsView />
    </div>
    <div v-show="activeTab === 'settings'">
      <SettingsView
        v-if="config"
        :config="config"
        @update:config="onConfigUpdated"
      />
    </div>
    <div v-show="activeTab === 'notifications'">
      <NotificationsView
        v-if="config"
        :config="config"
        @update:config="onConfigUpdated"
      />
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

const activeTab = ref<'cards' | 'settings' | 'notifications'>('cards')
const config = ref<Config | null>(null)
const toast = ref<InstanceType<typeof Toast> | null>(null)

function showToast(msg: string, type: 'success' | 'error' = 'success') {
  toast.value?.show(msg, type)
}

provide('showToast', showToast)

function onConfigUpdated(cfg: Config) {
  config.value = cfg
}

onMounted(async () => {
  try {
    config.value = await getConfig()
  } catch (err) {
    showToast('Failed to load config: ' + (err as Error).message, 'error')
  }
})
</script>
