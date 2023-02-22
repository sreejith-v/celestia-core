package remote

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	queryChainID = "chain_id"
	queryNodeID  = "node_id"
	queryType    = "type"
	// QueryAll is a special string that can be used to query all files.
	QueryAll = "*"
)

// QueryFiles returns a list of files that match the given query. It throws an
// error if there are no files for a given query.
func (s *Server) QueryFiles(chainID, nodeID, typ string) (fs []*LabeledFile, err error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if chainID == QueryAll {
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

	if nodeID == QueryAll {
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

	if typ == QueryAll {
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

// handleGetEvents handles the /get_events endpoint. It responds with a
// QueryResponse filled with any events that match the given query.
func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
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

	var events []Event
	for _, f := range fs {
		evs, err := ReadJsonLinesFile[Event](f)
		if err != nil {
			w.Write(NewErrorResponse(err.Error()))
			return
		}
		if len(evs) == 0 {
			w.Write(NewErrorResponse(fmt.Sprintf("empty file: %s", qType)))
			return
		}
		events = append(events, evs...)
	}

	resp := QueryResponse{
		Status: EventResponseStatusOK,
		Events: events,
	}

	rawEvents, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		w.Write(NewErrorResponse(err.Error()))
		return
	}

	s.logger.Info("read events", "events", len(events))

	w.Write(rawEvents)
}
