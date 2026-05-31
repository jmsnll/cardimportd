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

      <PathInput
        v-model="mirrorRoot"
        label="Mirror root"
        placeholder="/volume2/photos-mirror"
        help="Optional second directory where files are also written (mirroring)."
      />

      <div class="field">
        <label class="label" for="s-min-free-gb">Minimum free space (GB)</label>
        <div class="control">
          <input
            id="s-min-free-gb"
            class="input"
            type="number"
            step="0.1"
            min="0"
            v-model.number="minFreeGB"
          />
        </div>
        <p class="help">Import is skipped if free space falls below this threshold. Set to 0 to disable.</p>
      </div>

      <div class="field">
        <label class="label" for="s-post-import-hook">Post-import hook</label>
        <div class="control">
          <input
            id="s-post-import-hook"
            class="input"
            type="text"
            placeholder="/usr/local/bin/notify.sh"
            v-model="postImportHook"
          />
        </div>
        <p class="help">Script executed after each successful import. Receives card UUID and import path as arguments.</p>
      </div>

      <div class="field">
        <label class="label">Write manifest</label>
        <div class="control">
          <label class="checkbox"><input type="checkbox" v-model="writeManifest" /> Write manifest</label>
        </div>
        <p class="help">Write a manifest.json file in each import directory listing copied files.</p>
      </div>

      <div class="field">
        <label class="label" for="s-destination-template">Destination template</label>
        <div class="control">
          <input
            id="s-destination-template"
            class="input"
            type="text"
            placeholder="{{ .Owner }}/{{ .Year }}/{{ .Month }}/{{ .Day }}"
            v-model="destinationTemplate"
          />
        </div>
        <p class="help">Available variables: .Owner .Year .Month .Day .CameraModel .CardUUID</p>
      </div>
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
const mirrorRoot = ref(props.config.mirror_root ?? '')
const minFreeGB = ref<number>(props.config.min_free_gb ?? 0)
const postImportHook = ref(props.config.post_import_hook ?? '')
const writeManifest = ref(props.config.write_manifest ?? false)
const destinationTemplate = ref(props.config.destination_template ?? '')

watch(() => props.config, (cfg) => {
  importRoot.value = cfg.import_root ?? ''
  watchPaths.value = cfg.watch_paths ?? []
  extensionsText.value = (cfg.file_extensions ?? []).join('\n')
  logPath.value = cfg.log_path ?? ''
  mirrorRoot.value = cfg.mirror_root ?? ''
  minFreeGB.value = cfg.min_free_gb ?? 0
  postImportHook.value = cfg.post_import_hook ?? ''
  writeManifest.value = cfg.write_manifest ?? false
  destinationTemplate.value = cfg.destination_template ?? ''
})

async function save() {
  const updated: Config = {
    ...props.config,
    import_root: importRoot.value.trim(),
    watch_paths: watchPaths.value,
    file_extensions: extensionsText.value.split('\n').map(s => s.trim()).filter(Boolean),
    log_path: logPath.value.trim() || undefined,
    mirror_root: mirrorRoot.value.trim() || undefined,
    min_free_gb: minFreeGB.value || undefined,
    post_import_hook: postImportHook.value.trim() || undefined,
    write_manifest: writeManifest.value || undefined,
    destination_template: destinationTemplate.value.trim() || undefined,
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
