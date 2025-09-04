package dataloader

import (
	"context"
	"sync"
	"time"
)

// BatchLoader is a simple batch loader without caching
type BatchLoader[K comparable, V any] struct {
	fetch    func(context.Context, []K) ([]V, []error)
	wait     time.Duration
	maxBatch int

	mu    sync.Mutex
	batch []batchRequest[K, V]
	timer *time.Timer
}

type batchRequest[K comparable, V any] struct {
	key    K
	result chan result[V]
}

type result[V any] struct {
	value V
	err   error
}

// Ptr is a tiny helper to get pointer to value types in loaders
func Ptr[T any](v T) *T { return &v }

// NewBatchLoader creates a new batch loader without caching
func NewBatchLoader[K comparable, V any](
	fetch func(context.Context, []K) ([]V, []error),
	wait time.Duration,
	maxBatch int,
) *BatchLoader[K, V] {
	if maxBatch <= 0 {
		maxBatch = 100
	}
	if wait <= 0 {
		wait = 2 * time.Millisecond
	}

	return &BatchLoader[K, V]{
		fetch:    fetch,
		wait:     wait,
		maxBatch: maxBatch,
	}
}

// Load loads a single value
func (l *BatchLoader[K, V]) Load(ctx context.Context, key K) (V, error) {
	result := make(chan result[V], 1)

	l.mu.Lock()

	// Add to batch
	l.batch = append(l.batch, batchRequest[K, V]{
		key:    key,
		result: result,
	})

	// If batch is full, execute immediately
	if len(l.batch) >= l.maxBatch {
		batch := l.batch
		l.batch = nil
		if l.timer != nil {
			l.timer.Stop()
			l.timer = nil
		}
		l.mu.Unlock()

		go l.executeBatch(ctx, batch)
	} else {
		// Start timer if not already started
		if l.timer == nil {
			l.timer = time.AfterFunc(l.wait, func() {
				l.mu.Lock()
				batch := l.batch
				l.batch = nil
				l.timer = nil
				l.mu.Unlock()

				if len(batch) > 0 {
					l.executeBatch(ctx, batch)
				}
			})
		}
		l.mu.Unlock()
	}

	// Wait for result
	select {
	case r := <-result:
		return r.value, r.err
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	}
}

// LoadAll loads multiple values
func (l *BatchLoader[K, V]) LoadAll(ctx context.Context, keys []K) ([]V, error) {
	results := make([]V, len(keys))
	errors := make([]error, len(keys))

	var wg sync.WaitGroup
	wg.Add(len(keys))

	for i, key := range keys {
		go func(idx int, k K) {
			defer wg.Done()
			results[idx], errors[idx] = l.Load(ctx, k)
		}(i, key)
	}

	wg.Wait()

	// Check if any errors occurred
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (l *BatchLoader[K, V]) executeBatch(ctx context.Context, batch []batchRequest[K, V]) {
	if len(batch) == 0 {
		return
	}

	// Extract keys
	keys := make([]K, len(batch))
	for i, req := range batch {
		keys[i] = req.key
	}

	// Execute fetch
	values, errors := l.fetch(ctx, keys)

	// Send results
	for i, req := range batch {
		r := result[V]{}
		if i < len(values) {
			r.value = values[i]
		}
		if i < len(errors) && errors[i] != nil {
			r.err = errors[i]
		}

		select {
		case req.result <- r:
		default:
			// Request was cancelled
		}
		close(req.result)
	}
}
