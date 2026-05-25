import axios from 'axios'

const API_HOST = window.location.hostname || 'localhost'
const BASE_URL = import.meta.env.VITE_API_URL || `http://${API_HOST}:8080/api/v1`

const api = axios.create({
  baseURL: BASE_URL,
  headers: {
    'Content-Type': 'application/json'
  }
})

export const endpoints = {
  severity_count: '/stats/alerts/by-severity',
  stats: '/stats/overview',
  eventsTimeline: '/stats/events/timeline',
  alerts: '/alerts',
  events: '/events',
  sources: '/sources',
  rules: '/rules' // Assuming rules endpoint exists based on HTML template
}

export default api
