package missioncontrol

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
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
	StepValidatorKindDiscussion      StepValidatorKind = "discussion"
	StepValidatorKindStaticArtifact  StepValidatorKind = "static_artifact"
	StepValidatorKindOneShotCode     StepValidatorKind = "one_shot_code"
	StepValidatorKindLongRunningCode StepValidatorKind = "long_running_code"
	StepValidatorKindSystemAction    StepValidatorKind = "system_action"
	// The Frank spec names the validator contract wait_user; the runtime state remains waiting_user.
	StepValidatorKindWaitUser      StepValidatorKind = "wait_user"
	StepValidatorKindFinalResponse StepValidatorKind = "final_response"
)

type RuntimeToolCallEvidence struct {
	ToolName  string
	Arguments map[string]interface{}
	Result    string
}

type StepValidationInput struct {
	FinalResponse   string
	UserInput       string
	UserInputKind   WaitingUserInputKind
	ApprovalVia     string
	SessionChannel  string
	SessionChatID   string
	SuccessfulTools []RuntimeToolCallEvidence
}

type stepValidationResult struct {
	recordCompletion bool
	resultingState   *RuntimeResultingStateRecord
	rollback         *RuntimeRollbackRecord
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
	if err := validateRuntimeActiveStepReplayMarkers(*ec.Runtime, "complete"); err != nil {
		return JobRuntimeState{}, err
	}

	if ec.Step != nil && ec.Step.Type == StepTypeWaitUser && ec.Runtime != nil && ec.Runtime.State == JobStateRunning {
		return completeRunningStep(ec, now, input)
	}

	switch stepValidatorKind(ec) {
	case StepValidatorKindDiscussion:
		return completeRunningStep(ec, now, input)
	case StepValidatorKindLongRunningCode:
		return completeRunningStep(ec, now, input)
	case StepValidatorKindSystemAction:
		return completeRunningStep(ec, now, input)
	case StepValidatorKindWaitUser:
		return completeWaitUserStep(ec, now, input)
	case StepValidatorKindStaticArtifact:
		return completeRunningStep(ec, now, input)
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
	case StepTypeLongRunningCode:
		if hasLongRunningStartEvidence(input.SuccessfulTools) {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeLongRunningStartForbidden,
				StepID:  ec.Step.ID,
				Message: "long_running_code must not start a process; move start/stop semantics to system_action",
			}
		}
		if err := validateLongRunningCodeCompletion(*ec.Step, input.SuccessfulTools); err != nil {
			return JobRuntimeState{}, err
		}
		return pauseAfterValidatedCompletion(ec, now)
	case StepTypeSystemAction:
		result, err := validateSystemActionCompletion(*ec.Job, *ec.Step, input.SuccessfulTools)
		if err != nil {
			return JobRuntimeState{}, err
		}
		return pauseAfterValidatedCompletionWithResult(ec, now, result)
	case StepTypeWaitUser:
		return enterWaitUserStep(ec, now, input)
	case StepTypeStaticArtifact:
		if err := validateStaticArtifactCompletion(*ec.Step, input.SuccessfulTools); err != nil {
			return JobRuntimeState{}, err
		}
		return pauseAfterValidatedCompletion(ec, now)
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
		nextRuntime := *CloneJobRuntimeState(ec.Runtime)
		if requestedAction, scope, requiresApproval := approvalBindingForStep(*ec.Step); requiresApproval {
			content, _ := approvalRequestContentForStep(*ec.Job, *ec.Step)
			if reusableGrant, ok := FindReusableApprovalGrant(nextRuntime, now, ec.Job.ID, *ec.Step, input.SessionChannel, input.SessionChatID); ok {
				nextRuntime = appendGrantedApprovalRequest(nextRuntime, now, ApprovalRequest{
					JobID:           ec.Job.ID,
					StepID:          ec.Step.ID,
					RequestedAction: requestedAction,
					Scope:           scope,
					Content:         &content,
					RequestedVia:    ApprovalRequestedViaRuntime,
					GrantedVia:      reusableGrant.GrantedVia,
					SessionChannel:  reusableGrant.SessionChannel,
					SessionChatID:   reusableGrant.SessionChatID,
					Reason:          waitingReason,
					ExpiresAt:       reusableGrant.ExpiresAt,
				})
				return pauseAfterValidatedCompletion(ExecutionContext{
					Job:     ec.Job,
					Step:    ec.Step,
					Runtime: &nextRuntime,
				}, now)
			}
			nextRuntime = appendPendingApprovalRequest(nextRuntime, now, ApprovalRequest{
				JobID:           ec.Job.ID,
				StepID:          ec.Step.ID,
				RequestedAction: requestedAction,
				Scope:           scope,
				Content:         &content,
				RequestedVia:    ApprovalRequestedViaRuntime,
				Reason:          waitingReason,
			})
		}

		return TransitionJobRuntime(nextRuntime, JobStateWaitingUser, now, RuntimeTransitionOptions{
			StepID:        ec.Step.ID,
			WaitingReason: waitingReason,
		})
	}

	return pauseAfterValidatedCompletion(ec, now)
}

func enterWaitUserStep(ec ExecutionContext, now time.Time, input StepValidationInput) (JobRuntimeState, error) {
	if strings.TrimSpace(input.FinalResponse) == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "wait_user completion requires a non-empty response",
		}
	}
	if hasDiscussionSideEffects(input.SuccessfulTools) {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "wait_user completion cannot include side-effecting tool executions",
		}
	}

	waitingReason, ok := waitUserWaitingReason(ec.Step.Subtype)
	if !ok {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "wait_user step requires blocker, authorization, or definition subtype",
		}
	}

	nextRuntime := *CloneJobRuntimeState(ec.Runtime)
	if requestedAction, scope, requiresApproval := approvalBindingForStep(*ec.Step); requiresApproval {
		content, _ := approvalRequestContentForStep(*ec.Job, *ec.Step)
		nextRuntime = appendPendingApprovalRequest(nextRuntime, now, ApprovalRequest{
			JobID:           ec.Job.ID,
			StepID:          ec.Step.ID,
			RequestedAction: requestedAction,
			Scope:           scope,
			Content:         &content,
			RequestedVia:    ApprovalRequestedViaRuntime,
			Reason:          waitingReason,
		})
	}

	return TransitionJobRuntime(nextRuntime, JobStateWaitingUser, now, RuntimeTransitionOptions{
		StepID:        ec.Step.ID,
		WaitingReason: waitingReason,
	})
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

	current := *CloneJobRuntimeState(ec.Runtime)
	current, _ = RefreshApprovalRequests(current, now)
	if requestedAction, scope, requiresApproval := approvalBindingForStep(*ec.Step); requiresApproval && hasPendingApprovalRequest(&current, ec.Job.ID, ec.Step.ID, requestedAction, scope) {
		switch inputKind {
		case WaitingUserInputApproval:
			if input.ApprovalVia == "" {
				return JobRuntimeState{}, ValidationError{
					Code:    RejectionCodeStepValidationFailed,
					StepID:  ec.Step.ID,
					Message: "pending approval requires explicit operator approve command",
				}
			}
			ec.Runtime = &current
			return ApplyApprovalDecision(ec, now, ApprovalDecisionApprove, input.ApprovalVia)
		case WaitingUserInputRejection:
			if input.ApprovalVia == "" {
				return JobRuntimeState{}, ValidationError{
					Code:    RejectionCodeStepValidationFailed,
					StepID:  ec.Step.ID,
					Message: "pending approval requires explicit operator deny command",
				}
			}
			ec.Runtime = &current
			return ApplyApprovalDecision(ec, now, ApprovalDecisionDeny, input.ApprovalVia)
		case WaitingUserInputTimeout:
			nextRuntime, ok := ExpireActiveApprovalRequest(current, now, ec.Job.ID, ec.Step.ID, requestedAction, scope)
			if !ok {
				return current, nil
			}
			return nextRuntime, nil
		default:
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeStepValidationFailed,
				StepID:  ec.Step.ID,
				Message: "pending approval requires explicit operator approve or deny command",
			}
		}
	}
	if requestedAction, scope, requiresApproval := approvalBindingForStep(*ec.Step); requiresApproval {
		if requestIndex, ok := findLatestApprovalRequest(current.ApprovalRequests, ec.Job.ID, ec.Step.ID, requestedAction, scope); ok {
			switch current.ApprovalRequests[requestIndex].State {
			case ApprovalStateDenied:
				return JobRuntimeState{}, ValidationError{
					Code:    RejectionCodeStepValidationFailed,
					StepID:  ec.Step.ID,
					Message: "denied approval requires a new approval request before the step can complete",
				}
			case ApprovalStateExpired:
				return JobRuntimeState{}, ValidationError{
					Code:    RejectionCodeStepValidationFailed,
					StepID:  ec.Step.ID,
					Message: "expired approval requires a new approval request before the step can complete",
				}
			case ApprovalStateSuperseded:
				return JobRuntimeState{}, ValidationError{
					Code:    RejectionCodeStepValidationFailed,
					StepID:  ec.Step.ID,
					Message: "superseded approval requires a new approval request before the step can complete",
				}
			}
		}
	}

	ec.Runtime = &current
	return pauseAfterValidatedCompletion(ec, now)
}

func pauseAfterValidatedCompletion(ec ExecutionContext, now time.Time) (JobRuntimeState, error) {
	return pauseAfterValidatedCompletionWithResult(ec, now, &stepValidationResult{recordCompletion: true})
}

func pauseAfterValidatedCompletionWithResult(ec ExecutionContext, now time.Time, result *stepValidationResult) (JobRuntimeState, error) {
	if result == nil {
		result = &stepValidationResult{recordCompletion: true}
	}
	return TransitionJobRuntime(*CloneJobRuntimeState(ec.Runtime), JobStatePaused, now, RuntimeTransitionOptions{
		StepID:           ec.Step.ID,
		PausedReason:     RuntimePauseReasonStepComplete,
		validationResult: result,
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

func waitUserWaitingReason(subtype StepSubtype) (string, bool) {
	switch subtype {
	case StepSubtypeBlocker:
		return "wait_user_blocker", true
	case StepSubtypeAuthorization:
		return "wait_user_authorization", true
	case StepSubtypeDefinition:
		return "wait_user_definition", true
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
	case StepTypeLongRunningCode:
		return StepValidatorKindLongRunningCode
	case StepTypeSystemAction:
		return StepValidatorKindSystemAction
	case StepTypeStaticArtifact:
		return StepValidatorKindStaticArtifact
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

func hasLongRunningStartEvidence(tools []RuntimeToolCallEvidence) bool {
	for _, tool := range tools {
		if tool.ToolName != "exec" {
			continue
		}
		if isLongRunningStartCommand(toolArgStringSlice(tool.Arguments, "cmd")) {
			return true
		}
	}
	return false
}

func validateLongRunningCodeCompletion(step Step, tools []RuntimeToolCallEvidence) error {
	if !hasLongRunningStartupCommand(step) {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: "long_running_code completion requires explicit startup command metadata",
		}
	}
	artifactPath := longRunningArtifactPath(step)
	if artifactPath == "" {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: "long_running_code completion requires explicit artifact contract metadata",
		}
	}
	if !hasExactArtifactWrite(tools, artifactPath) {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: fmt.Sprintf("long_running_code completion requires writing %q", artifactPath),
		}
	}
	if !hasExactArtifactExistenceEvidence(tools, artifactPath) {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: fmt.Sprintf("long_running_code completion requires proving %q exists", artifactPath),
		}
	}
	if !hasOneShotCodeVerificationEvidence(tools) {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: "long_running_code completion requires validation, compile, read, or stat evidence",
		}
	}
	return nil
}

func isLongRunningStartCommand(cmd []string) bool {
	if len(cmd) == 0 {
		return false
	}

	name := strings.ToLower(filepathBase(cmd[0]))
	switch name {
	case "npm", "yarn", "pnpm":
		return len(cmd) > 1 && strings.EqualFold(cmd[1], "start")
	case "go":
		return len(cmd) > 1 && strings.EqualFold(cmd[1], "run")
	case "systemctl":
		return len(cmd) > 1 && (strings.EqualFold(cmd[1], "start") || strings.EqualFold(cmd[1], "restart"))
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

type staticArtifactSpec struct {
	path   string
	format string
}

func validateStaticArtifactCompletion(step Step, tools []RuntimeToolCallEvidence) error {
	spec := inferStaticArtifactSpec(step, tools)
	if spec.path == "" {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: "static_artifact completion requires an exact artifact file path",
		}
	}
	if !hasExactArtifactWrite(tools, spec.path) {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: fmt.Sprintf("static_artifact completion requires writing %q", spec.path),
		}
	}
	if !hasExactArtifactExistenceEvidence(tools, spec.path) {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: fmt.Sprintf("static_artifact completion requires proving %q exists", spec.path),
		}
	}

	readResult, ok := exactArtifactReadResult(tools, spec.path)
	if !ok {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: fmt.Sprintf("static_artifact completion requires a successful structure check for %q", spec.path),
		}
	}
	if !matchesArtifactStructure(spec.format, readResult) {
		return ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: fmt.Sprintf("static_artifact completion requires %q to match the expected %s structure", spec.path, spec.format),
		}
	}
	return nil
}

func inferStaticArtifactSpec(step Step, tools []RuntimeToolCallEvidence) staticArtifactSpec {
	path := extractExpectedArtifactPath(step.SuccessCriteria)
	if path == "" {
		path = singleArtifactPath(tools)
	}

	format := inferArtifactFormat(path, step.SuccessCriteria)
	if format == "" {
		format = "text"
	}
	return staticArtifactSpec{path: path, format: format}
}

func extractExpectedArtifactPath(criteria []string) string {
	pathRE := regexp.MustCompile("`([^`]+)`|\"([^\"]+)\"|'([^']+)'|\\b[[:alnum:]_./-]+\\.[[:alnum:]]+\\b")
	for _, criterion := range criteria {
		matches := pathRE.FindStringSubmatch(criterion)
		if len(matches) == 0 {
			continue
		}
		for _, match := range matches[1:] {
			if strings.TrimSpace(match) != "" {
				return filepath.Clean(strings.TrimSpace(match))
			}
		}
		if strings.TrimSpace(matches[0]) != "" {
			return filepath.Clean(strings.Trim(matches[0], "`\"'"))
		}
	}
	return ""
}

func singleArtifactPath(tools []RuntimeToolCallEvidence) string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, 1)
	for _, tool := range tools {
		path := artifactPathForTool(tool)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		ordered = append(ordered, path)
	}
	if len(ordered) != 1 {
		return ""
	}
	return ordered[0]
}

func artifactPathForTool(tool RuntimeToolCallEvidence) string {
	switch tool.ToolName {
	case "filesystem":
		return cleanedArtifactPath(toolArgString(tool.Arguments, "path"))
	case "exec":
		cmd := toolArgStringSlice(tool.Arguments, "cmd")
		if len(cmd) < 2 {
			return ""
		}
		switch filepathBase(cmd[0]) {
		case "frank_finish", "frank_py_finish", "frank_py_run":
			return cleanedArtifactPath(cmd[1])
		}
	}
	return ""
}

func cleanedArtifactPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(path))
}

func inferArtifactFormat(path string, criteria []string) string {
	text := strings.ToLower(strings.Join(criteria, " "))
	switch {
	case strings.Contains(text, "json"):
		return "json"
	case strings.Contains(text, "yaml"), strings.Contains(text, "yml"):
		return "yaml"
	case strings.Contains(text, "markdown"), strings.Contains(text, ".md"):
		return "markdown"
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md", ".markdown":
		return "markdown"
	default:
		return "text"
	}
}

func hasExactArtifactWrite(tools []RuntimeToolCallEvidence, path string) bool {
	for _, tool := range tools {
		switch tool.ToolName {
		case "filesystem":
			if toolArgString(tool.Arguments, "action") == "write" && artifactPathForTool(tool) == path {
				return true
			}
		case "exec":
			cmd := toolArgStringSlice(tool.Arguments, "cmd")
			if len(cmd) >= 2 && artifactPathForTool(tool) == path {
				switch filepathBase(cmd[0]) {
				case "frank_finish", "frank_py_finish":
					return true
				}
			}
		}
	}
	return false
}

func hasExactArtifactExistenceEvidence(tools []RuntimeToolCallEvidence, path string) bool {
	for _, tool := range tools {
		switch tool.ToolName {
		case "filesystem":
			if artifactPathForTool(tool) != path {
				continue
			}
			switch toolArgString(tool.Arguments, "action") {
			case "read":
				if strings.TrimSpace(tool.Result) != "" {
					return true
				}
			case "stat":
				if strings.Contains(tool.Result, "exists=true") && strings.Contains(tool.Result, "kind=file") {
					return true
				}
			}
		case "exec":
			cmd := toolArgStringSlice(tool.Arguments, "cmd")
			if len(cmd) >= 2 && artifactPathForTool(tool) == path {
				switch filepathBase(cmd[0]) {
				case "frank_finish", "frank_py_finish", "frank_py_run":
					return true
				}
			}
		}
	}
	return false
}

func exactArtifactReadResult(tools []RuntimeToolCallEvidence, path string) (string, bool) {
	for _, tool := range tools {
		switch tool.ToolName {
		case "filesystem":
			if toolArgString(tool.Arguments, "action") == "read" && artifactPathForTool(tool) == path {
				if strings.TrimSpace(tool.Result) != "" {
					return tool.Result, true
				}
			}
		case "exec":
			cmd := toolArgStringSlice(tool.Arguments, "cmd")
			if len(cmd) >= 2 && artifactPathForTool(tool) == path && filepathBase(cmd[0]) == "frank_py_run" {
				if strings.TrimSpace(tool.Result) != "" {
					return tool.Result, true
				}
			}
		}
	}
	return "", false
}

func matchesArtifactStructure(format, content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}

	switch format {
	case "json":
		if !json.Valid([]byte(trimmed)) {
			return false
		}
		var decoded interface{}
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return false
		}
		switch decoded.(type) {
		case map[string]interface{}, []interface{}:
			return true
		default:
			return false
		}
	case "yaml":
		lines := strings.Split(trimmed, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, ":") || strings.HasPrefix(line, "- ") {
				return true
			}
		}
		return false
	case "markdown":
		lines := strings.Split(trimmed, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
				return true
			}
		}
		return len(normalizedWords(trimmed)) >= 5
	default:
		return len(normalizedWords(trimmed)) > 0
	}
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
