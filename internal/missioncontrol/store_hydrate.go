package missioncontrol

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type hydratedCommittedStoreState struct {
	Runtime   JobRuntimeState
	Control   *RuntimeControlContext
	ActiveJob *ActiveJobRecord
	Artifacts []ArtifactRecord
}

func HydrateCommittedJobRuntimeState(root, jobID string, now time.Time) (JobRuntimeState, error) {
	hydrated, err := hydrateCommittedStoreState(root, jobID, now)
	if err != nil {
		return JobRuntimeState{}, err
	}
	return hydrated.Runtime, nil
}

func HydrateCommittedRuntimeControlContext(root, jobID string, now time.Time) (*RuntimeControlContext, error) {
	hydrated, err := hydrateCommittedStoreState(root, jobID, now)
	if err != nil {
		return nil, err
	}
	return CloneRuntimeControlContext(hydrated.Control), nil
}

func hydrateCommittedStoreState(root, jobID string, now time.Time) (hydratedCommittedStoreState, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	stepRecords, err := ListCommittedStepRuntimeRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	requestRecords, err := ListCommittedApprovalRequestRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	grantRecords, err := ListCommittedApprovalGrantRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	auditRecords, err := ListCommittedAuditEventRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	artifactRecords, err := ListCommittedArtifactRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	campaignZohoEmailOutboundActionRecords, err := ListCommittedCampaignZohoEmailOutboundActionRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	campaignZohoEmailReplyWorkItemRecords, err := ListCommittedCampaignZohoEmailReplyWorkItemRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	frankZohoSendReceiptRecords, err := ListCommittedFrankZohoSendReceiptRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	frankZohoInboundReplyRecords, err := ListCommittedFrankZohoInboundReplyRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	frankZohoBounceEvidenceRecords, err := ListCommittedFrankZohoBounceEvidenceRecords(root, jobID)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}

	runtime := hydrateCommittedJobRuntimeRecord(jobRuntime)
	completedSteps, failedSteps := hydrateCommittedRuntimeStepOutcomes(stepRecords)
	runtime.CompletedSteps = completedSteps
	runtime.FailedSteps = failedSteps
	runtime.ApprovalRequests = hydrateCommittedApprovalRequests(requestRecords)
	runtime.ApprovalGrants = hydrateCommittedApprovalGrants(grantRecords)
	runtime.AuditHistory = hydrateCommittedAuditHistory(auditRecords)
	runtime.CampaignZohoEmailOutboundActions = hydrateCommittedCampaignZohoEmailOutboundActions(campaignZohoEmailOutboundActionRecords)
	runtime.CampaignZohoEmailReplyWorkItems = hydrateCommittedCampaignZohoEmailReplyWorkItems(campaignZohoEmailReplyWorkItemRecords)
	runtime.FrankZohoSendReceipts = hydrateCommittedFrankZohoSendReceipts(frankZohoSendReceiptRecords)
	runtime.FrankZohoInboundReplies = hydrateCommittedFrankZohoInboundReplies(frankZohoInboundReplyRecords)
	runtime.FrankZohoBounceEvidence = hydrateCommittedFrankZohoBounceEvidence(frankZohoBounceEvidenceRecords)
	runtime, _ = NormalizeHydratedApprovalRequests(runtime, now)

	control, err := hydrateCommittedRuntimeControl(root, jobRuntime)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	activeJob, err := hydrateCommittedActiveJob(root, jobRuntime)
	if err != nil {
		return hydratedCommittedStoreState{}, err
	}
	artifacts := hydrateCommittedArtifacts(artifactRecords)

	if err := validateHydratedCommittedStoreState(runtime, control, activeJob, stepRecords); err != nil {
		return hydratedCommittedStoreState{}, err
	}

	return hydratedCommittedStoreState{
		Runtime:   runtime,
		Control:   control,
		ActiveJob: activeJob,
		Artifacts: artifacts,
	}, nil
}

func hydrateCommittedJobRuntimeRecord(record JobRuntimeRecord) JobRuntimeState {
	return JobRuntimeState{
		JobID:             record.JobID,
		ExecutionPlane:    strings.TrimSpace(record.ExecutionPlane),
		ExecutionHost:     strings.TrimSpace(record.ExecutionHost),
		MissionFamily:     strings.TrimSpace(record.MissionFamily),
		TargetSurfaces:    cloneJobSurfaceRefs(record.TargetSurfaces),
		MutableSurfaces:   cloneJobSurfaceRefs(record.MutableSurfaces),
		ImmutableSurfaces: cloneJobSurfaceRefs(record.ImmutableSurfaces),
		State:             record.State,
		ActiveStepID:      record.ActiveStepID,
		InspectablePlan:   CloneInspectablePlanContext(record.InspectablePlan),
		BudgetBlocker:     cloneRuntimeBudgetBlockerRecord(record.BudgetBlocker),
		WaitingReason:     record.WaitingReason,
		PausedReason:      record.PausedReason,
		AbortedReason:     record.AbortedReason,
		CreatedAt:         record.CreatedAt.UTC(),
		UpdatedAt:         record.UpdatedAt.UTC(),
		StartedAt:         record.StartedAt.UTC(),
		ActiveStepAt:      record.ActiveStepAt.UTC(),
		WaitingAt:         record.WaitingAt.UTC(),
		PausedAt:          record.PausedAt.UTC(),
		AbortedAt:         record.AbortedAt.UTC(),
		CompletedAt:       record.CompletedAt.UTC(),
		FailedAt:          record.FailedAt.UTC(),
	}
}

func hydrateCommittedRuntimeStepOutcomes(records []StepRuntimeRecord) ([]RuntimeStepRecord, []RuntimeStepRecord) {
	completedSource := make([]StepRuntimeRecord, 0, len(records))
	failedSource := make([]StepRuntimeRecord, 0, len(records))
	for _, record := range records {
		switch record.Status {
		case StepRuntimeStatusCompleted:
			completedSource = append(completedSource, record)
		case StepRuntimeStatusFailed:
			failedSource = append(failedSource, record)
		}
	}

	sort.SliceStable(completedSource, func(i, j int) bool {
		return committedStepRecordSortsBefore(completedSource[i], completedSource[j], StepRuntimeStatusCompleted)
	})
	sort.SliceStable(failedSource, func(i, j int) bool {
		return committedStepRecordSortsBefore(failedSource[i], failedSource[j], StepRuntimeStatusFailed)
	})

	completed := make([]RuntimeStepRecord, 0, len(completedSource))
	for _, record := range completedSource {
		completed = append(completed, RuntimeStepRecord{
			StepID:         record.StepID,
			At:             committedStepRecordTimestamp(record, StepRuntimeStatusCompleted),
			ResultingState: cloneRuntimeResultingStateRecord(record.ResultingState),
			Rollback:       cloneRuntimeRollbackRecord(record.Rollback),
		})
	}

	failed := make([]RuntimeStepRecord, 0, len(failedSource))
	for _, record := range failedSource {
		failed = append(failed, RuntimeStepRecord{
			StepID: record.StepID,
			Reason: record.Reason,
			At:     committedStepRecordTimestamp(record, StepRuntimeStatusFailed),
		})
	}

	return completed, failed
}

func committedStepRecordSortsBefore(left, right StepRuntimeRecord, status StepRuntimeStatus) bool {
	if left.LastSeq != right.LastSeq {
		return left.LastSeq < right.LastSeq
	}
	leftAt := committedStepRecordTimestamp(left, status)
	rightAt := committedStepRecordTimestamp(right, status)
	if !leftAt.Equal(rightAt) {
		return leftAt.Before(rightAt)
	}
	if left.StepID != right.StepID {
		return left.StepID < right.StepID
	}
	return left.StepType < right.StepType
}

func committedStepRecordTimestamp(record StepRuntimeRecord, status StepRuntimeStatus) time.Time {
	switch status {
	case StepRuntimeStatusCompleted:
		if !record.CompletedAt.IsZero() {
			return record.CompletedAt.UTC()
		}
	case StepRuntimeStatusFailed:
		if !record.FailedAt.IsZero() {
			return record.FailedAt.UTC()
		}
	}
	if !record.ActivatedAt.IsZero() {
		return record.ActivatedAt.UTC()
	}
	return time.Time{}
}

func hydrateCommittedApprovalRequests(records []ApprovalRequestRecord) []ApprovalRequest {
	if len(records) == 0 {
		return nil
	}

	requests := make([]ApprovalRequest, len(records))
	for i, record := range records {
		requests[i] = cloneApprovalRequest(ApprovalRequest{
			JobID:           record.JobID,
			StepID:          record.StepID,
			RequestedAction: record.RequestedAction,
			Scope:           record.Scope,
			Content:         record.Content,
			RequestedVia:    record.RequestedVia,
			GrantedVia:      record.GrantedVia,
			SessionChannel:  record.SessionChannel,
			SessionChatID:   record.SessionChatID,
			State:           record.State,
			Reason:          record.Reason,
			RequestedAt:     record.RequestedAt.UTC(),
			ExpiresAt:       record.ExpiresAt.UTC(),
			ResolvedAt:      record.ResolvedAt.UTC(),
			SupersededAt:    record.SupersededAt.UTC(),
			RevokedAt:       record.RevokedAt.UTC(),
		})
	}

	sort.SliceStable(requests, func(i, j int) bool {
		if reusableApprovalRequestSortsAfter(requests[i], requests[j]) {
			return false
		}
		if reusableApprovalRequestSortsAfter(requests[j], requests[i]) {
			return true
		}
		return false
	})

	return requests
}

func hydrateCommittedApprovalGrants(records []ApprovalGrantRecord) []ApprovalGrant {
	if len(records) == 0 {
		return nil
	}

	grants := make([]ApprovalGrant, len(records))
	for i, record := range records {
		grants[i] = ApprovalGrant{
			JobID:           record.JobID,
			StepID:          record.StepID,
			RequestedAction: record.RequestedAction,
			Scope:           record.Scope,
			GrantedVia:      record.GrantedVia,
			SessionChannel:  record.SessionChannel,
			SessionChatID:   record.SessionChatID,
			State:           record.State,
			GrantedAt:       record.GrantedAt.UTC(),
			ExpiresAt:       record.ExpiresAt.UTC(),
			RevokedAt:       record.RevokedAt.UTC(),
		}
	}

	sort.SliceStable(grants, func(i, j int) bool {
		if reusableApprovalGrantSortsAfter(grants[i], grants[j]) {
			return false
		}
		if reusableApprovalGrantSortsAfter(grants[j], grants[i]) {
			return true
		}
		return false
	})

	return grants
}

func hydrateCommittedAuditHistory(records []AuditEventRecord) []AuditEvent {
	if len(records) == 0 {
		return nil
	}

	sorted := make([]AuditEventRecord, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Seq != sorted[j].Seq {
			return sorted[i].Seq < sorted[j].Seq
		}
		left := normalizeAuditEvent(sorted[i].Event)
		right := normalizeAuditEvent(sorted[j].Event)
		if !left.Timestamp.Equal(right.Timestamp) {
			return left.Timestamp.Before(right.Timestamp)
		}
		if left.EventID != right.EventID {
			return left.EventID < right.EventID
		}
		if left.StepID != right.StepID {
			return left.StepID < right.StepID
		}
		return left.ToolName < right.ToolName
	})

	history := make([]AuditEvent, len(sorted))
	for i, record := range sorted {
		history[i] = normalizeAuditEvent(record.Event)
	}
	return CloneAuditHistory(history)
}

func hydrateCommittedArtifacts(records []ArtifactRecord) []ArtifactRecord {
	if len(records) == 0 {
		return nil
	}

	artifacts := make([]ArtifactRecord, len(records))
	copy(artifacts, records)
	sort.SliceStable(artifacts, func(i, j int) bool {
		if artifacts[i].LastSeq != artifacts[j].LastSeq {
			return artifacts[i].LastSeq < artifacts[j].LastSeq
		}
		if artifacts[i].StepID != artifacts[j].StepID {
			return artifacts[i].StepID < artifacts[j].StepID
		}
		if artifacts[i].ArtifactID != artifacts[j].ArtifactID {
			return artifacts[i].ArtifactID < artifacts[j].ArtifactID
		}
		if artifacts[i].Path != artifacts[j].Path {
			return artifacts[i].Path < artifacts[j].Path
		}
		if artifacts[i].State != artifacts[j].State {
			return artifacts[i].State < artifacts[j].State
		}
		return artifacts[i].SourceStepID < artifacts[j].SourceStepID
	})
	return artifacts
}

func hydrateCommittedCampaignZohoEmailOutboundActions(records []CampaignZohoEmailOutboundActionRecord) []CampaignZohoEmailOutboundAction {
	if len(records) == 0 {
		return nil
	}

	actions := make([]CampaignZohoEmailOutboundAction, len(records))
	for i, record := range records {
		actions[i] = NormalizeCampaignZohoEmailOutboundAction(CampaignZohoEmailOutboundAction{
			ActionID:                record.ActionID,
			StepID:                  record.StepID,
			CampaignID:              record.CampaignID,
			State:                   CampaignZohoEmailOutboundActionState(record.State),
			Provider:                record.Provider,
			ProviderAccountID:       record.ProviderAccountID,
			FromAddress:             record.FromAddress,
			FromDisplayName:         record.FromDisplayName,
			Addressing:              record.Addressing,
			Subject:                 record.Subject,
			BodyFormat:              record.BodyFormat,
			BodySHA256:              record.BodySHA256,
			PreparedAt:              record.PreparedAt,
			SentAt:                  record.SentAt,
			VerifiedAt:              record.VerifiedAt,
			FailedAt:                record.FailedAt,
			ReplyToInboundReplyID:   record.ReplyToInboundReplyID,
			ReplyToOutboundActionID: record.ReplyToOutboundActionID,
			ProviderMessageID:       record.ProviderMessageID,
			ProviderMailID:          record.ProviderMailID,
			MIMEMessageID:           record.MIMEMessageID,
			OriginalMessageURL:      record.OriginalMessageURL,
			Failure:                 record.Failure,
		})
	}
	return actions
}

func hydrateCommittedFrankZohoSendReceipts(records []FrankZohoSendReceiptRecord) []FrankZohoSendReceipt {
	if len(records) == 0 {
		return nil
	}

	receipts := make([]FrankZohoSendReceipt, len(records))
	for i, record := range records {
		receipts[i] = NormalizeFrankZohoSendReceipt(FrankZohoSendReceipt{
			StepID:             record.StepID,
			Provider:           record.Provider,
			ProviderAccountID:  record.ProviderAccountID,
			FromAddress:        record.FromAddress,
			FromDisplayName:    record.FromDisplayName,
			ProviderMessageID:  record.ProviderMessageID,
			ProviderMailID:     record.ProviderMailID,
			MIMEMessageID:      record.MIMEMessageID,
			OriginalMessageURL: record.OriginalMessageURL,
		})
	}
	return receipts
}

func hydrateCommittedFrankZohoInboundReplies(records []FrankZohoInboundReplyRecord) []FrankZohoInboundReply {
	if len(records) == 0 {
		return nil
	}

	replies := make([]FrankZohoInboundReply, len(records))
	for i, record := range records {
		replies[i] = NormalizeFrankZohoInboundReply(FrankZohoInboundReply{
			ReplyID:            record.ReplyID,
			StepID:             record.StepID,
			Provider:           record.Provider,
			ProviderAccountID:  record.ProviderAccountID,
			ProviderMessageID:  record.ProviderMessageID,
			ProviderMailID:     record.ProviderMailID,
			MIMEMessageID:      record.MIMEMessageID,
			InReplyTo:          record.InReplyTo,
			References:         append([]string(nil), record.References...),
			FromAddress:        record.FromAddress,
			FromDisplayName:    record.FromDisplayName,
			FromAddressCount:   record.FromAddressCount,
			Subject:            record.Subject,
			ReceivedAt:         record.ReceivedAt,
			OriginalMessageURL: record.OriginalMessageURL,
		})
	}
	return replies
}

func hydrateCommittedFrankZohoBounceEvidence(records []FrankZohoBounceEvidenceRecord) []FrankZohoBounceEvidence {
	if len(records) == 0 {
		return nil
	}

	values := make([]FrankZohoBounceEvidence, len(records))
	for i, record := range records {
		values[i] = NormalizeFrankZohoBounceEvidence(FrankZohoBounceEvidence{
			BounceID:                  record.BounceID,
			StepID:                    record.StepID,
			Provider:                  record.Provider,
			ProviderAccountID:         record.ProviderAccountID,
			ProviderMessageID:         record.ProviderMessageID,
			ProviderMailID:            record.ProviderMailID,
			MIMEMessageID:             record.MIMEMessageID,
			InReplyTo:                 record.InReplyTo,
			References:                append([]string(nil), record.References...),
			OriginalProviderMessageID: record.OriginalProviderMessageID,
			OriginalProviderMailID:    record.OriginalProviderMailID,
			OriginalMIMEMessageID:     record.OriginalMIMEMessageID,
			FinalRecipient:            record.FinalRecipient,
			DiagnosticCode:            record.DiagnosticCode,
			ReceivedAt:                record.ReceivedAt,
			OriginalMessageURL:        record.OriginalMessageURL,
			CampaignID:                record.CampaignID,
			OutboundActionID:          record.OutboundActionID,
		})
	}
	return values
}

func hydrateCommittedCampaignZohoEmailReplyWorkItems(records []CampaignZohoEmailReplyWorkItemRecord) []CampaignZohoEmailReplyWorkItem {
	if len(records) == 0 {
		return nil
	}

	items := make([]CampaignZohoEmailReplyWorkItem, len(records))
	for i, record := range records {
		items[i] = NormalizeCampaignZohoEmailReplyWorkItem(CampaignZohoEmailReplyWorkItem{
			ReplyWorkItemID:         record.ReplyWorkItemID,
			InboundReplyID:          record.InboundReplyID,
			CampaignID:              record.CampaignID,
			State:                   CampaignZohoEmailReplyWorkItemState(record.State),
			DeferredUntil:           record.DeferredUntil,
			ClaimedFollowUpActionID: record.ClaimedFollowUpActionID,
			CreatedAt:               record.CreatedAt,
			UpdatedAt:               record.UpdatedAt,
		})
	}
	return items
}

func hydrateCommittedRuntimeControl(root string, jobRuntime JobRuntimeRecord) (*RuntimeControlContext, error) {
	if !HoldsGlobalActiveJobOccupancy(jobRuntime.State) || jobRuntime.ActiveStepID == "" {
		return nil, nil
	}

	record, err := LoadCommittedRuntimeControlRecord(root, jobRuntime.JobID)
	if err != nil {
		if errors.Is(err, ErrRuntimeControlRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &RuntimeControlContext{
		JobID:             record.JobID,
		ExecutionPlane:    strings.TrimSpace(record.ExecutionPlane),
		ExecutionHost:     strings.TrimSpace(record.ExecutionHost),
		MissionFamily:     strings.TrimSpace(record.MissionFamily),
		TargetSurfaces:    cloneJobSurfaceRefs(record.TargetSurfaces),
		MutableSurfaces:   cloneJobSurfaceRefs(record.MutableSurfaces),
		ImmutableSurfaces: cloneJobSurfaceRefs(record.ImmutableSurfaces),
		MaxAuthority:      record.MaxAuthority,
		AllowedTools:      append([]string(nil), record.AllowedTools...),
		Step:              copyStep(record.Step),
	}, nil
}

func hydrateCommittedActiveJob(root string, jobRuntime JobRuntimeRecord) (*ActiveJobRecord, error) {
	if !HoldsGlobalActiveJobOccupancy(jobRuntime.State) {
		return nil, nil
	}

	record, err := LoadCommittedActiveJobRecord(root, jobRuntime.JobID)
	if err != nil {
		if errors.Is(err, ErrActiveJobRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func validateHydratedCommittedStoreState(runtime JobRuntimeState, control *RuntimeControlContext, activeJob *ActiveJobRecord, stepRecords []StepRuntimeRecord) error {
	if runtime.ActiveStepID == "" {
		if control != nil {
			return fmt.Errorf("hydrated runtime control step %q requires an active runtime step", control.Step.ID)
		}
		if activeJob != nil {
			return fmt.Errorf("hydrated active_job step %q requires an active runtime step", activeJob.ActiveStepID)
		}
		return nil
	}

	if control != nil && control.Step.ID != runtime.ActiveStepID {
		return fmt.Errorf("hydrated runtime control step %q does not match runtime active step %q", control.Step.ID, runtime.ActiveStepID)
	}
	if activeJob != nil && activeJob.ActiveStepID != "" && activeJob.ActiveStepID != runtime.ActiveStepID {
		return fmt.Errorf("hydrated active_job step %q does not match runtime active step %q", activeJob.ActiveStepID, runtime.ActiveStepID)
	}

	for _, record := range stepRecords {
		if record.StepID != runtime.ActiveStepID {
			continue
		}
		switch record.Status {
		case StepRuntimeStatusCompleted, StepRuntimeStatusFailed:
			return fmt.Errorf("hydrated runtime active step %q has terminal step status %q", runtime.ActiveStepID, record.Status)
		}
	}

	return nil
}
