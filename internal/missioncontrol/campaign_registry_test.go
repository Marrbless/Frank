package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCampaignRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 21, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	if err := StoreCampaignRecord(root, CampaignRecord{
		CampaignID:   "campaign-b",
		CampaignKind: CampaignKindCommunity,
		DisplayName:  "Community B",
		State:        CampaignStateDraft,
		Objective:    "Join communities without posting yet",
		GovernedExternalTargets: []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindPlatform,
				RegistryID: "community-platform-b",
			},
		},
		FrankObjectRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: "identity-community-b",
			},
		},
		IdentityMode:     IdentityModeAgentAlias,
		StopConditions:   []string{"stop after first accepted intro"},
		FailureThreshold: CampaignFailureThreshold{Metric: "rejections", Limit: 2},
		ComplianceChecks: []string{"community_rules_reviewed"},
		CreatedAt:        now,
		UpdatedAt:        now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreCampaignRecord(campaign-b) error = %v", err)
	}

	want := CampaignRecord{
		CampaignID:   "campaign-a",
		CampaignKind: CampaignKind(" outreach "),
		DisplayName:  " Outreach A ",
		State:        CampaignState(" active "),
		Objective:    " Reach aligned operators ",
		GovernedExternalTargets: []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: " provider-mail ",
			},
		},
		FrankObjectRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKind(" identity "),
				ObjectID: " identity-mail ",
			},
		},
		IdentityMode:     IdentityMode(" "),
		StopConditions:   []string{" stop after 3 replies "},
		FailureThreshold: CampaignFailureThreshold{Metric: " bounced_messages ", Limit: 3},
		ComplianceChecks: []string{" can-spam-reviewed "},
		CreatedAt:        now.Add(2 * time.Minute),
		UpdatedAt:        now.Add(3 * time.Minute),
	}
	if err := StoreCampaignRecord(root, want); err != nil {
		t.Fatalf("StoreCampaignRecord(campaign-a) error = %v", err)
	}

	got, err := LoadCampaignRecord(root, "campaign-a")
	if err != nil {
		t.Fatalf("LoadCampaignRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.CampaignKind = CampaignKindOutreach
	want.DisplayName = "Outreach A"
	want.State = CampaignStateActive
	want.Objective = "Reach aligned operators"
	want.GovernedExternalTargets = []AutonomyEligibilityTargetRef{
		{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-mail",
		},
	}
	want.FrankObjectRefs = []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: "identity-mail",
		},
	}
	want.IdentityMode = IdentityModeAgentAlias
	want.StopConditions = []string{"stop after 3 replies"}
	want.FailureThreshold = CampaignFailureThreshold{Metric: "bounced_messages", Limit: 3}
	want.ComplianceChecks = []string{"can-spam-reviewed"}
	want.CreatedAt = want.CreatedAt.UTC()
	want.UpdatedAt = want.UpdatedAt.UTC()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadCampaignRecord() = %#v, want %#v", got, want)
	}

	records, err := ListCampaignRecords(root)
	if err != nil {
		t.Fatalf("ListCampaignRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListCampaignRecords() len = %d, want 2", len(records))
	}
	if records[0].CampaignID != "campaign-a" || records[1].CampaignID != "campaign-b" {
		t.Fatalf("ListCampaignRecords() ids = [%q %q], want [campaign-a campaign-b]", records[0].CampaignID, records[1].CampaignID)
	}
}

func TestCampaignRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 22, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing campaign id",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.CampaignID = "   "
				}))
			},
			want: "mission store campaign campaign_id is required",
		},
		{
			name: "malformed campaign kind",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.CampaignKind = CampaignKind("surprise")
				}))
			},
			want: `mission store campaign campaign_kind "surprise" is invalid`,
		},
		{
			name: "malformed governed external target",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.GovernedExternalTargets = []AutonomyEligibilityTargetRef{
						{
							Kind:       EligibilityTargetKindProvider,
							RegistryID: "   ",
						},
					}
				}))
			},
			want: "mission store campaign governed_external_targets contain invalid target: autonomy eligibility target registry_id is required",
		},
		{
			name: "malformed frank object ref",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.FrankObjectRefs = []FrankRegistryObjectRef{
						{
							Kind:     FrankRegistryObjectKindIdentity,
							ObjectID: "   ",
						},
					}
				}))
			},
			want: "mission store campaign frank_object_refs contain invalid ref: Frank object ref object_id is required",
		},
		{
			name: "malformed identity mode",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.IdentityMode = IdentityMode("owner-ish")
				}))
			},
			want: `identity_mode "owner-ish" is invalid`,
		},
		{
			name: "missing stop conditions",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.StopConditions = nil
				}))
			},
			want: "mission store campaign stop_conditions are required",
		},
		{
			name: "missing failure threshold metric",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.FailureThreshold = CampaignFailureThreshold{Metric: " ", Limit: 1}
				}))
			},
			want: "mission store campaign failure_threshold.metric is required",
		},
		{
			name: "missing compliance checks",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.ComplianceChecks = nil
				}))
			},
			want: "mission store campaign compliance_checks are required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCampaignRecordRejectsDuplicateGovernedExternalTargetsAfterNormalization(t *testing.T) {
	t.Parallel()

	err := StoreCampaignRecord(t.TempDir(), validCampaignRecord(time.Date(2026, 4, 7, 23, 0, 0, 0, time.UTC), func(record *CampaignRecord) {
		record.GovernedExternalTargets = []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "provider-mail",
			},
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: " provider-mail ",
			},
		}
	}))
	if err == nil {
		t.Fatal("StoreCampaignRecord() error = nil, want duplicate target rejection")
	}
	if !strings.Contains(err.Error(), `mission store campaign governed_external_targets contain duplicate target kind "provider" registry_id "provider-mail"`) {
		t.Fatalf("StoreCampaignRecord() error = %q, want duplicate target rejection", err.Error())
	}
}

func TestCampaignRecordRejectsDuplicateFrankObjectRefsAfterNormalization(t *testing.T) {
	t.Parallel()

	err := StoreCampaignRecord(t.TempDir(), validCampaignRecord(time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), func(record *CampaignRecord) {
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindAccount,
				ObjectID: "account-mail",
			},
			{
				Kind:     FrankRegistryObjectKind(" account "),
				ObjectID: " account-mail ",
			},
		}
	}))
	if err == nil {
		t.Fatal("StoreCampaignRecord() error = nil, want duplicate ref rejection")
	}
	if !strings.Contains(err.Error(), `mission store campaign frank_object_refs contain duplicate ref kind "account" object_id "account-mail"`) {
		t.Fatalf("StoreCampaignRecord() error = %q, want duplicate ref rejection", err.Error())
	}
}

func TestCampaignRecordLoadFailsClosedOnMalformedStoredRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 1, 0, 0, 0, time.UTC)
	record := validCampaignRecord(now, nil)

	if err := WriteStoreJSONAtomic(StoreCampaignPath(root, "campaign-bad"), map[string]interface{}{
		"record_version": StoreRecordVersion,
		"campaign_id":    "campaign-bad",
		"campaign_kind":  string(record.CampaignKind),
		"display_name":   record.DisplayName,
		"state":          string(record.State),
		"objective":      record.Objective,
		"governed_external_targets": []map[string]interface{}{
			{
				"kind":        string(EligibilityTargetKindProvider),
				"registry_id": "provider-mail",
			},
		},
		"frank_object_refs": []map[string]interface{}{
			{
				"kind":      string(FrankRegistryObjectKindIdentity),
				"object_id": "identity-mail",
			},
		},
		"identity_mode": string(record.IdentityMode),
		"stop_conditions": []string{
			"",
		},
		"failure_threshold": map[string]interface{}{
			"metric": record.FailureThreshold.Metric,
			"limit":  record.FailureThreshold.Limit,
		},
		"compliance_checks": []string{
			record.ComplianceChecks[0],
		},
		"created_at": now,
		"updated_at": now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	_, err := LoadCampaignRecord(root, "campaign-bad")
	if err == nil {
		t.Fatal("LoadCampaignRecord() error = nil, want malformed-record rejection")
	}
	if !strings.Contains(err.Error(), "mission store campaign stop_conditions must not contain blanks") {
		t.Fatalf("LoadCampaignRecord() error = %q, want malformed-record rejection", err.Error())
	}
}

func TestCampaignRecordReusesExistingTargetObjectAndIdentityTypes(t *testing.T) {
	t.Parallel()

	campaignType := reflect.TypeOf(CampaignRecord{})

	targetsField, ok := campaignType.FieldByName("GovernedExternalTargets")
	if !ok {
		t.Fatal("CampaignRecord.GovernedExternalTargets field missing")
	}
	if targetsField.Type != reflect.TypeOf([]AutonomyEligibilityTargetRef(nil)) {
		t.Fatalf("CampaignRecord.GovernedExternalTargets type = %v, want %v", targetsField.Type, reflect.TypeOf([]AutonomyEligibilityTargetRef(nil)))
	}

	refsField, ok := campaignType.FieldByName("FrankObjectRefs")
	if !ok {
		t.Fatal("CampaignRecord.FrankObjectRefs field missing")
	}
	if refsField.Type != reflect.TypeOf([]FrankRegistryObjectRef(nil)) {
		t.Fatalf("CampaignRecord.FrankObjectRefs type = %v, want %v", refsField.Type, reflect.TypeOf([]FrankRegistryObjectRef(nil)))
	}

	modeField, ok := campaignType.FieldByName("IdentityMode")
	if !ok {
		t.Fatal("CampaignRecord.IdentityMode field missing")
	}
	if modeField.Type != reflect.TypeOf(IdentityMode("")) {
		t.Fatalf("CampaignRecord.IdentityMode type = %v, want %v", modeField.Type, reflect.TypeOf(IdentityMode("")))
	}
}

func TestCampaignRegistryScaffoldingDoesNotChangeCurrentV2RuntimePath(t *testing.T) {
	t.Parallel()

	ec, err := ResolveExecutionContext(testExecutionJob(), "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	if ec.Step == nil {
		t.Fatal("ResolveExecutionContext().Step = nil, want non-nil")
	}
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ResolveExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}
	if ec.GovernedExternalTargets != nil {
		t.Fatalf("ResolveExecutionContext().GovernedExternalTargets = %#v, want nil", ec.GovernedExternalTargets)
	}
	if ec.Step.FrankObjectRefs != nil {
		t.Fatalf("ResolveExecutionContext().Step.FrankObjectRefs = %#v, want nil", ec.Step.FrankObjectRefs)
	}
}

func validCampaignRecord(now time.Time, mutate func(*CampaignRecord)) CampaignRecord {
	record := CampaignRecord{
		CampaignID:   "campaign-1",
		CampaignKind: CampaignKindOutreach,
		DisplayName:  "Frank Outreach 1",
		State:        CampaignStateDraft,
		Objective:    "Reach aligned operators",
		GovernedExternalTargets: []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "provider-mail",
			},
		},
		FrankObjectRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: "identity-mail",
			},
		},
		IdentityMode:     IdentityModeAgentAlias,
		StopConditions:   []string{"stop after 3 replies"},
		FailureThreshold: CampaignFailureThreshold{Metric: "rejections", Limit: 3},
		ComplianceChecks: []string{"can-spam-reviewed"},
		CreatedAt:        now,
		UpdatedAt:        now.Add(time.Minute),
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
