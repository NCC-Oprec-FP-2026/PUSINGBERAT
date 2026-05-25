<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api, { endpoints } from '@/api'

const events = ref<any[]>([])
const loading = ref(true)

const fetchEvents = async () => {
  loading.value = true
  try {
    const res = await api.get(endpoints.events)
    events.value = res.data?.data || res.data || []
  } catch (error) {
    console.error("Failed to fetch events", error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchEvents()
})

</script>

<template>
  <div class="flex flex-col gap-6 w-full">
    <div class="flex justify-between items-center">
      <div>
        <h1 class="text-2xl font-bold text-slate-100">Events</h1>
        <p class="mt-1 text-sm text-slate-400">View raw security events ingested by the system.</p>
      </div>
      <button @click="fetchEvents" class="px-4 py-2 bg-slate-800 text-slate-200 text-sm font-semibold rounded-lg border border-slate-700 hover:bg-slate-700 transition">
        Refresh
      </button>
    </div>

    <!-- Events Table -->
    <div class="bg-slate-900/20 backdrop-blur-md rounded-2xl border border-slate-800/60 overflow-hidden shadow-[0_12px_40px_rgba(0,0,0,0.2)] w-full"> 
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-slate-900/50 text-xs font-semibold text-slate-400 uppercase tracking-widest border-b border-slate-800/60">
              <th class="p-4">Time</th>
              <th class="p-4">Source ID</th>
              <th class="p-4">Raw Data</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="3" class="p-4 text-center text-slate-500">Loading events...</td>
            </tr>
            <tr v-else-if="events.length === 0">
              <td colspan="3" class="p-4 text-center text-slate-500">No events found.</td>
            </tr>
            <tr v-for="event in events.slice(0, 100)" :key="event.id" class="border-b border-slate-800/40 hover:bg-slate-800/20 transition">
              <td class="p-4 text-xs text-slate-400 font-medium whitespace-nowrap">{{ new Date(event.event_time).toLocaleString() }}</td>
              <td class="p-4 text-xs font-mono text-slate-300">{{ event.source_id }}</td>
              <td class="p-4 text-xs font-mono text-emerald-400 truncate max-w-xl">{{ event.raw_line }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
