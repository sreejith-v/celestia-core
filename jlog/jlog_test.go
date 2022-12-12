package jlog

import (
	"testing"
	"time"
)

type TestStruct struct {
	Test string `json:"test"`
}

func Test(t *testing.T) {
	Log(TestStruct{Test: "test"})
	time.Sleep(time.Second)
}
