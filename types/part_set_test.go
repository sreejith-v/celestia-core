package types

import (
	"encoding/hex"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto/merkle"
	cmtrand "github.com/tendermint/tendermint/libs/rand"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

const (
	testPartSize = 65536 // 64KB ...  4096 // 4KB
)

func TestBasicPartSet(t *testing.T) {
	// Construct random data of size partSize * 100
	nParts := 100
	data := cmtrand.Bytes(testPartSize * nParts)
	partSet := NewPartSetFromData(data, testPartSize)

	assert.NotEmpty(t, partSet.Hash())
	assert.EqualValues(t, nParts, partSet.Total())
	assert.Equal(t, nParts, partSet.BitArray().Size())
	assert.True(t, partSet.HashesTo(partSet.Hash()))
	assert.True(t, partSet.IsComplete())
	assert.EqualValues(t, nParts, partSet.Count())
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
	assert.EqualValues(t, nParts, partSet2.Total())
	assert.EqualValues(t, nParts*testPartSize, partSet.ByteSize())
	assert.True(t, partSet2.IsComplete())

	// Reconstruct data, assert that they are equal.
	data2Reader := partSet2.GetReader()
	data2, err := io.ReadAll(data2Reader)
	require.NoError(t, err)

	assert.Equal(t, data, data2)
}

func TestWrongProof(t *testing.T) {
	// Construct random data of size partSize * 100
	data := cmtrand.Bytes(testPartSize * 100)
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
			data := cmtrand.Bytes(testPartSize * 100)
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
			data := cmtrand.Bytes(testPartSize * 100)
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

var (
	Val1Prop = "1281030ad0020a08080b10c4b6bcc801120a636f6d706163742d32321801220c0899d6e6af0610ccc59494022a0212003220e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b8553a203d96b7d238e7e0456f6af8e7cdf0a67bd6cf9c2089ecb559c659dcaa1f8803534220ae6420fc9ae285312bc6fefdb788141552826d9865da97aa5a1a02ad76a676544a20ae6420fc9ae285312bc6fefdb788141552826d9865da97aa5a1a02ad76a676545220614e3fdce077a12dc945c5dd8c7c14c777e56ab66c7f44ef1595f121c89437595a20e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b8556220e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b8556a20e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855721444a724c895430366b2b1165165a12a8d28e0f02a1224280132203d96b7d238e7e0456f6af8e7cdf0a67bd6cf9c2089ecb559c659dcaa1f8803531a0022041a0212001a2408011a20e811d08ae8e0ed1bd23019ee7b4c957b6d6adbd1039cd8e44cc1fda389267925"
	Val1Comp = "12ff020ad0020a08080b10c4b6bcc801120a636f6d706163742d32321801220c0899d6e6af0610ccc59494022a0212003220e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b8553a203d96b7d238e7e0456f6af8e7cdf0a67bd6cf9c2089ecb559c659dcaa1f8803534220ae6420fc9ae285312bc6fefdb788141552826d9865da97aa5a1a02ad76a676544a20ae6420fc9ae285312bc6fefdb788141552826d9865da97aa5a1a02ad76a676545220614e3fdce077a12dc945c5dd8c7c14c777e56ab66c7f44ef1595f121c89437595a20e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b8556220e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b8556a20e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855721444a724c895430366b2b1165165a12a8d28e0f02a122232203d96b7d238e7e0456f6af8e7cdf0a67bd6cf9c2089ecb559c659dcaa1f8803531a0022041a0212001a2408011a202ff9fcfee1b0b84ae23bdf644e09c875c5c2112159ba94c3417a36b4fee2b37c"
)

func TestDebug(t *testing.T) {
	parts := []string{
		Val1Prop,
		Val1Comp,
	}

	blocks := make([]*Block, 0, len(parts))

	for _, sp := range parts {
		p := &cmtproto.Part{}
		d, err := hex.DecodeString(sp)
		require.NoError(t, err)
		err = p.Unmarshal(d)
		require.NoError(t, err)

		pp, err := PartFromProto(p)
		require.NoError(t, err)

		r := NewPartSetReader([]*Part{pp})

		bz, err := io.ReadAll(r)
		require.NoError(t, err)

		var protoBlock cmtproto.Block
		err = protoBlock.Unmarshal(bz)
		require.NoError(t, err)

		bb, err := BlockFromProto(&protoBlock)
		require.NoError(t, err)

		blocks = append(blocks, bb)

		fmt.Println(bb.SquareSize)
		fmt.Println("------------------------------------------------")
	}

	require.Equal(t, blocks[0], blocks[1])
}
