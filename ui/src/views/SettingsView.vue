<template>
  <div class="box">
    <h2 class="title is-5 mb-4">General</h2>

    <fieldset>
      <legend class="is-sr-only">General settings</legend>

      <PathInput
        v-model="importRoot"
        label="Import root"
        placeholder="/volume1/photos"
        help="Base directory where imported files are written."
      />

      <div class="field">
        <label class="label">Watch paths</label>
        <p class="help mb-2">
          Mount-point prefixes the daemon monitors for USB card readers
          (e.g. <code>/volumeUSB1/usbshare</code>). One entry per USB port.
        </p>
        <TagInput v-model="watchPaths" />
      </div>

      <div class="field">
        <label class="label" for="s-extensions">File extensions</label>
        <div class="control">
          <textarea
            id="s-extensions"
            class="textarea"
            rows="4"
            v-model="extensionsText"
            placeholder=".jpg&#10;.raf&#10;.arw&#10;.mp4"
          ></textarea>
        </div>
        <p class="help">One extension per line, e.g. <code>.jpg</code> <code>.raf</code> <code>.arw</code></p>
      </div>

      <PathInput
        v-model="logPath"
        label="Log path"
        placeholder="/var/log/cardimportd.log"
      />
    </fieldset>

    <div class="field is-grouped mt-5">
      <div class="control">
        <button class="button is-link" @click="save">Save settings</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, inject } from 'vue'
import { saveConfig } from '../api'
import type { Config } from '../types'
import TagInput from '../components/TagInput.vue'
import PathInput from '../components/PathInput.vue'

const props = defineProps<{ config: Config }>()
const emit = defineEmits<{ (e: 'update:config', cfg: Config): void }>()

const showToast = inject<(msg: string, type?: 'success' | 'error') => void>('showToast')!

const importRoot = ref(props.config.import_root ?? '')
const watchPaths = ref<string[]>(props.config.watch_paths ?? [])
const extensionsText = ref((props.config.file_extensions ?? []).join('\n'))
const logPath = ref(props.config.log_path ?? '')

watch(() => props.config, (cfg) => {
  importRoot.value = cfg.import_root ?? ''
  watchPaths.value = cfg.watch_paths ?? []
  extensionsText.value = (cfg.file_extensions ?? []).join('\n')
  logPath.value = cfg.log_path ?? ''
})

async function save() {
  const updated: Config = {
    ...props.config,
    import_root: importRoot.value.trim(),
    watch_paths: watchPaths.value,
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
