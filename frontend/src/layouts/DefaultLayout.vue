<template>
  <v-navigation-drawer
    v-model="drawer"
    v-model:rail="rail"
    :permanent="!isMobile"
    :temporary="isMobile"
    :mobile-breakpoint="960"
    app
    class="sidebar-nav"
    width="240"
    rail-width="56"
  >
    <div class="sidebar-brand pa-4">
      <router-link to="/" class="d-flex align-center text-decoration-none">
        <v-icon class="mr-2" color="primary">mdi-walkie-talkie</v-icon>
        <span v-show="!rail" class="sidebar-brand-text">talkiepi</span>
      </router-link>
    </div>
    <v-divider class="sidebar-divider" />
    <v-list nav density="comfortable" class="sidebar-list">
      <v-list-item
        v-for="item in navItems"
        :key="item.to"
        :prepend-icon="item.icon"
        :title="item.title"
        :to="item.to"
        :active="isActive(item.to)"
        rounded="lg"
        class="sidebar-item"
      />
    </v-list>
  </v-navigation-drawer>

  <v-app-bar app flat class="content-header">
    <v-btn
      v-if="isMobile || rail"
      icon
      variant="text"
      class="app-bar-icon-btn px-2"
      :title="isMobile ? 'Menu' : (rail ? 'Expand' : 'Collapse')"
      @click="toggleDrawer"
    >
      <v-icon>mdi-menu</v-icon>
    </v-btn>
    <v-app-bar-title class="text-body-1 font-weight-medium">
      {{ pageTitle }}
    </v-app-bar-title>
    <v-spacer />
    <v-btn
      icon
      variant="text"
      size="small"
      class="px-2"
      :title="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
      @click="toggleTheme"
    >
      <v-icon>{{ isDark ? 'mdi-weather-sunny' : 'mdi-weather-night' }}</v-icon>
    </v-btn>
  </v-app-bar>

  <v-main>
    <slot />
  </v-main>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useTheme, useDisplay } from 'vuetify'
import { THEME_KEY } from '@/plugins/vuetify'

const route = useRoute()
const theme = useTheme()
const { width } = useDisplay()
const drawer = ref(null)
const rail = ref(false)

const navItems = [
  { title: 'Status', icon: 'mdi-access-point', to: '/' },
  { title: 'Server', icon: 'mdi-server', to: '/server' },
  { title: 'Audio', icon: 'mdi-volume-high', to: '/audio' },
  { title: 'Channels', icon: 'mdi-account-group', to: '/channels' },
  { title: 'Chat', icon: 'mdi-message-text', to: '/chat' },
]

const isMobile = computed(() => width.value < 960)
const isDark = computed(() => theme.global.current.value.dark)
const pageTitle = computed(() => {
  const match = navItems.find((item) => isActive(item.to))
  return match ? match.title : 'talkiepi'
})

watch(width, (w) => {
  if (w < 960) {
    rail.value = false
    drawer.value = false
  } else if (w < 1280) {
    rail.value = true
    drawer.value = true
  } else {
    rail.value = false
    drawer.value = true
  }
}, { immediate: true })

function isActive(to) {
  if (to === '/') {
    return route.path === '/' || route.path === ''
  }
  return route.path === to || route.path.startsWith(to + '/')
}

function toggleDrawer() {
  if (isMobile.value) {
    drawer.value = !drawer.value
  } else {
    rail.value = !rail.value
  }
}

function toggleTheme() {
  const next = theme.global.current.value.dark ? 'light' : 'dark'
  theme.change(next)
  localStorage.setItem(THEME_KEY, next)
}
</script>

<style scoped>
.content-header {
  background: rgb(var(--v-theme-surface)) !important;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.08);
}

.v-theme--dark .content-header {
  border-bottom-color: rgba(255, 255, 255, 0.12);
}

.sidebar-brand-text {
  font-weight: 600;
  color: rgb(var(--v-theme-on-surface));
}

.sidebar-section-label {
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: rgba(var(--v-theme-on-surface), 0.5);
}
</style>
