<template>
  <v-container>
    <StandardCard title="Chat">
      <v-alert v-if="error" type="error" density="compact" variant="tonal" class="mb-4">
        {{ error }}
      </v-alert>
      <v-alert v-if="success" type="success" density="compact" variant="tonal" class="mb-4">
        Message sent.
      </v-alert>

      <v-textarea
        v-model="message"
        label="Message"
        variant="outlined"
        density="compact"
        hide-details="auto"
        rows="3"
        auto-grow
        class="mb-4"
        @keydown.ctrl.enter="send"
      />
      <v-text-field
        v-model.number="channelId"
        label="Channel ID (0 = current)"
        type="number"
        variant="outlined"
        density="compact"
        hide-details="auto"
        autocomplete="off"
        class="mb-4"
      />

      <div class="d-flex justify-end mt-4">
        <v-btn
          color="primary"
          variant="elevated"
          :loading="sending"
          :disabled="!message.trim()"
          @click="send"
        >
          Send
        </v-btn>
      </div>
    </StandardCard>
  </v-container>
</template>

<script setup>
import { inject, ref } from 'vue'
import StandardCard from '@/components/common/StandardCard.vue'
import api from '@/utils/api'

const notify = inject('notify', () => {})

const message = ref('')
const channelId = ref(0)
const sending = ref(false)
const error = ref('')
const success = ref(false)

async function send() {
  if (!message.value.trim()) return
  error.value = ''
  success.value = false
  sending.value = true
  try {
    await api.post('/text', {
      message: message.value,
      channel_id: channelId.value || 0,
    })
    success.value = true
    message.value = ''
    notify('Message sent', 'success')
  } catch (e) {
    error.value = e.message || 'Failed to send'
    notify(error.value, 'error')
  } finally {
    sending.value = false
  }
}
</script>
