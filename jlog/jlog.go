package jlog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

var globalLogger *Logger

func init() {
	globalLogger = NewLogger(fmt.Sprintf("/home/celes/.jlog/%d.json", time.Now().Unix()))
	globalLogger.Start(context.TODO())
}

func Log(v interface{}) {
	globalLogger.queue <- v
}

type Logger struct {
	ctx   context.Context
	file  *os.File
	enc   *json.Encoder
	queue chan interface{}
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
		queue: make(chan interface{}, 10000),
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
