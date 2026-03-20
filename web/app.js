function dashboard() {
    return {
        tabs: ['Pipeline', 'Agents', 'Activity', 'Escalations', 'Cost', 'Logs'],
        activeTab: 0,
        version: '0.1.0',
        connected: false,
        lastRefresh: null,

        // Data
        requirements: [],
        stories: [],
        agents: [],
        events: [],
        escalations: [],
        dailyCost: 0,
        dailyLimit: 50,
        costPct: 0,
        costData: null,
        logLines: [],
        waveStatus: '',

        async init() {
            await this.fetchAll();
            this.connectSSE();
            setInterval(() => this.fetchAll(), 5000);
        },

        async fetchAll() {
            try {
                const [reqs, stories, agents, events, cost] = await Promise.all([
                    fetch('/api/requirements').then(r => r.json()),
                    fetch('/api/stories').then(r => r.json()),
                    fetch('/api/agents').then(r => r.json()),
                    fetch('/api/events?limit=50').then(r => r.json()),
                    fetch('/api/cost').then(r => r.json()),
                ]);
                this.requirements = reqs || [];
                this.stories = stories || [];
                this.agents = agents || [];
                this.events = events || [];
                if (cost) {
                    this.costData = cost;
                    this.dailyCost = cost.daily_total || 0;
                    this.dailyLimit = cost.daily_limit || 50;
                    this.costPct = this.dailyLimit > 0 ? (this.dailyCost / this.dailyLimit) * 100 : 0;
                }
                this.updateWaveStatus();
                this.lastRefresh = new Date().toLocaleTimeString();
            } catch (e) {
                console.error('Fetch error:', e);
            }
        },

        connectSSE() {
            const source = new EventSource('/api/stream');
            source.onopen = () => { this.connected = true; };
            source.onmessage = (e) => {
                try {
                    const msg = JSON.parse(e.data);
                    // Refresh data on any event
                    this.fetchAll();
                } catch (err) {}
            };
            source.onerror = () => {
                this.connected = false;
                // Auto-reconnect after 3 seconds
                setTimeout(() => this.connectSSE(), 3000);
            };
        },

        storiesByStatus(status) {
            const statusMap = {
                'planned': ['draft', 'estimated', 'planned'],
                'assigned': ['assigned'],
                'in_progress': ['in_progress'],
                'review': ['review'],
                'qa': ['qa', 'qa_failed'],
                'merged': ['pr_submitted', 'merged'],
            };
            const validStatuses = statusMap[status] || [status];
            return this.stories.filter(s => validStatuses.includes(s.Status));
        },

        statusLabel(status) {
            const labels = {
                'planned': 'PLANNED',
                'assigned': 'ASSIGNED',
                'in_progress': 'IN PROGRESS',
                'review': 'REVIEW',
                'qa': 'QA',
                'merged': 'MERGED',
            };
            return labels[status] || status.toUpperCase();
        },

        statusColor(status) {
            const colors = {
                'planned': 'text-gray-400',
                'assigned': 'text-blue-400',
                'in_progress': 'text-yellow-400',
                'review': 'text-cyan-400',
                'qa': 'text-purple-400',
                'merged': 'text-green-400',
            };
            return colors[status] || 'text-gray-400';
        },

        agentStatusColor(status) {
            const colors = {
                'active': 'text-green-400',
                'idle': 'text-gray-400',
                'stuck': 'text-red-400',
                'terminated': 'text-gray-500',
            };
            return colors[status] || 'text-gray-400';
        },

        updateWaveStatus() {
            const merged = this.stories.filter(s => s.Status === 'merged' || s.Status === 'pr_submitted').length;
            const total = this.stories.length;
            if (total > 0) {
                this.waveStatus = `${merged}/${total} stories merged`;
            } else {
                this.waveStatus = 'No stories';
            }
        },
    };
}
