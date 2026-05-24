const { ref, reactive } = Vue;

function useSIEM() {
    const totalEvents = ref(0);
    const criticalEvents = ref(0);
    const activeSources = ref(0);
    const recentAlerts = ref([]);
    const loading = ref(false);
    const severityStats = ref([0, 0, 0, 0, 0]); // [Crit, High, Med, Low, Info]
    const eventTimeline = reactive({ labels: [], data: [] });

    const fetchDashboardData = async () => {
        loading.value = true;
        
        // Fetch Alerts
        const alertsRes = await api.get(api.endpoints.alerts);
        if (alertsRes.data) {
            recentAlerts.value = alertsRes.data.map(alert => ({
                severity: alert.severity.toUpperCase(),
                label: alert.rule_name,
                time: alert.triggered_at,
                status: alert.acknowledged ? 'ACK' : 'NEW'
            }));
            
            criticalEvents.value = alertsRes.data.filter(a => a.severity === 'critical').length;

            // Parse for Donut Chart
            const counts = { CRITICAL: 0, HIGH: 0, MEDIUM: 0, LOW: 0, INFO: 0 };
            alertsRes.data.forEach(a => {
                const sev = a.severity.toUpperCase();
                if (counts.hasOwnProperty(sev)) counts[sev]++;
            });
            severityStats.value = [counts.CRITICAL, counts.HIGH, counts.MEDIUM, counts.LOW, counts.INFO];
        }

        // Fetch Sources
        const sourcesRes = await api.get(api.endpoints.sources);
        if (sourcesRes.data) {
            activeSources.value = sourcesRes.data.filter(s => s.status === 'active').length;
        }

        // Fetch Events (Total)
        const eventsRes = await api.get(api.endpoints.events);
        if (eventsRes.data) {
            totalEvents.value = eventsRes.meta?.total || eventsRes.data.length;

            // Parse for Line Chart (Group by 4-hour windows)
            const bins = { "12am": 0, "4am": 0, "8am": 0, "12pm": 0, "4pm": 0, "8pm": 0 };
            eventsRes.data.forEach(ev => {
                const hour = new Date(ev.event_time).getHours();
                if (hour < 4) bins["12am"]++;
                else if (hour < 8) bins["4am"]++;
                else if (hour < 12) bins["8am"]++;
                else if (hour < 16) bins["12pm"]++;
                else if (hour < 20) bins["4pm"]++;
                else bins["8pm"]++;
            });

            eventTimeline.labels = Object.keys(bins);
            eventTimeline.data = Object.values(bins);
        }

        loading.value = false;
    };

    return {
        totalEvents,
        criticalEvents,
        activeSources,
        recentAlerts,
        severityStats,
        eventTimeline,
        loading,
        fetchDashboardData
    };
}