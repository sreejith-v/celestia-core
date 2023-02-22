package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/libs/log"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

const (
	chainID = "test-chain"
	nodeID  = "test-node"
	typeID  = "test-type"
)

func TestServerSuite(t *testing.T) {
	suite.Run(t, NewTestSuite())
}

type ServerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc

	srv *Server
	cli *Client
}

func NewTestSuite() *ServerTestSuite {
	return &ServerTestSuite{}
}

func (s *ServerTestSuite) SetupTest() {
	logger := log.NewTMLogger(os.Stdout)
	s.srv = NewServer(s.T().TempDir(), logger)
	addr := "127.0.0.1:25570"
	go s.srv.Start(addr)
	time.Sleep(100 * time.Millisecond)

	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.cli = NewClient(
		s.ctx,
		logger,
		fmt.Sprintf("%s%s", "http://", addr),
		chainID,
		tmrand.Str(20),
		100,
		1,
	)

	go s.cli.Start()
}

func (s *ServerTestSuite) Test_handleEvent() {
	t := s.T()
	typeID := tmrand.Str(20)
	testData := testData()
	s.cli.QueueEvent(typeID, testData)
	time.Sleep(100 * time.Millisecond)
	_, has := s.srv.getFile(s.cli.chainID, s.cli.nodeID, typeID)
	s.True(has)

	// use the query handler to get the event
	res, err := s.cli.QueryEvents(fmt.Sprintf("%s/%s/%s", s.cli.chainID, s.cli.nodeID, typeID))
	require.NoError(t, err)
	require.Equal(t, 1, len(res))

	var vt TestingEvent
	err = json.Unmarshal(res[0].Data, &vt)
	require.NoError(t, err)
	require.Equal(t, vt, testData)
}

type TestingEvent struct {
	TData string `json:"test_data"`
}

func testData() TestingEvent {
	return TestingEvent{TData: tmrand.Str(20)}
}

func feedClientRandomEvents(t *testing.T, cli *Client) {
	for i := 0; i < 10; i++ {
		cli.QueueEvent(tmrand.Str(20), testData())
	}
}
