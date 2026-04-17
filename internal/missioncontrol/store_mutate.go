package missioncontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const storeProjectedWriterLeaseDuration = time.Minute

func PersistProjectedRuntimeState(root string, lease WriterLockLease, job *Job, runtime JobRuntimeState, control *RuntimeControlContext, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	projectedRuntime := *CloneJobRuntimeState(&runtime)
	projectedControl := CloneRuntimeControlContext(control)
	projectedJob := CloneJob(job)

	if strings.TrimSpace(projectedRuntime.JobID) == "" && projectedJob != nil {
		projectedRuntime.JobID = projectedJob.ID
	}
	if strings.TrimSpace(projectedRuntime.JobID) == "" {
		return fmt.Errorf("mission store projected runtime job_id is required")
	}

	lease.JobID = projectedRuntime.JobID
	lock, err := acquireProjectedWriterLock(root, lease, now)
	if err != nil {
		return err
	}

	batch, err := projectRuntimeStateToStoreBatch(root, lock, projectedJob, projectedRuntime, projectedControl, now)
	if err != nil {
		return err
	}
	return CommitStoreBatch(root, lock, batch)
}

func acquireProjectedWriterLock(root string, lease WriterLockLease, now time.Time) (WriterLockRecord, error) {
	lock, _, err := AcquireWriterLock(root, now, storeProjectedWriterLeaseDuration, lease)
	if err == nil {
		return lock, nil
	}
	if errors.Is(err, ErrWriterLockExpired) {
		return TakeoverWriterLock(root, now, storeProjectedWriterLeaseDuration, lease)
	}
	return WriterLockRecord{}, err
}

func projectRuntimeStateToStoreBatch(root string, lock WriterLockRecord, job *Job, runtime JobRuntimeState, control *RuntimeControlContext, now time.Time) (StoreBatch, error) {
	nextSeq, err := nextProjectedStoreSeq(root, runtime.JobID)
	if err != nil {
		return StoreBatch{}, err
	}
	committedAudits, err := committedAuditEventIDs(root, runtime.JobID)
	if err != nil {
		return StoreBatch{}, err
	}
	plan, err := projectedInspectablePlan(job, runtime)
	if err != nil {
		return StoreBatch{}, err
	}

	jobRuntime := projectJobRuntimeRecord(runtime, plan, lock.WriterEpoch, nextSeq, now)
	stepRecords, err := projectStepRuntimeRecords(runtime, control, plan, nextSeq)
	if err != nil {
		return StoreBatch{}, err
	}
	runtimeControl, err := projectRuntimeControlRecord(runtime, control, lock.WriterEpoch, nextSeq)
	if err != nil {
		return StoreBatch{}, err
	}

	activeJob, removeActiveJob, err := projectActiveJobRecord(runtime, lock, nextSeq, now)
	if err != nil {
		return StoreBatch{}, err
	}

	return StoreBatch{
		JobRuntime:                       jobRuntime,
		RuntimeControl:                   runtimeControl,
		StepRecords:                      stepRecords,
		ApprovalRequests:                 projectApprovalRequestRecords(runtime, nextSeq),
		ApprovalGrants:                   projectApprovalGrantRecords(runtime, nextSeq),
		AuditEvents:                      projectAuditEventRecords(runtime, nextSeq, committedAudits),
		Artifacts:                        projectArtifactRecords(runtime, plan, nextSeq),
		CampaignZohoEmailOutboundActions: projectCampaignZohoEmailOutboundActionRecords(runtime, nextSeq),
		CampaignZohoEmailReplyWorkItems:  projectCampaignZohoEmailReplyWorkItemRecords(runtime, nextSeq),
		FrankZohoSendReceipts:            projectFrankZohoSendReceiptRecords(runtime, nextSeq),
		FrankZohoInboundReplies:          projectFrankZohoInboundReplyRecords(runtime, nextSeq),
		FrankZohoBounceEvidence:          projectFrankZohoBounceEvidenceRecords(runtime, nextSeq),
		ActiveJob:                        activeJob,
		RemoveActiveJob:                  removeActiveJob,
	}, nil
}

func nextProjectedStoreSeq(root, jobID string) (uint64, error) {
	record, err := LoadCommittedJobRuntimeRecord(root, jobID)
	switch {
	case err == nil:
		return record.AppliedSeq + 1, nil
	case errors.Is(err, ErrJobRuntimeRecordNotFound):
		return 1, nil
	default:
		return 0, err
	}
}

func committedAuditEventIDs(root, jobID string) (map[string]struct{}, error) {
	records, err := ListCommittedAuditEventRecords(root, jobID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		event := normalizeAuditEvent(record.Event)
		seen[event.EventID] = struct{}{}
	}
	return seen, nil
}

func projectedInspectablePlan(job *Job, runtime JobRuntimeState) (*InspectablePlanContext, error) {
	if runtime.InspectablePlan != nil {
		return CloneInspectablePlanContext(runtime.InspectablePlan), nil
	}
	if job == nil {
		return nil, nil
	}
	plan, err := BuildInspectablePlanContext(*job)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func projectJobRuntimeRecord(runtime JobRuntimeState, plan *InspectablePlanContext, writerEpoch, nextSeq uint64, now time.Time) JobRuntimeRecord {
	createdAt := firstProjectedTime(runtime.CreatedAt, runtime.StartedAt, runtime.UpdatedAt, now)
	updatedAt := firstProjectedTime(runtime.UpdatedAt, runtime.CompletedAt, runtime.FailedAt, runtime.AbortedAt, runtime.PausedAt, runtime.WaitingAt, runtime.ActiveStepAt, createdAt)
	startedAt := firstProjectedTime(runtime.StartedAt, createdAt)

	return JobRuntimeRecord{
		RecordVersion:   StoreRecordVersion,
		WriterEpoch:     writerEpoch,
		AppliedSeq:      nextSeq,
		JobID:           runtime.JobID,
		State:           runtime.State,
		ActiveStepID:    runtime.ActiveStepID,
		InspectablePlan: CloneInspectablePlanContext(plan),
		BudgetBlocker:   cloneRuntimeBudgetBlockerRecord(runtime.BudgetBlocker),
		WaitingReason:   runtime.WaitingReason,
		PausedReason:    runtime.PausedReason,
		AbortedReason:   runtime.AbortedReason,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		StartedAt:       startedAt,
		ActiveStepAt:    runtime.ActiveStepAt.UTC(),
		WaitingAt:       runtime.WaitingAt.UTC(),
		PausedAt:        runtime.PausedAt.UTC(),
		AbortedAt:       runtime.AbortedAt.UTC(),
		CompletedAt:     runtime.CompletedAt.UTC(),
		FailedAt:        runtime.FailedAt.UTC(),
	}
}

func projectStepRuntimeRecords(runtime JobRuntimeState, control *RuntimeControlContext, plan *InspectablePlanContext, nextSeq uint64) ([]StepRuntimeRecord, error) {
	catalog := projectedStepCatalog(plan, control)
	emitted := make(map[string]struct{})
	records := make([]StepRuntimeRecord, 0, len(catalog))

	completedByID := make(map[string]RuntimeStepRecord, len(runtime.CompletedSteps))
	for _, completed := range runtime.CompletedSteps {
		completedByID[completed.StepID] = cloneRuntimeStepRecord(completed)
	}
	failedByID := make(map[string]RuntimeStepRecord, len(runtime.FailedSteps))
	for _, failed := range runtime.FailedSteps {
		failedByID[failed.StepID] = cloneRuntimeStepRecord(failed)
	}

	planOrder := projectedStepOrder(plan)
	for _, stepID := range planOrder {
		step, ok := catalog[stepID]
		if !ok {
			continue
		}
		if completed, ok := completedByID[stepID]; ok {
			records = append(records, newCompletedStepRuntimeRecord(runtime.JobID, nextSeq, step, completed))
			emitted[stepID] = struct{}{}
			continue
		}
		if failed, ok := failedByID[stepID]; ok {
			records = append(records, newFailedStepRuntimeRecord(runtime.JobID, nextSeq, step, failed))
			emitted[stepID] = struct{}{}
			continue
		}
		if runtime.ActiveStepID == stepID {
			records = append(records, newActiveStepRuntimeRecord(runtime.JobID, nextSeq, step, runtime))
			emitted[stepID] = struct{}{}
			continue
		}
		records = append(records, newPendingStepRuntimeRecord(runtime.JobID, nextSeq, step))
		emitted[stepID] = struct{}{}
	}

	extraStepIDs := make([]string, 0, len(runtime.CompletedSteps)+len(runtime.FailedSteps)+1)
	for stepID := range completedByID {
		if _, ok := emitted[stepID]; !ok {
			extraStepIDs = append(extraStepIDs, stepID)
		}
	}
	for stepID := range failedByID {
		if _, ok := emitted[stepID]; !ok {
			extraStepIDs = append(extraStepIDs, stepID)
		}
	}
	if runtime.ActiveStepID != "" {
		if _, ok := emitted[runtime.ActiveStepID]; !ok {
			extraStepIDs = append(extraStepIDs, runtime.ActiveStepID)
		}
	}
	sort.Strings(extraStepIDs)
	for _, stepID := range extraStepIDs {
		if _, ok := emitted[stepID]; ok {
			continue
		}
		step, err := resolveProjectedStep(stepID, catalog, control, completedByID[stepID], failedByID[stepID])
		if err != nil {
			return nil, err
		}
		if completed, ok := completedByID[stepID]; ok {
			records = append(records, newCompletedStepRuntimeRecord(runtime.JobID, nextSeq, step, completed))
			continue
		}
		if failed, ok := failedByID[stepID]; ok {
			records = append(records, newFailedStepRuntimeRecord(runtime.JobID, nextSeq, step, failed))
			continue
		}
		records = append(records, newActiveStepRuntimeRecord(runtime.JobID, nextSeq, step, runtime))
	}

	return records, nil
}

func projectedStepCatalog(plan *InspectablePlanContext, control *RuntimeControlContext) map[string]Step {
	size := 0
	if plan != nil {
		size += len(plan.Steps)
	}
	if control != nil && control.Step.ID != "" {
		size++
	}
	catalog := make(map[string]Step, size)
	if plan != nil {
		for _, step := range plan.Steps {
			catalog[step.ID] = copyStep(step)
		}
	}
	if control != nil && control.Step.ID != "" {
		catalog[control.Step.ID] = copyStep(control.Step)
	}
	return catalog
}

func projectedStepOrder(plan *InspectablePlanContext) []string {
	if plan == nil || len(plan.Steps) == 0 {
		return nil
	}
	order := make([]string, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		order = append(order, step.ID)
	}
	return order
}

func resolveProjectedStep(stepID string, catalog map[string]Step, control *RuntimeControlContext, completed RuntimeStepRecord, failed RuntimeStepRecord) (Step, error) {
	if step, ok := catalog[stepID]; ok {
		return copyStep(step), nil
	}
	if control != nil && control.Step.ID == stepID {
		return copyStep(control.Step), nil
	}
	if completed.ResultingState != nil && isValidStepType(StepType(completed.ResultingState.Kind)) {
		return Step{ID: stepID, Type: StepType(completed.ResultingState.Kind)}, nil
	}
	return Step{}, fmt.Errorf("mission store projected runtime step %q is missing inspectable plan metadata", stepID)
}

func newCompletedStepRuntimeRecord(jobID string, nextSeq uint64, step Step, completed RuntimeStepRecord) StepRuntimeRecord {
	return StepRuntimeRecord{
		RecordVersion:     StoreRecordVersion,
		LastSeq:           nextSeq,
		JobID:             jobID,
		StepID:            step.ID,
		StepType:          step.Type,
		Status:            StepRuntimeStatusCompleted,
		DependsOn:         append([]string(nil), step.DependsOn...),
		RequiredAuthority: step.RequiredAuthority,
		RequiresApproval:  step.RequiresApproval,
		CompletedAt:       completed.At.UTC(),
		ResultingState:    cloneRuntimeResultingStateRecord(completed.ResultingState),
		Rollback:          cloneRuntimeRollbackRecord(completed.Rollback),
	}
}

func newFailedStepRuntimeRecord(jobID string, nextSeq uint64, step Step, failed RuntimeStepRecord) StepRuntimeRecord {
	return StepRuntimeRecord{
		RecordVersion:     StoreRecordVersion,
		LastSeq:           nextSeq,
		JobID:             jobID,
		StepID:            step.ID,
		StepType:          step.Type,
		Status:            StepRuntimeStatusFailed,
		DependsOn:         append([]string(nil), step.DependsOn...),
		RequiredAuthority: step.RequiredAuthority,
		RequiresApproval:  step.RequiresApproval,
		FailedAt:          failed.At.UTC(),
		Reason:            failed.Reason,
	}
}

func newActiveStepRuntimeRecord(jobID string, nextSeq uint64, step Step, runtime JobRuntimeState) StepRuntimeRecord {
	return StepRuntimeRecord{
		RecordVersion:     StoreRecordVersion,
		LastSeq:           nextSeq,
		JobID:             jobID,
		StepID:            step.ID,
		StepType:          step.Type,
		Status:            StepRuntimeStatusActive,
		DependsOn:         append([]string(nil), step.DependsOn...),
		RequiredAuthority: step.RequiredAuthority,
		RequiresApproval:  step.RequiresApproval,
		ActivatedAt:       firstProjectedTime(runtime.ActiveStepAt, runtime.UpdatedAt, runtime.StartedAt),
	}
}

func newPendingStepRuntimeRecord(jobID string, nextSeq uint64, step Step) StepRuntimeRecord {
	return StepRuntimeRecord{
		RecordVersion:     StoreRecordVersion,
		LastSeq:           nextSeq,
		JobID:             jobID,
		StepID:            step.ID,
		StepType:          step.Type,
		Status:            StepRuntimeStatusPending,
		DependsOn:         append([]string(nil), step.DependsOn...),
		RequiredAuthority: step.RequiredAuthority,
		RequiresApproval:  step.RequiresApproval,
	}
}

func projectRuntimeControlRecord(runtime JobRuntimeState, control *RuntimeControlContext, writerEpoch, nextSeq uint64) (*RuntimeControlRecord, error) {
	if !HoldsGlobalActiveJobOccupancy(runtime.State) || runtime.ActiveStepID == "" {
		return nil, nil
	}
	if control == nil {
		return nil, fmt.Errorf("mission store projected runtime control requires active-step control for job %q step %q", runtime.JobID, runtime.ActiveStepID)
	}
	if control.Step.ID != runtime.ActiveStepID {
		return nil, fmt.Errorf("mission store projected runtime control step %q does not match runtime active step %q", control.Step.ID, runtime.ActiveStepID)
	}
	record := RuntimeControlRecord{
		RecordVersion: StoreRecordVersion,
		WriterEpoch:   writerEpoch,
		LastSeq:       nextSeq,
		JobID:         runtime.JobID,
		StepID:        control.Step.ID,
		MaxAuthority:  control.MaxAuthority,
		AllowedTools:  append([]string(nil), control.AllowedTools...),
		Step:          copyStep(control.Step),
	}
	return &record, nil
}

func projectActiveJobRecord(runtime JobRuntimeState, lock WriterLockRecord, nextSeq uint64, now time.Time) (*ActiveJobRecord, bool, error) {
	if !HoldsGlobalActiveJobOccupancy(runtime.State) || runtime.ActiveStepID == "" {
		return nil, true, nil
	}
	record, err := NewActiveJobRecord(
		lock.WriterEpoch,
		runtime.JobID,
		runtime.State,
		runtime.ActiveStepID,
		lock.LeaseHolderID,
		lock.LeaseExpiresAt,
		firstProjectedTime(runtime.UpdatedAt, runtime.ActiveStepAt, now),
		nextSeq,
	)
	if err != nil {
		return nil, false, err
	}
	return &record, false, nil
}

func projectApprovalRequestRecords(runtime JobRuntimeState, nextSeq uint64) []ApprovalRequestRecord {
	if len(runtime.ApprovalRequests) == 0 {
		return nil
	}
	records := make([]ApprovalRequestRecord, 0, len(runtime.ApprovalRequests))
	for _, request := range runtime.ApprovalRequests {
		req := cloneApprovalRequest(request)
		requestedVia := req.RequestedVia
		if strings.TrimSpace(requestedVia) == "" {
			requestedVia = ApprovalRequestedViaRuntime
		}
		requestedAt := firstProjectedTime(req.RequestedAt, req.ResolvedAt, req.SupersededAt, req.RevokedAt, runtime.UpdatedAt, time.Now().UTC())
		records = append(records, ApprovalRequestRecord{
			RecordVersion:   StoreRecordVersion,
			LastSeq:         nextSeq,
			RequestID:       projectedApprovalRequestID(req),
			BindingKey:      projectedApprovalBindingKey(req.JobID, req.StepID, req.RequestedAction, req.Scope),
			JobID:           req.JobID,
			StepID:          req.StepID,
			RequestedAction: req.RequestedAction,
			Scope:           req.Scope,
			Content:         projectedApprovalRequestContent(req.Content),
			RequestedVia:    requestedVia,
			GrantedVia:      req.GrantedVia,
			SessionChannel:  req.SessionChannel,
			SessionChatID:   req.SessionChatID,
			State:           req.State,
			Reason:          req.Reason,
			RequestedAt:     requestedAt,
			ExpiresAt:       req.ExpiresAt.UTC(),
			ResolvedAt:      req.ResolvedAt.UTC(),
			SupersededAt:    req.SupersededAt.UTC(),
			RevokedAt:       req.RevokedAt.UTC(),
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].RequestID < records[j].RequestID
	})
	return records
}

func projectApprovalGrantRecords(runtime JobRuntimeState, nextSeq uint64) []ApprovalGrantRecord {
	if len(runtime.ApprovalGrants) == 0 {
		return nil
	}
	records := make([]ApprovalGrantRecord, 0, len(runtime.ApprovalGrants))
	for _, grant := range runtime.ApprovalGrants {
		g := grant
		grantedVia := g.GrantedVia
		if strings.TrimSpace(grantedVia) == "" {
			grantedVia = ApprovalGrantedViaOperatorCommand
		}
		grantedAt := firstProjectedTime(g.GrantedAt, g.RevokedAt, runtime.UpdatedAt, time.Now().UTC())
		records = append(records, ApprovalGrantRecord{
			RecordVersion:   StoreRecordVersion,
			LastSeq:         nextSeq,
			GrantID:         projectedApprovalGrantID(g),
			RequestID:       projectedApprovalBindingRecordID(g.JobID, g.StepID, g.RequestedAction, g.Scope),
			JobID:           g.JobID,
			StepID:          g.StepID,
			RequestedAction: g.RequestedAction,
			Scope:           g.Scope,
			GrantedVia:      grantedVia,
			SessionChannel:  g.SessionChannel,
			SessionChatID:   g.SessionChatID,
			State:           g.State,
			GrantedAt:       grantedAt,
			ExpiresAt:       g.ExpiresAt.UTC(),
			RevokedAt:       g.RevokedAt.UTC(),
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].GrantID < records[j].GrantID
	})
	return records
}

func projectAuditEventRecords(runtime JobRuntimeState, nextSeq uint64, committed map[string]struct{}) []AuditEventRecord {
	if len(runtime.AuditHistory) == 0 {
		return nil
	}
	records := make([]AuditEventRecord, 0, len(runtime.AuditHistory))
	for _, raw := range runtime.AuditHistory {
		event := normalizeAuditEvent(raw)
		if _, ok := committed[event.EventID]; ok {
			continue
		}
		committed[event.EventID] = struct{}{}
		records = append(records, AuditEventRecord{
			RecordVersion: StoreRecordVersion,
			Seq:           nextSeq,
			Event:         event,
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		left := normalizeAuditEvent(records[i].Event)
		right := normalizeAuditEvent(records[j].Event)
		if !left.Timestamp.Equal(right.Timestamp) {
			return left.Timestamp.Before(right.Timestamp)
		}
		return left.EventID < right.EventID
	})
	return records
}

func projectArtifactRecords(runtime JobRuntimeState, plan *InspectablePlanContext, nextSeq uint64) []ArtifactRecord {
	candidates := collectOperatorStatusArtifactCandidates(runtime)
	if len(candidates) == 0 {
		return nil
	}
	stepByID := make(map[string]Step, len(candidates))
	if plan != nil {
		for _, step := range plan.Steps {
			stepByID[step.ID] = copyStep(step)
		}
	}
	records := make([]ArtifactRecord, 0, len(candidates))
	for _, candidate := range candidates {
		format := ""
		state := strings.TrimSpace(candidate.State)
		if state == "" {
			state = "verified"
		}
		if step, ok := stepByID[candidate.StepID]; ok && step.Type == StepTypeStaticArtifact {
			format = step.StaticArtifactFormat
		}
		records = append(records, ArtifactRecord{
			RecordVersion: StoreRecordVersion,
			LastSeq:       nextSeq,
			ArtifactID:    projectedArtifactID(candidate.StepType, candidate.StepID, candidate.Path),
			JobID:         runtime.JobID,
			StepID:        candidate.StepID,
			StepType:      candidate.StepType,
			Path:          candidate.Path,
			Format:        format,
			State:         state,
			SourceStepID:  candidate.StepID,
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].ArtifactID < records[j].ArtifactID
	})
	return records
}

func projectCampaignZohoEmailOutboundActionRecords(runtime JobRuntimeState, nextSeq uint64) []CampaignZohoEmailOutboundActionRecord {
	if len(runtime.CampaignZohoEmailOutboundActions) == 0 {
		return nil
	}
	records := make([]CampaignZohoEmailOutboundActionRecord, 0, len(runtime.CampaignZohoEmailOutboundActions))
	for _, action := range runtime.CampaignZohoEmailOutboundActions {
		normalized := NormalizeCampaignZohoEmailOutboundAction(action)
		records = append(records, CampaignZohoEmailOutboundActionRecord{
			RecordVersion:           StoreRecordVersion,
			LastSeq:                 nextSeq,
			ActionID:                normalized.ActionID,
			JobID:                   runtime.JobID,
			StepID:                  normalized.StepID,
			CampaignID:              normalized.CampaignID,
			State:                   string(normalized.State),
			Provider:                normalized.Provider,
			ProviderAccountID:       normalized.ProviderAccountID,
			FromAddress:             normalized.FromAddress,
			FromDisplayName:         normalized.FromDisplayName,
			Addressing:              normalized.Addressing,
			Subject:                 normalized.Subject,
			BodyFormat:              normalized.BodyFormat,
			BodySHA256:              normalized.BodySHA256,
			PreparedAt:              normalized.PreparedAt,
			SentAt:                  normalized.SentAt,
			VerifiedAt:              normalized.VerifiedAt,
			FailedAt:                normalized.FailedAt,
			ReplyToInboundReplyID:   normalized.ReplyToInboundReplyID,
			ReplyToOutboundActionID: normalized.ReplyToOutboundActionID,
			ProviderMessageID:       normalized.ProviderMessageID,
			ProviderMailID:          normalized.ProviderMailID,
			MIMEMessageID:           normalized.MIMEMessageID,
			OriginalMessageURL:      normalized.OriginalMessageURL,
			Failure:                 normalized.Failure,
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].ActionID < records[j].ActionID
	})
	return records
}

func projectFrankZohoSendReceiptRecords(runtime JobRuntimeState, nextSeq uint64) []FrankZohoSendReceiptRecord {
	if len(runtime.FrankZohoSendReceipts) == 0 {
		return nil
	}
	records := make([]FrankZohoSendReceiptRecord, 0, len(runtime.FrankZohoSendReceipts))
	for _, receipt := range runtime.FrankZohoSendReceipts {
		normalized := NormalizeFrankZohoSendReceipt(receipt)
		records = append(records, FrankZohoSendReceiptRecord{
			RecordVersion:      StoreRecordVersion,
			LastSeq:            nextSeq,
			ReceiptID:          projectedFrankZohoSendReceiptID(runtime.JobID, normalized),
			JobID:              runtime.JobID,
			StepID:             normalized.StepID,
			Provider:           normalized.Provider,
			ProviderAccountID:  normalized.ProviderAccountID,
			FromAddress:        normalized.FromAddress,
			FromDisplayName:    normalized.FromDisplayName,
			ProviderMessageID:  normalized.ProviderMessageID,
			ProviderMailID:     normalized.ProviderMailID,
			MIMEMessageID:      normalized.MIMEMessageID,
			OriginalMessageURL: normalized.OriginalMessageURL,
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].ReceiptID < records[j].ReceiptID
	})
	return records
}

func projectCampaignZohoEmailReplyWorkItemRecords(runtime JobRuntimeState, nextSeq uint64) []CampaignZohoEmailReplyWorkItemRecord {
	if len(runtime.CampaignZohoEmailReplyWorkItems) == 0 {
		return nil
	}
	records := make([]CampaignZohoEmailReplyWorkItemRecord, 0, len(runtime.CampaignZohoEmailReplyWorkItems))
	for _, item := range runtime.CampaignZohoEmailReplyWorkItems {
		normalized := NormalizeCampaignZohoEmailReplyWorkItem(item)
		records = append(records, CampaignZohoEmailReplyWorkItemRecord{
			RecordVersion:           StoreRecordVersion,
			LastSeq:                 nextSeq,
			ReplyWorkItemID:         normalized.ReplyWorkItemID,
			JobID:                   runtime.JobID,
			StepID:                  runtime.ActiveStepID,
			InboundReplyID:          normalized.InboundReplyID,
			CampaignID:              normalized.CampaignID,
			State:                   string(normalized.State),
			DeferredUntil:           normalized.DeferredUntil,
			ClaimedFollowUpActionID: normalized.ClaimedFollowUpActionID,
			CreatedAt:               normalized.CreatedAt,
			UpdatedAt:               normalized.UpdatedAt,
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].CreatedAt.Before(records[j].CreatedAt)
		}
		return records[i].ReplyWorkItemID < records[j].ReplyWorkItemID
	})
	return records
}

func projectFrankZohoInboundReplyRecords(runtime JobRuntimeState, nextSeq uint64) []FrankZohoInboundReplyRecord {
	if len(runtime.FrankZohoInboundReplies) == 0 {
		return nil
	}
	records := make([]FrankZohoInboundReplyRecord, 0, len(runtime.FrankZohoInboundReplies))
	for _, reply := range runtime.FrankZohoInboundReplies {
		normalized := NormalizeFrankZohoInboundReply(reply)
		records = append(records, FrankZohoInboundReplyRecord{
			RecordVersion:      StoreRecordVersion,
			LastSeq:            nextSeq,
			ReplyID:            normalized.ReplyID,
			JobID:              runtime.JobID,
			StepID:             normalized.StepID,
			Provider:           normalized.Provider,
			ProviderAccountID:  normalized.ProviderAccountID,
			ProviderMessageID:  normalized.ProviderMessageID,
			ProviderMailID:     normalized.ProviderMailID,
			MIMEMessageID:      normalized.MIMEMessageID,
			InReplyTo:          normalized.InReplyTo,
			References:         append([]string(nil), normalized.References...),
			FromAddress:        normalized.FromAddress,
			FromDisplayName:    normalized.FromDisplayName,
			FromAddressCount:   normalized.FromAddressCount,
			Subject:            normalized.Subject,
			ReceivedAt:         normalized.ReceivedAt,
			OriginalMessageURL: normalized.OriginalMessageURL,
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].ReceivedAt.Equal(records[j].ReceivedAt) {
			return records[i].ReceivedAt.Before(records[j].ReceivedAt)
		}
		return records[i].ReplyID < records[j].ReplyID
	})
	return records
}

func projectFrankZohoBounceEvidenceRecords(runtime JobRuntimeState, nextSeq uint64) []FrankZohoBounceEvidenceRecord {
	if len(runtime.FrankZohoBounceEvidence) == 0 {
		return nil
	}
	records := make([]FrankZohoBounceEvidenceRecord, 0, len(runtime.FrankZohoBounceEvidence))
	for _, evidence := range runtime.FrankZohoBounceEvidence {
		normalized := NormalizeFrankZohoBounceEvidence(evidence)
		records = append(records, FrankZohoBounceEvidenceRecord{
			RecordVersion:             StoreRecordVersion,
			LastSeq:                   nextSeq,
			BounceID:                  normalized.BounceID,
			JobID:                     runtime.JobID,
			StepID:                    normalized.StepID,
			Provider:                  normalized.Provider,
			ProviderAccountID:         normalized.ProviderAccountID,
			ProviderMessageID:         normalized.ProviderMessageID,
			ProviderMailID:            normalized.ProviderMailID,
			MIMEMessageID:             normalized.MIMEMessageID,
			InReplyTo:                 normalized.InReplyTo,
			References:                append([]string(nil), normalized.References...),
			OriginalProviderMessageID: normalized.OriginalProviderMessageID,
			OriginalProviderMailID:    normalized.OriginalProviderMailID,
			OriginalMIMEMessageID:     normalized.OriginalMIMEMessageID,
			FinalRecipient:            normalized.FinalRecipient,
			DiagnosticCode:            normalized.DiagnosticCode,
			ReceivedAt:                normalized.ReceivedAt,
			OriginalMessageURL:        normalized.OriginalMessageURL,
			CampaignID:                normalized.CampaignID,
			OutboundActionID:          normalized.OutboundActionID,
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].ReceivedAt.Equal(records[j].ReceivedAt) {
			return records[i].ReceivedAt.Before(records[j].ReceivedAt)
		}
		return records[i].BounceID < records[j].BounceID
	})
	return records
}

func projectedApprovalRequestContent(content *ApprovalRequestContent) *ApprovalRequestContent {
	if content == nil {
		return nil
	}
	cloned := *content
	return &cloned
}

func projectedApprovalRequestID(request ApprovalRequest) string {
	content := request.Content
	contentFields := []string{"", "", "", "", "", "", "", "", ""}
	if content != nil {
		contentFields = []string{
			content.ProposedAction,
			content.WhyNeeded,
			string(content.AuthorityTier),
			content.IdentityScope,
			content.PublicScope,
			content.FilesystemEffect,
			content.ProcessEffect,
			content.NetworkEffect,
			content.FallbackIfDenied,
		}
	}
	return "req_" + projectedStoreHash(
		request.JobID,
		request.StepID,
		request.RequestedAction,
		normalizeApprovalScope(request.Scope),
		request.RequestedVia,
		request.RequestedAt.UTC().Format(time.RFC3339Nano),
		contentFields[0],
		contentFields[1],
		contentFields[2],
		contentFields[3],
		contentFields[4],
		contentFields[5],
		contentFields[6],
		contentFields[7],
		contentFields[8],
	)
}

func projectedApprovalGrantID(grant ApprovalGrant) string {
	return "grant_" + projectedStoreHash(
		grant.JobID,
		grant.StepID,
		grant.RequestedAction,
		normalizeApprovalScope(grant.Scope),
		grant.GrantedVia,
		grant.SessionChannel,
		grant.SessionChatID,
		grant.GrantedAt.UTC().Format(time.RFC3339Nano),
	)
}

func projectedApprovalBindingRecordID(jobID, stepID, requestedAction, scope string) string {
	return "binding_" + projectedStoreHash(jobID, stepID, requestedAction, normalizeApprovalScope(scope))
}

func projectedApprovalBindingKey(jobID, stepID, requestedAction, scope string) string {
	return strings.Join([]string{jobID, stepID, requestedAction, normalizeApprovalScope(scope)}, "\x1f")
}

func projectedArtifactID(stepType StepType, stepID, path string) string {
	return "artifact_" + projectedStoreHash(string(stepType), stepID, cleanedArtifactPath(path))
}

func projectedFrankZohoSendReceiptID(jobID string, receipt FrankZohoSendReceipt) string {
	normalized := NormalizeFrankZohoSendReceipt(receipt)
	return "zoho_send_" + projectedStoreHash(
		jobID,
		normalized.StepID,
		normalized.Provider,
		normalized.ProviderAccountID,
		normalized.ProviderMessageID,
	)
}

func projectedStoreHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:16])
}

func firstProjectedTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Time{}
}
