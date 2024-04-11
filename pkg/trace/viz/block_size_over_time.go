package viz

import (
	"github.com/cometbft/cometbft/pkg/trace/schema"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func PlotBlockSizeOverTime(root, chainID string) error {
	data, err := LoadBlockSummaries(root, chainID)
	if err != nil {
		return err
	}

	ct := NewBlockSizeOverTime(chainID, data)

	p, err := ct.Plot(chainID)
	if err != nil {
		return err
	}

	return Save(p, root, chainID, "consensus_throughput", 16, 9)
}

// BlockSizeOverTime stores the data for the block size over time visualization.
type BlockSizeOverTime struct {
	Sizes   []float64
	Times   []float64
	Heights []float64
}

// NewBlockSizeOverTime creates a new BlockSizeOverTime. experiment is the id of
// the experiment.
func NewBlockSizeOverTime(experiment string, data []schema.BlockSummary) BlockSizeOverTime {
	ct := BlockSizeOverTime{
		Sizes:   make([]float64, len(data)-1),
		Times:   make([]float64, len(data)-1),
		Heights: make([]float64, len(data)-1),
	}

	// extract the relative times. note that the times are shifted forward one
	// to account for the block time actually being the time of the last block.
	startTime := data[0].UnixMillisecondTimestamp
	timeCursor := startTime
	for i := 1; i < len(data); i++ {
		ct.Heights[i-1] = float64(data[i-1].Height)

		// find the relative time and divide by 1000 to convert to seconds
		blockTime := (float64(data[i].UnixMillisecondTimestamp) - float64(timeCursor)) / 1000

		// find the block size in MB. Note we're using the last block to shift
		// the times up by one.
		blockSize := float64(data[i-1].BlockSize) / 1000000
		timeCursor = data[i].UnixMillisecondTimestamp

		ct.Sizes[i-1] = blockSize
		ct.Times[i-1] = blockTime
		ct.Heights[i-1] = float64(data[i-1].Height)
	}

	return ct
}

func (ct BlockSizeOverTime) Plot(experiment string) (*plot.Plot, error) {
	p, err := plot.New()
	if err != nil {
		return nil, err
	}

	p.Title.Text = "Block Size vs Block Time: " + experiment
	p.Title.Font.Size = 25
	p.X.Label.Text = "Height (Block Height)"
	p.X.Label.Font.Size = 20
	p.Y.Label.Text = "Block Size (MB)"
	p.Y.Label.Font.Size = 20

	sizePts := make(plotter.XYs, len(ct.Heights))
	timePts := make(plotter.XYs, len(ct.Heights))

	for i := range ct.Heights {
		sizePts[i].X = ct.Heights[i]
		sizePts[i].Y = ct.Sizes[i]
		timePts[i].X = ct.Heights[i]
		timePts[i].Y = ct.Times[i]
	}

	sizeL, err := plotter.NewLine(sizePts)
	if err != nil {
		return nil, err
	}
	sizeL.LineStyle.Width = vg.Points(1)

	timeL, err := plotter.NewLine(timePts)
	if err != nil {
		return nil, err
	}

	p.Add(sizeL, timeL)

	return p, nil
}
