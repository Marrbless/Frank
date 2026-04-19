package tools

import (
	"testing"

	"github.com/local/picobot/internal/missioncontrol"
)

func writeMalformedTreasuryRecordForTaskStateReadoutTest(t *testing.T, root string, treasury missioncontrol.TreasuryRecord) {
	t.Helper()

	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreTreasuryPath(root, treasury.TreasuryID), map[string]interface{}{
		"record_version":   treasury.RecordVersion,
		"treasury_id":      treasury.TreasuryID,
		"display_name":     treasury.DisplayName,
		"state":            string(treasury.State),
		"zero_seed_policy": string(treasury.ZeroSeedPolicy),
		"container_refs": []map[string]interface{}{
			{
				"kind":      string(treasury.ContainerRefs[0].Kind),
				"object_id": treasury.ContainerRefs[0].ObjectID,
			},
		},
		"created_at": treasury.CreatedAt,
		"updated_at": treasury.UpdatedAt,
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}
}
