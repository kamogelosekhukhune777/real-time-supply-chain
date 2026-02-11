// Package metrics constructs the metrics the application will track.
package metrics

import (
	"context"
	"expvar"
	"runtime"
)

// metrics represents the set of metrics we gather.
// Expvar handles atomic updates, so we don't need Mutexes here.
type metrics struct {
	goroutines *expvar.Int
	requests   *expvar.Int
	errors     *expvar.Int
	panics     *expvar.Int
}

// m is our singleton. Since expvar.NewInt registers globally
// with the HTTP server, we keep this private.
var m *metrics

func init() {
	m = &metrics{
		goroutines: expvar.NewInt("goroutines"),
		requests:   expvar.NewInt("requests"),
		errors:     expvar.NewInt("errors"),
		panics:     expvar.NewInt("panics"),
	}
}

// UpdateGoroutines captures the current snapshot of running goroutines.
func UpdateGoroutines(ctx context.Context) int64 {
	g := int64(runtime.NumGoroutine())
	get(ctx).goroutines.Set(g)
	return g
}

// AddRequest increments the request count.
func AddRequest(ctx context.Context) int64 {
	met := get(ctx)
	met.requests.Add(1)
	return met.requests.Value()
}

// AddError increments the error count.
func AddError(ctx context.Context) int64 {
	met := get(ctx)
	met.errors.Add(1)
	return met.errors.Value()
}

// AddPanic increments the panic count.
func AddPanic(ctx context.Context) int64 {
	met := get(ctx)
	met.panics.Add(1)
	return met.panics.Value()
}
