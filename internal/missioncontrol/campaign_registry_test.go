package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCampaignRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 7, 21, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	if err := StoreCampaignRecord(root, CampaignRecord{
		CampaignID:   "campaign-b",
		CampaignKind: CampaignKindCommunity,
		DisplayName:  "Community B",
		State:        CampaignStateDraft,
		Objective:    "Join communities without posting yet",
		GovernedExternalTargets: []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "provider-mail",
			},
		},
		FrankObjectRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: fixtures.identity.IdentityID,
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

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
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
			name: "malformed campaign id",
			run: func() error {
				return StoreCampaignRecord(root, validCampaignRecord(now, func(record *CampaignRecord) {
					record.CampaignID = "campaign/one"
				}))
			},
			want: `mission store campaign campaign_id "campaign/one" is invalid`,
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

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	err := StoreCampaignRecord(fixtures.root, validCampaignRecord(time.Date(2026, 4, 7, 23, 0, 0, 0, time.UTC), func(record *CampaignRecord) {
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

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	err := StoreCampaignRecord(fixtures.root, validCampaignRecord(time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), func(record *CampaignRecord) {
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

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
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

func TestCampaignRecordRequiresExistingGovernedTargetAndFrankObjectLinks(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 1, 30, 0, 0, time.UTC)

	t.Run("missing governed target", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		err := StoreCampaignRecord(fixtures.root, validCampaignRecord(now, func(record *CampaignRecord) {
			record.GovernedExternalTargets = []AutonomyEligibilityTargetRef{
				{
					Kind:       EligibilityTargetKindPlatform,
					RegistryID: "platform-missing",
				},
			}
		}))
		if err == nil {
			t.Fatal("StoreCampaignRecord() error = nil, want missing governed-target rejection")
		}
		if !strings.Contains(err.Error(), `mission store campaign governed_external_targets target kind "platform" registry_id "platform-missing": mission store frank registry eligibility_target_ref "platform-missing" has no linked eligibility registry record`) {
			t.Fatalf("StoreCampaignRecord() error = %q, want missing governed-target rejection", err.Error())
		}
	})

	t.Run("missing Frank object ref", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		err := StoreCampaignRecord(fixtures.root, validCampaignRecord(now, func(record *CampaignRecord) {
			record.FrankObjectRefs = []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindIdentity,
					ObjectID: "identity-missing",
				},
			}
		}))
		if err == nil {
			t.Fatal("StoreCampaignRecord() error = nil, want missing Frank-object rejection")
		}
		if !strings.Contains(err.Error(), `mission store campaign frank_object_refs ref kind "identity" object_id "identity-missing": resolve Frank object ref kind "identity" object_id "identity-missing": mission store Frank identity record not found`) {
			t.Fatalf("StoreCampaignRecord() error = %q, want missing Frank-object rejection", err.Error())
		}
	})
}

func TestLoadCampaignRecordFailsClosedWhenLinkedTargetOrFrankObjectIsMissing(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 2, 0, 0, 0, time.UTC)

	t.Run("missing governed target", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		record := validCampaignRecord(now, func(record *CampaignRecord) {
			record.CampaignID = "campaign-missing-governed-target"
			record.GovernedExternalTargets = []AutonomyEligibilityTargetRef{
				{
					Kind:       EligibilityTargetKindPlatform,
					RegistryID: "platform-missing",
				},
			}
		})
		if err := WriteStoreJSONAtomic(StoreCampaignPath(fixtures.root, record.CampaignID), map[string]interface{}{
			"record_version": StoreRecordVersion,
			"campaign_id":    record.CampaignID,
			"campaign_kind":  string(record.CampaignKind),
			"display_name":   record.DisplayName,
			"state":          string(record.State),
			"objective":      record.Objective,
			"governed_external_targets": []map[string]interface{}{
				{
					"kind":        string(record.GovernedExternalTargets[0].Kind),
					"registry_id": record.GovernedExternalTargets[0].RegistryID,
				},
			},
			"frank_object_refs": []map[string]interface{}{
				{
					"kind":      string(record.FrankObjectRefs[0].Kind),
					"object_id": record.FrankObjectRefs[0].ObjectID,
				},
			},
			"identity_mode": string(record.IdentityMode),
			"stop_conditions": []string{
				record.StopConditions[0],
			},
			"failure_threshold": map[string]interface{}{
				"metric": record.FailureThreshold.Metric,
				"limit":  record.FailureThreshold.Limit,
			},
			"compliance_checks": []string{
				record.ComplianceChecks[0],
			},
			"created_at": record.CreatedAt,
			"updated_at": record.UpdatedAt,
		}); err != nil {
			t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
		}

		_, err := LoadCampaignRecord(fixtures.root, record.CampaignID)
		if err == nil {
			t.Fatal("LoadCampaignRecord() error = nil, want missing governed-target rejection")
		}
		if !strings.Contains(err.Error(), `mission store campaign governed_external_targets target kind "platform" registry_id "platform-missing": mission store frank registry eligibility_target_ref "platform-missing" has no linked eligibility registry record`) {
			t.Fatalf("LoadCampaignRecord() error = %q, want missing governed-target rejection", err.Error())
		}
	})

	t.Run("missing Frank object ref", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		record := validCampaignRecord(now, func(record *CampaignRecord) {
			record.CampaignID = "campaign-missing-frank-object"
			record.FrankObjectRefs = []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindIdentity,
					ObjectID: "identity-missing",
				},
			}
		})
		if err := WriteStoreJSONAtomic(StoreCampaignPath(fixtures.root, record.CampaignID), map[string]interface{}{
			"record_version": StoreRecordVersion,
			"campaign_id":    record.CampaignID,
			"campaign_kind":  string(record.CampaignKind),
			"display_name":   record.DisplayName,
			"state":          string(record.State),
			"objective":      record.Objective,
			"governed_external_targets": []map[string]interface{}{
				{
					"kind":        string(record.GovernedExternalTargets[0].Kind),
					"registry_id": record.GovernedExternalTargets[0].RegistryID,
				},
			},
			"frank_object_refs": []map[string]interface{}{
				{
					"kind":      string(record.FrankObjectRefs[0].Kind),
					"object_id": record.FrankObjectRefs[0].ObjectID,
				},
			},
			"identity_mode": string(record.IdentityMode),
			"stop_conditions": []string{
				record.StopConditions[0],
			},
			"failure_threshold": map[string]interface{}{
				"metric": record.FailureThreshold.Metric,
				"limit":  record.FailureThreshold.Limit,
			},
			"compliance_checks": []string{
				record.ComplianceChecks[0],
			},
			"created_at": record.CreatedAt,
			"updated_at": record.UpdatedAt,
		}); err != nil {
			t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
		}

		_, err := LoadCampaignRecord(fixtures.root, record.CampaignID)
		if err == nil {
			t.Fatal("LoadCampaignRecord() error = nil, want missing Frank-object rejection")
		}
		if !strings.Contains(err.Error(), `mission store campaign frank_object_refs ref kind "identity" object_id "identity-missing": resolve Frank object ref kind "identity" object_id "identity-missing": mission store Frank identity record not found`) {
			t.Fatalf("LoadCampaignRecord() error = %q, want missing Frank-object rejection", err.Error())
		}
	})
}

func TestCampaignObjectViewPreservesCanonicalCampaignContractSurface(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 2, 30, 0, 0, time.UTC)
	record := validCampaignRecord(now, nil)

	view := record.AsObjectView()
	if view.CampaignID != record.CampaignID || view.Objective != record.Objective || view.IdentityMode != record.IdentityMode {
		t.Fatalf("CampaignRecord.AsObjectView() = %#v, want canonical campaign contract fields", view)
	}
	if view.PlatformOrChannel != "provider-mail" {
		t.Fatalf("CampaignRecord.AsObjectView().PlatformOrChannel = %q, want %q", view.PlatformOrChannel, "provider-mail")
	}
	if view.AudienceClassOrTarget != "" || view.MessageFamilyOrParticipationStyle != "" || view.Cadence != "" || view.Budget != "" {
		t.Fatalf("CampaignRecord.AsObjectView() unresolved optional fields = %#v, want zero values until storage adds durable sources", view)
	}
	if view.EscalationRules != nil {
		t.Fatalf("CampaignRecord.AsObjectView().EscalationRules = %#v, want nil until storage adds durable sources", view.EscalationRules)
	}
	if !reflect.DeepEqual(view.GovernedExternalTargets, record.GovernedExternalTargets) {
		t.Fatalf("CampaignRecord.AsObjectView().GovernedExternalTargets = %#v, want %#v", view.GovernedExternalTargets, record.GovernedExternalTargets)
	}
	if !reflect.DeepEqual(view.FrankObjectRefs, record.FrankObjectRefs) {
		t.Fatalf("CampaignRecord.AsObjectView().FrankObjectRefs = %#v, want %#v", view.FrankObjectRefs, record.FrankObjectRefs)
	}
}

func TestCampaignObjectViewKeepsUnresolvedEnvelopeFieldsNonDurableAfterLoad(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 2, 35, 0, 0, time.UTC)
	record := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-object-view-load"
	})

	if err := StoreCampaignRecord(root, record); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	loaded, err := LoadCampaignRecord(root, record.CampaignID)
	if err != nil {
		t.Fatalf("LoadCampaignRecord() error = %v", err)
	}

	view := loaded.AsObjectView()
	if view.PlatformOrChannel != "provider-mail" {
		t.Fatalf("loaded CampaignRecord.AsObjectView().PlatformOrChannel = %q, want %q", view.PlatformOrChannel, "provider-mail")
	}
	if view.AudienceClassOrTarget != "" || view.MessageFamilyOrParticipationStyle != "" || view.Cadence != "" || view.Budget != "" {
		t.Fatalf("loaded CampaignRecord.AsObjectView() unresolved non-durable fields = %#v, want zero values until a justified durable source exists", view)
	}
	if view.EscalationRules != nil {
		t.Fatalf("loaded CampaignRecord.AsObjectView().EscalationRules = %#v, want nil until a justified durable source exists", view.EscalationRules)
	}
}

func TestResolveCampaignPlatformOrChannelRequiresSingleProviderOrPlatformTarget(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 2, 45, 0, 0, time.UTC)

	platform, ok := ResolveCampaignPlatformOrChannel(validCampaignRecord(now, nil))
	if !ok || platform != "provider-mail" {
		t.Fatalf("ResolveCampaignPlatformOrChannel(validCampaignRecord) = (%q, %t), want (%q, true)", platform, ok, "provider-mail")
	}

	record := validCampaignRecord(now, func(record *CampaignRecord) {
		record.GovernedExternalTargets = []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindPlatform,
				RegistryID: "community-platform",
			},
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "provider-mail",
			},
		}
	})
	platform, ok = ResolveCampaignPlatformOrChannel(record)
	if ok || platform != "" {
		t.Fatalf("ResolveCampaignPlatformOrChannel(multi-target) = (%q, %t), want (\"\", false)", platform, ok)
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

func TestCampaignRefUsesSingleStepControlPlaneSurface(t *testing.T) {
	t.Parallel()

	stepType := reflect.TypeOf(Step{})
	campaignRefField, ok := stepType.FieldByName("CampaignRef")
	if !ok {
		t.Fatal("Step.CampaignRef field missing")
	}
	if campaignRefField.Type != reflect.TypeOf((*CampaignRef)(nil)) {
		t.Fatalf("Step.CampaignRef type = %v, want %v", campaignRefField.Type, reflect.TypeOf((*CampaignRef)(nil)))
	}

	executionContextType := reflect.TypeOf(ExecutionContext{})
	if _, ok := executionContextType.FieldByName("CampaignRef"); ok {
		t.Fatal("ExecutionContext.CampaignRef field present, want no duplicate top-level campaign-ref source")
	}
	if _, ok := executionContextType.FieldByName("CampaignID"); ok {
		t.Fatal("ExecutionContext.CampaignID field present, want no duplicate top-level campaign identity source")
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
	if ec.Step.CampaignRef != nil {
		t.Fatalf("ResolveExecutionContext().Step.CampaignRef = %#v, want nil", ec.Step.CampaignRef)
	}
}

func TestResolveExecutionContextCampaignRefZeroRefPathPreservesPriorBehavior(t *testing.T) {
	t.Parallel()

	ec, err := ResolveExecutionContext(testExecutionJob(), "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	if ec.Step == nil {
		t.Fatal("ResolveExecutionContext().Step = nil, want non-nil")
	}
	if ec.Step.CampaignRef != nil {
		t.Fatalf("ResolveExecutionContext().Step.CampaignRef = %#v, want nil", ec.Step.CampaignRef)
	}

	got, err := ResolveExecutionContextCampaignRef(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextCampaignRef() error = %v", err)
	}
	if got != nil {
		t.Fatalf("ResolveExecutionContextCampaignRef() = %#v, want nil for zero-campaign step", got)
	}
}

func TestResolveExecutionContextCampaignRefResolvesActiveCampaignRef(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 6, 0, 0, 0, time.UTC)
	record := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-active"
		record.IdentityMode = IdentityModeOwnerOnlyControl
		record.GovernedExternalTargets = []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "provider-mail",
			},
		}
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: fixtures.identity.IdentityID,
			},
		}
	})
	if err := StoreCampaignRecord(root, record); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}
	record.RecordVersion = StoreRecordVersion

	job := testExecutionJob()
	job.Plan.Steps[0].CampaignRef = &CampaignRef{
		CampaignID: " campaign-active ",
	}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	got, err := ResolveExecutionContextCampaignRef(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextCampaignRef() error = %v", err)
	}
	if got == nil {
		t.Fatal("ResolveExecutionContextCampaignRef() = nil, want resolved campaign record")
	}
	if !reflect.DeepEqual(*got, record) {
		t.Fatalf("ResolveExecutionContextCampaignRef() = %#v, want %#v", *got, record)
	}
}

func TestResolveExecutionContextCampaignRefFailsClosedOnMissingOrMalformedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tests := []struct {
		name string
		ref  *CampaignRef
		want string
	}{
		{
			name: "missing record",
			ref: &CampaignRef{
				CampaignID: "campaign-missing",
			},
			want: ErrCampaignRecordNotFound.Error(),
		},
		{
			name: "empty campaign id",
			ref: &CampaignRef{
				CampaignID: "   ",
			},
			want: "campaign_id is required",
		},
		{
			name: "malformed campaign id",
			ref: &CampaignRef{
				CampaignID: "campaign/malformed",
			},
			want: `campaign_id "campaign/malformed" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ec := testExecutionContext()
			ec.Step.CampaignRef = tc.ref
			ec.MissionStoreRoot = root

			_, err := ResolveExecutionContextCampaignRef(ec)
			if err == nil {
				t.Fatal("ResolveExecutionContextCampaignRef() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ResolveExecutionContextCampaignRef() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestResolveExecutionContextCampaignRefFailsClosedWithoutMissionStoreRoot(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Step.CampaignRef = &CampaignRef{
		CampaignID: "campaign-1",
	}

	_, err := ResolveExecutionContextCampaignRef(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextCampaignRef() error = nil, want missing mission store root rejection")
	}
	if !strings.Contains(err.Error(), "mission store root is required to resolve campaign refs") {
		t.Fatalf("ResolveExecutionContextCampaignRef() error = %q, want missing mission store root rejection", err.Error())
	}
}

func TestResolveExecutionContextCampaignRefDoesNotIntroduceIdentityEligibilityOrObjectSideChannel(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 7, 0, 0, 0, time.UTC)
	record := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-sidechannel"
		record.IdentityMode = IdentityModeOwnerOnlyControl
		record.GovernedExternalTargets = []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "provider-mail",
			},
		}
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: fixtures.identity.IdentityID,
			},
		}
	})
	if err := StoreCampaignRecord(root, record); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].CampaignRef = &CampaignRef{
		CampaignID: record.CampaignID,
	}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ResolveExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}
	if ec.GovernedExternalTargets != nil {
		t.Fatalf("ResolveExecutionContext().GovernedExternalTargets = %#v, want nil", ec.GovernedExternalTargets)
	}
	if ec.Step.FrankObjectRefs != nil {
		t.Fatalf("ResolveExecutionContext().Step.FrankObjectRefs = %#v, want nil", ec.Step.FrankObjectRefs)
	}

	got, err := ResolveExecutionContextCampaignRef(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextCampaignRef() error = %v", err)
	}
	if got == nil {
		t.Fatal("ResolveExecutionContextCampaignRef() = nil, want resolved campaign record")
	}
	if got.IdentityMode != IdentityModeOwnerOnlyControl {
		t.Fatalf("ResolveExecutionContextCampaignRef().IdentityMode = %q, want %q", got.IdentityMode, IdentityModeOwnerOnlyControl)
	}
	if len(got.GovernedExternalTargets) != 1 || got.GovernedExternalTargets[0].RegistryID != "provider-mail" {
		t.Fatalf("ResolveExecutionContextCampaignRef().GovernedExternalTargets = %#v, want campaign-owned target only", got.GovernedExternalTargets)
	}
	if len(got.FrankObjectRefs) != 1 || got.FrankObjectRefs[0].ObjectID != fixtures.identity.IdentityID {
		t.Fatalf("ResolveExecutionContextCampaignRef().FrankObjectRefs = %#v, want campaign-owned ref only", got.FrankObjectRefs)
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
