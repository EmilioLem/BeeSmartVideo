package webPageStats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Stats represents the metrics shown on the dashboard
type Stats struct {
	ActiveBees    int     `json:"activeBees"`
	CountUp       int     `json:"countUp"`
	CountDown     int     `json:"countDown"`
	AvgPathLength float64 `json:"avgPathLength"`
	AvgSpeed      float64 `json:"avgSpeed"`
	AvgDistance   float64 `json:"avgDistance"`
	FPS           float64 `json:"fps"`
}

var (
	currentStats Stats
	statsMu      sync.RWMutex
)

// UpdateStats updates the current global stats
func UpdateStats(active, up, down int, avgPath, avgSpeed, avgDist, fps float64) {
	statsMu.Lock()
	defer statsMu.Unlock()
	currentStats = Stats{
		ActiveBees:    active,
		CountUp:       up,
		CountDown:     down,
		AvgPathLength: avgPath,
		AvgSpeed:      avgSpeed,
		AvgDistance:   avgDist,
		FPS:           fps,
	}
}

// StartServer starts the HTTP server on the specified port
func StartServer(port int) {
	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		statsMu.RLock()
		defer statsMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentStats)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, dashboardHTML)
	})

	go func() {
		fmt.Printf("Dashboard available at http://localhost:%d\n", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			fmt.Printf("Failed to start stats server: %v\n", err)
		}
	}()
}

const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BeeSmart Live Dashboard</title>
    <style>
        :root {
            --primary: #f1c40f;
            --bg: #0f0f0f;
            --card-bg: rgba(255, 255, 255, 0.05);
            --text: #ffffff;
            --accent: #e67e22;
        }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background-color: var(--bg);
            color: var(--text);
            margin: 0;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            overflow: hidden;
        }

        .container {
            width: 90%;
            max-width: 1000px;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 40px;
            background: var(--card-bg);
            backdrop-filter: blur(10px);
            border-radius: 24px;
            border: 1px solid rgba(255, 255, 255, 0.1);
            box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.37);
        }

        .flow-container {
            grid-column: span 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            background: rgba(255, 255, 255, 0.03);
            border-radius: 16px;
            padding: 20px;
            border: 1px solid rgba(255, 255, 255, 0.05);
        }

        .flow-viz {
            width: 60px;
            height: 200px;
            background: rgba(255,255,255,0.05);
            border-radius: 30px;
            display: flex;
            flex-direction: column;
            overflow: hidden;
            position: relative;
        }

        .flow-up {
            background: linear-gradient(to bottom, #2ecc71, #27ae60);
            width: 100%;
            transition: height 0.5s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.5rem;
        }

        .flow-down {
            background: linear-gradient(to bottom, #e74c3c, #c0392b);
            width: 100%;
            transition: height 0.5s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.5rem;
        }

        .arrow {
            color: white;
            text-shadow: 0 2px 4px rgba(0,0,0,0.3);
        }

        header {
            margin-bottom: 40px;
            text-align: center;
        }

        h1 {
            font-size: 2.5rem;
            margin: 0;
            background: linear-gradient(45deg, var(--primary), var(--accent));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            letter-spacing: -1px;
        }

        .stat-card {
            background: rgba(255, 255, 255, 0.03);
            padding: 24px;
            border-radius: 16px;
            text-align: center;
            transition: transform 0.3s ease, background 0.3s ease;
            border: 1px solid rgba(255, 255, 255, 0.05);
        }

        .stat-card:hover {
            transform: translateY(-5px);
            background: rgba(255, 255, 255, 0.07);
        }

        .stat-value {
            font-size: 2.5rem;
            font-weight: 800;
            margin: 10px 0;
            color: var(--primary);
        }

        .stat-label {
            font-size: 0.9rem;
            text-transform: uppercase;
            letter-spacing: 1px;
            color: rgba(255, 255, 255, 0.5);
        }

        .live-indicator {
            display: inline-block;
            width: 10px;
            height: 10px;
            background-color: #2ecc71;
            border-radius: 50%;
            margin-right: 8px;
            box-shadow: 0 0 10px #2ecc71;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0% { opacity: 1; }
            50% { opacity: 0.4; }
            100% { opacity: 1; }
        }

        .footer {
            margin-top: 40px;
            font-size: 0.8rem;
            color: rgba(255, 255, 255, 0.3);
        }
    </style>
</head>
<body>
    <header>
        <h1>BeeSmart Live</h1>
        <div style="font-size: 0.9rem; color: rgba(255,255,255,0.6); margin-top: 10px;">
            <span class="live-indicator"></span> Real-time monitoring active
        </div>
    </header>

    <div class="container">
        <div class="flow-container">
            <div class="stat-label" style="margin-bottom: 15px;">Flow Proportion</div>
            <div class="flow-viz">
                <div id="flowUp" class="flow-up"><span class="arrow">↑</span></div>
                <div id="flowDown" class="flow-down"><span class="arrow">↓</span></div>
            </div>
            <div style="display: flex; justify-content: space-between; width: 100%; margin-top: 15px; font-size: 0.8rem;">
                <span style="color: #2ecc71">UP</span>
                <span style="color: #e74c3c">DOWN</span>
            </div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Active Bees</div>
            <div id="activeBees" class="stat-value">0</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Avg Speed</div>
            <div id="avgSpeed" class="stat-value">0.0</div>
            <div class="stat-label">px/frame</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Avg Distance</div>
            <div id="avgDist" class="stat-value">0.0</div>
            <div class="stat-label">pixels</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Avg Path Length</div>
            <div id="avgPath" class="stat-value">0.0</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Count Up</div>
            <div id="countUp" class="stat-value">0</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Count Down</div>
            <div id="countDown" class="stat-value">0</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Performance</div>
            <div id="fps" class="stat-value">0.0</div>
            <div class="stat-label">FPS</div>
        </div>
    </div>

    <div class="footer">
        BeeSmartVideo v2.0 &bull; Built with precision
    </div>

    <script>
        function updateStats() {
            fetch('/api/stats')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('activeBees').textContent = data.activeBees;
                    document.getElementById('countUp').textContent = data.countUp;
                    document.getElementById('countDown').textContent = data.countDown;
                    document.getElementById('avgPath').textContent = data.avgPathLength.toFixed(1);
                    document.getElementById('avgSpeed').textContent = data.avgSpeed.toFixed(1);
                    document.getElementById('avgDist').textContent = data.avgDistance.toFixed(1);
                    document.getElementById('fps').textContent = data.fps.toFixed(1);

                    // Update Flow Visualization
                    const total = data.countUp + data.countDown;
                    if (total > 0) {
                        const upPerc = (data.countUp / total) * 100;
                        const downPerc = (data.countDown / total) * 100;
                        document.getElementById('flowUp').style.height = upPerc + '%';
                        document.getElementById('flowDown').style.height = downPerc + '%';
                    } else {
                        document.getElementById('flowUp').style.height = '50%';
                        document.getElementById('flowDown').style.height = '50%';
                    }
                })
                .catch(err => console.error('Error fetching stats:', err));
        }

        // Update every 200ms for smooth live feel
        setInterval(updateStats, 200);
        updateStats();
    </script>
</body>
</html>
`
