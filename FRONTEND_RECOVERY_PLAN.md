# PUSINGBERAT Frontend Recovery Plan

## 1. Executive Summary
The goal is to unify the scattered frontend logic injected into `/frontend-server/templates/*.html` and `/frontend-server/static/` back into the structured Vue 3 SPA architecture located in `/frontend/src/`. All UI features will be rebuilt utilizing strict Vite + Tailwind styling, ignoring the massive legacy CSS files, and adhering strictly to a dark-themed, minimalist, modern card-based aesthetic.

## 2. API & Data Fetching Migration
The friend's implementation used a custom `api` object built on top of the native `fetch` API. This logic will be migrated to the standard Axios setup already configured in `package.json`.

**Tasks:**
- [x] **Axios Configuration**: Create or update `src/api/index.ts` to instantiate a configured Axios client pointing to `/api/v1`.
- [x] **Endpoints to Map**:
  - `/alerts` (GET lists, PATCH to acknowledge, DELETE to remove)
  - `/events` (GET lists)
  - `/sources` (GET lists)
  - `/stats/alerts/by-severity` (GET for Donut Chart)
  - `/stats/events/timeline` (GET for Line Chart)
- [x] **Composables**: Port the data fetching logic found in `frontend-server/templates/composables.js` (`useSIEM` hook) into a Pinia store (`src/stores/`) or Vue composable (`src/composables/useDashboard.ts`) for centralized state management.

## 3. Real-Time Logic (WebSockets)
**Current Implementation**: In `frontend-server/templates/index.html` (lines 534-614), a raw WebSocket client is initialized. It intercepts incoming `alert` payloads, dynamically unshifts them into a `recent_alerts` list, increments `critical_events`, and triggers forced updates on the Chart.js instances.

**Migration Plan**:
- [x] Enhance the existing `src/composables/useAlertSocket.ts`. Currently, it only handles pushing short-lived UI toasts.
- [x] Refactor the socket listener to dispatch events to a Pinia store or a reactive global state.
- [x] Ensure that the Dashboard views and Charts are reactively bound to this state so they automatically update when new alerts arrive, eliminating the need for manual Chart instance updates.

## 4. Components & Views Porting
The friend's HTML templates relied heavily on PrimeVue components via CDN (`p-dialog`, `p-datatable`, `p-card`, etc.). Since `package.json` specifies Tailwind and no PrimeVue, these must be rebuilt natively.

- [x] **DashboardView.vue**:
  - Replicate the layout defined in `index.html`.
  - Build 4 Tailwind-styled Metric Cards (Total Events, Critical Alerts, Active Sources, Log Resources Status).
  - Implement the **Donut Chart** (Severity) and **Line Chart** (Event Timeline) using `vue-chartjs` and `chart.js`.
  - Rebuild the "Recent Alerts" table using native HTML/Tailwind.
- [x] **AlertsView.vue**:
  - Extract the `p-dialog` alert details modal from `index.html` and the table from `alerts.html`.
  - Recreate the modal with standard Tailwind overlays and backdrop blur.
  - Port over the logic for **Deleting** an alert and **Acknowledging** an alert.
- [x] **EventsView.vue, RulesView.vue, LogSourcesView.vue**:
  - Migrate tabular data displays and CRUD logic from their respective Flask `.html` templates into pure Vue components.

## 5. Styling Strategy & Constraints
- [x] **Discard Legacy CSS**: The massive CSS files located in `frontend-server/static/css/*` will be completely ignored.
- [x] **Aesthetic Enforcement**: Rely strictly on Tailwind CSS utility classes. The design must be modern, card-based, minimalist, and dark-themed (e.g., using `bg-slate-900`, `border-slate-800`, `text-slate-200`, etc.).
- [x] Drop all PrimeVue dependencies and stick strictly to the Vue + Tailwind stack.
