# Technical overview

## Stack

| Layer | Technology |
|-------|------------|
| Language | Go **1.25** (`go.mod` may pin a patch such as 1.25.5) |
| Protocol | `github.com/dchote/go-mumble-server` **v0.2.0** → `pkg/mumble` |
| Client glue | `pkg/mumbleclient` (control, UDP, roster, voice) |
| Codec | libopus via CGO (`gopkg.in/hraban/opus.v2`) |
| Capture / playback | miniaudio (ALSA, PulseAudio, JACK, CoreAudio) |
| AEC / NS | SpeexDSP (optional WebRTC stub); `internal/audio/process` |
| HTTP | chi v5 REST + SSE; optional embedded SPA |
| Frontend | Vue 3, Vuetify 3, Vite, Yarn |
| GPIO | `warthog618/go-gpiocdev` (Linux) |
| Config | YAML (`goccy/go-yaml`) + `TALKIEPI_*` env + `/etc/default/talkiepi` |

## Architecture (high level)

```
┌─────────────────────────────────────────────────────────────┐
│ cmd/talkiepi                                                │
│  config.Load → controller → rest.NewRouter → http.Server    │
└────────────┬───────────────────────────┬────────────────────┘
             │                           │
             ▼                           ▼
   pkg/mumbleclient              frontend (embed_frontend)
   (TLS + UDP + roster)          Vue SPA on same port
             │
             ▼
   go-mumble-server/pkg/mumble
```

Runtime packages:

| Package | Responsibility |
|---------|----------------|
| `internal/config` | Defaults, YAML load/save, env overlay, validation |
| `internal/controller` | Connect lifecycle, PTT mux, status, SSE fan-out |
| `internal/communicator` | Half-duplex Communicator FSM |
| `internal/ptt` | Manual / GPIO / mux triggers |
| `internal/led` | Online / participants / transmit indicators |
| `internal/audio` | Device descriptors, resolve, ring buffers |
| `internal/audio/process` | Speex / none / webrtc stub processors |
| `internal/codec` | Opus profiles (48 kHz default, 16 kHz Esp32) |
| `internal/vad` | Energy VAD (esp32-mumble constants) |
| `internal/jitter` | PCM jitter buffer |
| `internal/hass` | MQTT discovery entity definitions |
| `internal/update` | GitHub release checker |
| `internal/rest` | `/health`, `/api/v1/*`, SPA |

See [architecture/client-architecture.md](architecture/client-architecture.md) and [architecture/audio-pipeline.md](architecture/audio-pipeline.md).

## `pkg/mumble` dependency

```text
require github.com/dchote/go-mumble-server v0.2.0
```

talkiepi does **not** vendor a separate protocol implementation. Version, Authenticate, CryptSetup, ServerSync, UserState, ChannelState, TextMessage, and voice packet layout come from the shared library. Client-only timing and crypto advertisement live in `pkg/mumbleclient` (see [protocol/handshake.md](protocol/handshake.md)).

## Configuration

Precedence (later wins for env overlays after YAML load):

1. Built-in defaults (`config.Defaults()`)
2. YAML file (`-config`, typically `/etc/talkiepi/talkiepi.yaml` or `/data/talkiepi.yaml` in the add-on)
3. Environment variables `TALKIEPI_*` (including values from `/etc/default/talkiepi` via systemd `EnvironmentFile`)

Service-level knobs for Debian live in [`configs/talkiepi.default`](../configs/talkiepi.default) → `/etc/default/talkiepi` (`TALKIEPI_CONFIG`, `TALKIEPI_WEB_*`, `TALKIEPI_STATE_DIR`, `TALKIEPI_ARGS`). Structured settings stay in YAML.

Example tree: [`configs/talkiepi.yaml`](../configs/talkiepi.yaml).

## Embedded frontend

- Dev: `make run` serves API only (`-frontend-embed=false`); Vite on a separate port.
- Release / add-on: `yarn build` → copy `frontend/dist` to `cmd/talkiepi/frontend-dist` → build with `-tags embed_frontend`.
- Never use `embed_frontend` unless `frontend-dist` is non-empty (`SKIP_FRONTEND=true` for binary-only builds).

## Audio pipeline (summary)

Capture → gain → optional Speex AEC/NS (far-end from playback) → VAD / mode gate → Opus encode → UDP (or TCP tunnel).

Receive → Opus decode → jitter buffer → playback; far-end samples fed to AEC as render reference.

Default voice sample rate is **48 kHz**; Esp32-compatible profile is **16 kHz**. Details: [audio-pipeline.md](architecture/audio-pipeline.md).

## Security model

The REST API and UI have **no authentication**. Deploy on a trusted network, behind HA ingress, or a reverse proxy that supplies auth. TLS to the Mumble server is required; `insecure_tls` skips certificate verification for lab servers.
