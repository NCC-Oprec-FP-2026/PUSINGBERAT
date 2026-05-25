<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import AppLayout from './components/layout/AppLayout.vue'
import { useAlertSocket } from './composables/useAlertSocket'

const { connectAlerts, disconnectAlerts, toasts } = useAlertSocket()

onMounted(connectAlerts)
onBeforeUnmount(disconnectAlerts)
</script>

<template>
  <AppLayout />
  <div class="fixed right-4 top-4 z-50 flex w-[min(24rem,calc(100vw-2rem))] flex-col gap-3">
    <div
      v-for="toast in toasts"
      :key="toast.id"
      class="border-l-4 bg-siem-surface px-4 py-3 shadow-xl ring-1 ring-siem-border"
      :class="{
        'border-severity-critical': toast.severity === 'critical',
        'border-severity-high': toast.severity === 'high',
        'border-severity-medium': toast.severity === 'medium',
        'border-severity-low': toast.severity === 'low',
        'border-severity-info': toast.severity === 'info',
      }"
    >
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <p class="text-sm font-semibold text-siem-text-primary">{{ toast.title }}</p>
          <p class="mt-1 text-xs uppercase text-siem-text-secondary">{{ toast.rule_name }} - {{ toast.severity }}</p>
          <p v-if="toast.raw_line" class="mt-2 line-clamp-2 text-xs text-siem-text-secondary">
            {{ toast.raw_line }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
