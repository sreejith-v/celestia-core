package viz

import (
	"errors"
	"os"
	"path"
	"path/filepath"

	"github.com/cometbft/cometbft/pkg/trace"
	"github.com/cometbft/cometbft/pkg/trace/schema"
)

func LoadBlockSummaries(root, chainID string) ([]schema.BlockSummary, error) {
	f, err := LoadFirstMatchingFile(root, chainID, schema.BlockTable)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := trace.DecodeFile[schema.BlockSummary](f)
	if err != nil {
		return nil, err
	}

	return trace.ExtractMsgs[schema.BlockSummary](data), nil
}

// LoadFirstMatchingFile searches through directories in the given root for the
// first file that matches the filename. The data is assumed to be organized by
// chainID.
func LoadFirstMatchingFile(rootDir, chainID, table string) (*os.File, error) {
	dirs, err := listDirectories(path.Join(rootDir, chainID))
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		filePath := filepath.Join(rootDir, chainID, dir, table+".jsonl")
		file, err := os.Open(filePath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return file, nil
	}

	return nil, errors.New("file not found")
}

// OpenFiles opens a list of files in a directory and returns a map of
// table name to file. The table name is the base name of the file without
// the extension. If the file does not exist, it is skipped and the error
// is returned.
func OpenFiles(root, chainID, nodeID string, tables []string) (map[string]*os.File, error) {
	out := make(map[string]*os.File)
	var returnErr error
	for _, table := range tables {
		f, err := os.Open(filepath.Join(root, chainID, nodeID, table+".jsonl"))
		if errors.Is(err, os.ErrNotExist) {
			returnErr = err
			continue
		}
		if err != nil {
			return nil, err
		}
		out[table] = f
	}
	return out, returnErr
}

// OpenChainIDFiles opens a list of files in a directory and returns a map of
// nodeID to a map of table name to file. The table name is the base name of
// the file without the extension. If the file does not exist, it is skipped
// and the error is returned.
func OpenChainIDFiles(root, chainID string, tables []string) (map[string]map[string]*os.File, error) {
	out := make(map[string]map[string]*os.File)
	var returnErr error
	dirs, err := listDirectories(path.Join(root, chainID))
	if err != nil {
		return nil, err
	}
	for _, nodeID := range dirs {
		nodeFiles, err := OpenFiles(root, chainID, nodeID, tables)
		if errors.Is(err, os.ErrNotExist) {
			returnErr = err
			continue
		}
		if err != nil {
			return nil, err
		}
		out[nodeID] = nodeFiles
	}
	return out, returnErr
}

// listDirectories returns a list of directory names inside the given path.
func listDirectories(path string) ([]string, error) {
	var dirs []string

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

// CloseFiles closes a map of files.
func CloseFiles(files map[string]*os.File) error {
	for _, f := range files {
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
