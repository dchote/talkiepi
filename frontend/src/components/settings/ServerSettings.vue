<template>
  <div>
      <v-progress-linear v-if="loading" indeterminate class="mb-4" />
      <v-alert v-if="error" type="error" density="compact" variant="tonal" class="mb-4">
        {{ error }}
      </v-alert>
      <v-alert v-if="success" type="success" density="compact" variant="tonal" class="mb-4">
        Settings saved.
      </v-alert>

      <v-form v-if="!loading" @submit.prevent="save">
        <div class="d-flex align-start mb-4">
          <v-text-field
            v-model="form.address"
            label="Server address"
            variant="outlined"
            density="compact"
            hide-details="auto"
            autocomplete="off"
            hint="Leave empty to auto-discover via mDNS (_mumble._tcp)"
            persistent-hint
            class="flex-grow-1 mr-2"
          />
          <v-btn
            variant="tonal"
            color="primary"
            :loading="discovering"
            class="mt-1"
            @click="discover"
          >
            Discover
          </v-btn>
        </div>

        <v-list
          v-if="discovered.length"
          density="compact"
          class="mb-4 border rounded"
        >
          <v-list-subheader>Discovered servers</v-list-subheader>
          <v-list-item
            v-for="(svc, i) in discovered"
            :key="i"
            :title="svc.name || svc.host"
            :subtitle="`${preferredHost(svc)}:${svc.port}`"
            @click="applyDiscovered(svc)"
          >
            <template #prepend>
              <v-icon icon="mdi-server-network" />
            </template>
          </v-list-item>
        </v-list>

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
          hint="Defaults to talkiepi-<device id> on first run"
          persistent-hint
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

        <div class="text-subtitle-2 mb-2 mt-2">GPIO push-to-talk</div>
        <v-checkbox
          v-model="form.gpio_enabled"
          label="Enable GPIO PTT"
          hide-details
          density="compact"
          class="mb-2"
        />
        <v-text-field
          v-model="form.gpio_chip"
          label="GPIO chip"
          variant="outlined"
          density="compact"
          hide-details="auto"
          hint="e.g. gpiochip0 or /dev/gpiochip0"
          persistent-hint
          :disabled="!form.gpio_enabled"
          class="mb-4"
        />
        <v-select
          v-model.number="form.gpio_line"
          :items="gpioPinItems"
          item-title="title"
          item-value="value"
          label="GPIO pin (BCM)"
          variant="outlined"
          density="compact"
          hide-details="auto"
          :disabled="!form.gpio_enabled"
          class="mb-4"
        />
        <v-checkbox
          v-model="form.active_low"
          label="Active low (button to ground)"
          hide-details
          density="compact"
          :disabled="!form.gpio_enabled"
          class="mb-4"
        />

        <v-checkbox
          v-model="form.auto_discover"
          label="Auto-discover server via mDNS when address is empty"
          hide-details
          density="compact"
          class="mb-2"
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
  </div>
</template>

<script setup>
import { inject, onMounted, ref } from 'vue'
import api from '@/utils/api'

const notify = inject('notify', () => {})

const form = ref({
  address: '',
  port: 64738,
  username: '',
  password: '',
  channel: 'Root',
  crypto: 'legacy',
  mode: 'push_to_talk',
  insecure_tls: true,
  auto_discover: true,
  gpio_enabled: false,
  gpio_chip: 'gpiochip0',
  gpio_line: 17,
  active_low: true,
})

const cryptoItems = [
  { title: 'Legacy (OCB2)', value: 'legacy' },
  { title: 'Lite (AES-GCM)', value: 'lite' },
]

const modeItems = [
  { title: 'Push to talk', value: 'push_to_talk' },
  { title: 'Always on', value: 'always_on' },
  { title: 'Communicator', value: 'communicator' },
]

const gpioPinItems = ref([{ title: 'BCM 17 (pin 11)', value: 17 }])

const loading = ref(true)
const saving = ref(false)
const discovering = ref(false)
const discovered = ref([])
const error = ref('')
const success = ref(false)

function preferredHost(svc) {
  const addrs = svc.addresses || []
  const v4 = addrs.find((a) => a.includes('.') && !a.includes(':'))
  return v4 || svc.host || addrs[0] || ''
}

function applyDiscovered(svc) {
  form.value.address = preferredHost(svc)
  if (svc.port) form.value.port = svc.port
  notify(`Selected ${svc.name || form.value.address}`, 'success')
}

async function discover() {
  error.value = ''
  discovering.value = true
  try {
    const res = await api.get('/discovery/servers?timeout_ms=4000')
    discovered.value = res.servers || []
    if (!discovered.value.length) {
      notify('No Mumble servers found on the LAN', 'warning')
    } else if (!form.value.address) {
      applyDiscovered(discovered.value[0])
    }
  } catch (e) {
    error.value = e.message || 'Discovery failed'
    notify(error.value, 'error')
  } finally {
    discovering.value = false
  }
}

onMounted(async () => {
  try {
    const [cfg, pins] = await Promise.all([
      api.get('/config'),
      api.get('/gpio/pins').catch(() => ({ pins: [] })),
    ])
    if (Array.isArray(pins.pins) && pins.pins.length) {
      gpioPinItems.value = pins.pins.map((p) => ({
        title: p.label || `BCM ${p.line}`,
        value: p.line,
      }))
    }
    form.value = {
      address: cfg.server?.address || '',
      port: cfg.server?.port ?? 64738,
      username: cfg.server?.username || '',
      password: cfg.server?.password || '',
      channel: cfg.server?.channel || 'Root',
      crypto: cfg.server?.crypto || 'legacy',
      mode: cfg.voice?.mode || 'push_to_talk',
      insecure_tls: cfg.server?.insecure_tls ?? true,
      auto_discover: cfg.server?.auto_discover ?? true,
      gpio_enabled: cfg.triggers?.gpio_enabled ?? false,
      gpio_chip: cfg.triggers?.gpio_chip || 'gpiochip0',
      gpio_line: cfg.triggers?.gpio_line ?? 17,
      active_low: cfg.triggers?.active_low ?? true,
    }
    if (!form.value.address && form.value.auto_discover) {
      discover()
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
        auto_discover: form.value.auto_discover,
      },
      voice: {
        mode: form.value.mode,
      },
      triggers: {
        gpio_enabled: form.value.gpio_enabled,
        gpio_chip: form.value.gpio_chip,
        gpio_line: form.value.gpio_line,
        active_low: form.value.active_low,
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
