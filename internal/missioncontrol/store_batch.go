package missioncontrol

import (
	"errors"
	"fmt"
	"time"
)

var storeBatchBeforeMutation = func(string) error { return nil }
var storeBatchNewAttemptID = func() string {
	return fmt.Sprintf("attempt-%020d", time.Now().UTC().UnixNano())
}

type StoreBatch struct {
	JobRuntime                       JobRuntimeRecord
	RuntimeControl                   *RuntimeControlRecord
	StepRecords                      []StepRuntimeRecord
	ApprovalRequests                 []ApprovalRequestRecord
	ApprovalGrants                   []ApprovalGrantRecord
	AuditEvents                      []AuditEventRecord
	Artifacts                        []ArtifactRecord
	CampaignZohoEmailOutboundActions []CampaignZohoEmailOutboundActionRecord
	FrankZohoSendReceipts            []FrankZohoSendReceiptRecord
	// active_job.json remains a fixed-path arbitration record and must be
	// reconciled against committed job_runtime.applied_seq during recovery.
	ActiveJob       *ActiveJobRecord
	RemoveActiveJob bool
}

func CommitStoreBatch(root string, heldLock WriterLockRecord, batch StoreBatch) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateStoreBatch(batch, heldLock); err != nil {
		return err
	}

	guard, err := acquireStoreWriterGuard(root)
	if err != nil {
		return err
	}
	defer func() { _ = guard.release() }()

	if err := ValidateHeldWriterLock(root, heldLock, time.Now().UTC()); err != nil {
		return err
	}
	if err := validateNextCommittedJobRuntime(root, batch.JobRuntime); err != nil {
		return err
	}
	attemptID := storeBatchNewAttemptID()

	for _, record := range batch.StepRecords {
		record.AttemptID = attemptID
		if err := storeBatchWriteRecord(storeStepRuntimeVersionPath(root, record.JobID, record.StepID, record.LastSeq, record.AttemptID), record); err != nil {
			return err
		}
	}
	for _, record := range batch.ApprovalRequests {
		record.AttemptID = attemptID
		if err := storeBatchWriteRecord(storeApprovalRequestVersionPath(root, record.JobID, record.RequestID, record.LastSeq, record.AttemptID), record); err != nil {
			return err
		}
	}
	for _, record := range batch.ApprovalGrants {
		record.AttemptID = attemptID
		if err := storeBatchWriteRecord(storeApprovalGrantVersionPath(root, record.JobID, record.GrantID, record.LastSeq, record.AttemptID), record); err != nil {
			return err
		}
	}
	for _, record := range batch.AuditEvents {
		record.AttemptID = attemptID
		record.Event = normalizeAuditEvent(record.Event)
		if err := storeBatchWriteRecord(storeAuditEventVersionPath(root, record.Event.JobID, record.Seq, record.AttemptID, record.Event.EventID), record); err != nil {
			return err
		}
	}
	for _, record := range batch.Artifacts {
		record.AttemptID = attemptID
		if err := storeBatchWriteRecord(storeArtifactVersionPath(root, record.JobID, record.ArtifactID, record.LastSeq, record.AttemptID), record); err != nil {
			return err
		}
	}
	for _, record := range batch.CampaignZohoEmailOutboundActions {
		record.AttemptID = attemptID
		if err := storeBatchWriteRecord(storeCampaignZohoEmailOutboundActionVersionPath(root, record.JobID, record.ActionID, record.LastSeq, record.AttemptID), record); err != nil {
			return err
		}
	}
	for _, record := range batch.FrankZohoSendReceipts {
		record.AttemptID = attemptID
		if err := storeBatchWriteRecord(storeFrankZohoSendReceiptVersionPath(root, record.JobID, record.ReceiptID, record.LastSeq, record.AttemptID), record); err != nil {
			return err
		}
	}
	if batch.RuntimeControl != nil {
		record := *batch.RuntimeControl
		record.AttemptID = attemptID
		if err := storeBatchWriteRecord(storeRuntimeControlVersionPath(root, record.JobID, record.LastSeq, record.AttemptID), record); err != nil {
			return err
		}
	}
	switch {
	case batch.ActiveJob != nil:
		record := *batch.ActiveJob
		record.AttemptID = attemptID
		if err := storeBatchWriteRecord(StoreActiveJobPath(root), record); err != nil {
			return err
		}
	case batch.RemoveActiveJob:
		if err := storeBatchRemoveActiveJob(root); err != nil {
			return err
		}
	}
	if err := storeBatchWriteRecord(storeBatchCommitPath(root, batch.JobRuntime.JobID, batch.JobRuntime.AppliedSeq), BatchCommitRecord{
		RecordVersion: StoreRecordVersion,
		JobID:         batch.JobRuntime.JobID,
		Seq:           batch.JobRuntime.AppliedSeq,
		AttemptID:     attemptID,
		CommittedAt:   time.Now().UTC(),
	}); err != nil {
		return err
	}

	return storeBatchWriteRecord(StoreJobRuntimePath(root, batch.JobRuntime.JobID), batch.JobRuntime)
}

func ValidateStoreBatch(batch StoreBatch, heldLock WriterLockRecord) error {
	if err := ValidateWriterLockRecord(heldLock); err != nil {
		return err
	}
	if err := ValidateJobRuntimeRecord(batch.JobRuntime); err != nil {
		return err
	}
	if batch.JobRuntime.WriterEpoch != heldLock.WriterEpoch {
		return fmt.Errorf("mission store batch job runtime writer_epoch %d does not match held writer epoch %d", batch.JobRuntime.WriterEpoch, heldLock.WriterEpoch)
	}
	targetSeq := batch.JobRuntime.AppliedSeq
	jobID := batch.JobRuntime.JobID

	if batch.RuntimeControl != nil {
		if err := ValidateRuntimeControlRecord(*batch.RuntimeControl); err != nil {
			return err
		}
		if batch.RuntimeControl.JobID != jobID {
			return fmt.Errorf("mission store batch runtime control job_id %q does not match job runtime %q", batch.RuntimeControl.JobID, jobID)
		}
		if batch.RuntimeControl.LastSeq != targetSeq {
			return fmt.Errorf("mission store batch runtime control last_seq %d does not match applied_seq %d", batch.RuntimeControl.LastSeq, targetSeq)
		}
		if batch.RuntimeControl.WriterEpoch != heldLock.WriterEpoch {
			return fmt.Errorf("mission store batch runtime control writer_epoch %d does not match held writer epoch %d", batch.RuntimeControl.WriterEpoch, heldLock.WriterEpoch)
		}
	}
	for _, record := range batch.StepRecords {
		if err := ValidateStepRuntimeRecord(record); err != nil {
			return err
		}
		if record.JobID != jobID {
			return fmt.Errorf("mission store batch step record job_id %q does not match job runtime %q", record.JobID, jobID)
		}
		if record.LastSeq != targetSeq {
			return fmt.Errorf("mission store batch step record %q last_seq %d does not match applied_seq %d", record.StepID, record.LastSeq, targetSeq)
		}
	}
	for _, record := range batch.ApprovalRequests {
		if err := ValidateApprovalRequestRecord(record); err != nil {
			return err
		}
		if record.JobID != jobID {
			return fmt.Errorf("mission store batch approval request %q job_id %q does not match job runtime %q", record.RequestID, record.JobID, jobID)
		}
		if record.LastSeq != targetSeq {
			return fmt.Errorf("mission store batch approval request %q last_seq %d does not match applied_seq %d", record.RequestID, record.LastSeq, targetSeq)
		}
	}
	for _, record := range batch.ApprovalGrants {
		if err := ValidateApprovalGrantRecord(record); err != nil {
			return err
		}
		if record.JobID != jobID {
			return fmt.Errorf("mission store batch approval grant %q job_id %q does not match job runtime %q", record.GrantID, record.JobID, jobID)
		}
		if record.LastSeq != targetSeq {
			return fmt.Errorf("mission store batch approval grant %q last_seq %d does not match applied_seq %d", record.GrantID, record.LastSeq, targetSeq)
		}
	}
	for _, record := range batch.AuditEvents {
		record.Event = normalizeAuditEvent(record.Event)
		if err := ValidateAuditEventRecord(record); err != nil {
			return err
		}
		if record.Event.JobID != jobID {
			return fmt.Errorf("mission store batch audit event %q job_id %q does not match job runtime %q", record.Event.EventID, record.Event.JobID, jobID)
		}
		if record.Seq != targetSeq {
			return fmt.Errorf("mission store batch audit event %q seq %d does not match applied_seq %d", record.Event.EventID, record.Seq, targetSeq)
		}
	}
	for _, record := range batch.Artifacts {
		if err := ValidateArtifactRecord(record); err != nil {
			return err
		}
		if record.JobID != jobID {
			return fmt.Errorf("mission store batch artifact %q job_id %q does not match job runtime %q", record.ArtifactID, record.JobID, jobID)
		}
		if record.LastSeq != targetSeq {
			return fmt.Errorf("mission store batch artifact %q last_seq %d does not match applied_seq %d", record.ArtifactID, record.LastSeq, targetSeq)
		}
	}
	for _, record := range batch.CampaignZohoEmailOutboundActions {
		if err := ValidateCampaignZohoEmailOutboundActionRecord(record); err != nil {
			return err
		}
		if record.JobID != jobID {
			return fmt.Errorf("mission store batch campaign zoho email outbound action %q job_id %q does not match job runtime %q", record.ActionID, record.JobID, jobID)
		}
		if record.LastSeq != targetSeq {
			return fmt.Errorf("mission store batch campaign zoho email outbound action %q last_seq %d does not match applied_seq %d", record.ActionID, record.LastSeq, targetSeq)
		}
	}
	for _, record := range batch.FrankZohoSendReceipts {
		if err := ValidateFrankZohoSendReceiptRecord(record); err != nil {
			return err
		}
		if record.JobID != jobID {
			return fmt.Errorf("mission store batch frank zoho send receipt %q job_id %q does not match job runtime %q", record.ReceiptID, record.JobID, jobID)
		}
		if record.LastSeq != targetSeq {
			return fmt.Errorf("mission store batch frank zoho send receipt %q last_seq %d does not match applied_seq %d", record.ReceiptID, record.LastSeq, targetSeq)
		}
	}
	if batch.ActiveJob != nil && batch.RemoveActiveJob {
		return fmt.Errorf("mission store batch cannot update and remove active_job in the same commit")
	}
	if batch.ActiveJob != nil {
		if err := ValidateActiveJobRecord(*batch.ActiveJob); err != nil {
			return err
		}
		if batch.ActiveJob.JobID != jobID {
			return fmt.Errorf("mission store batch active job %q does not match job runtime %q", batch.ActiveJob.JobID, jobID)
		}
		if batch.ActiveJob.WriterEpoch != heldLock.WriterEpoch {
			return fmt.Errorf("mission store batch active job writer_epoch %d does not match held writer epoch %d", batch.ActiveJob.WriterEpoch, heldLock.WriterEpoch)
		}
		if batch.ActiveJob.ActivationSeq != targetSeq {
			return fmt.Errorf("mission store batch active job activation_seq %d does not match applied_seq %d", batch.ActiveJob.ActivationSeq, targetSeq)
		}
		if !HoldsGlobalActiveJobOccupancy(batch.JobRuntime.State) {
			return fmt.Errorf("mission store batch active job requires occupancy state, got %q", batch.JobRuntime.State)
		}
	}
	return nil
}

func ValidateHeldWriterLock(root string, heldLock WriterLockRecord, now time.Time) error {
	if err := ValidateWriterEpochCoherence(root); err != nil {
		return err
	}
	storedLock, err := LoadWriterLock(root)
	if err != nil {
		return err
	}
	if storedLock.LeaseHolderID != heldLock.LeaseHolderID || storedLock.WriterEpoch != heldLock.WriterEpoch {
		return ErrWriterLockHeld
	}
	if !storedLock.LeaseExpiresAt.After(now.UTC()) {
		return ErrWriterLockExpired
	}
	return nil
}

func storeBatchWriteRecord(path string, value any) error {
	if err := storeBatchBeforeMutation(path); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(path, value)
}

func storeBatchRemoveActiveJob(root string) error {
	path := StoreActiveJobPath(root)
	if err := storeBatchBeforeMutation(path); err != nil {
		return err
	}
	return RemoveActiveJobRecord(root)
}

func validateNextCommittedJobRuntime(root string, jobRuntime JobRuntimeRecord) error {
	previous, err := LoadCommittedJobRuntimeRecord(root, jobRuntime.JobID)
	switch {
	case err == nil:
		if jobRuntime.AppliedSeq != previous.AppliedSeq+1 {
			return fmt.Errorf("mission store batch applied_seq %d must advance from committed seq %d", jobRuntime.AppliedSeq, previous.AppliedSeq)
		}
		return nil
	case errors.Is(err, ErrJobRuntimeRecordNotFound):
		return nil
	default:
		return err
	}
}
