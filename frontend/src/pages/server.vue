<template>
  <v-container>
    <StandardCard title="Server Settings">
      <v-progress-linear v-if="loading" indeterminate class="mb-4" />
      <v-alert v-if="error" type="error" density="compact" variant="tonal" class="mb-4">
        {{ error }}
      </v-alert>
      <v-alert v-if="success" type="success" density="compact" variant="tonal" class="mb-4">
        Settings saved.
      </v-alert>

      <v-form v-if="!loading" @submit.prevent="save">
        <v-text-field
          v-model="form.address"
          label="Server address"
          variant="outlined"
          density="compact"
          hide-details="auto"
          autocomplete="off"
          class="mb-4"
        />
        <v-text-field
          v-model.number="form.port"
          label="Port"
          type="number"
          variant="outlined"
          density="compact"
          hide-details="auto"
          autocomplete="off"
          class="mb-4"
        />
        <v-text-field
          v-model="form.username"
          label="Username"
          variant="outlined"
          density="compact"
          hide-details="auto"
          autocomplete="off"
          class="mb-4"
        />
        <v-text-field
          v-model="form.password"
          label="Password"
          type="password"
          variant="outlined"
          density="compact"
          hide-details="auto"
          autocomplete="new-password"
          class="mb-4"
        />
        <v-text-field
          v-model="form.channel"
          label="Channel"
          variant="outlined"
          density="compact"
          hide-details="auto"
          autocomplete="off"
          class="mb-4"
        />
        <v-select
          v-model="form.crypto"
          :items="cryptoItems"
          label="Crypto mode"
          variant="outlined"
          density="compact"
          hide-details="auto"
          class="mb-4"
        />
        <v-select
          v-model="form.mode"
          :items="modeItems"
          label="Voice mode"
          variant="outlined"
          density="compact"
          hide-details="auto"
          class="mb-4"
        />
        <v-checkbox
          v-model="form.insecure_tls"
          label="Allow insecure TLS"
          hide-details
          density="compact"
          class="mb-4"
        />

        <div class="d-flex justify-end mt-4">
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
  address: '',
  port: 64738,
  username: '',
  password: '',
  channel: 'Root',
  crypto: 'legacy',
  mode: 'always_on',
  insecure_tls: true,
})

const cryptoItems = [
  { title: 'Legacy (OCB2)', value: 'legacy' },
  { title: 'Lite (AES-GCM)', value: 'lite' },
]

const modeItems = [
  { title: 'Always on', value: 'always_on' },
  { title: 'Push to talk', value: 'push_to_talk' },
  { title: 'Communicator', value: 'communicator' },
]

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const success = ref(false)

onMounted(async () => {
  try {
    const cfg = await api.get('/config')
    form.value = {
      address: cfg.server?.address || '',
      port: cfg.server?.port ?? 64738,
      username: cfg.server?.username || '',
      password: cfg.server?.password || '',
      channel: cfg.server?.channel || 'Root',
      crypto: cfg.server?.crypto || 'legacy',
      mode: cfg.voice?.mode || 'always_on',
      insecure_tls: cfg.server?.insecure_tls ?? true,
    }
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
      server: {
        address: form.value.address,
        port: form.value.port,
        username: form.value.username,
        password: form.value.password,
        channel: form.value.channel,
        crypto: form.value.crypto,
        insecure_tls: form.value.insecure_tls,
      },
      voice: {
        mode: form.value.mode,
      },
    })
    success.value = true
    notify('Server settings saved', 'success')
  } catch (e) {
    error.value = e.message || 'Failed to save'
    notify(error.value, 'error')
  } finally {
    saving.value = false
  }
}
</script>
