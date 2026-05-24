# Day 11: Demo Day (Polish + Rehearsal)

## [ ] 09:00 AM Final Deployment Checklist
- [ ] SSH into the production VPS.
- [ ] Pull the latest Docker images or git commit (`git pull origin main`).
- [ ] Stop existing containers and prune networks: `docker compose down`.
- [ ] Rebuild and start the stack detached: `docker compose up -d --build`.
- [ ] Verify all containers are running and healthy: `docker compose ps`.
- [ ] Check backend and frontend container logs for startup errors: `docker compose logs -f backend frontend`.

## [ ] 09:30 AM End-to-End Verification
- [ ] Verify live production URL is accessible via HTTPS/HTTP.
- [ ] Check Nginx routing: Ensure frontend loads and API endpoints (`/api/v1/...`) respond correctly.
- [ ] Verify WebSocket connection: Open browser DevTools (Network -> WS) and ensure connection to `/ws` is `101 Switching Protocols` and stays open.
- [ ] Test Database Ingest: Manually append a harmless log to `/var/log/syslog` and verify it appears in the dashboard's event stream.
- [ ] Test Discord Webhook: Trigger a low-severity alert and confirm the message drops in the Discord `#alerts` channel.

## **10:00 AM CODE FREEZE**
**WARNING: NO CODE CHANGES ARE ALLOWED AFTER THIS POINT. The codebase is locked. Do not push any commits to `main`. Focus purely on rehearsing the script and testing the live environment.**
