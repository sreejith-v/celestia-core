package schema

import (
	"github.com/tendermint/tendermint/pkg/trace"
	"github.com/tendermint/tendermint/types"
)

// ExecutionTables returns the list of tables that are used for execution
// tracing.
func ExecutionTables() []string {
	return []string{
		BlockTable,
	}
}

const (
	// BlockTable is the name of the table that stores metadata about consensus blocks.
	// following schema:
	//
	//  | time  | height | timestamp | tx_count | square_size | block_size | proposer | last_commit_round |
	BlockTable = "execution_block"

	// UnixMillisecondTimestampFieldKey is the name of the field that stores the timestamp in
	// the last commit in unix milliseconds.
	UnixMillisecondTimestampFieldKey = "unix_millisecond_timestamp"

	// TxCountFieldKey is the name of the field that stores the number of
	// transactions in the block.
	TxCountFieldKey = "tx_count"

	// SquareSizeFieldKey is the name of the field that stores the square size
	// of the block. SquareSize is the number of shares in a single row or
	// column of the origianl data square.
	SquareSizeFieldKey = "square_size"

	// BlockSizeFieldKey is the name of the field that stores the size of
	// the block data in bytes.
	BlockSizeFieldKey = "block_size"

	// ProposerFieldKey is the name of the field that stores the proposer of
	// the block.
	ProposerFieldKey = "proposer"

	// LastCommitRoundFieldKey is the name of the field that stores the round
	// of the last commit.
	LastCommitRoundFieldKey = "last_commit_round"

	// ExecutionTimeFieldKey is the name of the field that stores the execution
	// time of the block in milliseconds.
	ExecutionTimeFieldKey = "execution_time"
)

func WriteBlock(client *trace.Client, block *types.Block, executionTime int64) {
	client.WritePoint(BlockTable, map[string]interface{}{
		HeightFieldKey:                   block.Height,
		UnixMillisecondTimestampFieldKey: block.Time.UnixMilli(),
		TxCountFieldKey:                  len(block.Data.Txs),
		SquareSizeFieldKey:               block.SquareSize,
		BlockSizeFieldKey:                block.Size(),
		ProposerFieldKey:                 block.ProposerAddress.String(),
		LastCommitRoundFieldKey:          block.LastCommit.Round,
		ExecutionTimeFieldKey:            executionTime,
	})
}
