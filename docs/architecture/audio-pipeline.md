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

Packages: `internal/audio`, `internal/audio/process`, `internal/codec`, `internal/vad`, `internal/jitter`, plus send/receive in `pkg/mumbleclient`.

## Sample rates and profiles

| Profile | Rate | Frame | Bitrate | Complexity | Use |
|---------|------|-------|---------|------------|-----|
| **Default** (`codec.Default`) | **48 kHz** | 20 ms (960 samples) | 40 kbps | 5 | talkiepi Pi / desktop |
| **Esp32Compatible** | **16 kHz** | 20 ms (320 samples) | 16 kbps | 1 | Interop with esp32-mumble |

Config keys: `voice.sample_rate`, `voice.bitrate`, `voice.frame_ms`, `voice.complexity`. Device `audio.input` / `audio.output` sample rates default to 48000 with `period_frames: 960`.

Servers and clients negotiate Opus; mixed 16 kHz / 48 kHz sessions work when Opus handles resampling, but matching rates reduces CPU and latency.

## Capture and playback

- Backend auto-select via miniaudio: ALSA, PulseAudio, JACK, CoreAudio, or `default`.
- Devices resolved by exact `device_id`, then `device_name`, then system default (`audio.Resolve`).
- Duplex: same or split devices; `audio.processing.duplex_mode` (`auto` by default).
- Output volume 0–100; input `gain_db`.

See [hardware/audio-devices.md](../hardware/audio-devices.md) for Pulse `module-echo-cancel` and tuning.

## Processing (AEC / NS)

`internal/audio/process`:

| Backend | Role |
|---------|------|
| `speex` | Default — SpeexDSP echo canceller + preprocessor (NS, optional AGC/HPF) |
| `none` | Pass-through |
| `webrtc` | Stub / future |

Far-end: every playback frame is `ProcessRender`. Near-end: `ProcessCapture` runs AEC then NS in place.

Useful knobs:

- `echo_cancel`, `echo_tail_ms` (default 150, max 500)
- `echo_delay_ms`, `adaptive_delay`
- `noise_suppress`, `ns_level`
- `agc`, `agc_target_dbfs`, `high_pass`, `dereverb`

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
