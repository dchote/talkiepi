package audio

import "math"

// GainLinearFromDB converts a gain in dB to a linear amplitude scale factor.
func GainLinearFromDB(db float64) float64 {
	return math.Pow(10, db/20)
}

// VolumeLinearFromPercent converts 0–100 volume to a linear scale factor.
// Values outside 0–100 are clamped.
func VolumeLinearFromPercent(pct float64) float64 {
	if pct <= 0 {
		return 0
	}
	if pct >= 100 {
		return 1
	}
	return pct / 100
}

// ApplyLinearGain scales PCM in place by linear (clamped to int16).
// A factor of 1 is a no-op; 0 clears the buffer.
func ApplyLinearGain(pcm []int16, linear float64) {
	if linear == 1 {
		return
	}
	if linear <= 0 {
		for i := range pcm {
			pcm[i] = 0
		}
		return
	}
	for i, s := range pcm {
		v := float64(s) * linear
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		pcm[i] = int16(v)
	}
}
