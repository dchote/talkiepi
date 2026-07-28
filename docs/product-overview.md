# Product overview

## Vision

talkiepi is a headless Mumble walkie-talkie client for Raspberry Pi (and Linux / macOS), built as a sibling of [go-mumble-server](https://github.com/dchote/go-mumble-server) and [esp32-mumble](https://github.com/dchote/esp32-mumble). It targets always-on intercom and PTT use on a trusted LAN or via Home Assistant ingress—not a full desktop softphone.

## Features

- Mumble client on `go-mumble-server/pkg/mumble` (no gumble / OpenAL)
- Opus encode/decode via libopus; system audio via miniaudio (ALSA / PulseAudio / CoreAudio)
- Device enumeration and selection; SpeexDSP echo cancellation and noise suppression
- Voice modes: always-on, push-to-talk, Communicator (half-duplex with chime)
- Multi-source PTT: GPIO, REST, MQTT, keyboard (when configured)
- Status LEDs (online / participants / transmit) on GPIO
- Embedded Vue 3 + Vuetify 3 web UI and REST API (no auth; trusted LAN / HA ingress)
- MQTT Home Assistant discovery entities aligned with esp32-mumble
- GitHub release update checks
- Debian `.deb` (amd64 + arm64) and Home Assistant add-on

## Voice modes

| Mode | Behavior |
|------|----------|
| `always_on` | Capture path transmits continuously (subject to mute / processing) |
| `push_to_talk` | Transmit only while a PTT source is pressed |
| `communicator` | Half-duplex: toggle opens channel (chime → mic), silence timeout closes (chime); matches esp32-mumble |

## Home Assistant

Three integration layers (see [deployment/homeassistant-addon.md](deployment/homeassistant-addon.md)):

1. **Add-on options** — Supervisor config for server, mode, audio, GPIO
2. **MQTT entities** — Discovery sensors / switches / buttons when Mosquitto is available
3. **Ingress web UI** — Full management UI through the HA sidebar

## Packaging targets

| Target | Notes |
|--------|--------|
| Debian `.deb` | systemd unit, `/etc/talkiepi/talkiepi.yaml`, `/etc/default/talkiepi` |
| Home Assistant add-on | `ghcr.io/dchote/{arch}-addon-talkiepi`, ingress on port 8080 |
| macOS (dev / desktop) | Homebrew Opus + SpeexDSP; CoreAudio; no GPIO |
| Source / `go build` | CGO required for Opus and SpeexDSP |

## Non-goals

- Public internet exposure without a reverse proxy / auth layer
- Replacing Murmur feature-for-feature as a server (use go-mumble-server)
- Running the full ESP32 firmware stack on Pi (use talkiepi instead)
