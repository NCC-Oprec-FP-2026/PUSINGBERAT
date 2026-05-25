<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api, { endpoints } from '@/api'

const rules = ref<any[]>([])
const loading = ref(true)

const fetchRules = async () => {
  loading.value = true
  try {
    const res = await api.get(endpoints.rules)
    rules.value = res.data || []
  } catch (error) {
    console.error("Failed to fetch rules", error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchRules()
})

const alertSeverityMap: Record<string, string> = {
  CRITICAL: 'bg-rose-500/20 text-rose-400 border border-rose-500/30',
  HIGH: 'bg-orange-500/20 text-orange-400 border border-orange-500/30',
  MEDIUM: 'bg-amber-500/20 text-amber-400 border border-amber-500/30',
  LOW: 'bg-blue-500/20 text-blue-400 border border-blue-500/30',
  INFO: 'bg-slate-500/20 text-slate-400 border border-slate-500/30'
}
</script>

<template>
  <div class="flex flex-col gap-6 w-full">
    <div class="flex justify-between items-center">
      <div>
        <h1 class="text-2xl font-bold text-slate-100">Rules</h1>
        <p class="mt-1 text-sm text-slate-400">Manage SIEM detection rules.</p>
      </div>
      <button @click="fetchRules" class="px-4 py-2 bg-slate-800 text-slate-200 text-sm font-semibold rounded-lg border border-slate-700 hover:bg-slate-700 transition">
        Refresh
      </button>
    </div>

    <!-- Rules Table -->
    <div class="bg-slate-900/20 backdrop-blur-md rounded-2xl border border-slate-800/60 overflow-hidden shadow-[0_12px_40px_rgba(0,0,0,0.2)] w-full"> 
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-slate-900/50 text-xs font-semibold text-slate-400 uppercase tracking-widest border-b border-slate-800/60">
              <th class="p-4">Name</th>
              <th class="p-4">Description</th>
              <th class="p-4">Severity</th>
              <th class="p-4">Threshold</th>
              <th class="p-4">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="5" class="p-4 text-center text-slate-500">Loading rules...</td>
            </tr>
            <tr v-else-if="rules.length === 0">
              <td colspan="5" class="p-4 text-center text-slate-500">No rules found.</td>
            </tr>
            <tr v-for="rule in rules" :key="rule.id" class="border-b border-slate-800/40 hover:bg-slate-800/20 transition">
              <td class="p-4 font-semibold text-slate-200 text-sm tracking-tight">{{ rule.name }}</td>
              <td class="p-4 text-xs text-slate-400 max-w-md truncate">{{ rule.description }}</td>
              <td class="p-4">
                <span :class="['px-2.5 py-1 text-[10px] font-bold tracking-wider rounded-lg uppercase', alertSeverityMap[(rule.severity || '').toUpperCase()] || alertSeverityMap.INFO]">
                  {{ rule.severity }}
                </span>
              </td>
              <td class="p-4 text-xs font-mono text-slate-300">{{ rule.threshold || 'N/A' }} / {{ rule.window_duration || 'N/A' }}s</td>
              <td class="p-4 text-xs">
                <span v-if="rule.enabled !== false" class="text-emerald-400 font-bold uppercase">Enabled</span>
                <span v-else class="text-rose-400 font-bold uppercase">Disabled</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
