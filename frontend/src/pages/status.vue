<template>
  <v-container>
    <StandardCard title="Connection Status">
      <template #header>
        <span class="text-h6 header-truncate">Connection Status</span>
        <v-spacer />
        <v-chip
          size="small"
          variant="tonal"
          density="compact"
          :color="statusColor"
          class="mr-2"
        >
          {{ statusLabel }}
        </v-chip>
      </template>

      <v-progress-linear v-if="loading" indeterminate class="mb-4" />
      <v-alert v-if="error" type="error" density="compact" variant="tonal" class="mb-4">
        {{ error }}
      </v-alert>

      <template v-if="status">
        <v-table density="compact" class="mb-4">
          <tbody>
            <tr>
              <td class="text-medium-emphasis">Address</td>
              <td>{{ status.address || '—' }}</td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Username</td>
              <td>{{ status.username || '—' }}</td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Channel</td>
              <td>{{ status.channel || '—' }}</td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Ping</td>
              <td>{{ status.connected ? `${Math.round(status.ping_ms)} ms` : '—' }}</td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Transport</td>
              <td>
                <v-chip
                  v-if="status.connected"
                  size="x-small"
                  variant="tonal"
                  density="compact"
                  :color="status.transport === 'UDP' ? 'success' : 'info'"
                >
                  {{ status.transport || 'TCP' }}
                </v-chip>
                <span v-else>—</span>
              </td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Mode</td>
              <td>{{ status.mode || '—' }}</td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Transmitting</td>
              <td>
                <v-icon
                  size="small"
                  :color="status.transmitting ? 'error' : 'default'"
                  class="mr-1"
                >
                  {{ status.transmitting ? 'mdi-microphone' : 'mdi-microphone-off' }}
                </v-icon>
                {{ status.transmitting ? 'Yes' : 'No' }}
              </td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Receiving</td>
              <td>
                <v-icon
                  size="small"
                  :color="status.receiving ? 'success' : 'default'"
                  class="mr-1"
                >
                  {{ status.receiving ? 'mdi-volume-high' : 'mdi-volume-off' }}
                </v-icon>
                {{ status.receiving ? 'Yes' : 'No' }}
              </td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Participants</td>
              <td>{{ status.participants ?? 0 }}</td>
            </tr>
            <tr>
              <td class="text-medium-emphasis">Version</td>
              <td>{{ status.version || '—' }}</td>
            </tr>
          </tbody>
        </v-table>

        <div class="d-flex flex-wrap align-center mt-4">
          <v-btn
            v-if="!status.connected && !status.connecting"
            class="mb-2"
            color="primary"
            variant="elevated"
            :loading="busy === 'connect'"
            @click="connect"
          >
            Connect
          </v-btn>
          <v-btn
            v-else
            class="mb-2"
            color="secondary"
            variant="outlined"
            :loading="busy === 'disconnect'"
            @click="disconnect"
          >
            Disconnect
          </v-btn>
          <v-spacer />
          <VoiceControls
            :mode="status.mode"
            :connected="!!status.connected"
            button-class="mb-2"
            @changed="loadStatus"
          />
        </div>
      </template>
    </StandardCard>
  </v-container>
</template>

<script setup>
import { computed, inject, onMounted, onUnmounted, ref } from 'vue'
import StandardCard from '@/components/common/StandardCard.vue'
import VoiceControls from '@/components/common/VoiceControls.vue'
import api from '@/utils/api'

const notify = inject('notify', () => {})

const status = ref(null)
const loading = ref(true)
const error = ref('')
const busy = ref('')
let pollTimer = null

const statusLabel = computed(() => {
  if (!status.value) return 'Unknown'
  if (status.value.connecting) return 'Connecting'
  if (status.value.connected) return 'Connected'
  return 'Disconnected'
})

const statusColor = computed(() => {
  if (!status.value) return 'default'
  if (status.value.connecting) return 'warning'
  if (status.value.connected) return 'success'
  return 'error'
})

async function loadStatus() {
  try {
    status.value = await api.get('/status')
    error.value = ''
  } catch (e) {
    error.value = e.message || 'Failed to load status'
  } finally {
    loading.value = false
  }
}

async function connect() {
  busy.value = 'connect'
  try {
    status.value = await api.post('/connection/connect')
    notify('Connected', 'success')
  } catch (e) {
    notify(e.message || 'Connect failed', 'error')
  } finally {
    busy.value = ''
  }
}

async function disconnect() {
  busy.value = 'disconnect'
  try {
    status.value = await api.post('/connection/disconnect')
    notify('Disconnected', 'info')
  } catch (e) {
    notify(e.message || 'Disconnect failed', 'error')
  } finally {
    busy.value = ''
  }
}

onMounted(async () => {
  await loadStatus()
  pollTimer = setInterval(loadStatus, 3000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
