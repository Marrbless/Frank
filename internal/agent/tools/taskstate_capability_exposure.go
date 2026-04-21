package tools

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
)

func (s *TaskState) applyNotificationsCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.notificationsCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresNotificationsCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedNotificationsCapabilityOnboardingProposal(ec); err != nil {
		return err
	}

	record, err := missioncontrol.ResolveNotificationsCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		return nil
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	_, err = missioncontrol.RequireExposedNotificationsCapabilityRecord(root)
	return err
}

func defaultNotificationsCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedNotificationsCapabilityOnboardingProposal(ec); err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("notifications capability exposure requires readable config: %w", err)
	}
	if !cfg.Channels.Telegram.Enabled || strings.TrimSpace(cfg.Channels.Telegram.Token) == "" || len(cfg.Channels.Telegram.AllowFrom) == 0 {
		return fmt.Errorf("notifications capability exposure requires configured Telegram owner-control channel")
	}

	_, err = missioncontrol.StoreTelegramNotificationsCapabilityExposure(root)
	return err
}

func (s *TaskState) applySharedStorageCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.sharedStorageCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresSharedStorageCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedSharedStorageCapabilityOnboardingProposal(ec); err != nil {
		return err
	}

	record, err := missioncontrol.ResolveSharedStorageCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		return nil
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	_, err = missioncontrol.RequireExposedSharedStorageCapabilityRecord(root)
	return err
}

func defaultSharedStorageCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedSharedStorageCapabilityOnboardingProposal(ec); err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("shared_storage capability exposure requires readable config: %w", err)
	}

	_, err = missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, cfg.Agents.Defaults.Workspace)
	return err
}

func (s *TaskState) applyContactsCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.contactsCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresContactsCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedContactsCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("contacts capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("contacts capability exposure requires readable config: %w", err)
	}

	record, err := missioncontrol.ResolveContactsCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		_, err = missioncontrol.RequireReadableContactsSourceRecord(root, cfg.Agents.Defaults.Workspace)
		return err
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	if _, err := missioncontrol.RequireExposedContactsCapabilityRecord(root); err != nil {
		return err
	}
	_, err = missioncontrol.RequireReadableContactsSourceRecord(root, cfg.Agents.Defaults.Workspace)
	return err
}

func defaultContactsCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedContactsCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("contacts capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("contacts capability exposure requires readable config: %w", err)
	}

	_, err = missioncontrol.StoreWorkspaceContactsCapabilityExposure(root, cfg.Agents.Defaults.Workspace)
	return err
}

func (s *TaskState) applyLocationCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.locationCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresLocationCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedLocationCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("location capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("location capability exposure requires readable config: %w", err)
	}

	record, err := missioncontrol.ResolveLocationCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		_, err = missioncontrol.RequireReadableLocationSourceRecord(root, cfg.Agents.Defaults.Workspace)
		return err
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	if _, err := missioncontrol.RequireExposedLocationCapabilityRecord(root); err != nil {
		return err
	}
	_, err = missioncontrol.RequireReadableLocationSourceRecord(root, cfg.Agents.Defaults.Workspace)
	return err
}

func defaultLocationCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedLocationCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("location capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("location capability exposure requires readable config: %w", err)
	}

	_, err = missioncontrol.StoreWorkspaceLocationCapabilityExposure(root, cfg.Agents.Defaults.Workspace)
	return err
}

func (s *TaskState) applyCameraCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.cameraCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresCameraCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedCameraCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("camera capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("camera capability exposure requires readable config: %w", err)
	}

	record, err := missioncontrol.ResolveCameraCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		_, err = missioncontrol.RequireReadableCameraSourceRecord(root, cfg.Agents.Defaults.Workspace)
		return err
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	if _, err := missioncontrol.RequireExposedCameraCapabilityRecord(root); err != nil {
		return err
	}
	_, err = missioncontrol.RequireReadableCameraSourceRecord(root, cfg.Agents.Defaults.Workspace)
	return err
}

func defaultCameraCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedCameraCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("camera capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("camera capability exposure requires readable config: %w", err)
	}

	_, err = missioncontrol.StoreWorkspaceCameraCapabilityExposure(root, cfg.Agents.Defaults.Workspace)
	return err
}

func (s *TaskState) applyMicrophoneCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.microphoneCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresMicrophoneCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedMicrophoneCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("microphone capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("microphone capability exposure requires readable config: %w", err)
	}

	record, err := missioncontrol.ResolveMicrophoneCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		_, err = missioncontrol.RequireReadableMicrophoneSourceRecord(root, cfg.Agents.Defaults.Workspace)
		return err
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	if _, err := missioncontrol.RequireExposedMicrophoneCapabilityRecord(root); err != nil {
		return err
	}
	_, err = missioncontrol.RequireReadableMicrophoneSourceRecord(root, cfg.Agents.Defaults.Workspace)
	return err
}

func defaultMicrophoneCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedMicrophoneCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("microphone capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("microphone capability exposure requires readable config: %w", err)
	}

	_, err = missioncontrol.StoreWorkspaceMicrophoneCapabilityExposure(root, cfg.Agents.Defaults.Workspace)
	return err
}

func (s *TaskState) applySMSPhoneCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.smsPhoneCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresSMSPhoneCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedSMSPhoneCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("sms_phone capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("sms_phone capability exposure requires readable config: %w", err)
	}

	record, err := missioncontrol.ResolveSMSPhoneCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		_, err = missioncontrol.RequireReadableSMSPhoneSourceRecord(root, cfg.Agents.Defaults.Workspace)
		return err
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	if _, err := missioncontrol.RequireExposedSMSPhoneCapabilityRecord(root); err != nil {
		return err
	}
	_, err = missioncontrol.RequireReadableSMSPhoneSourceRecord(root, cfg.Agents.Defaults.Workspace)
	return err
}

func defaultSMSPhoneCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedSMSPhoneCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("sms_phone capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("sms_phone capability exposure requires readable config: %w", err)
	}

	_, err = missioncontrol.StoreWorkspaceSMSPhoneCapabilityExposure(root, cfg.Agents.Defaults.Workspace)
	return err
}

func (s *TaskState) applyBluetoothNFCCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.bluetoothNFCCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresBluetoothNFCCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedBluetoothNFCCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("bluetooth_nfc capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("bluetooth_nfc capability exposure requires readable config: %w", err)
	}

	record, err := missioncontrol.ResolveBluetoothNFCCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		_, err = missioncontrol.RequireReadableBluetoothNFCSourceRecord(root, cfg.Agents.Defaults.Workspace)
		return err
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	if _, err := missioncontrol.RequireExposedBluetoothNFCCapabilityRecord(root); err != nil {
		return err
	}
	_, err = missioncontrol.RequireReadableBluetoothNFCSourceRecord(root, cfg.Agents.Defaults.Workspace)
	return err
}

func defaultBluetoothNFCCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedBluetoothNFCCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("bluetooth_nfc capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("bluetooth_nfc capability exposure requires readable config: %w", err)
	}

	_, err = missioncontrol.StoreWorkspaceBluetoothNFCCapabilityExposure(root, cfg.Agents.Defaults.Workspace)
	return err
}

func (s *TaskState) applyBroadAppControlCapabilityForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.broadAppControlCapabilityHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.StepRequiresBroadAppControlCapability(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	if _, err := missioncontrol.RequireApprovedBroadAppControlCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("broad_app_control capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("broad_app_control capability exposure requires readable config: %w", err)
	}

	record, err := missioncontrol.ResolveBroadAppControlCapabilityRecord(root)
	switch {
	case err == nil && record.Exposed:
		_, err = missioncontrol.RequireReadableBroadAppControlSourceRecord(root, cfg.Agents.Defaults.Workspace)
		return err
	case err == nil:
	case errors.Is(err, missioncontrol.ErrCapabilityRecordNotFound):
	default:
		return err
	}

	if hook != nil {
		if err := hook(root, ec, now); err != nil {
			return err
		}
	}

	if _, err := missioncontrol.RequireExposedBroadAppControlCapabilityRecord(root); err != nil {
		return err
	}
	_, err = missioncontrol.RequireReadableBroadAppControlSourceRecord(root, cfg.Agents.Defaults.Workspace)
	return err
}

func defaultBroadAppControlCapabilityExposureHook(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
	_ = now

	if _, err := missioncontrol.RequireApprovedBroadAppControlCapabilityOnboardingProposal(ec); err != nil {
		return err
	}
	if _, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return fmt.Errorf("broad_app_control capability requires shared_storage exposure: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("broad_app_control capability exposure requires readable config: %w", err)
	}

	_, err = missioncontrol.StoreWorkspaceBroadAppControlCapabilityExposure(root, cfg.Agents.Defaults.Workspace)
	return err
}
