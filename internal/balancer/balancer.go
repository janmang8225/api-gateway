package balancer

import (
	"sync/atomic"
)

type RoundRobin struct {
	backends []string
	counter  atomic.Uint64
}

func NewRoundRobin(backends []string) *RoundRobin {
	return &RoundRobin{
		backends: backends,
	}
}

func (rr *RoundRobin) Next() string {
	index := rr.counter.Add(1) - 1
	return rr.backends[index%uint64(len(rr.backends))]
}