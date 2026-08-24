package engine

import (
	"context"
	"sync"
	"time"
)

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

func (sm *BarrierManager) ArriveAndWait(ctx context.Context, name string, quorum int, grace, maxWait time.Duration) {
	sm.mu.Lock()
	b, exists := sm.barriers[name]
	if !exists {
		b = &barrier{
			releaseChan: make(chan struct{}),
			quorum:      quorum,
			closed:      false,
		}
		sm.barriers[name] = b
	}
	sm.mu.Unlock()

	b.mu.Lock()
	// let latecomers pass through immediately if barrier is closed
	if b.closed {
		b.mu.Unlock()
		return
	}

	b.arrived++

	if b.arrived >= b.quorum {
		b.closed = true
		select {
		case <-b.releaseChan:
		default:
			close(b.releaseChan)
		}
		b.mu.Unlock()

		// keep the map for 5s so late arrivals can still see b.closed == true
		time.AfterFunc(5*time.Second, func() {
			sm.mu.Lock()
			if sm.barriers[name] == b {
				delete(sm.barriers, name)
			}
			sm.mu.Unlock()
		})
		return
	}

	releaseCh := b.releaseChan
	b.mu.Unlock()

	waitLimit := grace
	if maxWait > 0 && maxWait < grace {
		waitLimit = maxWait
	}
	if waitLimit <= 0 {
		waitLimit = 5 * time.Second
	}

	timer := time.NewTimer(waitLimit)
	defer timer.Stop()

	select {
	case <-releaseCh:
		return

	case <-timer.C:
		b.mu.Lock()
		if !b.closed {
			b.closed = true // timeout close
			select {
			case <-b.releaseChan:
			default:
				close(b.releaseChan)
			}
		}
		b.mu.Unlock()

		sm.mu.Lock()
		if sm.barriers[name] == b {
			delete(sm.barriers, name)
		}
		sm.mu.Unlock()
		return

	case <-ctx.Done():
		b.mu.Lock()
		if !b.closed {
			b.closed = true
			select {
			case <-b.releaseChan:
			default:
				close(b.releaseChan)
			}
		}
		b.mu.Unlock()
		return
	}
}
