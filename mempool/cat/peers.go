package cat

import (
	"fmt"

	tmsync "github.com/tendermint/tendermint/libs/sync"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/p2p"
)

const firstPeerID = mempool.UnknownPeerID + 1

// mempoolIDs is a thread-safe map of peer IDs to shorter uint16 IDs used by the Reactor for tracking peer
// messages and peer state such as what transactions peers have seen
type mempoolIDs struct {
	mtx       tmsync.RWMutex
	peerMap   map[p2p.ID]uint16   // quick lookup table for peer ID to short ID
	nextID    uint16              // assumes that a node will never have over 65536 active peers
	activeIDs map[uint16]p2p.Peer // used to check if a given peerID key is used, the value doesn't matter
	knownIDs  map[uint16]p2p.ID
}

func newMempoolIDs() *mempoolIDs {
	return &mempoolIDs{
		peerMap:   make(map[p2p.ID]uint16),
		activeIDs: make(map[uint16]p2p.Peer),
		knownIDs:  make(map[uint16]p2p.ID),
		nextID:    firstPeerID, // reserve unknownPeerID(0) for mempoolReactor.BroadcastTx
	}
}

func (ids *mempoolIDs) getPeerFromID(id p2p.ID) (p2p.Peer, bool) {
	ids.mtx.RLock()
	defer ids.mtx.RUnlock()
	for _, peer := range ids.activeIDs {
		if peer.ID() == id {
			return peer, true
		}
	}
	return nil, false
}

// ReserveForPeer searches for the next unused ID and assigns it to the
// peer.
func (ids *mempoolIDs) ReserveForPeer(peer p2p.Peer) {

	// if _, ok := ids.peerMap[peer.ID()]; ok {
	// 	panic("duplicate peer added to mempool")
	// }

	curID := ids.getIDSafe(peer.ID())

	ids.mtx.Lock()
	defer ids.mtx.Unlock()

	ids.activeIDs[curID] = peer
}

// getIDSafe ensures that all known ids are accountted for
func (ids *mempoolIDs) getIDSafe(id p2p.ID) uint16 {
	ids.mtx.Lock()
	defer ids.mtx.Unlock()
	var newid uint16
	var seen bool

	for sid, pid := range ids.knownIDs {
		if id == pid {
			newid = sid
			seen = true
			break
		}
	}
	if !seen {
		newid = ids.nextPeerID()
	}

	ids.peerMap[id] = newid
	ids.knownIDs[newid] = id

	return newid
}

// nextPeerID returns the next unused peer ID to use.
// This assumes that ids's mutex is already locked.
func (ids *mempoolIDs) nextPeerID() uint16 {
	if (len(ids.activeIDs) + len(ids.knownIDs)) == mempool.MaxActiveIDs {
		panic(fmt.Sprintf("node has maximum %d active IDs and wanted to get one more", mempool.MaxActiveIDs))
	}

	_, idExists := ids.knownIDs[ids.nextID]
	for idExists {
		ids.nextID++
		_, idExists = ids.knownIDs[ids.nextID]
	}
	curID := ids.nextID
	ids.nextID++
	return curID
}

// Reclaim returns the ID reserved for the peer back to unused pool.
func (ids *mempoolIDs) Reclaim(peerID p2p.ID) uint16 {
	ids.mtx.Lock()
	defer ids.mtx.Unlock()

	removedID, ok := ids.peerMap[peerID]
	if ok {
		delete(ids.activeIDs, removedID)
		// delete(ids.peerMap, peerID)
		return removedID
	}
	return 0
}

// GetIDForPeer returns the shorthand ID reserved for the peer.
func (ids *mempoolIDs) GetIDForPeer(peerID p2p.ID) uint16 {
	ids.mtx.RLock()
	id, exists := ids.peerMap[peerID]
	ids.mtx.RUnlock()
	if !exists {
		id = ids.getIDSafe(peerID)
	}
	return id
}

// GetPeer returns the peer for the given shorthand ID.
func (ids *mempoolIDs) GetPeer(id uint16) p2p.Peer {
	ids.mtx.RLock()
	defer ids.mtx.RUnlock()

	return ids.activeIDs[id]
}

// GetAll returns all active peers.
func (ids *mempoolIDs) GetAll() map[uint16]p2p.Peer {
	ids.mtx.RLock()
	defer ids.mtx.RUnlock()

	// make a copy of the map.
	peers := make(map[uint16]p2p.Peer, len(ids.activeIDs))
	for id, peer := range ids.activeIDs {
		peers[id] = peer
	}
	return peers
}

// Len returns the number of active peers.
func (ids *mempoolIDs) Len() int {
	ids.mtx.RLock()
	defer ids.mtx.RUnlock()

	return len(ids.activeIDs)
}
