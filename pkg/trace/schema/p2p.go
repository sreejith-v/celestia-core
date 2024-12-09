package schema

import (
	"github.com/tendermint/tendermint/pkg/trace"
	"time"
)

// P2PTables returns the list of tables that are used for p2p tracing.
func P2PTables() []string {
	return []string{
		PeersTable,
		PendingBytesTable,
		ReceivedBytesTable,
		NetworkPacketsTable,
		// "conn_buf",
		TimedSentBytesTable,
		TimedReceivedBytesTable,
		MsgLatencyTable,
	}
}

const (
	// PeersTable is the name of the table that stores the p2p peer
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
	NetworkPacketsTable = "network_packets"
)

type NetworkPacket struct {
	PeerID  string `json:"peer_id"`
	Channel byte   `json:"channel"`
}

func (s NetworkPacket) Table() string {
	return NetworkPacketsTable
}

func WriteNetworkPacket(client trace.Tracer, peerID string, channel byte) {
	client.Write(NetworkPacket{PeerID: peerID, Channel: channel})
}

type ChannelPacketTracer struct {
	PeerID  string
	Channel byte
	Client  trace.Tracer
}

func (c ChannelPacketTracer) Trace() {
	WriteNetworkPacket(c.Client, c.PeerID, c.Channel)
}

type ConnectionBuffer struct {
	Size int `json:"size"`
}

func (c ConnectionBuffer) Table() string {
	return "conn_buf"
}

func WriteConnBuffer(client trace.Tracer, size int) {
	client.Write(ConnectionBuffer{Size: size})
}

const (
	TimedSentBytesTable = "timed_sent_bytes"
)

type TimedSentBytes struct {
	PeerID    string    `json:"peer_id"`
	Channel   byte      `json:"channel"`
	Bytes     int       `json:"bytes"`
	Time      time.Time `json:"time"`
	IPAddress string    `json:"ip_address"`
}

func (s TimedSentBytes) Table() string {
	return TimedSentBytesTable
}

func WriteTimedSentBytes(client trace.Tracer, peerID string, ipAddr string, channel byte, bytes int, t time.Time) {
	client.Write(TimedSentBytes{PeerID: peerID, Channel: channel, Bytes: bytes, Time: t, IPAddress: ipAddr})
}

const (
	TimedReceivedBytesTable = "timed_received_bytes"
)

type TimedReceivedBytes struct {
	PeerID    string    `json:"peer_id"`
	Channel   byte      `json:"channel"`
	Bytes     int       `json:"bytes"`
	Time      time.Time `json:"time"`
	IPAddress string    `json:"ip_address"`
}

func (s TimedReceivedBytes) Table() string {
	return TimedReceivedBytesTable
}

func WriteTimedReceivedBytes(client trace.Tracer, peerID string, ipAddr string, channel byte, bytes int, t time.Time) {
	client.Write(TimedReceivedBytes{PeerID: peerID, Channel: channel, Bytes: bytes, Time: t, IPAddress: ipAddr})
}

const (
	MsgLatencyTable = "msg_latency"
)

type MsgLatency struct {
	PeerID      string `json:"peer_id"`
	Channel     byte   `json:"channel"`
	Bytes       int    `json:"bytes"`
	ReceiveTime string `json:"receive_time"`
	SendTime    string `json:"send_time"`
	IPAddress   string `json:"ip_address"`
}

func (s MsgLatency) Table() string {
	return MsgLatencyTable
}

func WriteMsgLatency(client trace.Tracer, peerID string, ipAddr string, channel byte, bytes int, sendTime string, receiveTime string) {
	client.Write(MsgLatency{PeerID: peerID, Channel: channel, Bytes: bytes, SendTime: sendTime, ReceiveTime: receiveTime, IPAddress: ipAddr})
}
