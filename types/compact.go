package types

import (
	bits "github.com/tendermint/tendermint/libs/bits"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

func (b *Block) CompactBlock() cmtproto.CompactBlock {
	metaTxs := make([]cmtproto.TxMetadata, len(b.Data.Txs))
	for i, tx := range b.Data.Txs {
		metaTxs[i] = cmtproto.TxMetadata{
			Hash: tx.Hash(),
		}
	}
	return cmtproto.CompactBlock{
		TxMetadata: metaTxs,
	}
}

func FullHalves(size int) cmtproto.Haves {
	haves := bits.NewBitArray(size)
	for i := 0; i < size; i++ {
		haves.SetIndex(i, true)
	}

	return cmtproto.Haves{Haves: *haves.ToProto()}
}

/*
We need to be able to create haves and wants. Given a compact block, we want xxxxx. given a compact block, we
*/
