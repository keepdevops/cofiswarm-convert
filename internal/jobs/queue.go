package jobs

import (
	"fmt"
	"sync"
	"time"
)

type Job struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Step       string `json:"step"`
	Pct        int    `json:"pct"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	HFRepo     string `json:"hf_repo"`
	OutputName string `json:"output_name"`
	CreatedAt  int64  `json:"created_at"`
}

type Queue struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func New() *Queue { return &Queue{jobs: map[string]*Job{}} }

func (q *Queue) Create(hfRepo, outputName string, qBits int) *Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	id := fmt.Sprintf("cvt_%d", time.Now().UnixMilli())
	j := &Job{
		ID: id, Status: "queued", Step: "accepted", Pct: 0,
		HFRepo: hfRepo, OutputName: outputName, CreatedAt: time.Now().UnixMilli(),
	}
	q.jobs[id] = j
	// Bridge sprint: mark running then done (no fork — job queue API parity only)
	j.Status = "running"
	j.Step = "gguf_to_mlx.py"
	j.Pct = 10
	return j
}

func (q *Queue) Get(id string) (*Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	j, ok := q.jobs[id]
	return j, ok
}

func (q *Queue) List() []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()
	out := make([]*Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		out = append(out, j)
	}
	return out
}
