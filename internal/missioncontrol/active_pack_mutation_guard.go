package missioncontrol

import (
	"errors"
	"fmt"
)

func rejectRuntimePackComponentActiveAdhocMutation(root string, record RuntimePackComponentRecord) error {
	pointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		if errors.Is(err, ErrActiveRuntimePackPointerNotFound) {
			return nil
		}
		return err
	}
	activePack, err := LoadRuntimePackRecord(root, pointer.ActivePackID)
	if err != nil {
		return fmt.Errorf("mission store runtime pack component active_pack_id %q: %w", pointer.ActivePackID, err)
	}
	for _, ref := range RuntimePackComponentRefs(activePack) {
		if ref.Kind == record.Kind && ref.ComponentID == record.ComponentID {
			return fmt.Errorf("mission store runtime pack component %s/%s rejected: %s: active_pack_id %q references this component; mutate it through hot-update or rollback gate records", record.Kind, record.ComponentID, RejectionCodeV4ActivePackAdhocMutationForbidden, pointer.ActivePackID)
		}
	}
	return nil
}

func rejectPackageImportActiveAdhocMutation(root string, record PackageImportRecord) error {
	pointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		if errors.Is(err, ErrActiveRuntimePackPointerNotFound) {
			return nil
		}
		return err
	}
	if pointer.ActivePackID == record.CandidatePackID {
		return fmt.Errorf("mission store package import %q rejected: %s: candidate_pack_id %q is the current active_pack_id; package imports must remain candidate-only and enter through hot-update promotion", record.ImportID, RejectionCodeV4ActivePackAdhocMutationForbidden, record.CandidatePackID)
	}
	return nil
}
