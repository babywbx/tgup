package mcp

import (
	"context"
	"sync"
	"time"
)

// JobStore is a minimal in-memory MCP job store for bootstrap.
type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

// NewJobStore creates a bootstrap in-memory job store.
func NewJobStore() *JobStore {
	return &JobStore{jobs: make(map[string]Job)}
}

// Put inserts or updates a job.
func (s *JobStore) Put(_ context.Context, job Job) Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.UpdatedAt = time.Now().UTC()
	s.jobs[job.ID] = job
	return job
}

// Get returns one job by id.
func (s *JobStore) Get(_ context.Context, id string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}
