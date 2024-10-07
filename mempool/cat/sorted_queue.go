package cat

import (
	"container/heap"
	"context"
	"sync"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/pkg/trace"
	"github.com/tendermint/tendermint/pkg/trace/schema"
	protomem "github.com/tendermint/tendermint/proto/tendermint/mempool"
)

type TxPrioritizor struct {
	ctx         context.Context
	rootCancel  context.CancelFunc
	logger      log.Logger
	traceClient trace.Tracer

	mtx             *sync.RWMutex
	outgoing        map[p2p.ID]*SortedQueue
	outgoingCancels map[p2p.ID]context.CancelFunc
}

func NewTxPrioritizor(logger log.Logger, traceClient trace.Tracer) *TxPrioritizor {
	ctx, cancel := context.WithCancel(context.Background())

	tp := &TxPrioritizor{
		outgoing:        make(map[p2p.ID]*SortedQueue),
		outgoingCancels: make(map[p2p.ID]context.CancelFunc),
		mtx:             &sync.RWMutex{},
		ctx:             ctx,
		rootCancel:      cancel,
		logger:          logger,
		traceClient:     traceClient,
	}

	return tp
}

func (tp *TxPrioritizor) AddPeer(peer p2p.Peer) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()

	sq := NewSortedQueue()
	tp.outgoing[peer.ID()] = sq

	ctx, cancel := context.WithCancel(tp.ctx)
	tp.outgoingCancels[peer.ID()] = cancel

	go func(ctx context.Context, peer p2p.Peer, sq *SortedQueue) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sq.ready:
				otx, has := sq.Pop()
				if !has || otx == nil {
					panic("nil tx popped when we were supposed to have txs to send")
				}

				wtx := otx.(*wrappedTx) // btw, this is only required cause the heap package doesn't yet use generics

				if p2p.SendEnvelopeShim(peer, p2p.Envelope{ //nolint:staticcheck
					ChannelID: mempool.MempoolChannel,
					Message:   &protomem.Txs{Txs: [][]byte{wtx.tx}, Valprio: wtx.valPrio},
				}, tp.logger) {
					// memR.mempool.PeerHasTx(peerID, txKey)
					schema.WriteMempoolTx(
						tp.traceClient,
						string(peer.ID()),
						wtx.key[:],
						len(wtx.tx),
						wtx.valPrio,
						schema.Upload,
					)
				} else {
					// reinsert the tx if we are unable to send it.
					sq.Insert(wtx)
				}

			}
		}
	}(ctx, peer, sq)
}

func (tp *TxPrioritizor) RemovePeer(peer p2p.ID) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	if cancel, has := tp.outgoingCancels[peer]; has {
		cancel()
	}
}

func (tp *TxPrioritizor) Send(peer p2p.Peer, wtx *wrappedTx) {
	tp.mtx.RLock()
	sq, has := tp.outgoing[peer.ID()]
	if !has {
		tp.logger.Error("yo there's no outgoing peer")
	}
	tp.mtx.RUnlock()
	sq.Insert(wtx)
}

// Ordered is an interface that requires a LessThan method for comparison
type Ordered interface {
	LessThan(other Ordered) bool
}

// SortedQueue represents a thread-safe priority queue for elements implementing the Ordered interface
type SortedQueue struct {
	mu    sync.RWMutex  // Mutex for concurrency control
	heap  priorityQueue // Internal min-heap for sorted queue
	ready chan struct{}
}

// NewSortedQueue initializes a new SortedQueue
func NewSortedQueue() *SortedQueue {
	sq := &SortedQueue{
		heap:  priorityQueue{},
		ready: make(chan struct{}, 6000),
	}
	heap.Init(&sq.heap)
	return sq
}

// Insert adds a new element to the sorted queue and maintains order
func (sq *SortedQueue) Insert(value Ordered) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	heap.Push(&sq.heap, value)
	sq.ready <- struct{}{} // the popper that things can be popped
}

// Pop removes and returns the smallest element from the queue
func (sq *SortedQueue) Pop() (Ordered, bool) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	if sq.heap.Len() == 0 {
		return nil, false // Queue is empty
	}
	item := heap.Pop(&sq.heap).(Ordered)
	return item, true
}

// Peek returns the smallest element without removing it
func (sq *SortedQueue) Peek() (Ordered, bool) {
	sq.mu.RLock()
	defer sq.mu.RUnlock()

	if sq.heap.Len() == 0 {
		return nil, false // Queue is empty
	}
	return sq.heap[0], true
}

// Len returns the number of elements in the queue
func (sq *SortedQueue) Len() int {
	sq.mu.RLock()
	defer sq.mu.RUnlock()

	return sq.heap.Len()
}

// priorityQueue is the internal heap structure that implements heap.Interface
type priorityQueue []Ordered

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Use the LessThan method for comparison
	return pq[i].LessThan(pq[j])
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x any) {
	*pq = append(*pq, x.(Ordered))
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}
