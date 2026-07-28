package audio

import (
	"testing"
)

func TestRingWriteRead(t *testing.T) {
	r := NewRing(8)
	in := []int16{1, 2, 3, 4}
	if r.Write(in) != 4 {
		t.Fatal("write")
	}
	if r.Len() != 4 {
		t.Fatalf("len=%d", r.Len())
	}
	out := make([]int16, 4)
	if n := r.Read(out); n != 4 || out[0] != 1 || out[3] != 4 {
		t.Fatalf("read n=%d out=%v", n, out)
	}
	if r.Len() != 0 {
		t.Fatal("empty")
	}
}

func TestRingOverflowDropsOldest(t *testing.T) {
	r := NewRing(4)
	r.Write([]int16{1, 2, 3, 4})
	r.Write([]int16{5, 6})
	out := make([]int16, 4)
	n := r.Read(out)
	if n != 4 {
		t.Fatalf("n=%d", n)
	}
	// Oldest 1,2 dropped; remaining 3,4,5,6
	if out[0] != 3 || out[3] != 6 {
		t.Fatalf("out=%v", out)
	}
}

func TestRingPartialRead(t *testing.T) {
	r := NewRing(16)
	r.Write([]int16{1, 2, 3})
	out := make([]int16, 2)
	if r.Read(out) != 2 {
		t.Fatal("partial")
	}
	if r.Len() != 1 {
		t.Fatalf("len=%d", r.Len())
	}
}

func TestRingClear(t *testing.T) {
	r := NewRing(4)
	r.Write([]int16{1, 2})
	r.Clear()
	if r.Len() != 0 {
		t.Fatal("clear")
	}
}
