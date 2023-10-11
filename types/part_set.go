package types

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/celestiaorg/rsmt2d"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/libs/bits"
	cmtbytes "github.com/tendermint/tendermint/libs/bytes"
	cmtjson "github.com/tendermint/tendermint/libs/json"
	cmtmath "github.com/tendermint/tendermint/libs/math"
	cmtsync "github.com/tendermint/tendermint/libs/sync"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

var (
	ErrPartSetUnexpectedIndex = errors.New("error part set unexpected index")
	ErrPartSetInvalidProof    = errors.New("error part set invalid proof")
	// DefaultCodec is the default codec creator used for data erasure.
	DefaultCodec = rsmt2d.NewLeoRSCodec
)

type Part struct {
	Index uint32            `json:"index"`
	Bytes cmtbytes.HexBytes `json:"bytes"`
	Proof merkle.Proof      `json:"proof"`
}

// ValidateBasic performs basic validation.
func (part *Part) ValidateBasic() error {
	if len(part.Bytes) > int(BlockPartSizeBytes) {
		return fmt.Errorf("too big: %d bytes, max: %d", len(part.Bytes), BlockPartSizeBytes)
	}
	if err := part.Proof.ValidateBasic(); err != nil {
		return fmt.Errorf("wrong Proof: %w", err)
	}
	return nil
}

// String returns a string representation of Part.
//
// See StringIndented.
func (part *Part) String() string {
	return part.StringIndented("")
}

// StringIndented returns an indented Part.
//
// See merkle.Proof#StringIndented
func (part *Part) StringIndented(indent string) string {
	return fmt.Sprintf(`Part{#%v
%s  Bytes: %X...
%s  Proof: %v
%s}`,
		part.Index,
		indent, cmtbytes.Fingerprint(part.Bytes),
		indent, part.Proof.StringIndented(indent+"  "),
		indent)
}

func (part *Part) ToProto() (*cmtproto.Part, error) {
	if part == nil {
		return nil, errors.New("nil part")
	}
	pb := new(cmtproto.Part)
	proof := part.Proof.ToProto()

	pb.Index = part.Index
	pb.Bytes = part.Bytes
	pb.Proof = *proof

	return pb, nil
}

func PartFromProto(pb *cmtproto.Part) (*Part, error) {
	if pb == nil {
		return nil, errors.New("nil part")
	}

	part := new(Part)
	proof, err := merkle.ProofFromProto(&pb.Proof)
	if err != nil {
		return nil, err
	}
	part.Index = pb.Index
	part.Bytes = pb.Bytes
	part.Proof = *proof

	return part, part.ValidateBasic()
}

//-------------------------------------

type PartSetHeader struct {
	Total    uint32            `json:"total"`
	Hash     cmtbytes.HexBytes `json:"hash"`
	ByteSize uint64            `json:"byte_size"`
}

// String returns a string representation of PartSetHeader.
//
// 1. total number of parts
// 2. first 6 bytes of the hash
func (psh PartSetHeader) String() string {
	return fmt.Sprintf("%v:%X", psh.Total, cmtbytes.Fingerprint(psh.Hash))
}

func (psh PartSetHeader) IsZero() bool {
	return psh.Total == 0 && len(psh.Hash) == 0
}

func (psh PartSetHeader) Equals(other PartSetHeader) bool {
	return psh.Total == other.Total && bytes.Equal(psh.Hash, other.Hash) && psh.ByteSize == other.ByteSize
}

// ValidateBasic performs basic validation.
func (psh PartSetHeader) ValidateBasic() error {
	// Hash can be empty in case of POLBlockID.PartSetHeader in Proposal.
	if err := ValidateHash(psh.Hash); err != nil {
		return fmt.Errorf("wrong Hash: %w", err)
	}
	// TODO: Uncomment and update tests
	// // total must be even since the data is being erasure encoded
	// if psh.Total%2 != 0 {
	// 	return fmt.Errorf("total must be even: %d", psh.Total)
	// }
	// if psh.ByteSize > uint64(psh.Total)*uint64(BlockPartSizeBytes) {
	// 	return fmt.Errorf("byte size cannot be greater than %d", uint64(psh.Total)*uint64(BlockPartSizeBytes))
	// }
	// if psh.ByteSize < uint64(psh.Total-1)*uint64(BlockPartSizeBytes) {
	// 	return fmt.Errorf("byte size cannot be less than %d", uint64(psh.Total-1)*uint64(BlockPartSizeBytes))
	// }
	return nil
}

// ToProto converts PartSetHeader to protobuf
func (psh *PartSetHeader) ToProto() cmtproto.PartSetHeader {
	if psh == nil {
		return cmtproto.PartSetHeader{}
	}

	return cmtproto.PartSetHeader{
		Total:    psh.Total,
		Hash:     psh.Hash,
		ByteSize: psh.ByteSize,
	}
}

// FromProto sets a protobuf PartSetHeader to the given pointer
func PartSetHeaderFromProto(ppsh *cmtproto.PartSetHeader) (*PartSetHeader, error) {
	if ppsh == nil {
		return nil, errors.New("nil PartSetHeader")
	}
	psh := new(PartSetHeader)
	psh.Total = ppsh.Total
	psh.Hash = ppsh.Hash
	psh.ByteSize = ppsh.ByteSize

	return psh, psh.ValidateBasic()
}

//-------------------------------------

type PartSet struct {
	total uint32
	hash  []byte

	mtx           cmtsync.Mutex
	parts         []*Part
	partsBitArray *bits.BitArray
	count         uint32
	// a count of the total size (in bytes). Used to ensure that the
	// part set doesn't exceed the maximum block bytes
	byteSize int64

	codec rsmt2d.Codec
}

// Returns an immutable, full PartSet from the data bytes.
// The data bytes are split into "partSize" chunks, and merkle tree computed.
// CONTRACT: partSize is greater than zero.
func NewPartSetFromData(data []byte, partSize uint32) *PartSet {
	codec := DefaultCodec()
	// divide data into parts.
	total := (uint32(len(data)) + partSize - 1) / partSize
	// multiply by 2 to accomodate the erasure data
	extendedTotal := 2 * total
	parts := make([]*Part, extendedTotal)
	partsBytes := make([][]byte, extendedTotal)
	partsBitArray := bits.NewBitArray(int(extendedTotal))
	for i := uint32(0); i < total; i++ {
		part := &Part{
			Index: i,
			Bytes: data[i*partSize : cmtmath.MinInt(len(data), int((i+1)*partSize))],
		}
		parts[i] = part
		partsBytes[i] = part.Bytes
		partsBitArray.SetIndex(int(i), true)
	}

	// pad the last part bytes for erasure encoding
	lastPart := parts[total-1]
	lastPartBytes, _ := zeroPadIfNecessary(lastPart.Bytes, int(partSize))
	lastPart.Bytes = lastPartBytes
	partsBytes[total-1] = lastPartBytes
	parts[total-1] = lastPart

	// erasure encode the data
	erasureData, err := codec.Encode(partsBytes[:total])
	if err != nil {
		// TODO get rid of or bubble this panic
		panic(err)
	}

	for i := total; i < extendedTotal; i++ {
		part := &Part{
			Index: i,
			Bytes: erasureData[i-total],
		}
		parts[i] = part
		partsBytes[i] = part.Bytes
		partsBitArray.SetIndex(int(i), true)
	}

	// Compute merkle proofs
	root, proofs := merkle.ProofsFromByteSlices(partsBytes)
	for i := uint32(0); i < extendedTotal; i++ {
		parts[i].Proof = *proofs[i]
	}
	return &PartSet{
		total:         extendedTotal,
		hash:          root,
		parts:         parts,
		partsBitArray: partsBitArray,
		count:         extendedTotal,
		byteSize:      int64(len(data)),
		codec:         codec,
	}
}

// Returns an empty PartSet ready to be populated.
func NewPartSetFromHeader(header PartSetHeader) *PartSet {
	return &PartSet{
		total:         header.Total,
		hash:          header.Hash,
		parts:         make([]*Part, header.Total),
		partsBitArray: bits.NewBitArray(int(header.Total)),
		count:         0,
		byteSize:      int64(header.ByteSize),
		codec:         DefaultCodec(),
	}
}

func (ps *PartSet) Header() PartSetHeader {
	if ps == nil {
		return PartSetHeader{}
	}
	return PartSetHeader{
		Total:    ps.total,
		Hash:     ps.hash,
		ByteSize: uint64(ps.byteSize),
	}
}

func (ps *PartSet) HasHeader(header PartSetHeader) bool {
	if ps == nil {
		return false
	}
	return ps.Header().Equals(header)
}

func (ps *PartSet) BitArray() *bits.BitArray {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return ps.partsBitArray.Copy()
}

func (ps *PartSet) Hash() []byte {
	if ps == nil {
		return merkle.HashFromByteSlices(nil)
	}
	return ps.hash
}

func (ps *PartSet) HashesTo(hash []byte) bool {
	if ps == nil {
		return false
	}
	return bytes.Equal(ps.hash, hash)
}

func (ps *PartSet) Count() uint32 {
	if ps == nil {
		return 0
	}
	return ps.count
}

func (ps *PartSet) ByteSize() int64 {
	if ps == nil {
		return 0
	}
	return ps.byteSize
}

func (ps *PartSet) Total() uint32 {
	if ps == nil {
		return 0
	}
	return ps.total
}

func (ps *PartSet) AddPart(part *Part) (bool, error) {
	if ps == nil {
		return false, nil
	}
	fmt.Println("adding part", part.Index)
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	// Invalid part index
	if part.Index >= ps.total {
		return false, ErrPartSetUnexpectedIndex
	}

	// If part already exists, return false.
	if ps.parts[part.Index] != nil {
		return false, nil
	}

	// Check hash proof
	if part.Proof.Verify(ps.Hash(), part.Bytes) != nil {
		return false, ErrPartSetInvalidProof
	}

	// Add part
	ps.parts[part.Index] = part
	ps.partsBitArray.SetIndex(int(part.Index), true)
	ps.count++
	return true, nil
}

func (ps *PartSet) GetPart(index int) *Part {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return ps.parts[index]
}

func (ps *PartSet) IsComplete() (bool, error) {
	if ps.count >= ps.total/2 {
		err := ps.Fill()
		if err != nil {
			return false, err
		}
		fmt.Println("part set is complete", ps.count, ps.total)
		return true, nil
	}
	fmt.Println("part set is incomplete")
	return false, nil
}

// Fill will use the erasure encoded data to fill in the missing parts.
func (ps *PartSet) Fill() error {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	if ps.count < ps.total/2 {
		return fmt.Errorf("cannot fill incomplete part set")
	}
	if ps.count == ps.total {
		return nil
	}

	// extract the data from the parts
	data := make([][]byte, ps.total)
	for i := uint32(0); i < ps.total; i++ {
		if ps.parts[i] == nil {
			continue
		}
		data[i] = ps.parts[i].Bytes
	}
	// decode the data
	decodedData, err := ps.codec.Decode(data)
	if err != nil {
		return err
	}

	_, proofs := merkle.ProofsFromByteSlices(decodedData)

	// fill in the parts from the decoded data
	for i := uint32(0); i < ps.total; i++ {
		if ps.parts[i] != nil {
			continue
		}

		ps.parts[i] = &Part{
			Index: i,
			Bytes: decodedData[i],
			Proof: *proofs[i],
		}
		ps.partsBitArray.SetIndex(int(i), true)
		ps.count++
	}

	return nil
}

func (ps *PartSet) BlockBytes() ([]byte, error) {
	complete, err := ps.IsComplete()
	if !complete {
		return nil, fmt.Errorf("cannot read block from incomplete part set")
	}
	if err != nil {
		return nil, err
	}
	bz := make([]byte, 0, ps.ByteSize())
	for i := uint32(0); i < ((ps.Total() / 2) - 1); i++ {
		bz = append(bz, ps.parts[i].Bytes...)
	}
	bz = append(bz, ps.parts[(ps.Total()/2)-1].Bytes[:ps.lastPartNonPadding()]...)
	return bz, nil
}

// LastPartPadding returns the number of bytes in the last part that are not
// padding.
func (ps *PartSet) lastPartNonPadding() int {
	cursor := int(ps.ByteSize()) % int(BlockPartSizeBytes)
	if cursor == 0 {
		return int(BlockPartSizeBytes)
	}
	return cursor
}

func (ps *PartSet) ReadBlock() (*Block, error) {
	bz, err := ps.BlockBytes()
	if err != nil {
		return nil, err
	}

	pbb := new(cmtproto.Block)
	err = proto.Unmarshal(bz, pbb)
	if err != nil {
		return nil, err
	}

	return BlockFromProto(pbb)
}

// func (ps *PartSet) GetReader() (io.Reader, error) {
// 	complete, err := ps.IsComplete()
// 	if !complete {
// 		return nil, errors.New("cannot get reader from incomplete part set")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}
// 	return NewPartSetReader(ps.parts), nil
// }

type PartSetReader struct {
	i      int
	parts  []*Part
	reader *bytes.Reader
}

func NewPartSetReader(parts []*Part) *PartSetReader {
	return &PartSetReader{
		i:      0,
		parts:  parts,
		reader: bytes.NewReader(parts[0].Bytes),
	}
}

// func (psr *PartSetReader) Read(p []byte) (n int, err error) {
// 	readerLen := psr.reader.Len()
// 	if readerLen >= len(p) {
// 		return psr.reader.Read(p)
// 	} else if readerLen > 0 {
// 		n1, err := psr.Read(p[:readerLen])
// 		if err != nil {
// 			return n1, err
// 		}
// 		n2, err := psr.Read(p[readerLen:])
// 		return n1 + n2, err
// 	}

// 	psr.i++
// 	if psr.i >= len(psr.parts) {
// 		return 0, io.EOF
// 	}
// 	psr.reader = bytes.NewReader(psr.parts[psr.i].Bytes)
// 	return psr.Read(p)
// }

// StringShort returns a short version of String.
//
// (Count of Total)
func (ps *PartSet) StringShort() string {
	if ps == nil {
		return "nil-PartSet"
	}
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return fmt.Sprintf("(%v of %v)", ps.Count(), ps.Total())
}

func (ps *PartSet) MarshalJSON() ([]byte, error) {
	if ps == nil {
		return []byte("{}"), nil
	}

	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	return cmtjson.Marshal(struct {
		CountTotal    string         `json:"count/total"`
		PartsBitArray *bits.BitArray `json:"parts_bit_array"`
	}{
		fmt.Sprintf("%d/%d", ps.Count(), ps.Total()),
		ps.partsBitArray,
	})
}

// zeroPadIfNecessary pads the share with trailing zero bytes if the provided
// share has fewer bytes than width. Returns the share unmodified if the
// len(share) is greater than or equal to width.
func zeroPadIfNecessary(share []byte, width int) (padded []byte, bytesOfPadding int) {
	oldLen := len(share)
	if oldLen >= width {
		return share, 0
	}

	missingBytes := width - oldLen
	padByte := []byte{0}
	padding := bytes.Repeat(padByte, missingBytes)
	share = append(share, padding...)
	return share, missingBytes
}
