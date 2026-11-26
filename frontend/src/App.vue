<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <RouterView :key="$route.name === 'monitor-detail' ? $route.path : undefined" />
</template>
