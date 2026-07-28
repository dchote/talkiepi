<template>
  <v-container>
    <StandardCard title="Audio Settings">
      <v-progress-linear v-if="loading" indeterminate class="mb-4" />
      <v-alert v-if="error" type="error" density="compact" variant="tonal" class="mb-4">
        {{ error }}
      </v-alert>
      <v-alert v-if="success" type="success" density="compact" variant="tonal" class="mb-4">
        Settings saved.
      </v-alert>

      <v-form v-if="!loading" @submit.prevent="save">
        <h3 class="text-subtitle-1 font-weight-bold mb-3">Devices</h3>
        <v-select
          v-model="form.input_backend"
          :items="backends"
          label="Input backend"
          variant="outlined"
          density="compact"
          hide-details="auto"
          clearable
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
          clearable
          class="mb-4"
        />
        <v-select
          v-model="form.output_backend"
          :items="backends"
          label="Output backend"
          variant="outlined"
          density="compact"
          hide-details="auto"
          clearable
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
          clearable
          class="mb-4"
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
            :loading="refreshing"
            @click="refreshDevices"
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
    </StandardCard>
  </v-container>
</template>

<script setup>
import { inject, onMounted, ref } from 'vue'
import StandardCard from '@/components/common/StandardCard.vue'
import api from '@/utils/api'

const notify = inject('notify', () => {})

const form = ref({
  input_backend: '',
  input_device_id: '',
  output_backend: '',
  output_device_id: '',
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

const backends = ref(['default'])
const captureDevices = ref([{ id: '', label: 'System default' }])
const playbackDevices = ref([{ id: '', label: 'System default' }])
const loading = ref(true)
const saving = ref(false)
const refreshing = ref(false)
const error = ref('')
const success = ref(false)

function deviceLabel(d) {
  const name = d.name || d.Name || d.device_name || d.id || d.ID || 'Unknown'
  const id = d.id || d.ID || d.device_id || ''
  return id ? `${name} (${id})` : name
}

function normalizeDevices(list) {
  if (!Array.isArray(list)) return [{ id: '', label: 'System default' }]
  return [
    { id: '', label: 'System default' },
    ...list.map((d) => ({
      id: d.id || d.ID || d.device_id || '',
      label: deviceLabel(d),
    })),
  ]
}

async function refreshDevices() {
  refreshing.value = true
  try {
    const [backendList, devices] = await Promise.all([
      api.get('/audio/backends').catch(() => ['default']),
      api.get('/audio/devices').catch(() => ({ capture: [], playback: [] })),
    ])
    backends.value = Array.isArray(backendList) && backendList.length
      ? backendList
      : ['default']
    captureDevices.value = normalizeDevices(devices.capture)
    playbackDevices.value = normalizeDevices(devices.playback)
  } catch (e) {
    notify(e.message || 'Failed to refresh devices', 'error')
  } finally {
    refreshing.value = false
  }
}

onMounted(async () => {
  try {
    const cfg = await api.get('/config')
    form.value = {
      input_backend: cfg.audio?.input?.backend || '',
      input_device_id: cfg.audio?.input?.device_id || '',
      output_backend: cfg.audio?.output?.backend || '',
      output_device_id: cfg.audio?.output?.device_id || '',
      volume: cfg.audio?.output?.volume ?? 65,
      gain_db: cfg.audio?.input?.gain_db ?? 0,
      processing_backend: cfg.audio?.processing?.backend || 'speex',
      echo_cancel: cfg.audio?.processing?.echo_cancel ?? true,
      noise_suppress: cfg.audio?.processing?.noise_suppress ?? true,
      agc: cfg.audio?.processing?.agc ?? false,
      high_pass: cfg.audio?.processing?.high_pass ?? true,
      dereverb: cfg.audio?.processing?.dereverb ?? false,
    }
    await refreshDevices()
  } catch (e) {
    error.value = e.message || 'Failed to load config'
  } finally {
    loading.value = false
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
          gain_db: form.value.gain_db,
        },
        output: {
          backend: form.value.output_backend || '',
          device_id: form.value.output_device_id || '',
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
