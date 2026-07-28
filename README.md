# talkiepi

Headless Mumble walkie-talkie client for Raspberry Pi (and macOS), with an embedded web UI, GPIO/REST/MQTT PTT, Communicator mode, SpeexDSP echo cancellation, Debian packages, and a Home Assistant add-on.

## Related projects

talkiepi is one of three sibling projects in the same Mumble family:

| Project | Role |
|---------|------|
| [go-mumble-server](https://github.com/dchote/go-mumble-server) | Go Mumble server and the `pkg/mumble` protocol library this client imports |
| [esp32-mumble](https://github.com/dchote/esp32-mumble) | ESP32/ESPHome hardware client — shares Communicator mode, VAD tuning, and voice wire handling |
| **talkiepi** | Raspberry Pi / Linux / macOS client with web UI and HA add-on |

See [docs/ecosystem.md](docs/ecosystem.md) for how the three fit together.

## Features

- Mumble client built on `github.com/dchote/go-mumble-server/pkg/mumble` (no gumble/OpenAL)
- Opus voice via libopus; system audio via miniaudio (ALSA / PulseAudio / CoreAudio)
- Device enumeration and selection, SpeexDSP echo cancellation / noise suppression
- Modes: always-on, push-to-talk, Communicator (half-duplex with chime)
- Embedded Vue 3 + Vuetify 3 web UI and REST API (no auth; trusted LAN / HA ingress)
- MQTT discovery entities mirroring esp32-mumble for Home Assistant
- GitHub release update notifications
- Debian `.deb` packages (amd64 + arm64) buildable from macOS via Docker
- Home Assistant add-on with supervisor ingress

## Quick start (development)

```bash
# Dependencies (macOS)
brew install opus speexdsp

# Run API without embedded frontend
make run
# In another terminal:
cd frontend && yarn install && yarn dev
```

```bash
# Tests
make check

# Binary only
SKIP_FRONTEND=true ./scripts/build.sh
```

Default config: [configs/talkiepi.yaml](configs/talkiepi.yaml). Service knobs: [configs/talkiepi.default](configs/talkiepi.default) → `/etc/default/talkiepi`.

## Documentation

- [Product overview](docs/product-overview.md)
- [Technical overview](docs/technical-overview.md)
- [Build and test](docs/build-and-test.md)
- [Ecosystem](docs/ecosystem.md)
- [Hardware](docs/hardware/)
- [Deployment](docs/deployment/)

## License

Mozilla Public License 2.0 — see [LICENSE](LICENSE).
