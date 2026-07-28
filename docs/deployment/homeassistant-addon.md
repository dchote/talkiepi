# Home Assistant add-on

talkiepi ships as a Supervisor add-on (`addon/`) with images `ghcr.io/dchote/{arch}-addon-talkiepi`. It is designed for **parity with [esp32-mumble](https://github.com/dchote/esp32-mumble)** config naming and MQTT discovery, on Pi-class hardware with PulseAudio and optional GPIO.

Install notes: [addon/README.md](../../addon/README.md).

## Three layers

```mermaid
flowchart TB
  OPT[1. Add-on options<br/>Supervisor UI]
  MQTT[2. MQTT entities<br/>HA discovery]
  UI[3. Ingress web UI<br/>REST + Vue SPA]
  YAML[/data/talkiepi.yaml]
  BIN[talkiepi process]

  OPT -->|env on start| BIN
  OPT -->|seed / force| YAML
  YAML --> BIN
  BIN --> MQTT
  BIN --> UI
```

### 1. Add-on options

Defined in [`addon/config.yaml`](../../addon/config.yaml): server, port, username, password, channel, mode, crypto, speaker volume, mic gain, audio device names, echo/NS toggles, PTT GPIO, web port, log level, `force_options`.

[`addon/rootfs/etc/services.d/talkiepi/run`](../../addon/rootfs/etc/services.d/talkiepi/run) exports these as `TALKIEPI_*` and seeds `/data/talkiepi.yaml` on first run.

### 2. MQTT entities

When the Mosquitto service is available, the run script enables MQTT + HA discovery. Entity set from `internal/hass` mirrors esp32-mumble:

- Config texts: `1. Server` … `5. Channel`
- Selects: `6. Mode`, `7. Crypto`
- Number: speaker volume
- Switch: mute
- Buttons: PTT, Communicator toggle, reconnect, reset config
- Sensors: connected, voice received/sending, ping, transport, channel, participants, ERLE

Topics under `talkiepi/<node>/…` with discovery prefix `homeassistant` (configurable).

### 3. Ingress web UI

`ingress: true`, `ingress_port: 8080`. **Open Web UI** in the add-on panel reaches the same SPA/API as a bare metal install, through Supervisor ingress (no extra port publish required).

## Config precedence

| Priority | Source | Notes |
|----------|--------|--------|
| Highest | Process env `TALKIEPI_*` | Set every start from Supervisor options (+ MQTT broker creds) |
| | `/data/talkiepi.yaml` | Persisted; patched by UI / API |
| Lowest | Built-in defaults | Used when seeding YAML |

**`force_options: true`** deletes `/data/talkiepi.yaml` before start so options fully re-seed the file (Supervisor options win over stale UI edits).

With `force_options: false` (default), YAML persists across restarts; env still overlays matching keys on load via `config.applyEnv()`.

## Parity with esp32-mumble

| Area | Alignment |
|------|-----------|
| Mode names | `always_on`, `push_to_talk`, `communicator` |
| Crypto | `legacy`, `lite` |
| Numbered HA config entities | Same labels / ordering |
| Communicator / PTT buttons | Same MQTT button semantics |
| Voice constants | Shared VAD / timing philosophy (Pi uses 48 kHz default vs ESP 16 kHz) |

## Images

Workflow [`.github/workflows/addon.yml`](../../.github/workflows/addon.yml) builds amd64 and aarch64 and pushes to GHCR. Keep `addon/config.yaml` `version` aligned with the image tag (without `v`). Packages must be **public** for Supervisor pulls.
