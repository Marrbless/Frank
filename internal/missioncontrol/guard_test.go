package missioncontrol

import (
	"context"
	"testing"
)

func TestExecutionContextRoundTrip(t *testing.T) {
	t.Parallel()

	job := &Job{ID: "job-1"}
	step := &Step{ID: "step-1"}

	ctx := WithExecutionContext(context.Background(), ExecutionContext{
		Job:  job,
		Step: step,
	})

	got, ok := ExecutionContextFromContext(ctx)
	if !ok {
		t.Fatal("ExecutionContextFromContext() ok = false, want true")
	}

	if got.Job != job {
		t.Fatalf("ExecutionContextFromContext().Job = %p, want %p", got.Job, job)
	}

	if got.Step != step {
		t.Fatalf("ExecutionContextFromContext().Step = %p, want %p", got.Step, step)
	}
}

func TestExecutionContextFromContextMissing(t *testing.T) {
	t.Parallel()

	if _, ok := ExecutionContextFromContext(context.Background()); ok {
		t.Fatal("ExecutionContextFromContext() ok = true, want false")
	}
}
