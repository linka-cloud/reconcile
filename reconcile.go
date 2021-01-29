package reconcile

import (
	"context"
	"time"

	"go.linka.cloud/reconcile/object"
)

// EventType is Event triggering cause
type EventType int

func (e EventType) String() string {
	switch e {
	case Created:
		return "created"
	case Updated:
		return "updated"
	case Deleted:
		return "deleted"
	default:
		return "unknown"
	}
}

const (
	// Created is the event type when a resource is added
	Created EventType = iota
	// Updated is the event type when a resource is modified
	Updated
	// Deleted is the event type when a resource is removed
	Deleted
)

// Event contains the context needed by a watch loop
type Event interface {
	// Type is the type of event
	Type() EventType
	// New is the new Data, should be nil on delete
	New() object.Any
	// Old is the replaced Data, should be nil on create
	Old() object.Any
	// Revision is the revision number (incremented on each revision)
	Revision() int
}

// Storage is a read-write storage
type Storage interface {
	RStorage
	WStorage
}

// RStorage is a read-only storage
type RStorage interface {
	// Read reads the resource using the resource object as search definition
	Read(ctx context.Context, resource object.Any) error
	// List returns the resource objects matching the given the resource object as list filter definition
	List(ctx context.Context, resource object.Any) ([]object.Any, error)
	// Watch returns an event channel streaming events matching the resource object as filter definition
	Watch(ctx context.Context, resource object.Any) (<-chan Event, error)

	Close() error
}

// WStorage is a write-only storage
type WStorage interface {
	// Create creates a new resource
	Create(ctx context.Context, resource object.Any) error
	// Update updates the resource object, returns error when not found
	Update(ctx context.Context, resource object.Any) error
	// Delete deletes the given resource object, returns error when not found
	Delete(ctx context.Context, resource object.Any) error

	Close() error
}

// Informer holds the storage events callbacks
type Informer interface {
	// OnCreate is called when a resource is created
	OnCreate(new object.Any) error
	// OnUpdate is called when a resource is modified
	OnUpdate(new object.Any, old object.Any) error
	// OnDelete is called when a resource is deleted
	OnDelete(old object.Any) error

	Close() error
}

// Cache is read-only in-memory storage providing resource watching through the Informer interface
type Cache interface {
	RStorage
	Informer
}

// Result is the object returned by the Reconcile function
// It holds instructions for re-queuing or forgetting the reconcile event
type Result struct {
	// Requeue should be true if the reconcile event should be requeue after some times
	// Always true when the Reconciler returns an error
	Requeue bool
	// RequeueAfter defines when the reconcile event should be requeue
	RequeueAfter time.Duration
}

// Reconciler is a generic reconcile interface, exposing the reconcile function
// It returns a Result to know if the resource should be added back to the queue or forgotten
// If error is not nil, Result.Requeue will always be interpreted as true
// Use (Result{}, nil) to not requeue a reconcile event
type Reconciler interface {
	Reconcile(ctx context.Context, o object.Any) (Result, error)
}

// ReconcilerFunc is the main Reconciler function as a type implementing the Reconciler interface.
type ReconcilerFunc func(ctx context.Context, o object.Any) (Result, error)

// Reconcile implements the Reconciler interface
func (r ReconcilerFunc) Reconcile(ctx context.Context, o object.Any) (Result, error) {
	return r(ctx, o)
}


