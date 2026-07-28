package mumbleclient

import (
	"testing"

	"github.com/dchote/go-mumble-server/pkg/mumble/audio"
)

func TestBuildClientVoiceForSend(t *testing.T) {
	buf := make([]byte, 256)
	n, err := BuildClientVoice(buf, audio.CodecOpus, 0, 3, []byte{9, 8, 7}, false)
	if err != nil || n < 4 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	cv, err := ParseClientVoice(buf[:n])
	if err != nil {
		t.Fatal(err)
	}
	if cv.Sequence != 3 || len(cv.Payload) != 3 {
		t.Fatalf("%+v", cv)
	}
}

func TestSendVoiceRequiresConnection(t *testing.T) {
	c := New(Config{Address: "127.0.0.1:1", Username: "t"})
	if err := c.SendVoice([]byte{1}, false); err != ErrNotConnected {
		t.Fatalf("got %v", err)
	}
}
