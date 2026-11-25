<script setup lang="ts">
import { Activity } from '@lucide/vue'
defineProps<{ compact?: boolean }>()
</script>

<template>
  <span class="console-brand" aria-label="Vigil">
    <span class="brand-symbol"><Activity :size="22" :stroke-width="2.5" aria-hidden="true" /></span>
    <span v-if="!compact" class="brand-word" aria-hidden="true">vigil<span>.</span></span>
  </span>
</template>
