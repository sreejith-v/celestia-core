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
	batchMethod     = "/batch"
	eventMethod     = "/event"
	getEventsMethod = "/get_events"
)

// Server is a simple json server that will write and serve events to files.
type Server struct {
	writeDir string
	logger   log.Logger

	mut *sync.RWMutex
	// the server expects the files to be in the following format:
	// <chainID>/<nodeID>/<type>
	files map[string]map[string]map[string]*LabeledFile
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
	http.HandleFunc(batchMethod, s.handleBatch)
	http.HandleFunc(eventMethod, s.handleEvent)
	http.HandleFunc(getEventsMethod, s.handleGetFiles)
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
//
// TODO: pass the parameters as separate arguments instead of a path.
func (s *Server) GetFile(path string) (lf *LabeledFile, err error) {
	chainID, nodeID, typ, err := parseRelativePath(path)
	if err != nil {
		return nil, err
	}

	lf, has := s.getFile(chainID, nodeID, typ)
	if !has {
		err = os.MkdirAll(filepath.Join(s.writeDir, chainID, nodeID), os.ModePerm)
		if err != nil {
			return nil, err
		}

		lf, err = OpenLabeledFile(filepath.Join(s.writeDir, chainID, nodeID, typ+".json"))
		if err != nil {
			return nil, err
		}
		lf.Label(chainID, nodeID, typ)

		// add the file to the server's filesystem
		s.mut.Lock()
		s.files = createPath(s.files, chainID, nodeID)
		s.files[chainID][nodeID][typ] = lf
		s.mut.Unlock()
	}

	return lf, nil
}

func (s *Server) getFile(chainID, nodeID, typ string) (*LabeledFile, bool) {
	s.mut.RLock()
	defer s.mut.RUnlock()
	f, has := s.files[chainID][nodeID][typ]
	return f, has
}

func loadFiles(directory string) map[string]map[string]map[string]*LabeledFile {
	// Walk the directory recursively and print out the relative file paths
	files := make(map[string]map[string]map[string]*LabeledFile)
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			if !strings.Contains(info.Name(), ".json") {
				return nil
			}

			lf, err := OpenLabeledFile(path)
			if err != nil {
				lf.Close()
				return nil
			}

			relativePath, err := filepath.Rel(directory, path)
			if err != nil {
				return nil
			}

			chainid, nodeID, typ, err := parseRelativePath(relativePath)
			if err != nil {
				lf.Close()
				return nil
			}
			lf.Label(chainid, nodeID, typ)
			files = createPath(files, chainid, nodeID)
			files[chainid][nodeID][typ] = lf
		}
		return nil
	})
	return files
}

func createPath(files map[string]map[string]map[string]*LabeledFile, chainID, nodeID string) map[string]map[string]map[string]*LabeledFile {
	if _, has := files[chainID]; !has {
		files[chainID] = make(map[string]map[string]*LabeledFile)
	}
	if _, has := files[chainID][nodeID]; !has {
		files[chainID][nodeID] = make(map[string]*LabeledFile)
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
			return "", "", "", errors.New("empty path is invalid")
		}
	}
	split[2] = strings.ReplaceAll(split[2], ".json", "")
	return split[0], split[1], split[2], nil
}
