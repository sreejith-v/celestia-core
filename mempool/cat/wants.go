package cat

import (
	"sync"

	"github.com/tendermint/tendermint/types"
)

type wantState struct {
	mtx   *sync.RWMutex
	wants map[types.TxKey]map[uint16]struct{}
}

func NewWantState() *wantState {
	return &wantState{
		wants: make(map[types.TxKey]map[uint16]struct{}),
		mtx:   &sync.RWMutex{},
	}
}

func (f *wantState) GetWants(tx types.TxKey) (map[uint16]struct{}, bool) {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	out, has := f.wants[tx]
	return out, has
}

func (f *wantState) Add(tx types.TxKey, peer uint16) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if _, exists := f.wants[tx]; !exists {
		f.wants[tx] = make(map[uint16]struct{})
	}
	f.wants[tx][peer] = struct{}{}
}
