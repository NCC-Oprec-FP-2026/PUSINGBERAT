<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useDashboard } from '@/composables/useDashboard'
import api, { endpoints } from '@/api'

const { recentAlerts, fetchDashboardData } = useDashboard()

onMounted(async () => {
  if (recentAlerts.value.length === 0) {
    await fetchDashboardData()
  }
})

const alertSeverityMap: Record<string, string> = {
  CRITICAL: 'bg-rose-500/20 text-rose-400 border border-rose-500/30',
  HIGH: 'bg-orange-500/20 text-orange-400 border border-orange-500/30',
  MEDIUM: 'bg-amber-500/20 text-amber-400 border border-amber-500/30',
  LOW: 'bg-blue-500/20 text-blue-400 border border-blue-500/30',
  INFO: 'bg-slate-500/20 text-slate-400 border border-slate-500/30'
}

const selectedAlert = ref<any | null>(null)
const isModalOpen = ref(false)

const openAlert = async (id: string) => {
  try {
    const res = await api.get(`${endpoints.alerts}/${id}`)
    selectedAlert.value = res.data
    isModalOpen.value = true
  } catch (error) {
    console.error("Failed to fetch alert details", error)
  }
}

const closeModal = () => {
  isModalOpen.value = false
  selectedAlert.value = null
}

const acknowledgeAlert = async (id: string) => {
  try {
    await api.patch(`${endpoints.alerts}/${id}/acknowledge`)
    await fetchDashboardData()
    closeModal()
  } catch (error) {
    console.error("Failed to acknowledge alert", error)
  }
}

const deleteAlert = async (id: string) => {
  if (!confirm("Are you sure you want to delete this alert?")) return
  
  try {
    await api.delete(`${endpoints.alerts}/${id}`)
    await fetchDashboardData()
    closeModal()
  } catch (error) {
    console.error("Failed to delete alert", error)
  }
}

</script>

<template>
  <div class="flex flex-col gap-6 w-full">
    <div class="flex justify-between items-center">
      <div>
        <h1 class="text-2xl font-bold text-slate-100">Alerts</h1>
        <p class="mt-1 text-sm text-slate-400">View and manage system security alerts.</p>
      </div>
      <button @click="fetchDashboardData" class="px-4 py-2 bg-slate-800 text-slate-200 text-sm font-semibold rounded-lg border border-slate-700 hover:bg-slate-700 transition">
        Refresh
      </button>
    </div>

    <!-- Alert Modal -->
    <div v-if="isModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4" @click.self="closeModal">
      <div class="bg-[#030712] border border-slate-800/80 rounded-2xl w-full max-w-3xl overflow-hidden shadow-2xl flex flex-col max-h-[90vh]">
        <div class="p-6 border-b border-slate-800 flex justify-between items-center bg-slate-900/40">
          <h2 class="text-lg font-semibold text-slate-200">Alert Details: {{ selectedAlert?.rule_name }}</h2>
          <button @click="closeModal" class="text-slate-400 hover:text-slate-200 transition">&times;</button>
        </div>
        
        <div class="p-6 overflow-y-auto flex-1 flex flex-col gap-4">
          <div class="flex justify-between items-center bg-slate-900/40 p-4 rounded-xl border border-slate-800/60 shadow-sm">
            <div>
              <span class="text-[10px] text-slate-400 block uppercase font-bold tracking-widest">Incident Title</span>
              <span class="text-base font-semibold text-slate-200 tracking-tight">{{ selectedAlert?.title || selectedAlert?.rule_name }}</span>
            </div>
            <span :class="['px-3 py-1 text-xs font-bold uppercase rounded-lg', alertSeverityMap[(selectedAlert?.severity || '').toUpperCase()] || alertSeverityMap.INFO]">
              {{ selectedAlert?.severity?.toUpperCase() }}
            </span>
          </div>

          <div class="space-y-1.5">
            <span class="text-[10px] text-slate-400 block uppercase font-bold tracking-widest">Description</span>
            <p class="m-0 text-slate-300 text-sm leading-relaxed bg-slate-950 border border-slate-800/60 p-3.5 rounded-xl shadow-inner">
              {{ selectedAlert?.description || 'No description available.' }}
            </p>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="bg-slate-950/60 p-3.5 rounded-xl border border-slate-800/40 space-y-1">
              <span class="text-[10px] text-slate-400 block font-bold tracking-wider uppercase">TRIGGERED TIME</span>
              <span class="text-xs font-mono text-indigo-400 block">{{ new Date(selectedAlert?.triggered_at).toLocaleString() }}</span>
            </div>
            <div class="bg-slate-950/60 p-3.5 rounded-xl border border-slate-800/40 space-y-1 flex flex-col justify-between">
              <span class="text-[10px] text-slate-400 block font-bold tracking-wider uppercase">STATUS</span>
              <div>
                <span v-if="selectedAlert?.acknowledged" class="bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 px-2 py-1 text-[10px] font-bold rounded uppercase">Acknowledged</span>
                <span v-else class="bg-rose-500/20 text-rose-400 border border-rose-500/30 px-2 py-1 text-[10px] font-bold rounded uppercase">New Incident</span>
              </div>
            </div>
            <div class="bg-slate-950/60 p-3.5 rounded-xl border border-slate-800/40 space-y-1 sm:col-span-2">
              <span class="text-[10px] text-slate-400 block font-bold tracking-wider uppercase">INCIDENT ID (UUID)</span>
              <span class="text-xs font-mono text-slate-300 block truncate select-all">{{ selectedAlert?.id }}</span>
            </div>
          </div>

          <div class="space-y-1.5">
            <span class="text-[10px] text-slate-400 block uppercase font-bold tracking-widest">Raw Log Entry Payload</span>
            <pre class="m-0 p-4 bg-slate-950 text-emerald-400 font-mono text-xs rounded-xl overflow-x-auto whitespace-pre-wrap shadow-md border-l-2 border-emerald-500/80">{{ selectedAlert?.raw_line }}</pre>
          </div>
        </div>

        <div class="p-6 border-t border-slate-800 flex justify-end gap-3 bg-slate-900/40">
          <button @click="deleteAlert(selectedAlert?.id)" class="px-4 py-2 bg-rose-500/10 text-rose-500 hover:bg-rose-500/20 border border-rose-500/20 rounded-lg text-sm font-semibold transition">
            Delete
          </button>
          <button @click="acknowledgeAlert(selectedAlert?.id)" :disabled="selectedAlert?.acknowledged" class="px-4 py-2 bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg text-sm font-semibold shadow-md transition">
            Acknowledge
          </button>
        </div>
      </div>
    </div>

    <!-- Alerts Table -->
    <div class="bg-slate-900/20 backdrop-blur-md rounded-2xl border border-slate-800/60 overflow-hidden shadow-[0_12px_40px_rgba(0,0,0,0.2)] w-full"> 
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-slate-900/50 text-xs font-semibold text-slate-400 uppercase tracking-widest border-b border-slate-800/60">
              <th class="p-4">Severity</th>
              <th class="p-4">Title</th>
              <th class="p-4">Time</th>
              <th class="p-4">Status</th>
              <th class="p-4">Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="recentAlerts.length === 0">
              <td colspan="5" class="p-4 text-center text-slate-500">No alerts found.</td>
            </tr>
            <tr v-for="alert in recentAlerts" :key="alert.id" class="border-b border-slate-800/40 hover:bg-slate-800/20 transition">
              <td class="p-4">
                <span :class="['px-2.5 py-1 text-[10px] font-bold tracking-wider rounded-lg uppercase', alertSeverityMap[alert.severity] || alertSeverityMap.INFO]">
                  {{ alert.severity }}
                </span>
              </td>
              <td class="p-4 font-semibold text-slate-200 text-sm tracking-tight">{{ alert.title }}</td>
              <td class="p-4 text-xs text-slate-400 font-medium">{{ new Date(alert.time).toLocaleString() }}</td>
              <td class="p-4 text-xs">
                <span :class="alert.status === 'ACKed' ? 'text-emerald-400' : 'text-rose-400 font-bold'">{{ alert.status }}</span>
              </td>
              <td class="p-4">
                <button @click="openAlert(alert.id)" class="px-3 py-1.5 bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-slate-100 rounded border border-slate-700 transition text-xs font-semibold">
                  View
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
