# hctb-mqtt

Go service that polls the unofficial **Here Comes The Bus** (Synovia) SOAP API and publishes retained JSON to Mosquitto, with optional Home Assistant MQTT discovery.

Unofficial API — HCTB/Synovia can change endpoints without notice. Based on [hcb_soap_client](https://github.com/pcartwright81/hcb_soap_client).

## Topics (default prefix `home/bus`)

| Topic | Contents |
|-------|----------|
| `home/bus/summary` | Account health, roster, last success/error |
| `home/bus/<student>/current` | Active AM/PM route, stop name, vehicle present |
| `home/bus/<student>/vehicle` | Lat/lon, speed, heading, address, ignition, log time |
| `home/bus/<student>/stop` | Scheduled stops for the active route |
| `home/bus/<student>/distance` | Meters/miles to stop (and home if configured), ETA seconds |

`<student>` is a slug from the first name (e.g. `kaylen`).

## Home Assistant

With discovery enabled:

- Bridge device **HCTB MQTT** (`ok`, last update)
- One device per student with `device_tracker`, speed/address/heading/distance/ETA sensors, and ignition / on-map / vehicle-available binary sensors

Gate notifications on `input_boolean.school_day` (and weekday) in automations — the poller itself uses morning/afternoon windows.

## Quick start

```bash
cp .env.example .env
# fill HCTB_SCHOOL_CODE, HCTB_USERNAME, HCTB_PASSWORD, HCTB_TIMEZONE
go test ./...
mise run hctb    # loads .env
```

## Configuration

See [`.env.example`](.env.example).

| Variable | Purpose |
|----------|---------|
| `HCTB_SCHOOL_CODE` | District code from the HCTB app |
| `HCTB_USERNAME` / `HCTB_PASSWORD` | Parent login |
| `HCTB_POLL_INTERVAL_SECONDS` | Active window poll (default 90) |
| `HCTB_IDLE_POLL_INTERVAL_SECONDS` | Outside windows (default 900) |
| `HCTB_TIMEZONE` | IANA TZ for AM/PM and windows |
| `HCTB_MORNING_*` / `HCTB_AFTERNOON_*` | Active windows `HH:MM` |
| `HCTB_HOME_LAT` / `HCTB_HOME_LON` | Optional distance-to-home |
| `HCTB_STUDENT_IDS` | Allowlist (IDs or name slugs); empty = all |

## Layout

```
cmd/hctb/                 entrypoint
internal/hctb/            config, payloads, discovery, publish loop
internal/lib/hctb/        Synovia SOAP client + fixtures
internal/lib/mqttpub/     retained JSON + discovery
internal/lib/env/         env helpers
internal/lib/poll/        SIGINT/SIGTERM + Wait
```

## Docker

```bash
docker build -t hctb-mqtt --build-arg SERVICE=hctb .
# image: ghcr.io/resnostyle/hctb-mqtt:latest
```

Compose example: [`docker-compose.yml`](docker-compose.yml).

## Notes

- Credentials never go in MQTT payloads or info logs.
- Rate-limit / backoff on API failures to avoid account lockouts.
- Addresses are PII — decide whether to retain them in HA history.
