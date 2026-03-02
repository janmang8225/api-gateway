package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type RouteMetrics struct {
	RequestCount atomic.Uint64
	ErrorCount   atomic.Uint64
	TotalLatency atomic.Int64 // in microseconds
}

type Metrics struct {
	mu     sync.RWMutex
	routes map[string]*RouteMetrics
}

func New() *Metrics {
	return &Metrics{
		routes: make(map[string]*RouteMetrics),
	}
}

func (m *Metrics) getOrCreate(path string) *RouteMetrics {
	m.mu.RLock()
	rm, exists := m.routes[path]
	m.mu.RUnlock()

	if exists {
		return rm
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// double check after acquiring write lock
	rm, exists = m.routes[path]
	if exists {
		return rm
	}

	rm = &RouteMetrics{}
	m.routes[path] = rm
	return rm
}

func (m *Metrics) Record(path string, status int, latency time.Duration) {
	rm := m.getOrCreate(path)
	rm.RequestCount.Add(1)
	rm.TotalLatency.Add(latency.Microseconds())

	if status >= 400 {
		rm.ErrorCount.Add(1)
	}
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain")

		for path, rm := range m.routes {
			count := rm.RequestCount.Load()
			errors := rm.ErrorCount.Load()
			totalLatency := rm.TotalLatency.Load()

			var avgLatency float64
			if count > 0 {
				avgLatency = float64(totalLatency) / float64(count)
			}

			fmt.Fprintf(w, "route: %s\n", path)
			fmt.Fprintf(w, "  requests:      %d\n", count)
			fmt.Fprintf(w, "  errors:        %d\n", errors)
			fmt.Fprintf(w, "  avg_latency:   %.2f µs\n", avgLatency)
			fmt.Fprintf(w, "\n")
		}
	})
}