package mumbleclient

import (
	"strings"
	"sync"

	"github.com/dchote/go-mumble-server/pkg/mumble/protocol/messages"
)

// Channel is a roster channel node.
type Channel struct {
	ID          uint32   `json:"id"`
	ParentID    uint32   `json:"parent_id"`
	HasParent   bool     `json:"has_parent"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Temporary   bool     `json:"temporary"`
	Position    int32    `json:"position"`
	MaxUsers    uint32   `json:"max_users"`
	Links       []uint32 `json:"links,omitempty"`
}

// User is a roster user entry.
type User struct {
	Session         uint32 `json:"session"`
	Name            string `json:"name"`
	UserID          uint32 `json:"user_id"`
	ChannelID       uint32 `json:"channel_id"`
	Mute            bool   `json:"mute"`
	Deaf            bool   `json:"deaf"`
	Suppress        bool   `json:"suppress"`
	SelfMute        bool   `json:"self_mute"`
	SelfDeaf        bool   `json:"self_deaf"`
	Comment         string `json:"comment,omitempty"`
	Hash            string `json:"hash,omitempty"`
	PrioritySpeaker bool   `json:"priority_speaker"`
	Recording       bool   `json:"recording"`
}

// Roster is a thread-safe channel tree and user registry.
type Roster struct {
	mu       sync.RWMutex
	channels map[uint32]*Channel
	users    map[uint32]*User
}

// NewRoster returns an empty roster.
func NewRoster() *Roster {
	return &Roster{
		channels: make(map[uint32]*Channel),
		users:    make(map[uint32]*User),
	}
}

// Reset clears all channels and users.
func (r *Roster) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels = make(map[uint32]*Channel)
	r.users = make(map[uint32]*User)
}

// ApplyChannelState merges a ChannelState into the roster.
// added is true when the channel ID was not previously known.
func (r *Roster) ApplyChannelState(cs *messages.ChannelState) (ch Channel, added bool) {
	if cs == nil {
		return Channel{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.channels[cs.ChannelID]
	if !ok {
		existing = &Channel{ID: cs.ChannelID}
		r.channels[cs.ChannelID] = existing
		added = true
	}
	mergeChannelState(existing, cs)
	return cloneChannel(existing), added
}

// RemoveChannel deletes a channel by ID.
func (r *Roster) RemoveChannel(id uint32) (ch Channel, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.channels[id]
	if !ok {
		return Channel{}, false
	}
	ch = cloneChannel(existing)
	delete(r.channels, id)
	return ch, true
}

// ApplyUserState merges a UserState into the roster.
// added is true when the session was not previously known.
func (r *Roster) ApplyUserState(us *messages.UserState) (u User, added bool) {
	if us == nil {
		return User{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.users[us.Session]
	if !ok {
		existing = &User{Session: us.Session}
		r.users[us.Session] = existing
		added = true
	}
	mergeUserState(existing, us)
	return cloneUser(existing), added
}

// RemoveUser deletes a user by session.
func (r *Roster) RemoveUser(session uint32) (u User, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.users[session]
	if !ok {
		return User{}, false
	}
	u = cloneUser(existing)
	delete(r.users, session)
	return u, true
}

// Channels returns a snapshot of all channels.
func (r *Roster) Channels() []Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Channel, 0, len(r.channels))
	for _, ch := range r.channels {
		out = append(out, cloneChannel(ch))
	}
	return out
}

// Users returns a snapshot of all users.
func (r *Roster) Users() []User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]User, 0, len(r.users))
	for _, u := range r.users {
		out = append(out, cloneUser(u))
	}
	return out
}

// ChannelByName finds a channel by name (case-insensitive).
func (r *Roster) ChannelByName(name string) (Channel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	want := strings.ToLower(name)
	for _, ch := range r.channels {
		if strings.ToLower(ch.Name) == want {
			return cloneChannel(ch), true
		}
	}
	return Channel{}, false
}

// ChannelByID returns a channel by ID.
func (r *Roster) ChannelByID(id uint32) (Channel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ch, ok := r.channels[id]
	if !ok {
		return Channel{}, false
	}
	return cloneChannel(ch), true
}

// UserBySession returns a user by session ID.
// UsersInChannel returns users currently in channel id.
func (r *Roster) UsersInChannel(id uint32) []User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]User, 0)
	for _, u := range r.users {
		if u.ChannelID == id {
			out = append(out, cloneUser(u))
		}
	}
	return out
}

func (r *Roster) UserBySession(session uint32) (User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[session]
	if !ok {
		return User{}, false
	}
	return cloneUser(u), true
}

func mergeChannelState(dst *Channel, cs *messages.ChannelState) {
	dst.ID = cs.ChannelID
	if cs.HasParent {
		dst.ParentID = cs.Parent
		dst.HasParent = true
	} else if cs.ChannelID == 0 {
		dst.ParentID = 0
		dst.HasParent = false
	}
	if cs.Name != "" {
		dst.Name = cs.Name
	}
	if cs.Description != "" {
		dst.Description = cs.Description
	}
	if cs.Temporary {
		dst.Temporary = true
	}
	// Position is omitted on the wire when 0; apply when explicitly non-zero or
	// when this looks like a full channel snapshot (name present).
	if cs.Position != 0 || cs.Name != "" {
		dst.Position = cs.Position
	}
	if cs.MaxUsers != 0 {
		dst.MaxUsers = cs.MaxUsers
	}
	if len(cs.Links) > 0 {
		dst.Links = append([]uint32(nil), cs.Links...)
	}
	if len(cs.LinksAdd) > 0 || len(cs.LinksRemove) > 0 {
		linkSet := make(map[uint32]struct{}, len(dst.Links)+len(cs.LinksAdd))
		for _, id := range dst.Links {
			linkSet[id] = struct{}{}
		}
		for _, id := range cs.LinksAdd {
			linkSet[id] = struct{}{}
		}
		for _, id := range cs.LinksRemove {
			delete(linkSet, id)
		}
		dst.Links = dst.Links[:0]
		for id := range linkSet {
			dst.Links = append(dst.Links, id)
		}
	}
}

func mergeUserState(dst *User, us *messages.UserState) {
	dst.Session = us.Session
	if us.Has(messages.UserStateSetName) || us.Name != "" {
		dst.Name = us.Name
	}
	if us.Has(messages.UserStateSetUserID) || us.UserID != 0 {
		dst.UserID = us.UserID
	}
	if us.Has(messages.UserStateSetChannelID) || us.ChannelID != 0 {
		dst.ChannelID = us.ChannelID
	}
	// channel_id 0 (root) requires the presence bit
	if us.Has(messages.UserStateSetChannelID) {
		dst.ChannelID = us.ChannelID
	}
	if us.Has(messages.UserStateSetMute) {
		dst.Mute = us.Mute
	} else if us.Mute {
		dst.Mute = true
	}
	if us.Has(messages.UserStateSetDeaf) {
		dst.Deaf = us.Deaf
	} else if us.Deaf {
		dst.Deaf = true
	}
	if us.Has(messages.UserStateSetSuppress) {
		dst.Suppress = us.Suppress
	} else if us.Suppress {
		dst.Suppress = true
	}
	if us.Has(messages.UserStateSetSelfMute) {
		dst.SelfMute = us.SelfMute
	} else if us.SelfMute {
		dst.SelfMute = true
	}
	if us.Has(messages.UserStateSetSelfDeaf) {
		dst.SelfDeaf = us.SelfDeaf
	} else if us.SelfDeaf {
		dst.SelfDeaf = true
	}
	if us.Has(messages.UserStateSetComment) || us.Comment != "" {
		dst.Comment = us.Comment
	}
	if us.Has(messages.UserStateSetHash) || us.Hash != "" {
		dst.Hash = us.Hash
	}
	if us.Has(messages.UserStateSetPrioritySpeaker) {
		dst.PrioritySpeaker = us.PrioritySpeaker
	} else if us.PrioritySpeaker {
		dst.PrioritySpeaker = true
	}
	if us.Has(messages.UserStateSetRecording) {
		dst.Recording = us.Recording
	} else if us.Recording {
		dst.Recording = true
	}
}

func cloneChannel(ch *Channel) Channel {
	out := *ch
	if ch.Links != nil {
		out.Links = append([]uint32(nil), ch.Links...)
	}
	return out
}

func cloneUser(u *User) User {
	return *u
}
