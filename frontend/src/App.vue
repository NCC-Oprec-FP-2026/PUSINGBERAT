<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import AppLayout from './components/layout/AppLayout.vue'

type AlertToast = {
  id: string
  rule_name: string
  severity: string
  title: string
  raw_line?: string
}

type AlertEnvelope = {
  type: string
  data: AlertToast
}

const toasts = ref<AlertToast[]>([])
let socket: WebSocket | null = null
let reconnectTimer: number | undefined

function websocketURL() {
  const configured = import.meta.env.VITE_WS_URL || import.meta.env.VITE_WS_BASE_URL
  if (configured) {
    return configured
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws`
}

function connectAlerts() {
  socket = new WebSocket(websocketURL())

  socket.onmessage = (event) => {
    let envelope: AlertEnvelope
    try {
      envelope = JSON.parse(event.data) as AlertEnvelope
    } catch {
      return
    }
    if (envelope.type !== 'alert') {
      return
    }

    const alert = envelope.data
    toasts.value = [alert, ...toasts.value].slice(0, 4)
    window.setTimeout(() => {
      toasts.value = toasts.value.filter((toast) => toast.id !== alert.id)
    }, 6000)
  }

  socket.onclose = () => {
    reconnectTimer = window.setTimeout(connectAlerts, 2000)
  }
}

onMounted(connectAlerts)

onBeforeUnmount(() => {
  if (reconnectTimer) {
    window.clearTimeout(reconnectTimer)
  }
  socket?.close()
})
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
