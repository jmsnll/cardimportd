<template>
  <div class="surface">
    <div class="panel-heading">General</div>

    <div class="form-group">
      <label for="s-import-root">Import root</label>
      <input id="s-import-root" type="text" v-model="importRoot" />
      <span class="form-hint">Base directory, e.g. /volume1/photos</span>
    </div>

    <div class="form-group">
      <label for="s-watch-paths">Watch paths (one per line)</label>
      <textarea id="s-watch-paths" rows="3" v-model="watchPathsText"></textarea>
      <span class="form-hint">Mount-point prefixes, e.g. /volumeUSB1/usbshare</span>
    </div>

    <div class="form-group">
      <label for="s-extensions">File extensions (one per line)</label>
      <textarea id="s-extensions" rows="4" v-model="extensionsText"></textarea>
      <span class="form-hint">e.g. .jpg .raf .arw .mp4 .mov</span>
    </div>

    <div class="form-group">
      <label for="s-log-path">Log path</label>
      <input id="s-log-path" type="text" v-model="logPath" />
    </div>

    <div class="form-actions">
      <button class="btn btn-primary" @click="save">Save</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, inject } from 'vue'
import { saveConfig } from '../api'
import type { Config } from '../types'

const props = defineProps<{ config: Config }>()
const emit = defineEmits<{ (e: 'update:config', cfg: Config): void }>()

const showToast = inject<(msg: string, type?: 'success' | 'error') => void>('showToast')!

const importRoot = ref(props.config.import_root ?? '')
const watchPathsText = ref((props.config.watch_paths ?? []).join('\n'))
const extensionsText = ref((props.config.file_extensions ?? []).join('\n'))
const logPath = ref(props.config.log_path ?? '')

watch(() => props.config, (cfg) => {
  importRoot.value = cfg.import_root ?? ''
  watchPathsText.value = (cfg.watch_paths ?? []).join('\n')
  extensionsText.value = (cfg.file_extensions ?? []).join('\n')
  logPath.value = cfg.log_path ?? ''
})

async function save() {
  const updated: Config = {
    ...props.config,
    import_root: importRoot.value.trim(),
    watch_paths: watchPathsText.value.split('\n').map(s => s.trim()).filter(Boolean),
    file_extensions: extensionsText.value.split('\n').map(s => s.trim()).filter(Boolean),
    log_path: logPath.value.trim() || undefined,
  }
  try {
    await saveConfig(updated)
    emit('update:config', updated)
    showToast('Settings saved')
  } catch (err) {
    showToast((err as Error).message, 'error')
  }
}
</script>
