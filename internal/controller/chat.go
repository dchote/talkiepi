package controller

import (
	"sync"
	"time"
)

const chatHistoryLimit = 500

// ChatMessage is a stored channel/direct text message.
type ChatMessage struct {
	ID         string    `json:"id"`
	At         time.Time `json:"at"`
	Actor      uint32    `json:"actor"`
	Author     string    `json:"author"`
	Message    string    `json:"message"`
	ChannelIDs []uint32  `json:"channel_ids,omitempty"`
	Sessions   []uint32  `json:"sessions,omitempty"`
	TreeIDs    []uint32  `json:"tree_ids,omitempty"`
	Outbound   bool      `json:"outbound,omitempty"`
}

type chatStore struct {
	mu   sync.RWMutex
	msgs []ChatMessage
	seq  uint64
}

func (s *chatStore) append(msg ChatMessage) ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	if msg.ID == "" {
		msg.ID = time.Now().UTC().Format("20060102150405") + "-" + itoa(s.seq)
	}
	if msg.At.IsZero() {
		msg.At = time.Now().UTC()
	}
	s.msgs = append(s.msgs, msg)
	if len(s.msgs) > chatHistoryLimit {
		s.msgs = append([]ChatMessage(nil), s.msgs[len(s.msgs)-chatHistoryLimit:]...)
	}
	return msg
}

func (s *chatStore) list(channelID *uint32, limit int) []ChatMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > chatHistoryLimit {
		limit = 200
	}
	out := make([]ChatMessage, 0, limit)
	for i := len(s.msgs) - 1; i >= 0 && len(out) < limit; i-- {
		m := s.msgs[i]
		if channelID != nil {
			if !messageMatchesChannel(m, *channelID) {
				continue
			}
		}
		out = append(out, m)
	}
	// Reverse to chronological order.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func messageMatchesChannel(m ChatMessage, channelID uint32) bool {
	if len(m.Sessions) > 0 && len(m.ChannelIDs) == 0 && len(m.TreeIDs) == 0 {
		return true // DMs visible in any channel view (client filters further)
	}
	for _, id := range m.ChannelIDs {
		if id == channelID {
			return true
		}
	}
	for _, id := range m.TreeIDs {
		if id == channelID {
			return true
		}
	}
	return false
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
