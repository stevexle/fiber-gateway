package balancer

import (
	"math/rand"
	"sync"
	"sync/atomic"
)

// Balancer defines an interface for various load balancing strategies
type Balancer interface {
	Next() string
	Update(target string, activeDelta int) // Used for LeastConn
	Targets() []string                     // Returns the list of all targets
}

// ─── Round Robin ─────────────────────────────────────────────────────────────

type roundRobin struct {
	targets []string
	counter uint64
}

func NewRoundRobin(targets []string) Balancer {
	return &roundRobin{targets: targets}
}

func (b *roundRobin) Next() string {
	if len(b.targets) == 0 {
		return ""
	}
	idx := atomic.AddUint64(&b.counter, 1) % uint64(len(b.targets))
	return b.targets[idx]
}

func (b *roundRobin) Update(string, int) {}

func (b *roundRobin) Targets() []string {
	return b.targets
}

// ─── Random ──────────────────────────────────────────────────────────────────

type random struct {
	targets []string
}

func NewRandom(targets []string) Balancer {
	return &random{targets: targets}
}

func (b *random) Next() string {
	if len(b.targets) == 0 {
		return ""
	}
	return b.targets[rand.Intn(len(b.targets))]
}

func (b *random) Update(string, int) {}

func (b *random) Targets() []string {
	return b.targets
}

// ─── Least Connections ────────────────────────────────────────────────────────

type leastConn struct {
	targets     []string
	activeConns map[string]*int32
	mu          sync.RWMutex
}

func NewLeastConn(targets []string) Balancer {
	conns := make(map[string]*int32)
	for _, t := range targets {
		var val int32
		conns[t] = &val
	}
	return &leastConn{targets: targets, activeConns: conns}
}

func (b *leastConn) Next() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.targets) == 0 {
		return ""
	}

	var bestTarget string
	var minConn int32 = -1

	for _, t := range b.targets {
		conns := atomic.LoadInt32(b.activeConns[t])
		if minConn == -1 || conns < minConn {
			minConn = conns
			bestTarget = t
		}
	}
	return bestTarget
}

func (b *leastConn) Update(target string, delta int) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if conns, ok := b.activeConns[target]; ok {
		atomic.AddInt32(conns, int32(delta))
	}
}

func (b *leastConn) Targets() []string {
	return b.targets
}
