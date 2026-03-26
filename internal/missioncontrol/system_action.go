package missioncontrol

import (
	"fmt"
	"strings"
)

type SystemActionKind string

const (
	SystemActionKindService SystemActionKind = "service"
	SystemActionKindProcess SystemActionKind = "process"
)

type SystemActionOperation string

const (
	SystemActionOperationStart   SystemActionOperation = "start"
	SystemActionOperationStop    SystemActionOperation = "stop"
	SystemActionOperationInspect SystemActionOperation = "inspect"
	SystemActionOperationStatus  SystemActionOperation = "status"
)

type SystemAction struct {
	Kind         SystemActionKind       `json:"kind,omitempty"`
	Operation    SystemActionOperation  `json:"operation,omitempty"`
	Target       string                 `json:"target,omitempty"`
	Command      []string               `json:"command,omitempty"`
	SourceStepID string                 `json:"source_step_id,omitempty"`
	PostState    *SystemActionPostState `json:"post_state,omitempty"`
	Rollback     *SystemActionRollback  `json:"rollback,omitempty"`
}

type SystemActionPostState struct {
	Command         []string `json:"command,omitempty"`
	SuccessContains []string `json:"success_contains,omitempty"`
	FailureContains []string `json:"failure_contains,omitempty"`
}

type SystemActionRollback struct {
	Command []string `json:"command,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

type RuntimeResultingStateRecord struct {
	Kind                string   `json:"kind,omitempty"`
	Target              string   `json:"target,omitempty"`
	Operation           string   `json:"operation,omitempty"`
	State               string   `json:"state,omitempty"`
	ActionCommand       []string `json:"action_command,omitempty"`
	VerificationCommand []string `json:"verification_command,omitempty"`
	VerificationOutput  string   `json:"verification_output,omitempty"`
	SourceStepID        string   `json:"source_step_id,omitempty"`
}

type RuntimeRollbackRecord struct {
	Available bool     `json:"available"`
	Command   []string `json:"command,omitempty"`
	Reason    string   `json:"reason,omitempty"`
}

func cloneSystemActionSpec(action *SystemAction) *SystemAction {
	if action == nil {
		return nil
	}

	cloned := *action
	cloned.Command = append([]string(nil), action.Command...)
	if action.PostState != nil {
		postState := *action.PostState
		postState.Command = append([]string(nil), action.PostState.Command...)
		postState.SuccessContains = append([]string(nil), action.PostState.SuccessContains...)
		postState.FailureContains = append([]string(nil), action.PostState.FailureContains...)
		cloned.PostState = &postState
	} else {
		cloned.PostState = nil
	}
	if action.Rollback != nil {
		rollback := *action.Rollback
		rollback.Command = append([]string(nil), action.Rollback.Command...)
		cloned.Rollback = &rollback
	} else {
		cloned.Rollback = nil
	}
	return &cloned
}

func cloneRuntimeResultingStateRecord(record *RuntimeResultingStateRecord) *RuntimeResultingStateRecord {
	if record == nil {
		return nil
	}

	cloned := *record
	cloned.ActionCommand = append([]string(nil), record.ActionCommand...)
	cloned.VerificationCommand = append([]string(nil), record.VerificationCommand...)
	return &cloned
}

func cloneRuntimeRollbackRecord(record *RuntimeRollbackRecord) *RuntimeRollbackRecord {
	if record == nil {
		return nil
	}

	cloned := *record
	cloned.Command = append([]string(nil), record.Command...)
	return &cloned
}

func validateSystemActionStep(job Job, step Step) []ValidationError {
	if step.SystemAction == nil {
		return []ValidationError{{
			Code:    RejectionCodeInvalidStepType,
			StepID:  step.ID,
			Message: "system_action step requires explicit system_action metadata",
		}}
	}

	spec := step.SystemAction
	errors := make([]ValidationError, 0, 5)

	if !isValidSystemActionKind(spec.Kind) {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeInvalidStepType,
			StepID:  step.ID,
			Message: "system_action step requires kind service or process",
		})
	}
	if !isValidSystemActionOperation(spec.Operation) {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeInvalidStepType,
			StepID:  step.ID,
			Message: "system_action step requires operation start, stop, inspect, or status",
		})
	}
	if strings.TrimSpace(spec.Target) == "" {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeInvalidStepType,
			StepID:  step.ID,
			Message: "system_action step requires explicit target metadata",
		})
	}
	if _, err := resolveSystemActionCommand(job, step); err != nil {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeInvalidStepType,
			StepID:  step.ID,
			Message: err.Error(),
		})
	}
	if err := validateSystemActionPostState(step); err != nil {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeInvalidStepType,
			StepID:  step.ID,
			Message: err.Error(),
		})
	}

	return errors
}

func validateSystemActionCompletion(job Job, step Step, tools []RuntimeToolCallEvidence) (*stepValidationResult, error) {
	if step.SystemAction == nil {
		return nil, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: "system_action completion requires explicit system_action metadata",
		}
	}

	command, err := resolveSystemActionCommand(job, step)
	if err != nil {
		return nil, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: err.Error(),
		}
	}
	if _, ok := findExecEvidenceForCommand(tools, command); !ok {
		return nil, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: fmt.Sprintf("system_action completion requires executing %q", commandLabel(command)),
		}
	}

	output, err := verifySystemActionPostState(step, tools)
	if err != nil {
		return nil, err
	}

	spec := step.SystemAction
	return &stepValidationResult{
		recordCompletion: true,
		resultingState: &RuntimeResultingStateRecord{
			Kind:                string(StepTypeSystemAction),
			Target:              strings.TrimSpace(spec.Target),
			Operation:           string(spec.Operation),
			State:               systemActionVerifiedState(spec.Operation),
			ActionCommand:       append([]string(nil), command...),
			VerificationCommand: append([]string(nil), spec.PostState.Command...),
			VerificationOutput:  output,
			SourceStepID:        strings.TrimSpace(spec.SourceStepID),
		},
		rollback: buildSystemActionRollbackRecord(*spec),
	}, nil
}

func verifySystemActionPostState(step Step, tools []RuntimeToolCallEvidence) (string, error) {
	if err := validateSystemActionPostState(step); err != nil {
		return "", ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: err.Error(),
		}
	}

	postState := step.SystemAction.PostState
	verification, ok := findExecEvidenceForCommand(tools, postState.Command)
	if !ok {
		return "", ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  step.ID,
			Message: fmt.Sprintf("system_action completion requires verification command %q", commandLabel(postState.Command)),
		}
	}

	output := verification.Result
	for _, needle := range postState.SuccessContains {
		trimmed := strings.TrimSpace(needle)
		if trimmed == "" {
			continue
		}
		if !strings.Contains(output, trimmed) {
			return "", ValidationError{
				Code:    RejectionCodeStepValidationFailed,
				StepID:  step.ID,
				Message: fmt.Sprintf("system_action post-state verification for %q is missing %q", step.SystemAction.Target, trimmed),
			}
		}
	}
	for _, needle := range postState.FailureContains {
		trimmed := strings.TrimSpace(needle)
		if trimmed == "" {
			continue
		}
		if strings.Contains(output, trimmed) {
			return "", ValidationError{
				Code:    RejectionCodeStepValidationFailed,
				StepID:  step.ID,
				Message: fmt.Sprintf("system_action post-state verification for %q still reports %q", step.SystemAction.Target, trimmed),
			}
		}
	}

	return output, nil
}

func validateSystemActionPostState(step Step) error {
	if step.SystemAction == nil || step.SystemAction.PostState == nil {
		return fmt.Errorf("system_action step requires explicit post_state verification contract")
	}
	postState := step.SystemAction.PostState
	command := normalizeCommandArgs(postState.Command)
	if len(command) == 0 {
		return fmt.Errorf("system_action step requires explicit post_state verification contract")
	}
	if len(trimmedNonEmptyStrings(postState.SuccessContains)) == 0 {
		return fmt.Errorf("system_action step requires explicit post_state verification contract")
	}
	return nil
}

func resolveSystemActionCommand(job Job, step Step) ([]string, error) {
	if step.SystemAction == nil {
		return nil, fmt.Errorf("system_action step requires explicit system_action metadata")
	}

	spec := step.SystemAction
	explicit := normalizeCommandArgs(spec.Command)
	sourceStepID := strings.TrimSpace(spec.SourceStepID)

	if len(explicit) > 0 && sourceStepID != "" {
		return nil, fmt.Errorf("system_action step must not set both command and source_step_id")
	}
	if len(explicit) > 0 {
		return explicit, nil
	}
	if sourceStepID == "" {
		return nil, fmt.Errorf("system_action step requires explicit command metadata")
	}
	if spec.Operation != SystemActionOperationStart {
		return nil, fmt.Errorf("system_action source_step_id is only supported for start operations")
	}
	if !containsStepDependency(step.DependsOn, sourceStepID) {
		return nil, fmt.Errorf("system_action step source_step_id must appear in depends_on")
	}

	sourceStep, ok := findPlanStep(job.Plan, sourceStepID)
	if !ok || sourceStep.Type != StepTypeLongRunningCode {
		return nil, fmt.Errorf("system_action step source_step_id must reference a long_running_code step")
	}
	command := longRunningStartupCommand(sourceStep)
	if len(command) == 0 {
		return nil, fmt.Errorf("system_action step source_step_id must reference a long_running_code step with explicit long_running_startup_command metadata")
	}
	if longRunningArtifactPath(sourceStep) == "" {
		return nil, fmt.Errorf("system_action step source_step_id must reference a long_running_code step with explicit long_running_artifact_path metadata")
	}
	return command, nil
}

func isValidSystemActionKind(kind SystemActionKind) bool {
	switch kind {
	case SystemActionKindService, SystemActionKindProcess:
		return true
	default:
		return false
	}
}

func isValidSystemActionOperation(operation SystemActionOperation) bool {
	switch operation {
	case SystemActionOperationStart, SystemActionOperationStop, SystemActionOperationInspect, SystemActionOperationStatus:
		return true
	default:
		return false
	}
}

func normalizeCommandArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" {
			return nil
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func trimmedNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}

func containsStepDependency(dependsOn []string, stepID string) bool {
	for _, dependencyID := range dependsOn {
		if dependencyID == stepID {
			return true
		}
	}
	return false
}

func findPlanStep(plan Plan, stepID string) (Step, bool) {
	for _, candidate := range plan.Steps {
		if candidate.ID == stepID {
			return candidate, true
		}
	}
	return Step{}, false
}

func findExecEvidenceForCommand(tools []RuntimeToolCallEvidence, command []string) (RuntimeToolCallEvidence, bool) {
	normalized := normalizeCommandArgs(command)
	if len(normalized) == 0 {
		return RuntimeToolCallEvidence{}, false
	}
	for _, tool := range tools {
		if tool.ToolName != "exec" {
			continue
		}
		if commandsEqual(normalized, toolArgStringSlice(tool.Arguments, "cmd")) {
			return tool, true
		}
	}
	return RuntimeToolCallEvidence{}, false
}

func commandsEqual(left []string, right []string) bool {
	left = normalizeCommandArgs(left)
	right = normalizeCommandArgs(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func commandLabel(command []string) string {
	return strings.Join(normalizeCommandArgs(command), " ")
}

func systemActionVerifiedState(operation SystemActionOperation) string {
	switch operation {
	case SystemActionOperationStart:
		return "running"
	case SystemActionOperationStop:
		return "stopped"
	case SystemActionOperationInspect:
		return "inspected"
	case SystemActionOperationStatus:
		return "status_observed"
	default:
		return "verified"
	}
}

func buildSystemActionRollbackRecord(spec SystemAction) *RuntimeRollbackRecord {
	record := &RuntimeRollbackRecord{}
	switch {
	case spec.Rollback != nil && len(normalizeCommandArgs(spec.Rollback.Command)) > 0:
		record.Available = true
		record.Command = append([]string(nil), normalizeCommandArgs(spec.Rollback.Command)...)
		record.Reason = strings.TrimSpace(spec.Rollback.Reason)
	case spec.Operation == SystemActionOperationInspect || spec.Operation == SystemActionOperationStatus:
		record.Reason = "read_only_action"
	default:
		record.Reason = "rollback_not_declared"
	}
	return record
}

func systemActionAuditAction(job *Job, step *Step, args map[string]interface{}) string {
	if step == nil || step.SystemAction == nil {
		return "system_action"
	}

	spec := *step.SystemAction
	command := toolArgStringSlice(args, "cmd")
	switch {
	case job != nil:
		if actionCommand, err := resolveSystemActionCommand(*job, *step); err == nil && commandsEqual(command, actionCommand) {
			return systemActionAuditLabel(spec, "execute")
		}
	case len(command) == 0:
		return systemActionAuditLabel(spec, "attempt")
	}

	if spec.PostState != nil && commandsEqual(command, spec.PostState.Command) {
		return systemActionAuditLabel(spec, "verify_post_state")
	}

	return systemActionAuditLabel(spec, "attempt")
}

func systemActionAuditLabel(spec SystemAction, phase string) string {
	return fmt.Sprintf(
		"system_action:%s:%s:%s:%s",
		strings.TrimSpace(phase),
		strings.TrimSpace(string(spec.Operation)),
		strings.TrimSpace(string(spec.Kind)),
		strings.TrimSpace(spec.Target),
	)
}
