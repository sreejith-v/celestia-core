package trace

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

// ProcessorFunc is a function that processes an event. It is used to process
// streamed data.
type ProcessorFunc[T any] func(Event[T]) error

// Apply reads a file and applies a processor function to each event in the
// file. The processor function should return an error if the processing fails.
func Apply[T any](f *os.File, p ProcessorFunc[T]) error {
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		var e Event[T]
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return err
		}

		if err := p(e); err != nil {
			return err
		}
	}
	return nil
}

// FilterFunc is a function that determines if an event should be kept or not
// when applying a filter.
type FilterFunc[T any] func(Event[T]) bool

// Filter holds the results and filter functions of a filter operation.
type Filter[T Entry] struct {
	results   []Event[T]
	keepFuncs []FilterFunc[T]
}

// NewFilter creates a new filter with the given filter functions.
func NewFilter[T Entry](keepFuncs ...FilterFunc[T]) *Filter[T] {
	return &Filter[T]{keepFuncs: keepFuncs}
}

// Process applies the filter functions to the event and keeps the event if all
// filter functions return true.
func (f *Filter[T]) Process(e Event[T]) error {
	for _, keepFunc := range f.keepFuncs {
		if keep := keepFunc(e); !keep {
			return nil
		}
	}
	f.results = append(f.results, e)
	return nil
}

// Results returns the results of the filter operation.
func (f *Filter[T]) Results() []Event[T] {
	return f.results
}

// HasHeight is an interface that describes an object that has a height. This is
// used for filtering based on height.
type HasHeight interface {
	GetHeight() int64
}

// NewHeightFilterFunc creates a new filter function that filters based on the
// height of the event.
func NewHeightFilterFunc[T HasHeight](height int64) FilterFunc[T] {
	return func(e Event[T]) bool {
		return e.Msg.GetHeight() == height
	}
}

// HasPeer is an interface that describes an object that has a peer. This is
// used for filtering based on the peer.
type HasPeer interface {
	GetPeer() string
}

// NewPeerFilterFunc creates a new filter function that filters based on the
// peer of the event.
func NewPeerFilterFunc[T HasPeer](peer string) FilterFunc[T] {
	return func(e Event[T]) bool {
		return e.Msg.GetPeer() == peer
	}
}

// HasTransferType is an interface that describes an object that has a transfer
// type. This is used for filtering based on the upload or download.
type HasTransferType interface {
	TransferType() string
}

// NewTransferTypeFilterFunc creates a new filter function that filters based on
// the transfer type of the event.
func NewTransferTypeFilterFunc[T HasTransferType](transferType string) FilterFunc[T] {
	return func(e Event[T]) bool {
		return e.Msg.TransferType() == transferType
	}
}
