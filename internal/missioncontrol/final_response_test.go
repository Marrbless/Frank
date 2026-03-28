package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeFinalResponseAppendsDeterministicMissionSummary(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{ID: "alpha", Type: StepTypeStaticArtifact, StaticArtifactPath: "alpha.txt", StaticArtifactFormat: "text"},
		{ID: "beta", Type: StepTypeStaticArtifact, StaticArtifactPath: "beta.txt", StaticArtifactFormat: "txt"},
		{ID: "gamma", Type: StepTypeOneShotCode, OneShotArtifactPath: "gamma.txt"},
		{ID: "delta", Type: StepTypeLongRunningCode, LongRunningStartupCommand: []string{"./delta"}, LongRunningArtifactPath: "delta.txt"},
		{ID: "epsilon", Type: StepTypeStaticArtifact, StaticArtifactPath: "epsilon.txt", StaticArtifactFormat: "text"},
		{ID: "zeta", Type: StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
		{ID: "review", Type: StepTypeDiscussion, Subtype: StepSubtypeBlocker},
		{ID: "hold", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "followup", Type: StepTypeDiscussion, Subtype: StepSubtypeDefinition},
		{ID: "verify", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "publish", Type: StepTypeDiscussion, Subtype: StepSubtypeDefinition},
		{ID: "cleanup", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"cleanup"}},
	})

	ec := testStepValidationExecutionContextForJob(job, "final", JobStateRunning)
	ec.Runtime.CompletedSteps = []RuntimeStepRecord{
		{StepID: "zeta", At: time.Date(2026, 3, 27, 12, 6, 0, 0, time.UTC)},
		{StepID: "beta", At: time.Date(2026, 3, 27, 12, 2, 0, 0, time.UTC), ResultingState: &RuntimeResultingStateRecord{Kind: string(StepTypeStaticArtifact), Target: "beta.txt", State: "already_present"}},
		{StepID: "epsilon", At: time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC)},
		{StepID: "delta", At: time.Date(2026, 3, 27, 12, 4, 0, 0, time.UTC)},
		{StepID: "gamma", At: time.Date(2026, 3, 27, 12, 3, 0, 0, time.UTC)},
		{StepID: "alpha", At: time.Date(2026, 3, 27, 12, 1, 0, 0, time.UTC)},
	}
	ec.Runtime.FailedSteps = []RuntimeStepRecord{
		{StepID: "review", Reason: "validator failed", At: time.Date(2026, 3, 27, 12, 7, 0, 0, time.UTC)},
	}

	got := NormalizeFinalResponse(ec, "Here is the final answer with the requested outputs.")
	want := "Here is the final answer with the requested outputs.\n\nMission summary:\n" +
		"Blocked steps: review (validator failed)\n" +
		"Artifacts: alpha.txt; beta.txt (already present); gamma.txt; delta.txt; epsilon.txt; +1 more omitted\n" +
		"Pending steps: hold; followup; verify; publish; cleanup"
	if got != want {
		t.Fatalf("NormalizeFinalResponse() = %q, want %q", got, want)
	}
}

func TestNormalizeFinalResponseTruncatesRawBodyWithOmissionMarker(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:   "final",
		Type: StepTypeFinalResponse,
	}, JobStateRunning)

	raw := strings.Repeat("a", finalResponseBodyRuneLimit+7)
	got := NormalizeFinalResponse(ec, raw)
	want := strings.Repeat("a", finalResponseBodyRuneLimit) +
		"\n\n[final response body truncated; 7 characters omitted]"
	if got != want {
		t.Fatalf("NormalizeFinalResponse() = %q, want %q", got, want)
	}
}

func TestNormalizeFinalResponsePrioritizesBlockedStepsInsidePendingSummaryCap(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{ID: "alpha", Type: StepTypeDiscussion},
		{ID: "beta", Type: StepTypeDiscussion},
		{ID: "gamma", Type: StepTypeDiscussion},
		{ID: "delta", Type: StepTypeDiscussion},
		{ID: "epsilon", Type: StepTypeDiscussion},
		{ID: "zeta", Type: StepTypeDiscussion},
		{ID: "eta", Type: StepTypeDiscussion},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"eta"}},
	})

	ec := testStepValidationExecutionContextForJob(job, "final", JobStateRunning)
	ec.Runtime.FailedSteps = []RuntimeStepRecord{
		{StepID: "eta", Reason: "awaiting operator approval", At: time.Date(2026, 3, 27, 12, 7, 0, 0, time.UTC)},
	}

	got := NormalizeFinalResponse(ec, "Here is the final answer with the current mission state.")
	want := "Here is the final answer with the current mission state.\n\nMission summary:\n" +
		"Blocked steps: eta (awaiting operator approval)\n" +
		"Pending steps: alpha; beta; gamma; delta; epsilon; +1 more omitted"
	if got != want {
		t.Fatalf("NormalizeFinalResponse() = %q, want %q", got, want)
	}
}

func TestNormalizeFinalResponseShowsAllBlockedReasonsBeforeArtifacts(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{ID: "alpha", Type: StepTypeDiscussion},
		{ID: "beta", Type: StepTypeDiscussion},
		{ID: "gamma", Type: StepTypeDiscussion},
		{ID: "delta", Type: StepTypeDiscussion},
		{ID: "epsilon", Type: StepTypeDiscussion},
		{ID: "zeta", Type: StepTypeDiscussion},
		{ID: "artifact", Type: StepTypeStaticArtifact, StaticArtifactPath: "result.txt", StaticArtifactFormat: "text"},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"artifact"}},
	})

	ec := testStepValidationExecutionContextForJob(job, "final", JobStateRunning)
	ec.Runtime.CompletedSteps = []RuntimeStepRecord{
		{StepID: "artifact", At: time.Date(2026, 3, 27, 12, 8, 0, 0, time.UTC)},
	}
	ec.Runtime.FailedSteps = []RuntimeStepRecord{
		{StepID: "alpha", Reason: "missing input", At: time.Date(2026, 3, 27, 12, 1, 0, 0, time.UTC)},
		{StepID: "beta", Reason: "awaiting approval", At: time.Date(2026, 3, 27, 12, 2, 0, 0, time.UTC)},
		{StepID: "gamma", Reason: "validator failed", At: time.Date(2026, 3, 27, 12, 3, 0, 0, time.UTC)},
		{StepID: "delta", Reason: "operator denied", At: time.Date(2026, 3, 27, 12, 4, 0, 0, time.UTC)},
		{StepID: "epsilon", Reason: "timeout", At: time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC)},
		{StepID: "zeta", Reason: "blocked dependency", At: time.Date(2026, 3, 27, 12, 6, 0, 0, time.UTC)},
	}

	got := NormalizeFinalResponse(ec, "Here is the final answer with the blocked mission state.")
	want := "Here is the final answer with the blocked mission state.\n\nMission summary:\n" +
		"Blocked steps: alpha (missing input); beta (awaiting approval); gamma (validator failed); delta (operator denied); epsilon (timeout); zeta (blocked dependency)\n" +
		"Artifacts: result.txt"
	if got != want {
		t.Fatalf("NormalizeFinalResponse() = %q, want %q", got, want)
	}
}
