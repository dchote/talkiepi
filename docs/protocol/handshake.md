# Handshake (client deltas)

## Canonical reference

The Mumble control handshake, message catalog, encryption, and voice packet layout are documented in **go-mumble-server**:

- [Control messages](https://github.com/dchote/go-mumble-server/blob/main/docs/protocol/control-messages.md)
- [Encryption](https://github.com/dchote/go-mumble-server/blob/main/docs/protocol/encryption.md)
- [Security modes](https://github.com/dchote/go-mumble-server/blob/main/docs/protocol/security-modes.md)
- [Voice data](https://github.com/dchote/go-mumble-server/blob/main/docs/protocol/voice-data.md)

talkiepi implements the client side via `github.com/dchote/go-mumble-server/pkg/mumble` **v0.2.0**. This page lists **client-only deltas**—do not treat it as a protocol spec.

## Sequence (talkiepi)

Implemented in `pkg/mumbleclient.Client.handshake`:

1. TLS connect (`InsecureSkipVerify` when `server.insecure_tls`)
2. Read server **Version**
3. Send client **Version** (`Release` default `"talkiepi"`, `OS` `"Go"`, `CryptoModes` bitmask)
4. Send **Authenticate** (`username`, `password`, `opus=true`)
5. Loop until **ServerSync**: apply **CryptSetup**, ingest **ChannelState** / **UserState**, ignore CodecVersion / ServerConfig; **Reject** → error
6. Optional **UserState** channel join by configured name
7. Start UDP transport + TCP/UDP ping loops

## Client deltas vs generic Mumble

| Topic | talkiepi behavior |
|-------|-------------------|
| Crypto advertisement | `Version.CryptoModes` defaults to **lite \| legacy** (`DefaultCryptoModes`), matching esp32-mumble |
| Preferred mode | Config `server.crypto`: `legacy` or `lite` (selection after CryptSetup key shape) |
| Opus only | Authenticate always sets `opus=true`; no CELT |
| Release string | `"talkiepi"` unless overridden |
| TCP ping | Every **15s**; read deadline **45s** |
| Reconnect | Exponential backoff **1s → 30s** (unless `DisableReconnect`) |
| UDP ping | **5** probes at **500ms**, then every **5s** |
| Channel join | After sync, by **name** from `server.channel` (not ACL-aware beyond server reject) |
| Library | Shared structs/codecs from go-mumble-server—not a barnard/gumble stack |

## CryptSetup handling

- Full setup when key and/or both nonces present → `crypto.NewCryptState` + `SetKey`
- Lone server nonce → decrypt-nonce resync
- Empty client nonce with no key → reply with client encrypt nonce (resync), same pattern as family clients

## Out of scope here

Server ACL evaluation, virtual servers, and admin UserState presence rules belong in go-mumble-server docs. talkiepi is a single-session voice client with roster + text.
