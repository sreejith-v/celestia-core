package remote

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	chainID0, nodeID0, typ0 = "chainID-0", "nodeID-0", "type-0.json"
	chainID1, nodeID1, typ1 = "chainID-1", "nodeID-1", "type-1.json"
)

func Test_loadFiles(t *testing.T) {
	dir := t.TempDir()

	setupInitTestFiles(t, dir)

	files := loadFiles(dir)
	require.Len(t, files, 2)
	require.NotNil(t, files[chainID0][nodeID0][typ0])
	require.NotNil(t, files[chainID0][nodeID1][typ0])
	require.NotNil(t, files[chainID0][nodeID0][typ1])
	require.NotNil(t, files[chainID0][nodeID1][typ1])
	require.NotNil(t, files[chainID1][nodeID1][typ1])
}

func TestQueryFiles(t *testing.T) {
	dir := t.TempDir()
	setupInitTestFiles(t, dir)
	s := NewServer(dir, log.NewNopLogger())

	type test struct {
		chainID string
		nodeID  string
		typ     string
		err     bool
		want    int
	}
	tests := []test{
		{chainID0, nodeID0, "type-0", false, 1},
		{chainID0, nodeID1, "type-0", false, 1},
		{chainID0, nodeID0, "type-1", false, 1},
		{chainID0, nodeID1, "type-1", false, 1},
		{chainID1, nodeID1, "type-1", false, 1},
		{chainID0, nodeID0, typ0, true, 0}, // has json on the end, so it should fail
		{QueryAll, "", "", false, 5},
		{chainID0, QueryAll, "", false, 4},
		{chainID1, QueryAll, "", false, 1},
		{chainID0, nodeID0, QueryAll, false, 2},
	}

	for _, tt := range tests {
		t.Run(tt.chainID+tt.nodeID+tt.typ, func(t *testing.T) {
			fs, err := s.QueryFiles(tt.chainID, tt.nodeID, tt.typ)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Len(t, fs, tt.want)
		})
	}
}

func setupInitTestFiles(t *testing.T, dir string) {
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
}

func TestReadWriteJsonLinesFile(t *testing.T) {
	// open a file
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.json")
	lf, err := OpenLabeledFile(fpath)
	require.NoError(t, err)
	defer lf.Close()

	// create some events
	count := 3
	evs := make([]Event, count)
	for i := 0; i < count; i++ {
		evs[i] = Event{
			ChainID: chainID0,
			NodeID:  nodeID0,
			Type:    typ0,
			Time:    time.Now(),
			Data:    mustMarshal(testData()),
		}
	}

	// write the events to the file
	err = WriteJsonLinesFile(lf, evs)
	require.NoError(t, err)

	// read the file
	sevs, err := ReadJsonLinesFile[Event](lf)
	require.NoError(t, err)
	require.Len(t, sevs, 3)
}
