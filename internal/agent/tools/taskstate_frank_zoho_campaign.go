package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func (s *TaskState) RecordFrankZohoSendReceipt(result string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return nil
	}

	nextRuntime, appended, err := missioncontrol.AppendFrankZohoSendReceipt(*ec.Runtime, ec.Step.ID, result)
	if err != nil || !appended {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) SyncFrankZohoCampaignInboundReplies() (int, error) {
	return s.syncFrankZohoCampaignInboundRepliesAt(taskStateTransitionTimestamp(taskStateNowUTC()))
}

func (s *TaskState) syncFrankZohoCampaignInboundRepliesAt(now time.Time) (int, error) {
	if s == nil {
		return 0, nil
	}
	now = taskStateTransitionTimestamp(now)

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Step.CampaignRef == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return 0, nil
	}

	if err := missioncontrol.RequireExecutionContextCampaignReadiness(ec); err != nil {
		return 0, err
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return 0, err
	}
	sender, err := resolveFrankZohoCampaignSender(preflight, false)
	if err != nil {
		return 0, err
	}

	replies, err := readFrankZohoCampaignInboundReplies(context.Background(), sender.ProviderAccountID)
	if err != nil {
		return 0, err
	}
	bounces, err := readFrankZohoCampaignBounceEvidence(context.Background(), sender.ProviderAccountID)
	if err != nil {
		return 0, err
	}
	var outboundRecords []missioncontrol.CampaignZohoEmailOutboundActionRecord
	if len(bounces) > 0 {
		outboundRecords, err = missioncontrol.ListCommittedAllCampaignZohoEmailOutboundActionRecords(ec.MissionStoreRoot)
		if err != nil {
			return 0, err
		}
	}

	nextRuntime := *missioncontrol.CloneJobRuntimeState(ec.Runtime)
	appended := 0
	for _, reply := range replies {
		reply.StepID = ec.Step.ID
		updatedRuntime, changed, err := missioncontrol.AppendFrankZohoInboundReply(nextRuntime, reply)
		if err != nil {
			return 0, err
		}
		if !changed {
			continue
		}
		nextRuntime = updatedRuntime
		appended++
	}
	evidenceChanged := false
	for _, bounce := range bounces {
		bounce.StepID = ec.Step.ID
		if action, ok := missioncontrol.AttributedCampaignZohoEmailOutboundActionForBounce(bounce, outboundRecords); ok {
			bounce.CampaignID = strings.TrimSpace(action.CampaignID)
			bounce.OutboundActionID = strings.TrimSpace(action.ActionID)
		} else {
			bounce.CampaignID = ""
			bounce.OutboundActionID = ""
		}
		updatedRuntime, changed, err := missioncontrol.AppendFrankZohoBounceEvidence(nextRuntime, bounce)
		if err != nil {
			return 0, err
		}
		if !changed {
			continue
		}
		nextRuntime = updatedRuntime
		evidenceChanged = true
	}
	if appended > 0 {
		s.mu.Lock()
		err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
		s.mu.Unlock()
		if err != nil {
			return 0, err
		}
		s.notifyRuntimeChanged()

		s.mu.Lock()
		ec = missioncontrol.CloneExecutionContext(s.executionContext)
		hasExecutionContext = s.hasExecutionContext
		s.mu.Unlock()
		if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
			return appended, nil
		}
		nextRuntime = *missioncontrol.CloneJobRuntimeState(ec.Runtime)
	}
	if appended == 0 && evidenceChanged {
		s.mu.Lock()
		err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
		s.mu.Unlock()
		if err != nil {
			return 0, err
		}
		s.notifyRuntimeChanged()

		s.mu.Lock()
		ec = missioncontrol.CloneExecutionContext(s.executionContext)
		hasExecutionContext = s.hasExecutionContext
		s.mu.Unlock()
		if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
			return appended, nil
		}
		nextRuntime = *missioncontrol.CloneJobRuntimeState(ec.Runtime)
	}

	workItems, err := missioncontrol.LoadMissingCommittedCampaignZohoEmailReplyWorkItems(ec.MissionStoreRoot, preflight.Campaign.CampaignID, now)
	if err != nil {
		return 0, err
	}
	workItemChanged := false
	for _, item := range workItems {
		if _, exists := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(nextRuntime, item.InboundReplyID); exists {
			continue
		}
		updatedRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(nextRuntime, item)
		if err != nil {
			return 0, err
		}
		if !changed {
			continue
		}
		nextRuntime = updatedRuntime
		workItemChanged = true
	}
	if appended == 0 && !workItemChanged {
		return 0, nil
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return appended, err
}

func (s *TaskState) PrepareFrankZohoCampaignSend(args map[string]interface{}) (string, bool, error) {
	if s == nil {
		return "", false, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return "", false, nil
	}
	operationNow := taskStateTransitionTimestamp(taskStateNowUTC())

	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return "", false, err
	}
	_, hasInboundReplyID, err := frankZohoOptionalStringArg(args, "inbound_reply_id")
	if err != nil {
		return "", false, err
	}
	if hasInboundReplyID || (preflight.Campaign != nil && missioncontrol.CampaignZohoEmailStopConditionsRequireInboundReplies(preflight.Campaign.StopConditions)) {
		if _, err := s.syncFrankZohoCampaignInboundRepliesAt(operationNow); err != nil {
			return "", false, err
		}
		s.mu.Lock()
		ec = missioncontrol.CloneExecutionContext(s.executionContext)
		hasExecutionContext = s.hasExecutionContext
		s.mu.Unlock()
		if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
			return "", false, nil
		}
	}

	intent, err := buildFrankZohoCampaignSendIntent(ec, args, operationNow)
	if err != nil {
		return "", false, err
	}
	action := intent.PreparedAction
	if existing, ok := missioncontrol.FindCampaignZohoEmailOutboundAction(*ec.Runtime, action.ActionID); ok {
		switch existing.State {
		case missioncontrol.CampaignZohoEmailOutboundActionStateVerified:
			nextRuntime, runtimeChanged, err := transitionFrankZohoCampaignReplyWorkItemResponded(*ec.Runtime, existing, operationNow)
			if err != nil {
				return "", true, err
			}
			if runtimeChanged {
				s.mu.Lock()
				err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
				s.mu.Unlock()
				if err != nil {
					return "", true, err
				}
				s.notifyRuntimeChanged()
				ec.Runtime = &nextRuntime
			}
			receipt, err := frankZohoSendReceiptFromCampaignAction(existing)
			if err != nil {
				return "", false, err
			}
			return receipt, true, nil
		case missioncontrol.CampaignZohoEmailOutboundActionStateSent:
			verifiedProof, err := verifyFrankZohoCampaignSendProof(context.Background(), frankZohoCampaignProofFromAction(existing))
			if err != nil {
				return "", true, fmt.Errorf("%s: campaign outbound action %q remains blocked until provider-mailbox verification/finalize succeeds: %w", frankZohoSendEmailToolName, existing.ActionID, err)
			}
			if len(verifiedProof) != 1 {
				return "", true, fmt.Errorf("%s: campaign outbound action %q remains blocked until provider-mailbox verification/finalize returns exactly one proof record", frankZohoSendEmailToolName, existing.ActionID)
			}
			finalized, err := finalizeFrankZohoCampaignActionFromProof(existing, verifiedProof[0], taskStateTransitionTimestamp(operationNow, existing.SentAt))
			if err != nil {
				return "", true, fmt.Errorf("%s: campaign outbound action %q remains blocked until provider-mailbox verification/finalize reconciles it: %w", frankZohoSendEmailToolName, existing.ActionID, err)
			}
			nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailOutboundAction(*ec.Runtime, finalized)
			if err != nil {
				return "", true, err
			}
			nextRuntime, workItemChanged, err := transitionFrankZohoCampaignReplyWorkItemResponded(nextRuntime, finalized, operationNow)
			if err != nil {
				return "", true, err
			}
			if changed || workItemChanged {
				s.mu.Lock()
				err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
				s.mu.Unlock()
				if err != nil {
					return "", true, err
				}
				s.notifyRuntimeChanged()
				ec.Runtime = &nextRuntime
			}
			receipt, err := frankZohoSendReceiptFromCampaignAction(finalized)
			if err != nil {
				return "", true, err
			}
			return receipt, true, nil
		case missioncontrol.CampaignZohoEmailOutboundActionStatePrepared:
			return "", true, fmt.Errorf("%s: campaign outbound action %q is already prepared without provider receipt proof; refusing to resend until reconciled", frankZohoSendEmailToolName, existing.ActionID)
		case missioncontrol.CampaignZohoEmailOutboundActionStateFailed:
			return "", true, fmt.Errorf("%s: campaign outbound action %q is terminally failed and will not be resent automatically", frankZohoSendEmailToolName, existing.ActionID)
		default:
			return "", true, fmt.Errorf("%s: campaign outbound action %q has unsupported state %q", frankZohoSendEmailToolName, existing.ActionID, existing.State)
		}
	}
	if action.ReplyToInboundReplyID != "" {
		nextRuntime, err := claimFrankZohoCampaignReplyWorkItem(ec, *ec.Runtime, action, operationNow)
		if err != nil {
			return "", false, err
		}
		ec.Runtime = &nextRuntime
	}
	if hasInboundReplyID || action.ReplyToInboundReplyID != "" {
		inboundReplyID := action.ReplyToInboundReplyID
		followUpActions, err := missioncontrol.ListCommittedCampaignZohoEmailFollowUpActionsByInboundReply(ec.MissionStoreRoot, inboundReplyID)
		if err != nil {
			return "", false, err
		}
		for _, record := range followUpActions {
			if strings.TrimSpace(record.ActionID) == action.ActionID {
				continue
			}
			switch missioncontrol.CampaignZohoEmailOutboundActionState(strings.TrimSpace(record.State)) {
			case missioncontrol.CampaignZohoEmailOutboundActionStatePrepared, missioncontrol.CampaignZohoEmailOutboundActionStateSent:
				return "", false, fmt.Errorf("%s: inbound_reply_id %q already has unresolved follow-up action %q in state %q; refusing to prepare another follow-up until it is finalized", frankZohoSendEmailToolName, inboundReplyID, strings.TrimSpace(record.ActionID), strings.TrimSpace(record.State))
			}
		}
	}

	nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailOutboundAction(*ec.Runtime, action)
	if err != nil || !changed {
		return "", false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return "", false, err
}

func (s *TaskState) RecordFrankZohoCampaignSend(args map[string]interface{}, result string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return nil
	}
	operationNow := taskStateTransitionTimestamp(taskStateNowUTC())

	prepared, err := buildFrankZohoPreparedCampaignAction(ec, args, operationNow)
	if err != nil {
		return err
	}
	if existing, ok := missioncontrol.FindCampaignZohoEmailOutboundAction(*ec.Runtime, prepared.ActionID); ok {
		prepared = existing
	}
	receipt, err := missioncontrol.ParseFrankZohoSendReceipt(result)
	if err != nil {
		return err
	}
	receipt.StepID = ec.Step.ID
	if err := missioncontrol.ValidateFrankZohoSendReceipt(receipt); err != nil {
		return err
	}
	sent, err := missioncontrol.BuildCampaignZohoEmailOutboundSentAction(prepared, receipt, taskStateTransitionTimestamp(operationNow, prepared.PreparedAt))
	if err != nil {
		return err
	}

	nextRuntime, _, err := missioncontrol.UpsertCampaignZohoEmailOutboundAction(*ec.Runtime, sent)
	if err != nil {
		return err
	}
	nextRuntime, _, err = missioncontrol.AppendFrankZohoSendReceipt(nextRuntime, ec.Step.ID, result)
	if err != nil {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) RecordFrankZohoCampaignSendFailure(args map[string]interface{}, sendErr error) error {
	if s == nil || sendErr == nil {
		return nil
	}

	var terminalFailure interface {
		Failure() missioncontrol.CampaignZohoEmailOutboundFailure
	}
	if !errors.As(sendErr, &terminalFailure) {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return nil
	}
	operationNow := taskStateTransitionTimestamp(taskStateNowUTC())

	prepared, err := buildFrankZohoPreparedCampaignAction(ec, args, operationNow)
	if err != nil {
		return err
	}
	if existing, ok := missioncontrol.FindCampaignZohoEmailOutboundAction(*ec.Runtime, prepared.ActionID); ok {
		prepared = existing
	}
	if prepared.State != missioncontrol.CampaignZohoEmailOutboundActionStatePrepared {
		return nil
	}
	failed, err := missioncontrol.BuildCampaignZohoEmailOutboundFailedAction(prepared, terminalFailure.Failure(), taskStateTransitionTimestamp(operationNow, prepared.PreparedAt))
	if err != nil {
		return err
	}
	nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailOutboundAction(*ec.Runtime, failed)
	if err != nil || !changed {
		return err
	}
	nextRuntime, workItemChanged, err := transitionFrankZohoCampaignReplyWorkItemOnFailure(ec, nextRuntime, failed, operationNow)
	if err != nil {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil && (changed || workItemChanged) {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) ManageFrankZohoCampaignReplyWorkItem(args map[string]interface{}) (string, bool, error) {
	if s == nil {
		return "", true, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Step.CampaignRef == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return "", true, nil
	}

	if _, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec); err != nil {
		return "", true, err
	}
	inboundReplyID, err := frankZohoRequiredStringArg(args, "inbound_reply_id")
	if err != nil {
		return "", true, err
	}
	action, err := frankZohoRequiredStringArg(args, "action")
	if err != nil {
		return "", true, err
	}
	now := time.Now().UTC()
	nextRuntime, item, err := ensureFrankZohoCampaignReplyWorkItem(ec, *ec.Runtime, inboundReplyID, now)
	if err != nil {
		return "", true, err
	}

	switch item.State {
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateClaimed:
		return "", true, fmt.Errorf("%s: inbound_reply_id %q is currently claimed by follow-up action %q", frankZohoManageReplyWorkItemToolName, inboundReplyID, item.ClaimedFollowUpActionID)
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateResponded, missioncontrol.CampaignZohoEmailReplyWorkItemStateIgnored:
		return "", true, fmt.Errorf("%s: inbound_reply_id %q is already terminal in state %q", frankZohoManageReplyWorkItemToolName, inboundReplyID, item.State)
	}

	var mutated missioncontrol.CampaignZohoEmailReplyWorkItem
	switch action {
	case "ignore":
		mutated, err = missioncontrol.BuildCampaignZohoEmailReplyWorkItemIgnored(item, now)
	case "defer":
		deferUntilText, argErr := frankZohoRequiredStringArg(args, "defer_until")
		if argErr != nil {
			return "", true, argErr
		}
		deferUntil, parseErr := time.Parse(time.RFC3339, deferUntilText)
		if parseErr != nil {
			return "", true, fmt.Errorf("%s: defer_until must be RFC3339: %w", frankZohoManageReplyWorkItemToolName, parseErr)
		}
		if !deferUntil.UTC().After(now) {
			return "", true, fmt.Errorf("%s: defer_until must be in the future", frankZohoManageReplyWorkItemToolName)
		}
		mutated, err = missioncontrol.BuildCampaignZohoEmailReplyWorkItemDeferred(item, deferUntil.UTC(), now)
	default:
		return "", true, fmt.Errorf("%s: action %q is not supported", frankZohoManageReplyWorkItemToolName, action)
	}
	if err != nil {
		return "", true, err
	}

	nextRuntime, _, err = missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(nextRuntime, mutated)
	if err != nil {
		return "", true, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err != nil {
		return "", true, err
	}
	s.notifyRuntimeChanged()

	payload, err := json.Marshal(struct {
		InboundReplyID string `json:"inbound_reply_id"`
		State          string `json:"state"`
		DeferredUntil  string `json:"deferred_until,omitempty"`
	}{
		InboundReplyID: mutated.InboundReplyID,
		State:          string(mutated.State),
		DeferredUntil:  formatTaskStateRFC3339(mutated.DeferredUntil),
	})
	if err != nil {
		return "", true, err
	}
	return string(payload), true, nil
}

func claimFrankZohoCampaignReplyWorkItem(ec missioncontrol.ExecutionContext, runtime missioncontrol.JobRuntimeState, action missioncontrol.CampaignZohoEmailOutboundAction, now time.Time) (missioncontrol.JobRuntimeState, error) {
	if strings.TrimSpace(action.ReplyToInboundReplyID) == "" {
		return runtime, nil
	}
	item, ok := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime, action.ReplyToInboundReplyID)
	if !ok {
		loaded, found, err := missioncontrol.LoadCommittedCampaignZohoEmailReplyWorkItemByInboundReply(ec.MissionStoreRoot, action.CampaignID, action.ReplyToInboundReplyID)
		if err != nil {
			return missioncontrol.JobRuntimeState{}, err
		}
		if !found {
			return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q is missing a committed reply work item", frankZohoSendEmailToolName, action.ReplyToInboundReplyID)
		}
		item = loaded
	}
	switch item.State {
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateOpen:
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateDeferred:
		if item.DeferredUntil.After(now.UTC()) {
			return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q is deferred until %s", frankZohoSendEmailToolName, action.ReplyToInboundReplyID, item.DeferredUntil.Format(time.RFC3339))
		}
		reopened, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemReopened(item, taskStateTransitionTimestamp(now, item.CreatedAt, item.UpdatedAt))
		if err != nil {
			return missioncontrol.JobRuntimeState{}, err
		}
		item = reopened
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateClaimed:
		if strings.TrimSpace(item.ClaimedFollowUpActionID) != action.ActionID {
			return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q already has claimed follow-up action %q", frankZohoSendEmailToolName, action.ReplyToInboundReplyID, item.ClaimedFollowUpActionID)
		}
		return runtime, nil
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateResponded, missioncontrol.CampaignZohoEmailReplyWorkItemStateIgnored:
		return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q is not eligible for follow-up in state %q", frankZohoSendEmailToolName, action.ReplyToInboundReplyID, item.State)
	default:
		return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q has unsupported reply work item state %q", frankZohoSendEmailToolName, action.ReplyToInboundReplyID, item.State)
	}
	claimed, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemClaimed(item, action.ActionID, taskStateTransitionTimestamp(now, item.CreatedAt, item.UpdatedAt, action.PreparedAt))
	if err != nil {
		return missioncontrol.JobRuntimeState{}, err
	}
	nextRuntime, _, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(runtime, claimed)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, err
	}
	return nextRuntime, nil
}

func transitionFrankZohoCampaignReplyWorkItemResponded(runtime missioncontrol.JobRuntimeState, action missioncontrol.CampaignZohoEmailOutboundAction, now time.Time) (missioncontrol.JobRuntimeState, bool, error) {
	if strings.TrimSpace(action.ReplyToInboundReplyID) == "" {
		return runtime, false, nil
	}
	item, ok := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime, action.ReplyToInboundReplyID)
	if !ok {
		return runtime, false, nil
	}
	if item.State == missioncontrol.CampaignZohoEmailReplyWorkItemStateResponded {
		return runtime, false, nil
	}
	if item.State != missioncontrol.CampaignZohoEmailReplyWorkItemStateClaimed || strings.TrimSpace(item.ClaimedFollowUpActionID) != action.ActionID {
		return runtime, false, nil
	}
	responded, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemResponded(item, taskStateTransitionTimestamp(now, item.CreatedAt, item.UpdatedAt, action.SentAt, action.VerifiedAt))
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(runtime, responded)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	return nextRuntime, changed, nil
}

func transitionFrankZohoCampaignReplyWorkItemOnFailure(ec missioncontrol.ExecutionContext, runtime missioncontrol.JobRuntimeState, action missioncontrol.CampaignZohoEmailOutboundAction, now time.Time) (missioncontrol.JobRuntimeState, bool, error) {
	if strings.TrimSpace(action.ReplyToInboundReplyID) == "" {
		return runtime, false, nil
	}
	item, ok := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime, action.ReplyToInboundReplyID)
	if !ok {
		return runtime, false, nil
	}
	if item.State != missioncontrol.CampaignZohoEmailReplyWorkItemStateClaimed || strings.TrimSpace(item.ClaimedFollowUpActionID) != action.ActionID {
		return runtime, false, nil
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	decision, err := missioncontrol.DeriveCampaignZohoEmailSendGateDecisionFromRuntime(*preflight.Campaign, runtime)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	if !decision.Allowed {
		return runtime, false, nil
	}
	reopened, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemReopened(item, taskStateTransitionTimestamp(now, item.CreatedAt, item.UpdatedAt, action.FailedAt))
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(runtime, reopened)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	return nextRuntime, changed, nil
}

func ensureFrankZohoCampaignReplyWorkItem(ec missioncontrol.ExecutionContext, runtime missioncontrol.JobRuntimeState, inboundReplyID string, now time.Time) (missioncontrol.JobRuntimeState, missioncontrol.CampaignZohoEmailReplyWorkItem, error) {
	if item, ok := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime, inboundReplyID); ok {
		return runtime, item, nil
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
	}
	missingItems, err := missioncontrol.LoadMissingCommittedCampaignZohoEmailReplyWorkItems(ec.MissionStoreRoot, preflight.Campaign.CampaignID, now)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
	}
	nextRuntime := runtime
	for _, item := range missingItems {
		updatedRuntime, _, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(nextRuntime, item)
		if err != nil {
			return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
		}
		nextRuntime = updatedRuntime
		if item.InboundReplyID == inboundReplyID {
			return nextRuntime, item, nil
		}
	}
	loaded, ok, err := missioncontrol.LoadCommittedCampaignZohoEmailReplyWorkItemByInboundReply(ec.MissionStoreRoot, preflight.Campaign.CampaignID, inboundReplyID)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
	}
	if !ok {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, fmt.Errorf("%s: inbound_reply_id %q does not resolve to a committed reply work item", frankZohoManageReplyWorkItemToolName, inboundReplyID)
	}
	nextRuntime, _, err = missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(nextRuntime, loaded)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
	}
	return nextRuntime, loaded, nil
}
