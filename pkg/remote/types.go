package remote

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	EventResponseStatusOK    = "ok"
	EventResponseStatusError = "error"
)

// Event is the event sent to the server. It can hold arbitrary data.
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

// BatchRequest is the request sent to the server to batch events.
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

type Query struct {
	NodeID  string `json:"node_id"`
	ChainID string `json:"chain_id"`
	Type    string `json:"type"`
}

// NewQuery creates a new query from a path string. The path string should be in
// the format of chainID/nodeID/type.
func NewQuery(path string) (Query, error) {
	if path == "" {
		return Query{}, nil
	}
	split := strings.Split(path, "/")
	if len(split) != 3 {
		return Query{}, fmt.Errorf("invalid path length: %d", len(split))
	}
	return Query{NodeID: split[1], ChainID: split[0], Type: split[2]}, nil
}

func (q Query) Path() string {
	return fmt.Sprintf("%s/%s/%s", q.ChainID, q.NodeID, q.Type)
}

// URLValues returns the query as a URL encoded string.
func (q Query) URLValues() string {
	return fmt.Sprintf("?node_id=%s&chain_id=%s&type=%s", q.NodeID, q.ChainID, q.Type)
}

func (q Query) IsEmpty() bool {
	return q.ChainID == ""
}

func (q Query) ValidateBasic() error {
	if q.ChainID == "" {
		return fmt.Errorf("chainID cannot be empty")
	}

	return nil
}

type QueryResponse struct {
	Status string  `json:"status"`
	Error  string  `json:"error"`
	Events []Event `json:"events"`
}

// LabeledFile is a file that is labeled with a chainID, nodeID, and type.
type LabeledFile struct {
	ChainID, NodeID, Type string
	// we have two files because we want to be able simultaneously read and
	// write to the same file.
	wf, rf *os.File
}

// OpenLabeledFile opens a labeled file.
func OpenLabeledFile(path string) (*LabeledFile, error) {
	// open two copies of the file so we can read and write to it
	// simultaneously.
	wf, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	rf, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &LabeledFile{
		wf: wf,
		rf: rf,
	}, nil
}

// Close closes the underlying files.
func (lf *LabeledFile) Close() error {
	if err := lf.wf.Close(); err != nil {
		return err
	}
	return lf.rf.Close()
}

func (lf *LabeledFile) Label(chainID, nodeID, eventType string) {
	lf.ChainID = chainID
	lf.NodeID = nodeID
	lf.Type = eventType
}

// ReadJsonLinesFile reads a file, one line at a time, and decodes each line.
func ReadJsonLinesFile[T any](f *LabeledFile) ([]T, error) {
	// reset the file pointer to the beginning of the file
	defer f.rf.Seek(0, io.SeekStart)
	// iterate over the file and decode each line into an event
	var events []T
	d := json.NewDecoder(f.rf)
	for {
		var v T
		if err := d.Decode(&v); err == io.EOF {
			break // done decoding file
		} else if err != nil {
			if err != nil {
				return nil, err
			}
		}
		events = append(events, v)
	}

	return events, nil
}

// WriteJsonLinesFile writes a list of events to a file, one event per line.
func WriteJsonLinesFile[T any](f *LabeledFile, evs []T) error {
	if len(evs) == 0 {
		return nil
	}

	// this should never be hit, but could be if the filesystem was manually
	// manipulated, we should return an error instead of panicking to avoid
	// crashing the server.
	if f == nil || f.wf == nil {
		return fmt.Errorf("cannot write to a nil file")
	}

	w := bufio.NewWriter(f.wf)
	defer w.Flush()
	for _, ev := range evs {
		rawEv, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		_, err = w.Write(rawEv)
		if err != nil {
			return err
		}
	}

	return nil
}
