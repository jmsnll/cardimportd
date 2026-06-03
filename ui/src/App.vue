<template>
  <!-- Header bar -->
  <header class="bg-slate-900 text-white px-4 h-12 flex items-center justify-between shrink-0">
    <div class="flex items-center gap-3">
      <span class="text-sm font-semibold tracking-tight">Card Importer</span>
      <span class="text-slate-400 text-xs">cardimportd</span>
    </div>
    <div v-if="activeImport" class="flex items-center gap-2 text-xs" aria-live="polite" aria-label="Import progress">
      <span v-if="activeImport.event === 'import_completed'" class="rounded-full px-2 py-0.5 text-xs font-medium bg-emerald-50 text-emerald-700">✓ Done</span>
      <span v-else-if="activeImport.event === 'import_failed'" class="rounded-full px-2 py-0.5 text-xs font-medium bg-red-100 text-red-700">✗ Failed</span>
      <span v-else class="rounded-full px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-700">Importing…</span>
      <span class="text-slate-300">
        <template v-if="activeImport.owner">{{ activeImport.owner }} · </template>
        {{ activeImport.imported }}/{{ activeImport.total }} files
      </span>
    </div>
  </header>

  <!-- Tab navigation -->
  <nav class="bg-white border-b border-slate-200 px-4" role="tablist" aria-label="Main navigation" @keydown="onTabKeydown">
    <div class="flex">
      <a
        v-for="tab in tabs"
        :key="tab"
        :id="`tab-${tab}`"
        role="tab"
        :aria-selected="activeTab === tab"
        :tabindex="activeTab === tab ? 0 : -1"
        :href="`#panel-${tab}`"
        :class="[
          'px-4 py-3 text-sm font-medium border-b-2 -mb-px transition-colors capitalize',
          activeTab === tab
            ? 'border-slate-900 text-slate-900'
            : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
        ]"
        @click.prevent="activeTab = tab"
      >{{ tab === 'notifications' ? 'Notifications' : tab.charAt(0).toUpperCase() + tab.slice(1) }}</a>
    </div>
  </nav>

  <!-- Main content -->
  <main class="bg-slate-50 min-h-[calc(100vh-6rem)]">
    <div class="max-w-5xl mx-auto px-4 py-6">

      <section id="panel-dashboard" role="tabpanel" aria-labelledby="tab-dashboard" v-show="activeTab === 'dashboard'">
        <DashboardView />
      </section>

      <section id="panel-cards" role="tabpanel" aria-labelledby="tab-cards" v-show="activeTab === 'cards'">
        <CardsView />
      </section>

      <section id="panel-settings" role="tabpanel" aria-labelledby="tab-settings" v-show="activeTab === 'settings'">
        <div v-if="configError" class="rounded bg-red-50 border border-red-200 text-sm text-red-700 p-3" role="alert">
          Failed to load config: {{ configError }}
        </div>
        <SettingsView
          v-else-if="config"
          :config="config"
          @update:config="onConfigUpdated"
        />
      </section>

      <section id="panel-notifications" role="tabpanel" aria-labelledby="tab-notifications" v-show="activeTab === 'notifications'">
        <div v-if="configError" class="rounded bg-red-50 border border-red-200 text-sm text-red-700 p-3" role="alert">
          Failed to load config: {{ configError }}
        </div>
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
import SettingsView from './views/SettingsView.vue'
import NotificationsView from './views/NotificationsView.vue'
import DashboardView from './views/DashboardView.vue'

const { activeImport, mountedCards } = useEventStream()

type Tab = 'dashboard' | 'cards' | 'settings' | 'notifications'
const tabs: Tab[] = ['dashboard', 'cards', 'settings', 'notifications']

const activeTab = ref<Tab>('dashboard')
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
