<template>
  <v-card
    :class="['standard-page-card', cardClass]"
    :elevation="elevation"
    v-bind="$attrs"
  >
    <template v-if="title || $slots.header">
      <v-card-title :class="['standard-card-title', 'd-flex', 'align-center', headerClass]">
        <slot name="header">
          <span :class="[titleClass, 'header-truncate']">{{ title }}</span>
        </slot>
      </v-card-title>
      <v-divider />
    </template>
    <div v-if="$slots.toolbar" class="standard-card-toolbar px-4 py-2">
      <slot name="toolbar" />
    </div>
    <v-card-text :class="contentClass">
      <slot />
    </v-card-text>
    <template v-if="$slots.actions">
      <v-divider />
      <v-card-actions class="card-actions">
        <slot name="actions" />
      </v-card-actions>
    </template>
  </v-card>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    default: '',
  },
  cardClass: {
    type: String,
    default: '',
  },
  contentClass: {
    type: String,
    default: '',
  },
  titleClass: {
    type: String,
    default: 'text-h6 font-weight-bold',
  },
  headerClass: {
    type: String,
    default: '',
  },
  elevation: {
    type: [Number, String],
    default: undefined,
  },
})
</script>
