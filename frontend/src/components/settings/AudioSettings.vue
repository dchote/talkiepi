<template>
  <div>
      <v-progress-linear v-if="loading" indeterminate class="mb-4" />
      <v-alert v-if="error" type="error" density="compact" variant="tonal" class="mb-4">
        {{ error }}
      </v-alert>
      <v-alert v-if="success" type="success" density="compact" variant="tonal" class="mb-4">
        Settings saved.
      </v-alert>
      <v-alert
        v-if="captureError"
        type="warning"
        density="compact"
        variant="tonal"
        class="mb-4"
      >
        Capture enumeration: {{ captureError }}
      </v-alert>
      <v-alert
        v-if="playbackError"
        type="warning"
        density="compact"
        variant="tonal"
        class="mb-4"
      >
        Playback enumeration: {{ playbackError }}
      </v-alert>

      <v-form v-if="!loading" @submit.prevent="save">
        <h3 class="text-subtitle-1 font-weight-bold mb-3">Microphone</h3>
        <v-select
          v-model="form.input_backend"
          :items="backends"
          item-title="label"
          item-value="id"
          label="Input backend"
          variant="outlined"
          density="compact"
          hide-details="auto"
          class="mb-4"
        />
        <v-select
          v-model="form.input_device_id"
          :items="captureDevices"
          item-title="label"
          item-value="id"
          label="Capture device"
          variant="outlined"
          density="compact"
          hide-details="auto"
          :loading="refreshingCapture"
          :disabled="refreshingCapture"
          class="mb-4"
          @update:model-value="onCaptureDevicePicked"
        />

        <h3 class="text-subtitle-1 font-weight-bold mb-3 mt-4">Speaker</h3>
        <v-select
          v-model="form.output_backend"
          :items="backends"
          item-title="label"
          item-value="id"
          label="Output backend"
          variant="outlined"
          density="compact"
          hide-details="auto"
          class="mb-4"
        />
        <v-select
          v-model="form.output_device_id"
          :items="playbackDevices"
          item-title="label"
          item-value="id"
          label="Playback device"
          variant="outlined"
          density="compact"
          hide-details="auto"
          :loading="refreshingPlayback"
          :disabled="refreshingPlayback"
          class="mb-4"
          @update:model-value="onPlaybackDevicePicked"
        />
        <v-text-field
          v-model.number="form.volume"
          label="Speaker volume"
          type="number"
          min="0"
          max="100"
          variant="outlined"
          density="compact"
          hide-details="auto"
          autocomplete="off"
          class="mb-4"
        />
        <v-text-field
          v-model.number="form.gain_db"
          label="Mic gain (dB)"
          type="number"
          variant="outlined"
          density="compact"
          hide-details="auto"
          autocomplete="off"
          class="mb-4"
        />

        <h3 class="text-subtitle-1 font-weight-bold mb-3 mt-4">Processing</h3>
        <v-select
          v-model="form.processing_backend"
          :items="processingBackends"
          label="Processing backend"
          variant="outlined"
          density="compact"
          hide-details="auto"
          class="mb-4"
        />
        <v-checkbox
          v-model="form.echo_cancel"
          label="Echo cancellation"
          hide-details
          density="compact"
          class="mb-4"
        />
        <v-checkbox
          v-model="form.noise_suppress"
          label="Noise suppression"
          hide-details
          density="compact"
          class="mb-4"
        />
        <v-checkbox
          v-model="form.agc"
          label="Automatic gain control"
          hide-details
          density="compact"
          class="mb-4"
        />
        <v-checkbox
          v-model="form.high_pass"
          label="High-pass filter"
          hide-details
          density="compact"
          class="mb-4"
        />
        <v-checkbox
          v-model="form.dereverb"
          label="Dereverb"
          hide-details
          density="compact"
          class="mb-4"
        />

        <div class="d-flex justify-end mt-4">
          <v-btn
            class="mr-2"
            variant="outlined"
            :loading="refreshingCapture || refreshingPlayback"
            @click="refreshAllDevices"
          >
            Refresh devices
          </v-btn>
          <v-btn
            type="submit"
            color="primary"
            variant="elevated"
            :loading="saving"
          >
            Save
          </v-btn>
        </div>
      </v-form>
  </div>
</template>

<script setup>
import { inject, onMounted, ref, watch } from 'vue'
import api from '@/utils/api'

const notify = inject('notify', () => {})

const form = ref({
  input_backend: '',
  input_device_id: '',
  input_device_name: '',
  output_backend: '',
  output_device_id: '',
  output_device_name: '',
  volume: 65,
  gain_db: 0,
  processing_backend: 'speex',
  echo_cancel: true,
  noise_suppress: true,
  agc: false,
  high_pass: true,
  dereverb: false,
})

const processingBackends = [
  { title: 'SpeexDSP', value: 'speex' },
  { title: 'WebRTC', value: 'webrtc' },
  { title: 'None', value: 'none' },
]

const backends = ref([{ id: '', label: 'Auto (system default)' }])
const captureDevices = ref([{ id: '', label: 'System default' }])
const playbackDevices = ref([{ id: '', label: 'System default' }])
const captureRaw = ref([])
const playbackRaw = ref([])
const loading = ref(true)
const saving = ref(false)
const refreshingCapture = ref(false)
const refreshingPlayback = ref(false)
const error = ref('')
const captureError = ref('')
const playbackError = ref('')
const success = ref(false)
const ready = ref(false)

function deviceLabel(d) {
  const name = d.name || d.device_name || 'Unknown'
  return d.is_default ? `${name} (default)` : name
}

function normalizeDevices(list) {
  if (!Array.isArray(list)) return [{ id: '', label: 'System default' }]
  return [
    { id: '', label: 'System default' },
    ...list.map((d) => ({
      id: d.device_id || '',
      label: deviceLabel(d),
      name: d.name || d.device_name || '',
    })),
  ]
}

function ensureSelection(list, selectedId) {
  if (!selectedId) return
  if (list.some((d) => d.id === selectedId)) return
  // Keep a stale selection visible so the user can see what was configured.
  list.splice(1, 0, {
    id: selectedId,
    label: `Previously selected (${selectedId.slice(0, 12)}…)`,
  })
}

async function loadBackends() {
  try {
    const list = await api.get('/audio/backends')
    if (Array.isArray(list) && list.length) {
      backends.value = list.map((b) =>
        typeof b === 'string'
          ? { id: b, label: b || 'Auto (system default)' }
          : { id: b.id ?? '', label: b.label || b.id || 'Auto (system default)' },
      )
    }
  } catch {
    backends.value = [{ id: '', label: 'Auto (system default)' }]
  }
}

async function refreshCaptureDevices() {
  refreshingCapture.value = true
  captureError.value = ''
  try {
    const devices = await api.get('/audio/devices', {
      capture_backend: form.value.input_backend || '',
      playback_backend: form.value.output_backend || '',
    })
    captureRaw.value = devices.capture || []
    captureDevices.value = normalizeDevices(devices.capture)
    ensureSelection(captureDevices.value, form.value.input_device_id)
    if (devices.capture_error) {
      captureError.value = devices.capture_error
    }
  } catch (e) {
    captureError.value = e.message || 'Failed to list capture devices'
    captureDevices.value = [{ id: '', label: 'System default' }]
  } finally {
    refreshingCapture.value = false
  }
}

async function refreshPlaybackDevices() {
  refreshingPlayback.value = true
  playbackError.value = ''
  try {
    const devices = await api.get('/audio/devices', {
      capture_backend: form.value.input_backend || '',
      playback_backend: form.value.output_backend || '',
    })
    playbackRaw.value = devices.playback || []
    playbackDevices.value = normalizeDevices(devices.playback)
    ensureSelection(playbackDevices.value, form.value.output_device_id)
    if (devices.playback_error) {
      playbackError.value = devices.playback_error
    }
  } catch (e) {
    playbackError.value = e.message || 'Failed to list playback devices'
    playbackDevices.value = [{ id: '', label: 'System default' }]
  } finally {
    refreshingPlayback.value = false
  }
}

async function refreshAllDevices() {
  await Promise.all([refreshCaptureDevices(), refreshPlaybackDevices()])
}

function onCaptureDevicePicked(id) {
  const d = captureRaw.value.find((x) => (x.device_id || '') === id)
  form.value.input_device_name = d?.name || d?.device_name || ''
}

function onPlaybackDevicePicked(id) {
  const d = playbackRaw.value.find((x) => (x.device_id || '') === id)
  form.value.output_device_name = d?.name || d?.device_name || ''
}

watch(
  () => form.value.input_backend,
  async () => {
    if (!ready.value) return
    // Backend change invalidates prior device id for that pipeline.
    form.value.input_device_id = ''
    form.value.input_device_name = ''
    await refreshCaptureDevices()
  },
)

watch(
  () => form.value.output_backend,
  async () => {
    if (!ready.value) return
    form.value.output_device_id = ''
    form.value.output_device_name = ''
    await refreshPlaybackDevices()
  },
)

onMounted(async () => {
  try {
    const cfg = await api.get('/config')
    form.value = {
      input_backend: cfg.audio?.input?.backend || '',
      input_device_id: cfg.audio?.input?.device_id || '',
      input_device_name: cfg.audio?.input?.device_name || '',
      output_backend: cfg.audio?.output?.backend || '',
      output_device_id: cfg.audio?.output?.device_id || '',
      output_device_name: cfg.audio?.output?.device_name || '',
      volume: cfg.audio?.output?.volume ?? 65,
      gain_db: cfg.audio?.input?.gain_db ?? 0,
      processing_backend: cfg.audio?.processing?.backend || 'speex',
      echo_cancel: cfg.audio?.processing?.echo_cancel ?? true,
      noise_suppress: cfg.audio?.processing?.noise_suppress ?? true,
      agc: cfg.audio?.processing?.agc ?? false,
      high_pass: cfg.audio?.processing?.high_pass ?? true,
      dereverb: cfg.audio?.processing?.dereverb ?? false,
    }
    await loadBackends()
    await refreshAllDevices()
  } catch (e) {
    error.value = e.message || 'Failed to load config'
  } finally {
    loading.value = false
    ready.value = true
  }
})

async function save() {
  error.value = ''
  success.value = false
  saving.value = true
  try {
    await api.patch('/config', {
      audio: {
        input: {
          backend: form.value.input_backend || '',
          device_id: form.value.input_device_id || '',
          device_name: form.value.input_device_name || '',
          gain_db: form.value.gain_db,
        },
        output: {
          backend: form.value.output_backend || '',
          device_id: form.value.output_device_id || '',
          device_name: form.value.output_device_name || '',
          volume: form.value.volume,
        },
        processing: {
          backend: form.value.processing_backend,
          echo_cancel: form.value.echo_cancel,
          noise_suppress: form.value.noise_suppress,
          agc: form.value.agc,
          high_pass: form.value.high_pass,
          dereverb: form.value.dereverb,
        },
      },
    })
    success.value = true
    notify('Audio settings saved', 'success')
  } catch (e) {
    error.value = e.message || 'Failed to save'
    notify(error.value, 'error')
  } finally {
    saving.value = false
  }
}
</script>
