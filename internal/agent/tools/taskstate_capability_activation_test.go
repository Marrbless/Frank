package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestTaskStateActivateStepNotificationsCapabilityPathCallsHookOnce(t *testing.T) {
	t.Parallel()

	root := writeTaskStateNotificationsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.notificationsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreTelegramNotificationsCapabilityExposure(root); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("notificationsCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedNotificationsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedNotificationsCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.NotificationsTelegramCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.NotificationsTelegramCapabilityID)
	}
}

func TestTaskStateActivateStepNotificationsCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateNotificationsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	home := t.TempDir()
	configDir := filepath.Join(home, ".picobot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	configJSON := `{"channels":{"telegram":{"enabled":true,"token":"telegram-token","allowFrom":["12345"]}}}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}
	t.Setenv("HOME", home)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedNotificationsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedNotificationsCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.NotificationsTelegramCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.NotificationsTelegramCapabilityValidator)
	}
}

func TestTaskStateActivateStepNotificationsCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateNotificationsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.notificationsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("notificationsCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-notifications", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepNotificationsCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	t.Parallel()

	root := writeTaskStateNotificationsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.notificationsCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed notifications exposure rejection")
	}
	if !strings.Contains(err.Error(), `notifications capability requires one committed capability record named "notifications"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepSharedStorageCapabilityPathCallsHookOnce(t *testing.T) {
	t.Parallel()

	root := writeTaskStateSharedStorageCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.sharedStorageCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, filepath.Join(t.TempDir(), "workspace")); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("sharedStorageCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedSharedStorageCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.SharedStorageWorkspaceCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.SharedStorageWorkspaceCapabilityID)
	}
}

func TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateSharedStorageCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateSharedStorageCapabilityConfigFixture(t)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedSharedStorageCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.SharedStorageWorkspaceCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.SharedStorageWorkspaceCapabilityValidator)
	}
	if _, err := os.Stat(filepath.Join(workspace, "SOUL.md")); err != nil {
		t.Fatalf("Stat(SOUL.md) error = %v", err)
	}
}

func TestTaskStateActivateStepSharedStorageCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateSharedStorageCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.sharedStorageCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("sharedStorageCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-shared-storage", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepSharedStorageCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	t.Parallel()

	root := writeTaskStateSharedStorageCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.sharedStorageCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed shared_storage exposure rejection")
	}
	if !strings.Contains(err.Error(), `shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepContactsCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateContactsCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.contactsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceContactsCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("contactsCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedContactsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedContactsCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.ContactsLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.ContactsLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableContactsSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableContactsSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepContactsCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateContactsCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedContactsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedContactsCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.ContactsLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.ContactsLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableContactsSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableContactsSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.ContactsLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.ContactsLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepContactsCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.contactsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("contactsCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-contacts", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepContactsCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateContactsCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.contactsCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed contacts exposure rejection")
	}
	if !strings.Contains(err.Error(), `contacts capability requires one committed capability record named "contacts"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepContactsCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.contactsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("contactsCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `contacts capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}

func TestTaskStateActivateStepLocationCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateLocationCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.locationCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceLocationCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("locationCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedLocationCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedLocationCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.LocationLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.LocationLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableLocationSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableLocationSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepLocationCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateLocationCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedLocationCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedLocationCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.LocationLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.LocationLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableLocationSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableLocationSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.LocationLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.LocationLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepLocationCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.locationCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("locationCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-location", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepLocationCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateLocationCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.locationCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed location exposure rejection")
	}
	if !strings.Contains(err.Error(), `location capability requires one committed capability record named "location"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepLocationCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.locationCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("locationCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `location capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}

func TestTaskStateActivateStepCameraCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateCameraCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateCameraCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.CameraCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-camera",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.cameraCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceCameraCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("cameraCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedCameraCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedCameraCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.CameraLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.CameraLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableCameraSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableCameraSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepCameraCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateCameraCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateCameraCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.CameraCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-camera",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedCameraCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedCameraCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.CameraLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.CameraLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableCameraSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableCameraSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.CameraLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.CameraLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepCameraCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateCameraCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.CameraCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-camera",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.cameraCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("cameraCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-camera", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepCameraCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateCameraCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateCameraCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.CameraCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-camera",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.cameraCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed camera exposure rejection")
	}
	if !strings.Contains(err.Error(), `camera capability requires one committed capability record named "camera"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepCameraCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateCameraCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.CameraCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-camera",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.cameraCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("cameraCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `camera capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}

func TestTaskStateActivateStepMicrophoneCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateMicrophoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateMicrophoneCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.MicrophoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-microphone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.microphoneCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceMicrophoneCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("microphoneCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedMicrophoneCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedMicrophoneCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.MicrophoneLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.MicrophoneLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableMicrophoneSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableMicrophoneSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepMicrophoneCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateMicrophoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateMicrophoneCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.MicrophoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-microphone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedMicrophoneCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedMicrophoneCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.MicrophoneLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.MicrophoneLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableMicrophoneSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableMicrophoneSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.MicrophoneLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.MicrophoneLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepMicrophoneCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateMicrophoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.MicrophoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-microphone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.microphoneCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("microphoneCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-microphone", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepMicrophoneCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateMicrophoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateMicrophoneCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.MicrophoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-microphone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.microphoneCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed microphone exposure rejection")
	}
	if !strings.Contains(err.Error(), `microphone capability requires one committed capability record named "microphone"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepMicrophoneCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateMicrophoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.MicrophoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-microphone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.microphoneCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("microphoneCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `microphone capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}

func TestTaskStateActivateStepSMSPhoneCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateSMSPhoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateSMSPhoneCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SMSPhoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-sms-phone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.smsPhoneCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceSMSPhoneCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("smsPhoneCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedSMSPhoneCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedSMSPhoneCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.SMSPhoneLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.SMSPhoneLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableSMSPhoneSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableSMSPhoneSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepSMSPhoneCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateSMSPhoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateSMSPhoneCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SMSPhoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-sms-phone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedSMSPhoneCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedSMSPhoneCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.SMSPhoneLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.SMSPhoneLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableSMSPhoneSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableSMSPhoneSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.SMSPhoneLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.SMSPhoneLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepSMSPhoneCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateSMSPhoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SMSPhoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-sms-phone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.smsPhoneCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("smsPhoneCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-sms-phone", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepSMSPhoneCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateSMSPhoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateSMSPhoneCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SMSPhoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-sms-phone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.smsPhoneCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed sms_phone exposure rejection")
	}
	if !strings.Contains(err.Error(), `sms_phone capability requires one committed capability record named "sms_phone"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepSMSPhoneCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateSMSPhoneCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SMSPhoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-sms-phone",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.smsPhoneCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("smsPhoneCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `sms_phone capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}

func TestTaskStateActivateStepBluetoothNFCCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateBluetoothNFCCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateBluetoothNFCCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BluetoothNFCCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-bluetooth-nfc",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.bluetoothNFCCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceBluetoothNFCCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("bluetoothNFCCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedBluetoothNFCCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedBluetoothNFCCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.BluetoothNFCLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.BluetoothNFCLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableBluetoothNFCSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableBluetoothNFCSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepBluetoothNFCCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateBluetoothNFCCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateBluetoothNFCCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BluetoothNFCCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-bluetooth-nfc",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedBluetoothNFCCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedBluetoothNFCCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.BluetoothNFCLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.BluetoothNFCLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableBluetoothNFCSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableBluetoothNFCSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.BluetoothNFCLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.BluetoothNFCLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepBluetoothNFCCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateBluetoothNFCCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BluetoothNFCCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-bluetooth-nfc",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.bluetoothNFCCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("bluetoothNFCCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-bluetooth-nfc", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepBluetoothNFCCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateBluetoothNFCCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateBluetoothNFCCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BluetoothNFCCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-bluetooth-nfc",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.bluetoothNFCCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed bluetooth_nfc exposure rejection")
	}
	if !strings.Contains(err.Error(), `bluetooth_nfc capability requires one committed capability record named "bluetooth_nfc"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepBluetoothNFCCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateBluetoothNFCCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BluetoothNFCCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-bluetooth-nfc",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.bluetoothNFCCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("bluetoothNFCCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `bluetooth_nfc capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}

func TestTaskStateActivateStepBroadAppControlCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateBroadAppControlCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateBroadAppControlCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BroadAppControlCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-broad-app-control",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.broadAppControlCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceBroadAppControlCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("broadAppControlCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedBroadAppControlCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedBroadAppControlCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.BroadAppControlLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.BroadAppControlLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableBroadAppControlSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableBroadAppControlSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepBroadAppControlCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateBroadAppControlCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateBroadAppControlCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BroadAppControlCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-broad-app-control",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedBroadAppControlCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedBroadAppControlCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.BroadAppControlLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.BroadAppControlLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableBroadAppControlSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableBroadAppControlSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.BroadAppControlLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.BroadAppControlLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepBroadAppControlCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateBroadAppControlCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BroadAppControlCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-broad-app-control",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.broadAppControlCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("broadAppControlCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-broad-app-control", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepBroadAppControlCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateBroadAppControlCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateBroadAppControlCapabilityConfigFixture(t)
	storeTaskStateSharedStorageCapabilityExposure(t, root, workspace)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BroadAppControlCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-broad-app-control",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.broadAppControlCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed broad_app_control exposure rejection")
	}
	if !strings.Contains(err.Error(), `broad_app_control capability requires one committed capability record named "broad_app_control"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepBroadAppControlCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateBroadAppControlCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BroadAppControlCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-broad-app-control",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.broadAppControlCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("broadAppControlCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `broad_app_control capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}
