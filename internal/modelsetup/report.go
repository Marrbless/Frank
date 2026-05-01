package modelsetup

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func PrintPlan(w io.Writer, plan Plan) {
	fmt.Fprintf(w, "preset: %s\n", plan.PresetName)
	fmt.Fprintf(w, "status: %s\n", plan.Status)
	fmt.Fprintf(w, "approved: %t\n", plan.Approved)
	fmt.Fprintf(w, "ready: %t\n", plan.Ready)
	if plan.ReadinessStatus != "" {
		fmt.Fprintf(w, "readiness_status: %s\n", plan.ReadinessStatus)
	}
	fmt.Fprintf(w, "platform: %s\n", emptyAsUnknown(plan.Environment.Platform))
	fmt.Fprintf(w, "os: %s\n", emptyAsUnknown(plan.Environment.OS))
	fmt.Fprintf(w, "arch: %s\n", emptyAsUnknown(plan.Environment.Arch))
	fmt.Fprintf(w, "config_path: %s\n", emptyAsUnknown(plan.Environment.ConfigPath))
	fmt.Fprintf(w, "termux: %s\n", emptyStateAsUnknown(plan.Environment.Termux))
	fmt.Fprintf(w, "termux_boot: %s\n", emptyStateAsUnknown(plan.Environment.TermuxBoot))
	fmt.Fprintf(w, "tmux: %s\n", emptyStateAsUnknown(plan.Environment.Tmux))
	fmt.Fprintf(w, "ollama: %s\n", emptyStateAsUnknown(plan.Environment.Ollama))
	fmt.Fprintf(w, "llamacpp: %s\n", emptyStateAsUnknown(plan.Environment.LlamaCPP))
	fmt.Fprintf(w, "runtime_kind: %s\n", plan.RuntimeKind)
	fmt.Fprintf(w, "provider_ref: %s\n", plan.ProviderRef)
	fmt.Fprintf(w, "model_ref: %s\n", plan.ModelRef)
	fmt.Fprintf(w, "provider_model: %s\n", plan.ProviderModel)
	if plan.BindAddress != "" {
		fmt.Fprintf(w, "bind_address: %s\n", plan.BindAddress)
	}
	if plan.Port > 0 {
		fmt.Fprintf(w, "port: %d\n", plan.Port)
	}
	fmt.Fprintf(w, "supports_tools: %t\n", plan.ToolSupport)
	fmt.Fprintf(w, "authority_tier: %s\n", plan.AuthorityTier)
	fmt.Fprintf(w, "cloud_fallback: %t\n", plan.CloudFallback)
	printStringSection(w, "assumptions", plan.Assumptions)
	printStringSection(w, "warnings", plan.Warnings)
	printStringSection(w, "blocked_reasons", plan.BlockedReasons)
	printStringSection(w, "manual_instructions", plan.ManualInstructions)
	fmt.Fprintln(w, "steps:")
	for _, step := range plan.Steps {
		fmt.Fprintf(w, "- id: %s\n", step.ID)
		fmt.Fprintf(w, "  status: %s\n", step.Status)
		fmt.Fprintf(w, "  summary: %s\n", step.Summary)
		fmt.Fprintf(w, "  side_effect: %s\n", step.SideEffect)
		if len(step.Command) > 0 {
			fmt.Fprintf(w, "  command: %s\n", shellJoin(step.Command))
		}
		if len(step.FilesToRead) > 0 {
			fmt.Fprintf(w, "  files_to_read: %s\n", strings.Join(step.FilesToRead, ", "))
		}
		if len(step.FilesToWrite) > 0 {
			fmt.Fprintf(w, "  files_to_write: %s\n", strings.Join(step.FilesToWrite, ", "))
		}
		if step.NetworkURL != "" {
			fmt.Fprintf(w, "  network_url: %s\n", step.NetworkURL)
		}
		if step.ChecksumSHA256 != "" {
			fmt.Fprintf(w, "  checksum_sha256: %s\n", step.ChecksumSHA256)
		}
		if step.ExpectedDownloadSize != "" {
			fmt.Fprintf(w, "  expected_download_size: %s\n", step.ExpectedDownloadSize)
		}
		if step.ExpectedDiskImpact != "" {
			fmt.Fprintf(w, "  expected_disk_impact: %s\n", step.ExpectedDiskImpact)
		}
		if step.RuntimeBindAddress != "" {
			fmt.Fprintf(w, "  runtime_bind_address: %s\n", step.RuntimeBindAddress)
		}
		if step.RuntimePort > 0 {
			fmt.Fprintf(w, "  runtime_port: %d\n", step.RuntimePort)
		}
		fmt.Fprintf(w, "  approval_required: %t\n", step.ApprovalRequired)
		if step.ApprovalReason != "" {
			fmt.Fprintf(w, "  approval_reason: %s\n", step.ApprovalReason)
		}
		fmt.Fprintf(w, "  idempotency_key: %s\n", step.IdempotencyKey)
		if step.AlreadyPresentRule != "" {
			fmt.Fprintf(w, "  already_present_rule: %s\n", step.AlreadyPresentRule)
		}
		if step.RollbackCleanup != "" {
			fmt.Fprintf(w, "  rollback_cleanup: %s\n", step.RollbackCleanup)
		}
		if len(step.Dependencies) > 0 {
			fmt.Fprintf(w, "  dependencies: %s\n", strings.Join(step.Dependencies, ", "))
		}
		if len(step.ManualInstructions) > 0 {
			fmt.Fprintf(w, "  manual_instructions: %s\n", strings.Join(step.ManualInstructions, " "))
		}
	}
	fmt.Fprintf(w, "redaction_policy: %s\n", plan.RedactionPolicy)
	fmt.Fprintf(w, "truncation_policy: %s\n", plan.TruncationPolicy)
}

func TruncateReportSteps(steps []PlanStep, maxSteps int) []PlanStep {
	if maxSteps <= 0 || len(steps) <= maxSteps {
		return append([]PlanStep(nil), steps...)
	}
	preserved := make([]PlanStep, 0, len(steps))
	var candidates []PlanStep
	for _, step := range steps {
		if step.Status == PlanStatusFailed || step.Status == PlanStatusBlocked || step.Status == PlanStatusRolledBack || step.Status == PlanStatusManualRequired || step.PreserveWhenTruncating {
			preserved = append(preserved, step)
			continue
		}
		candidates = append(candidates, step)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].DiagnosticsPriority != candidates[j].DiagnosticsPriority {
			return candidates[i].DiagnosticsPriority > candidates[j].DiagnosticsPriority
		}
		return candidates[i].ID < candidates[j].ID
	})
	out := append([]PlanStep(nil), preserved...)
	for _, step := range candidates {
		if len(out) >= maxSteps {
			break
		}
		out = append(out, step)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func printStringSection(w io.Writer, label string, values []string) {
	if len(values) == 0 {
		return
	}
	values = sortedStrings(values)
	fmt.Fprintf(w, "%s:\n", label)
	for _, value := range values {
		fmt.Fprintf(w, "- %s\n", value)
	}
}

func emptyAsUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return strings.TrimSpace(value)
}

func emptyStateAsUnknown(value State) string {
	if value == "" {
		return string(StateUnknown)
	}
	return string(value)
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\n\"'") {
			quoted = append(quoted, fmt.Sprintf("%q", arg))
			continue
		}
		quoted = append(quoted, arg)
	}
	return strings.Join(quoted, " ")
}
