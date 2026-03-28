package missioncontrol

import "strings"

const finalResponseSummaryMarker = "\n\nMission summary:\n"
const finalResponseBodyOmissionMarker = "\n\n[final response body truncated; "
const finalResponseBodyRuneLimit = 1200
const FinalResponseSummaryArtifactLimit = OperatorStatusArtifactLimit
const FinalResponseSummaryPendingLimit = 5

func NormalizeFinalResponse(ec ExecutionContext, response string) string {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" || ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
		return trimmed
	}
	if ec.Step.Type != StepTypeFinalResponse {
		return trimmed
	}
	if strings.Contains(trimmed, finalResponseSummaryMarker) {
		return trimmed
	}

	trimmed = truncateFinalResponseBody(trimmed)
	lines := finalResponseSummaryLines(ec)
	if len(lines) == 0 {
		return trimmed
	}
	return trimmed + finalResponseSummaryMarker + strings.Join(lines, "\n")
}

func truncateFinalResponseBody(body string) string {
	runes := []rune(body)
	if len(runes) <= finalResponseBodyRuneLimit {
		return body
	}

	omitted := len(runes) - finalResponseBodyRuneLimit
	truncated := strings.TrimRight(string(runes[:finalResponseBodyRuneLimit]), " \t\r\n")
	if truncated == "" {
		truncated = string(runes[:finalResponseBodyRuneLimit])
	}
	return truncated + finalResponseBodyOmissionMarker + itoa(omitted) + " characters omitted]"
}

func finalResponseSummaryLines(ec ExecutionContext) []string {
	lines := make([]string, 0, 2)
	if artifacts := finalResponseArtifactsLine(ec); artifacts != "" {
		lines = append(lines, artifacts)
	}
	if pending := finalResponsePendingStepsLine(ec); pending != "" {
		lines = append(lines, pending)
	}
	return lines
}

func finalResponseArtifactsLine(ec ExecutionContext) string {
	runtime := *CloneJobRuntimeState(ec.Runtime)
	if runtime.InspectablePlan == nil && ec.Job != nil {
		if plan, err := BuildInspectablePlanContext(*ec.Job); err == nil {
			runtime.InspectablePlan = &plan
		}
	}

	artifacts := selectOperatorStatusArtifacts(runtime)
	if len(artifacts) == 0 {
		return ""
	}

	parts := make([]string, 0, len(artifacts)+1)
	for _, artifact := range artifacts {
		part := artifact.Path
		if artifact.State == "already_present" {
			part += " (already present)"
		}
		parts = append(parts, part)
	}

	total := len(collectOperatorStatusArtifactCandidates(runtime))
	if omitted := total - len(artifacts); omitted > 0 {
		parts = append(parts, "+"+itoa(omitted)+" more omitted")
	}
	return "Artifacts: " + strings.Join(parts, "; ")
}

func finalResponsePendingStepsLine(ec ExecutionContext) string {
	if ec.Job == nil {
		return ""
	}

	completed := make(map[string]struct{}, len(ec.Runtime.CompletedSteps))
	for _, record := range ec.Runtime.CompletedSteps {
		completed[record.StepID] = struct{}{}
	}
	failed := make(map[string]struct{}, len(ec.Runtime.FailedSteps))
	for _, record := range ec.Runtime.FailedSteps {
		failed[record.StepID] = struct{}{}
	}

	pending := make([]string, 0, len(ec.Job.Plan.Steps))
	for _, step := range ec.Job.Plan.Steps {
		if step.ID == ec.Step.ID {
			continue
		}
		if _, ok := completed[step.ID]; ok {
			continue
		}
		label := step.ID
		if _, ok := failed[step.ID]; ok {
			label += " (blocked)"
		}
		pending = append(pending, label)
	}
	if len(pending) == 0 {
		return ""
	}

	if len(pending) > FinalResponseSummaryPendingLimit {
		visible := append([]string(nil), pending[:FinalResponseSummaryPendingLimit]...)
		visible = append(visible, "+"+itoa(len(pending)-FinalResponseSummaryPendingLimit)+" more omitted")
		pending = visible
	}
	return "Pending/blocked steps: " + strings.Join(pending, "; ")
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := [20]byte{}
	i := len(digits)
	for v > 0 {
		i--
		digits[i] = byte('0' + (v % 10))
		v /= 10
	}
	return string(digits[i:])
}
