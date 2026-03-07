// Package metrics provides Prometheus-compatible metrics collection without external dependencies.
package metrics

import (
	"math"
	"sync"
	"sync/atomic"
)

// Counter is a monotonically increasing value.
type Counter struct {
	name  string
	help  string
	value atomic.Int64
}

// NewCounter creates a new counter metric.
func NewCounter(name, help string) *Counter {
	return &Counter{
		name: name,
		help: help,
	}
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	c.value.Add(1)
}

// Add increments the counter by the given value.
func (c *Counter) Add(v int64) {
	c.value.Add(v)
}

// Value returns the current counter value.
func (c *Counter) Value() int64 {
	return c.value.Load()
}

// Name returns the metric name.
func (c *Counter) Name() string {
	return c.name
}

// Help returns the metric help text.
func (c *Counter) Help() string {
	return c.help
}

// Gauge can go up and down.
type Gauge struct {
	name  string
	help  string
	value atomic.Uint64 // stores float64 bits
}

// NewGauge creates a new gauge metric.
func NewGauge(name, help string) *Gauge {
	return &Gauge{
		name: name,
		help: help,
	}
}

// Set sets the gauge to the given value.
func (g *Gauge) Set(v float64) {
	g.value.Store(math.Float64bits(v))
}

// Inc increments the gauge by 1.
func (g *Gauge) Inc() {
	for {
		oldBits := g.value.Load()
		oldVal := math.Float64frombits(oldBits)
		newVal := oldVal + 1
		newBits := math.Float64bits(newVal)
		if g.value.CompareAndSwap(oldBits, newBits) {
			return
		}
	}
}

// Dec decrements the gauge by 1.
func (g *Gauge) Dec() {
	for {
		oldBits := g.value.Load()
		oldVal := math.Float64frombits(oldBits)
		newVal := oldVal - 1
		newBits := math.Float64bits(newVal)
		if g.value.CompareAndSwap(oldBits, newBits) {
			return
		}
	}
}

// Value returns the current gauge value.
func (g *Gauge) Value() float64 {
	return math.Float64frombits(g.value.Load())
}

// Name returns the metric name.
func (g *Gauge) Name() string {
	return g.name
}

// Help returns the metric help text.
func (g *Gauge) Help() string {
	return g.help
}

// Histogram tracks distribution of values in buckets.
type Histogram struct {
	name    string
	help    string
	buckets []float64
	counts  []atomic.Int64 // one per bucket + inf
	sum     atomic.Uint64  // float64 bits
	count   atomic.Int64
	mu      sync.Mutex
}

// NewHistogram creates a new histogram metric with the given buckets.
// Buckets must be sorted in increasing order.
func NewHistogram(name, help string, buckets []float64) *Histogram {
	h := &Histogram{
		name:    name,
		help:    help,
		buckets: make([]float64, len(buckets)),
		counts:  make([]atomic.Int64, len(buckets)+1), // +1 for +Inf
	}
	copy(h.buckets, buckets)
	return h
}

// Observe adds a single observation to the histogram.
func (h *Histogram) Observe(v float64) {
	h.count.Add(1)

	for {
		oldBits := h.sum.Load()
		oldVal := math.Float64frombits(oldBits)
		newVal := oldVal + v
		newBits := math.Float64bits(newVal)
		if h.sum.CompareAndSwap(oldBits, newBits) {
			break
		}
	}

	for i, bound := range h.buckets {
		if v <= bound {
			h.counts[i].Add(1)
		}
	}
	h.counts[len(h.buckets)].Add(1)
}

// Count returns the total number of observations.
func (h *Histogram) Count() int64 {
	return h.count.Load()
}

// Sum returns the sum of all observed values.
func (h *Histogram) Sum() float64 {
	return math.Float64frombits(h.sum.Load())
}

// Buckets returns the bucket boundaries.
func (h *Histogram) Buckets() []float64 {
	return h.buckets
}

// BucketCount returns the count for the i-th bucket.
func (h *Histogram) BucketCount(i int) int64 {
	if i < 0 || i >= len(h.counts) {
		return 0
	}
	return h.counts[i].Load()
}

// Name returns the metric name.
func (h *Histogram) Name() string {
	return h.name
}

// Help returns the metric help text.
func (h *Histogram) Help() string {
	return h.help
}

// Registry holds all metrics.
type Registry struct {
	mu         sync.RWMutex
	counters   []*Counter
	gauges     []*Gauge
	histograms []*Histogram
}

// NewRegistry creates a new metrics registry.
func NewRegistry() *Registry {
	return &Registry{
		counters:   make([]*Counter, 0),
		gauges:     make([]*Gauge, 0),
		histograms: make([]*Histogram, 0),
	}
}

// NewCounter creates and registers a new counter.
func (r *Registry) NewCounter(name, help string) *Counter {
	c := NewCounter(name, help)
	r.mu.Lock()
	r.counters = append(r.counters, c)
	r.mu.Unlock()
	return c
}

// NewGauge creates and registers a new gauge.
func (r *Registry) NewGauge(name, help string) *Gauge {
	g := NewGauge(name, help)
	r.mu.Lock()
	r.gauges = append(r.gauges, g)
	r.mu.Unlock()
	return g
}

// NewHistogram creates and registers a new histogram.
func (r *Registry) NewHistogram(name, help string, buckets []float64) *Histogram {
	h := NewHistogram(name, help, buckets)
	r.mu.Lock()
	r.histograms = append(r.histograms, h)
	r.mu.Unlock()
	return h
}

// Counters returns all registered counters.
func (r *Registry) Counters() []*Counter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Counter, len(r.counters))
	copy(result, r.counters)
	return result
}

// Gauges returns all registered gauges.
func (r *Registry) Gauges() []*Gauge {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Gauge, len(r.gauges))
	copy(result, r.gauges)
	return result
}

// Histograms returns all registered histograms.
func (r *Registry) Histograms() []*Histogram {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Histogram, len(r.histograms))
	copy(result, r.histograms)
	return result
}

// DefaultRegistry is the global default registry.
var DefaultRegistry = NewRegistry()

// DefaultBuckets are the default histogram buckets (in seconds).
var DefaultBuckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Pre-defined application metrics.
var (
	RequestsTotal   *Counter
	RequestDuration *Histogram
	ActiveRequests  *Gauge
)

func init() {
	RequestsTotal = DefaultRegistry.NewCounter(
		"http_requests_total",
		"Total number of HTTP requests",
	)
	RequestDuration = DefaultRegistry.NewHistogram(
		"http_request_duration_seconds",
		"HTTP request duration in seconds",
		DefaultBuckets,
	)
	ActiveRequests = DefaultRegistry.NewGauge(
		"http_active_requests",
		"Number of active HTTP requests",
	)
}
