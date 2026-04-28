package missioncontrol

import "testing"

func TestFindFrozenPolicySurfaceDeclaration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		declarations      []string
		wantFound         bool
		wantDeclaration   string
		wantPolicySurface string
	}{
		{
			name:              "exact authority surface",
			declarations:      []string{"authority"},
			wantFound:         true,
			wantDeclaration:   "authority",
			wantPolicySurface: "authority",
		},
		{
			name:              "approval syntax path",
			declarations:      []string{"prompts", "approval/syntax"},
			wantFound:         true,
			wantDeclaration:   "approval/syntax",
			wantPolicySurface: "approval",
		},
		{
			name:              "treasury policy permission",
			declarations:      []string{"local_read", "treasury_policy_write"},
			wantFound:         true,
			wantDeclaration:   "treasury_policy_write",
			wantPolicySurface: "treasury",
		},
		{
			name:              "campaign stop condition",
			declarations:      []string{"campaign.stop_conditions"},
			wantFound:         true,
			wantDeclaration:   "campaign.stop_conditions",
			wantPolicySurface: "campaign",
		},
		{
			name:         "campaign template is not a policy surface",
			declarations: []string{"campaign_templates"},
		},
		{
			name:         "local extension permissions are not policy surfaces",
			declarations: []string{"local_read", "network_post"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := findFrozenPolicySurfaceDeclaration(tc.declarations)
			if ok != tc.wantFound {
				t.Fatalf("findFrozenPolicySurfaceDeclaration() found = %v, want %v", ok, tc.wantFound)
			}
			if !tc.wantFound {
				return
			}
			if got.Declaration != tc.wantDeclaration || got.PolicySurface != tc.wantPolicySurface {
				t.Fatalf("findFrozenPolicySurfaceDeclaration() = %#v, want declaration=%q policy_surface=%q", got, tc.wantDeclaration, tc.wantPolicySurface)
			}
		})
	}
}
