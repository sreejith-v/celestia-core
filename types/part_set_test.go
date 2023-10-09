package types

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/merkle"
	cmtrand "github.com/tendermint/tendermint/libs/rand"
)

const (
	testPartSize = BlockPartSizeBytes // 64KB ...  4096 // 4KB
)

func TestBasicPartSet(t *testing.T) {
	// Construct random data of size partSize * 100
	nParts := uint32(100)
	data := cmtrand.Bytes(int(testPartSize) * int(nParts))
	partSet := NewPartSetFromData(data, testPartSize)

	assert.NotEmpty(t, partSet.Hash())
	assert.EqualValues(t, nParts*2, partSet.Total())
	assert.Equal(t, int(nParts*2), partSet.BitArray().Size())
	assert.True(t, partSet.HashesTo(partSet.Hash()))
	complete, err := partSet.IsComplete()
	assert.True(t, complete)
	require.NoError(t, err)
	assert.EqualValues(t, uint32(nParts*2), partSet.Count())
	assert.EqualValues(t, testPartSize*nParts, partSet.ByteSize())

	// Test adding parts to a new partSet.
	partSet2 := NewPartSetFromHeader(partSet.Header())

	assert.True(t, partSet2.HasHeader(partSet.Header()))
	for i := 0; i < int(partSet.Total()); i++ {
		part := partSet.GetPart(i)
		// t.Logf("\n%v", part)
		added, err := partSet2.AddPart(part)
		if !added || err != nil {
			t.Errorf("failed to add part %v, error: %v", i, err)
		}
	}
	// adding part with invalid index
	added, err := partSet2.AddPart(&Part{Index: 10000})
	assert.False(t, added)
	assert.Error(t, err)
	// adding existing part
	added, err = partSet2.AddPart(partSet2.GetPart(0))
	assert.False(t, added)
	assert.Nil(t, err)

	assert.Equal(t, partSet.Hash(), partSet2.Hash())
	assert.EqualValues(t, nParts*2, partSet2.Total())
	assert.EqualValues(t, nParts*testPartSize, partSet.ByteSize())
	complete, err = partSet2.IsComplete()
	assert.True(t, complete)
	require.NoError(t, err)

	// Reconstruct data, assert that they are equal.
	data2, err := partSet2.BlockBytes()
	require.NoError(t, err)

	assert.Equal(t, data, data2)
}

func TestWrongProof(t *testing.T) {
	// Construct random data of size partSize * 100
	data := cmtrand.Bytes(int(testPartSize) * 100)
	partSet := NewPartSetFromData(data, testPartSize)

	// Test adding a part with wrong data.
	partSet2 := NewPartSetFromHeader(partSet.Header())

	// Test adding a part with wrong trail.
	part := partSet.GetPart(0)
	part.Proof.Aunts[0][0] += byte(0x01)
	added, err := partSet2.AddPart(part)
	if added || err == nil {
		t.Errorf("expected to fail adding a part with bad trail.")
	}

	// Test adding a part with wrong bytes.
	part = partSet.GetPart(1)
	part.Bytes[0] += byte(0x01)
	added, err = partSet2.AddPart(part)
	if added || err == nil {
		t.Errorf("expected to fail adding a part with bad bytes.")
	}
}

func TestPartSetHeaderValidateBasic(t *testing.T) {
	testCases := []struct {
		testName              string
		malleatePartSetHeader func(*PartSetHeader)
		expectErr             bool
	}{
		{"Good PartSet", func(psHeader *PartSetHeader) {}, false},
		{"Invalid Hash", func(psHeader *PartSetHeader) { psHeader.Hash = make([]byte, 1) }, true},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			data := cmtrand.Bytes(int(testPartSize) * 100)
			ps := NewPartSetFromData(data, testPartSize)
			psHeader := ps.Header()
			tc.malleatePartSetHeader(&psHeader)
			assert.Equal(t, tc.expectErr, psHeader.ValidateBasic() != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestPartValidateBasic(t *testing.T) {
	testCases := []struct {
		testName     string
		malleatePart func(*Part)
		expectErr    bool
	}{
		{"Good Part", func(pt *Part) {}, false},
		{"Too big part", func(pt *Part) { pt.Bytes = make([]byte, BlockPartSizeBytes+1) }, true},
		{"Too big proof", func(pt *Part) {
			pt.Proof = merkle.Proof{
				Total:    1,
				Index:    1,
				LeafHash: make([]byte, 1024*1024),
			}
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			data := cmtrand.Bytes(int(testPartSize) * 100)
			ps := NewPartSetFromData(data, testPartSize)
			part := ps.GetPart(0)
			tc.malleatePart(part)
			assert.Equal(t, tc.expectErr, part.ValidateBasic() != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestParSetHeaderProtoBuf(t *testing.T) {
	testCases := []struct {
		msg     string
		ps1     *PartSetHeader
		expPass bool
	}{
		{"success empty", &PartSetHeader{}, true},
		{
			"success",
			&PartSetHeader{Total: 1, Hash: []byte("hash")}, true,
		},
	}

	for _, tc := range testCases {
		protoBlockID := tc.ps1.ToProto()

		psh, err := PartSetHeaderFromProto(&protoBlockID)
		if tc.expPass {
			require.Equal(t, tc.ps1, psh, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

func TestPartProtoBuf(t *testing.T) {
	proof := merkle.Proof{
		Total:    1,
		Index:    1,
		LeafHash: cmtrand.Bytes(32),
	}
	testCases := []struct {
		msg     string
		ps1     *Part
		expPass bool
	}{
		{"failure empty", &Part{}, false},
		{"failure nil", nil, false},
		{
			"success",
			&Part{Index: 1, Bytes: cmtrand.Bytes(32), Proof: proof}, true,
		},
	}

	for _, tc := range testCases {
		proto, err := tc.ps1.ToProto()
		if tc.expPass {
			require.NoError(t, err, tc.msg)
		}

		p, err := PartFromProto(proto)
		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, tc.ps1, p, tc.msg)
		}
	}
}

func TestPartSet_BlockBytes(t *testing.T) {
	tests := []struct {
		name     string
		dataSize int
	}{
		{
			name:     "data of size 1000 bytes",
			dataSize: 1000,
		},
		{
			name:     "data of size BlockPartSizeBytes + 1",
			dataSize: int(BlockPartSizeBytes) + 1,
		},
		{
			name:     "data of size BlockPartSizeBytes*2 + 1",
			dataSize: int(BlockPartSizeBytes*2) + 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := cmtrand.Bytes(tt.dataSize)
			partSet := NewPartSetFromData(data, testPartSize)
			blockBytes, err := partSet.BlockBytes()
			require.NoError(t, err)
			assert.Equal(t, data, blockBytes)
		})
	}
}

func TestPartSet_LastPartPadding(t *testing.T) {
	bzSize := 1000
	data := cmtrand.Bytes(bzSize)
	partSet := NewPartSetFromData(data, testPartSize)
	assert.Equal(t, int(BlockPartSizeBytes-uint32(bzSize)), partSet.LastPartPadding())
}

func TestPartSet_ReadBlock(t *testing.T) {
	h := cmtrand.Int63()
	c1 := randCommit(time.Now())
	b1 := MakeBlock(h, makeData([]Tx{Tx([]byte{1})}), &Commit{Signatures: []CommitSig{}}, []Evidence{})
	b1.ProposerAddress = cmtrand.Bytes(crypto.AddressSize)

	evidenceTime := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	evi := NewMockDuplicateVoteEvidence(h, evidenceTime, "block-test-chain")
	b2 := MakeBlock(h, makeData([]Tx{Tx([]byte{1})}), c1, []Evidence{evi})
	b2.ProposerAddress = cmtrand.Bytes(crypto.AddressSize)
	b2.Evidence.ByteSize()

	b3 := MakeBlock(h, makeData([]Tx{}), c1, []Evidence{})
	b3.ProposerAddress = cmtrand.Bytes(crypto.AddressSize)
	testCases := []struct {
		msg      string
		b1       *Block
		expPass  bool
		expPass2 bool
	}{
		// {"nil block", nil, false, false},
		{"b1", b1, true, true},
		{"b2", b2, true, true},
		{"b3", b3, true, true},
	}
	for _, tc := range testCases {
		pb, err := tc.b1.ToProto()
		if tc.expPass {
			require.NoError(t, err, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}

		// marshal the proto block
		bz, err := proto.Marshal(pb)
		require.NoError(t, err, tc.msg)

		// create a part set from the block bytes
		partSet := NewPartSetFromData(bz, testPartSize)
		block, err := partSet.ReadBlock()
		if tc.expPass2 {
			require.NoError(t, err, tc.msg)
			require.EqualValues(t, tc.b1.Header, block.Header, tc.msg)
			require.EqualValues(t, tc.b1.Data, block.Data, tc.msg) // todo
			require.EqualValues(t, tc.b1.Evidence.Evidence, block.Evidence.Evidence, tc.msg)
			require.EqualValues(t, *tc.b1.LastCommit, *block.LastCommit, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}
