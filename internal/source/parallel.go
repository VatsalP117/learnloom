package source

import (
	"context"
	"sync"
)

type parallelOutcome[T any] struct {
	value T
	err   error
}

func parallelMapOrdered[Input, Output any](
	ctx context.Context,
	inputs []Input,
	limit int,
	work func(context.Context, Input) (Output, error),
) []parallelOutcome[Output] {
	outcomes := make([]parallelOutcome[Output], len(inputs))
	if len(inputs) == 0 {
		return outcomes
	}
	if limit < 1 {
		limit = 1
	}
	limit = min(limit, len(inputs))

	jobs := make(chan int)
	var workers sync.WaitGroup
	workers.Add(limit)
	for range limit {
		go func() {
			defer workers.Done()
			for index := range jobs {
				value, err := work(ctx, inputs[index])
				outcomes[index] = parallelOutcome[Output]{value: value, err: err}
			}
		}()
	}
	for index := range inputs {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	return outcomes
}
