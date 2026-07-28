# macOS deployment

macOS is a **development and desktop client** target (CoreAudio). GPIO PTT/LEDs are stubbed; use the web UI or REST for PTT.

## Dependencies

```bash
brew install go opus speexdsp pkg-config node yarn
```

Go **1.25+**, Node **20+**.

## Run from source

```bash
# API (no embedded frontend)
make run

# UI (separate terminal)
cd frontend && yarn install && yarn dev
```

Or build a local binary:

```bash
SKIP_FRONTEND=true ./scripts/build.sh
./build/talkiepi -config configs/talkiepi.yaml -frontend-embed=false
```

Full embed:

```bash
make frontend
./scripts/build.sh
./build/talkiepi -config configs/talkiepi.yaml
```

Open `http://127.0.0.1:8080` when the frontend is embedded.

## Audio

- Backend: CoreAudio (`audio.*.backend: coreaudio` or auto).
- Grant microphone permission to the terminal/app when prompted.
- SpeexDSP AEC works in-process; Bluetooth headsets may need higher `echo_tail_ms`.

## Debian packages from macOS

Use Docker + QEMU (same as CI):

```bash
./scripts/build-deb.sh
```

See [build-and-test.md](../build-and-test.md).

## Limits vs Linux / Pi

| Feature | macOS |
|---------|--------|
| CoreAudio I/O | Yes |
| Speex AEC / Opus | Yes (CGO + brew libs) |
| GPIO PTT / LEDs | No (stubs) |
| systemd / `.deb` install | N/A (cross-build only) |
| HA add-on | Run on HA OS / supervised host, not as a native Mac app |
