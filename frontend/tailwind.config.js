/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Core UI palette (Section 9.1)
        siem: {
          bg: '#0F172A',
          surface: '#1E293B',
          border: '#334155',
          'text-primary': '#F1F5F9',
          'text-secondary': '#94A3B8',
        },
        // Severity colors (Section 9.1)
        severity: {
          critical: '#EF4444',
          high: '#F97316',
          medium: '#EAB308',
          low: '#3B82F6',
          info: '#6B7280',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
    },
  },
  plugins: [],
}
