package missioncontrol

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	RejectionCodeStepValidationFailed RejectionCode = "step_validation_failed"
	RejectionCodeFalseCompletionClaim RejectionCode = "false_completion_claim"

	RuntimePauseReasonStepComplete = "step_complete"
)

type WaitingUserInputKind string

const (
	WaitingUserInputNone          WaitingUserInputKind = ""
	WaitingUserInputApproval      WaitingUserInputKind = "approval"
	WaitingUserInputRejection     WaitingUserInputKind = "rejection"
	WaitingUserInputClarification WaitingUserInputKind = "clarification"
	WaitingUserInputTimeout       WaitingUserInputKind = "timeout"
)

type StepValidatorKind string

const (
	StepValidatorKindDiscussion  StepValidatorKind = "discussion"
	StepValidatorKindOneShotCode StepValidatorKind = "one_shot_code"
	// The Frank spec names the validator contract wait_user; the runtime state remains waiting_user.
	StepValidatorKindWaitUser      StepValidatorKind = "wait_user"
	StepValidatorKindFinalResponse StepValidatorKind = "final_response"
)

type RuntimeToolCallEvidence struct {
	ToolName  string
	Arguments map[string]interface{}
}

type StepValidationInput struct {
	FinalResponse   string
	UserInput       string
	UserInputKind   WaitingUserInputKind
	SuccessfulTools []RuntimeToolCallEvidence
}

type stepValidationResult struct {
	recordCompletion bool
}

func CompleteRuntimeStep(ec ExecutionContext, now time.Time, input StepValidationInput) (JobRuntimeState, error) {
	if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "step completion requires active job, step, and runtime state",
		}
	}
	if ec.Runtime.JobID != "" && ec.Runtime.JobID != ec.Job.ID {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime job %q does not match active job %q", ec.Runtime.JobID, ec.Job.ID),
		}
	}
	if ec.Runtime.ActiveStepID == "" || ec.Runtime.ActiveStepID != ec.Step.ID {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime active step %q does not match active execution step %q", ec.Runtime.ActiveStepID, ec.Step.ID),
		}
	}

	switch stepValidatorKind(ec) {
	case StepValidatorKindDiscussion:
		return completeRunningStep(ec, now, input)
	case StepValidatorKindWaitUser:
		return completeWaitUserStep(ec, now, input)
	case StepValidatorKindOneShotCode:
		return completeRunningStep(ec, now, input)
	case StepValidatorKindFinalResponse:
		return completeRunningStep(ec, now, input)
	default:
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("cannot validate step completion while job is %q", ec.Runtime.State),
		}
	}
}

func ClassifyWaitingUserInput(input string) WaitingUserInputKind {
	normalized := strings.ToLower(strings.TrimSpace(input))
	switch {
	case normalized == "":
		return WaitingUserInputNone
	case containsAnyPhrase(normalized, "timeout", "timed out"):
		return WaitingUserInputTimeout
	case containsAnyPhrase(normalized, "approve", "approved", "go ahead", "proceed", "authorized", "authorization granted"):
		return WaitingUserInputApproval
	case containsAnyPhrase(normalized, "reject", "rejected", "decline", "declined", "deny", "denied", "stop", "do not proceed", "don't proceed"):
		return WaitingUserInputRejection
	case containsAnyPhrase(normalized, "clarify", "clarification", "need more detail", "need more details", "define"):
		return WaitingUserInputClarification
	default:
		return WaitingUserInputNone
	}
}

func completeRunningStep(ec ExecutionContext, now time.Time, input StepValidationInput) (JobRuntimeState, error) {
	if ec.Runtime.State != JobStateRunning {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			StepID:  ec.Step.ID,
			Message: fmt.Sprintf("step %q requires running runtime state, got %q", ec.Step.ID, ec.Runtime.State),
		}
	}

	switch ec.Step.Type {
	case StepTypeDiscussion:
		return completeDiscussionStep(ec, now, input)
	case StepTypeOneShotCode:
		if !hasOneShotCodeArtifactEvidence(input.SuccessfulTools) {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeStepValidationFailed,
				StepID:  ec.Step.ID,
				Message: "one_shot_code completion requires artifact or code-change evidence",
			}
		}
		if !hasOneShotCodeVerificationEvidence(input.SuccessfulTools) {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeStepValidationFailed,
				StepID:  ec.Step.ID,
				Message: "one_shot_code completion requires validation, run, compile, read, or stat evidence",
			}
		}
		return pauseAfterValidatedCompletion(ec, now)
	case StepTypeFinalResponse:
		finalResponse := strings.TrimSpace(input.FinalResponse)
		if finalResponse == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeStepValidationFailed,
				StepID:  ec.Step.ID,
				Message: "final_response completion requires a non-empty response",
			}
		}
		if isFalseCompletionClaim(finalResponse) {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeFalseCompletionClaim,
				StepID:  ec.Step.ID,
				Message: "final_response must contain the actual response, not a bare completion claim",
			}
		}
		if !hasTruthfulFinalResponseShape(finalResponse) {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeStepValidationFailed,
				StepID:  ec.Step.ID,
				Message: "final_response must include minimally truthful user-facing result content",
			}
		}
		return TransitionJobRuntime(*CloneJobRuntimeState(ec.Runtime), JobStateCompleted, now, RuntimeTransitionOptions{
			StepID:           ec.Step.ID,
			validationResult: &stepValidationResult{recordCompletion: true},
		})
	default:
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: fmt.Sprintf("no runtime validator is implemented for step type %q", ec.Step.Type),
		}
	}
}

func completeDiscussionStep(ec ExecutionContext, now time.Time, input StepValidationInput) (JobRuntimeState, error) {
	if strings.TrimSpace(input.FinalResponse) == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "discussion completion requires a non-empty response",
		}
	}
	if hasDiscussionSideEffects(input.SuccessfulTools) {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "discussion completion cannot include side-effecting tool executions",
		}
	}

	if waitingReason, ok := discussionWaitingReason(ec.Step.Subtype); ok {
		return TransitionJobRuntime(*CloneJobRuntimeState(ec.Runtime), JobStateWaitingUser, now, RuntimeTransitionOptions{
			StepID:        ec.Step.ID,
			WaitingReason: waitingReason,
		})
	}

	return pauseAfterValidatedCompletion(ec, now)
}

func completeWaitUserStep(ec ExecutionContext, now time.Time, input StepValidationInput) (JobRuntimeState, error) {
	if ec.Runtime.State != JobStateWaitingUser {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			StepID:  ec.Step.ID,
			Message: fmt.Sprintf("wait_user completion requires waiting_user runtime state, got %q", ec.Runtime.State),
		}
	}

	inputKind := input.UserInputKind
	if inputKind == WaitingUserInputNone {
		inputKind = ClassifyWaitingUserInput(input.UserInput)
	}
	if inputKind == WaitingUserInputNone {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "waiting_user completion requires approval, rejection, clarification, or timeout input",
		}
	}

	return pauseAfterValidatedCompletion(ec, now)
}

func pauseAfterValidatedCompletion(ec ExecutionContext, now time.Time) (JobRuntimeState, error) {
	return TransitionJobRuntime(*CloneJobRuntimeState(ec.Runtime), JobStatePaused, now, RuntimeTransitionOptions{
		StepID:           ec.Step.ID,
		PausedReason:     RuntimePauseReasonStepComplete,
		validationResult: &stepValidationResult{recordCompletion: true},
	})
}

func discussionWaitingReason(subtype StepSubtype) (string, bool) {
	switch subtype {
	case StepSubtypeBlocker:
		return "discussion_blocker", true
	case StepSubtypeAuthorization:
		return "discussion_authorization", true
	case StepSubtypeDefinition:
		return "discussion_definition", true
	default:
		return "", false
	}
}

func isFalseCompletionClaim(response string) bool {
	normalized := strings.Trim(strings.ToLower(strings.TrimSpace(response)), ".! ")
	switch normalized {
	case "done", "completed", "finished", "all set":
		return true
	}

	return strings.Contains(normalized, "completed processing") ||
		strings.Contains(normalized, "completed the task") ||
		strings.Contains(normalized, "task completed") ||
		strings.Contains(normalized, "no response to give")
}

func hasTruthfulFinalResponseShape(response string) bool {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return false
	}

	structured := strings.Contains(trimmed, "\n") || strings.Contains(trimmed, ":") || strings.Contains(trimmed, "`")
	words := normalizedWords(trimmed)
	if len(words) < 5 && !structured {
		return false
	}
	if isMetaOnlyFinalResponse(words) {
		return false
	}
	return true
}

func isMetaOnlyFinalResponse(words []string) bool {
	metaWords := map[string]struct{}{
		"i": {}, "we": {}, "have": {}, "completed": {}, "complete": {}, "done": {}, "finished": {},
		"handled": {}, "implemented": {}, "processed": {}, "provided": {}, "here": {}, "is": {},
		"the": {}, "final": {}, "answer": {}, "response": {}, "result": {}, "task": {}, "request": {},
		"change": {}, "work": {}, "your": {}, "this": {}, "that": {}, "it": {}, "all": {}, "set": {},
	}
	for _, word := range words {
		if _, ok := metaWords[word]; !ok {
			return false
		}
	}
	return true
}

func normalizedWords(input string) []string {
	return strings.FieldsFunc(strings.ToLower(input), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func stepValidatorKind(ec ExecutionContext) StepValidatorKind {
	if ec.Runtime != nil && ec.Runtime.State == JobStateWaitingUser {
		return StepValidatorKindWaitUser
	}
	if ec.Step == nil {
		return ""
	}
	switch ec.Step.Type {
	case StepTypeDiscussion:
		return StepValidatorKindDiscussion
	case StepTypeOneShotCode:
		return StepValidatorKindOneShotCode
	case StepTypeFinalResponse:
		return StepValidatorKindFinalResponse
	default:
		return ""
	}
}

func hasDiscussionSideEffects(tools []RuntimeToolCallEvidence) bool {
	for _, tool := range tools {
		switch tool.ToolName {
		case "exec", "message", "write_memory", "edit_memory", "delete_memory", "cron", "spawn", "create_skill", "delete_skill":
			return true
		case "filesystem":
			if toolArgString(tool.Arguments, "action") == "write" {
				return true
			}
		}
	}
	return false
}

func hasOneShotCodeArtifactEvidence(tools []RuntimeToolCallEvidence) bool {
	for _, tool := range tools {
		switch tool.ToolName {
		case "filesystem":
			if toolArgString(tool.Arguments, "action") == "write" {
				return true
			}
		case "exec":
			cmd := toolArgStringSlice(tool.Arguments, "cmd")
			if len(cmd) > 0 {
				switch filepathBase(cmd[0]) {
				case "frank_finish", "frank_py_finish":
					return true
				}
			}
		}
	}
	return false
}

func hasOneShotCodeVerificationEvidence(tools []RuntimeToolCallEvidence) bool {
	for _, tool := range tools {
		switch tool.ToolName {
		case "filesystem":
			action := toolArgString(tool.Arguments, "action")
			if action == "read" || action == "stat" {
				return true
			}
		case "exec":
			cmd := toolArgStringSlice(tool.Arguments, "cmd")
			if isVerificationCommand(cmd) {
				return true
			}
		}
	}
	return false
}

func isVerificationCommand(cmd []string) bool {
	if len(cmd) == 0 {
		return false
	}

	base := filepathBase(cmd[0])
	switch base {
	case "frank_py_run", "frank_py_finish":
		return true
	case "go":
		if len(cmd) > 1 {
			switch cmd[1] {
			case "test", "build", "run":
				return true
			}
		}
	case "python", "python3":
		if len(cmd) > 2 && cmd[1] == "-m" && cmd[2] == "py_compile" {
			return true
		}
		if len(cmd) > 1 && strings.HasSuffix(cmd[1], ".py") {
			return true
		}
	}

	return false
}

func toolArgString(args map[string]interface{}, key string) string {
	if args == nil {
		return ""
	}
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	s, _ := value.(string)
	return s
}

func toolArgStringSlice(args map[string]interface{}, key string) []string {
	if args == nil {
		return nil
	}
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			s, ok := item.(string)
			if !ok {
				return nil
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

func filepathBase(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

func containsAnyPhrase(input string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(input, phrase) {
			return true
		}
	}
	return false
}
