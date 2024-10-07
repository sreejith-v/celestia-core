package schema

import (
	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/pkg/trace"
)

// MempoolTables returns the list of tables for mempool tracing.
func MempoolTables() []string {
	return []string{
		MempoolTxTable,
		MempoolPeerStateTable,
		MempoolRecoveryTable,
		MempoolRejectedTable,
	}
}

// Schema constants for the mempool_tx table
const (
	// MempoolTxTable is the tracing "measurement" (aka table) for the mempool
	// that stores tracing data related to gossiping transactions.
	MempoolTxTable = "mempool_tx"
)

// MemPoolTx describes the schema for the "mempool_tx" table.
type MempoolTx struct {
	TxHash       string       `json:"tx_hash"`
	Peer         string       `json:"peer"`
	Size         int          `json:"size"`
	TransferType TransferType `json:"transfer_type"`
	ValPrio      uint64       `json:"valprio"`
}

// Table returns the table name for the MempoolTx struct.
func (m MempoolTx) Table() string {
	return MempoolTxTable
}

// WriteMempoolTx writes a tracing point for a tx using the predetermined
// schema for mempool tracing.
func WriteMempoolTx(client trace.Tracer, peer string, txHash []byte, size int, valprio uint64, transferType TransferType) {
	// this check is redundant to what is checked during client.Write, although it
	// is an optimization to avoid allocations from the map of fields.
	if !client.IsCollecting(MempoolTxTable) {
		return
	}
	client.Write(MempoolTx{
		TxHash:       bytes.HexBytes(txHash).String(),
		Peer:         peer,
		Size:         size,
		TransferType: transferType,
		ValPrio:      valprio,
	})
}

const (
	// MempoolPeerState is the tracing "measurement" (aka table) for the mempool
	// that stores tracing data related to mempool state, specifically
	// the gossipping of "SeenTx" and "WantTx".
	MempoolPeerStateTable = "mempool_peer_state"
)

type MempoolStateUpdateType string

const (
	SeenTx  MempoolStateUpdateType = "SeenTx"
	WantTx  MempoolStateUpdateType = "WantTx"
	Unknown MempoolStateUpdateType = "Unknown"
)

// MempoolPeerState describes the schema for the "mempool_peer_state" table.
type MempoolPeerState struct {
	Peer         string                 `json:"peer"`
	StateUpdate  MempoolStateUpdateType `json:"state_update"`
	TxHash       string                 `json:"tx_hash"`
	TransferType TransferType           `json:"transfer_type"`
	Validator    string                 `json:"validator,omitempty"` // originating valiator for seenTxs
}

// Table returns the table name for the MempoolPeerState struct.
func (m MempoolPeerState) Table() string {
	return MempoolPeerStateTable
}

// WriteMempoolPeerState writes a tracing point for the mempool state using
// the predetermined schema for mempool tracing.
func WriteMempoolPeerState(
	client trace.Tracer,
	peer string,
	stateUpdate MempoolStateUpdateType,
	txHash []byte,
	transferType TransferType,
	validator string,
) {
	// this check is redundant to what is checked during client.Write, although it
	// is an optimization to avoid allocations from creating the map of fields.
	if !client.IsCollecting(MempoolPeerStateTable) {
		return
	}
	client.Write(MempoolPeerState{
		Peer:         peer,
		StateUpdate:  stateUpdate,
		TransferType: transferType,
		TxHash:       bytes.HexBytes(txHash).String(),
		Validator:    validator,
	})
}

const (
	MempoolRecoveryTable = "mempool_recovery"
	MempoolRejectedTable = "mempool_rejected"
)

type MempoolRecovery struct {
	Missing   int      `json:"missing"`
	Recovered int      `json:"recovered"`
	Total     int      `json:"total"`
	TimeTaken uint64   `json:"time_taken"`
	Hashes    []string `json:"hashes"`
}

func (m MempoolRecovery) Table() string {
	return MempoolRecoveryTable
}

func WriteMempoolRecoveryStats(
	client trace.Tracer,
	missing int,
	recovered int,
	total int,
	timeTaken uint64,
	hashes []string,
) {
	client.Write(MempoolRecovery{
		Missing:   missing,
		Recovered: recovered,
		Total:     total,
		TimeTaken: timeTaken,
		Hashes:    hashes,
	})
}

type MempoolRejected struct {
	Code   uint64 `json:"code"`
	Error  string `json:"error"`
	TxHash string `json:"tx_hash"`
	Peer   string `json:"peer"`
}

func (m MempoolRejected) Table() string {
	return MempoolRejectedTable
}

func WriteMempoolRejected(
	client trace.Tracer,
	peer string,
	txHash []byte,
	code uint64,
	err error,
) {
	// this check is redundant to what is checked during client.Write, although it
	// is an optimization to avoid allocations from creating the map of fields.
	if !client.IsCollecting(MempoolRejectedTable) {
		return
	}
	var errS string
	if err != nil {
		errS = err.Error()
	}
	client.Write(MempoolRejected{
		Peer:   peer,
		Code:   code,
		TxHash: bytes.HexBytes(txHash).String(),
		Error:  errS,
	})
}
