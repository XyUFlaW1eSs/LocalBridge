package syncstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	fileVersion     = 1
	maxStoreBytes   = 16 * 1024 * 1024
	maxPayloadBytes = 4 * 1024 * 1024
)

type Store struct {
	path      string
	maxJobs   int
	retention time.Duration

	mu   sync.RWMutex
	jobs map[string]Job
}

func New(path string, maxJobs int, retention time.Duration) (*Store, error) {
	if path == "" || maxJobs < 1 || retention <= 0 {
		return nil, errors.New("invalid sync store configuration")
	}
	s := &Store{path: path, maxJobs: maxJobs, retention: retention, jobs: make(map[string]Job)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Create(envelope Envelope) (Job, error) {
	if envelope.ID == "" || envelope.Kind == "" || len(envelope.Payload) > maxPayloadBytes {
		return Job{}, errors.New("invalid sync envelope")
	}
	if envelope.CreatedAt.IsZero() {
		envelope.CreatedAt = time.Now().UTC()
	}
	now := time.Now().UTC()
	job := Job{ID: envelope.ID, Envelope: envelope, State: StatePending, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.jobs[job.ID]; ok {
		return existing, nil
	}
	if len(s.jobs) >= s.maxJobs {
		s.pruneLocked(now)
	}
	if len(s.jobs) >= s.maxJobs {
		return Job{}, errors.New("sync job store is full")
	}
	s.jobs[job.ID] = job
	if err := s.saveLocked(); err != nil {
		delete(s.jobs, job.ID)
		return Job{}, err
	}
	return job, nil
}

func (s *Store) Update(id string, state State, lastError string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, errors.New("sync job not found")
	}
	if !validTransition(job.State, state) {
		return Job{}, fmt.Errorf("invalid sync job transition: %s -> %s", job.State, state)
	}
	job.State = state
	job.LastError = lastError
	job.UpdatedAt = time.Now().UTC()
	if state == StateDelivering {
		job.Attempts++
	}
	s.jobs[id] = job
	if err := s.saveLocked(); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (s *Store) Get(id string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}

func (s *Store) List(limit int) []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].UpdatedAt.After(jobs[j].UpdatedAt) })
	if limit > 0 && len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return jobs
}

func (s *Store) Prune() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := len(s.jobs)
	s.pruneLocked(time.Now().UTC())
	if len(s.jobs) == before {
		return nil
	}
	return s.saveLocked()
}

func validTransition(from, to State) bool {
	if from == to {
		return true
	}
	switch from {
	case StatePending:
		return to == StateDelivering || to == StateCanceled || to == StateExpired
	case StateDelivering:
		return to == StateDelivered || to == StateFailed || to == StateCanceled || to == StateExpired
	case StateFailed:
		return to == StateDelivering || to == StateCanceled || to == StateExpired
	default:
		return false
	}
}

func (s *Store) pruneLocked(now time.Time) {
	for id, job := range s.jobs {
		if now.Sub(job.UpdatedAt) > s.retention || (job.Envelope.ExpiresAt != nil && now.After(*job.Envelope.ExpiresAt)) {
			delete(s.jobs, id)
		}
	}
	if len(s.jobs) <= s.maxJobs {
		return
	}
	jobs := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].UpdatedAt.Before(jobs[j].UpdatedAt) })
	for len(jobs) > s.maxJobs {
		delete(s.jobs, jobs[0].ID)
		jobs = jobs[1:]
	}
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read sync job store: %w", err)
	}
	if len(data) > maxStoreBytes {
		return fmt.Errorf("sync job store exceeds %d bytes", maxStoreBytes)
	}
	var state fileState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse sync job store: %w", err)
	}
	if state.Version != fileVersion {
		return fmt.Errorf("unsupported sync job store version: %d", state.Version)
	}
	for _, job := range state.Jobs {
		if job.ID != "" && len(job.Envelope.Payload) <= maxPayloadBytes {
			s.jobs[job.ID] = job
		}
	}
	s.mu.Lock()
	s.pruneLocked(time.Now().UTC())
	s.mu.Unlock()
	return nil
}

func (s *Store) saveLocked() error {
	jobs := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	data, err := json.MarshalIndent(fileState{Version: fileVersion, Jobs: jobs}, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxStoreBytes {
		return fmt.Errorf("sync job store exceeds %d bytes", maxStoreBytes)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create sync job store directory: %w", err)
	}
	return os.WriteFile(s.path, append(data, '\n'), 0600)
}
