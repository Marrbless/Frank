package agent

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/missioncontrol"
)

func TestProcessDirectRollbackApplyRecordCommandCreatesOrSelectsWorkflowAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	ecBefore, ok := ag.ActiveMissionStep()
	if !ok || ecBefore.Runtime == nil {
		t.Fatalf("ActiveMissionStep() before = %#v, want active runtime", ecBefore)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD first) error = %v", err)
	}
	if resp != "Recorded rollback-apply workflow job=job-1 rollback=rollback-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD first) response = %q, want create acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.RollbackID != "rollback-1" {
		t.Fatalf("RollbackApplyRecord.RollbackID = %q, want rollback-1", record.RollbackID)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseRecorded {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want recorded", record.Phase)
	}
	if record.ActivationState != missioncontrol.RollbackApplyActivationStateUnchanged {
		t.Fatalf("RollbackApplyRecord.ActivationState = %q, want unchanged", record.ActivationState)
	}
	if record.CreatedBy != "operator" {
		t.Fatalf("RollbackApplyRecord.CreatedBy = %q, want operator", record.CreatedBy)
	}
	if !record.RequestedAt.Equal(record.CreatedAt) {
		t.Fatalf("RollbackApplyRecord timestamps = (%v, %v), want equal requested_at and created_at", record.RequestedAt, record.CreatedAt)
	}

	firstBytes, err := os.ReadFile(missioncontrol.StoreRollbackApplyPath(root, "apply-1"))
	if err != nil {
		t.Fatalf("ReadFile(first apply) error = %v", err)
	}

	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD second) error = %v", err)
	}
	if resp != "Selected rollback-apply workflow job=job-1 rollback=rollback-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD second) response = %q, want select acknowledgement", resp)
	}

	secondBytes, err := os.ReadFile(missioncontrol.StoreRollbackApplyPath(root, "apply-1"))
	if err != nil {
		t.Fatalf("ReadFile(second apply) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("rollback-apply file changed on select path\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}

	ecAfter, ok := ag.ActiveMissionStep()
	if !ok || ecAfter.Runtime == nil {
		t.Fatalf("ActiveMissionStep() after = %#v, want active runtime", ecAfter)
	}
	if ecAfter.Runtime.State != ecBefore.Runtime.State {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ecAfter.Runtime.State, ecBefore.Runtime.State)
	}
	if ecAfter.Runtime.ActiveStepID != ecBefore.Runtime.ActiveStepID {
		t.Fatalf("ActiveMissionStep().Runtime.ActiveStepID = %q, want %q", ecAfter.Runtime.ActiveStepID, ecBefore.Runtime.ActiveStepID)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackIdentity == nil {
		t.Fatal("RollbackIdentity = nil, want rollback identity block")
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RollbackIdentity.State != "configured" {
		t.Fatalf("RollbackIdentity.State = %q, want configured", summary.RollbackIdentity.State)
	}
	if summary.RollbackApplyIdentity.State != "configured" {
		t.Fatalf("RollbackApplyIdentity.State = %q, want configured", summary.RollbackApplyIdentity.State)
	}
	if len(summary.RollbackIdentity.Rollbacks) != 1 {
		t.Fatalf("RollbackIdentity.Rollbacks len = %d, want 1", len(summary.RollbackIdentity.Rollbacks))
	}
	if len(summary.RollbackApplyIdentity.Applies) != 1 {
		t.Fatalf("RollbackApplyIdentity.Applies len = %d, want 1", len(summary.RollbackApplyIdentity.Applies))
	}
	if summary.RollbackIdentity.Rollbacks[0].RollbackID != "rollback-1" {
		t.Fatalf("RollbackIdentity.Rollbacks[0].RollbackID = %q, want rollback-1", summary.RollbackIdentity.Rollbacks[0].RollbackID)
	}
	if summary.RollbackApplyIdentity.Applies[0].RollbackApplyID != "apply-1" {
		t.Fatalf("RollbackApplyIdentity.Applies[0].RollbackApplyID = %q, want apply-1", summary.RollbackApplyIdentity.Applies[0].RollbackApplyID)
	}
	if summary.RollbackApplyIdentity.Applies[0].RollbackID != "rollback-1" {
		t.Fatalf("RollbackApplyIdentity.Applies[0].RollbackID = %q, want rollback-1", summary.RollbackApplyIdentity.Applies[0].RollbackID)
	}
	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want compact recovery summary")
	}
	if summary.V4Summary.State != "rollback_apply_recorded" {
		t.Fatalf("V4Summary.State = %q, want rollback_apply_recorded", summary.V4Summary.State)
	}
	if summary.V4Summary.SelectedRollbackID != "rollback-1" {
		t.Fatalf("V4Summary.SelectedRollbackID = %q, want rollback-1", summary.V4Summary.SelectedRollbackID)
	}
	if summary.V4Summary.SelectedRollbackApplyID != "apply-1" {
		t.Fatalf("V4Summary.SelectedRollbackApplyID = %q, want apply-1", summary.V4Summary.SelectedRollbackApplyID)
	}
	if !summary.V4Summary.HasRollback || !summary.V4Summary.HasRollbackApply {
		t.Fatalf("V4Summary recovery booleans = rollback %t apply %t, want both true", summary.V4Summary.HasRollback, summary.V4Summary.HasRollbackApply)
	}
}

func TestProcessDirectRollbackApplyRecordCommandFailsClosedWhenRollbackIsMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 missing-rollback apply-missing", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_RECORD) error = nil, want missing rollback rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrRollbackRecordNotFound.Error()) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %q, want missing rollback rejection", err)
	}
	if _, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-missing"); err != missioncontrol.ErrRollbackApplyRecordNotFound {
		t.Fatalf("LoadRollbackApplyRecord() error = %v, want %v", err, missioncontrol.ErrRollbackApplyRecordNotFound)
	}
}

func TestProcessDirectRollbackApplyPhaseCommandAdvancesWorkflowAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	beforeBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if resp != "Advanced rollback-apply workflow job=job-1 apply=apply-1 phase=validated." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) response = %q, want validated acknowledgement", resp)
	}

	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}
	if resp != "Advanced rollback-apply workflow job=job-1 apply=apply-1 phase=ready_to_apply." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) response = %q, want ready acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseReadyToApply {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want ready_to_apply", record.Phase)
	}
	if record.ActivationState != missioncontrol.RollbackApplyActivationStateUnchanged {
		t.Fatalf("RollbackApplyRecord.ActivationState = %q, want unchanged", record.ActivationState)
	}
	if record.PhaseUpdatedBy != "operator" {
		t.Fatalf("RollbackApplyRecord.PhaseUpdatedBy = %q, want operator", record.PhaseUpdatedBy)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforeBytes), string(afterBytes))
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RollbackApplyIdentity.State != "configured" {
		t.Fatalf("RollbackApplyIdentity.State = %q, want configured", summary.RollbackApplyIdentity.State)
	}
	if len(summary.RollbackApplyIdentity.Applies) != 1 {
		t.Fatalf("RollbackApplyIdentity.Applies len = %d, want 1", len(summary.RollbackApplyIdentity.Applies))
	}
	apply := summary.RollbackApplyIdentity.Applies[0]
	if apply.RollbackApplyID != "apply-1" {
		t.Fatalf("RollbackApplyIdentity.Applies[0].RollbackApplyID = %q, want apply-1", apply.RollbackApplyID)
	}
	if apply.Phase != string(missioncontrol.RollbackApplyPhaseReadyToApply) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].Phase = %q, want ready_to_apply", apply.Phase)
	}
	if apply.ActivationState != string(missioncontrol.RollbackApplyActivationStateUnchanged) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].ActivationState = %q, want unchanged", apply.ActivationState)
	}
	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want compact recovery summary")
	}
	if summary.V4Summary.State != "rollback_apply_recorded" {
		t.Fatalf("V4Summary.State = %q, want rollback_apply_recorded", summary.V4Summary.State)
	}
	if summary.V4Summary.SelectedRollbackID != "rollback-1" {
		t.Fatalf("V4Summary.SelectedRollbackID = %q, want rollback-1", summary.V4Summary.SelectedRollbackID)
	}
	if summary.V4Summary.SelectedRollbackApplyID != "apply-1" {
		t.Fatalf("V4Summary.SelectedRollbackApplyID = %q, want apply-1", summary.V4Summary.SelectedRollbackApplyID)
	}
}

func TestProcessDirectRollbackApplyPhaseCommandRejectsInvalidTransition(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	beforeBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_PHASE) error = nil, want invalid transition rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `phase transition "recorded" -> "ready_to_apply" is invalid`) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE) error = %q, want invalid transition rejection", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseRecorded {
		t.Fatalf("LoadRollbackApplyRecord().Phase = %q, want recorded after rejection", record.Phase)
	}
	if record.ActivationState != missioncontrol.RollbackApplyActivationStateUnchanged {
		t.Fatalf("LoadRollbackApplyRecord().ActivationState = %q, want unchanged after rejection", record.ActivationState)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforeBytes), string(afterBytes))
	}
}

func TestProcessDirectRollbackApplyExecuteCommandSwitchesPointerAndIsReplaySafe(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)
	now := time.Date(2026, 4, 21, 12, 5, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now,
		VerifiedBy:        "operator",
		RollbackRecordRef: "promotion:promotion-1",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}

	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE first) error = %v", err)
	}
	if resp != "Executed rollback-apply pointer switch job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE first) response = %q, want execute acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhasePointerSwitchedReloadPending {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want pointer_switched_reload_pending", record.Phase)
	}
	if record.ActivationState != missioncontrol.RollbackApplyActivationStateUnchanged {
		t.Fatalf("RollbackApplyRecord.ActivationState = %q, want unchanged", record.ActivationState)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ActivePackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-base", gotPointer.ActivePackID)
	}
	if gotPointer.PreviousActivePackID != "pack-candidate" {
		t.Fatalf("LoadActiveRuntimePackPointer().PreviousActivePackID = %q, want pack-candidate", gotPointer.PreviousActivePackID)
	}
	if gotPointer.LastKnownGoodPackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().LastKnownGoodPackID = %q, want pack-base", gotPointer.LastKnownGoodPackID)
	}
	if gotPointer.UpdateRecordRef != "rollback_apply:apply-1" {
		t.Fatalf("LoadActiveRuntimePackPointer().UpdateRecordRef = %q, want rollback_apply:apply-1", gotPointer.UpdateRecordRef)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}

	firstPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer first) error = %v", err)
	}

	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE second) error = %v", err)
	}
	if resp != "Selected rollback-apply pointer switch job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE second) response = %q, want replay acknowledgement", resp)
	}

	secondPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer second) error = %v", err)
	}
	if string(firstPointerBytes) != string(secondPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on replay\nfirst:\n%s\nsecond:\n%s", string(firstPointerBytes), string(secondPointerBytes))
	}

	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RuntimePackIdentity == nil {
		t.Fatal("RuntimePackIdentity = nil, want runtime pack identity block")
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RuntimePackIdentity.Active.ActivePackID != "pack-base" {
		t.Fatalf("RuntimePackIdentity.Active.ActivePackID = %q, want pack-base", summary.RuntimePackIdentity.Active.ActivePackID)
	}
	if summary.RuntimePackIdentity.Active.PreviousActivePackID != "pack-candidate" {
		t.Fatalf("RuntimePackIdentity.Active.PreviousActivePackID = %q, want pack-candidate", summary.RuntimePackIdentity.Active.PreviousActivePackID)
	}
	if summary.RollbackApplyIdentity.Applies[0].Phase != string(missioncontrol.RollbackApplyPhasePointerSwitchedReloadPending) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].Phase = %q, want pointer_switched_reload_pending", summary.RollbackApplyIdentity.Applies[0].Phase)
	}
	if summary.RollbackApplyIdentity.Applies[0].ActivationState != string(missioncontrol.RollbackApplyActivationStateUnchanged) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].ActivationState = %q, want unchanged", summary.RollbackApplyIdentity.Applies[0].ActivationState)
	}
}

func TestProcessDirectRollbackApplyExecuteCommandRejectsInvalidPhaseWithoutPointerMutation(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = nil, want invalid phase rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `phase "recorded" does not permit pointer switch execution`) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = %q, want invalid phase rejection", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseRecorded {
		t.Fatalf("LoadRollbackApplyRecord().Phase = %q, want recorded after rejection", record.Phase)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestProcessDirectRollbackApplyReloadCommandSucceedsWithoutSecondPointerMutation(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)
	now := time.Date(2026, 4, 21, 12, 6, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now,
		VerifiedBy:        "operator",
		RollbackRecordRef: "promotion:promotion-1",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before reload) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before reload) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RELOAD job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD first) error = %v", err)
	}
	if resp != "Executed rollback-apply reload/apply job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD first) response = %q, want reload/apply acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseReloadApplySucceeded {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want reload_apply_succeeded", record.Phase)
	}
	if record.ExecutionError != "" {
		t.Fatalf("RollbackApplyRecord.ExecutionError = %q, want empty", record.ExecutionError)
	}

	firstPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer first) error = %v", err)
	}
	if string(beforePointerBytes) != string(firstPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(firstPointerBytes))
	}

	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_RELOAD job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD second) error = %v", err)
	}
	if resp != "Selected rollback-apply reload/apply job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD second) response = %q, want replay acknowledgement", resp)
	}

	secondPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer second) error = %v", err)
	}
	if string(firstPointerBytes) != string(secondPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on reload/apply replay\nfirst:\n%s\nsecond:\n%s", string(firstPointerBytes), string(secondPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after reload) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ActivePackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-base", gotPointer.ActivePackID)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RollbackApplyIdentity.Applies[0].Phase != string(missioncontrol.RollbackApplyPhaseReloadApplySucceeded) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].Phase = %q, want reload_apply_succeeded", summary.RollbackApplyIdentity.Applies[0].Phase)
	}
}

func TestProcessDirectRollbackApplyReloadCommandRetriesFromRecoveryNeeded(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)
	now := time.Date(2026, 4, 21, 12, 8, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now,
		VerifiedBy:        "operator",
		RollbackRecordRef: "promotion:promotion-1",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = %v", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	recoveryAt := record.CreatedAt.Add(time.Minute)
	record.Phase = missioncontrol.RollbackApplyPhaseReloadApplyInProgress
	record.ExecutionError = ""
	record.PhaseUpdatedAt = recoveryAt.UTC()
	record.PhaseUpdatedBy = "operator"
	record = missioncontrol.NormalizeRollbackApplyRecord(record)
	if err := missioncontrol.ValidateRollbackApplyRecord(record); err != nil {
		t.Fatalf("ValidateRollbackApplyRecord() error = %v", err)
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreRollbackApplyPath(root, "apply-1"), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(reload_apply_in_progress) error = %v", err)
	}
	if _, changed, err := missioncontrol.ReconcileRollbackApplyRecoveryNeeded(root, "apply-1", "operator", recoveryAt.Add(time.Minute)); err != nil {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileRollbackApplyRecoveryNeeded() changed = false, want true")
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before retry) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before retry) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RELOAD job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD retry) error = %v", err)
	}
	if resp != "Executed rollback-apply reload/apply job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD retry) response = %q, want reload/apply acknowledgement", resp)
	}

	record, err = missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord(retry result) error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseReloadApplySucceeded {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want reload_apply_succeeded", record.Phase)
	}
	if record.ExecutionError != "" {
		t.Fatalf("RollbackApplyRecord.ExecutionError = %q, want empty", record.ExecutionError)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after retry) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during retry reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after retry) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during retry reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}
}

func TestProcessDirectRollbackApplyFailCommandResolvesRecoveryNeededAndPreservesStatus(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)
	now := time.Date(2026, 4, 21, 12, 10, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now,
		VerifiedBy:        "operator",
		RollbackRecordRef: "promotion:promotion-1",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = %v", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	recoveryAt := record.CreatedAt.Add(time.Minute)
	record.Phase = missioncontrol.RollbackApplyPhaseReloadApplyInProgress
	record.ExecutionError = ""
	record.PhaseUpdatedAt = recoveryAt.UTC()
	record.PhaseUpdatedBy = "operator"
	record = missioncontrol.NormalizeRollbackApplyRecord(record)
	if err := missioncontrol.ValidateRollbackApplyRecord(record); err != nil {
		t.Fatalf("ValidateRollbackApplyRecord() error = %v", err)
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreRollbackApplyPath(root, "apply-1"), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(reload_apply_in_progress) error = %v", err)
	}
	if _, changed, err := missioncontrol.ReconcileRollbackApplyRecoveryNeeded(root, "apply-1", "operator", recoveryAt.Add(time.Minute)); err != nil {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileRollbackApplyRecoveryNeeded() changed = false, want true")
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before fail) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before fail) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_FAIL job-1 apply-1 operator requested stop after recovery review", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL) error = %v", err)
	}
	if resp != "Resolved rollback-apply terminal failure job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL) response = %q, want terminal failure acknowledgement", resp)
	}

	record, err = missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord(result) error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseReloadApplyFailed {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want reload_apply_failed", record.Phase)
	}
	if record.ExecutionError != "operator_terminal_failure: operator requested stop after recovery review" {
		t.Fatalf("RollbackApplyRecord.ExecutionError = %q, want deterministic terminal failure detail", record.ExecutionError)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after fail) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during terminal failure resolution\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after fail) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during terminal failure resolution\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RollbackApplyIdentity.Applies[0].Phase != string(missioncontrol.RollbackApplyPhaseReloadApplyFailed) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].Phase = %q, want reload_apply_failed", summary.RollbackApplyIdentity.Applies[0].Phase)
	}
}

func TestProcessDirectRollbackApplyFailCommandRequiresReasonAndRejectsInvalidStartingPhase(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_FAIL job-1 apply-1", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_FAIL missing reason) error = nil, want required reason rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL missing reason) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), "terminal failure reason is required") {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL missing reason) error = %q, want required reason rejection", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_FAIL job-1 apply-1 operator requested stop after recovery review", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_FAIL) error = nil, want invalid phase rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `phase "recorded" does not permit terminal failure resolution`) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL) error = %q, want invalid phase rejection", err)
	}
	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed after invalid terminal failure rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestProcessDirectRollbackApplyReloadCommandRejectsInvalidStartingPhase(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RELOAD job-1 apply-1", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_RELOAD) error = nil, want invalid phase rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `phase "recorded" does not permit reload/apply execution`) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD) error = %q, want invalid phase rejection", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseRecorded {
		t.Fatalf("LoadRollbackApplyRecord().Phase = %q, want recorded after rejection", record.Phase)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}
