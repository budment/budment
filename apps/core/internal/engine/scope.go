package engine

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/vunas/blaster/internal/fastconv"
	"github.com/vunas/blaster/internal/template"
)

type QueueStore struct {
	qMutex sync.RWMutex
	queues map[string]chan any
}

func (q *QueueStore) Push(queueName string, val any) bool {
	q.qMutex.Lock()
	ch, exists := q.queues[queueName]
	if !exists {
		ch = make(chan any, 1000)
		q.queues[queueName] = ch
	}
	q.qMutex.Unlock()
	select {
	case ch <- val:
		return true
	default:
		return false
	}
}

func (q *QueueStore) Pop(queueName string) any {
	q.qMutex.RLock()
	ch, exists := q.queues[queueName]
	q.qMutex.RUnlock()
	if !exists {
		return nil
	}
	select {
	case val := <-ch:
		return val
	default:
		return nil
	}
}

type GlobalState struct {
	store sync.Map
	QueueStore
}

func NewGlobalState() *GlobalState {
	return &GlobalState{
		QueueStore: QueueStore{
			queues: make(map[string]chan any),
		},
	}
}
func (g *GlobalState) Set(key string, val any) {
	g.store.Store(key, val)
}
func (g *GlobalState) Get(key string) (any, bool) {
	return g.store.Load(key)
}
func (g *GlobalState) StoreDistribution(key string, items []any, fallback any) {}

type LocalState struct {
	store sync.Map
	QueueStore
	distStore  sync.Map
	distIndex  sync.Map
	cachedKeys []string
	keysMu     sync.RWMutex
}

func NewLocalState() *LocalState {
	return &LocalState{
		QueueStore: QueueStore{
			queues: make(map[string]chan any),
		}}
}
func (l *LocalState) Set(key string, val any) {
	l.store.Store(key, val)
}
func (l *LocalState) Get(key string) (any, bool) {
	return l.store.Load(key)
}

type distData struct {
	items    []any
	fallback any
}

func (l *LocalState) StoreDistribution(key string, items []any, fallback any) {
	if len(items) == 0 {
		return
	}
	l.distStore.Store(key, &distData{items: items, fallback: fallback})

	if _, exists := l.distIndex.Load(key); !exists {
		var idx uint64 = 0
		l.distIndex.LoadOrStore(key, &idx)

		l.keysMu.Lock()
		l.cachedKeys = append(l.cachedKeys, key)
		l.keysMu.Unlock()
	}
}

func (l *LocalState) DistributeNext(key string) any {
	dataIface, ok := l.distStore.Load(key)
	if !ok {
		return nil
	}
	data := dataIface.(*distData)
	idxIface, _ := l.distIndex.Load(key)
	idxPtr := idxIface.(*uint64)
	currentIndex := atomic.AddUint64(idxPtr, 1) - 1

	if data.fallback != nil && currentIndex >= uint64(len(data.items)) {
		return data.fallback
	}
	return data.items[currentIndex%uint64(len(data.items))]
}

func (l *LocalState) GetAllDistributionKeys() []string {
	l.keysMu.RLock()
	defer l.keysMu.RUnlock()
	return l.cachedKeys
}

type WorkerScope struct {
	localMap      map[string]any
	templateCache map[string]*template.FastTemplate
	Local         *LocalState
	Global        *GlobalState
	abortFlag     bool
}

func NewWorkerScope(ls *LocalState, gs *GlobalState) *WorkerScope {
	return &WorkerScope{
		localMap:      make(map[string]any, 16),
		templateCache: make(map[string]*template.FastTemplate, 8),
		Local:         ls,
		Global:        gs,
	}
}

func (s *WorkerScope) Set(key string, val any)    { s.localMap[key] = val }
func (s *WorkerScope) Get(key string) (any, bool) { val, ok := s.localMap[key]; return val, ok }
func (s *WorkerScope) Delete(key string)          { delete(s.localMap, key) }
func (s *WorkerScope) Reset() {
	clear(s.localMap)
	s.SetFlags(false)
}

func (s *WorkerScope) Interpolate(raw any) string {
	if raw == nil {
		return ""
	}

	str, ok := raw.(string)
	if !ok {
		str = fastconv.String(raw)
	}

	tmpl, exists := s.templateCache[str]
	if !exists {
		tmpl = template.Compile(str)
		s.templateCache[str] = tmpl
	}

	return tmpl.Render(s)
}

func (s *WorkerScope) SetFlags(abort bool) {
	s.abortFlag = abort
}
func (s *WorkerScope) IsAbortFlag() bool { return s.abortFlag }

func (s *WorkerScope) Resolve(name string) (any, bool) {
	if after, match := strings.CutPrefix(name, "@local:"); match {
		if s.Local != nil {
			return s.Local.Get(after)
		}
		return nil, false
	}

	if after, match := strings.CutPrefix(name, "@global:"); match {
		if s.Global != nil {
			return s.Global.Get(after)
		}
		return nil, false
	}

	if after, match := strings.CutPrefix(name, "@pop:local:"); match {
		if s.Local != nil {
			val := s.Local.Pop(after)
			return val, val != nil
		}
		return nil, false
	}

	if after, match := strings.CutPrefix(name, "@pop:global:"); match {
		if s.Global != nil {
			val := s.Global.Pop(after)
			return val, val != nil
		}
		return nil, false
	}

	val, ok := s.localMap[name]
	return val, ok
}
