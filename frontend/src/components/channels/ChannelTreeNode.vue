<template>
  <div class="tree-node">
    <button
      type="button"
      class="channel-row"
      :class="{
        'channel-row--selected': selectedId === node.id,
        'channel-row--current': currentId === node.id,
      }"
      :style="{ paddingLeft: `${10 + depth * 14}px` }"
      :disabled="joining != null"
      @click="$emit('select', node.id)"
    >
      <v-icon size="14" class="channel-hash mr-1">mdi-pound</v-icon>
      <span class="channel-name text-truncate">{{ node.name }}</span>
      <v-progress-circular
        v-if="joining === node.id"
        indeterminate
        size="12"
        width="2"
        class="ml-2"
      />
      <span v-else-if="node.users.length" class="user-count text-caption text-medium-emphasis">
        {{ node.users.length }}
      </span>
    </button>

    <div
      v-for="u in node.users"
      :key="userSession(u)"
      class="user-row text-truncate"
      :style="{ paddingLeft: `${28 + depth * 14}px` }"
      :title="userName(u)"
    >
      <v-icon size="12" class="mr-1" :color="micColor(u)">{{ micIcon(u) }}</v-icon>
      {{ userName(u) }}
    </div>

    <ChannelTreeNode
      v-for="child in node.children"
      :key="child.id"
      :node="child"
      :depth="depth + 1"
      :selected-id="selectedId"
      :current-id="currentId"
      :joining="joining"
      @select="$emit('select', $event)"
    />
  </div>
</template>

<script setup>
defineProps({
  node: { type: Object, required: true },
  depth: { type: Number, default: 0 },
  selectedId: { default: null },
  currentId: { default: null },
  joining: { default: null },
})

defineEmits(['select'])

function userSession(u) {
  return u.session ?? u.Session ?? 0
}

function userName(u) {
  return u.name || u.Name || `User ${userSession(u)}`
}

function micIcon(u) {
  if (u.deaf || u.Deaf || u.self_deaf || u.SelfDeaf) return 'mdi-headset-off'
  if (u.mute || u.Mute || u.self_mute || u.SelfMute) return 'mdi-microphone-off'
  return 'mdi-microphone-outline'
}

function micColor(u) {
  if (u.deaf || u.Deaf || u.self_deaf || u.SelfDeaf || u.mute || u.Mute || u.self_mute || u.SelfMute) {
    return 'warning'
  }
  return 'medium-emphasis'
}
</script>

<style scoped>
.channel-row {
  display: flex;
  align-items: center;
  width: 100%;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  padding: 6px 12px 6px 10px;
  cursor: pointer;
  border-radius: 6px;
  margin: 1px 6px;
  width: calc(100% - 12px);
}

.channel-row:hover {
  background: rgba(var(--v-theme-on-surface), 0.06);
}

.channel-row--selected {
  background: rgba(var(--v-theme-primary), 0.14);
  color: rgb(var(--v-theme-primary));
}

.channel-row--current:not(.channel-row--selected) {
  font-weight: 600;
}

.channel-name {
  flex: 1;
  min-width: 0;
  font-size: 0.9rem;
}

.channel-hash {
  opacity: 0.65;
}

.user-count {
  margin-left: 6px;
}

.user-row {
  display: flex;
  align-items: center;
  font-size: 0.78rem;
  color: rgba(var(--v-theme-on-surface), 0.7);
  padding: 2px 12px 2px 28px;
  margin: 0 6px;
}
</style>
