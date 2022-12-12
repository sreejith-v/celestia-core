package jlog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/tendermint/tendermint/libs/rand"
)

var globalLogger *Logger

func init() {
	globalLogger = NewLogger(fmt.Sprintf("/home/celes/.jlog/%s-%d.json", rand.Str(5), time.Now().Minute()))
	globalLogger.Start(context.TODO())
}

func Log(name string, data interface{}, metadata interface{}) {
	wlog := WrappedLog{Data: data, Name: name, MetaData: metadata}
	globalLogger.queue <- wlog
}

type Logger struct {
	ctx   context.Context
	file  *os.File
	enc   *json.Encoder
	queue chan WrappedLog
}

func NewLogger(path string) *Logger {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		panic(err)
	}
	enc := json.NewEncoder(file)
	return &Logger{
		enc:   enc,
		file:  file,
		queue: make(chan WrappedLog, 10000),
	}
}

func (l *Logger) Start(ctx context.Context) {
	go func() {
		defer l.Stop()
		for {
			select {
			case <-l.ctx.Done():
				return
			case d := <-l.queue:
				globalLogger.enc.Encode(d)
			}
		}
	}()
}

func (l *Logger) Stop() {
	for d := range l.queue {
		err := l.enc.Encode(d)
		if err != nil {
			panic(err)
		}
	}
	l.file.Close()
}

type WrappedLog struct {
	Data     interface{} `json:"data"`
	Name     string      `json:"name"`
	MetaData interface{} `json:"metadata,omitempty"`
}
