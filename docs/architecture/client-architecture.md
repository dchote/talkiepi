# Client architecture

## Process layout

`cmd/talkiepi` loads config, constructs a `controller.Controller`, starts background PTT / connect loops, and serves HTTP (REST + optional embedded SPA) on `web.host`:`web.port` (default `0.0.0.0:8080`).

```
main
 ├─ config.Load(path) + TALKIEPI_* env
 ├─ update.Checker (optional)
 ├─ led.Indicator (memory or GPIO)
 ├─ controller.New / Start
 │   ├─ ptt.Mux (manual REST + optional GPIO)
 │   ├─ communicator.Communicator
 │   └─ mumbleclient.Client (dial / reconnect)
 └─ rest.NewRouter → http.Server
```

## Controller

`internal/controller` is the façade used by REST and (eventually) MQTT:

- **Status** — connected, channel, ping, transport, mode, transmitting, participants
- **Config** — live `*config.Config`; `ApplyConfig` patches YAML and persists when a path is set
- **Connect / Disconnect** — dials `ServerHostPort()` with TLS options from config
- **Manual PTT** — `ptt.Manual` for REST `/api/v1/ptt`
- **Communicator** — toggle half-duplex state
- **SSE** — `Subscribe` / `Unsubscribe` fan-out for `/api/v1/events`

The controller owns at most one `mumbleclient.Client` at a time.

## Mumble client (`pkg/mumbleclient`)

Built on `go-mumble-server/pkg/mumble`:

| Concern | Module |
|---------|--------|
| TLS dial, handshake, reconnect | `client.go`, `control.go` |
| CryptSetup / UDP encrypt | `udp.go` + `pkg/mumble/crypto` |
| Channel / user roster | `roster.go` |
| Opus voice send/receive | `voice.go` |
| Text messages | `text.go` |
| Typed events | `events.go` |

Timing constants (TCP ping 15s, reconnect backoff 1–30s, UDP ping cadence) match [esp32-mumble](https://github.com/dchote/esp32-mumble). CryptoModes default to lite \| legacy.

## Triggers and LEDs

- **PTT mux** aggregates sources; press/release drives transmit state for `push_to_talk`.
- **GPIO PTT** — Linux `gpiochip` line, configurable active-low ([wiring.md](../hardware/wiring.md)).
- **LEDs** — online, participants (others in channel), transmit; lines default 18 / 23 / 24.

## Voice modes

| Mode | Gate |
|------|------|
| `always_on` | Transmit path open |
| `push_to_talk` | Mux pressed |
| `communicator` | `Communicator.ShouldTransmit()` after open chime; silence window then close chime |

Communicator FSM states: `Idle` → `OpenChime` → `MicActive` → `SilenceWindow` → `CloseChime` → `Idle` (`SilenceTimeout` = 2s).

## Web / API

- chi router: `/health`, `/api/v1/*`, SPA fallback when frontend FS present
- No auth; ingress or LAN trust assumed
- OpenAPI: [`api/openapi.yaml`](../../api/openapi.yaml)

## Related

- [audio-pipeline.md](audio-pipeline.md)
- [ecosystem.md](../ecosystem.md)
- [technical-overview.md](../technical-overview.md)
