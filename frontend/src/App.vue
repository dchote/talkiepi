<template>
  <v-app>
    <DefaultLayout>
      <router-view />
    </DefaultLayout>
    <v-snackbar
      v-model="snackbar.show"
      :color="snackbar.color"
      :timeout="snackbar.timeout"
      location="bottom"
    >
      {{ snackbar.message }}
    </v-snackbar>
  </v-app>
</template>

<script setup>
import { provide, reactive } from 'vue'
import DefaultLayout from '@/layouts/DefaultLayout.vue'

const snackbar = reactive({
  show: false,
  message: '',
  color: 'info',
  timeout: 4000,
})

function notify(message, color = 'info', timeout = 4000) {
  snackbar.message = message
  snackbar.color = color
  snackbar.timeout = timeout
  snackbar.show = true
}

provide('notify', notify)
</script>
