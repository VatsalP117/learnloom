package telemetry

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// HistogramFamily is a small, dependency-free Prometheus histogram family.
// Callers must keep label values bounded; every distinct label set is retained
// for the lifetime of the process.
type HistogramFamily struct {
	mu      sync.Mutex
	buckets []float64
	series  map[string]*histogramSeries
}

type histogramSeries struct {
	labels  map[string]string
	buckets []uint64
	count   uint64
	sum     float64
}

type HistogramSeries struct {
	Labels  map[string]string
	Buckets []uint64
	Count   uint64
	Sum     float64
}

func NewHistogramFamily(buckets []float64) *HistogramFamily {
	result := &HistogramFamily{}
	result.buckets = normalizedBuckets(buckets)
	return result
}

func (f *HistogramFamily) Observe(labels map[string]string, value float64) {
	if f == nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.buckets) == 0 {
		f.buckets = normalizedBuckets(nil)
	}
	if f.series == nil {
		f.series = make(map[string]*histogramSeries)
	}
	key := labelKey(labels)
	series := f.series[key]
	if series == nil {
		series = &histogramSeries{
			labels:  cloneLabels(labels),
			buckets: make([]uint64, len(f.buckets)),
		}
		f.series[key] = series
	}
	series.count++
	series.sum += value
	for index, upperBound := range f.buckets {
		if value <= upperBound {
			series.buckets[index]++
		}
	}
}

func (f *HistogramFamily) Snapshot() ([]float64, []HistogramSeries) {
	if f == nil {
		return normalizedBuckets(nil), nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	buckets := append([]float64(nil), f.buckets...)
	if len(buckets) == 0 {
		buckets = normalizedBuckets(nil)
	}
	keys := make([]string, 0, len(f.series))
	for key := range f.series {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	series := make([]HistogramSeries, 0, len(keys))
	for _, key := range keys {
		value := f.series[key]
		series = append(series, HistogramSeries{
			Labels:  cloneLabels(value.labels),
			Buckets: append([]uint64(nil), value.buckets...),
			Count:   value.count,
			Sum:     value.sum,
		})
	}
	return buckets, series
}

func (f *HistogramFamily) WritePrometheus(
	writer io.Writer,
	name string,
	help string,
) error {
	if _, err := fmt.Fprintf(writer, "# HELP %s %s\n# TYPE %s histogram\n", name, help, name); err != nil {
		return err
	}
	buckets, allSeries := f.Snapshot()
	for _, series := range allSeries {
		for index, upperBound := range buckets {
			labels := cloneLabels(series.Labels)
			labels["le"] = strconv.FormatFloat(upperBound, 'g', -1, 64)
			if _, err := fmt.Fprintf(
				writer,
				"%s_bucket%s %d\n",
				name,
				prometheusLabels(labels),
				series.Buckets[index],
			); err != nil {
				return err
			}
		}
		infiniteLabels := cloneLabels(series.Labels)
		infiniteLabels["le"] = "+Inf"
		if _, err := fmt.Fprintf(
			writer,
			"%s_bucket%s %d\n%s_sum%s %.9g\n%s_count%s %d\n",
			name,
			prometheusLabels(infiniteLabels),
			series.Count,
			name,
			prometheusLabels(series.Labels),
			series.Sum,
			name,
			prometheusLabels(series.Labels),
			series.Count,
		); err != nil {
			return err
		}
	}
	return nil
}

func normalizedBuckets(values []float64) []float64 {
	if len(values) == 0 {
		values = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 1800, 3600}
	}
	result := append([]float64(nil), values...)
	sort.Float64s(result)
	return result
}

func labelKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var result strings.Builder
	for _, key := range keys {
		result.WriteString(key)
		result.WriteByte(0)
		result.WriteString(labels[key])
		result.WriteByte(0)
	}
	return result.String()
}

func prometheusLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.NewReplacer(
			`\`, `\\`,
			"\n", `\n`,
			`"`, `\"`,
		).Replace(labels[key])
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, value))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func cloneLabels(labels map[string]string) map[string]string {
	result := make(map[string]string, len(labels))
	for key, value := range labels {
		result[key] = value
	}
	return result
}
