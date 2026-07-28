# talkiepi Home Assistant Add-on

Headless Mumble client with a web UI for connection status, server/audio settings, channel roster, chat, and PTT/communicator controls. Designed for Raspberry Pi–style walkie-talkie use inside Home Assistant, with audio via PulseAudio and optional GPIO PTT.

## Installation

1. In Home Assistant, go to **Settings** → **Add-ons** → **Add-on store**.
2. Click the three dots (⋮) → **Repositories**.
3. Add this repository URL: `https://github.com/dchote/talkiepi`
4. Find **talkiepi** in the add-on list and click **Install**.
5. Configure options if needed, then **Start** the add-on.

## Configuration

Options mirror the esp32-mumble / talkiepi client settings:

| Option | Description |
|--------|-------------|
| **Server / Port** | Mumble server to join. Empty server = auto-detect when supported. |
| **Username / Password / Channel** | Login and start channel. |
| **Mode** | `always_on`, `push_to_talk`, or `communicator`. |
| **Crypto** | `legacy` or `lite`. |
| **Speaker volume / Mic gain** | Output volume (0–100) and input gain (dB). |
| **Audio input / output** | Device names (empty = default). |
| **Echo cancel / Noise suppress** | Processing toggles. |
| **PTT GPIO** | Optional chip/line for a hardware PTT button. |
| **Web port** | Ingress REST/UI port (default 8080). |
| **Log level** | `debug`, `info`, `warn`, or `error`. |
| **Force options** | Re-apply add-on options over `/data/talkiepi.yaml` on every start. |

Runtime state and config live under the add-on’s persistent `/data` directory. Use **Open Web UI** for day-to-day control; MQTT discovery (when Mosquitto is available) can expose entities similar to esp32-mumble.

## Web UI (ingress)

Click **Open Web UI** in the add-on panel to:

- View connection status, ping, and transport
- Connect / disconnect and use PTT or communicator controls
- Edit server and audio settings
- Browse channels/users and send text messages

The UI is served through Home Assistant ingress (no separate firewall rules).

## Audio and GPIO

- `audio: true` provides PulseAudio from the supervisor for capture/playback.
- `/dev/gpiochip0` is mapped for optional PTT hardware.
- Runtime libraries: `libopus0`, `libspeexdsp1`.

## Publishing (maintainers)

1. Build and push images (e.g. GitHub Actions addon workflow) as `ghcr.io/dchote/{arch}-addon-talkiepi:<version>`.
2. Make the GHCR packages **public** so the Supervisor can pull them.
3. Keep `addon/config.yaml` `version` aligned with the image tag.

## Support

- [GitHub repository](https://github.com/dchote/talkiepi)
- [Project documentation](https://github.com/dchote/talkiepi#readme)
