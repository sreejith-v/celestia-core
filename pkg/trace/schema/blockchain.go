package schema

import (
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/pkg/trace"
)

// ConsensusTables returns the list of tables that are used for consensus
// tracing.
func BlockchainTables() []string {
	return []string{
		BlocksyncCommsTable,
	}
}

// Schema constants for the consensus round state tracing database.
const (
	// BlocksyncCommsTable is the name of the table that stores requests and
	// responses from the blockchain reactor.
	//
	// | time | peer | msg type | transfer type |
	BlocksyncCommsTable = "blocksync_comms"

	// BlocksyncCommsTypeFieldKey is the name of the field that stores the
	// type of the message.
	BlocksyncCommsTypeFieldKey = "msg_type"

	// StatusRequestFieldValue is the value of the type field for a status
	// request message.
	StatusRequestFieldValue = "status_request"

	// StatusResponseFieldValue is the value of the type field for a status
	// response message.
	StatusResponseFieldValue = "status_response"

	// BlockRequestFieldValue is the value of the type field for a block
	// request message.
	BlockRequestFieldValue = "block_request"

	// BlockResponseFieldValue is the value of the type field for a block
	// response message.
	BlockResponseFieldValue = "block_response"

	// NoBlockResponseFieldValue is the value of the type field for a no block
	// response message.
	NoBlockResponseFieldValue = "no_block_response"
)

func WriteBlocksyncComms(
	client *trace.Client,
	peer p2p.ID,
	msgType string,
	transferType string,
) {
	client.WritePoint(BlocksyncCommsTable, map[string]interface{}{
		PeerFieldKey:               peer,
		BlocksyncCommsTypeFieldKey: msgType,
		TransferTypeFieldKey:       transferType,
	})
}
