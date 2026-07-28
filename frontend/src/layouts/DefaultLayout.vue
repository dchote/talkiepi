<template>
  <!--
    Vuetify application layout (drawer + app-bar + main as siblings of v-app).
    Brand is a list item so rail mode shows the icon automatically.
    Breakpoints: <960 temporary, 960–1280 rail, ≥1280 expanded.
  -->
  <v-navigation-drawer
    v-model="drawer"
    v-model:rail="rail"
    :permanent="!isMobile"
    :temporary="isMobile"
    :mobile-breakpoint="960"
    app
    theme="dark"
    class="sidebar-nav"
    width="260"
    rail-width="56"
  >
    <v-list nav density="comfortable" class="pa-2">
      <v-list-item
        prepend-icon="mdi-radio-handheld"
        title="talkiepi"
        to="/"
        rounded="lg"
        class="sidebar-brand-item mb-1"
      />

      <v-divider class="sidebar-divider my-2" />

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

    <template #append>
      <div v-show="!rail" class="px-4 pb-4">
        <a
          :href="docsHref"
          target="_blank"
          rel="noopener"
          class="sidebar-docs-link"
        >
          API Docs
        </a>
      </div>
    </template>
  </v-navigation-drawer>

  <v-app-bar app flat theme="dark" class="content-header">
    <v-app-bar-nav-icon
      v-if="isMobile || rail"
      @click="toggleDrawer"
    />

    <v-app-bar-title v-if="isMobile">
      talkiepi
    </v-app-bar-title>

    <v-spacer />

    <v-btn
      icon
      variant="text"
      :title="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
      @click="toggleTheme"
    >
      <v-icon :icon="isDark ? 'mdi-weather-sunny' : 'mdi-weather-night'" />
    </v-btn>

    <v-chip
      class="mx-3"
      size="small"
      variant="tonal"
      :color="connected ? 'success' : 'error'"
    >
      <v-icon
        start
        size="small"
        :icon="connected ? 'mdi-access-point' : 'mdi-access-point-off'"
      />
      {{ connected ? 'Connected' : 'Offline' }}
    </v-chip>
  </v-app-bar>

  <v-main>
    <slot />
  </v-main>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useTheme, useDisplay } from 'vuetify'
import { THEME_KEY } from '@/plugins/vuetify'
import api from '@/utils/api'
import { getIngressBase } from '@/utils/ingress'

const route = useRoute()
const theme = useTheme()
const { width } = useDisplay()

const drawer = ref(null)
const rail = ref(false)
const connected = ref(false)
let statusTimer = null

const navItems = [
  { title: 'Channels', icon: 'mdi-forum', to: '/' },
  { title: 'Status', icon: 'mdi-access-point', to: '/status' },
  { title: 'Settings', icon: 'mdi-cog', to: '/settings' },
]

const docsHref = computed(() => `${getIngressBase()}/docs`)
const isMobile = computed(() => width.value < 960)
const isDark = computed(() => theme.global.current.value.dark)

watch(width, (w) => {
  if (w < 960) {
    drawer.value = false
    rail.value = false
  } else if (w < 1280) {
    drawer.value = true
    rail.value = true
  } else {
    drawer.value = true
    rail.value = false
  }
}, { immediate: true })

function isActive(to) {
  if (to === '/') {
    return route.path === '/' || route.path === '' || route.path === '/channels' || route.path === '/chat'
  }
  if (to === '/settings') {
    return route.path === '/settings' || route.path.startsWith('/settings/')
  }
  return route.path === to || route.path.startsWith(`${to}/`)
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

async function refreshConnected() {
  try {
    const st = await api.get('/status')
    connected.value = !!st.connected
  } catch {
    connected.value = false
  }
}

onMounted(() => {
  refreshConnected()
  statusTimer = setInterval(refreshConnected, 3000)
})

onUnmounted(() => {
  if (statusTimer) clearInterval(statusTimer)
})
</script>

<style scoped>
.sidebar-nav {
  background: #1f2937 !important;
  border-right: none !important;
}

.sidebar-nav :deep(.v-navigation-drawer__content) {
  background: #1f2937;
}

.sidebar-brand-item :deep(.v-list-item-title) {
  font-weight: 600;
  font-size: 1.1rem;
}

.sidebar-nav :deep(.v-list-item) {
  color: rgba(255, 255, 255, 0.85);
}

.sidebar-nav :deep(.v-list-item:hover) {
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
}

.sidebar-item :deep(.v-list-item--active) {
  background: rgba(255, 255, 255, 0.1) !important;
  color: #fff !important;
}

.sidebar-divider {
  border-color: rgba(255, 255, 255, 0.12) !important;
}

.sidebar-docs-link {
  color: rgba(255, 255, 255, 0.55);
  font-size: 0.8rem;
  text-decoration: none;
}

.sidebar-docs-link:hover {
  color: #fff;
  text-decoration: underline;
}

.content-header {
  background: #1f2937 !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
}
</style>
