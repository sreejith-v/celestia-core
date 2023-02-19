package remote

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var eventRequest Event
	err := json.NewDecoder(r.Body).Decode(&eventRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("SERVER: failure to decode event request: ", err)
		return
	}

	err = s.writeEvent(eventRequest)
	if err != nil {
		w.Write(NewErrorResponse(err.Error()))
		return
	}

	s.logger.Info(
		"wrote event",
		"nodeID", eventRequest.NodeID,
		"chainID", eventRequest.ChainID,
		"type", eventRequest.Type,
	)

	w.Write(NewOKResponse())
}

func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var batchRequest BatchRequest
	err := json.NewDecoder(r.Body).Decode(&batchRequest)
	if err != nil {
		fmt.Println("SERVER: failure to decode event request: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.writeBatch(batchRequest)
	if err != nil {
		w.Write(NewErrorResponse(err.Error()))
		return
	}

	s.logger.Info(
		"wrote batch",
		"nodeID", batchRequest.NodeID,
		"chainID", batchRequest.ChainID,
		"events", len(batchRequest.Events),
	)

	w.Write(NewOKResponse())
}

func (s *Server) writeEvent(ev Event) error {
	f, err := s.GetFile(ev.FilePath())
	if err != nil {
		return err
	}

	return json.NewEncoder(f).Encode(ev)
}

func (s *Server) writeBatch(br BatchRequest) error {
	for _, ev := range br.Events {
		err := s.writeEvent(ev)
		fmt.Println("SERVER: wrote event: ", ev, " err: ", err)
		if err != nil {
			return err
		}
	}

	return nil
}
