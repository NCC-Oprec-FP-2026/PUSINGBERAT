<script setup lang="ts">
import { useRoute, RouterLink } from 'vue-router'
import {
  LayoutDashboard,
  Bell,
  Activity,
  HardDrive,
  Shield,
} from 'lucide-vue-next'

interface NavItem {
  name: string
  path: string
  icon: typeof LayoutDashboard
}

const route = useRoute()

const navItems: NavItem[] = [
  { name: 'Dashboard', path: '/', icon: LayoutDashboard },
  { name: 'Alerts', path: '/alerts', icon: Bell },
  { name: 'Events', path: '/events', icon: Activity },
  { name: 'Log Sources', path: '/sources', icon: HardDrive },
  { name: 'Rules', path: '/rules', icon: Shield },
]

const isActive = (path: string): boolean => {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}
</script>

<template>
  <aside
    class="fixed left-0 top-0 z-40 flex h-screen w-60 flex-col border-r border-siem-border bg-siem-surface"
  >
    <!-- Logo / Brand -->
    <div class="flex h-16 items-center gap-3 border-b border-siem-border px-5">
      <div
        class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-cyan-400"
      >
        <Shield class="h-4 w-4 text-white" :stroke-width="2.5" />
      </div>
      <div>
        <span class="text-sm font-bold tracking-wide text-siem-text-primary">PUSINGBERAT</span>
        <p class="text-[10px] font-medium uppercase tracking-widest text-siem-text-secondary">
          SIEM Platform
        </p>
      </div>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 space-y-1 px-3 py-4">
      <RouterLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        :class="[
          'group flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200',
          isActive(item.path)
            ? 'bg-blue-500/10 text-blue-400'
            : 'text-siem-text-secondary hover:bg-white/5 hover:text-siem-text-primary',
        ]"
      >
        <component
          :is="item.icon"
          :class="[
            'h-5 w-5 flex-shrink-0 transition-colors duration-200',
            isActive(item.path)
              ? 'text-blue-400'
              : 'text-siem-text-secondary group-hover:text-siem-text-primary',
          ]"
          :stroke-width="1.75"
        />
        <span>{{ item.name }}</span>

        <!-- Active indicator bar -->
        <div
          v-if="isActive(item.path)"
          class="ml-auto h-1.5 w-1.5 rounded-full bg-blue-400"
        />
      </RouterLink>
    </nav>

    <!-- Footer -->
    <div class="border-t border-siem-border px-5 py-4">
      <p class="text-[11px] text-siem-text-secondary">
        &copy; 2026 PUSINGBERAT
      </p>
    </div>
  </aside>
</template>
