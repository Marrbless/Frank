package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
)

type v4DirectCommandSurface struct {
	name   string
	sample string
	re     *regexp.Regexp
}

func TestProcessDirectHotUpdateHelpRunbookAndParserStayConsistent(t *testing.T) {
	t.Parallel()

	commands := canonicalV4DirectCommandSurfaces()
	runbook := readHotUpdateOperatorRunbook(t)
	help := processHotUpdateHelpWithNoProviderOrStoreMutation(t)

	for _, command := range commands {
		if !command.re.MatchString(command.sample) {
			t.Fatalf("parser regex for %s did not match safe sample %q", command.name, command.sample)
		}
		if !strings.Contains(help, command.name) {
			t.Fatalf("HOT_UPDATE_HELP output missing command %s\nhelp:\n%s", command.name, help)
		}
		if !strings.Contains(runbook, command.name) {
			t.Fatalf("docs/HOT_UPDATE_OPERATOR_RUNBOOK.md missing command %s", command.name)
		}
	}

	assertHelpAndRunbookInvariant(t, help, runbook, "eligible-only promotion decision contract", "CandidatePromotionDecisionRecord remains eligible-only", "`CandidatePromotionDecisionRecord` remains strictly")
	assertHelpAndRunbookInvariant(t, help, runbook, "exact canary owner approval decisions", "Canary owner approval uses exact granted or rejected decisions", "owner approval decision=granted", "owner approval decision=rejected")
	assertHelpAndRunbookInvariant(t, help, runbook, "natural-language owner approval separation", "natural-language yes/no/approve/deny aliases are not canary owner approval authority", "Natural-language aliases", "intentionally not bound")
	assertHelpAndRunbookInvariant(t, help, runbook, "canary execution guards", "Canary-derived gates are guarded before phase advancement", "Canary-derived gates are guarded by readiness checks")
	assertHelpAndRunbookInvariant(t, help, runbook, "outcome promotion audit lineage", "Outcomes and promotions preserve canary_ref and approval_ref audit lineage", "Outcome and promotion records preserve audit lineage by copying `canary_ref` and `approval_ref`")
	assertHelpAndRunbookInvariant(t, help, runbook, "generic rollback LKG recovery", "Rollback, rollback-apply, and LKG recertification remain generic recovery flows", "Rollback and rollback-apply remain generic recovery flows", "LKG recertification remains generic")
}

func canonicalV4DirectCommandSurfaces() []v4DirectCommandSurface {
	return []v4DirectCommandSurface{
		{name: "STATUS", sample: "STATUS job-1", re: runtimeCommandRE},
		{name: "HOT_UPDATE_HELP", sample: "HOT_UPDATE_HELP", re: hotUpdateHelpCommandRE},
		{name: "HELP HOT_UPDATE", sample: "HELP HOT_UPDATE", re: hotUpdateHelpCommandRE},
		{name: "HELP V4", sample: "HELP V4", re: hotUpdateHelpCommandRE},
		{name: "HOT_UPDATE_GATE_RECORD", sample: "HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", re: hotUpdateGateRecordCommandRE},
		{name: "HOT_UPDATE_GATE_FROM_DECISION", sample: "HOT_UPDATE_GATE_FROM_DECISION job-1 decision-1", re: hotUpdateGateFromDecisionCommandRE},
		{name: "HOT_UPDATE_EXECUTION_READY", sample: "HOT_UPDATE_EXECUTION_READY job-1 hot-update-1 60 ready", re: hotUpdateExecutionReadyCommandRE},
		{name: "HOT_UPDATE_GATE_PHASE", sample: "HOT_UPDATE_GATE_PHASE job-1 hot-update-1 validated", re: hotUpdateGatePhaseCommandRE},
		{name: "HOT_UPDATE_GATE_EXECUTE", sample: "HOT_UPDATE_GATE_EXECUTE job-1 hot-update-1", re: hotUpdateGateExecuteCommandRE},
		{name: "HOT_UPDATE_GATE_RELOAD", sample: "HOT_UPDATE_GATE_RELOAD job-1 hot-update-1", re: hotUpdateGateReloadCommandRE},
		{name: "HOT_UPDATE_GATE_FAIL", sample: "HOT_UPDATE_GATE_FAIL job-1 hot-update-1 operator stopped", re: hotUpdateGateFailCommandRE},
		{name: "HOT_UPDATE_OUTCOME_CREATE", sample: "HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1", re: hotUpdateOutcomeCreateCommandRE},
		{name: "HOT_UPDATE_PROMOTION_CREATE", sample: "HOT_UPDATE_PROMOTION_CREATE job-1 outcome-1", re: hotUpdatePromotionCreateCommandRE},
		{name: "HOT_UPDATE_LKG_RECERTIFY", sample: "HOT_UPDATE_LKG_RECERTIFY job-1 promotion-1", re: hotUpdateLKGRecertifyCommandRE},
		{name: "HOT_UPDATE_CANARY_REQUIREMENT_CREATE", sample: "HOT_UPDATE_CANARY_REQUIREMENT_CREATE job-1 result-1", re: hotUpdateCanaryRequirementCreateCommandRE},
		{name: "HOT_UPDATE_CANARY_EVIDENCE_CREATE", sample: "HOT_UPDATE_CANARY_EVIDENCE_CREATE job-1 requirement-1 passed 2026-04-26T04:30:00Z canary passed", re: hotUpdateCanaryEvidenceCreateCommandRE},
		{name: "HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE", sample: "HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE job-1 requirement-1", re: hotUpdateCanarySatisfactionAuthorityCreateCommandRE},
		{name: "HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE", sample: "HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE job-1 authority-1", re: hotUpdateOwnerApprovalRequestCreateCommandRE},
		{name: "HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE", sample: "HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE job-1 request-1 granted owner approved", re: hotUpdateOwnerApprovalDecisionCreateCommandRE},
		{name: "HOT_UPDATE_CANARY_GATE_CREATE", sample: "HOT_UPDATE_CANARY_GATE_CREATE job-1 authority-1 decision-1", re: hotUpdateCanaryGateCreateCommandRE},
		{name: "ROLLBACK_RECORD", sample: "ROLLBACK_RECORD job-1 promotion-1 rollback-1", re: rollbackRecordCommandRE},
		{name: "ROLLBACK_APPLY_RECORD", sample: "ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", re: rollbackApplyRecordCommandRE},
		{name: "ROLLBACK_APPLY_PHASE", sample: "ROLLBACK_APPLY_PHASE job-1 apply-1 validated", re: rollbackApplyPhaseCommandRE},
		{name: "ROLLBACK_APPLY_EXECUTE", sample: "ROLLBACK_APPLY_EXECUTE job-1 apply-1", re: rollbackApplyExecuteCommandRE},
		{name: "ROLLBACK_APPLY_RELOAD", sample: "ROLLBACK_APPLY_RELOAD job-1 apply-1", re: rollbackApplyReloadCommandRE},
		{name: "ROLLBACK_APPLY_FAIL", sample: "ROLLBACK_APPLY_FAIL job-1 apply-1 operator stopped", re: rollbackApplyFailCommandRE},
	}
}

func processHotUpdateHelpWithNoProviderOrStoreMutation(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	before := snapshotRelativeFiles(t, root)
	b := chat.NewHub(10)
	prov := &hotUpdateHelpProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)

	help, err := ag.ProcessDirect("HOT_UPDATE_HELP", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_HELP) error = %v", err)
	}
	for _, alias := range []string{"HELP HOT_UPDATE", "HELP V4"} {
		got, err := ag.ProcessDirect(alias, 2*time.Second)
		if err != nil {
			t.Fatalf("ProcessDirect(%s) error = %v", alias, err)
		}
		if got != help {
			t.Fatalf("ProcessDirect(%s) help differs from HOT_UPDATE_HELP\nHOT_UPDATE_HELP:\n%s\n%s:\n%s", alias, help, alias, got)
		}
	}
	if prov.calls != 0 {
		t.Fatalf("provider calls = %d, want 0 for static help aliases", prov.calls)
	}
	after := snapshotRelativeFiles(t, root)
	if strings.Join(after, "\n") != strings.Join(before, "\n") {
		t.Fatalf("mission store files changed after help aliases\nbefore=%v\nafter=%v", before, after)
	}
	return help
}

func readHotUpdateOperatorRunbook(t *testing.T) string {
	t.Helper()

	path := filepath.Join("..", "..", "docs", "HOT_UPDATE_OPERATOR_RUNBOOK.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(content)
}

func snapshotRelativeFiles(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root || entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		t.Fatalf("WalkDir(%s) error = %v", root, err)
	}
	return files
}

func assertHelpAndRunbookInvariant(t *testing.T, help, runbook, name, helpSnippet string, runbookSnippets ...string) {
	t.Helper()

	if !strings.Contains(help, helpSnippet) {
		t.Fatalf("HOT_UPDATE_HELP missing %s invariant snippet %q\nhelp:\n%s", name, helpSnippet, help)
	}
	for _, snippet := range runbookSnippets {
		if !strings.Contains(runbook, snippet) {
			t.Fatalf("runbook missing %s invariant snippet %q", name, snippet)
		}
	}
}
