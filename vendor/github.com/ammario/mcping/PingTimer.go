package mcping

import (
	"time"
)

type pingTimer struct {
	start   uint64 //Start time in ms
	end     uint64 //End time in ms
	latency uint64 //Latency time in ms
}

func getMS() uint64 {
	return uint64(time.Now().UnixNano() / int64(time.Millisecond))
}

func (t *pingTimer) Start() {
	t.start = getMS()
}

func (t *pingTimer) End() (latency uint64) {
	t.end = getMS()
	t.latency = t.end - t.start
	latency = t.latency
	return
}
