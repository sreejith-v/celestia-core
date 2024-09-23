package cat

import (
	"time"

	"github.com/tendermint/tendermint/types"
)

// wrappedTx defines a wrapper around a raw transaction with additional metadata
// that is used for indexing. With the exception of the map of peers who have
// seen this transaction, this struct should never be modified
type wrappedTx struct {
	// these fields are immutable
	tx        types.Tx    // the original transaction data
	key       types.TxKey // the transaction hash
	height    int64       // height when this transaction was initially checked (for expiry)
	timestamp time.Time   // time when transaction was entered (for TTL)
	gasWanted int64       // app: gas required to execute this transaction
	priority  int64       // app: priority value for this transaction
	sender    string      // app: assigned sender label
	evictable bool        // whether this transaction can be evicted from the mempool. This is false when the transaction is a part of a proposed block nolint:lll

	selfTx bool // keeps track of if this tx originated from this node. If so, then it will not be included in a block. This is a hack.
	// temporary var only used for sorting when reaping
	seenCount int
}

func newWrappedTx(tx types.Tx, key types.TxKey, height, gasWanted, priority int64, sender string, self bool) *wrappedTx {
	return &wrappedTx{
		tx:        tx,
		key:       key,
		height:    height,
		timestamp: time.Now().UTC(),
		gasWanted: gasWanted,
		priority:  priority,
		sender:    sender,
		evictable: true,
		selfTx:    self,
	}
}

// Size reports the size of the raw transaction in bytes.
func (w *wrappedTx) size() int64 { return int64(len(w.tx)) }
