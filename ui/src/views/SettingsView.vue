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

  <div class="box mt-4">
    <div class="level mb-3">
      <div class="level-left">
        <div class="level-item"><h2 class="title is-5 mb-0">People</h2></div>
      </div>
      <div class="level-right">
        <div class="level-item">
          <button class="button is-small is-link" @click="showAddUser = true" v-if="!showAddUser">Add person</button>
        </div>
      </div>
    </div>

    <div v-if="showAddUser" class="field has-addons mb-3">
      <div class="control is-expanded">
        <input class="input is-small" type="text" placeholder="Name" v-model="newUserName" @keyup.enter="addUser" />
      </div>
      <div class="control">
        <button class="button is-small is-link" @click="addUser" :disabled="!newUserName.trim()">Add</button>
      </div>
      <div class="control">
        <button class="button is-small is-light" @click="showAddUser = false; newUserName = ''">Cancel</button>
      </div>
    </div>

    <div v-if="usersLoading" class="has-text-grey is-size-7 py-3">Loading…</div>
    <div v-else-if="users.length === 0" class="has-text-grey is-size-7 py-3">
      No people registered yet. People registered here can be selected from a dropdown when assigning cards.
    </div>
    <table v-else class="table is-fullwidth is-narrow mb-0">
      <thead>
        <tr>
          <th>Name</th>
          <th>Destination template override</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.name">
          <td>
            <input v-if="editingUser === user.name" class="input is-small" v-model="editUserName" />
            <span v-else>{{ user.name }}</span>
          </td>
          <td>
            <input v-if="editingUser === user.name" class="input is-small" v-model="editUserTemplate" placeholder="{{ .Owner }}/{{ .Year }}/…" />
            <span v-else class="has-text-grey is-size-7">{{ user.destination_template || '—' }}</span>
          </td>
          <td class="is-narrow">
            <div class="buttons are-small is-right">
              <template v-if="editingUser === user.name">
                <button class="button is-success is-small" @click="saveUser(user.name)">Save</button>
                <button class="button is-light is-small" @click="editingUser = null">Cancel</button>
              </template>
              <template v-else>
                <button class="button is-light is-small" @click="startEditUser(user)">Edit</button>
                <button class="button is-danger is-light is-small" @click="removeUser(user.name)">Remove</button>
              </template>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>

  <div class="box mt-4">
    <h2 class="title is-5 mb-2">Template Builder</h2>
    <p class="is-size-7 has-text-grey mb-3">
      Build a destination template by clicking variable chips below. The template controls where imported files are placed under the import root.
    </p>

    <div class="field">
      <label class="label is-small">Template</label>
      <div class="field has-addons">
        <div class="control is-expanded">
          <input
            id="tb-template"
            class="input is-small"
            type="text"
            placeholder="{{ .Owner }}/{{ .Year }}/{{ .Month }}/{{ .Day }}"
            v-model="tbTemplate"
          />
        </div>
        <div class="control">
          <button class="button is-small is-light" @click="tbTemplate = ''" :disabled="!tbTemplate">Clear</button>
        </div>
      </div>
    </div>

    <div class="field">
      <label class="label is-small">Variables</label>
      <div class="tags">
        <span
          v-for="v in templateVars"
          :key="v.token"
          class="tag is-link is-light is-clickable"
          @click="insertTemplateVar(v.token)"
          :title="v.description"
        >{{ v.token }}</span>
      </div>
      <p class="help">Click a variable to append it to the template.</p>
    </div>

    <div class="field" v-if="tbTemplate">
      <label class="label is-small">Preview</label>
      <div class="notification is-light py-2 px-3 mb-0 is-size-7 has-text-weight-semibold" style="font-family: monospace">
        {{ tbPreview }}
      </div>
      <p class="help">Example render using placeholder values.</p>
    </div>

    <div class="field is-grouped mt-4">
      <div class="control">
        <button class="button is-small is-link" @click="applyTemplate" :disabled="!tbTemplate">
          Apply to global template
        </button>
      </div>
    </div>
  </div>

  <div class="box mt-4">
    <h2 class="title is-5 mb-2">Watcher status</h2>
    <div v-if="watchPaths.length === 0" class="has-text-grey is-size-7">No watch paths configured.</div>
    <div v-else>
      <p class="is-size-7 has-text-grey mb-3">
        The daemon polls these paths for USB card readers. Changes require a config save and daemon restart.
      </p>
      <table class="table is-narrow is-fullwidth is-size-7">
        <thead><tr><th>Watch path</th><th>Status</th></tr></thead>
        <tbody>
          <tr v-for="path in watchPaths" :key="path">
            <td style="font-family: monospace">{{ path }}</td>
            <td>
              <span
                class="tag is-size-7"
                :class="mountedPaths.has(path) ? 'is-success is-light' : 'is-light'"
              >{{ mountedPaths.has(path) ? 'card detected' : 'watching' }}</span>
            </td>
          </tr>
        </tbody>
      </table>
      <p class="help">A "card detected" path has an active mount under it right now.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, inject, onMounted, computed } from 'vue'
import { saveConfig, getUsers, createUser, updateUser, deleteUser, getStatus } from '../api'
import type { Config, User } from '../types'
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

const users = ref<User[]>([])
const usersLoading = ref(false)
const showAddUser = ref(false)
const newUserName = ref('')
const editingUser = ref<string | null>(null)
const editUserName = ref('')
const editUserTemplate = ref('')

async function loadUsers() {
  usersLoading.value = true
  try { users.value = await getUsers() } catch { /* non-fatal */ }
  finally { usersLoading.value = false }
}

async function addUser() {
  const name = newUserName.value.trim()
  if (!name) return
  try {
    await createUser({ name })
    showToast('Person added')
    newUserName.value = ''
    showAddUser.value = false
    await loadUsers()
  } catch (err) { showToast((err as Error).message, 'error') }
}

function startEditUser(user: User) {
  editingUser.value = user.name
  editUserName.value = user.name
  editUserTemplate.value = user.destination_template ?? ''
}

async function saveUser(oldName: string) {
  try {
    await updateUser(oldName, { name: editUserName.value.trim() || oldName, destination_template: editUserTemplate.value.trim() || undefined })
    showToast('Person updated')
    editingUser.value = null
    await loadUsers()
  } catch (err) { showToast((err as Error).message, 'error') }
}

async function removeUser(name: string) {
  try {
    await deleteUser(name)
    showToast('Person removed')
    await loadUsers()
  } catch (err) {
    const msg = (err as Error).message
    showToast(msg.includes('referenced') ? msg + ' — reassign cards first' : msg, 'error')
  }
}

// Template builder
const tbTemplate = ref('')
const templateVars = [
  { token: '{{ .Owner }}', description: 'Card owner name' },
  { token: '{{ .Year }}', description: 'Year (4-digit)' },
  { token: '{{ .Month }}', description: 'Month (2-digit)' },
  { token: '{{ .Day }}', description: 'Day (2-digit)' },
  { token: '{{ .CameraModel }}', description: 'Camera model from EXIF' },
  { token: '{{ .CardUUID }}', description: 'Card filesystem UUID' },
]

function insertTemplateVar(token: string) {
  tbTemplate.value = tbTemplate.value
    ? tbTemplate.value + '/' + token
    : token
}

const tbPreview = computed(() => {
  if (!tbTemplate.value) return ''
  return tbTemplate.value
    .replace(/\{\{\s*\.Owner\s*\}\}/g, 'Alice')
    .replace(/\{\{\s*\.Year\s*\}\}/g, '2024')
    .replace(/\{\{\s*\.Month\s*\}\}/g, '06')
    .replace(/\{\{\s*\.Day\s*\}\}/g, '15')
    .replace(/\{\{\s*\.CameraModel\s*\}\}/g, 'Sony-A7IV')
    .replace(/\{\{\s*\.CardUUID\s*\}\}/g, 'abc12345')
})

function applyTemplate() {
  destinationTemplate.value = tbTemplate.value
  showToast('Template applied — remember to save settings')
}

// Watcher status
const mountedPaths = ref<Set<string>>(new Set())

async function loadWatcherStatus() {
  try {
    const status = await getStatus()
    const paths = new Set<string>()
    for (const card of status.mounted_cards) {
      for (const wp of watchPaths.value) {
        if (card.mount_point.startsWith(wp)) {
          paths.add(wp)
        }
      }
    }
    mountedPaths.value = paths
  } catch { /* non-fatal */ }
}

onMounted(() => {
  loadUsers()
  loadWatcherStatus()
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
