<template>
  <v-container>
    <v-row>
      <v-col cols="12" md="8">
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
                  <td class="text-medium-emphasis">Participants</td>
                  <td>{{ status.participants ?? 0 }}</td>
                </tr>
                <tr>
                  <td class="text-medium-emphasis">Version</td>
                  <td>{{ status.version || updateInfo?.current || '—' }}</td>
                </tr>
              </tbody>
            </v-table>

            <div class="d-flex flex-wrap mt-4">
              <v-btn
                class="mr-2 mb-2"
                color="primary"
                variant="elevated"
                :loading="busy === 'connect'"
                :disabled="status.connected || status.connecting"
                @click="connect"
              >
                Connect
              </v-btn>
              <v-btn
                class="mr-2 mb-2"
                variant="outlined"
                :loading="busy === 'disconnect'"
                :disabled="!status.connected && !status.connecting"
                @click="disconnect"
              >
                Disconnect
              </v-btn>
              <v-btn
                class="mr-2 mb-2"
                color="error"
                variant="tonal"
                :disabled="!status.connected || status.mode === 'always_on'"
                @mousedown="ptt(true)"
                @mouseup="ptt(false)"
                @mouseleave="ptt(false)"
                @touchstart.prevent="ptt(true)"
                @touchend.prevent="ptt(false)"
              >
                PTT
              </v-btn>
              <v-btn
                class="mb-2"
                color="warning"
                variant="tonal"
                :loading="busy === 'comm'"
                :disabled="!status.connected || status.mode !== 'communicator'"
                @click="toggleCommunicator"
              >
                Toggle Communicator
              </v-btn>
            </div>
          </template>
        </StandardCard>
      </v-col>

      <v-col cols="12" md="4">
        <StandardCard title="Updates" class="mb-4">
          <v-progress-linear v-if="updateLoading" indeterminate class="mb-4" />
          <template v-else-if="updateInfo">
            <div class="mb-2">
              <span class="text-medium-emphasis">Current:</span>
              {{ updateInfo.current || '—' }}
            </div>
            <div class="mb-2">
              <span class="text-medium-emphasis">Latest:</span>
              {{ updateInfo.latest || '—' }}
            </div>
            <v-chip
              v-if="updateInfo.available"
              size="small"
              color="warning"
              variant="tonal"
              density="compact"
              class="mb-4"
            >
              Update available
            </v-chip>
            <div class="d-flex flex-wrap mt-4">
              <v-btn
                class="mr-2 mb-2"
                size="small"
                variant="outlined"
                :loading="updateLoading"
                @click="refreshUpdate"
              >
                Check
              </v-btn>
              <v-btn
                v-if="updateInfo.url"
                size="small"
                variant="text"
                :href="updateInfo.url"
                target="_blank"
                rel="noopener"
                class="mb-2"
              >
                Release notes
              </v-btn>
            </div>
          </template>
        </StandardCard>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup>
import { computed, inject, onMounted, onUnmounted, ref } from 'vue'
import StandardCard from '@/components/common/StandardCard.vue'
import api from '@/utils/api'

const notify = inject('notify', () => {})

const status = ref(null)
const updateInfo = ref(null)
const loading = ref(true)
const updateLoading = ref(false)
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

async function loadUpdate(refresh = false) {
  updateLoading.value = true
  try {
    updateInfo.value = await api.get(refresh ? '/update?refresh=1' : '/update')
  } catch (e) {
    notify(e.message || 'Failed to load update info', 'error')
  } finally {
    updateLoading.value = false
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

async function ptt(pressed) {
  try {
    await api.post('/ptt', { pressed })
    await loadStatus()
  } catch (e) {
    notify(e.message || 'PTT failed', 'error')
  }
}

async function toggleCommunicator() {
  busy.value = 'comm'
  try {
    const res = await api.post('/communicator/toggle')
    notify(`Communicator: ${res.state}`, 'info')
    await loadStatus()
  } catch (e) {
    notify(e.message || 'Communicator toggle failed', 'error')
  } finally {
    busy.value = ''
  }
}

function refreshUpdate() {
  return loadUpdate(true)
}

onMounted(async () => {
  await loadStatus()
  await loadUpdate(false)
  pollTimer = setInterval(loadStatus, 3000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
