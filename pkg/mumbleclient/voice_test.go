package mumbleclient

import (
	"bytes"
	"testing"

	"github.com/dchote/go-mumble-server/pkg/mumble/audio"
)

func TestBuildParseClientVoiceRoundTrip(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0xaa}
	buf := make([]byte, 256)

	n, err := BuildClientVoice(buf, audio.CodecOpus, 0, 42, payload, false)
	if err != nil {
		t.Fatalf("BuildClientVoice: %v", err)
	}
	got, err := ParseClientVoice(buf[:n])
	if err != nil {
		t.Fatalf("ParseClientVoice: %v", err)
	}
	if got.Codec != audio.CodecOpus || got.Target != 0 || got.Sequence != 42 {
		t.Fatalf("header mismatch: %+v", got)
	}
	if got.IsTerminator {
		t.Fatal("expected non-terminator")
	}
	if !bytes.Equal(got.Payload, payload) {
		t.Fatalf("payload: got %v want %v", got.Payload, payload)
	}

	// Must agree with library ParsePacket.
	lib, err := audio.ParsePacket(buf[:n])
	if err != nil {
		t.Fatalf("audio.ParsePacket: %v", err)
	}
	if lib.Sequence != got.Sequence || lib.IsTerminator != got.IsTerminator || !bytes.Equal(lib.Payload, got.Payload) {
		t.Fatalf("library mismatch: lib=%+v got=%+v", lib, got)
	}
}

func TestBuildClientVoiceTerminator(t *testing.T) {
	buf := make([]byte, 64)
	n, err := BuildClientVoice(buf, audio.CodecOpus, 0, 7, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseClientVoice(buf[:n])
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsTerminator {
		t.Fatal("expected terminator")
	}
	if len(got.Payload) != 0 {
		t.Fatalf("payload len=%d", len(got.Payload))
	}
}

func TestParseServerVoice(t *testing.T) {
	// Manually craft: header | session=5 | seq=10 | len=3 | aaa
	buf := make([]byte, 64)
	buf[0] = (audio.CodecOpus << 5) | 0
	pos := 1
	pos += audio.EncodeVarint(buf[pos:], 5)
	pos += audio.EncodeVarint(buf[pos:], 10)
	pos += audio.EncodeVarint(buf[pos:], 3)
	copy(buf[pos:], []byte("aaa"))
	pos += 3

	got, err := ParseServerVoice(buf[:pos])
	if err != nil {
		t.Fatal(err)
	}
	if got.Session != 5 || got.Sequence != 10 || !bytes.Equal(got.Payload, []byte("aaa")) {
		t.Fatalf("got %+v", got)
	}
}

func TestParseServerVoiceTerminator(t *testing.T) {
	buf := make([]byte, 64)
	buf[0] = audio.CodecOpus << 5
	pos := 1
	pos += audio.EncodeVarint(buf[pos:], 1)
	pos += audio.EncodeVarint(buf[pos:], 0)
	pos += audio.EncodeVarint(buf[pos:], TerminatorBit) // empty terminator
	got, err := ParseServerVoice(buf[:pos])
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsTerminator || len(got.Payload) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseServerVoiceErrors(t *testing.T) {
	if _, err := ParseServerVoice(nil); err != ErrVoiceTooShort {
		t.Fatalf("nil: %v", err)
	}
	if _, err := ParseServerVoice([]byte{0x80}); err != ErrVoiceTooShort {
		t.Fatalf("short: %v", err)
	}
	// Truncated payload
	buf := make([]byte, 16)
	buf[0] = audio.CodecOpus << 5
	pos := 1
	pos += audio.EncodeVarint(buf[pos:], 1)
	pos += audio.EncodeVarint(buf[pos:], 0)
	pos += audio.EncodeVarint(buf[pos:], 10) // claims 10 bytes
	if _, err := ParseServerVoice(buf[:pos]); err != ErrVoiceTruncated {
		t.Fatalf("truncated: %v", err)
	}
}

func TestUDPPingRoundTrip(t *testing.T) {
	buf := make([]byte, 32)
	n, err := BuildUDPPing(buf, 123456789)
	if err != nil {
		t.Fatal(err)
	}
	ts, ok := ParseUDPPing(buf[:n])
	if !ok || ts != 123456789 {
		t.Fatalf("got ts=%d ok=%v", ts, ok)
	}
	if _, ok := ParseUDPPing([]byte{0x80, 0x01}); ok {
		t.Fatal("non-ping should fail")
	}
}

func TestBuildClientVoiceBufferTooSmall(t *testing.T) {
	if _, err := BuildClientVoice(make([]byte, 2), audio.CodecOpus, 0, 1, []byte{1, 2, 3}, false); err != ErrVoiceEncodeSize {
		t.Fatalf("got %v", err)
	}
}
