package remote

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tendermint/tendermint/libs/log"
)

const (
	batchMethod = "batch"
	eventMethod = "event"
)

// Server is a simple json server that will write and serve events to files.
type Server struct {
	writeDir string
	logger   log.Logger

	mut *sync.RWMutex
	// the server expects the files to be in the following format:
	// <chainID>/<nodeID>/<type>
	files map[string]map[string]map[string]*os.File
}

func NewServer(dir string, logger log.Logger) *Server {
	return &Server{
		writeDir: dir,
		mut:      &sync.RWMutex{},
		files:    loadFiles(dir),
		logger:   logger,
	}
}

func (s *Server) Start(addr string) error {
	defer s.Stop()
	http.HandleFunc("/batch", s.handleBatch)
	http.HandleFunc("/event", s.handleEvent)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) Stop() {
	s.mut.RLock()
	defer s.mut.RUnlock()
	for _, i := range s.files {
		for _, j := range i {
			for _, f := range j {
				_ = f.Close()
			}
		}
	}
}

// GetFile attempts to fetch loaded file. If the file does not exist, then it
// will be created. path is expected to be in the following format:
// <chainID>/<nodeID>/<type>
func (s *Server) GetFile(path string) (fs *os.File, err error) {
	chainID, nodeID, typ, err := parseRelativePath(path)
	if err != nil {
		return nil, err
	}

	f, has := s.getFile(chainID, nodeID, typ)
	if !has {
		err = os.MkdirAll(filepath.Join(s.writeDir, chainID, nodeID), os.ModePerm)
		if err != nil {
			return nil, err
		}
		f, err = os.OpenFile(
			filepath.Join(s.writeDir, chainID, nodeID, typ+".json"),
			os.O_APPEND|os.O_CREATE|os.O_RDWR,
			0644,
		)
		if err != nil {
			return nil, err
		}
		s.mut.Lock()
		s.files = createPath(s.files, chainID, nodeID)
		s.files[chainID][nodeID][typ] = f
		s.mut.Unlock()
	}

	return f, nil
}

func (s *Server) getFile(chainID, nodeID, typ string) (*os.File, bool) {
	s.mut.RLock()
	defer s.mut.RUnlock()
	f, has := s.files[chainID][nodeID][typ]
	return f, has
}

func loadFiles(directory string) map[string]map[string]map[string]*os.File {
	// Walk the directory recursively and print out the relative file paths
	files := make(map[string]map[string]map[string]*os.File)
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			if !strings.Contains(info.Name(), ".json") {
				return nil
			}

			f, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR, 0644)
			if err != nil {
				f.Close()
				return nil
			}

			relativePath, err := filepath.Rel(directory, path)
			if err != nil {
				return nil
			}

			chainid, nodeID, typ, err := parseRelativePath(relativePath)
			if err != nil {
				f.Close()
				return nil
			}
			files = createPath(files, chainid, nodeID)
			files[chainid][nodeID][typ] = f
		}
		return nil
	})
	return files
}

func createPath(files map[string]map[string]map[string]*os.File, chainID, nodeID string) map[string]map[string]map[string]*os.File {
	if _, has := files[chainID]; !has {
		files[chainID] = make(map[string]map[string]*os.File)
	}
	if _, has := files[chainID][nodeID]; !has {
		files[chainID][nodeID] = make(map[string]*os.File)
	}
	return files
}

// parseRelativePath parses a relative path into its chainID, nodeID, and type.
// If .json is not already added to the type, it will be added.
func parseRelativePath(path string) (chainID, nodeID, typ string, err error) {
	split := strings.Split(path, "/")
	if len(split) != 3 {
		return "", "", "", fmt.Errorf("invalid path length: %s", path)
	}
	for _, p := range split {
		if p == "" {
			fmt.Println("SERVER: bad split", split, len(split), path)
			return "", "", "", errors.New("empty path is invalid")
		}
	}
	split[2] = strings.ReplaceAll(split[2], ".json", "")
	return split[0], split[1], split[2], nil
}
