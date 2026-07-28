<template>
  <div class="channels-page">
    <div v-if="!connected" class="channels-disconnected pa-6">
      <v-alert type="info" variant="tonal" density="compact" class="mb-4">
        Connect to a Mumble server to browse channels and chat.
      </v-alert>
      <v-btn color="primary" variant="elevated" to="/status">
        Go to Status
      </v-btn>
    </div>

    <div v-else class="channels-shell">
      <!-- Channel tree -->
      <aside class="channel-sidebar">
        <div class="channel-sidebar-header d-flex align-center px-3">
          <span class="text-caption font-weight-bold text-medium-emphasis text-uppercase">
            Channels
          </span>
          <v-spacer />
          <v-btn
            icon
            size="x-small"
            variant="text"
            :loading="loading"
            title="Refresh"
            @click="refresh"
          >
            <v-icon size="small">mdi-refresh</v-icon>
          </v-btn>
        </div>
        <v-divider />
        <v-progress-linear v-if="loading && !channelTree.length" indeterminate />
        <v-alert
          v-if="error"
          type="error"
          density="compact"
          variant="tonal"
          class="ma-2"
        >
          {{ error }}
        </v-alert>
        <div class="channel-tree">
          <ChannelTreeNode
            v-for="node in channelTree"
            :key="node.id"
            :node="node"
            :depth="0"
            :selected-id="selectedChannelId"
            :current-id="currentChannelId"
            :joining="joining"
            @select="selectChannel"
          />
          <div
            v-if="!loading && channelTree.length === 0"
            class="text-medium-emphasis text-caption pa-4"
          >
            No channels yet.
          </div>
        </div>
      </aside>

      <!-- Chat pane -->
      <section class="chat-pane">
        <header class="chat-header px-4 d-flex align-center">
          <v-icon size="small" class="mr-2 text-medium-emphasis">mdi-pound</v-icon>
          <div class="min-w-0">
            <div class="text-subtitle-1 font-weight-medium text-truncate">
              {{ selectedChannelName }}
            </div>
            <div class="text-caption text-medium-emphasis text-truncate">
              {{ selectedUsers.length }} online
              <span v-if="selectedChannelDesc"> · {{ selectedChannelDesc }}</span>
            </div>
          </div>
          <v-spacer />
          <VoiceControls
            class="mr-2"
            size="small"
            button-class="mr-2"
            :mode="status?.mode || ''"
            :connected="connected"
            @changed="refresh"
          />
          <v-menu v-if="selectedUsers.length" location="bottom end">
            <template #activator="{ props }">
              <v-btn v-bind="props" size="small" variant="text" prepend-icon="mdi-account-multiple">
                Members
              </v-btn>
            </template>
            <v-list density="compact" min-width="200">
              <v-list-item
                v-for="u in selectedUsers"
                :key="userSession(u)"
                :title="userName(u)"
              >
                <template #prepend>
                  <v-icon
                    size="small"
                    :color="userTalkingColor(u)"
                    class="mr-2"
                  >
                    {{ userMicIcon(u) }}
                  </v-icon>
                </template>
              </v-list-item>
            </v-list>
          </v-menu>
        </header>
        <v-divider />

        <div ref="messageListEl" class="chat-messages">
          <div
            v-if="visibleMessages.length === 0"
            class="chat-empty text-medium-emphasis text-center"
          >
            No messages in this channel yet. Say hello.
          </div>
          <div
            v-for="msg in visibleMessages"
            :key="msg.id"
            class="chat-message"
          >
            <div class="chat-message-meta">
              <span class="font-weight-medium">{{ msg.author }}</span>
              <span class="text-caption text-medium-emphasis ml-2">{{ formatTime(msg.at) }}</span>
              <v-chip
                v-if="msg.direct"
                size="x-small"
                variant="tonal"
                density="compact"
                class="ml-2"
              >
                Direct
              </v-chip>
            </div>
            <div class="chat-message-body" v-text="msg.text" />
          </div>
        </div>

        <footer class="chat-composer pa-3">
          <v-textarea
            v-model="draft"
            :placeholder="`Message #${selectedChannelName}`"
            variant="outlined"
            density="compact"
            hide-details
            rows="1"
            max-rows="6"
            auto-grow
            :disabled="selectedChannelId == null"
            @keydown.enter.exact.prevent="send"
          />
          <div class="d-flex justify-end mt-2">
            <v-btn
              color="primary"
              variant="elevated"
              size="small"
              :loading="sending"
              :disabled="!draft.trim()"
              @click="send"
            >
              Send
            </v-btn>
          </div>
        </footer>
      </section>
    </div>
  </div>
</template>

<script setup>
import {
  computed,
  inject,
  nextTick,
  onMounted,
  onUnmounted,
  ref,
  watch,
} from 'vue'
import api from '@/utils/api'
import { getIngressBase } from '@/utils/ingress'
import ChannelTreeNode from '@/components/channels/ChannelTreeNode.vue'
import VoiceControls from '@/components/common/VoiceControls.vue'

const notify = inject('notify', () => {})

const channels = ref([])
const users = ref([])
const status = ref(null)
const loading = ref(true)
const joining = ref(null)
const error = ref('')
const selectedChannelId = ref(null)
const draft = ref('')
const sending = ref(false)
const messages = ref([])
const messageListEl = ref(null)
let pollTimer = null
let es = null
let msgSeq = 0

const connected = computed(() => !!status.value?.connected)

const currentChannelId = computed(() => {
  const name = status.value?.channel
  if (!name) return null
  const ch = channels.value.find((c) => channelName(c) === name)
  return ch ? channelId(ch) : null
})

const channelTree = computed(() => buildChannelTree(channels.value, users.value))

const selectedChannel = computed(() =>
  channels.value.find((c) => channelId(c) === selectedChannelId.value) || null,
)

const selectedChannelName = computed(() =>
  selectedChannel.value ? channelName(selectedChannel.value) : 'channel',
)

const selectedChannelDesc = computed(() => {
  const d = selectedChannel.value?.Description || selectedChannel.value?.description || ''
  return stripTags(d).slice(0, 80)
})

const selectedUsers = computed(() =>
  users.value.filter((u) => userChannelId(u) === selectedChannelId.value),
)

const visibleMessages = computed(() => {
  const chId = selectedChannelId.value
  const mySession = status.value?.session
  return messages.value.filter((m) => {
    if (m.direct) {
      if (chId !== currentChannelId.value) return false
      return m.sessions?.includes(mySession) || m.actor === mySession || m.local
    }
    if ((m.channelIds || []).length || (m.treeIds || []).length) {
      return (m.channelIds || []).includes(chId) || (m.treeIds || []).includes(chId)
    }
    return m.channelId === chId
  })
})

function channelId(ch) {
  return ch.id ?? ch.ID ?? 0
}

function channelName(ch) {
  return ch.name || ch.Name || `Channel ${channelId(ch)}`
}

function userSession(u) {
  return u.session ?? u.Session ?? 0
}

function userName(u) {
  return u.name || u.Name || `User ${userSession(u)}`
}

function userChannelId(u) {
  return u.channel_id ?? u.ChannelID ?? 0
}

function userMicIcon(u) {
  if (u.deaf || u.Deaf || u.self_deaf || u.SelfDeaf) return 'mdi-headset-off'
  if (u.mute || u.Mute || u.self_mute || u.SelfMute) return 'mdi-microphone-off'
  return 'mdi-microphone'
}

function userTalkingColor(u) {
  if (u.deaf || u.Deaf || u.self_deaf || u.SelfDeaf || u.mute || u.Mute || u.self_mute || u.SelfMute) {
    return 'warning'
  }
  return 'success'
}

function stripTags(html) {
  return String(html || '')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<[^>]+>/g, '')
    .replace(/&nbsp;/g, ' ')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&amp;/g, '&')
    .trim()
}

function formatTime(ts) {
  try {
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ''
  }
}

function buildChannelTree(flat, userList) {
  const nodes = new Map()
  for (const ch of flat) {
    const id = channelId(ch)
    nodes.set(id, {
      id,
      name: channelName(ch),
      position: ch.position ?? ch.Position ?? 0,
      description: ch.description || ch.Description || '',
      children: [],
      users: [],
    })
  }
  const roots = []
  for (const ch of flat) {
    const id = channelId(ch)
    const node = nodes.get(id)
    const hasParent = ch.has_parent ?? ch.HasParent ?? false
    const parentId = ch.parent_id ?? ch.ParentID ?? 0
    if (hasParent && nodes.has(parentId) && parentId !== id) {
      nodes.get(parentId).children.push(node)
    } else {
      roots.push(node)
    }
  }
  const sortNodes = (list) => {
    list.sort((a, b) => a.position - b.position || a.name.localeCompare(b.name))
    for (const n of list) sortNodes(n.children)
  }
  sortNodes(roots)

  for (const u of userList) {
    const node = nodes.get(userChannelId(u))
    if (node) node.users.push(u)
  }
  for (const node of nodes.values()) {
    node.users.sort((a, b) => userName(a).localeCompare(userName(b)))
  }
  return roots
}

function authorName(actor) {
  if (actor === status.value?.session) return status.value?.username || 'You'
  const u = users.value.find((x) => userSession(x) === actor)
  return u ? userName(u) : (actor ? `User ${actor}` : 'Server')
}

function pushMessage(partial) {
  msgSeq += 1
  messages.value.push({
    id: partial.id || `${Date.now()}-${msgSeq}`,
    at: partial.at || Date.now(),
    ...partial,
  })
  nextTick(scrollMessages)
}

function scrollMessages() {
  const el = messageListEl.value
  if (el) el.scrollTop = el.scrollHeight
}

async function loadChatHistory(channelIdToLoad) {
  if (channelIdToLoad == null) return
  try {
    const res = await api.get('/chat/messages', {
      channel_id: channelIdToLoad,
      limit: 200,
    })
    const hist = Array.isArray(res.messages) ? res.messages : []
    const mapped = hist.map((m) => ({
      id: m.id || `${m.at}-${m.actor}`,
      at: m.at ? Date.parse(m.at) || Date.now() : Date.now(),
      author: m.author || authorName(m.actor),
      text: stripTags(m.message),
      actor: m.actor,
      channelId: (m.channel_ids && m.channel_ids[0]) ?? channelIdToLoad,
      channelIds: m.channel_ids || [],
      treeIds: m.tree_ids || [],
      sessions: m.sessions || [],
      direct: !!(m.sessions?.length && !m.channel_ids?.length),
      local: !!m.outbound,
    }))
    // Merge history with any live messages not already present.
    const seen = new Set(mapped.map((m) => m.id))
    for (const m of messages.value) {
      if (!seen.has(m.id)) mapped.push(m)
    }
    mapped.sort((a, b) => a.at - b.at)
    messages.value = mapped
    nextTick(scrollMessages)
  } catch {
    /* history optional */
  }
}

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    const [st, ch, us] = await Promise.all([
      api.get('/status'),
      api.get('/channels'),
      api.get('/users'),
    ])
    status.value = st
    channels.value = Array.isArray(ch) ? ch : []
    users.value = Array.isArray(us) ? us : []

    if (!st.connected) {
      selectedChannelId.value = null
      return
    }

    if (selectedChannelId.value == null) {
      const cur = currentChannelId.value
      selectedChannelId.value = cur != null ? cur : (channels.value[0] ? channelId(channels.value[0]) : 0)
    }
    await loadChatHistory(selectedChannelId.value)
  } catch (e) {
    error.value = e.message || 'Failed to load roster'
  } finally {
    loading.value = false
  }
}

async function selectChannel(id) {
  if (joining.value != null) return
  selectedChannelId.value = id
  await loadChatHistory(id)
  if (currentChannelId.value === id) return
  joining.value = id
  try {
    await api.post(`/channels/${id}/join`)
    await refresh()
    notify(`Joined ${selectedChannelName.value}`, 'success')
  } catch (e) {
    notify(e.message || 'Join failed', 'error')
  } finally {
    joining.value = null
  }
}

async function send() {
  const text = draft.value.trim()
  if (!text || selectedChannelId.value == null) return
  sending.value = true
  try {
    const channelIdToSend = selectedChannelId.value
    const res = await api.post('/text', {
      message: text,
      channel_id: channelIdToSend,
    })
    const m = res.message
    if (m?.id && !messages.value.some((x) => x.id === m.id)) {
      pushMessage({
        id: m.id,
        at: m.at ? Date.parse(m.at) || Date.now() : Date.now(),
        author: m.author || status.value?.username || 'You',
        text: stripTags(m.message || text),
        actor: m.actor ?? status.value?.session,
        channelId: channelIdToSend,
        channelIds: m.channel_ids || [channelIdToSend],
        treeIds: [],
        sessions: [],
        local: true,
        direct: false,
      })
    }
    draft.value = ''
  } catch (e) {
    notify(e.message || 'Failed to send', 'error')
  } finally {
    sending.value = false
  }
}

function handleSSE(data) {
  if (!data || typeof data !== 'object') return
  if (data.connected !== undefined && data.type === undefined) {
    status.value = { ...status.value, ...data }
    return
  }
  switch (data.type) {
    case 'status':
      if (data.status) status.value = data.status
      break
    case 'connected':
    case 'disconnected':
    case 'roster':
      refresh()
      break
    case 'text': {
      const channelIds = data.channel_ids || []
      const sessions = data.sessions || []
      const treeIds = data.tree_ids || []
      const direct = sessions.length > 0 && channelIds.length === 0 && treeIds.length === 0
      if (data.id && messages.value.some((m) => m.id === data.id)) break
      pushMessage({
        id: data.id,
        at: data.at ? Date.parse(data.at) || Date.now() : Date.now(),
        author: data.author || authorName(data.actor),
        text: stripTags(data.message),
        actor: data.actor,
        channelId: channelIds[0],
        channelIds,
        treeIds,
        sessions,
        direct,
        local: !!data.outbound,
      })
      break
    }
    default:
      break
  }
}

function connectSSE() {
  const base = getIngressBase()
  const url = `${base}/api/v1/events`
  es = new EventSource(url)
  es.onmessage = (ev) => {
    try {
      handleSSE(JSON.parse(ev.data))
    } catch {
      /* ignore */
    }
  }
}

watch(visibleMessages, () => nextTick(scrollMessages))

onMounted(async () => {
  await refresh()
  if (!connected.value) {
    loading.value = false
  }
  connectSSE()
  pollTimer = setInterval(async () => {
    try {
      status.value = await api.get('/status')
      if (status.value.connected) {
        const [ch, us] = await Promise.all([api.get('/channels'), api.get('/users')])
        channels.value = Array.isArray(ch) ? ch : []
        users.value = Array.isArray(us) ? us : []
      }
    } catch {
      /* ignore */
    }
  }, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (es) es.close()
})
</script>

<style scoped>
.channels-page {
  height: calc(100vh - 64px);
  max-height: calc(100dvh - 64px);
  overflow: hidden;
}

.channels-disconnected {
  max-width: 560px;
}

.channels-shell {
  display: flex;
  height: 100%;
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.08);
}

.channel-sidebar {
  width: 280px;
  min-width: 220px;
  max-width: 340px;
  display: flex;
  flex-direction: column;
  border-right: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  background: rgba(var(--v-theme-on-surface), 0.02);
}

.channel-sidebar-header {
  min-height: 48px;
}

.channel-tree {
  flex: 1;
  overflow: auto;
  padding: 4px 0 12px;
}

.chat-pane {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: rgb(var(--v-theme-surface));
}

.chat-header {
  min-height: 56px;
}

.chat-messages {
  flex: 1;
  overflow: auto;
  padding: 16px 20px;
}

.chat-empty {
  margin-top: 20vh;
}

.chat-message {
  margin-bottom: 14px;
}

.chat-message-body {
  white-space: pre-wrap;
  word-break: break-word;
  margin-top: 2px;
}

.chat-composer {
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  background: rgba(var(--v-theme-on-surface), 0.02);
}

.min-w-0 {
  min-width: 0;
}

@media (max-width: 960px) {
  .channels-shell {
    flex-direction: column;
  }

  .channel-sidebar {
    width: 100%;
    max-width: none;
    max-height: 40%;
    border-right: none;
    border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  }
}
</style>
