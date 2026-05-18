<script setup lang="ts">
import { useRoute } from 'vue-router'
import { computed } from 'vue'

const route = useRoute()

const pageTitle = computed<string>(() => {
  const name = route.name as string | undefined
  if (!name) return 'PUSINGBERAT'

  // Convert route name to display title
  const titles: Record<string, string> = {
    Dashboard: 'Dashboard',
    Alerts: 'Alerts',
    Events: 'Events',
    LogSources: 'Log Sources',
    Rules: 'Rules',
  }
  return titles[name] ?? name
})
</script>

<template>
  <header
    class="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-siem-border bg-siem-surface/80 px-6 backdrop-blur-sm"
  >
    <!-- Left: Page title -->
    <div>
      <h2 class="text-lg font-semibold text-siem-text-primary">
        {{ pageTitle }}
      </h2>
    </div>

    <!-- Right: Status indicators -->
    <div class="flex items-center gap-4">
      <!-- WebSocket connection indicator (placeholder) -->
      <div class="flex items-center gap-2 rounded-full border border-siem-border bg-siem-bg px-3 py-1.5">
        <span class="relative flex h-2.5 w-2.5">
          <!-- Pulse animation ring -->
          <span
            class="absolute inline-flex h-full w-full animate-ping rounded-full bg-severity-info opacity-75"
          />
          <!-- Solid dot -->
          <span class="relative inline-flex h-2.5 w-2.5 rounded-full bg-severity-info" />
        </span>
        <span class="text-xs font-medium text-siem-text-secondary">Disconnected</span>
      </div>
    </div>
  </header>
</template>
