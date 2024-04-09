package trace

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

// DecodeFile reads a file and decodes it into a slice of events via
// scanning. The table parameter is used to determine the type of the events.
// The file should be a jsonl file. The generic here are passed to the event
// type.
func DecodeFile[T any](f *os.File) ([]Event[T], error) {
	var out []Event[T]
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		var e Event[T]
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, err
		}

		out = append(out, e)
	}

	return out, nil
}

type ProcessorFunc[T any] func(Event[T]) error

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

type FilterFunc[T any] func(Event[T]) (bool, error)

type Filter[T Entry] struct {
	results   []Event[T]
	keepFuncs []FilterFunc[T]
}

func NewFilter[T Entry](keepFuncs ...FilterFunc[T]) *Filter[T] {
	return &Filter[T]{keepFuncs: keepFuncs}
}

func (f *Filter[T]) Process(e Event[T]) error {
	for _, keepFunc := range f.keepFuncs {
		if keep, err := keepFunc(e); !keep {
			return nil
		} else if err != nil {
			return err
		}
	}
	f.results = append(f.results, e)
	return nil
}

func (f *Filter[T]) Results() []Event[T] {
	return f.results
}

func NewHeightFilterFunc[T HasHeight](height int64) FilterFunc[T] {
	return func(e Event[T]) (bool, error) {
		return e.Msg.Height() == height, nil
	}
}

type HasHeight interface {
	Height() int64
}

func NewPeerFilterFunc[T HasPeer](peer string) FilterFunc[T] {
	return func(e Event[T]) (bool, error) {
		return e.Msg.Peer() == peer, nil
	}
}

type HasPeer interface {
	Peer() string
}

func NewTransferTypeFilterFunc[T HasTransferType](transferType string) FilterFunc[T] {
	return func(e Event[T]) (bool, error) {
		return e.Msg.TransferType() == transferType, nil
	}
}

type HasTransferType interface {
	TransferType() string
}
