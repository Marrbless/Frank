package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestAssessRuntimeExtensionPermissionWideningAllowsCompatibleSamePermissions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	storeRuntimeExtensionPermissionFixture(t, root, now, nil)

	assessment, err := AssessRuntimeExtensionPermissionWidening(root, "extension-pack-base", "extension-pack-candidate")
	if err != nil {
		t.Fatalf("AssessRuntimeExtensionPermissionWidening() error = %v", err)
	}
	if assessment.State != RuntimeExtensionPermissionAssessmentStateAllowed {
		t.Fatalf("AssessRuntimeExtensionPermissionWidening().State = %q blockers=%#v, want allowed", assessment.State, assessment.Blockers)
	}
	if len(assessment.Blockers) != 0 || len(assessment.WidenedPermissions) != 0 || len(assessment.NewExternalTools) != 0 {
		t.Fatalf("AssessRuntimeExtensionPermissionWidening() = %#v, want no blockers", assessment)
	}
}

func TestAssessRuntimeExtensionPermissionWideningBlocksNewPermissionsAndExternalTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		edit       func(*RuntimeExtensionPackRecord)
		wantCode   RejectionCode
		wantReason string
	}{
		{
			name: "new permission",
			edit: func(record *RuntimeExtensionPackRecord) {
				record.DeclaredPermissions = []string{"local_read", "network_post"}
			},
			wantCode:   RejectionCodeV4ExtensionPermissionWidening,
			wantReason: `new permission "network_post"`,
		},
		{
			name: "new external side effect",
			edit: func(record *RuntimeExtensionPackRecord) {
				record.ExternalSideEffects = []string{"network_post"}
			},
			wantCode:   RejectionCodeV4ExtensionPermissionWidening,
			wantReason: `new external side effect "network_post"`,
		},
		{
			name: "new external side-effect tool",
			edit: func(record *RuntimeExtensionPackRecord) {
				record.DeclaredTools = append(record.DeclaredTools, RuntimeExtensionToolDeclaration{
					ToolName:           "post_public_update",
					PermissionRefs:     []string{"network_post"},
					ExternalSideEffect: true,
				})
				record.DeclaredPermissions = []string{"local_read", "network_post"}
			},
			wantCode:   RejectionCodeV4ExtensionPermissionWidening,
			wantReason: `external side-effect tool "post_public_update"`,
		},
		{
			name: "compatibility contract change",
			edit: func(record *RuntimeExtensionPackRecord) {
				record.CompatibilityContractRef = "compat-v2"
			},
			wantCode:   RejectionCodeV4ExtensionCompatibilityRequired,
			wantReason: `compatibility_contract_ref "compat-v2"`,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 10, 13, 0, 0, 0, time.UTC)
			storeRuntimeExtensionPermissionFixture(t, root, now, tc.edit)

			assessment, err := AssessRuntimeExtensionPermissionWidening(root, "extension-pack-base", "extension-pack-candidate")
			if err != nil {
				t.Fatalf("AssessRuntimeExtensionPermissionWidening() error = %v", err)
			}
			if assessment.State != RuntimeExtensionPermissionAssessmentStateBlocked {
				t.Fatalf("AssessRuntimeExtensionPermissionWidening().State = %q, want blocked", assessment.State)
			}
			found := false
			for _, blocker := range assessment.Blockers {
				if blocker.Code == tc.wantCode && strings.Contains(blocker.Reason, tc.wantReason) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("AssessRuntimeExtensionPermissionWidening().Blockers = %#v, want %s with %q", assessment.Blockers, tc.wantCode, tc.wantReason)
			}
		})
	}
}

func storeRuntimeExtensionPermissionFixture(t *testing.T, root string, now time.Time, candidateEdit func(*RuntimeExtensionPackRecord)) {
	t.Helper()

	if _, _, err := StoreRuntimeExtensionPackRecord(root, validRuntimeExtensionPackRecord(now, "extension-pack-base", nil)); err != nil {
		t.Fatalf("StoreRuntimeExtensionPackRecord(base) error = %v", err)
	}
	if _, _, err := StoreRuntimeExtensionPackRecord(root, validRuntimeExtensionPackRecord(now.Add(time.Minute), "extension-pack-candidate", func(record *RuntimeExtensionPackRecord) {
		record.ParentExtensionPackID = "extension-pack-base"
		record.ChangeSummary = "candidate extension update"
		if candidateEdit != nil {
			candidateEdit(record)
		}
	})); err != nil {
		t.Fatalf("StoreRuntimeExtensionPackRecord(candidate) error = %v", err)
	}
}
