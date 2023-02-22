package remote

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
)

type Client struct {
	service.BaseService

	ctx    context.Context
	logger log.Logger
	c      *http.Client

	// configs
	serverAddr      string
	nodeID, chainID string

	// maxAttempts is the maximum number of attempts to send a batch of events
	// before we begine dropping events.
	maxAttempts int

	queue     chan goEvent
	batchSize int
}

func NewClient(
	ctx context.Context,
	logger log.Logger,
	serverAddr string,
	chainID,
	nodeID string,
	queueSize,
	batchSize int,
) *Client {
	httpcli := &http.Client{
		Timeout: 4 * time.Second, // todo: make this configurable
	}
	cli := &Client{
		ctx:         ctx,
		serverAddr:  serverAddr,
		nodeID:      nodeID,
		chainID:     chainID,
		queue:       make(chan goEvent, queueSize),
		logger:      logger,
		c:           httpcli,
		batchSize:   batchSize,
		maxAttempts: 10,
	}
	cli.BaseService = *service.NewBaseService(nil, "RemoteClient", cli)
	return cli
}

// QueueEvent queues an event to be sent to the server. It does not block.
func (c *Client) QueueEvent(eventType string, data interface{}) {
	if len(c.queue) == cap(c.queue) {
		c.logger.Error("event queue is full, dropping event")
		return
	}
	c.queue <- newGoEvent(eventType, data)
}

// Start implements service.Service. It starts the event loop that sends events.
// If the server is down or not responding, the client will begin dropping
// events after the configured maxAttempts.
func (c *Client) Start() error {
	pendingBatch := make([]Event, 0, c.batchSize)
	// keep track of how many times we've tried to send a batch. If we've tried
	// too many times, start throwing away data to avoid blocking.
	attempts := 0
	for {
		if len(pendingBatch) >= c.batchSize {
			attempts++
			err := c.SendBatch(pendingBatch)
			if err != nil {
				c.logger.Error("failed to send batched events", "error", err.Error())
			} else {
				attempts = 0
				pendingBatch = make([]Event, 0, c.batchSize)
			}
		}
		if attempts >= c.maxAttempts {
			c.logger.Error("failed to send batched events too many times, dropping events")
			pendingBatch = make([]Event, 0, c.batchSize)
			attempts = 0
		}
		select {
		case <-c.ctx.Done():
			// there's a possibility that we have some pending events still in
			// the queue, but we don't want to block here, so we just drop them
			return nil
		case gev := <-c.queue:
			ev, err := gev.Event()
			if err != nil {
				c.logger.Error(err.Error())
			}
			ev.ChainID = c.chainID
			ev.NodeID = c.nodeID
			pendingBatch = append(pendingBatch, ev)
		}
	}

}

// CreateServerURL creates a URL for the given server method and query. If no
// query is desired, then provide an empty query struct.
func (c *Client) CreateServerURL(srvMethod string, q Query) string {
	base := fmt.Sprintf("%s%s", c.serverAddr, srvMethod)
	if q.IsEmpty() {
		return base
	}
	return fmt.Sprintf("%s%s", base, q.URLValues())
}
