package viz

import (
	"os"
	"path"
	"path/filepath"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

func Save(p *plot.Plot, root, chainID, plotName string, width, height float64) error {
	path := path.Join(root, chainID, "plots")
	err := os.MkdirAll(path, os.ModePerm)
	filepath := filepath.Join(path, plotName+".png")
	if err != nil {
		return err
	}
	return p.Save(vg.Length(width)*vg.Inch, vg.Length(height)*vg.Inch, filepath)
}
