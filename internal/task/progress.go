package task

import (
	"sync"
	"time"
)

type ResultUpdate struct {
	Provider         string     `json:"provider"`
	Status           string     `json:"status"`
	Progress         int        `json:"progress"`
	OutputURL        string     `json:"output_url,omitempty"`
	FileCode         string     `json:"file_code,omitempty"`
	ProviderFileName string     `json:"provider_file_name,omitempty"`
	ProviderFileSize int64      `json:"provider_file_size,omitempty"`
	Error            string     `json:"error,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type TaskUpdate struct {
	ID      string         `json:"id"`
	Status  string         `json:"status"`
	Results []ResultUpdate `json:"results"`
	Done    bool           `json:"done"`
}

type subscriber struct {
	ch   chan TaskUpdate
	done chan struct{}
}

type ProgressHub struct {
	mu     sync.RWMutex
	subs   map[string][]*subscriber
	tasks  map[string]*TaskCache
}

type TaskCache struct {
	results map[string]ResultUpdate
	status  string
	done    bool
}

func NewProgressHub() *ProgressHub {
	return &ProgressHub{
		subs:  make(map[string][]*subscriber),
		tasks: make(map[string]*TaskCache),
	}
}

func (h *ProgressHub) Subscribe(taskID string) (<-chan TaskUpdate, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan TaskUpdate, 16)
	s := &subscriber{ch: ch, done: make(chan struct{})}
	h.subs[taskID] = append(h.subs[taskID], s)

	if cached, ok := h.tasks[taskID]; ok {
		ch <- buildUpdate(taskID, cached)
	}

	unsub := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subs[taskID]
		for i, sub := range subs {
			if sub == s {
				h.subs[taskID] = append(subs[:i], subs[i+1:]...)
				close(sub.done)
				close(sub.ch)
				break
			}
		}
		if len(h.subs[taskID]) == 0 {
			delete(h.subs, taskID)
			go func() {
				time.Sleep(30 * time.Second)
				h.mu.Lock()
				delete(h.tasks, taskID)
				h.mu.Unlock()
			}()
		}
	}

	return ch, unsub
}

func (h *ProgressHub) PublishResult(taskID, taskStatus string, update ResultUpdate) {
	h.mu.Lock()
	cached, ok := h.tasks[taskID]
	if !ok {
		cached = &TaskCache{results: make(map[string]ResultUpdate)}
		h.tasks[taskID] = cached
	}
	cached.results[update.Provider] = update
	if taskStatus != "" {
		cached.status = taskStatus
	}
	taskUpdate := buildUpdate(taskID, cached)

	subs := h.subs[taskID]
	subsCopy := make([]*subscriber, len(subs))
	copy(subsCopy, subs)
	h.mu.Unlock()

	for _, s := range subsCopy {
		select {
		case s.ch <- taskUpdate:
		case <-s.done:
		default:
		}
	}
}

func (h *ProgressHub) MarkDone(taskID, taskStatus string) {
	h.mu.Lock()
	cached, ok := h.tasks[taskID]
	if ok {
		cached.done = true
		if taskStatus != "" {
			cached.status = taskStatus
		}
		taskUpdate := buildUpdate(taskID, cached)
		subs := h.subs[taskID]
		subsCopy := make([]*subscriber, len(subs))
		copy(subsCopy, subs)
		h.mu.Unlock()

		for _, s := range subsCopy {
			select {
			case s.ch <- taskUpdate:
			case <-s.done:
			default:
			}
		}
	} else {
		h.mu.Unlock()
	}
}

func buildUpdate(id string, cached *TaskCache) TaskUpdate {
	results := make([]ResultUpdate, 0, len(cached.results))
	for _, r := range cached.results {
		results = append(results, r)
	}
	return TaskUpdate{
		ID:      id,
		Status:  cached.status,
		Results: results,
		Done:    cached.done,
	}
}
