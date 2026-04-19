package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

type taskStateCapabilityProposalFixtureSpec struct {
	proposalID      string
	capabilityName  string
	whyNeeded       string
	missionFamilies []string
	risks           []string
	validators      []string
	killSwitch      string
	dataAccessed    []string
	createdAt       time.Time
}

func writeTaskStateCapabilityProposalFixture(t *testing.T, spec taskStateCapabilityProposalFixtureSpec, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	root := t.TempDir()
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       spec.proposalID,
		CapabilityName:   spec.capabilityName,
		WhyNeeded:        spec.whyNeeded,
		MissionFamilies:  spec.missionFamilies,
		Risks:            spec.risks,
		Validators:       spec.validators,
		KillSwitch:       spec.killSwitch,
		DataAccessed:     spec.dataAccessed,
		ApprovalRequired: true,
		CreatedAt:        spec.createdAt,
		State:            state,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	return root
}

func writeTaskStateNotificationsCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-notifications",
		capabilityName:  missioncontrol.NotificationsCapabilityName,
		whyNeeded:       "mission requires operator-facing notifications",
		missionFamilies: []string{"outreach"},
		risks:           []string{"notification spam"},
		validators:      []string{"telegram owner-control channel confirmed"},
		killSwitch:      "disable telegram channel and revoke proposal",
		dataAccessed:    []string{"notifications"},
		createdAt:       time.Date(2026, 4, 18, 16, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateSharedStorageCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-shared-storage",
		capabilityName:  missioncontrol.SharedStorageCapabilityName,
		whyNeeded:       "mission requires shared workspace storage",
		missionFamilies: []string{"workspace"},
		risks:           []string{"workspace data exposure"},
		validators:      []string{"configured workspace root initialized and writable"},
		killSwitch:      "disable workspace-backed shared_storage exposure and revoke proposal",
		dataAccessed:    []string{"shared storage"},
		createdAt:       time.Date(2026, 4, 18, 17, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateContactsCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-contacts",
		capabilityName:  missioncontrol.ContactsCapabilityName,
		whyNeeded:       "mission requires local shared contacts access",
		missionFamilies: []string{"workspace"},
		risks:           []string{"local contacts exposure"},
		validators:      []string{"shared_storage exposed and committed contacts source file exists and is readable"},
		killSwitch:      "disable contacts capability exposure and remove committed contacts source reference",
		dataAccessed:    []string{"contacts"},
		createdAt:       time.Date(2026, 4, 18, 22, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateLocationCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-location",
		capabilityName:  missioncontrol.LocationCapabilityName,
		whyNeeded:       "mission requires local shared location access",
		missionFamilies: []string{"workspace"},
		risks:           []string{"local location exposure"},
		validators:      []string{"shared_storage exposed and committed location source file exists and is readable"},
		killSwitch:      "disable location capability exposure and remove committed location source reference",
		dataAccessed:    []string{"location"},
		createdAt:       time.Date(2026, 4, 19, 1, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateCameraCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-camera",
		capabilityName:  missioncontrol.CameraCapabilityName,
		whyNeeded:       "mission requires local shared camera-image access",
		missionFamilies: []string{"workspace"},
		risks:           []string{"local camera source exposure"},
		validators:      []string{"shared_storage exposed and committed camera source file exists and is readable"},
		killSwitch:      "disable camera capability exposure and remove committed camera source reference",
		dataAccessed:    []string{"camera"},
		createdAt:       time.Date(2026, 4, 19, 3, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateMicrophoneCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-microphone",
		capabilityName:  missioncontrol.MicrophoneCapabilityName,
		whyNeeded:       "mission requires local shared microphone-audio access",
		missionFamilies: []string{"workspace"},
		risks:           []string{"local microphone source exposure"},
		validators:      []string{"shared_storage exposed and committed microphone source file exists and is readable"},
		killSwitch:      "disable microphone capability exposure and remove committed microphone source reference",
		dataAccessed:    []string{"microphone"},
		createdAt:       time.Date(2026, 4, 19, 5, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateSMSPhoneCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-sms-phone",
		capabilityName:  missioncontrol.SMSPhoneCapabilityName,
		whyNeeded:       "mission requires local shared SMS/phone source access",
		missionFamilies: []string{"workspace"},
		risks:           []string{"local SMS/phone source exposure"},
		validators:      []string{"shared_storage exposed and committed sms_phone source file exists and is readable"},
		killSwitch:      "disable sms_phone capability exposure and remove committed sms_phone source reference",
		dataAccessed:    []string{"SMS/phone"},
		createdAt:       time.Date(2026, 4, 19, 8, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateBluetoothNFCCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-bluetooth-nfc",
		capabilityName:  missioncontrol.BluetoothNFCCapabilityName,
		whyNeeded:       "mission requires local shared Bluetooth/NFC source access",
		missionFamilies: []string{"workspace"},
		risks:           []string{"local Bluetooth/NFC source exposure"},
		validators:      []string{"shared_storage exposed and committed bluetooth_nfc source file exists and is readable"},
		killSwitch:      "disable bluetooth_nfc capability exposure and remove committed bluetooth_nfc source reference",
		dataAccessed:    []string{"Bluetooth/NFC"},
		createdAt:       time.Date(2026, 4, 19, 11, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateBroadAppControlCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	return writeTaskStateCapabilityProposalFixture(t, taskStateCapabilityProposalFixtureSpec{
		proposalID:      "proposal-broad-app-control",
		capabilityName:  missioncontrol.BroadAppControlCapabilityName,
		whyNeeded:       "mission requires local shared broad app control source access",
		missionFamilies: []string{"workspace"},
		risks:           []string{"local broad app control source exposure"},
		validators:      []string{"shared_storage exposed and committed broad_app_control source file exists and is readable"},
		killSwitch:      "disable broad_app_control capability exposure and remove committed broad_app_control source reference",
		dataAccessed:    []string{"broad app control"},
		createdAt:       time.Date(2026, 4, 19, 11, 0, 0, 0, time.UTC),
	}, state)
}

func writeTaskStateWorkspaceCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	configDir := filepath.Join(home, ".picobot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	workspace := filepath.Join(home, "workspace-root")
	configPath := filepath.Join(configDir, "config.json")
	configJSON := fmt.Sprintf(`{"agents":{"defaults":{"workspace":%q}}}`, workspace)
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}
	t.Setenv("HOME", home)
	return workspace
}

func writeTaskStateContactsCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	return writeTaskStateWorkspaceCapabilityConfigFixture(t)
}

func writeTaskStateSharedStorageCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	return writeTaskStateWorkspaceCapabilityConfigFixture(t)
}

func writeTaskStateLocationCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	return writeTaskStateWorkspaceCapabilityConfigFixture(t)
}

func writeTaskStateCameraCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	return writeTaskStateWorkspaceCapabilityConfigFixture(t)
}

func writeTaskStateMicrophoneCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	return writeTaskStateWorkspaceCapabilityConfigFixture(t)
}

func writeTaskStateSMSPhoneCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	return writeTaskStateWorkspaceCapabilityConfigFixture(t)
}

func writeTaskStateBluetoothNFCCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	return writeTaskStateWorkspaceCapabilityConfigFixture(t)
}

func writeTaskStateBroadAppControlCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	return writeTaskStateWorkspaceCapabilityConfigFixture(t)
}

func storeTaskStateSharedStorageCapabilityExposure(t *testing.T, root string, workspace string) {
	t.Helper()

	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
}
