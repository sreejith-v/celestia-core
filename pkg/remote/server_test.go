package remote

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_loadFiles(t *testing.T) {
	dir := t.TempDir()
	chainID0, nodeID0, typ0 := "chainID-0", "nodeID-0", "type-0.json"
	chainID1, nodeID1, typ1 := "chainID-1", "nodeID-1", "type-1.json"

	ev := Event{
		ChainID: chainID0,
		NodeID:  nodeID0,
		Type:    typ0,
		Time:    time.Now(),
		Data:    []byte("[{\"foo\":\"bar\"}]"),
	}

	jsonBytes, err := json.Marshal(ev)
	require.NoError(t, err)

	paths := [][]string{
		{dir, chainID0, nodeID0, typ0},
		{dir, chainID0, nodeID1, typ0},
		{dir, chainID0, nodeID0, typ1},
		{dir, chainID0, nodeID1, typ1},
		{dir, chainID1, nodeID1, typ1},
	}

	for _, path := range paths {
		err = os.MkdirAll(filepath.Join(path[0], path[1], path[2]), os.ModePerm)
		require.NoError(t, err)
		fpath := filepath.Join(path...)
		f, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		assert.NoError(t, err)
		_, err = f.Write(jsonBytes)
		assert.NoError(t, err)
		_ = f.Close()
	}

	files := loadFiles(dir)
	require.Len(t, files, 2)
	require.NotNil(t, files[chainID0][nodeID0][typ0])
	require.NotNil(t, files[chainID0][nodeID1][typ0])
	require.NotNil(t, files[chainID0][nodeID0][typ1])
	require.NotNil(t, files[chainID0][nodeID1][typ1])
	require.NotNil(t, files[chainID1][nodeID1][typ1])
}
