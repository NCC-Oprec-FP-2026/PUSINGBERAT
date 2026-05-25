import { ref, reactive } from 'vue'
import api, { endpoints } from '@/api'

const totalEvents = ref(0)
const criticalEvents = ref(0)
const activeSources = ref(0)
const recentAlerts = ref<any[]>([])
const loading = ref(false)
const severityStats = ref([0, 0, 0, 0, 0]) // [Crit, High, Med, Low, Info]
const eventTimeline = reactive({ labels: [] as string[], data: [] as number[] })
const totalAlerts = ref(0)

export function useDashboard() {

  const fetchDashboardData = async () => {
    loading.value = true
    try {
      // Fetch Alerts
      const alertsRes = await api.get(endpoints.alerts)
      if (alertsRes.data) {
        recentAlerts.value = alertsRes.data.map((alert: any) => ({
          ...alert,
          severity: alert.severity.toUpperCase(),
          title: alert.rule_name || alert.title,
          time: alert.triggered_at,
          status: alert.acknowledged ? 'ACKed' : 'NEW'
        }))
        totalAlerts.value = alertsRes.data.length
        criticalEvents.value = alertsRes.data.filter((a: any) => a.severity === 'critical').length
      }

      // Fetch Sources
      const sourcesRes = await api.get(endpoints.sources)
      if (sourcesRes.data) {
        activeSources.value = sourcesRes.data.filter((s: any) => s.status === 'active').length
      }

      // Fetch Events (Total)
      const eventsRes = await api.get(endpoints.events)
      if (eventsRes.data) {
        totalEvents.value = eventsRes.data.meta?.total || eventsRes.data.data?.length || eventsRes.data.length || 0
      }

      // Fetch Timeline
      const timelineRes = await api.get(endpoints.eventsTimeline)
      if (timelineRes.data && Array.isArray(timelineRes.data)) {
        eventTimeline.labels = timelineRes.data.map((point: any) => {
          const date = new Date(point.hour)
          return date.toLocaleTimeString('en-US', { hour: 'numeric', hour12: true })
        })
        eventTimeline.data = timelineRes.data.map((point: any) => point.count)
      }

      // Fetch Severity Count
      const sevCountRes = await api.get(endpoints.severity_count)
      if (sevCountRes.data) {
        const counts = sevCountRes.data
        severityStats.value = [
          counts.critical || 0,
          counts.high || 0,
          counts.medium || 0,
          counts.low || 0,
          counts.info || 0
        ]
      }
    } catch (e) {
      console.error('Failed to fetch dashboard data', e)
    } finally {
      loading.value = false
    }
  }

  return {
    totalEvents,
    criticalEvents,
    activeSources,
    recentAlerts,
    totalAlerts,
    severityStats,
    eventTimeline,
    loading,
    fetchDashboardData
  }
}
