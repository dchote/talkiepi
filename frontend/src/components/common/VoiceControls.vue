<template>
  <div v-if="showPtt || showCommunicator" class="d-flex flex-wrap align-center">
    <v-btn
      v-if="showPtt"
      :class="buttonClass"
      :size="size"
      color="error"
      variant="tonal"
      :disabled="!connected"
      @mousedown="ptt(true)"
      @mouseup="ptt(false)"
      @mouseleave="ptt(false)"
      @touchstart.prevent="ptt(true)"
      @touchend.prevent="ptt(false)"
    >
      PTT
    </v-btn>
    <v-btn
      v-if="showCommunicator"
      :class="buttonClass"
      :size="size"
      color="warning"
      variant="tonal"
      :loading="busy"
      :disabled="!connected"
      @click="toggleCommunicator"
    >
      Toggle Communicator
    </v-btn>
  </div>
</template>

<script setup>
import { computed, inject, ref } from 'vue'
import api from '@/utils/api'

const props = defineProps({
  mode: {
    type: String,
    default: '',
  },
  connected: {
    type: Boolean,
    default: false,
  },
  size: {
    type: String,
    default: 'default',
  },
  /** Extra class on each button (e.g. mr-2 mb-2 on status page) */
  buttonClass: {
    type: String,
    default: 'mr-2',
  },
})

const emit = defineEmits(['changed'])
const notify = inject('notify', () => {})
const busy = ref(false)

const showPtt = computed(() => props.mode === 'push_to_talk')
const showCommunicator = computed(() => props.mode === 'communicator')

async function ptt(pressed) {
  try {
    await api.post('/ptt', { pressed })
    emit('changed')
  } catch (e) {
    notify(e.message || 'PTT failed', 'error')
  }
}

async function toggleCommunicator() {
  busy.value = true
  try {
    const res = await api.post('/communicator/toggle')
    notify(`Communicator: ${res.state}`, 'info')
    emit('changed')
  } catch (e) {
    notify(e.message || 'Communicator toggle failed', 'error')
  } finally {
    busy.value = false
  }
}
</script>
