package mumbleclient

import (
	"sync"
	"testing"

	"github.com/dchote/go-mumble-server/pkg/mumble/protocol/messages"
)

func TestApplyChannelStateAddAndUpdate(t *testing.T) {
	r := NewRoster()

	ch, added := r.ApplyChannelState(&messages.ChannelState{
		ChannelID: 0,
		Name:      "Root",
	})
	if !added || ch.Name != "Root" || ch.HasParent {
		t.Fatalf("root add: %+v added=%v", ch, added)
	}

	ch, added = r.ApplyChannelState(&messages.ChannelState{
		ChannelID: 1,
		Parent:    0,
		HasParent: true,
		Name:      "Lounge",
		Position:  2,
	})
	if !added || ch.Name != "Lounge" || ch.ParentID != 0 || !ch.HasParent || ch.Position != 2 {
		t.Fatalf("child add: %+v added=%v", ch, added)
	}

	ch, added = r.ApplyChannelState(&messages.ChannelState{
		ChannelID: 1,
		Name:      "Lounge-2",
	})
	if added || ch.Name != "Lounge-2" || ch.ParentID != 0 || !ch.HasParent {
		t.Fatalf("child update should preserve parent: %+v added=%v", ch, added)
	}

	got, ok := r.ChannelByName("lounge-2")
	if !ok || got.ID != 1 {
		t.Fatalf("ChannelByName: ok=%v got=%+v", ok, got)
	}
	if _, ok := r.ChannelByName("missing"); ok {
		t.Fatal("expected miss")
	}
}

func TestChannelLinksAddRemove(t *testing.T) {
	r := NewRoster()
	r.ApplyChannelState(&messages.ChannelState{
		ChannelID: 1,
		Name:      "A",
		Links:     []uint32{2, 3},
	})
	ch, _ := r.ApplyChannelState(&messages.ChannelState{
		ChannelID:   1,
		LinksAdd:    []uint32{4},
		LinksRemove: []uint32{2},
	})
	if len(ch.Links) != 2 {
		t.Fatalf("links=%v", ch.Links)
	}
	set := map[uint32]bool{}
	for _, id := range ch.Links {
		set[id] = true
	}
	if !set[3] || !set[4] || set[2] {
		t.Fatalf("links=%v", ch.Links)
	}
}

func TestRemoveChannel(t *testing.T) {
	r := NewRoster()
	r.ApplyChannelState(&messages.ChannelState{ChannelID: 5, Name: "X"})
	ch, ok := r.RemoveChannel(5)
	if !ok || ch.Name != "X" {
		t.Fatalf("remove: %+v ok=%v", ch, ok)
	}
	if _, ok := r.ChannelByID(5); ok {
		t.Fatal("still present")
	}
	if _, ok := r.RemoveChannel(5); ok {
		t.Fatal("double remove")
	}
}

func TestApplyUserStateAddUpdateRemove(t *testing.T) {
	r := NewRoster()

	u, added := r.ApplyUserState(&messages.UserState{
		Session:   10,
		Name:      "alice",
		ChannelID: 1,
		SetFields: messages.UserStateSetSession | messages.UserStateSetName | messages.UserStateSetChannelID,
	})
	if !added || u.Name != "alice" || u.ChannelID != 1 {
		t.Fatalf("add: %+v added=%v", u, added)
	}

	u, added = r.ApplyUserState(&messages.UserState{
		Session:   10,
		SelfMute:  true,
		SetFields: messages.UserStateSetSession | messages.UserStateSetSelfMute,
	})
	if added || !u.SelfMute || u.Name != "alice" || u.ChannelID != 1 {
		t.Fatalf("partial update: %+v added=%v", u, added)
	}

	// Move to root (channel 0) via presence bit.
	u, _ = r.ApplyUserState(&messages.UserState{
		Session:   10,
		ChannelID: 0,
		SetFields: messages.UserStateSetSession | messages.UserStateSetChannelID,
	})
	if u.ChannelID != 0 {
		t.Fatalf("root move: %+v", u)
	}

	// Explicit unmute via presence bit.
	u, _ = r.ApplyUserState(&messages.UserState{
		Session:   10,
		SelfMute:  false,
		SetFields: messages.UserStateSetSession | messages.UserStateSetSelfMute,
	})
	if u.SelfMute {
		t.Fatalf("unmute: %+v", u)
	}

	got, ok := r.UserBySession(10)
	if !ok || got.Name != "alice" {
		t.Fatalf("UserBySession: %+v ok=%v", got, ok)
	}

	removed, ok := r.RemoveUser(10)
	if !ok || removed.Name != "alice" {
		t.Fatalf("RemoveUser: %+v ok=%v", removed, ok)
	}
	if _, ok := r.UserBySession(10); ok {
		t.Fatal("user still present")
	}
}

func TestRosterSnapshotsAndReset(t *testing.T) {
	r := NewRoster()
	r.ApplyChannelState(&messages.ChannelState{ChannelID: 0, Name: "Root"})
	r.ApplyChannelState(&messages.ChannelState{ChannelID: 1, Name: "A", HasParent: true})
	r.ApplyUserState(&messages.UserState{Session: 1, Name: "u1", SetFields: messages.UserStateSetName})
	r.ApplyUserState(&messages.UserState{Session: 2, Name: "u2", SetFields: messages.UserStateSetName})

	if len(r.Channels()) != 2 {
		t.Fatalf("channels=%d", len(r.Channels()))
	}
	if len(r.Users()) != 2 {
		t.Fatalf("users=%d", len(r.Users()))
	}

	r.Reset()
	if len(r.Channels()) != 0 || len(r.Users()) != 0 {
		t.Fatal("reset failed")
	}
}

func TestRosterConcurrentAccess(t *testing.T) {
	r := NewRoster()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := uint32(i)
			r.ApplyChannelState(&messages.ChannelState{ChannelID: id, Name: "c"})
			r.ApplyUserState(&messages.UserState{
				Session:   id,
				Name:      "u",
				SetFields: messages.UserStateSetName,
			})
			_, _ = r.ChannelByName("c")
			_, _ = r.UserBySession(id)
			_ = r.Channels()
			_ = r.Users()
			if i%2 == 0 {
				r.RemoveUser(id)
				r.RemoveChannel(id)
			}
		}(i)
	}
	wg.Wait()
}

func TestChannelByNameCaseInsensitive(t *testing.T) {
	r := NewRoster()
	r.ApplyChannelState(&messages.ChannelState{ChannelID: 3, Name: "Ops"})
	ch, ok := r.ChannelByName("ops")
	if !ok || ch.ID != 3 {
		t.Fatalf("got %+v ok=%v", ch, ok)
	}
	ch, ok = r.ChannelByName("OPS")
	if !ok || ch.ID != 3 {
		t.Fatalf("got %+v ok=%v", ch, ok)
	}
}
