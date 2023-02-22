package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/libs/log"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

const (
	chainID = "test-chain"
	nodeID  = "test-node"
	typeID  = "test-type"

	addr = "127.0.0.1:25570"
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
	tID := tmrand.Str(20)
	testData := testData()
	s.cli.QueueEvent(tID, testData)
	time.Sleep(100 * time.Millisecond)

	// use the query handler to get the event
	query := fmt.Sprintf("%s/%s/%s", s.cli.chainID, s.cli.nodeID, tID)
	res, err := s.cli.QueryEvents(query)
	require.NoError(t, err)
	require.Equal(t, 1, len(res))

	var vt TestingEvent
	err = json.Unmarshal(res[0].Data, &vt)
	require.NoError(t, err)
	require.Equal(t, vt, testData)
}

// TestManyBatches tests that the server can handle many batches at once. It
// writes and then reads 1_000_000 events twice.
func (s *ServerTestSuite) TestManyBatches() {
	t := s.T()
	// t.Skip()
	// create many batches and send them to the server
	wg := &sync.WaitGroup{}
	clientCount := 1000
	eventCount := 1000
	nodeIDs := make([]string, clientCount)
	tID := tmrand.Str(20)
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		nID := tmrand.Str(20)
		nodeIDs[i] = nID
		go func() {
			defer wg.Done()
			cli := NewClient(
				s.ctx,
				log.NewNopLogger(),
				fmt.Sprintf("%s%s", "http://", addr),
				chainID,
				nID,
				100,
				1,
			)
			defer cli.Stop()
			err := cli.SendBatch(generateEvents(eventCount, chainID, nID, tID))
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	// query all of the events for a specific node
	for _, nID := range nodeIDs {
		wg.Add(1)
		go func(nnID string) {
			defer wg.Done()
			cli := NewClient(
				s.ctx,
				log.NewNopLogger(),
				fmt.Sprintf("%s%s", "http://", addr),
				chainID,
				nnID,
				100,
				1,
			)
			evs, err := cli.QueryEvents(fmt.Sprintf("%s/%s/%s", s.cli.chainID, nnID, "*"))
			assert.NoError(t, err)
			assert.Equal(t, eventCount, len(evs))
		}(nID)
	}
	wg.Wait()

	// query all of the events at once
	evs, err := s.cli.QueryEvents(fmt.Sprintf("%s/%s/%s", s.cli.chainID, "*", ""))
	require.NoError(t, err)
	require.Equal(t, clientCount*eventCount, len(evs))
}

type TestingEvent struct {
	TData string `json:"test_data"`
}

func testData() TestingEvent {
	return TestingEvent{TData: tmrand.Str(20)}
}

func generateEvents(count int, chainID, nodeID, typ string) []Event {
	events := make([]Event, count)
	for i := 0; i < count; i++ {
		events[i] = Event{
			ChainID: chainID,
			NodeID:  nodeID,
			Type:    typ,
			Time:    time.Now(),
			Data:    mustMarshal(testData()),
		}
	}
	return events
}
