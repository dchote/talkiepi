# 0001 — Next-generation rewrite

| Field | Value |
|-------|--------|
| **Status** | **Implemented** |
| **Related** | [ecosystem.md](../ecosystem.md), [technical-overview.md](../technical-overview.md) |

## Summary

talkiepi was rewritten as a **sibling client** in the go-mumble-server / esp32-mumble family—not as a maintenance fork of barnard, gumble, or OpenAL.

### Replaced

- gumble-based control plane → `github.com/dchote/go-mumble-server/pkg/mumble` **v0.2.0** via `pkg/mumbleclient`
- OpenAL playback/capture → miniaudio (ALSA / Pulse / CoreAudio)
- Ad-hoc Pi-only wiring → structured YAML + env, REST, optional MQTT/HA discovery

### Added

- Vue 3 + Vuetify embedded management UI (shared style rules with go-mumble-server)
- SpeexDSP echo cancellation / noise suppression pipeline
- Communicator mode and VAD defaults aligned with esp32-mumble
- Opus profiles: 48 kHz default and 16 kHz Esp32-compatible
- Debian packaging (goreleaser-cross) and Home Assistant add-on with ingress
- Multi-source PTT (GPIO, REST, MQTT) and status LEDs
- GitHub Actions CI / release / add-on image workflows

### Non-goals of the rewrite

- Preserving barnard CLI flags or OpenAL device strings
- Shipping a Mumble server (use go-mumble-server)
- Replacing esp32-mumble firmware (complementary hardware client)

## Outcome

A single Go module (`github.com/dchote/talkiepi`) that builds headless Pi/desktop clients and an HA add-on, speaking the same protocol library and operational conventions as the rest of the family.
