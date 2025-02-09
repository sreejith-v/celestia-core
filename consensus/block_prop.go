package consensus

import (
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/pkg/trace"
	"github.com/tendermint/tendermint/pkg/trace/schema"
	"github.com/tendermint/tendermint/proto/tendermint/consensus"
	"reflect"

	"github.com/tendermint/tendermint/libs/bits"
	"github.com/tendermint/tendermint/types"
)

// DataChannel the channel ID for the legacy block prop mechanism.
const DataChannel = byte(0x21)

// BlockProp is an interface for the block propagation mechanism.
type BlockProp interface {
	ReceiveEnvelope(e p2p.Envelope)
}

//-------------------------------------

// LegacyBlockProp the existing tendermint block propagation mechanism.
type LegacyBlockProp struct {
	logger      log.Logger
	traceClient trace.Tracer
	// we might find a better way to pass these
	conS             *State
	stopPeerForError func(peer p2p.Peer, reason interface{})
	Metrics          *Metrics
}

var _ BlockProp = (*LegacyBlockProp)(nil)

func (blockProp *LegacyBlockProp) ReceiveEnvelope(e p2p.Envelope) {
	// todo (rach-id): the msg and ps can be refactor somewhere. this is duplicate logic.
	m := e.Message
	if wm, ok := m.(p2p.Wrapper); ok {
		m = wm.Wrap()
	}
	msg, err := MsgFromProto(m.(*consensus.Message))
	if err != nil {
		blockProp.logger.Error("Error decoding message", "src", e.Src, "chId", e.ChannelID, "err", err)
		blockProp.stopPeerForError(e.Src, err)
		return
	}

	if err = msg.ValidateBasic(); err != nil {
		blockProp.logger.Error("Peer sent us invalid msg", "peer", e.Src, "msg", e.Message, "err", err)
		blockProp.stopPeerForError(e.Src, err)
		return
	}
	// Get peer states
	ps, ok := e.Src.Get(types.PeerStateKey).(*PeerState)
	if !ok {
		panic(fmt.Sprintf("Peer %v has no state", e.Src))
	}
	switch msg := msg.(type) {
	case *ProposalMessage:
		ps.SetHasProposal(msg.Proposal)
		blockProp.conS.peerMsgQueue <- msgInfo{msg, e.Src.ID()}
		schema.WriteProposal(
			blockProp.traceClient,
			msg.Proposal.Height,
			msg.Proposal.Round,
			string(e.Src.ID()),
			schema.Download,
		)
	case *ProposalPOLMessage:
		ps.ApplyProposalPOLMessage(msg)
		schema.WriteConsensusState(
			blockProp.traceClient,
			msg.Height,
			msg.ProposalPOLRound,
			string(e.Src.ID()),
			schema.ConsensusPOL,
			schema.Download,
		)
	case *BlockPartMessage:
		ps.SetHasProposalBlockPart(msg.Height, msg.Round, int(msg.Part.Index))
		blockProp.Metrics.BlockParts.With("peer_id", string(e.Src.ID())).Add(1)
		schema.WriteBlockPart(blockProp.traceClient, msg.Height, msg.Round, msg.Part.Index, false, string(e.Src.ID()), schema.Download)
		blockProp.conS.peerMsgQueue <- msgInfo{msg, e.Src.ID()}
	default:
		blockProp.logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
	return
}

//-------------------------------------

// ProposalMessage is sent when a new block is proposed.
type ProposalMessage struct {
	Proposal *types.Proposal
}

// ValidateBasic performs basic validation.
func (m *ProposalMessage) ValidateBasic() error {
	return m.Proposal.ValidateBasic()
}

// String returns a string representation.
func (m *ProposalMessage) String() string {
	return fmt.Sprintf("[Proposal %v]", m.Proposal)
}

//-------------------------------------

// ProposalPOLMessage is sent when a previous proposal is re-proposed.
type ProposalPOLMessage struct {
	Height           int64
	ProposalPOLRound int32
	ProposalPOL      *bits.BitArray
}

// ValidateBasic performs basic validation.
func (m *ProposalPOLMessage) ValidateBasic() error {
	if m.Height < 0 {
		return errors.New("negative Height")
	}
	if m.ProposalPOLRound < 0 {
		return errors.New("negative ProposalPOLRound")
	}
	if m.ProposalPOL.Size() == 0 {
		return errors.New("empty ProposalPOL bit array")
	}
	if m.ProposalPOL.Size() > types.MaxVotesCount {
		return fmt.Errorf("proposalPOL bit array is too big: %d, max: %d", m.ProposalPOL.Size(), types.MaxVotesCount)
	}
	return nil
}

// String returns a string representation.
func (m *ProposalPOLMessage) String() string {
	return fmt.Sprintf("[ProposalPOL H:%v POLR:%v POL:%v]", m.Height, m.ProposalPOLRound, m.ProposalPOL)
}

//-------------------------------------

// BlockPartMessage is sent when gossipping a piece of the proposed block.
type BlockPartMessage struct {
	Height int64
	Round  int32
	Part   *types.Part
}

// ValidateBasic performs basic validation.
func (m *BlockPartMessage) ValidateBasic() error {
	if m.Height < 0 {
		return errors.New("negative Height")
	}
	if m.Round < 0 {
		return errors.New("negative Round")
	}
	if err := m.Part.ValidateBasic(); err != nil {
		return fmt.Errorf("wrong Part: %v", err)
	}
	return nil
}

// String returns a string representation.
func (m *BlockPartMessage) String() string {
	return fmt.Sprintf("[BlockPart H:%v R:%v P:%v]", m.Height, m.Round, m.Part)
}
