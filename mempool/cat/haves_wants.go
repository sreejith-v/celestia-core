package cat

import (
	"github.com/tendermint/tendermint/libs/bits"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/types"
)

func (txmp *TxPool) InclusionCheck(cblock cmtproto.CompactBlock) (cmtproto.Haves, cmtproto.Wants) {
	haves := bits.NewBitArray(len(cblock.TxMetadata))
	wants := bits.NewBitArray(len(cblock.TxMetadata))
	for i, tx := range cblock.TxMetadata {
		if txmp.Has(types.TxKey(tx.Hash)) {
			haves.SetIndex(i, true)
			wants.SetIndex(i, false)
			continue
		}
		haves.SetIndex(i, false)
		wants.SetIndex(i, true)
	}
	return cmtproto.Haves{
		Haves: *haves.ToProto(),
	}, cmtproto.Wants{Wants: *wants.ToProto()}
}
