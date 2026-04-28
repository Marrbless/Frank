package missioncontrol

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAssessRuntimePackComponentAdmissionForHotUpdateAdmitsDeclaredMutableSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	storeRuntimePackComponentAdmissionFixture(t, root, now, nil, nil, nil)

	statuses, err := AssessRuntimePackComponentAdmissionForHotUpdate(root, "hot-update-components")
	if err != nil {
		t.Fatalf("AssessRuntimePackComponentAdmissionForHotUpdate() error = %v", err)
	}
	if len(statuses) != 4 {
		t.Fatalf("AssessRuntimePackComponentAdmissionForHotUpdate() len = %d, want 4", len(statuses))
	}
	for _, status := range statuses {
		if status.State != RuntimePackComponentAdmissionStateAdmitted {
			t.Fatalf("component %s/%s state = %q reason=%q, want admitted", status.Kind, status.ComponentID, status.State, status.Reason)
		}
		if status.HotUpdateID != "hot-update-components" || status.CandidatePackID != "pack-candidate" {
			t.Fatalf("component status identity = %#v, want hot-update-components/pack-candidate", status)
		}
		if len(status.DeclaredSurfaces) != 1 {
			t.Fatalf("component %s declared surfaces = %#v, want one", status.ComponentID, status.DeclaredSurfaces)
		}
	}
	if err := RequireRuntimePackComponentAdmissionForHotUpdate(root, "hot-update-components"); err != nil {
		t.Fatalf("RequireRuntimePackComponentAdmissionForHotUpdate() error = %v", err)
	}
}

func TestAssessRuntimePackComponentAdmissionBlocksUndeclaredAndImmutableSurfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		componentEdit func(*RuntimePackComponentRecord)
		packEdit      func(*RuntimePackRecord)
		gateEdit      func(*HotUpdateGateRecord)
		wantCode      RejectionCode
		wantReason    string
	}{
		{
			name: "surface not in candidate mutable surfaces",
			componentEdit: func(record *RuntimePackComponentRecord) {
				if record.Kind == RuntimePackComponentKindPromptPack {
					record.DeclaredSurfaces = []string{"authority"}
				}
			},
			wantCode:   RejectionCodeV4ForbiddenSurfaceChange,
			wantReason: `declared surface "authority" is immutable`,
		},
		{
			name: "surface not targeted by gate",
			componentEdit: func(record *RuntimePackComponentRecord) {
				if record.Kind == RuntimePackComponentKindSkillPack {
					record.DeclaredSurfaces = []string{"skills"}
				}
			},
			gateEdit: func(record *HotUpdateGateRecord) {
				record.TargetSurfaces = []string{"prompts", "routing", "extensions"}
			},
			wantCode:   RejectionCodeV4MutationScopeViolation,
			wantReason: `declared surface "skills" is not targeted`,
		},
		{
			name: "missing surface class",
			componentEdit: func(record *RuntimePackComponentRecord) {
				if record.Kind == RuntimePackComponentKindManifestPack {
					record.SurfaceClass = " "
				}
			},
			wantCode:   RejectionCodeV4SurfaceClassRequired,
			wantReason: "surface_class is required",
		},
		{
			name: "not hot reloadable",
			componentEdit: func(record *RuntimePackComponentRecord) {
				if record.Kind == RuntimePackComponentKindExtensionPack {
					record.HotReloadable = false
				}
			},
			wantCode:   RejectionCodeV4ReloadModeUnsupported,
			wantReason: "not hot_reloadable",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC)
			storeRuntimePackComponentAdmissionFixture(t, root, now, tc.componentEdit, tc.packEdit, tc.gateEdit)

			statuses, err := AssessRuntimePackComponentAdmissionForHotUpdate(root, "hot-update-components")
			if err != nil {
				t.Fatalf("AssessRuntimePackComponentAdmissionForHotUpdate() error = %v", err)
			}
			var blocked RuntimePackComponentAdmissionStatus
			for _, status := range statuses {
				if status.State == RuntimePackComponentAdmissionStateBlocked {
					blocked = status
					break
				}
			}
			if blocked.State != RuntimePackComponentAdmissionStateBlocked {
				t.Fatalf("statuses = %#v, want one blocked component", statuses)
			}
			if blocked.RejectionCode != tc.wantCode {
				t.Fatalf("blocked rejection_code = %q, want %q", blocked.RejectionCode, tc.wantCode)
			}
			if !strings.Contains(blocked.Reason, tc.wantReason) {
				t.Fatalf("blocked reason = %q, want substring %q", blocked.Reason, tc.wantReason)
			}
			if err := RequireRuntimePackComponentAdmissionForHotUpdate(root, "hot-update-components"); err == nil {
				t.Fatal("RequireRuntimePackComponentAdmissionForHotUpdate() error = nil, want blocked admission")
			} else if !strings.Contains(err.Error(), string(tc.wantCode)) {
				t.Fatalf("RequireRuntimePackComponentAdmissionForHotUpdate() error = %q, want code %s", err.Error(), tc.wantCode)
			}
		})
	}
}

func TestAssessRuntimePackComponentAdmissionRequiresHotUpdateGateContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := AssessRuntimePackComponentAdmissionForHotUpdate(root, "missing-hot-update"); err == nil {
		t.Fatal("AssessRuntimePackComponentAdmissionForHotUpdate(missing gate) error = nil, want fail-closed rejection")
	} else if !errors.Is(err, ErrHotUpdateGateRecordNotFound) {
		t.Fatalf("AssessRuntimePackComponentAdmissionForHotUpdate(missing gate) error = %v, want %v", err, ErrHotUpdateGateRecordNotFound)
	} else if !strings.Contains(err.Error(), `hot_update_id "missing-hot-update"`) {
		t.Fatalf("AssessRuntimePackComponentAdmissionForHotUpdate(missing gate) error = %q, want hot-update context", err.Error())
	}
}

func storeRuntimePackComponentAdmissionFixture(t *testing.T, root string, now time.Time, componentEdit func(*RuntimePackComponentRecord), packEdit func(*RuntimePackRecord), gateEdit func(*HotUpdateGateRecord)) {
	t.Helper()

	base := validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
		record.MutableSurfaces = []string{"prompts", "skills", "routing", "extensions"}
	})
	if err := StoreRuntimePackRecord(root, base); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-base) error = %v", err)
	}
	if _, _, err := StoreRuntimeExtensionPackRecord(root, validRuntimeExtensionPackRecord(now, "extension-pack-root", nil)); err != nil {
		t.Fatalf("StoreRuntimeExtensionPackRecord(extension-pack-root) error = %v", err)
	}
	candidate := validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.PromptPackRef = "prompt-pack-candidate"
		record.SkillPackRef = "skill-pack-candidate"
		record.ManifestRef = "manifest-pack-candidate"
		record.ExtensionPackRef = "extension-pack-candidate"
		record.MutableSurfaces = []string{"prompts", "skills", "routing", "extensions"}
		record.SurfaceClasses = []string{"class_1"}
		if packEdit != nil {
			packEdit(record)
		}
	})
	if err := StoreRuntimePackRecord(root, candidate); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-candidate) error = %v", err)
	}
	if _, _, err := StoreRuntimeExtensionPackRecord(root, validRuntimeExtensionPackRecord(now.Add(time.Minute), "extension-pack-candidate", nil)); err != nil {
		t.Fatalf("StoreRuntimeExtensionPackRecord(extension-pack-candidate) error = %v", err)
	}

	storeRuntimePackAdmissionComponent(t, root, now, RuntimePackComponentKindPromptPack, "prompt-pack-candidate", "prompts", componentEdit)
	storeRuntimePackAdmissionComponent(t, root, now, RuntimePackComponentKindSkillPack, "skill-pack-candidate", "skills", componentEdit)
	storeRuntimePackAdmissionComponent(t, root, now, RuntimePackComponentKindManifestPack, "manifest-pack-candidate", "routing", componentEdit)
	storeRuntimePackAdmissionComponent(t, root, now, RuntimePackComponentKindExtensionPack, "extension-pack-candidate", "extensions", componentEdit)

	gate := HotUpdateGateRecord{
		HotUpdateID:              "hot-update-components",
		Objective:                "admit candidate components",
		CandidatePackID:          "pack-candidate",
		PreviousActivePackID:     "pack-base",
		RollbackTargetPackID:     "pack-base",
		TargetSurfaces:           []string{"prompts", "skills", "routing", "extensions"},
		SurfaceClasses:           []string{"class_1"},
		ReloadMode:               HotUpdateReloadModePackReload,
		CompatibilityContractRef: "compat-v1",
		PreparedAt:               now.Add(2 * time.Minute),
		PhaseUpdatedAt:           now.Add(2 * time.Minute),
		PhaseUpdatedBy:           "operator",
		State:                    HotUpdateGateStatePrepared,
		Decision:                 HotUpdateGateDecisionKeepStaged,
	}
	if gateEdit != nil {
		gateEdit(&gate)
	}
	if err := StoreHotUpdateGateRecord(root, gate); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
}

func storeRuntimePackAdmissionComponent(t *testing.T, root string, now time.Time, kind RuntimePackComponentKind, componentID string, surface string, edit func(*RuntimePackComponentRecord)) {
	t.Helper()

	record := validRuntimePackComponentRecord(now, kind, componentID, func(record *RuntimePackComponentRecord) {
		record.SurfaceClass = "class_1"
		record.HotReloadable = true
		record.DeclaredSurfaces = []string{surface}
		if edit != nil {
			edit(record)
		}
	})
	if _, _, err := StoreRuntimePackComponentRecord(root, record); err != nil {
		t.Fatalf("StoreRuntimePackComponentRecord(%s/%s) error = %v", kind, componentID, err)
	}
}
