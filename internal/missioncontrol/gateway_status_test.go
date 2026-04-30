package missioncontrol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGatewayStatusSnapshotSchemaUnchanged(t *testing.T) {
	t.Parallel()

	expected := []struct {
		name string
		tag  string
	}{
		{name: "MissionRequired", tag: `json:"mission_required"`},
		{name: "Active", tag: `json:"active"`},
		{name: "MissionFile", tag: `json:"mission_file"`},
		{name: "JobID", tag: `json:"job_id"`},
		{name: "StepID", tag: `json:"step_id"`},
		{name: "StepType", tag: `json:"step_type"`},
		{name: "RequiredAuthority", tag: `json:"required_authority"`},
		{name: "RequiresApproval", tag: `json:"requires_approval"`},
		{name: "AllowedTools", tag: `json:"allowed_tools"`},
		{name: "UpdatedAt", tag: `json:"updated_at"`},
	}

	typ := reflect.TypeOf(GatewayStatusSnapshot{})
	if typ.NumField() != len(expected) {
		t.Fatalf("GatewayStatusSnapshot field count = %d, want %d", typ.NumField(), len(expected))
	}
	for i, want := range expected {
		field := typ.Field(i)
		if field.Name != want.name {
			t.Fatalf("GatewayStatusSnapshot field[%d].Name = %q, want %q", i, field.Name, want.name)
		}
		if string(field.Tag) != want.tag {
			t.Fatalf("GatewayStatusSnapshot field[%d].Tag = %q, want %q", i, string(field.Tag), want.tag)
		}
	}
}

func TestGatewayStatusGoldenFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		golden   string
		snapshot MissionStatusSnapshot
	}{
		{
			name:   "active",
			golden: "gateway_status_active.golden.json",
			snapshot: MissionStatusSnapshot{
				MissionRequired:   true,
				Active:            true,
				MissionFile:       "mission.json",
				JobID:             "job-active",
				StepID:            "build",
				StepType:          string(StepTypeOneShotCode),
				RequiredAuthority: AuthorityTierLow,
				AllowedTools:      []string{"read"},
				Runtime: &JobRuntimeState{
					JobID:        "job-active",
					State:        JobStateRunning,
					ActiveStepID: "build",
				},
				UpdatedAt: "2026-04-30T18:00:00Z",
			},
		},
		{
			name:   "paused",
			golden: "gateway_status_paused.golden.json",
			snapshot: MissionStatusSnapshot{
				MissionRequired:   true,
				Active:            false,
				MissionFile:       "mission.json",
				JobID:             "job-paused",
				StepID:            "owner-approval",
				StepType:          string(StepTypeWaitUser),
				RequiredAuthority: AuthorityTierMedium,
				RequiresApproval:  true,
				AllowedTools:      []string{"read", "reply"},
				Runtime: &JobRuntimeState{
					JobID:        "job-paused",
					State:        JobStatePaused,
					ActiveStepID: "owner-approval",
				},
				UpdatedAt: "2026-04-30T18:05:00Z",
			},
		},
		{
			name:   "failed",
			golden: "gateway_status_failed.golden.json",
			snapshot: MissionStatusSnapshot{
				MissionRequired:   true,
				Active:            false,
				MissionFile:       "mission.json",
				JobID:             "job-failed",
				StepID:            "build",
				StepType:          string(StepTypeOneShotCode),
				RequiredAuthority: AuthorityTierLow,
				AllowedTools:      []string{"read"},
				Runtime: &JobRuntimeState{
					JobID:        "job-failed",
					State:        JobStateFailed,
					ActiveStepID: "build",
				},
				UpdatedAt: "2026-04-30T18:10:00Z",
			},
		},
		{
			name:   "no-active-mission",
			golden: "gateway_status_no_active_mission.golden.json",
			snapshot: MissionStatusSnapshot{
				AllowedTools: []string{},
				UpdatedAt:    "2026-04-30T18:15:00Z",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.MarshalIndent(ProjectGatewayStatusSnapshot(tc.snapshot), "", "  ")
			if err != nil {
				t.Fatalf("MarshalIndent() error = %v", err)
			}
			data = append(data, '\n')

			want, err := os.ReadFile(filepath.Join("testdata", tc.golden))
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", tc.golden, err)
			}
			if string(data) != string(want) {
				t.Fatalf("gateway status fixture %s mismatch\ngot:\n%s\nwant:\n%s", tc.golden, string(data), string(want))
			}
		})
	}
}

func TestWriteGatewayStatusSnapshotAtomicAndProjectedShareAtomicWriter(t *testing.T) {
	storeFixture := writeMissionStoreRuntimeFixture(t)
	root := storeFixture.root
	now := storeFixture.now
	job := storeFixture.job

	originalWrite := missionStatusSnapshotWriteFileAtomic
	t.Cleanup(func() { missionStatusSnapshotWriteFileAtomic = originalWrite })

	calls := make([]string, 0, 2)
	missionStatusSnapshotWriteFileAtomic = func(path string, data []byte) error {
		calls = append(calls, path)
		if len(data) == 0 {
			t.Fatal("atomic writer data = empty, want encoded snapshot bytes")
		}
		if strings.Contains(string(data), `"runtime_control"`) {
			t.Fatalf("atomic writer data = %q, want sanitized gateway envelope without runtime_control", string(data))
		}
		return nil
	}

	livePath := filepath.Join(t.TempDir(), "gateway-status-live.json")
	if err := WriteGatewayStatusSnapshotAtomic(livePath, GatewayStatusSnapshot{
		MissionRequired: true,
		Active:          true,
		MissionFile:     "mission.json",
		JobID:           job.ID,
		StepID:          "build",
		StepType:        string(StepTypeOneShotCode),
		AllowedTools:    []string{"read"},
		UpdatedAt:       now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("WriteGatewayStatusSnapshotAtomic() error = %v", err)
	}

	projectedPath := filepath.Join(t.TempDir(), "gateway-status-projected.json")
	if err := WriteProjectedGatewayStatusSnapshot(projectedPath, root, job.ID, GatewayStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("WriteProjectedGatewayStatusSnapshot() error = %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("atomic writer calls = %d, want 2", len(calls))
	}
	if calls[0] != livePath {
		t.Fatalf("atomic writer live path = %q, want %q", calls[0], livePath)
	}
	if calls[1] != projectedPath {
		t.Fatalf("atomic writer projected path = %q, want %q", calls[1], projectedPath)
	}
}

func TestLoadGatewayStatusObservationAndFileUseMissionStatusReadPath(t *testing.T) {
	originalRead := missionStatusSnapshotReadFile
	originalDecode := missionStatusSnapshotDecode
	t.Cleanup(func() {
		missionStatusSnapshotReadFile = originalRead
		missionStatusSnapshotDecode = originalDecode
	})

	missionStatusSnapshotReadFile = func(path string) ([]byte, error) {
		return []byte(`{"job_id":"job-1"}`), nil
	}

	calls := 0
	missionStatusSnapshotDecode = func(path string, data []byte) (MissionStatusSnapshot, error) {
		calls++
		if path != "gateway-status.json" {
			t.Fatalf("decode helper path = %q, want %q", path, "gateway-status.json")
		}
		return MissionStatusSnapshot{
			MissionRequired: true,
			Active:          true,
			MissionFile:     "mission.json",
			JobID:           "job-1",
			StepID:          "build",
			StepType:        string(StepTypeOneShotCode),
			AllowedTools:    []string{"read"},
			Runtime: &JobRuntimeState{
				JobID:        "job-1",
				State:        JobStatePaused,
				ActiveStepID: "build",
			},
			RuntimeSummary: &OperatorStatusSummary{
				JobID:        "job-1",
				State:        JobStatePaused,
				ActiveStepID: "build",
				Artifacts: []OperatorArtifactStatus{
					{StepID: "build", Path: "/tmp/secret.txt"},
				},
			},
			RuntimeControl: &RuntimeControlContext{
				JobID: "job-1",
				Step:  Step{ID: "build"},
			},
			UpdatedAt: "2026-04-12T12:00:00Z",
		}, nil
	}

	observation, err := LoadGatewayStatusObservation("gateway-status.json")
	if err != nil {
		t.Fatalf("LoadGatewayStatusObservation() error = %v", err)
	}
	if observation.JobID != "job-1" {
		t.Fatalf("LoadGatewayStatusObservation().JobID = %q, want %q", observation.JobID, "job-1")
	}

	data, err := LoadGatewayStatusObservationFile("gateway-status.json")
	if err != nil {
		t.Fatalf("LoadGatewayStatusObservationFile() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("shared decode helper calls = %d, want 2", calls)
	}
	got := string(data)
	if strings.Contains(got, `"runtime"`) {
		t.Fatalf("LoadGatewayStatusObservationFile() = %q, want sanitized gateway envelope without runtime", got)
	}
	if strings.Contains(got, `"runtime_summary"`) {
		t.Fatalf("LoadGatewayStatusObservationFile() = %q, want sanitized gateway envelope without runtime_summary", got)
	}
	if strings.Contains(got, `"deferred_scheduler_triggers"`) {
		t.Fatalf("LoadGatewayStatusObservationFile() = %q, want canonical gateway envelope without deferred scheduler trigger visibility", got)
	}
	if strings.Contains(got, `"runtime_control"`) {
		t.Fatalf("LoadGatewayStatusObservationFile() = %q, want sanitized gateway envelope without runtime_control", got)
	}
	if strings.Contains(got, `/tmp/secret.txt`) {
		t.Fatalf("LoadGatewayStatusObservationFile() = %q, want sanitized gateway envelope without artifact paths", got)
	}
	if !strings.Contains(got, `"job_id": "job-1"`) {
		t.Fatalf("LoadGatewayStatusObservationFile() = %q, want projected gateway job_id", got)
	}
}
