package flow_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ITProLabDev/ethbacknode/tools/flow"
)

var errBoom = errors.New("boom")

// --- Stage (direct) -------------------------------------------------------

func TestStage_RunsStepAndUpdatesPayload(t *testing.T) {
	proc := flow.Stage[int]("double", func(_ context.Context, v int) (int, error) {
		return v * 2, nil
	})
	out := proc.Process(&flow.Envelope[int]{Ctx: context.Background(), Payload: 21})
	env, ok := out.(*flow.Envelope[int])
	if !ok {
		t.Fatalf("Stage must return *Envelope[int], got %T", out)
	}
	if env.Err != nil {
		t.Fatalf("unexpected error: %v", env.Err)
	}
	if env.Payload != 42 {
		t.Fatalf("payload = %d, want 42", env.Payload)
	}
}

func TestStage_CapturesErrorAsStageError(t *testing.T) {
	proc := flow.Stage[int]("validate", func(_ context.Context, v int) (int, error) {
		return 999, errBoom // returned payload must be ignored on error
	})
	out := proc.Process(&flow.Envelope[int]{Ctx: context.Background(), Payload: 7})
	env := out.(*flow.Envelope[int])

	if env.Err == nil {
		t.Fatal("expected error to be captured")
	}
	if !errors.Is(env.Err, errBoom) {
		t.Fatalf("errors.Is(errBoom) failed: %v", env.Err)
	}
	var se *flow.StageError
	if !errors.As(env.Err, &se) {
		t.Fatalf("error must be a *StageError, got %T", env.Err)
	}
	if se.Stage != "validate" {
		t.Fatalf("StageError.Stage = %q, want %q", se.Stage, "validate")
	}
	if env.Payload != 7 {
		t.Fatalf("payload must be preserved on error: got %d, want 7", env.Payload)
	}
}

func TestStage_ShortCircuitsWhenErrAlreadySet(t *testing.T) {
	called := false
	proc := flow.Stage[int]("never", func(_ context.Context, v int) (int, error) {
		called = true
		return v, nil
	})
	in := &flow.Envelope[int]{Ctx: context.Background(), Payload: 5, Err: errBoom}
	out := proc.Process(in).(*flow.Envelope[int])

	if called {
		t.Fatal("step must not run when Err is already set")
	}
	if out.Payload != 5 {
		t.Fatalf("payload must be untouched: got %d", out.Payload)
	}
	if !errors.Is(out.Err, errBoom) {
		t.Fatalf("existing error must be preserved: %v", out.Err)
	}
}

func TestStage_ShortCircuitsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	proc := flow.Stage[int]("guarded", func(_ context.Context, v int) (int, error) {
		called = true
		return v, nil
	})
	out := proc.Process(&flow.Envelope[int]{Ctx: ctx, Payload: 5}).(*flow.Envelope[int])

	if called {
		t.Fatal("step must not run when context is cancelled")
	}
	if !errors.Is(out.Err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", out.Err)
	}
	var se *flow.StageError
	if !errors.As(out.Err, &se) || se.Stage != "guarded" {
		t.Fatalf("cancellation must be wrapped as StageError naming the stage, got %v", out.Err)
	}
}

// --- Sequence -------------------------------------------------------------

func TestSequence_ThreadsPayloadInOrder(t *testing.T) {
	var order []string
	out, err := flow.Sequence(context.Background(), 1,
		flow.NamedStep[int]{Name: "a", Step: func(_ context.Context, v int) (int, error) {
			order = append(order, "a")
			return v + 1, nil
		}},
		flow.NamedStep[int]{Name: "b", Step: func(_ context.Context, v int) (int, error) {
			order = append(order, "b")
			return v * 10, nil
		}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != 20 { // (1+1)*10
		t.Fatalf("out = %d, want 20", out)
	}
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("steps ran out of order: %v", order)
	}
}

func TestSequence_StopsAtFirstError(t *testing.T) {
	cRan := false
	out, err := flow.Sequence(context.Background(), 1,
		flow.NamedStep[int]{Name: "a", Step: func(_ context.Context, v int) (int, error) { return v + 1, nil }},
		flow.NamedStep[int]{Name: "b", Step: func(_ context.Context, v int) (int, error) { return 0, errBoom }},
		flow.NamedStep[int]{Name: "c", Step: func(_ context.Context, v int) (int, error) { cRan = true; return v, nil }},
	)
	if cRan {
		t.Fatal("step c must not run after b errors")
	}
	if !errors.Is(err, errBoom) {
		t.Fatalf("want errBoom, got %v", err)
	}
	var se *flow.StageError
	if !errors.As(err, &se) || se.Stage != "b" {
		t.Fatalf("error must name failing stage b, got %v", err)
	}
	if out != 2 { // a applied (1+1), b preserved input on error
		t.Fatalf("out = %d, want 2 (last good payload)", out)
	}
}

func TestSequence_NoSteps_ReturnsInputUnchanged(t *testing.T) {
	out, err := flow.Sequence(context.Background(), 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != 99 {
		t.Fatalf("out = %d, want 99", out)
	}
}

func TestSequence_CancelledMidway_StopsRemaining(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var laterRan int32

	out, err := flow.Sequence(ctx, 1,
		flow.NamedStep[int]{Name: "first", Step: func(_ context.Context, v int) (int, error) {
			cancel() // cancel during the pipeline
			return v + 1, nil
		}},
		flow.NamedStep[int]{Name: "second", Step: func(_ context.Context, v int) (int, error) {
			atomic.AddInt32(&laterRan, 1)
			return v + 1, nil
		}},
		flow.NamedStep[int]{Name: "third", Step: func(_ context.Context, v int) (int, error) {
			atomic.AddInt32(&laterRan, 1)
			return v + 1, nil
		}},
	)
	if n := atomic.LoadInt32(&laterRan); n != 0 {
		t.Fatalf("steps after cancellation ran %d times, want 0", n)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if out != 2 { // only "first" applied
		t.Fatalf("out = %d, want 2", out)
	}
	var se *flow.StageError
	if !errors.As(err, &se) || se.Stage != "second" {
		t.Fatalf("cancellation should surface at stage second, got %v", err)
	}
}

func TestSequence_NilContext_UsesBackground(t *testing.T) {
	//lint:ignore SA1012 deliberately passing nil to verify it is replaced with Background.
	out, err := flow.Sequence[int](nil, 3, //nolint:staticcheck
		flow.NamedStep[int]{Name: "inc", Step: func(ctx context.Context, v int) (int, error) {
			if ctx == nil {
				t.Fatal("step received nil context")
			}
			return v + 1, nil
		}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != 4 {
		t.Fatalf("out = %d, want 4", out)
	}
}

// --- FanOut ---------------------------------------------------------------

func TestFanOut_AppliesToAllItems(t *testing.T) {
	out, err := flow.FanOut(context.Background(), []int{1, 2, 3}, 2,
		func(_ context.Context, v int) (int, error) { return v * v, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 4, 9}
	if !equalInts(out, want) {
		t.Fatalf("out = %v, want %v", out, want)
	}
}

func TestFanOut_PreservesOrder(t *testing.T) {
	items := []int{0, 1, 2, 3, 4}
	// Item 0 sleeps longest so it finishes last; order must still hold by index.
	out, err := flow.FanOut(context.Background(), items, 0,
		func(_ context.Context, v int) (int, error) {
			time.Sleep(time.Duration(len(items)-v) * 5 * time.Millisecond)
			return v * 10, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{0, 10, 20, 30, 40}
	if !equalInts(out, want) {
		t.Fatalf("out = %v, want %v", out, want)
	}
}

func TestFanOut_BoundsConcurrency(t *testing.T) {
	const concurrency = 3
	var cur, max int32
	items := make([]int, 20)
	_, err := flow.FanOut(context.Background(), items, concurrency,
		func(_ context.Context, v int) (int, error) {
			n := atomic.AddInt32(&cur, 1)
			for {
				m := atomic.LoadInt32(&max)
				if n <= m || atomic.CompareAndSwapInt32(&max, m, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			atomic.AddInt32(&cur, -1)
			return v, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&max); got > concurrency {
		t.Fatalf("observed %d concurrent steps, must not exceed %d", got, concurrency)
	}
}

func TestFanOut_RunsConcurrently(t *testing.T) {
	const concurrency = 3
	var entered int32
	ready := make(chan struct{})
	var closedOnce sync.Once
	var didClose int32

	items := make([]int, 2*concurrency)
	_, err := flow.FanOut(context.Background(), items, concurrency,
		func(_ context.Context, v int) (int, error) {
			if atomic.AddInt32(&entered, 1) == concurrency {
				closedOnce.Do(func() { atomic.StoreInt32(&didClose, 1); close(ready) })
			}
			select {
			case <-ready:
			case <-time.After(2 * time.Second):
			}
			return v, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&didClose) != 1 {
		t.Fatalf("fewer than %d steps ran concurrently — no real parallelism", concurrency)
	}
}

func TestFanOut_UnlimitedWhenConcurrencyZero(t *testing.T) {
	out, err := flow.FanOut(context.Background(), []int{1, 2, 3, 4}, 0,
		func(_ context.Context, v int) (int, error) { return v + 1, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalInts(out, []int{2, 3, 4, 5}) {
		t.Fatalf("out = %v", out)
	}
}

func TestFanOut_ReturnsFirstError(t *testing.T) {
	out, err := flow.FanOut(context.Background(), []int{0, 1, 2, 3, 4}, 2,
		func(_ context.Context, v int) (int, error) {
			if v == 1 || v == 3 {
				return 0, errBoom
			}
			return v + 100, nil
		})
	if !errors.Is(err, errBoom) {
		t.Fatalf("want errBoom, got %v", err)
	}
	var se *flow.StageError
	if !errors.As(err, &se) {
		t.Fatalf("want *StageError, got %T", err)
	}
	// Results must still be present; failed items keep their original payload.
	want := []int{100, 1, 102, 3, 104}
	if !equalInts(out, want) {
		t.Fatalf("out = %v, want %v", out, want)
	}
}

func TestFanOut_Empty_ReturnsEmptyNoError(t *testing.T) {
	out, err := flow.FanOut(context.Background(), []int{}, 4,
		func(_ context.Context, v int) (int, error) { return v, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("out = %v, want empty", out)
	}
}

func TestFanOut_CancelledContext_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var ran int32
	out, err := flow.FanOut(ctx, []int{1, 2, 3}, 2,
		func(_ context.Context, v int) (int, error) {
			atomic.AddInt32(&ran, 1)
			return v + 1, nil
		})
	if n := atomic.LoadInt32(&ran); n != 0 {
		t.Fatalf("step ran %d times despite cancelled context, want 0", n)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if !equalInts(out, []int{1, 2, 3}) {
		t.Fatalf("payloads must be preserved on cancellation: %v", out)
	}
}

// --- StageError -----------------------------------------------------------

func TestStageError_MessageAndUnwrap(t *testing.T) {
	se := &flow.StageError{Stage: "x", Err: errBoom}
	if !errors.Is(se, errBoom) {
		t.Fatal("Unwrap must expose the wrapped error")
	}
	if se.Error() == "" {
		t.Fatal("Error() must be non-empty")
	}
}

// --- helpers --------------------------------------------------------------

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
