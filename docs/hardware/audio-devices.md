# Audio devices

## Backends

talkiepi enumerates devices through miniaudio. Config `audio.input.backend` / `audio.output.backend`:

| Value | Platform |
|-------|----------|
| `""` / auto | Prefer system default stack |
| `alsa` | Linux direct ALSA |
| `pulseaudio` | PulseAudio (also common under PipeWire’s Pulse compat) |
| `jack` | JACK |
| `coreaudio` | macOS |
| `default` | Backend-agnostic default device |

Resolve order: exact `device_id` → `device_name` → default device ([`audio.Resolve`](../../internal/audio/device.go)).

## Duplex vs split

| Setup | When |
|-------|------|
| **Duplex** | Same USB/HAT device for mic + speaker; lowest AEC delay when clocks match |
| **Split** | Mic on one device, speaker on another; set `echo_delay_ms` or rely on `adaptive_delay` |

`audio.processing.duplex_mode: auto` picks a sensible path from the opened streams.

## AEC tuning (in-process Speex)

Defaults favor speakerphone-style Pi hardware:

```yaml
audio:
  processing:
    backend: speex
    echo_cancel: true
    echo_tail_ms: 150      # raise toward 200–300 for larger rooms / Bluetooth
    echo_delay_ms: 0
    adaptive_delay: true
    noise_suppress: true
    ns_level: moderate
    high_pass: true
```

Tips:

- Prefer wired I/O over Bluetooth for AEC stability.
- Keep period size near 20 ms (`period_frames` ≈ rate/50).
- If echo remains, increase `echo_tail_ms` before disabling cancel.
- Set `backend: none` only for debugging.

## PulseAudio / PipeWire `module-echo-cancel`

When the system already runs a good echo-cancel sink/source, you can point talkiepi at those virtual devices and turn off in-process AEC:

```bash
pactl load-module module-echo-cancel \
  aec_method=webrtc \
  source_master=alsa_input.platform-....analog-stereo \
  sink_master=alsa_output.platform-....analog-stereo
```

Then set `audio.input.device_name` / `audio.output.device_name` to the echo-cancel source/sink names, and:

```yaml
audio:
  processing:
    echo_cancel: false   # or backend: none
```

Under PipeWire, use PulseAudio compatibility (`pipewire-pulse`) or WirePlumber equivalents; device names still appear as Pulse-style strings to talkiepi when using the pulse backend.

## Home Assistant add-on

The add-on sets `audio: true` so the supervisor provides PulseAudio into the container. Prefer device names from the UI’s audio page; GPIO PTT is independent of the audio path.

## Related

- [audio-pipeline.md](../architecture/audio-pipeline.md)
- [wiring.md](wiring.md)
