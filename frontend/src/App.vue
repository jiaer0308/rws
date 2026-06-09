<script setup lang="ts">
import AppLayout from './components/layout/app-layout.vue'
</script>

<template>
  <AppLayout>
    <router-view />
  </AppLayout>
</template>

<style>
/* Global scrollbar style for dark theme */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--bg-main);
}

::-webkit-scrollbar-thumb {
  background: var(--bg-card-border);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}
</style>
