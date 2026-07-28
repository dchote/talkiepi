# Audio pipeline

## Overview

```
Mic ──► capture device ──► gain ──► process (AEC/NS) ──► VAD / mode gate
                                                              │
                                                         Opus encode
                                                              │
                                                    UDP voice / TCP tunnel
                                                              │
Server ◄──────────────────────────────────────────────────────┘

Server ──► Opus decode ──► jitter buffer ──► playback device
                              │
                              └──► ProcessRender (AEC far-end)
```

Packages: `internal/audio` (Engine + malgo DeviceIO), `internal/audio/process`, `internal/codec` (Opus encode/decode with `-tags opus`), `internal/vad`, `internal/jitter`, plus `pkg/mumbleclient.SendVoice` / voice parse. The controller starts the Engine after a successful Mumble dial, tears it down on disconnect, and **restarts it on auto-reconnect** (`Connected` after `Disconnected`).

## Sample rates and profiles

| Profile | Rate | Frame | Bitrate | Complexity | Use |
|---------|------|-------|---------|------------|-----|
| **Default** (`codec.Default`) | **48 kHz** | 20 ms (960 samples) | 40 kbps | 5 | talkiepi Pi / desktop |
| **Esp32Compatible** | **16 kHz** | 20 ms (320 samples) | 16 kbps | 1 | Interop with esp32-mumble |

Config keys: `voice.sample_rate`, `voice.bitrate`, `voice.frame_ms`, `voice.complexity`. Device `audio.input` / `audio.output` sample rates default to 48000 with `period_frames: 960` (informational; the engine uses the voice profile for device open).

Servers and clients negotiate Opus; mixed 16 kHz / 48 kHz sessions work when Opus handles resampling, but matching rates reduces CPU and latency.

## Capture and playback

- Backend auto-select via miniaudio: ALSA, PulseAudio, JACK, CoreAudio, or `default`.
- Devices resolved by exact `device_id`, then `device_name`, then system default (`audio.Resolve`).
- Output `volume` (0–100) scales playback after the mixer; input `gain_db` scales capture after AEC/NS.
- While a mode allows TX (`push_to_talk` held, `always_on`, or Communicator open), the VAD transmit gate is **held open** each frame so hangover does not end TX mid-hold.

See [hardware/audio-devices.md](../hardware/audio-devices.md) for Pulse `module-echo-cancel` and tuning.

## Processing (AEC / NS)

`internal/audio/process`:

| Backend | Role |
|---------|------|
| `speex` | Default — SpeexDSP echo canceller + preprocessor (NS, optional AGC) + optional software HPF |
| `none` | Pass-through |
| `webrtc` | Stub / future |

Far-end: every playback frame is `ProcessRender` (optional fixed `echo_delay_ms` alignment). Near-end: `ProcessCapture` runs AEC then NS; `high_pass` applies an ~80 Hz one-pole HPF afterward.

## Communicator chime

Open and close chimes use the same embedded asset as esp32-mumble: `esphome/sounds/communicator.wav` (16-bit mono, 16 kHz), vendored at `internal/audio/sounds/communicator.wav`. Playback resamples to the engine rate and applies volume scale `0.25` (matching esp32 `CHIME_VOLUME_SCALE`).

Useful knobs:

- `echo_cancel`, `echo_tail_ms` (default 150, max 500)
- `echo_delay_ms` (fixed far-end delay before AEC)
- `noise_suppress`, `ns_level` (`mild` / `moderate` / `aggressive`)
- `agc`, `agc_target_dbfs`, `high_pass`, `dereverb`

Reserved for future device-open policy (stored, not yet applied): `duplex_mode`, `adaptive_delay`.

## VAD (esp32-mumble constants)

`internal/vad` defaults are credited to [esp32-mumble](https://github.com/dchote/esp32-mumble) voice capture:

| Constant | Default |
|----------|---------|
| Calibration frames | 12 |
| Attack frames | 3 |
| Hangover frames | 15 |
| Min threshold | 120 |
| Margin | 2.5 |

Used for speech detection / hangover in modes that gate on energy (and Communicator silence window coordination).

## Jitter buffer

`internal/jitter` holds PCM frames by sequence. Late packets drop; gaps yield zeros (caller may use Opus PLC before push). Capacity / target frame counts bound latency vs underruns.

## Voice on the wire

Encoded Opus packets use `pkg/mumble` voice framing (UDP encrypted with CryptState, or `UDPTunnel` over TLS). Send/receive sequence and terminator bits follow the same conventions as esp32-mumble and go-mumble-server [voice-data](https://github.com/dchote/go-mumble-server/blob/main/docs/protocol/voice-data.md).
