package source

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallelMapOrderedBoundsConcurrencyAndPreservesOrder(t *testing.T) {
	var active atomic.Int32
	var peak atomic.Int32
	outcomes := parallelMapOrdered(
		context.Background(),
		[]int{4, 3, 2, 1},
		2,
		func(_ context.Context, value int) (int, error) {
			current := active.Add(1)
			for {
				observed := peak.Load()
				if current <= observed || peak.CompareAndSwap(observed, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			active.Add(-1)
			return value * 10, nil
		},
	)

	if peak.Load() != 2 {
		t.Fatalf("peak concurrency=%d, want 2", peak.Load())
	}
	for index, expected := range []int{40, 30, 20, 10} {
		if outcomes[index].err != nil || outcomes[index].value != expected {
			t.Fatalf("outcome %d=%#v, want %d", index, outcomes[index], expected)
		}
	}
}
