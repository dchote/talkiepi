<template>
  <div>
    <v-progress-linear v-if="loading" indeterminate class="mb-4" />
    <v-alert v-if="error" type="error" density="compact" variant="tonal" class="mb-4">
      {{ error }}
    </v-alert>

    <template v-if="info && !loading">
      <div class="mb-2">
        <span class="text-medium-emphasis">Current:</span>
        {{ info.current || '—' }}
      </div>
      <div class="mb-2">
        <span class="text-medium-emphasis">Latest:</span>
        {{ info.latest || '—' }}
      </div>
      <v-chip
        v-if="info.available"
        size="small"
        color="warning"
        variant="tonal"
        density="compact"
        class="mb-4"
      >
        Update available
      </v-chip>
      <div class="d-flex flex-wrap justify-end mt-4">
        <v-btn
          class="mr-2 mb-2"
          variant="outlined"
          :loading="loading"
          @click="refresh"
        >
          Check for updates
        </v-btn>
        <v-btn
          v-if="info.url"
          class="mb-2"
          variant="text"
          :href="info.url"
          target="_blank"
          rel="noopener"
        >
          Release notes
        </v-btn>
      </div>
    </template>
  </div>
</template>

<script setup>
import { inject, onMounted, ref } from 'vue'
import api from '@/utils/api'

const notify = inject('notify', () => {})

const info = ref(null)
const loading = ref(true)
const error = ref('')

async function load(refreshCheck = false) {
  loading.value = true
  error.value = ''
  try {
    info.value = await api.get(refreshCheck ? '/update?refresh=1' : '/update')
  } catch (e) {
    error.value = e.message || 'Failed to load update info'
    if (refreshCheck) notify(error.value, 'error')
  } finally {
    loading.value = false
  }
}

function refresh() {
  return load(true)
}

onMounted(() => load(false))
</script>
