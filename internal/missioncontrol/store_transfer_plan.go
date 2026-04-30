package missioncontrol

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type StoreTransferPlanDirection string

const (
	StoreTransferPlanDirectionBackup  StoreTransferPlanDirection = "backup"
	StoreTransferPlanDirectionRestore StoreTransferPlanDirection = "restore"
)

type StoreTransferPlan struct {
	Direction       StoreTransferPlanDirection `json:"direction"`
	DryRun          bool                       `json:"dry_run"`
	SourceRoot      string                     `json:"source_root"`
	DestinationRoot string                     `json:"destination_root"`
	Entries         []StoreTransferPlanEntry   `json:"entries"`
	TotalEntries    int                        `json:"total_entries"`
	CopyableEntries int                        `json:"copyable_entries"`
	SkippedEntries  int                        `json:"skipped_entries"`
	BlockedEntries  int                        `json:"blocked_entries"`
	CopyableBytes   int64                      `json:"copyable_bytes"`
}

type StoreTransferPlanEntry struct {
	RelPath           string `json:"relpath"`
	Kind              string `json:"kind"`
	SizeBytes         int64  `json:"size_bytes,omitempty"`
	Action            string `json:"action"`
	DestinationExists bool   `json:"destination_exists"`
	DestinationKind   string `json:"destination_kind,omitempty"`
	WouldWrite        bool   `json:"would_write"`
	WouldReplace      bool   `json:"would_replace,omitempty"`
	Blocked           bool   `json:"blocked,omitempty"`
	BlockedReason     string `json:"blocked_reason,omitempty"`
}

func ValidateStoreTransferPlanDirection(direction StoreTransferPlanDirection) error {
	switch direction {
	case StoreTransferPlanDirectionBackup, StoreTransferPlanDirectionRestore:
		return nil
	default:
		return fmt.Errorf("mission store transfer plan direction %q is invalid", strings.TrimSpace(string(direction)))
	}
}

func BuildStoreTransferPlan(missionStoreRoot, snapshotRoot string, direction StoreTransferPlanDirection) (StoreTransferPlan, error) {
	missionStoreRoot = strings.TrimSpace(missionStoreRoot)
	snapshotRoot = strings.TrimSpace(snapshotRoot)
	if err := ValidateStoreRoot(missionStoreRoot); err != nil {
		return StoreTransferPlan{}, err
	}
	if snapshotRoot == "" {
		return StoreTransferPlan{}, fmt.Errorf("mission store snapshot root is required")
	}
	if err := ValidateStoreTransferPlanDirection(direction); err != nil {
		return StoreTransferPlan{}, err
	}

	sourceRoot := missionStoreRoot
	destinationRoot := snapshotRoot
	if direction == StoreTransferPlanDirectionRestore {
		sourceRoot = snapshotRoot
		destinationRoot = missionStoreRoot
	}

	if err := validateStoreTransferPlanSourceRoot(sourceRoot); err != nil {
		return StoreTransferPlan{}, err
	}
	if err := validateStoreTransferPlanDestinationRoot(destinationRoot); err != nil {
		return StoreTransferPlan{}, err
	}

	plan := StoreTransferPlan{
		Direction:       direction,
		DryRun:          true,
		SourceRoot:      sourceRoot,
		DestinationRoot: destinationRoot,
	}

	err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceRoot {
			return nil
		}

		relPath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		info, err := entry.Info()
		if err != nil {
			return err
		}

		planEntry := buildStoreTransferPlanEntry(relPath, info, filepath.Join(destinationRoot, filepath.FromSlash(relPath)))
		plan.Entries = append(plan.Entries, planEntry)
		return nil
	})
	if err != nil {
		return StoreTransferPlan{}, err
	}

	sort.Slice(plan.Entries, func(i, j int) bool {
		return plan.Entries[i].RelPath < plan.Entries[j].RelPath
	})
	for _, entry := range plan.Entries {
		plan.TotalEntries++
		if entry.Blocked {
			plan.BlockedEntries++
			continue
		}
		if entry.WouldWrite {
			plan.CopyableEntries++
			plan.CopyableBytes += entry.SizeBytes
			continue
		}
		plan.SkippedEntries++
	}

	return plan, nil
}

func validateStoreTransferPlanSourceRoot(root string) error {
	info, err := os.Lstat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("mission store transfer plan source root %q not found", root)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("mission store transfer plan source root %q must be a directory", root)
	}
	return nil
}

func validateStoreTransferPlanDestinationRoot(root string) error {
	info, err := os.Lstat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("mission store transfer plan destination root %q must be a directory when it already exists", root)
	}
	return nil
}

func buildStoreTransferPlanEntry(relPath string, sourceInfo fs.FileInfo, destinationPath string) StoreTransferPlanEntry {
	entry := StoreTransferPlanEntry{
		RelPath: relPath,
		Kind:    storeTransferPlanKind(sourceInfo),
		Action:  "blocked_unsupported_source",
	}

	destinationInfo, destinationErr := os.Lstat(destinationPath)
	switch {
	case destinationErr == nil:
		entry.DestinationExists = true
		entry.DestinationKind = storeTransferPlanKind(destinationInfo)
	case errors.Is(destinationErr, os.ErrNotExist):
	default:
		entry.Blocked = true
		entry.BlockedReason = destinationErr.Error()
		return entry
	}

	if isStoreTransferPlanTransientRelPath(relPath) {
		entry.Action = "skip_transient_lock"
		return entry
	}

	switch {
	case sourceInfo.IsDir():
		if !entry.DestinationExists {
			entry.Action = "create_directory"
			entry.WouldWrite = true
			return entry
		}
		if destinationInfo.IsDir() {
			entry.Action = "keep_directory"
			return entry
		}
		entry.Action = "blocked_destination_conflict"
		entry.Blocked = true
		entry.BlockedReason = fmt.Sprintf("destination exists as %s", entry.DestinationKind)
		return entry
	case sourceInfo.Mode().IsRegular():
		entry.SizeBytes = sourceInfo.Size()
		if !entry.DestinationExists {
			entry.Action = "copy_file"
			entry.WouldWrite = true
			return entry
		}
		if destinationInfo.Mode().IsRegular() {
			entry.Action = "replace_file"
			entry.WouldWrite = true
			entry.WouldReplace = true
			return entry
		}
		entry.Action = "blocked_destination_conflict"
		entry.Blocked = true
		entry.BlockedReason = fmt.Sprintf("destination exists as %s", entry.DestinationKind)
		return entry
	default:
		entry.Blocked = true
		entry.BlockedReason = fmt.Sprintf("source kind %s is not supported", entry.Kind)
		return entry
	}
}

func storeTransferPlanKind(info fs.FileInfo) string {
	if info == nil {
		return ""
	}
	mode := info.Mode()
	switch {
	case mode.IsDir():
		return "directory"
	case mode.IsRegular():
		return "file"
	case mode&os.ModeSymlink != 0:
		return "symlink"
	default:
		return "special"
	}
}

func isStoreTransferPlanTransientRelPath(relPath string) bool {
	switch filepath.ToSlash(relPath) {
	case storeWriterLockFile, storeWriterGuardFile:
		return true
	default:
		return false
	}
}
