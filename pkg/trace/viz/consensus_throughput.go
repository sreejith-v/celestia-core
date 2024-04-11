package viz

import (
	"github.com/cometbft/cometbft/pkg/trace/schema"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
)

func PlotConsensusThroughput(root, chainID string, timeoutCommitMS int64) error {
	data, err := LoadBlockSummaries(root, chainID)
	if err != nil {
		return err
	}

	ct := NewConsensusThroughput(chainID, timeoutCommitMS, data)

	p, err := ct.Plot(chainID)
	if err != nil {
		return err
	}

	return Save(p, root, chainID, "consensus_throughput", 16, 9)
}

// ConsensusThroughput stores the data for the consensus throughput
// visualization.
type ConsensusThroughput struct {
	Throughput []float64
	Heights    []float64
}

// NewConsensusThroughput creates a new ConsensusThroughput. experiment is the
// id of the experiment. timeoutCommitMS is the timeout commit in milliseconds.
// This value is subtracted from the block time.
func NewConsensusThroughput(experiment string, timeoutCommitMS int64, data []schema.BlockSummary) ConsensusThroughput {
	ct := ConsensusThroughput{
		Throughput: make([]float64, len(data)-1),
		Heights:    make([]float64, len(data)-1),
	}

	// extract the relative times. note that the times are shifted forward one
	// to account for the block time actually being the time of the last block.
	startTime := data[0].UnixMillisecondTimestamp
	timeCursor := startTime
	for i := 1; i < len(data); i++ {
		ct.Heights[i-1] = float64(data[i-1].Height)

		// find the relative time and divide by 1000 to convert to seconds
		blockTime := (float64(data[i].UnixMillisecondTimestamp-timeoutCommitMS) - float64(timeCursor)) / 1000

		// find the block size in MB. Note we're using the last block to shift
		// the times up by one.
		blockSize := float64(data[i-1].BlockSize) / 1000000
		timeCursor = data[i].UnixMillisecondTimestamp

		ct.Throughput[i-1] = blockSize / blockTime
	}

	return ct
}

func (ct ConsensusThroughput) Plot(experiment string) (*plot.Plot, error) {
	p, err := plot.New()
	if err != nil {
		return nil, err
	}

	p.Title.Text = "Consensus Throughput (Block Size / Block Time): " + experiment
	p.Title.Font.Size = 25
	p.X.Label.Text = "Height (Block Height)"
	p.X.Label.Font.Size = 20
	p.Y.Label.Text = "Throughput (MB/s)"
	p.Y.Label.Font.Size = 20

	pts := make(plotter.XYs, len(ct.Heights))

	for i := range pts {
		pts[i].X = ct.Heights[i]
		pts[i].Y = ct.Throughput[i]
	}

	err = plotutil.AddLinePoints(p, "Throughput (MB/s)", pts)
	if err != nil {
		return nil, err
	}

	return p, nil
}
