package missioncontrol

import (
	"reflect"
	"testing"
)

func TestClassifyKnownToolRiskCoversBuiltInAgentToolInventory(t *testing.T) {
	t.Parallel()

	builtInTools := []string{
		"message",
		"frank_zoho_send_email",
		"frank_zoho_manage_reply_work_item",
		"filesystem",
		"exec",
		"web",
		"web_search",
		"cron",
		"write_memory",
		"list_memory",
		"read_memory",
		"edit_memory",
		"delete_memory",
		"create_skill",
		"list_skills",
		"read_skill",
		"delete_skill",
	}

	for _, name := range builtInTools {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			profile, ok := ClassifyKnownToolRisk(name)
			if !ok {
				t.Fatalf("ClassifyKnownToolRisk(%q) ok = false, want true", name)
			}
			if profile.ToolName != name {
				t.Fatalf("ToolName = %q, want %q", profile.ToolName, name)
			}
			if profile.Evidence == "" {
				t.Fatalf("Evidence is empty for %q", name)
			}
		})
	}
}

func TestClassifyKnownToolRiskLeavesDynamicMCPToolsUnclassified(t *testing.T) {
	t.Parallel()

	if profile, ok := ClassifyKnownToolRisk("mcp_mail_send"); ok {
		t.Fatalf("ClassifyKnownToolRisk(mcp_mail_send) = %#v, true; want unclassified", profile)
	}
	if profile, ok := ClassifyKnownToolRisk("unknown_tool"); ok {
		t.Fatalf("ClassifyKnownToolRisk(unknown_tool) = %#v, true; want unclassified", profile)
	}
}

func TestClassifyKnownToolRiskFlagsRepresentativeClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want ToolRiskProfile
	}{
		{
			name: "exec",
			want: ToolRiskProfile{ToolName: "exec", Exec: true, Write: true, SideEffect: true},
		},
		{
			name: "web_search",
			want: ToolRiskProfile{ToolName: "web_search", ReadOnly: true, Network: true},
		},
		{
			name: "read_memory",
			want: ToolRiskProfile{ToolName: "read_memory", ReadOnly: true, Filesystem: true},
		},
		{
			name: "frank_zoho_send_email",
			want: ToolRiskProfile{ToolName: "frank_zoho_send_email", Network: true, Account: true, Write: true, SideEffect: true},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ClassifyKnownToolRisk(tt.name)
			if !ok {
				t.Fatalf("ClassifyKnownToolRisk(%q) ok = false, want true", tt.name)
			}
			got.Evidence = ""
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ClassifyKnownToolRisk(%q) = %#v, want %#v", tt.name, got, tt.want)
			}
		})
	}
}

func TestToolRiskProfileDangerousForLowAuthority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{name: "exec", want: true},
		{name: "web", want: true},
		{name: "read_memory", want: true},
		{name: "list_memory", want: true},
	}

	for _, tt := range tests {
		profile, ok := ClassifyKnownToolRisk(tt.name)
		if !ok {
			t.Fatalf("ClassifyKnownToolRisk(%q) ok = false, want true", tt.name)
		}
		if got := profile.DangerousForLowAuthority(); got != tt.want {
			t.Fatalf("%s DangerousForLowAuthority() = %t, want %t", tt.name, got, tt.want)
		}
	}
}

func TestKnownToolRiskProfilesStableOrder(t *testing.T) {
	t.Parallel()

	got := KnownToolRiskProfiles()
	if len(got) == 0 {
		t.Fatal("KnownToolRiskProfiles() returned no profiles")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].ToolName > got[i].ToolName {
			t.Fatalf("KnownToolRiskProfiles() not sorted at %d: %q > %q", i, got[i-1].ToolName, got[i].ToolName)
		}
	}
}
