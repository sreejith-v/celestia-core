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

	ctx             context.Context
	logger          log.Logger
	serverAddr      string
	c               *http.Client
	nodeID, chainID string

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
		Timeout: 10 * time.Second,
	}
	cli := &Client{
		ctx:        ctx,
		serverAddr: serverAddr,
		nodeID:     nodeID,
		chainID:    chainID,
		queue:      make(chan goEvent, queueSize),
		logger:     logger,
		c:          httpcli,
		batchSize:  batchSize,
	}
	cli.BaseService = *service.NewBaseService(nil, "RemoteClient", cli)
	return cli
}

func (c *Client) QueueEvent(eventType string, data interface{}) {
	c.queue <- newGoEvent(eventType, data)
}

func (c *Client) Start() error {
	pendingBatch := make([]Event, 0, c.batchSize)
	for {
		if len(pendingBatch) >= c.batchSize {
			fmt.Println("sending batched events")
			err := c.SendBatch(c.ctx, pendingBatch)
			if err != nil {
				fmt.Println("failed to send batched events", err)
				c.logger.Error("failed to send batched events", "error", err.Error())
			} else {
				pendingBatch = make([]Event, 0, c.batchSize)
			}
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
			fmt.Println("appending event to queue")
			pendingBatch = append(pendingBatch, ev)
		}
	}

}

func (c *Client) CreateServerURL(srvMethod, query string) string {
	if query == "" {
		return fmt.Sprintf("%s/%s", c.serverAddr, srvMethod)
	}
	return fmt.Sprintf("%s%s?%s", c.serverAddr, srvMethod, query)
}
