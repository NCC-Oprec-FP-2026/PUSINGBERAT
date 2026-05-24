# PUSINGBERAT SIEM - Live Demo Script

## 1. Rehearsal Flow (8-Minute Choreography)

* **0:00 - 1:00 (Dev B): Introduction & Dashboard Tour.** 
  * Open the live dashboard. Introduce the project: "Welcome to PUSINGBERAT, a lightweight, real-time Security Information and Event Management (SIEM) system." 
  * Show the UI, the real-time event timeline, and the connected log sources.
* **1:00 - 3:00 (Dev A): Backend Architecture Deep Dive.** 
  * (See Dev A's Speaking Part below). Explain the ingest, rules, and WS fan-out.
* **3:00 - 5:00 (Dev A & B): The "Live Hack" (Attack Simulation).**
  * Dev B switches back to the Dashboard, focuses on the Alerts panel.
  * Dev A splits screen to show a terminal. Dev A explains they are a malicious actor.
  * Dev A runs the SSH brute force bash command.
  * *Crowd watches the WebSocket alert pop up instantly on the dashboard.*
  * Dev A runs the Nginx 5xx command. Another alert pops up.
  * Dev B shows the Discord channel receiving the webhooks in real time.
* **5:00 - 7:00 (Dev C): Deployment & Infrastructure.**
  * Dev C explains the VPS setup, Docker Compose networking, Nginx reverse proxy, and the PostgreSQL persistence layer. 
  * Explain how everything is containerized for easy scaling.
* **7:00 - 8:00 (All): Q&A and Outro.**
  * Wrap up, thank the audience, and invite questions.

---

## 2. Developer A's Speaking Part (Backend Architecture)

*(Ensure the README Architecture Diagram is on screen or referenced)*

**"Thanks! Now, let's talk about the engine powering PUSINGBERAT.**

Building a SIEM means we have to handle high-velocity data without blocking. To do this, we designed a reactive, asynchronous backend pipeline written entirely in Go.

It starts at the **Ingest Layer**. Instead of heavily polling files, we use `fsnotify` to hook directly into the operating system's filesystem events. The moment a log is written to `syslog` or Nginx logs, our watcher detects it, reads only the new bytes, and streams it into our parsing engine.

Once parsed, the event hits our **YAML Rule Engine**. We designed this to be completely customizable without recompiling the code. Every event is evaluated against a sliding-window threshold. For example, our SSH Brute Force rule isn't just looking for a single failed login; it's keeping state in memory, grouping by the attacker's IP address, and waiting until it sees exactly 5 failures within a 60-second window.

When a threshold is breached, the rule engine fires an alert into a background Go routine. From there, we do a **Fan-Out broadcast**. The alert is persisted to PostgreSQL for auditing, fired off to Discord via webhooks for our SecOps team, and simultaneously pushed through our WebSocket Hub directly to the Vue frontend you just saw—giving us millisecond-latency alerts. 

It's lightweight, highly concurrent, and designed to catch anomalies in real-time. Let's actually see it in action."

---

## 3. The "Live Hack" Terminal Commands

Keep these commands pre-staged in your terminal or a scratchpad. Be ready to copy-paste them during the demo.

### Trigger 1: SSH Brute Force (5 failed attempts)
*This command uses a `for` loop to write 5 "Failed password" lines to syslog within 1 second, instantly breaching the 5-event/60-second sliding window.*

```bash
for i in {1..5}; do
  echo "$(date '+%b %d %H:%M:%S') pusingberat-vps sshd[$RANDOM]: Failed password for root from 10.0.0.$i" | sudo tee -a /var/log/syslog > /dev/null
done
```

### Trigger 2: Nginx High Error Rate (10 HTTP 5xx errors)
*This command writes 10 HTTP 500 error logs to the Nginx access log, breaching the 10-event/30-second rule threshold.*

```bash
for i in {1..10}; do
  echo "127.0.0.1 - - [$(date +'%d/%b/%Y:%H:%M:%S %z')] \"GET /api/v1/faulty-endpoint HTTP/1.1\" 500 512 \"-\" \"LoadTester/1.0\"" | sudo tee -a /var/log/nginx/access.log > /dev/null
done
```

> **Pro-Tip for the Demo:** Make sure your backend Docker container has the correct volume mounts to read the host's `/var/log/syslog` and `/var/log/nginx/access.log`. If testing locally, you can change the path to your mock log files (e.g. `tmp/syslog.log`).
