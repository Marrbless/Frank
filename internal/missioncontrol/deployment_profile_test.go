package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDeploymentProfileRecordPhoneResidentWithFakeCaps(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	record := validDeploymentProfileRecord(now, DeploymentProfilePhoneResident, ExecutionHostPhone, true, fakePhoneDeploymentCapabilities(), nil)
	got, changed, err := StoreDeploymentProfileRecord(root, record)
	if err != nil {
		t.Fatalf("StoreDeploymentProfileRecord(phone) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreDeploymentProfileRecord(phone) changed = false, want true")
	}
	record.RecordVersion = StoreRecordVersion
	record = NormalizeDeploymentProfileRecord(record)
	if !reflect.DeepEqual(got, record) {
		t.Fatalf("StoreDeploymentProfileRecord() = %#v, want %#v", got, record)
	}
	if !got.Assessment.Ready || got.Assessment.ExecutionHost != ExecutionHostPhone || !got.Assessment.StrictPhoneOnly {
		t.Fatalf("Assessment = %#v, want ready strict phone", got.Assessment)
	}

	loaded, err := LoadDeploymentProfileRecord(root, record.DeploymentID)
	if err != nil {
		t.Fatalf("LoadDeploymentProfileRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, record) {
		t.Fatalf("LoadDeploymentProfileRecord() = %#v, want %#v", loaded, record)
	}
	replayed, changed, err := StoreDeploymentProfileRecord(root, record)
	if err != nil {
		t.Fatalf("StoreDeploymentProfileRecord(replay) error = %v", err)
	}
	if changed || !reflect.DeepEqual(replayed, record) {
		t.Fatalf("replay = %#v changed=%v, want idempotent %#v", replayed, changed, record)
	}
}

func TestAssessDeploymentProfilePhoneOnlyAndDesktopDevModes(t *testing.T) {
	t.Parallel()

	phone := AssessDeploymentProfile(DeploymentProfilePhoneResident, ExecutionHostPhone, true, fakePhoneDeploymentCapabilities())
	if !phone.Ready {
		t.Fatalf("phone assessment = %#v, want ready", phone)
	}
	desktop := AssessDeploymentProfile(DeploymentProfileDesktopDev, ExecutionHostDesktopDev, false, WorkspaceRunnerHostCapabilities{
		LocalWorkspaceAvailable:  true,
		NetworkDisabled:          true,
		ExternalServicesDisabled: true,
	})
	if !desktop.Ready {
		t.Fatalf("desktop_dev assessment = %#v, want ready outside strict mode", desktop)
	}
	strictDesktop := AssessDeploymentProfile(DeploymentProfileDesktopDev, ExecutionHostDesktopDev, true, WorkspaceRunnerHostCapabilities{
		LocalWorkspaceAvailable:  true,
		NetworkDisabled:          true,
		ExternalServicesDisabled: true,
	})
	if strictDesktop.Ready || !strings.Contains(strings.Join(strictDesktop.Blockers, "; "), "strict_phone_only") {
		t.Fatalf("strict desktop assessment = %#v, want strict phone-only blocker", strictDesktop)
	}
	nonPhone := AssessDeploymentProfile(DeploymentProfilePhoneResident, ExecutionHostDesktopDev, true, fakePhoneDeploymentCapabilities())
	if nonPhone.Ready || !strings.Contains(strings.Join(nonPhone.Blockers, "; "), "execution_host phone") {
		t.Fatalf("non-phone phone_resident assessment = %#v, want phone host blocker", nonPhone)
	}
}

func TestDeploymentProfileRecordRejectsNotReadyAssessment(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 20, 11, 0, 0, 0, time.UTC)
	record := validDeploymentProfileRecord(now, DeploymentProfilePhoneResident, ExecutionHostPhone, true, WorkspaceRunnerHostCapabilities{
		LocalWorkspaceAvailable:  true,
		NetworkDisabled:          true,
		ExternalServicesDisabled: true,
	}, nil)
	if _, _, err := StoreDeploymentProfileRecord(root, record); err == nil {
		t.Fatal("StoreDeploymentProfileRecord() error = nil, want missing fake phone blocker")
	} else if !strings.Contains(err.Error(), "fake_phone_profile_available") {
		t.Fatalf("StoreDeploymentProfileRecord() error = %q, want fake phone blocker", err.Error())
	}
}

func TestDeploymentProfileRecordRejectsDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	record := validDeploymentProfileRecord(now, DeploymentProfilePhoneResident, ExecutionHostPhone, true, fakePhoneDeploymentCapabilities(), nil)
	if _, _, err := StoreDeploymentProfileRecord(root, record); err != nil {
		t.Fatalf("StoreDeploymentProfileRecord(first) error = %v", err)
	}
	record.RecordVersion = StoreRecordVersion
	record = NormalizeDeploymentProfileRecord(record)
	divergent := record
	divergent.CreatedBy = "different"
	if _, _, err := StoreDeploymentProfileRecord(root, divergent); err == nil {
		t.Fatal("StoreDeploymentProfileRecord(divergent) error = nil, want duplicate rejection")
	}
}

func validDeploymentProfileRecord(now time.Time, profileName, executionHost string, strictPhoneOnly bool, capabilities WorkspaceRunnerHostCapabilities, edit func(*DeploymentProfileRecord)) DeploymentProfileRecord {
	assessment := AssessDeploymentProfile(profileName, executionHost, strictPhoneOnly, capabilities)
	record := DeploymentProfileRecord{
		DeploymentID:    DeploymentProfileID(profileName, executionHost, strictPhoneOnly),
		ProfileName:     profileName,
		ExecutionHost:   executionHost,
		StrictPhoneOnly: strictPhoneOnly,
		Capabilities:    capabilities,
		Assessment:      assessment,
		CreatedAt:       now,
		CreatedBy:       "operator",
	}
	if edit != nil {
		edit(&record)
	}
	return record
}

func fakePhoneDeploymentCapabilities() WorkspaceRunnerHostCapabilities {
	return WorkspaceRunnerHostCapabilities{
		HostProfile:               "dev-host",
		LocalWorkspaceAvailable:   true,
		FakePhoneProfileAvailable: true,
		NetworkDisabled:           true,
		ExternalServicesDisabled:  true,
	}
}
