<template>
  <v-container>
    <v-row>
      <v-col cols="12" md="6">
        <StandardCard title="Channels">
          <template #toolbar>
            <div class="d-flex align-center">
              <v-btn
                size="small"
                variant="outlined"
                :loading="loading"
                @click="refresh"
              >
                Refresh
              </v-btn>
            </div>
          </template>

          <v-progress-linear v-if="loading" indeterminate class="mb-4" />
          <v-alert v-if="error" type="error" density="compact" variant="tonal" class="mb-4">
            {{ error }}
          </v-alert>
          <v-alert
            v-else-if="!loading && channels.length === 0"
            type="info"
            density="compact"
            variant="tonal"
            class="mb-4"
          >
            No channels (not connected?).
          </v-alert>

          <v-list v-else density="compact">
            <v-list-item
              v-for="ch in channels"
              :key="channelId(ch)"
              :title="channelName(ch)"
              :subtitle="`ID ${channelId(ch)}`"
              rounded="lg"
              class="mb-2"
            >
              <template #append>
                <v-btn
                  size="small"
                  variant="tonal"
                  color="primary"
                  :loading="joining === channelId(ch)"
                  @click="joinChannel(ch)"
                >
                  Join
                </v-btn>
              </template>
            </v-list-item>
          </v-list>
        </StandardCard>
      </v-col>

      <v-col cols="12" md="6">
        <StandardCard title="Users">
          <v-progress-linear v-if="loading" indeterminate class="mb-4" />
          <v-alert
            v-if="!loading && users.length === 0"
            type="info"
            density="compact"
            variant="tonal"
            class="mb-4"
          >
            No users online.
          </v-alert>

          <v-table v-else density="compact">
            <thead>
              <tr>
                <th>Name</th>
                <th>Channel</th>
                <th>Flags</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in users" :key="userSession(u)">
                <td>{{ userName(u) }}</td>
                <td>{{ channelNameById(userChannelId(u)) }}</td>
                <td>
                  <v-chip
                    v-if="u.Mute || u.mute"
                    size="x-small"
                    variant="tonal"
                    density="compact"
                    class="mr-2 mb-2"
                  >
                    Mute
                  </v-chip>
                  <v-chip
                    v-if="u.Deaf || u.deaf"
                    size="x-small"
                    variant="tonal"
                    density="compact"
                    class="mr-2 mb-2"
                  >
                    Deaf
                  </v-chip>
                  <v-chip
                    v-if="u.SelfMute || u.self_mute"
                    size="x-small"
                    variant="tonal"
                    density="compact"
                    class="mr-2 mb-2"
                  >
                    Self mute
                  </v-chip>
                  <span
                    v-if="!u.Mute && !u.mute && !u.Deaf && !u.deaf && !u.SelfMute && !u.self_mute"
                    class="text-medium-emphasis"
                  >
                    —
                  </span>
                </td>
              </tr>
            </tbody>
          </v-table>
        </StandardCard>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup>
import { inject, onMounted, onUnmounted, ref } from 'vue'
import StandardCard from '@/components/common/StandardCard.vue'
import api from '@/utils/api'

const notify = inject('notify', () => {})

const channels = ref([])
const users = ref([])
const loading = ref(true)
const joining = ref(null)
const error = ref('')
let pollTimer = null

function channelId(ch) {
  return ch.ID ?? ch.id ?? 0
}

function channelName(ch) {
  return ch.Name || ch.name || `Channel ${channelId(ch)}`
}

function userSession(u) {
  return u.Session ?? u.session ?? 0
}

function userName(u) {
  return u.Name || u.name || `User ${userSession(u)}`
}

function userChannelId(u) {
  return u.ChannelID ?? u.channel_id ?? 0
}

function channelNameById(id) {
  const ch = channels.value.find((c) => channelId(c) === id)
  return ch ? channelName(ch) : String(id)
}

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    const [ch, us] = await Promise.all([
      api.get('/channels'),
      api.get('/users'),
    ])
    channels.value = Array.isArray(ch) ? ch : []
    users.value = Array.isArray(us) ? us : []
  } catch (e) {
    error.value = e.message || 'Failed to load roster'
  } finally {
    loading.value = false
  }
}

async function joinChannel(ch) {
  const id = channelId(ch)
  joining.value = id
  try {
    await api.post(`/channels/${id}/join`)
    notify(`Joined ${channelName(ch)}`, 'success')
    await refresh()
  } catch (e) {
    notify(e.message || 'Join failed', 'error')
  } finally {
    joining.value = null
  }
}

onMounted(async () => {
  await refresh()
  pollTimer = setInterval(refresh, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
