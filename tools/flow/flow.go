// Package flow is a thin, type-safe layer over github.com/pysyun/go_pysyun_pipeline.
//
// The underlying library executes stages that exchange untyped any values and
// offers neither context cancellation nor an error channel. flow keeps that
// library as the execution engine but enforces a project convention: exactly
// one typed container — the Envelope — travels through a pipeline. The any
// boxing the library requires happens only at its boundary; stage authors work
// with typed payloads, a context.Context, and explicit errors.
//
// Two entry points cover the common shapes:
//
//   - Sequence runs steps as a synchronous linear chain.
//   - FanOut runs one step over many items with bounded, order-preserving
//     concurrency.
package flow

import (
	"context"
	"fmt"

	pipeline "github.com/pysyun/go_pysyun_pipeline"
)

// Envelope is the single typed container passed through a flow pipeline.
//
// Convention: stages accept and return *Envelope[T], never a bare payload.
// Ctx carries cancellation/deadlines; Err holds the first error encountered
// (which short-circuits all later stages); FailedStage names where it arose.
type Envelope[T any] struct {
	Ctx         context.Context
	Payload     T
	Err         error
	FailedStage string
}

// Step is the typed processing function authored by callers. Returning a
// non-nil error halts the pipeline; the returned payload is ignored on error.
type Step[T any] func(ctx context.Context, payload T) (T, error)

// NamedStep pairs a Step with a label used for diagnostics and StageError.
type NamedStep[T any] struct {
	Name string
	Step Step[T]
}

// StageError wraps an error with the name of the stage that produced it.
type StageError struct {
	Stage string
	Err   error
}

func (e *StageError) Error() string {
	return fmt.Sprintf("stage %q: %v", e.Stage, e.Err)
}

// Unwrap exposes the wrapped error to errors.Is / errors.As.
func (e *StageError) Unwrap() error { return e.Err }

// Stage adapts a typed Step into a pipeline.Processor operating on *Envelope[T].
//
// It short-circuits when the envelope already carries an error, honors context
// cancellation before running the step, and wraps any returned error as a
// *StageError naming this stage (preserving the inbound payload on failure).
func Stage[T any](name string, step Step[T]) pipeline.Processor {
	return pipeline.ProcessorFunc(func(data any) any {
		env, ok := data.(*Envelope[T])
		if !ok {
			// Programming error: only *Envelope[T] may cross a stage boundary.
			panic(fmt.Sprintf("flow: stage %q received %T, want *flow.Envelope[%T]", name, data, *new(T)))
		}
		if env.Err != nil {
			return env
		}
		ctx := env.Ctx
		if ctx == nil {
			ctx = context.Background()
		}
		if err := ctx.Err(); err != nil {
			env.Err = &StageError{Stage: name, Err: err}
			env.FailedStage = name
			return env
		}
		out, err := step(ctx, env.Payload)
		if err != nil {
			env.Err = &StageError{Stage: name, Err: err}
			env.FailedStage = name
			return env
		}
		env.Payload = out
		return env
	})
}

// Sequence runs steps as a synchronous linear chain, threading a single
// Envelope through each. It stops at the first error and returns the last
// good payload alongside that error. A nil ctx is treated as context.Background.
func Sequence[T any](ctx context.Context, in T, steps ...NamedStep[T]) (T, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	env := &Envelope[T]{Ctx: ctx, Payload: in}

	stages := make([]*pipeline.Chainable, len(steps))
	for i, ns := range steps {
		stages[i] = pipeline.NewChainable(Stage(ns.Name, ns.Step))
	}

	out := pipeline.Pipe(stages...).Process(env).(*Envelope[T])
	return out.Payload, out.Err
}

// FanOut runs step over items with bounded, order-preserving concurrency,
// delegating the fan-out to pipeline.ChainableGroup. concurrency <= 0 spawns
// one goroutine per item. Every item's resulting payload is returned in input
// order; the first error (by item index) is returned, if any. Items that fail
// keep their original payload in the returned slice. A nil ctx is treated as
// context.Background.
func FanOut[T any](ctx context.Context, items []T, concurrency int, step Step[T]) ([]T, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	results := make([]T, len(items))
	if len(items) == 0 {
		return results, nil
	}

	boxed := make([]any, len(items))
	for i, it := range items {
		boxed[i] = &Envelope[T]{Ctx: ctx, Payload: it}
	}

	group := pipeline.NewChainableGroup(concurrency).
		Pipe(pipeline.NewChainable(Stage("fanout", step)))
	processed := group.Process(boxed)

	var firstErr error
	for i, p := range processed {
		env := p.(*Envelope[T])
		results[i] = env.Payload
		if firstErr == nil && env.Err != nil {
			firstErr = env.Err
		}
	}
	return results, firstErr
}
