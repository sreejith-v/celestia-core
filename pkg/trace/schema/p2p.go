package schema

import "github.com/cometbft/cometbft/pkg/trace"

// P2PTables returns the list of tables that are used for p2p tracing.
func P2PTables() []string {
	return []string{
		PeersTable,
		PendingBytesTable,
		ReceivedBytesTable,
	}
}

const (
	// PeerUpdateTable is the name of the table that stores the p2p peer
	// updates.
	PeersTable = "peers"
)

// P2PPeerUpdate is an enum that represents the different types of p2p
// trace data.
type P2PPeerUpdate string

const (
	// PeerJoin is the action for when a peer is connected.
	PeerJoin P2PPeerUpdate = "connect"
	// PeerDisconnect is the action for when a peer is disconnected.
	PeerDisconnect P2PPeerUpdate = "disconnect"
)

// PeerUpdate describes schema for the "peer_update" table.
type PeerUpdate struct {
	PeerID string `json:"peer_id"`
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// Table returns the table name for the PeerUpdate struct.
func (p PeerUpdate) Table() string {
	return PeersTable
}

// GetPeerID returns the peer id for the PeerUpdate struct.
func (p PeerUpdate) GetPeer() string {
	return p.PeerID
}

func (p PeerUpdate) GetAction() string {
	return p.Action
}

func (p PeerUpdate) GetReason() string {
	return p.Reason
}

// WritePeerUpdate writes a tracing point for a peer update using the predetermined
// schema for p2p tracing.
func WritePeerUpdate(client trace.Tracer, peerID string, action P2PPeerUpdate, reason string) {
	client.Write(PeerUpdate{PeerID: peerID, Action: string(action), Reason: reason})
}

const (
	PendingBytesTable = "pending_bytes"
)

type PendingBytes struct {
	PeerID string       `json:"peer_id"`
	Bytes  map[byte]int `json:"bytes"`
}

func (s PendingBytes) Table() string {
	return PendingBytesTable
}

func (pb PendingBytes) GetPeer() string {
	return pb.PeerID
}

func WritePendingBytes(client trace.Tracer, peerID string, bytes map[byte]int) {
	client.Write(PendingBytes{PeerID: peerID, Bytes: bytes})
}

const (
	ReceivedBytesTable = "received_bytes"
)

type ReceivedBytes struct {
	PeerID  string `json:"peer_id"`
	Channel byte   `json:"channel"`
	Bytes   int    `json:"bytes"`
}

func (s ReceivedBytes) Table() string {
	return ReceivedBytesTable
}

func (rb ReceivedBytes) GetPeer() string {
	return rb.PeerID
}

func (rb ReceivedBytes) GetChannel() byte {
	return rb.Channel
}

func WriteReceivedBytes(client trace.Tracer, peerID string, channel byte, bytes int) {
	client.Write(ReceivedBytes{PeerID: peerID, Channel: channel, Bytes: bytes})
}
