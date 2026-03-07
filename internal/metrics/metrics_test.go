package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestCounterIncrement(t *testing.T) {
	c := NewCounter("test_counter", "Test counter")

	if c.Value() != 0 {
		t.Errorf("Expected initial value 0, got %d", c.Value())
	}

	c.Inc()
	if c.Value() != 1 {
		t.Errorf("Expected value 1 after Inc(), got %d", c.Value())
	}

	c.Add(5)
	if c.Value() != 6 {
		t.Errorf("Expected value 6 after Add(5), got %d", c.Value())
	}
}

func TestGaugeSetIncDec(t *testing.T) {
	g := NewGauge("test_gauge", "Test gauge")

	if g.Value() != 0 {
		t.Errorf("Expected initial value 0, got %f", g.Value())
	}

	g.Set(10.5)
	if g.Value() != 10.5 {
		t.Errorf("Expected value 10.5 after Set(10.5), got %f", g.Value())
	}

	g.Inc()
	if g.Value() != 11.5 {
		t.Errorf("Expected value 11.5 after Inc(), got %f", g.Value())
	}

	g.Dec()
	if g.Value() != 10.5 {
		t.Errorf("Expected value 10.5 after Dec(), got %f", g.Value())
	}
}

func TestHistogramObserve(t *testing.T) {
	buckets := []float64{1, 5, 10}
	h := NewHistogram("test_histogram", "Test histogram", buckets)

	h.Observe(0.5)
	h.Observe(3.0)
	h.Observe(7.0)
	h.Observe(15.0)

	if h.Count() != 4 {
		t.Errorf("Expected count 4, got %d", h.Count())
	}

	expectedSum := 0.5 + 3.0 + 7.0 + 15.0
	if h.Sum() != expectedSum {
		t.Errorf("Expected sum %f, got %f", expectedSum, h.Sum())
	}

	if h.BucketCount(0) != 1 {
		t.Errorf("Expected bucket[0] count 1, got %d", h.BucketCount(0))
	}

	if h.BucketCount(1) != 2 {
		t.Errorf("Expected bucket[1] count 2, got %d", h.BucketCount(1))
	}

	if h.BucketCount(2) != 3 {
		t.Errorf("Expected bucket[2] count 3, got %d", h.BucketCount(2))
	}

	if h.BucketCount(3) != 4 {
		t.Errorf("Expected bucket[3] (+Inf) count 4, got %d", h.BucketCount(3))
	}
}

func TestRegistryNewMetrics(t *testing.T) {
	reg := NewRegistry()

	c := reg.NewCounter("registry_counter", "Counter from registry")
	g := reg.NewGauge("registry_gauge", "Gauge from registry")
	h := reg.NewHistogram("registry_histogram", "Histogram from registry", []float64{1, 10})

	counters := reg.Counters()
	if len(counters) != 1 {
		t.Errorf("Expected 1 counter, got %d", len(counters))
	}
	if counters[0] != c {
		t.Error("Counter not found in registry")
	}

	gauges := reg.Gauges()
	if len(gauges) != 1 {
		t.Errorf("Expected 1 gauge, got %d", len(gauges))
	}
	if gauges[0] != g {
		t.Error("Gauge not found in registry")
	}

	histograms := reg.Histograms()
	if len(histograms) != 1 {
		t.Errorf("Expected 1 histogram, got %d", len(histograms))
	}
	if histograms[0] != h {
		t.Error("Histogram not found in registry")
	}
}

func TestMetricsHandler(t *testing.T) {
	reg := NewRegistry()
	c := reg.NewCounter("test_requests_total", "Total requests")
	g := reg.NewGauge("test_active_connections", "Active connections")
	h := reg.NewHistogram("test_duration_seconds", "Request duration", []float64{0.1, 1.0})

	c.Add(42)
	g.Set(3)
	h.Observe(0.05)
	h.Observe(0.5)
	h.Observe(2.0)

	handler := Handler(reg)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	expectedContentType := "text/plain; version=0.0.4; charset=utf-8"
	if contentType != expectedContentType {
		t.Errorf("Expected Content-Type %q, got %q", expectedContentType, contentType)
	}

	body := w.Body.String()

	expectedStrings := []string{
		"# HELP test_requests_total Total requests",
		"# TYPE test_requests_total counter",
		"test_requests_total 42",
		"# HELP test_active_connections Active connections",
		"# TYPE test_active_connections gauge",
		"test_active_connections 3",
		"# HELP test_duration_seconds Request duration",
		"# TYPE test_duration_seconds histogram",
		"test_duration_seconds_bucket{le=\"0.1\"} 1",
		"test_duration_seconds_bucket{le=\"1\"} 2",
		"test_duration_seconds_bucket{le=\"+Inf\"} 3",
		"test_duration_seconds_sum 2.55",
		"test_duration_seconds_count 3",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(body, expected) {
			t.Errorf("Expected output to contain %q, body:\n%s", expected, body)
		}
	}
}

func TestMetricsMiddleware(t *testing.T) {
	reg := NewRegistry()

	RequestsTotal = reg.NewCounter("http_requests_total", "Total requests")
	ActiveRequests = reg.NewGauge("http_active_requests", "Active requests")
	RequestDuration = reg.NewHistogram("http_request_duration_seconds", "Request duration", DefaultBuckets)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := MetricsMiddleware(reg)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if RequestsTotal.Value() != 1 {
		t.Errorf("Expected RequestsTotal 1, got %d", RequestsTotal.Value())
	}

	if ActiveRequests.Value() != 0 {
		t.Errorf("Expected ActiveRequests 0 after request completes, got %f", ActiveRequests.Value())
	}

	if RequestDuration.Count() != 1 {
		t.Errorf("Expected RequestDuration count 1, got %d", RequestDuration.Count())
	}
}

func TestConcurrentMetrics(t *testing.T) {
	c := NewCounter("concurrent_counter", "Concurrent counter")
	g := NewGauge("concurrent_gauge", "Concurrent gauge")
	h := NewHistogram("concurrent_histogram", "Concurrent histogram", []float64{1, 10})

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c.Inc()
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				g.Inc()
				g.Dec()
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				h.Observe(float64(j % 20))
			}
		}()
	}

	wg.Wait()

	expectedCounterValue := int64(goroutines * iterations)
	if c.Value() != expectedCounterValue {
		t.Errorf("Expected counter value %d, got %d", expectedCounterValue, c.Value())
	}

	if g.Value() != 0 {
		t.Errorf("Expected gauge value 0, got %f", g.Value())
	}

	expectedHistogramCount := int64(goroutines * iterations)
	if h.Count() != expectedHistogramCount {
		t.Errorf("Expected histogram count %d, got %d", expectedHistogramCount, h.Count())
	}
}
