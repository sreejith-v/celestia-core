package cat

import (
	"sync"

	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/types"
)

type wantState struct {
	mtx   *sync.RWMutex
	wants map[types.TxKey]map[p2p.ID]struct{}
}

func NewWantState() *wantState {
	return &wantState{
		wants: make(map[types.TxKey]map[p2p.ID]struct{}),
		mtx:   &sync.RWMutex{},
	}
}

func (f *wantState) GetWants(tx types.TxKey) (map[p2p.ID]struct{}, bool) {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	out, has := f.wants[tx]
	return out, has
}

func (f *wantState) Add(tx types.TxKey, peer p2p.ID) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if _, exists := f.wants[tx]; !exists {
		f.wants[tx] = make(map[p2p.ID]struct{})
	}
	f.wants[tx][peer] = struct{}{}
}

func (f *wantState) Delete(tx types.TxKey, peer p2p.ID) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	ws, has := f.wants[tx]
	if !has {
		return
	}
	_, has = ws[peer]
	if !has {
		return
	}
	delete(ws, peer)
	f.wants[tx] = ws
	if len(f.wants[tx]) == 0 {
		delete(f.wants, tx)
	}
}
