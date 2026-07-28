<template>
  <v-container>
    <StandardCard title="Settings">
      <v-tabs
        :model-value="currentTab"
        color="primary"
        density="compact"
        @update:model-value="goTab"
      >
        <v-tab value="server" class="mr-2" to="/settings/server">
          <v-icon start size="small" class="mr-2">mdi-server</v-icon>
          Server
        </v-tab>
        <v-tab value="audio" class="mr-2" to="/settings/audio">
          <v-icon start size="small" class="mr-2">mdi-volume-high</v-icon>
          Audio
        </v-tab>
        <v-tab value="updates" to="/settings/updates">
          <v-icon start size="small" class="mr-2">mdi-update</v-icon>
          Updates
        </v-tab>
      </v-tabs>
      <v-divider class="mb-4" />

      <RouterView />
    </StandardCard>
  </v-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import StandardCard from '@/components/common/StandardCard.vue'

const route = useRoute()
const router = useRouter()

const currentTab = computed(() => {
  if (route.path.includes('/settings/audio')) return 'audio'
  if (route.path.includes('/settings/updates')) return 'updates'
  return 'server'
})

function goTab(value) {
  const paths = {
    server: '/settings/server',
    audio: '/settings/audio',
    updates: '/settings/updates',
  }
  const path = paths[value] || paths.server
  if (route.path !== path) {
    router.push(path)
  }
}
</script>
