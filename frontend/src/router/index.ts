import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

import DashboardView from '@/views/DashboardView.vue'
import AlertsView from '@/views/AlertsView.vue'
import EventsView from '@/views/EventsView.vue'
import LogSourcesView from '@/views/LogSourcesView.vue'
import RulesView from '@/views/RulesView.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Dashboard',
    component: DashboardView,
  },
  {
    path: '/alerts',
    name: 'Alerts',
    component: AlertsView,
  },
  {
    path: '/events',
    name: 'Events',
    component: EventsView,
  },
  {
    path: '/sources',
    name: 'LogSources',
    component: LogSourcesView,
  },
  {
    path: '/rules',
    name: 'Rules',
    component: RulesView,
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

export default router
