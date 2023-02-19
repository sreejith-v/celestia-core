package remote

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	queryChainID = "chainid"
	queryNodeID  = "nodeid"
	queryType    = "type"
	queryAll     = "*"
)

func (s *Server) QueryFiles(chainID, nodeID, typ string) (fs []*os.File, err error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if chainID == queryAll {
		for _, i := range s.files {
			for _, j := range i {
				for _, f := range j {
					fs = append(fs, f)
				}
			}
		}
		return fs, nil
	}

	chainIDRemaining, has := s.files[chainID]
	if !has {
		return nil, fmt.Errorf("chainID %s does not exist", chainID)
	}

	if nodeID == queryAll {
		for _, j := range chainIDRemaining {
			for _, f := range j {
				fs = append(fs, f)
			}
		}
		return fs, nil
	}

	nodeIDRemaining, has := chainIDRemaining[nodeID]
	if !has {
		return nil, fmt.Errorf("nodeID %s does not exist", nodeID)
	}

	if typ == queryAll {
		for _, f := range nodeIDRemaining {
			fs = append(fs, f)
		}
		return fs, nil
	}

	typRemaining, has := nodeIDRemaining[typ]
	if !has {
		return nil, fmt.Errorf("type %s does not exist", typ)
	}

	fs = append(fs, typRemaining)

	return fs, nil
}

func (s *Server) handleReadFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	vs := r.URL.Query()
	qChainID, qNodeID, qType := vs.Get(queryChainID), vs.Get(queryNodeID), vs.Get(queryType)
	fs, err := s.QueryFiles(qChainID, qNodeID, qType)
	if err != nil {
		w.Write(NewErrorResponse(err.Error()))
		return
	}

	var events []string
	for _, f := range fs {
		evs, err := ReadJsonLinesFile[string](f)
		if err != nil {
			w.Write(NewErrorResponse(err.Error()))
			return
		}
		events = append(events, evs...)
	}

	rawEvents, err := json.Marshal(events)
	if err != nil {
		w.Write(NewErrorResponse(err.Error()))
		return
	}

	s.logger.Info("read events", "events", len(events))

	w.Write(rawEvents)
}

func ReadJsonLinesFile[T any](f *os.File) ([]T, error) {
	// iterate over the file and decode each line into an event
	s := bufio.NewScanner(f)
	var events []T
	for s.Scan() {
		var ev T
		if err := json.Unmarshal(s.Bytes(), &ev); err != nil {
			log.Fatal(err)
		}
		events = append(events, ev)
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	return events, nil
}
