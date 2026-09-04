package engine

import (
	"context"
	"sync"
	"time"
)

// Coordinates synchronization points across Virtual Users.
type BarrierManager struct {
	mu       sync.Mutex
	barriers map[string]*barrier
}

type barrier struct {
	mu          sync.Mutex
	releaseChan chan struct{}
	arrived     int
	quorum      int
	closed      bool
}

func NewBarrierManager() *BarrierManager {
	return &BarrierManager{
		barriers: make(map[string]*barrier),
	}
}

// Blocks until quorum is reached or maxWait expires.
func (sm *BarrierManager) ArriveAndWait(ctx context.Context, name string, quorum int, maxWait time.Duration) {
	if maxWait <= 0 {
		maxWait = 30 * time.Second
	}

	sm.mu.Lock()
	b, exists := sm.barriers[name]
	if !exists {
		b = &barrier{
			releaseChan: make(chan struct{}),
			quorum:      quorum,
		}
		sm.barriers[name] = b
	}

	b.mu.Lock()
	// let latecomers pass through immediately if barrier is closed
	if b.closed {
		b.mu.Unlock()
		b = &barrier{
			releaseChan: make(chan struct{}),
			quorum:      quorum,
		}
		sm.barriers[name] = b
		b.mu.Lock()
	}

	b.arrived++

	if b.arrived >= b.quorum {
		b.closed = true
		close(b.releaseChan)
		delete(sm.barriers, name)
		b.mu.Unlock()
		sm.mu.Unlock()
		return
	}

	releaseCh := b.releaseChan
	b.mu.Unlock()
	sm.mu.Unlock()

	timer := time.NewTimer(maxWait)
	defer timer.Stop()

	select {
	case <-releaseCh: // Sync succeeded.
		return

	case <-timer.C: // timeout.
		sm.mu.Lock()
		b.mu.Lock()
		if !b.closed {
			b.closed = true
			close(b.releaseChan)
			if cur, ok := sm.barriers[name]; ok && cur == b {
				delete(sm.barriers, name)
			}
		}
		b.mu.Unlock()
		sm.mu.Unlock()
		return

	case <-ctx.Done():
		sm.mu.Lock()
		b.mu.Lock()
		if !b.closed {
			b.closed = true
			close(b.releaseChan)
			if cur, ok := sm.barriers[name]; ok && cur == b {
				delete(sm.barriers, name)
			}
		}
		b.mu.Unlock()
		sm.mu.Unlock()
		return
	}
}
