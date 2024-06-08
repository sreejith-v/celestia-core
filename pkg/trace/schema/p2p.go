package schema

import (
	"time"

	"github.com/tendermint/tendermint/pkg/trace"
)

// P2PTables returns the list of tables that are used for p2p tracing.
func P2PTables() []string {
	return []string{
		PeersTable,
		PendingBytesTable,
		ReceivedBytesTable,
		MessageProcessingTable,
		GenericTraceTable,
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
	PeerID  string `json:"peer_id"`
	Reactor string `json:"reactor"`
	Action  string `json:"action"`
	Reason  string `json:"reason"`
}

// Table returns the table name for the PeerUpdate struct.
func (p PeerUpdate) Table() string {
	return PeersTable
}

func (p PeerUpdate) GetPeerID() string {
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
func WritePeerUpdate(client trace.Tracer, peerID string, action P2PPeerUpdate, reactor, reason string) {
	client.Write(PeerUpdate{PeerID: peerID, Action: string(action), Reactor: reactor, Reason: reason})
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

func WriteReceivedBytes(client trace.Tracer, peerID string, channel byte, bytes int) {
	client.Write(ReceivedBytes{PeerID: peerID, Channel: channel, Bytes: bytes})
}

const (
	MessageProcessingTable = "proc"
)

type MessageProcessing struct {
	Time       time.Duration `json:"dur,omitempty"`
	Channel    byte          `json:"chan"`
	BufferSize int           `json:"bs"`
}

func (m MessageProcessing) Table() string {
	return MessageProcessingTable
}

func WriteMessageProcessing(client trace.Tracer, channel byte, bufferSize int) {
	client.Write(MessageProcessing{Channel: channel, BufferSize: bufferSize})
}

const (
	GenericTraceTable = "g_t"
)

type GenericTrace struct {
	Status int         `json:"stat"`
	Type   string      `json:"type,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func (q GenericTrace) Table() string {
	return GenericTraceTable
}

func WriteGenericTrace(client trace.Tracer, status int, typ string, data interface{}) {
	client.Write(GenericTrace{Status: status, Type: typ, Data: data})
}

type Times []time.Time
