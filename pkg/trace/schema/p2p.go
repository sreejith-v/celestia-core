package schema

import (
	"github.com/tendermint/tendermint/pkg/trace"
)

// P2PTables returns the list of tables that are used for p2p
// tracing.
func P2PTables() []string {
	return []string{
		ChannelTable,
	}
}

// Schema constants for the consensus round state tracing database.
const (
	// ChannelTable is the name of the table that stores the channel gossiping
	// traces. Follows this schema:
	//
	// | time | bytes (sent or received) | transfer_type | channel_id |
	ChannelTable = "channel_table"

	// BytesFieldKey is the name of the field that stores the bytes sent or
	// received. The value is an integer.
	BytesFieldKey = "bytes"

	// ChannelIDKey is the name of the field that stores the channel ID. The
	// value is a byte.
	ChannelIDKey = "channel_id"
)

// WriteChannelTrace writes a tracing point for a tx using the predetermined
// schema for consensus state tracing. This is used to create a table in the
// following schema:
//
// | time | bytes (sent or received) | transfer_type | channel_id |
func WriteChannelTrace(client *trace.Client, bytes int, transferType string, channelID byte) {
	client.WritePoint(RoundStateTable, map[string]interface{}{
		BytesFieldKey:        bytes,
		TransferTypeFieldKey: transferType,
		ChannelIDKey:         channelID,
	})
}
