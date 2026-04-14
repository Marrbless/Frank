package missioncontrol

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type StepRuntimeStatus string

const (
	StepRuntimeStatusPending   StepRuntimeStatus = "pending"
	StepRuntimeStatusActive    StepRuntimeStatus = "active"
	StepRuntimeStatusCompleted StepRuntimeStatus = "completed"
	StepRuntimeStatusFailed    StepRuntimeStatus = "failed"
)

type JobRuntimeRecord struct {
	RecordVersion   int                         `json:"record_version"`
	WriterEpoch     uint64                      `json:"writer_epoch"`
	AppliedSeq      uint64                      `json:"applied_seq"`
	JobID           string                      `json:"job_id"`
	State           JobState                    `json:"state"`
	ActiveStepID    string                      `json:"active_step_id,omitempty"`
	InspectablePlan *InspectablePlanContext     `json:"inspectable_plan,omitempty"`
	BudgetBlocker   *RuntimeBudgetBlockerRecord `json:"budget_blocker,omitempty"`
	WaitingReason   string                      `json:"waiting_reason,omitempty"`
	PausedReason    string                      `json:"paused_reason,omitempty"`
	AbortedReason   string                      `json:"aborted_reason,omitempty"`
	CreatedAt       time.Time                   `json:"created_at,omitempty"`
	UpdatedAt       time.Time                   `json:"updated_at,omitempty"`
	StartedAt       time.Time                   `json:"started_at,omitempty"`
	ActiveStepAt    time.Time                   `json:"active_step_at,omitempty"`
	WaitingAt       time.Time                   `json:"waiting_at,omitempty"`
	PausedAt        time.Time                   `json:"paused_at,omitempty"`
	AbortedAt       time.Time                   `json:"aborted_at,omitempty"`
	CompletedAt     time.Time                   `json:"completed_at,omitempty"`
	FailedAt        time.Time                   `json:"failed_at,omitempty"`
}

type StepRuntimeRecord struct {
	RecordVersion     int                          `json:"record_version"`
	LastSeq           uint64                       `json:"last_seq"`
	JobID             string                       `json:"job_id"`
	StepID            string                       `json:"step_id"`
	AttemptID         string                       `json:"attempt_id,omitempty"`
	StepType          StepType                     `json:"step_type"`
	Status            StepRuntimeStatus            `json:"status"`
	DependsOn         []string                     `json:"depends_on,omitempty"`
	RequiredAuthority AuthorityTier                `json:"required_authority,omitempty"`
	RequiresApproval  bool                         `json:"requires_approval,omitempty"`
	ActivatedAt       time.Time                    `json:"activated_at,omitempty"`
	CompletedAt       time.Time                    `json:"completed_at,omitempty"`
	FailedAt          time.Time                    `json:"failed_at,omitempty"`
	Reason            string                       `json:"reason,omitempty"`
	ResultingState    *RuntimeResultingStateRecord `json:"resulting_state,omitempty"`
	Rollback          *RuntimeRollbackRecord       `json:"rollback,omitempty"`
}

type RuntimeControlRecord struct {
	RecordVersion int           `json:"record_version"`
	WriterEpoch   uint64        `json:"writer_epoch"`
	LastSeq       uint64        `json:"last_seq"`
	JobID         string        `json:"job_id"`
	StepID        string        `json:"step_id"`
	AttemptID     string        `json:"attempt_id,omitempty"`
	MaxAuthority  AuthorityTier `json:"max_authority"`
	AllowedTools  []string      `json:"allowed_tools,omitempty"`
	Step          Step          `json:"step"`
}

type ApprovalRequestRecord struct {
	RecordVersion   int                     `json:"record_version"`
	LastSeq         uint64                  `json:"last_seq"`
	RequestID       string                  `json:"request_id"`
	BindingKey      string                  `json:"binding_key,omitempty"`
	JobID           string                  `json:"job_id"`
	StepID          string                  `json:"step_id"`
	AttemptID       string                  `json:"attempt_id,omitempty"`
	RequestedAction string                  `json:"requested_action"`
	Scope           string                  `json:"scope"`
	Content         *ApprovalRequestContent `json:"content,omitempty"`
	RequestedVia    string                  `json:"requested_via"`
	GrantedVia      string                  `json:"granted_via,omitempty"`
	SessionChannel  string                  `json:"session_channel,omitempty"`
	SessionChatID   string                  `json:"session_chat_id,omitempty"`
	State           ApprovalState           `json:"state"`
	Reason          string                  `json:"reason,omitempty"`
	RequestedAt     time.Time               `json:"requested_at,omitempty"`
	ExpiresAt       time.Time               `json:"expires_at,omitempty"`
	ResolvedAt      time.Time               `json:"resolved_at,omitempty"`
	SupersededAt    time.Time               `json:"superseded_at,omitempty"`
	RevokedAt       time.Time               `json:"revoked_at,omitempty"`
}

type ApprovalGrantRecord struct {
	RecordVersion   int           `json:"record_version"`
	LastSeq         uint64        `json:"last_seq"`
	GrantID         string        `json:"grant_id"`
	RequestID       string        `json:"request_id"`
	JobID           string        `json:"job_id"`
	StepID          string        `json:"step_id"`
	AttemptID       string        `json:"attempt_id,omitempty"`
	RequestedAction string        `json:"requested_action"`
	Scope           string        `json:"scope"`
	GrantedVia      string        `json:"granted_via"`
	SessionChannel  string        `json:"session_channel,omitempty"`
	SessionChatID   string        `json:"session_chat_id,omitempty"`
	State           ApprovalState `json:"state"`
	GrantedAt       time.Time     `json:"granted_at,omitempty"`
	ExpiresAt       time.Time     `json:"expires_at,omitempty"`
	RevokedAt       time.Time     `json:"revoked_at,omitempty"`
}

type AuditEventRecord struct {
	RecordVersion int        `json:"record_version"`
	Seq           uint64     `json:"seq"`
	AttemptID     string     `json:"attempt_id,omitempty"`
	Event         AuditEvent `json:"event"`
}

type ArtifactRecord struct {
	RecordVersion       int       `json:"record_version"`
	LastSeq             uint64    `json:"last_seq"`
	ArtifactID          string    `json:"artifact_id"`
	JobID               string    `json:"job_id"`
	StepID              string    `json:"step_id"`
	AttemptID           string    `json:"attempt_id,omitempty"`
	StepType            StepType  `json:"step_type"`
	Path                string    `json:"path"`
	Format              string    `json:"format,omitempty"`
	State               string    `json:"state"`
	SourceStepID        string    `json:"source_step_id,omitempty"`
	Operation           string    `json:"operation,omitempty"`
	VerifiedAt          time.Time `json:"verified_at,omitempty"`
	VerificationCommand []string  `json:"verification_command,omitempty"`
	VerificationOutput  string    `json:"verification_output,omitempty"`
}

type FrankZohoSendReceiptRecord struct {
	RecordVersion      int    `json:"record_version"`
	LastSeq            uint64 `json:"last_seq"`
	ReceiptID          string `json:"receipt_id"`
	JobID              string `json:"job_id"`
	StepID             string `json:"step_id"`
	AttemptID          string `json:"attempt_id,omitempty"`
	Provider           string `json:"provider"`
	ProviderAccountID  string `json:"provider_account_id"`
	FromAddress        string `json:"from_address,omitempty"`
	FromDisplayName    string `json:"from_display_name,omitempty"`
	ProviderMessageID  string `json:"provider_message_id"`
	ProviderMailID     string `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string `json:"mime_message_id,omitempty"`
	OriginalMessageURL string `json:"original_message_url"`
}

type BatchCommitRecord struct {
	RecordVersion int       `json:"record_version"`
	JobID         string    `json:"job_id"`
	Seq           uint64    `json:"seq"`
	AttemptID     string    `json:"attempt_id"`
	CommittedAt   time.Time `json:"committed_at"`
}

var (
	ErrJobRuntimeRecordNotFound     = errors.New("mission store job runtime record not found")
	ErrBatchCommitRecordNotFound    = errors.New("mission store batch commit record not found")
	ErrRuntimeControlRecordNotFound = errors.New("mission store runtime control record not found")
)

func ValidateJobRuntimeRecord(record JobRuntimeRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store job runtime record_version must be positive")
	}
	if record.WriterEpoch == 0 {
		return fmt.Errorf("mission store job runtime writer_epoch must be positive")
	}
	if record.AppliedSeq == 0 {
		return fmt.Errorf("mission store job runtime applied_seq must be positive")
	}
	if record.JobID == "" {
		return fmt.Errorf("mission store job runtime job_id is required")
	}
	if record.State == "" {
		return fmt.Errorf("mission store job runtime state is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store job runtime created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store job runtime updated_at is required")
	}
	return nil
}

func ValidateStepRuntimeRecord(record StepRuntimeRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store step runtime record_version must be positive")
	}
	if record.LastSeq == 0 {
		return fmt.Errorf("mission store step runtime last_seq must be positive")
	}
	if record.JobID == "" {
		return fmt.Errorf("mission store step runtime job_id is required")
	}
	if record.StepID == "" {
		return fmt.Errorf("mission store step runtime step_id is required")
	}
	if !isValidStepType(record.StepType) {
		return fmt.Errorf("mission store step runtime step_type %q is invalid", record.StepType)
	}
	switch record.Status {
	case StepRuntimeStatusPending, StepRuntimeStatusActive, StepRuntimeStatusCompleted, StepRuntimeStatusFailed:
	default:
		return fmt.Errorf("mission store step runtime status %q is invalid", record.Status)
	}
	return nil
}

func ValidateRuntimeControlRecord(record RuntimeControlRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store runtime control record_version must be positive")
	}
	if record.WriterEpoch == 0 {
		return fmt.Errorf("mission store runtime control writer_epoch must be positive")
	}
	if record.LastSeq == 0 {
		return fmt.Errorf("mission store runtime control last_seq must be positive")
	}
	if record.JobID == "" {
		return fmt.Errorf("mission store runtime control job_id is required")
	}
	if record.StepID == "" {
		return fmt.Errorf("mission store runtime control step_id is required")
	}
	if record.MaxAuthority == "" {
		return fmt.Errorf("mission store runtime control max_authority is required")
	}
	if record.Step.ID == "" {
		return fmt.Errorf("mission store runtime control step.id is required")
	}
	if record.Step.ID != record.StepID {
		return fmt.Errorf("mission store runtime control step_id %q does not match step.id %q", record.StepID, record.Step.ID)
	}
	return nil
}

func ValidateApprovalRequestRecord(record ApprovalRequestRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store approval request record_version must be positive")
	}
	if record.LastSeq == 0 {
		return fmt.Errorf("mission store approval request last_seq must be positive")
	}
	if record.RequestID == "" {
		return fmt.Errorf("mission store approval request request_id is required")
	}
	if record.JobID == "" {
		return fmt.Errorf("mission store approval request job_id is required")
	}
	if record.StepID == "" {
		return fmt.Errorf("mission store approval request step_id is required")
	}
	if record.RequestedAction == "" {
		return fmt.Errorf("mission store approval request requested_action is required")
	}
	if record.Scope == "" {
		return fmt.Errorf("mission store approval request scope is required")
	}
	if record.RequestedVia == "" {
		return fmt.Errorf("mission store approval request requested_via is required")
	}
	if record.State == "" {
		return fmt.Errorf("mission store approval request state is required")
	}
	if record.RequestedAt.IsZero() {
		return fmt.Errorf("mission store approval request requested_at is required")
	}
	return nil
}

func ValidateApprovalGrantRecord(record ApprovalGrantRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store approval grant record_version must be positive")
	}
	if record.LastSeq == 0 {
		return fmt.Errorf("mission store approval grant last_seq must be positive")
	}
	if record.GrantID == "" {
		return fmt.Errorf("mission store approval grant grant_id is required")
	}
	if record.RequestID == "" {
		return fmt.Errorf("mission store approval grant request_id is required")
	}
	if record.JobID == "" {
		return fmt.Errorf("mission store approval grant job_id is required")
	}
	if record.StepID == "" {
		return fmt.Errorf("mission store approval grant step_id is required")
	}
	if record.RequestedAction == "" {
		return fmt.Errorf("mission store approval grant requested_action is required")
	}
	if record.Scope == "" {
		return fmt.Errorf("mission store approval grant scope is required")
	}
	if record.GrantedVia == "" {
		return fmt.Errorf("mission store approval grant granted_via is required")
	}
	if record.State == "" {
		return fmt.Errorf("mission store approval grant state is required")
	}
	if record.GrantedAt.IsZero() {
		return fmt.Errorf("mission store approval grant granted_at is required")
	}
	return nil
}

func ValidateAuditEventRecord(record AuditEventRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store audit event record_version must be positive")
	}
	if record.Seq == 0 {
		return fmt.Errorf("mission store audit event seq must be positive")
	}
	event := normalizeAuditEvent(record.Event)
	if event.EventID == "" {
		return fmt.Errorf("mission store audit event event_id is required")
	}
	if event.JobID == "" {
		return fmt.Errorf("mission store audit event job_id is required")
	}
	if event.Timestamp.IsZero() {
		return fmt.Errorf("mission store audit event timestamp is required")
	}
	return nil
}

func ValidateArtifactRecord(record ArtifactRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store artifact record_version must be positive")
	}
	if record.LastSeq == 0 {
		return fmt.Errorf("mission store artifact last_seq must be positive")
	}
	if record.ArtifactID == "" {
		return fmt.Errorf("mission store artifact artifact_id is required")
	}
	if record.JobID == "" {
		return fmt.Errorf("mission store artifact job_id is required")
	}
	if record.StepID == "" {
		return fmt.Errorf("mission store artifact step_id is required")
	}
	if !isValidStepType(record.StepType) {
		return fmt.Errorf("mission store artifact step_type %q is invalid", record.StepType)
	}
	if record.Path == "" {
		return fmt.Errorf("mission store artifact path is required")
	}
	if record.State == "" {
		return fmt.Errorf("mission store artifact state is required")
	}
	return nil
}

func ValidateFrankZohoSendReceiptRecord(record FrankZohoSendReceiptRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store frank zoho send receipt record_version must be positive")
	}
	if record.LastSeq == 0 {
		return fmt.Errorf("mission store frank zoho send receipt last_seq must be positive")
	}
	if strings.TrimSpace(record.ReceiptID) == "" {
		return fmt.Errorf("mission store frank zoho send receipt receipt_id is required")
	}
	if strings.TrimSpace(record.JobID) == "" {
		return fmt.Errorf("mission store frank zoho send receipt job_id is required")
	}
	if strings.TrimSpace(record.StepID) == "" {
		return fmt.Errorf("mission store frank zoho send receipt step_id is required")
	}
	receipt := FrankZohoSendReceipt{
		StepID:             record.StepID,
		Provider:           record.Provider,
		ProviderAccountID:  record.ProviderAccountID,
		FromAddress:        record.FromAddress,
		FromDisplayName:    record.FromDisplayName,
		ProviderMessageID:  record.ProviderMessageID,
		ProviderMailID:     record.ProviderMailID,
		MIMEMessageID:      record.MIMEMessageID,
		OriginalMessageURL: record.OriginalMessageURL,
	}
	if err := ValidateFrankZohoSendReceipt(receipt); err != nil {
		return err
	}
	return nil
}

func ValidateBatchCommitRecord(record BatchCommitRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store batch commit record_version must be positive")
	}
	if record.JobID == "" {
		return fmt.Errorf("mission store batch commit job_id is required")
	}
	if record.Seq == 0 {
		return fmt.Errorf("mission store batch commit seq must be positive")
	}
	if record.AttemptID == "" {
		return fmt.Errorf("mission store batch commit attempt_id is required")
	}
	if record.CommittedAt.IsZero() {
		return fmt.Errorf("mission store batch commit committed_at is required")
	}
	return nil
}

func StoreJobRuntimeRecord(root string, record JobRuntimeRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateJobRuntimeRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreJobRuntimePath(root, record.JobID), record)
}

func LoadJobRuntimeRecord(root string, jobID string) (JobRuntimeRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return JobRuntimeRecord{}, err
	}
	var record JobRuntimeRecord
	if err := LoadStoreJSON(StoreJobRuntimePath(root, jobID), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return JobRuntimeRecord{}, ErrJobRuntimeRecordNotFound
		}
		return JobRuntimeRecord{}, err
	}
	if err := ValidateJobRuntimeRecord(record); err != nil {
		return JobRuntimeRecord{}, err
	}
	return record, nil
}

func StoreStepRuntimeRecord(root string, record StepRuntimeRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateStepRuntimeRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(storeStepRuntimeVersionPath(root, record.JobID, record.StepID, record.LastSeq, record.AttemptID), record)
}

func LoadStepRuntimeRecord(root, jobID, stepID string) (StepRuntimeRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return StepRuntimeRecord{}, err
	}
	record, err := loadLatestVersionedJSONRecord(
		storeStepRuntimeVersionsDir(root, jobID, stepID),
		loadStepRuntimeRecordFile,
		func(record StepRuntimeRecord) uint64 { return record.LastSeq },
	)
	if err != nil {
		return StepRuntimeRecord{}, err
	}
	return record, nil
}

func StoreRuntimeControlRecord(root string, record RuntimeControlRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateRuntimeControlRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(storeRuntimeControlVersionPath(root, record.JobID, record.LastSeq, record.AttemptID), record)
}

func LoadRuntimeControlRecord(root, jobID string) (RuntimeControlRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimeControlRecord{}, err
	}
	record, err := loadLatestVersionedJSONRecord(
		storeRuntimeControlVersionsDir(root, jobID),
		loadRuntimeControlRecordFile,
		func(record RuntimeControlRecord) uint64 { return record.LastSeq },
	)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimeControlRecord{}, ErrRuntimeControlRecordNotFound
		}
		return RuntimeControlRecord{}, err
	}
	return record, nil
}

func StoreApprovalRequestRecord(root string, record ApprovalRequestRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateApprovalRequestRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(storeApprovalRequestVersionPath(root, record.JobID, record.RequestID, record.LastSeq, record.AttemptID), record)
}

func StoreApprovalGrantRecord(root string, record ApprovalGrantRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateApprovalGrantRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(storeApprovalGrantVersionPath(root, record.JobID, record.GrantID, record.LastSeq, record.AttemptID), record)
}

func StoreAuditEventRecord(root string, record AuditEventRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record.Event = normalizeAuditEvent(record.Event)
	if err := ValidateAuditEventRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(storeAuditEventVersionPath(root, record.Event.JobID, record.Seq, record.AttemptID, record.Event.EventID), record)
}

func StoreArtifactRecord(root string, record ArtifactRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateArtifactRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(storeArtifactVersionPath(root, record.JobID, record.ArtifactID, record.LastSeq, record.AttemptID), record)
}

func StoreFrankZohoSendReceiptRecord(root string, record FrankZohoSendReceiptRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateFrankZohoSendReceiptRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(storeFrankZohoSendReceiptVersionPath(root, record.JobID, record.ReceiptID, record.LastSeq, record.AttemptID), record)
}

func StoreBatchCommitRecord(root string, record BatchCommitRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateBatchCommitRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(storeBatchCommitPath(root, record.JobID, record.Seq), record)
}

func LoadBatchCommitRecord(root, jobID string, seq uint64) (BatchCommitRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return BatchCommitRecord{}, err
	}
	var record BatchCommitRecord
	if err := LoadStoreJSON(storeBatchCommitPath(root, jobID, seq), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return BatchCommitRecord{}, ErrBatchCommitRecordNotFound
		}
		return BatchCommitRecord{}, err
	}
	if err := ValidateBatchCommitRecord(record); err != nil {
		return BatchCommitRecord{}, err
	}
	return record, nil
}

func LoadCommittedJobRuntimeRecord(root, jobID string) (JobRuntimeRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return JobRuntimeRecord{}, err
	}
	return LoadJobRuntimeRecord(root, jobID)
}

func LoadCommittedRuntimeControlRecord(root, jobID string) (RuntimeControlRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimeControlRecord{}, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		return RuntimeControlRecord{}, err
	}
	resolver := newStoreCommittedAttemptResolver(root, jobID)
	record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
		storeRuntimeControlVersionsDir(root, jobID),
		jobRuntime.AppliedSeq,
		resolver,
		loadRuntimeControlRecordFile,
		func(record RuntimeControlRecord) uint64 { return record.LastSeq },
		func(record RuntimeControlRecord) string { return record.AttemptID },
	)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimeControlRecord{}, ErrRuntimeControlRecordNotFound
		}
		return RuntimeControlRecord{}, err
	}
	return record, nil
}

func ListCommittedStepRuntimeRecords(root, jobID string) ([]StepRuntimeRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrJobRuntimeRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	resolver := newStoreCommittedAttemptResolver(root, jobID)
	stepIDs, err := listStoreRecordKeys(storeStepsDir(root, jobID))
	if err != nil {
		return nil, err
	}
	records := make([]StepRuntimeRecord, 0, len(stepIDs))
	for _, stepID := range stepIDs {
		record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
			storeStepRuntimeVersionsDir(root, jobID, stepID),
			jobRuntime.AppliedSeq,
			resolver,
			loadStepRuntimeRecordFile,
			func(record StepRuntimeRecord) uint64 { return record.LastSeq },
			func(record StepRuntimeRecord) string { return record.AttemptID },
		)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func ListCommittedApprovalRequestRecords(root, jobID string) ([]ApprovalRequestRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrJobRuntimeRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	resolver := newStoreCommittedAttemptResolver(root, jobID)
	requestIDs, err := listStoreRecordKeys(StoreApprovalRequestsDir(root, jobID))
	if err != nil {
		return nil, err
	}
	records := make([]ApprovalRequestRecord, 0, len(requestIDs))
	for _, requestID := range requestIDs {
		record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
			storeApprovalRequestVersionsDir(root, jobID, requestID),
			jobRuntime.AppliedSeq,
			resolver,
			loadApprovalRequestRecordFile,
			func(record ApprovalRequestRecord) uint64 { return record.LastSeq },
			func(record ApprovalRequestRecord) string { return record.AttemptID },
		)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func ListCommittedApprovalGrantRecords(root, jobID string) ([]ApprovalGrantRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrJobRuntimeRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	resolver := newStoreCommittedAttemptResolver(root, jobID)
	grantIDs, err := listStoreRecordKeys(StoreApprovalGrantsDir(root, jobID))
	if err != nil {
		return nil, err
	}
	records := make([]ApprovalGrantRecord, 0, len(grantIDs))
	for _, grantID := range grantIDs {
		record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
			storeApprovalGrantVersionsDir(root, jobID, grantID),
			jobRuntime.AppliedSeq,
			resolver,
			loadApprovalGrantRecordFile,
			func(record ApprovalGrantRecord) uint64 { return record.LastSeq },
			func(record ApprovalGrantRecord) string { return record.AttemptID },
		)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func ListCommittedAuditEventRecords(root, jobID string) ([]AuditEventRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrJobRuntimeRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	resolver := newStoreCommittedAttemptResolver(root, jobID)
	records, err := listStoreJSONRecords(StoreAuditDir(root, jobID), loadAuditEventRecordFile)
	if err != nil {
		return nil, err
	}
	visible := make([]AuditEventRecord, 0, len(records))
	for _, record := range records {
		if record.Seq > jobRuntime.AppliedSeq {
			continue
		}
		allowed, err := resolver.allows(record.Seq, record.AttemptID)
		if err != nil {
			return nil, err
		}
		if allowed {
			visible = append(visible, record)
		}
	}
	return visible, nil
}

func ListCommittedArtifactRecords(root, jobID string) ([]ArtifactRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrJobRuntimeRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	resolver := newStoreCommittedAttemptResolver(root, jobID)
	artifactIDs, err := listStoreRecordKeys(StoreArtifactsDir(root, jobID))
	if err != nil {
		return nil, err
	}
	records := make([]ArtifactRecord, 0, len(artifactIDs))
	for _, artifactID := range artifactIDs {
		record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
			storeArtifactVersionsDir(root, jobID, artifactID),
			jobRuntime.AppliedSeq,
			resolver,
			loadArtifactRecordFile,
			func(record ArtifactRecord) uint64 { return record.LastSeq },
			func(record ArtifactRecord) string { return record.AttemptID },
		)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func ListCommittedFrankZohoSendReceiptRecords(root, jobID string) ([]FrankZohoSendReceiptRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrJobRuntimeRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	resolver := newStoreCommittedAttemptResolver(root, jobID)
	receiptIDs, err := listStoreRecordKeys(StoreFrankZohoSendReceiptsDir(root, jobID))
	if err != nil {
		return nil, err
	}
	records := make([]FrankZohoSendReceiptRecord, 0, len(receiptIDs))
	for _, receiptID := range receiptIDs {
		record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
			storeFrankZohoSendReceiptVersionsDir(root, jobID, receiptID),
			jobRuntime.AppliedSeq,
			resolver,
			loadFrankZohoSendReceiptRecordFile,
			func(record FrankZohoSendReceiptRecord) uint64 { return record.LastSeq },
			func(record FrankZohoSendReceiptRecord) string { return record.AttemptID },
		)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].ProviderMessageID != records[j].ProviderMessageID {
			return records[i].ProviderMessageID < records[j].ProviderMessageID
		}
		return records[i].ReceiptID < records[j].ReceiptID
	})
	return records, nil
}

func LoadCommittedActiveJobRecord(root, jobID string) (ActiveJobRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ActiveJobRecord{}, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		return ActiveJobRecord{}, err
	}
	if !HoldsGlobalActiveJobOccupancy(jobRuntime.State) {
		return ActiveJobRecord{}, ErrActiveJobRecordNotFound
	}
	record, err := LoadActiveJobRecord(root)
	if err != nil {
		return ActiveJobRecord{}, err
	}
	if record.JobID != jobID || record.ActivationSeq > jobRuntime.AppliedSeq {
		return ActiveJobRecord{}, ErrActiveJobRecordNotFound
	}
	if record.AttemptID != "" {
		commit, err := LoadBatchCommitRecord(root, jobID, record.ActivationSeq)
		if err != nil {
			if errors.Is(err, ErrBatchCommitRecordNotFound) {
				return ActiveJobRecord{}, ErrActiveJobRecordNotFound
			}
			return ActiveJobRecord{}, err
		}
		if commit.AttemptID != record.AttemptID {
			return ActiveJobRecord{}, ErrActiveJobRecordNotFound
		}
	}
	return record, nil
}

func loadStepRuntimeRecordFile(path string) (StepRuntimeRecord, error) {
	var record StepRuntimeRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return StepRuntimeRecord{}, err
	}
	if err := ValidateStepRuntimeRecord(record); err != nil {
		return StepRuntimeRecord{}, err
	}
	return record, nil
}

func loadRuntimeControlRecordFile(path string) (RuntimeControlRecord, error) {
	var record RuntimeControlRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RuntimeControlRecord{}, err
	}
	if err := ValidateRuntimeControlRecord(record); err != nil {
		return RuntimeControlRecord{}, err
	}
	return record, nil
}

func loadApprovalRequestRecordFile(path string) (ApprovalRequestRecord, error) {
	var record ApprovalRequestRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return ApprovalRequestRecord{}, err
	}
	if err := ValidateApprovalRequestRecord(record); err != nil {
		return ApprovalRequestRecord{}, err
	}
	return record, nil
}

func loadApprovalGrantRecordFile(path string) (ApprovalGrantRecord, error) {
	var record ApprovalGrantRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return ApprovalGrantRecord{}, err
	}
	if err := ValidateApprovalGrantRecord(record); err != nil {
		return ApprovalGrantRecord{}, err
	}
	return record, nil
}

func loadAuditEventRecordFile(path string) (AuditEventRecord, error) {
	var record AuditEventRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return AuditEventRecord{}, err
	}
	record.Event = normalizeAuditEvent(record.Event)
	if err := ValidateAuditEventRecord(record); err != nil {
		return AuditEventRecord{}, err
	}
	return record, nil
}

func loadArtifactRecordFile(path string) (ArtifactRecord, error) {
	var record ArtifactRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return ArtifactRecord{}, err
	}
	if err := ValidateArtifactRecord(record); err != nil {
		return ArtifactRecord{}, err
	}
	return record, nil
}

func loadFrankZohoSendReceiptRecordFile(path string) (FrankZohoSendReceiptRecord, error) {
	var record FrankZohoSendReceiptRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return FrankZohoSendReceiptRecord{}, err
	}
	if err := ValidateFrankZohoSendReceiptRecord(record); err != nil {
		return FrankZohoSendReceiptRecord{}, err
	}
	return record, nil
}

type storeCommittedAttemptResolver struct {
	root     string
	jobID    string
	attempts map[uint64]string
	missing  map[uint64]bool
}

func newStoreCommittedAttemptResolver(root, jobID string) *storeCommittedAttemptResolver {
	return &storeCommittedAttemptResolver{
		root:     root,
		jobID:    jobID,
		attempts: make(map[uint64]string),
		missing:  make(map[uint64]bool),
	}
}

func (r *storeCommittedAttemptResolver) allows(seq uint64, attemptID string) (bool, error) {
	if attemptID == "" {
		return true, nil
	}
	winner, ok, err := r.attemptIDForSeq(seq)
	if err != nil {
		return false, err
	}
	return ok && winner == attemptID, nil
}

func (r *storeCommittedAttemptResolver) attemptIDForSeq(seq uint64) (string, bool, error) {
	if attemptID, ok := r.attempts[seq]; ok {
		return attemptID, true, nil
	}
	if r.missing[seq] {
		return "", false, nil
	}
	record, err := LoadBatchCommitRecord(r.root, r.jobID, seq)
	if err != nil {
		if errors.Is(err, ErrBatchCommitRecordNotFound) {
			r.missing[seq] = true
			return "", false, nil
		}
		return "", false, err
	}
	r.attempts[seq] = record.AttemptID
	return record.AttemptID, true, nil
}

func storeRuntimeControlVersionsDir(root, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "runtime_control")
}

func storeRuntimeControlVersionPath(root, jobID string, seq uint64, attemptID string) string {
	return filepath.Join(storeRuntimeControlVersionsDir(root, jobID), storeVersionFilename(seq, attemptID))
}

func storeStepsDir(root, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "steps")
}

func storeStepRuntimeVersionsDir(root, jobID, stepID string) string {
	return filepath.Join(storeStepsDir(root, jobID), stepID)
}

func storeStepRuntimeVersionPath(root, jobID, stepID string, seq uint64, attemptID string) string {
	return filepath.Join(storeStepRuntimeVersionsDir(root, jobID, stepID), storeVersionFilename(seq, attemptID))
}

func storeApprovalRequestVersionsDir(root, jobID, requestID string) string {
	return filepath.Join(StoreApprovalRequestsDir(root, jobID), requestID)
}

func storeApprovalRequestVersionPath(root, jobID, requestID string, seq uint64, attemptID string) string {
	return filepath.Join(storeApprovalRequestVersionsDir(root, jobID, requestID), storeVersionFilename(seq, attemptID))
}

func storeApprovalGrantVersionsDir(root, jobID, grantID string) string {
	return filepath.Join(StoreApprovalGrantsDir(root, jobID), grantID)
}

func storeApprovalGrantVersionPath(root, jobID, grantID string, seq uint64, attemptID string) string {
	return filepath.Join(storeApprovalGrantVersionsDir(root, jobID, grantID), storeVersionFilename(seq, attemptID))
}

func storeArtifactVersionsDir(root, jobID, artifactID string) string {
	return filepath.Join(StoreArtifactsDir(root, jobID), artifactID)
}

func storeArtifactVersionPath(root, jobID, artifactID string, seq uint64, attemptID string) string {
	return filepath.Join(storeArtifactVersionsDir(root, jobID, artifactID), storeVersionFilename(seq, attemptID))
}

func storeFrankZohoSendReceiptVersionsDir(root, jobID, receiptID string) string {
	return filepath.Join(StoreFrankZohoSendReceiptsDir(root, jobID), receiptID)
}

func storeFrankZohoSendReceiptVersionPath(root, jobID, receiptID string, seq uint64, attemptID string) string {
	return filepath.Join(storeFrankZohoSendReceiptVersionsDir(root, jobID, receiptID), storeVersionFilename(seq, attemptID))
}

func storeBatchCommitsDir(root, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "commits")
}

func storeBatchCommitPath(root, jobID string, seq uint64) string {
	return filepath.Join(storeBatchCommitsDir(root, jobID), storeSeqFilename(seq))
}

func storeAuditEventVersionPath(root, jobID string, seq uint64, attemptID string, eventID string) string {
	if attemptID == "" {
		return filepath.Join(StoreAuditDir(root, jobID), fmt.Sprintf("%020d--%s.json", seq, eventID))
	}
	return filepath.Join(StoreAuditDir(root, jobID), fmt.Sprintf("%020d--%s--%s.json", seq, attemptID, eventID))
}

func storeSeqFilename(seq uint64) string {
	return fmt.Sprintf("%020d.json", seq)
}

func storeVersionFilename(seq uint64, attemptID string) string {
	if attemptID == "" {
		return storeSeqFilename(seq)
	}
	return fmt.Sprintf("%020d--%s.json", seq, attemptID)
}

func listStoreJSONRecords[T any](dir string, load func(string) (T, error)) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	records := make([]T, 0, len(names))
	for _, name := range names {
		record, err := load(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func listStoreRecordKeys(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func loadLatestVersionedJSONRecord[T any](dir string, load func(string) (T, error), seqOf func(T) uint64) (T, error) {
	return loadLatestVisibleVersionedJSONRecordAtOrBefore(dir, math.MaxUint64, nil, load, seqOf, func(T) string { return "" })
}

func loadLatestVisibleVersionedJSONRecordAtOrBefore[T any](dir string, maxSeq uint64, resolver *storeCommittedAttemptResolver, load func(string) (T, error), seqOf func(T) uint64, attemptOf func(T) string) (T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		var zero T
		if errors.Is(err, os.ErrNotExist) {
			return zero, os.ErrNotExist
		}
		return zero, err
	}

	var (
		best    T
		bestSeq uint64
		found   bool
	)
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		record, err := load(filepath.Join(dir, entry.Name()))
		if err != nil {
			var zero T
			return zero, err
		}
		seq := seqOf(record)
		if seq > maxSeq {
			continue
		}
		if resolver != nil {
			allowed, err := resolver.allows(seq, attemptOf(record))
			if err != nil {
				var zero T
				return zero, err
			}
			if !allowed {
				continue
			}
		}
		if !found || seq > bestSeq {
			best = record
			bestSeq = seq
			found = true
		}
	}
	if !found {
		var zero T
		return zero, os.ErrNotExist
	}
	return best, nil
}

func isStoreJSONDataFile(name string) bool {
	return strings.HasSuffix(name, ".json")
}
