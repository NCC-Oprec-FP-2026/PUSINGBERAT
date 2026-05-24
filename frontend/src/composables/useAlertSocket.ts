import { computed, ref } from 'vue'

export type AlertToast = {
  id: string
  rule_name: string
  severity: string
  title: string
  raw_line?: string
}

type AlertEnvelope = {
  type: string
  data?: AlertToast
  payload?: AlertToast
}

type SocketStatus = 'connected' | 'connecting' | 'disconnected'

const toasts = ref<AlertToast[]>([])
const status = ref<SocketStatus>('disconnected')

let socket: WebSocket | null = null
let reconnectTimer: number | undefined
let shouldReconnect = false

const statusLabel = computed(() => {
  if (status.value === 'connected') return 'Connected'
  if (status.value === 'connecting') return 'Connecting'
  return 'Disconnected'
})

const isConnected = computed(() => status.value === 'connected')

function websocketURL() {
  const configured = import.meta.env.VITE_WS_URL || import.meta.env.VITE_WS_BASE_URL
  if (configured) {
    return configured
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.port === '5173' ? `${window.location.hostname}:8080` : window.location.host
  return `${protocol}//${host}/ws`
}

function clearReconnectTimer() {
  if (reconnectTimer) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = undefined
  }
}

function scheduleReconnect() {
  clearReconnectTimer()
  reconnectTimer = window.setTimeout(connectAlerts, 2000)
}

function pushToast(alert: AlertToast) {
  toasts.value = [alert, ...toasts.value].slice(0, 4)
  window.setTimeout(() => {
    toasts.value = toasts.value.filter((toast) => toast.id !== alert.id)
  }, 6000)
}

function connectAlerts() {
  if (socket || status.value === 'connecting') {
    return
  }

  shouldReconnect = true
  status.value = 'connecting'
  socket = new WebSocket(websocketURL())

  socket.onopen = () => {
    status.value = 'connected'
  }

  socket.onmessage = (event) => {
    let envelope: AlertEnvelope
    try {
      envelope = JSON.parse(event.data) as AlertEnvelope
    } catch {
      return
    }
    const alert = envelope.payload ?? envelope.data
    if (envelope.type === 'alert' && alert) {
      pushToast(alert)
    }
  }

  socket.onerror = () => {
    status.value = 'disconnected'
  }

  socket.onclose = () => {
    socket = null
    status.value = 'disconnected'
    if (shouldReconnect) {
      scheduleReconnect()
    }
  }
}

function disconnectAlerts() {
  shouldReconnect = false
  clearReconnectTimer()
  socket?.close()
  socket = null
  status.value = 'disconnected'
}

export function useAlertSocket() {
  return {
    connectAlerts,
    disconnectAlerts,
    isConnected,
    status,
    statusLabel,
    toasts,
  }
}
