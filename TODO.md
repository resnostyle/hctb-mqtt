# hctb-mqtt — todo list

Poll **Here Comes The Bus**, publish retained MQTT + HA discovery.

## Done (v1 service)

- [x] Module / cmd / Docker / CI / README identity as `hctb-mqtt`
- [x] Synovia SOAP client (`internal/lib/hctb`) + fixture tests
- [x] Config: credentials, poll/idle intervals, timezone, AM/PM windows, home lat/lon, student allowlist
- [x] MQTT topics under `home/bus` (summary + per-student current/vehicle/stop/distance)
- [x] HA discovery: one device per student + bridge OK/last-update
- [x] Poll loop with session reuse, adaptive interval, backoff, graceful shutdown
- [x] GHCR image wiring (`ghcr.io/resnostyle/hctb-mqtt`)

## Remaining

### Local smoke

- [ ] Local `.env` with real credentials (gitignored)
- [ ] `docker compose` smoke against local Mosquitto
- [ ] Confirm distroless image has outbound HTTPS for SOAP

### HA home integration

- [ ] Confirm MQTT discovery entities for each kid
- [ ] Map card with bus device trackers
- [ ] Automation: bus approaching (distance / ETA threshold) → notify / Awtrix
- [ ] Gate on `input_boolean.school_day` (and weekday) so weekends stay quiet
- [ ] Morning vs afternoon notification copy
- [ ] Optional: combine with Kaylen/Maya school presence automations

### Hardening

- [ ] Decide log retention / whether to publish raw addresses (PII)
- [ ] Archive or trim this checklist when v1 is deployed
