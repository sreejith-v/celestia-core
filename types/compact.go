package types

import (
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
