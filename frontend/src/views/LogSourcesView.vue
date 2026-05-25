<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api, { endpoints } from '@/api'

const sources = ref<any[]>([])
const loading = ref(true)

const fetchSources = async () => {
  loading.value = true
  try {
    const res = await api.get(endpoints.sources)
    sources.value = res.data || []
  } catch (error) {
    console.error("Failed to fetch log sources", error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchSources()
})

</script>

<template>
  <div class="flex flex-col gap-6 w-full">
    <div class="flex justify-between items-center">
      <div>
        <h1 class="text-2xl font-bold text-slate-100">Log Sources</h1>
        <p class="mt-1 text-sm text-slate-400">Manage and monitor log ingestion sources.</p>
      </div>
      <button @click="fetchSources" class="px-4 py-2 bg-slate-800 text-slate-200 text-sm font-semibold rounded-lg border border-slate-700 hover:bg-slate-700 transition">
        Refresh
      </button>
    </div>

    <!-- Sources Table -->
    <div class="bg-slate-900/20 backdrop-blur-md rounded-2xl border border-slate-800/60 overflow-hidden shadow-[0_12px_40px_rgba(0,0,0,0.2)] w-full"> 
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-slate-900/50 text-xs font-semibold text-slate-400 uppercase tracking-widest border-b border-slate-800/60">
              <th class="p-4">Name</th>
              <th class="p-4">Type</th>
              <th class="p-4">File Path</th>
              <th class="p-4">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="p-4 text-center text-slate-500">Loading sources...</td>
            </tr>
            <tr v-else-if="sources.length === 0">
              <td colspan="4" class="p-4 text-center text-slate-500">No log sources found.</td>
            </tr>
            <tr v-for="source in sources" :key="source.id" class="border-b border-slate-800/40 hover:bg-slate-800/20 transition">
              <td class="p-4 font-semibold text-slate-200 text-sm tracking-tight">{{ source.name }}</td>
              <td class="p-4 text-xs text-slate-400 uppercase">{{ source.type }}</td>
              <td class="p-4 text-xs font-mono text-slate-300">{{ source.file_path }}</td>
              <td class="p-4 text-xs">
                <span v-if="source.status === 'active'" class="bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 px-2 py-1 font-bold rounded uppercase">Active</span>
                <span v-else class="bg-rose-500/20 text-rose-400 border border-rose-500/30 px-2 py-1 font-bold rounded uppercase">{{ source.status || 'Inactive' }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
