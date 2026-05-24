const { createApp, onMounted, ref } = Vue;
const { useTimeAgo } = VueUse;

const DashboardApp = {
    setup() {
        const { 
            totalEvents, 
            criticalEvents, 
            activeSources, 
            recentAlerts, 
            severityStats,
            eventTimeline,
            fetchDashboardData 
        } = useSIEM();

        const log_sources = ref(true);
        const donutCanvas = ref(null);
        const lineCanvas = ref(null);

        const alert_severity_map = {
            'CRITICAL': 'error',
            'HIGH': 'warn',
            'MEDIUM': 'info',
            'LOW': 'secondary'
        };

        const donut_data = ref({
            labels: ['Critical', 'High', 'Medium', 'Low', 'Info'],
            datasets: [{
                data: [0, 0, 0, 0, 0],
                backgroundColor: ['#EF4444', '#F97316', '#EAB308', '#3B82F6', '#6B7280']
            }]
        });

        const line_data = ref({
            labels: ["12am", "4am", "8am", "12pm", "4pm", "8pm"],
            datasets: [{
                label: "Events",
                data: [65, 59, 80, 81, 56, 55],
                fill: false,
                borderColor: 'rgb(75, 192, 192)',
                tension: 0.1
            }]
        });

        onMounted(async () => {
            await fetchDashboardData();

            // Initialize Donut Chart with Parsed Data
            if (donutCanvas.value) {
                const donutChart = new Chart(donutCanvas.value.getContext('2d'), {
                    type: 'doughnut',
                    data: {
                        labels: ['Critical', 'High', 'Medium', 'Low', 'Info'],
                        datasets: [{
                            data: severityStats.value,
                            backgroundColor: ['#EF4444', '#F97316', '#EAB308', '#3B82F6', '#6B7280']
                        }]
                    },
                    options: { maintainAspectRatio: false }
                });
            }

            // Initialize Line Chart with Parsed Data
            if (lineCanvas.value) {
                new Chart(lineCanvas.value.getContext('2d'), {
                    type: 'line',
                    data: {
                        labels: eventTimeline.labels,
                        datasets: [{
                            label: "Events",
                            data: eventTimeline.data,
                            fill: false,
                            borderColor: 'rgb(75, 192, 192)',
                            tension: 0.1
                        }]
                    }
                });
            }
        });

        return {
            total_events: totalEvents,
            critical_events: criticalEvents,
            active_sources: activeSources,
            recent_alerts: recentAlerts,
            log_sources,
            donutCanvas,
            lineCanvas,
            donut_data,
            line_data,
            alert_severity_map,
            useTimeAgo
        };
    }
};

// This allows linking in HTML: <script src="/statics/js/dashboard.js"></script>
window.DashboardApp = DashboardApp;