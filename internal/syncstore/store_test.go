package syncstore

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLifecycleAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	store, err := New(path, 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create(Envelope{ID: "job-1", Kind: "clipboard.push", Type: "text", Payload: []byte(`{"content":"hello"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if job.State != StatePending {
		t.Fatalf("unexpected initial state: %s", job.State)
	}
	if _, err := store.Update(job.ID, StateDelivering, ""); err != nil {
		t.Fatal(err)
	}
	completed, err := store.Update(job.ID, StateDelivered, "")
	if err != nil {
		t.Fatal(err)
	}
	if completed.Attempts != 1 || completed.State != StateDelivered {
		t.Fatalf("unexpected completed job: %#v", completed)
	}
	reloaded, err := New(path, 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if loaded, ok := reloaded.Get(job.ID); !ok || loaded.State != StateDelivered {
		t.Fatalf("persisted job missing: %#v %v", loaded, ok)
	}
}

func TestStoreRejectsInvalidTransition(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "jobs.json"), 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create(Envelope{ID: "job-1", Kind: "test", Payload: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(job.ID, StateDelivered, ""); err == nil {
		t.Fatal("expected invalid pending -> delivered transition to fail")
	}
}
