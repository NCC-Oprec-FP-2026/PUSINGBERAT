<script setup lang="ts">
import AppSidebar from './AppSidebar.vue'
import AppTopBar from './AppTopBar.vue'
</script>

<template>
  <div class="min-h-screen bg-siem-bg">
    <!-- Sidebar (fixed, 240px / w-60) -->
    <AppSidebar />

    <!-- Main content area (offset by sidebar width) -->
    <div class="ml-60 flex min-h-screen flex-col">
      <!-- Top bar -->
      <AppTopBar />

      <!-- Page content -->
      <main class="flex-1 p-6">
        <router-view />
      </main>
    </div>
  </div>
</template>
