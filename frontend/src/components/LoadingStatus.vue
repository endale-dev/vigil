<script setup lang="ts">
import { LoaderCircle } from '@lucide/vue'
</script>

<template>
  <div class="loading-status" role="status">
    <LoaderCircle :size="17" class="refreshing" aria-hidden="true" />
    <slot />
  </div>
</template>
