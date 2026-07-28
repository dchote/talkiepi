# Debian deployment

## Install from `.deb`

Build locally (including from macOS via Docker):

```bash
./scripts/build-deb.sh
sudo apt install ./dist/talkiepi_*_amd64.deb   # or *_arm64.deb
```

Release artifacts are published from the [Release workflow](../../.github/workflows/release.yml) on `v*` tags.

Runtime dependencies: `libopus0`, `libspeexdsp1`.

## Layout

| Path | Purpose |
|------|---------|
| `/usr/bin/talkiepi` | Binary (embedded UI when built with `embed_frontend`) |
| `/usr/share/talkiepi/talkiepi.yaml` | Packaged example config |
| `/etc/talkiepi/talkiepi.yaml` | Live config (created/copied by postinstall) |
| `/etc/default/talkiepi` | systemd `EnvironmentFile` (service knobs) |
| `/lib/systemd/system/talkiepi.service` | Unit |
| `/var/lib/talkiepi` | State (update cache, etc.) |

## Enable and start

```bash
sudo systemctl enable --now talkiepi
sudo systemctl status talkiepi
journalctl -u talkiepi -f
```

Edit `/etc/talkiepi/talkiepi.yaml` for server, audio, GPIO, MQTT. Optional overrides in `/etc/default/talkiepi`:

```bash
#TALKIEPI_CONFIG=/etc/talkiepi/talkiepi.yaml
#TALKIEPI_WEB_HOST=0.0.0.0
#TALKIEPI_WEB_PORT=8080
#TALKIEPI_STATE_DIR=/var/lib/talkiepi
#TALKIEPI_ARGS=
```

Any `TALKIEPI_*` key supported by `internal/config` can be set via the environment file.

## Service user

The unit runs as user/group `talkiepi` with supplementary groups `audio` and `gpio`. Ensure sound devices and `/dev/gpiochip*` are accessible to those groups.

## Firewall

UI/API listen on `web.port` (default **8080**). Restrict to LAN or terminate TLS/auth at a reverse proxy—the API has no built-in authentication.

## Related

- [homeassistant-addon.md](homeassistant-addon.md)
- [build-and-test.md](../build-and-test.md)
