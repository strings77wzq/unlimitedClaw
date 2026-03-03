package metrics

import (
	"fmt"
	"io"
	"net/http"
	"sort"
)

// Handler returns an HTTP handler that exposes metrics in Prometheus exposition format.
func Handler(reg *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		writeMetrics(w, reg)
	})
}

func writeMetrics(w io.Writer, reg *Registry) {
	counters := reg.Counters()
	gauges := reg.Gauges()
	histograms := reg.Histograms()

	sort.Slice(counters, func(i, j int) bool {
		return counters[i].Name() < counters[j].Name()
	})
	sort.Slice(gauges, func(i, j int) bool {
		return gauges[i].Name() < gauges[j].Name()
	})
	sort.Slice(histograms, func(i, j int) bool {
		return histograms[i].Name() < histograms[j].Name()
	})

	for _, c := range counters {
		fmt.Fprintf(w, "# HELP %s %s\n", c.Name(), c.Help())
		fmt.Fprintf(w, "# TYPE %s counter\n", c.Name())
		fmt.Fprintf(w, "%s %d\n", c.Name(), c.Value())
		fmt.Fprintln(w)
	}

	for _, g := range gauges {
		fmt.Fprintf(w, "# HELP %s %s\n", g.Name(), g.Help())
		fmt.Fprintf(w, "# TYPE %s gauge\n", g.Name())
		fmt.Fprintf(w, "%s %g\n", g.Name(), g.Value())
		fmt.Fprintln(w)
	}

	for _, h := range histograms {
		fmt.Fprintf(w, "# HELP %s %s\n", h.Name(), h.Help())
		fmt.Fprintf(w, "# TYPE %s histogram\n", h.Name())

		buckets := h.Buckets()
		for i, bound := range buckets {
			count := h.BucketCount(i)
			fmt.Fprintf(w, "%s_bucket{le=\"%g\"} %d\n", h.Name(), bound, count)
		}
		infCount := h.BucketCount(len(buckets))
		fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", h.Name(), infCount)
		fmt.Fprintf(w, "%s_sum %g\n", h.Name(), h.Sum())
		fmt.Fprintf(w, "%s_count %d\n", h.Name(), h.Count())
		fmt.Fprintln(w)
	}
}
