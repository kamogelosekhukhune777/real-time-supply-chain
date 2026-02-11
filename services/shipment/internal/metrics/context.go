package metrics

import "context"

// Use a custom type for context keys to avoid collisions.
type ctxKey int

const key ctxKey = 1

// Inject adds the metrics instance to the context.
// Note: In an event-driven system, call this at the consumer/entry point.
func Inject(ctx context.Context) context.Context {
	return context.WithValue(ctx, key, m)
}

// get retrieves the metrics from context or returns the global singleton.
// This ensures that even if Inject wasn't called, your code won't crash.
func get(ctx context.Context) *metrics {
	if v, ok := ctx.Value(key).(*metrics); ok {
		return v
	}
	return m
}
