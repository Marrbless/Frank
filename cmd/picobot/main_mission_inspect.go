package main

import (
	"fmt"
	"strings"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
)

type missionInspectSummary = missioncontrol.InspectSummary
type missionInspectNotificationsCapability = missioncontrol.CapabilityRecord
type missionInspectSharedStorageCapability = missioncontrol.CapabilityRecord
type missionInspectContactsCapability struct {
	Capability missioncontrol.CapabilityRecord     `json:"capability"`
	Source     missioncontrol.ContactsSourceRecord `json:"source"`
}
type missionInspectLocationCapability struct {
	Capability missioncontrol.CapabilityRecord     `json:"capability"`
	Source     missioncontrol.LocationSourceRecord `json:"source"`
}
type missionInspectCameraCapability struct {
	Capability missioncontrol.CapabilityRecord   `json:"capability"`
	Source     missioncontrol.CameraSourceRecord `json:"source"`
}
type missionInspectMicrophoneCapability struct {
	Capability missioncontrol.CapabilityRecord       `json:"capability"`
	Source     missioncontrol.MicrophoneSourceRecord `json:"source"`
}
type missionInspectSMSPhoneCapability struct {
	Capability missioncontrol.CapabilityRecord     `json:"capability"`
	Source     missioncontrol.SMSPhoneSourceRecord `json:"source"`
}
type missionInspectBluetoothNFCCapability struct {
	Capability missioncontrol.CapabilityRecord         `json:"capability"`
	Source     missioncontrol.BluetoothNFCSourceRecord `json:"source"`
}
type missionInspectBroadAppControlCapability struct {
	Capability missioncontrol.CapabilityRecord            `json:"capability"`
	Source     missioncontrol.BroadAppControlSourceRecord `json:"source"`
}

func newMissionInspectSummary(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectSummary, error) {
	return missioncontrol.NewInspectSummaryWithCampaignAndTreasuryPreflight(job, stepID, storeRoot)
}

func newMissionInspectNotificationsCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectNotificationsCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectNotificationsCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectNotificationsCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresNotificationsCapability(*ec.Step) {
			return missionInspectNotificationsCapability{}, fmt.Errorf("step %q does not require notifications capability", stepID)
		}
	}

	record, err := missioncontrol.ResolveNotificationsCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectNotificationsCapability{}, err
	}
	return *record, nil
}

func newMissionInspectSharedStorageCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectSharedStorageCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectSharedStorageCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectSharedStorageCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresSharedStorageCapability(*ec.Step) {
			return missionInspectSharedStorageCapability{}, fmt.Errorf("step %q does not require shared_storage capability", stepID)
		}
	}

	record, err := missioncontrol.ResolveSharedStorageCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectSharedStorageCapability{}, err
	}
	return *record, nil
}

func newMissionInspectContactsCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectContactsCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectContactsCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectContactsCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresContactsCapability(*ec.Step) {
			return missionInspectContactsCapability{}, fmt.Errorf("step %q does not require contacts capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectContactsCapability{}, fmt.Errorf("contacts capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveContactsCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectContactsCapability{}, err
	}
	source, err := missioncontrol.RequireReadableContactsSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectContactsCapability{}, err
	}
	return missionInspectContactsCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectLocationCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectLocationCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectLocationCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectLocationCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresLocationCapability(*ec.Step) {
			return missionInspectLocationCapability{}, fmt.Errorf("step %q does not require location capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectLocationCapability{}, fmt.Errorf("location capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveLocationCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectLocationCapability{}, err
	}
	source, err := missioncontrol.RequireReadableLocationSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectLocationCapability{}, err
	}
	return missionInspectLocationCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectCameraCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectCameraCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectCameraCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectCameraCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresCameraCapability(*ec.Step) {
			return missionInspectCameraCapability{}, fmt.Errorf("step %q does not require camera capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectCameraCapability{}, fmt.Errorf("camera capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveCameraCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectCameraCapability{}, err
	}
	source, err := missioncontrol.RequireReadableCameraSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectCameraCapability{}, err
	}
	return missionInspectCameraCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectMicrophoneCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectMicrophoneCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectMicrophoneCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectMicrophoneCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresMicrophoneCapability(*ec.Step) {
			return missionInspectMicrophoneCapability{}, fmt.Errorf("step %q does not require microphone capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectMicrophoneCapability{}, fmt.Errorf("microphone capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveMicrophoneCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectMicrophoneCapability{}, err
	}
	source, err := missioncontrol.RequireReadableMicrophoneSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectMicrophoneCapability{}, err
	}
	return missionInspectMicrophoneCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectSMSPhoneCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectSMSPhoneCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectSMSPhoneCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectSMSPhoneCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresSMSPhoneCapability(*ec.Step) {
			return missionInspectSMSPhoneCapability{}, fmt.Errorf("step %q does not require sms_phone capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectSMSPhoneCapability{}, fmt.Errorf("sms_phone capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveSMSPhoneCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectSMSPhoneCapability{}, err
	}
	source, err := missioncontrol.RequireReadableSMSPhoneSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectSMSPhoneCapability{}, err
	}
	return missionInspectSMSPhoneCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectBluetoothNFCCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectBluetoothNFCCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectBluetoothNFCCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectBluetoothNFCCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresBluetoothNFCCapability(*ec.Step) {
			return missionInspectBluetoothNFCCapability{}, fmt.Errorf("step %q does not require bluetooth_nfc capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectBluetoothNFCCapability{}, fmt.Errorf("bluetooth_nfc capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveBluetoothNFCCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectBluetoothNFCCapability{}, err
	}
	source, err := missioncontrol.RequireReadableBluetoothNFCSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectBluetoothNFCCapability{}, err
	}
	return missionInspectBluetoothNFCCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectBroadAppControlCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectBroadAppControlCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectBroadAppControlCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectBroadAppControlCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresBroadAppControlCapability(*ec.Step) {
			return missionInspectBroadAppControlCapability{}, fmt.Errorf("step %q does not require broad_app_control capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectBroadAppControlCapability{}, fmt.Errorf("broad_app_control capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveBroadAppControlCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectBroadAppControlCapability{}, err
	}
	source, err := missioncontrol.RequireReadableBroadAppControlSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectBroadAppControlCapability{}, err
	}
	return missionInspectBroadAppControlCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}
