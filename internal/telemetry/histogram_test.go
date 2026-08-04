package telemetry

import (
	"strings"
	"testing"
)

func TestHistogramFamilyWritesCumulativePrometheusBuckets(t *testing.T) {
	family := NewHistogramFamily([]float64{0.1, 1})
	labels := map[string]string{"route": `/api/items/:id`, "outcome": "success"}
	family.Observe(labels, 0.05)
	family.Observe(labels, 0.5)

	var output strings.Builder
	if err := family.WritePrometheus(&output, "request_seconds", "Request time."); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`request_seconds_bucket{le="0.1",outcome="success",route="/api/items/:id"} 1`,
		`request_seconds_bucket{le="1",outcome="success",route="/api/items/:id"} 2`,
		`request_seconds_bucket{le="+Inf",outcome="success",route="/api/items/:id"} 2`,
		`request_seconds_sum{outcome="success",route="/api/items/:id"} 0.55`,
		`request_seconds_count{outcome="success",route="/api/items/:id"} 2`,
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("missing %q in:\n%s", expected, output.String())
		}
	}
}

func TestHistogramFamilyIgnoresInvalidObservations(t *testing.T) {
	family := NewHistogramFamily(nil)
	family.Observe(nil, -1)
	_, series := family.Snapshot()
	if len(series) != 0 {
		t.Fatalf("series=%#v", series)
	}
}
