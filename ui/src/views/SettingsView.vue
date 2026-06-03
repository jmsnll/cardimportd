<template>
  <!-- General -->
  <div class="bg-white border border-slate-200 rounded-lg p-6 mb-4">
    <h2 class="text-sm font-semibold text-slate-900 mb-4">General</h2>

    <div class="mb-5">
      <PathInput
        v-model="importRoot"
        label="Import root"
        placeholder="/volume1/photos"
        help="Base directory where imported files are written."
      />
    </div>

    <div class="mb-5">
      <label class="block text-sm font-medium text-slate-700 mb-1">Watch paths</label>
      <p class="text-xs text-slate-500 mb-1">
        Mount-point prefixes the daemon monitors for USB card readers
        (e.g. <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">/volumeUSB1/usbshare</code>). One entry per USB port.
      </p>
      <TagInput v-model="watchPaths" />
    </div>

    <div class="mb-5">
      <label class="block text-sm font-medium text-slate-700 mb-1" for="s-extensions">File extensions</label>
      <textarea
        id="s-extensions"
        class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500 resize-y"
        rows="4"
        v-model="extensionsText"
        placeholder=".jpg&#10;.raf&#10;.arw&#10;.mp4"
      ></textarea>
      <p class="text-xs text-slate-500 mt-1">
        One extension per line, e.g.
        <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">.jpg</code>
        <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">.raf</code>
        <code class="font-mono text-xs bg-slate-100 px-1 py-0.5 rounded text-slate-700">.arw</code>
      </p>
    </div>

    <div class="mb-5">
      <PathInput
        v-model="logPath"
        label="Log path"
        placeholder="/var/log/cardimportd.log"
      />
    </div>

    <div class="mb-5">
      <PathInput
        v-model="mirrorRoot"
        label="Mirror root"
        placeholder="/volume2/photos-mirror"
        help="Optional second directory where files are also written (mirroring)."
      />
    </div>

    <div class="mb-5">
      <label class="block text-sm font-medium text-slate-700 mb-1" for="s-min-free-gb">Minimum free space (GB)</label>
      <input
        id="s-min-free-gb"
        class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
        type="number"
        step="0.1"
        min="0"
        v-model.number="minFreeGB"
      />
      <p class="text-xs text-slate-500 mt-1">Import is skipped if free space falls below this threshold. Set to 0 to disable.</p>
    </div>

    <div class="mb-5">
      <label class="block text-sm font-medium text-slate-700 mb-1" for="s-post-import-hook">Post-import hook</label>
      <input
        id="s-post-import-hook"
        class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
        type="text"
        placeholder="/usr/local/bin/notify.sh"
        v-model="postImportHook"
      />
      <p class="text-xs text-slate-500 mt-1">Script executed after each successful import. Receives card UUID and import path as arguments.</p>
    </div>

    <div class="mb-5">
      <label class="block text-sm font-medium text-slate-700 mb-1">Write manifest</label>
      <label class="inline-flex items-center text-sm text-slate-700">
        <input type="checkbox" class="rounded border-slate-300 text-slate-900 focus:ring-slate-500 mr-2 align-middle" v-model="writeManifest" />
        Write manifest
      </label>
      <p class="text-xs text-slate-500 mt-1">Write a manifest.json file in each import directory listing copied files.</p>
    </div>

    <div class="mb-5">
      <label class="block text-sm font-medium text-slate-700 mb-1" for="s-destination-template">Destination template</label>
      <input
        id="s-destination-template"
        class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
        type="text"
        placeholder="{{ .Owner }}/{{ .Year }}/{{ .Month }}/{{ .Day }}"
        v-model="destinationTemplate"
      />
      <p class="text-xs text-slate-500 mt-1">Available variables: .Owner .Year .Month .Day .CameraModel .CardUUID</p>
    </div>

    <div class="mt-5">
      <button class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700 disabled:opacity-40" @click="save">
        Save settings
      </button>
    </div>
  </div>

  <!-- People -->
  <div class="bg-white border border-slate-200 rounded-lg p-6 mb-4">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-sm font-semibold text-slate-900">People</h2>
      <button
        v-if="!showAddUser"
        class="inline-flex items-center rounded px-2 py-1 text-xs font-medium bg-slate-900 text-white hover:bg-slate-700 disabled:opacity-40"
        @click="showAddUser = true"
      >
        Add person
      </button>
    </div>

    <div v-if="showAddUser" class="flex gap-2 mb-4">
      <input
        class="flex-1 block rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
        type="text"
        placeholder="Name"
        v-model="newUserName"
        @keyup.enter="addUser"
      />
      <button
        class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700 disabled:opacity-40"
        @click="addUser"
        :disabled="!newUserName.trim()"
      >
        Add
      </button>
      <button
        class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-40"
        @click="showAddUser = false; newUserName = ''"
      >
        Cancel
      </button>
    </div>

    <div v-if="usersLoading" class="text-xs text-slate-500 py-3">Loading…</div>
    <div v-else-if="users.length === 0" class="text-xs text-slate-500 py-3">
      No people registered yet. People registered here can be selected from a dropdown when assigning cards.
    </div>
    <table v-else class="w-full">
      <thead>
        <tr>
          <th class="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wide bg-slate-50">Name</th>
          <th class="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wide bg-slate-50">Destination template override</th>
          <th class="px-4 py-2 bg-slate-50"></th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-100">
        <tr v-for="user in users" :key="user.name">
          <td class="px-4 py-3 text-sm text-slate-700">
            <input
              v-if="editingUser === user.name"
              class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
              v-model="editUserName"
            />
            <span v-else>{{ user.name }}</span>
          </td>
          <td class="px-4 py-3 text-sm text-slate-700">
            <input
              v-if="editingUser === user.name"
              class="block w-full rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
              v-model="editUserTemplate"
              placeholder="{{ .Owner }}/{{ .Year }}/…"
            />
            <span v-else class="text-xs text-slate-500">{{ user.destination_template || '—' }}</span>
          </td>
          <td class="px-4 py-3 text-sm text-slate-700 whitespace-nowrap text-right">
            <template v-if="editingUser === user.name">
              <button class="inline-flex items-center rounded px-2 py-1 text-xs font-medium bg-slate-900 text-white hover:bg-slate-700 disabled:opacity-40 mr-1" @click="saveUser(user.name)">Save</button>
              <button class="inline-flex items-center rounded px-2 py-1 text-xs font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-40" @click="editingUser = null">Cancel</button>
            </template>
            <template v-else>
              <button class="inline-flex items-center rounded px-2 py-1 text-xs font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-40 mr-1" @click="startEditUser(user)">Edit</button>
              <button class="inline-flex items-center text-sm text-red-600 hover:text-red-700" @click="removeUser(user.name)">Remove</button>
            </template>
          </td>
        </tr>
      </tbody>
    </table>
  </div>

  <!-- Template Builder -->
  <div class="bg-white border border-slate-200 rounded-lg p-6 mb-4">
    <h2 class="text-sm font-semibold text-slate-900 mb-4">Template Builder</h2>
    <p class="text-xs text-slate-500 mb-4">
      Build a destination template by clicking variable chips below. The template controls where imported files are placed under the import root.
    </p>

    <div class="mb-5">
      <label class="block text-sm font-medium text-slate-700 mb-1" for="tb-template">Template</label>
      <div class="flex gap-2">
        <input
          id="tb-template"
          class="flex-1 block rounded border border-slate-300 text-sm px-3 py-2 text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-slate-500"
          type="text"
          placeholder="{{ .Owner }}/{{ .Year }}/{{ .Month }}/{{ .Day }}"
          v-model="tbTemplate"
        />
        <button
          class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-40"
          @click="tbTemplate = ''"
          :disabled="!tbTemplate"
        >
          Clear
        </button>
      </div>
    </div>

    <div class="mb-5">
      <label class="block text-sm font-medium text-slate-700 mb-1">Variables</label>
      <div class="flex flex-wrap gap-1 mb-1">
        <span
          v-for="v in templateVars"
          :key="v.token"
          class="inline-flex cursor-pointer rounded-full px-2.5 py-0.5 text-xs font-medium bg-slate-100 text-slate-700 hover:bg-slate-200 border border-slate-200 mr-1 mb-1"
          @click="insertTemplateVar(v.token)"
          :title="v.description"
        >{{ v.token }}</span>
      </div>
      <p class="text-xs text-slate-500 mt-1">Click a variable to append it to the template.</p>
    </div>

    <div class="mb-5" v-if="tbTemplate">
      <label class="block text-sm font-medium text-slate-700 mb-1">Preview</label>
      <div class="font-mono text-sm bg-slate-50 border border-slate-200 rounded px-3 py-2 text-slate-700">{{ tbPreview }}</div>
      <p class="text-xs text-slate-500 mt-1">Example render using placeholder values.</p>
    </div>

    <div class="mt-4">
      <button
        class="inline-flex items-center rounded px-3 py-1.5 text-sm font-medium bg-slate-900 text-white hover:bg-slate-700 disabled:opacity-40"
        @click="applyTemplate"
        :disabled="!tbTemplate"
      >
        Apply to global template
      </button>
    </div>
  </div>

  <!-- Watcher status -->
  <div class="bg-white border border-slate-200 rounded-lg p-6 mb-4">
    <h2 class="text-sm font-semibold text-slate-900 mb-4">Watcher status</h2>
    <div v-if="watchPaths.length === 0" class="text-xs text-slate-500">No watch paths configured.</div>
    <div v-else>
      <table class="w-full">
        <thead>
          <tr>
            <th class="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wide bg-slate-50">Watch path</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wide bg-slate-50">Status</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr v-for="path in watchPaths" :key="path">
            <td class="px-4 py-3 text-sm text-slate-700 font-mono">{{ path }}</td>
            <td class="px-4 py-3 text-sm text-slate-700">
              <span
                v-if="mountedPaths.has(path)"
                class="rounded-full px-2 py-0.5 text-xs font-medium bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-600/20"
              >card detected</span>
              <span
                v-else
                class="rounded-full px-2 py-0.5 text-xs font-medium bg-slate-100 text-slate-600"
              >watching</span>
            </td>
          </tr>
        </tbody>
      </table>
      <p class="text-xs text-slate-500 mt-1">A "card detected" path has an active mount under it right now.</p>
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
