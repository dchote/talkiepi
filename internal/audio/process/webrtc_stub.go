package process

import "fmt"

func newWebRTC(cfg Config) (Processor, error) {
	return nil, fmt.Errorf("process: webrtc backend requires build tag webrtc_apm and libwebrtc-audio-processing-1")
}
