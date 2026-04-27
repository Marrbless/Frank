package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/providers"
)

type hotUpdateHelpProvider struct {
	calls int
}

func (p *hotUpdateHelpProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	p.calls++
	return providers.LLMResponse{Content: "provider fallback"}, nil
}

func (p *hotUpdateHelpProvider) GetDefaultModel() string {
	return "test"
}

func TestProcessDirectHotUpdateHelpListsV4LifecycleCommands(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &hotUpdateHelpProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)

	resp, err := ag.ProcessDirect("HELP HOT_UPDATE", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HELP HOT_UPDATE) error = %v", err)
	}
	if prov.calls != 0 {
		t.Fatalf("provider calls = %d, want 0 for static help", prov.calls)
	}

	for _, want := range []string{
		"docs/HOT_UPDATE_OPERATOR_RUNBOOK.md",
		"STATUS <job_id>",
		"HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>",
		"HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]",
		"HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>",
		"HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]",
		"HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]",
		"HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>",
		"HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>",
		"ROLLBACK_APPLY_RELOAD <job_id> <apply_id>",
		"HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>",
		"CandidatePromotionDecisionRecord remains eligible-only",
		"natural-language yes/no/approve/deny aliases are not canary owner approval authority",
		"canary_ref and approval_ref audit lineage",
	} {
		if !strings.Contains(resp, want) {
			t.Fatalf("ProcessDirect(HELP HOT_UPDATE) response missing %q\nresponse:\n%s", want, resp)
		}
	}

	aliasResp, err := ag.ProcessDirect("HOT_UPDATE_HELP", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_HELP) error = %v", err)
	}
	if aliasResp != resp {
		t.Fatalf("HOT_UPDATE_HELP response differs from HELP HOT_UPDATE\nHELP HOT_UPDATE:\n%s\nHOT_UPDATE_HELP:\n%s", resp, aliasResp)
	}
	if prov.calls != 0 {
		t.Fatalf("provider calls after alias = %d, want 0 for static help", prov.calls)
	}

	v4Resp, err := ag.ProcessDirect("HELP V4", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HELP V4) error = %v", err)
	}
	if v4Resp != resp {
		t.Fatalf("HELP V4 response differs from HELP HOT_UPDATE\nHELP HOT_UPDATE:\n%s\nHELP V4:\n%s", resp, v4Resp)
	}
	if prov.calls != 0 {
		t.Fatalf("provider calls after HELP V4 = %d, want 0 for static help", prov.calls)
	}
}
