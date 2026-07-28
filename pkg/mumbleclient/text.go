package mumbleclient

import (
	"github.com/dchote/go-mumble-server/pkg/mumble/protocol"
	"github.com/dchote/go-mumble-server/pkg/mumble/protocol/messages"
)

// SendTextMessage sends a text message to a channel (channelID 0 = root).
func (c *Client) SendTextMessage(message string, channelID uint32) error {
	return c.WriteMessage(protocol.MessageTextMessage, &messages.TextMessage{
		ChannelID: []uint32{channelID},
		Message:   message,
	})
}

// SendTextMessageToSessions sends a private text message to one or more sessions.
func (c *Client) SendTextMessageToSessions(message string, sessions []uint32) error {
	return c.WriteMessage(protocol.MessageTextMessage, &messages.TextMessage{
		Session: append([]uint32(nil), sessions...),
		Message: message,
	})
}

// SendTreeMessage sends a text message to a channel and its descendants.
func (c *Client) SendTreeMessage(message string, channelID uint32) error {
	return c.WriteMessage(protocol.MessageTextMessage, &messages.TextMessage{
		TreeID:  []uint32{channelID},
		Message: message,
	})
}
