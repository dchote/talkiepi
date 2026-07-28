package jitter

import (
	"testing"
)

func pcm(n int, v int16) []int16 {
	s := make([]int16, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func TestBufferPrebuffer(t *testing.T) {
	b := New(16, 2)
	b.Push(0, pcm(4, 1))
	if _, ok := b.Pop(); ok {
		t.Fatal("should not pop before target")
	}
	b.Push(1, pcm(4, 2))
	if !b.Primed() {
		t.Fatal("expected primed")
	}
	s, ok := b.Pop()
	if !ok || s[0] != 1 {
		t.Fatalf("first pop: ok=%v s=%v", ok, s)
	}
	s, ok = b.Pop()
	if !ok || s[0] != 2 {
		t.Fatalf("second pop: ok=%v s=%v", ok, s)
	}
}

func TestBufferDropsLate(t *testing.T) {
	b := New(16, 1)
	b.Push(5, pcm(2, 5))
	b.Pop() // advances next to 6
	b.Push(4, pcm(2, 4))
	drops, _ := b.Stats()
	if drops == 0 {
		t.Fatal("expected late drop")
	}
}

func TestBufferUnderrun(t *testing.T) {
	b := New(16, 1)
	b.Push(0, pcm(2, 1))
	b.Pop()
	if _, ok := b.Pop(); ok {
		t.Fatal("expected underrun")
	}
	_, underruns := b.Stats()
	if underruns != 1 {
		t.Fatalf("underruns=%d", underruns)
	}
}

func TestBufferOutOfOrder(t *testing.T) {
	b := New(16, 2)
	b.Push(1, pcm(2, 2))
	b.Push(0, pcm(2, 1))
	s, ok := b.Pop()
	if !ok || s[0] != 1 {
		t.Fatalf("want seq0 first: ok=%v s=%v", ok, s)
	}
	s, ok = b.Pop()
	if !ok || s[0] != 2 {
		t.Fatalf("want seq1 second: ok=%v s=%v", ok, s)
	}
}

func TestBufferCapacityEvict(t *testing.T) {
	b := New(2, 1)
	b.Push(0, pcm(1, 0))
	b.Push(1, pcm(1, 1))
	b.Push(2, pcm(1, 2))
	if b.Len() > 2 {
		t.Fatalf("len=%d", b.Len())
	}
}

func TestBufferReset(t *testing.T) {
	b := New(8, 1)
	b.Push(0, pcm(1, 1))
	b.Reset()
	if b.Len() != 0 || b.Primed() {
		t.Fatal("reset incomplete")
	}
}
