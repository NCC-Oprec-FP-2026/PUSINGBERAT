<script setup lang="ts">
import { onMounted } from 'vue'
import { useDashboard } from '@/composables/useDashboard'
import { Doughnut, Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  ArcElement,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement
} from 'chart.js'

ChartJS.register(Title, Tooltip, Legend, ArcElement, CategoryScale, LinearScale, PointElement, LineElement)

const {
  totalEvents,
  criticalEvents,
  activeSources,
  recentAlerts,
  severityStats,
  eventTimeline,
  loading,
  fetchDashboardData
} = useDashboard()

onMounted(async () => {
  await fetchDashboardData()
})

const alertSeverityMap: Record<string, string> = {
  CRITICAL: 'bg-rose-500/20 text-rose-400 border border-rose-500/30',
  HIGH: 'bg-orange-500/20 text-orange-400 border border-orange-500/30',
  MEDIUM: 'bg-amber-500/20 text-amber-400 border border-amber-500/30',
  LOW: 'bg-blue-500/20 text-blue-400 border border-blue-500/30',
  INFO: 'bg-slate-500/20 text-slate-400 border border-slate-500/30'
}

const donutOptions = {
  maintainAspectRatio: false,
  plugins: { legend: { position: 'bottom' as const, labels: { color: '#e2e8f0' } } }
}

const lineOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      mode: 'index' as const,
      intersect: false,
      backgroundColor: 'rgba(15, 23, 42, 0.9)',
      titleColor: '#ffffff',
      bodyColor: '#e2e8f0',
    }
  },
  scales: {
    x: {
      grid: { display: false },
      ticks: { color: '#64748b' }
    },
    y: {
      beginAtZero: true,
      grid: { color: 'rgba(15, 23, 42, 0.08)' },
      ticks: { color: '#64748b' }
    }
  }
}
</script>

<template>
  <div class="flex flex-col gap-6 w-full">
    <div class="important_metrics flex flex-col md:flex-row justify-between gap-4 w-full"> 
      <div class="flex-1 bg-slate-900/40 border border-slate-800/60 rounded-2xl p-6 backdrop-blur-md shadow-sm">
        <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-2">Total Events</h3>
        <span class="text-4xl sm:text-5xl font-extrabold text-slate-100 tracking-tight">{{ totalEvents }}</span>
      </div>

      <div class="flex-1 bg-slate-900/40 border border-slate-800/60 rounded-2xl p-6 backdrop-blur-md shadow-sm">
        <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-2">Critical Alerts</h3>
        <span class="text-4xl sm:text-5xl font-extrabold text-rose-400 tracking-tight drop-shadow-[0_0_15px_rgba(244,63,94,0.1)]">{{ criticalEvents }}</span>
      </div>

      <div class="flex-1 bg-slate-900/40 border border-slate-800/60 rounded-2xl p-6 backdrop-blur-md shadow-sm">
        <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-2">Active Sources</h3>
        <span class="text-4xl sm:text-5xl font-extrabold text-slate-100 tracking-tight">{{ activeSources }}</span>
      </div>

      <div class="flex-1 bg-slate-900/40 border border-slate-800/60 rounded-2xl p-6 backdrop-blur-md shadow-sm flex flex-col justify-between">
        <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest">Log Resources</h3>
        <span class="text-xl sm:text-2xl font-extrabold tracking-wider block mt-2" :class="activeSources > 0 ? 'text-emerald-400' : 'text-rose-400'">
            {{ activeSources > 0 ? "OK" : "NOT OK" }}
        </span>
      </div>
    </div>

    <div class="flex flex-col lg:flex-row gap-6 w-full">
      <div class="w-full lg:w-1/3 bg-slate-900/30 backdrop-blur-md border border-slate-800/60 rounded-2xl p-4 flex flex-col items-center"> 
          <h3 class="w-full text-sm font-semibold text-slate-300 mb-4 text-left">Alerts by Severity</h3>
          <div class="w-full h-64">
              <Doughnut 
                  :data="{
                      labels: ['Critical', 'High', 'Medium', 'Low', 'Info'],
                      datasets: [{
                          data: severityStats,
                          backgroundColor: ['#EF4444', '#F97316', '#EAB308', '#3B82F6', '#6B7280'],
                          borderWidth: 0
                      }]
                  }" 
                  :options="donutOptions"
              />
          </div>
      </div>

      <div class="w-full lg:w-2/3 bg-slate-900/30 backdrop-blur-md border border-slate-800/60 rounded-2xl p-4 flex flex-col items-center"> 
          <h3 class="w-full text-sm font-semibold text-slate-300 mb-4 text-left">Events Timeline</h3>
          <div class="w-full h-64">
              <Line 
                  :data="{
                      labels: eventTimeline.labels,
                      datasets: [{
                          label: 'Events',
                          data: eventTimeline.data,
                          fill: true,
                          backgroundColor: 'rgba(59, 130, 246, 0.15)',
                          borderColor: 'rgba(59, 130, 246, 1)',
                          pointBackgroundColor: '#ffffff',
                          pointBorderColor: 'rgba(59, 130, 246, 1)',
                          tension: 0.25
                      }]
                  }" 
                  :options="lineOptions"
              />
          </div>
      </div>
    </div>

    <div class="recent_alert bg-slate-900/20 backdrop-blur-md rounded-2xl border border-slate-800/60 overflow-hidden shadow-[0_12px_40px_rgba(0,0,0,0.2)] w-full"> 
      <h3 class="text-sm font-semibold text-slate-300 m-4">Recent Alerts</h3>
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-slate-900/50 text-xs font-semibold text-slate-400 uppercase tracking-widest border-b border-slate-800/60">
              <th class="p-4">Severity</th>
              <th class="p-4">Title</th>
              <th class="p-4">Time</th>
              <th class="p-4">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && recentAlerts.length === 0">
              <td colspan="4" class="p-4 text-center text-slate-500">Loading...</td>
            </tr>
            <tr v-for="alert in recentAlerts.slice(0, 5)" :key="alert.id" class="border-b border-slate-800/40 hover:bg-slate-800/20 transition">
              <td class="p-4">
                <span :class="['px-2.5 py-1 text-[10px] font-bold tracking-wider rounded-lg uppercase', alertSeverityMap[alert.severity] || alertSeverityMap.INFO]">
                  {{ alert.severity }}
                </span>
              </td>
              <td class="p-4 font-semibold text-slate-200 text-sm tracking-tight">{{ alert.title }}</td>
              <td class="p-4 text-xs text-slate-400 font-medium">{{ new Date(alert.time).toLocaleString() }}</td>
              <td class="p-4 text-xs text-slate-300">{{ alert.status }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
