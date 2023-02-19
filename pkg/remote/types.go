package remote

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	EventResponseStatusOK    = "ok"
	EventResponseStatusError = "error"
)

type Event struct {
	NodeID  string          `json:"node_id"`
	ChainID string          `json:"chain_id"`
	Type    string          `json:"type"`
	Time    time.Time       `json:"time"`
	Data    json.RawMessage `json:"data"`
}

func (e Event) FilePath() string {
	return fmt.Sprintf("%s/%s/%s.json", e.ChainID, e.NodeID, e.Type)
}

type BatchRequest struct {
	NodeID  string  `json:"node_id"`
	ChainID string  `json:"chain_id"`
	Events  []Event `json:"events"`
}

// go event is the same as an Event, except it has a json.Marshaler instead of
// raw data. This allows for the event data to be queued before being marshaled
// as to add minimal overhead to an existing process.
type goEvent struct {
	Type string
	Time time.Time
	Data interface{}
}

func (ge goEvent) Event() (Event, error) {
	d, err := json.Marshal(ge.Data)
	if err != nil {
		return Event{}, err
	}
	return Event{Type: ge.Type, Time: ge.Time, Data: d}, nil
}

func NewEvent(eventType string, data []byte) Event {
	return Event{Type: eventType, Time: time.Now().UTC(), Data: data}
}

func newGoEvent(eventType string, data interface{}) goEvent {
	return goEvent{Type: eventType, Time: time.Now().UTC(), Data: data}
}

type GenericResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func NewOKResponse() []byte {
	return mustMarshal(GenericResponse{Status: EventResponseStatusOK})
}

func NewErrorResponse(err string) []byte {
	return mustMarshal(GenericResponse{Status: EventResponseStatusError, Error: err})
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
