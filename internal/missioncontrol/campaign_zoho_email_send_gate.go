package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var campaignZohoEmailVerifiedSendStopPattern = regexp.MustCompile(`^stop after ([1-9][0-9]*) verified sends?$`)
var campaignZohoEmailReplyStopPattern = regexp.MustCompile(`^stop after ([1-9][0-9]*) replies?$`)

type CampaignZohoEmailSendGateDecision struct {
	CampaignID             string `json:"campaign_id"`
	Allowed                bool   `json:"allowed"`
	Halted                 bool   `json:"halted"`
	Reason                 string `json:"reason,omitempty"`
	TriggeredStopCondition string `json:"triggered_stop_condition,omitempty"`
	VerifiedSuccessCount   int    `json:"verified_success_count"`
	AttributedReplyCount   int    `json:"attributed_reply_count"`
	FailureCount           int    `json:"failure_count"`
	AmbiguousOutcomeCount  int    `json:"ambiguous_outcome_count"`
	FailureThresholdMetric string `json:"failure_threshold_metric,omitempty"`
	FailureThresholdLimit  int    `json:"failure_threshold_limit,omitempty"`
}

func ListCommittedAllCampaignZohoEmailOutboundActionRecords(root string) ([]CampaignZohoEmailOutboundActionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreJobsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	byActionID := make(map[string]CampaignZohoEmailOutboundActionRecord)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		jobID := strings.TrimSpace(entry.Name())
		if jobID == "" {
			continue
		}
		records, err := ListCommittedCampaignZohoEmailOutboundActionRecords(root, jobID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			existing, ok := byActionID[record.ActionID]
			if !ok || campaignZohoEmailOutboundRecordPreferred(record, existing) {
				byActionID[record.ActionID] = record
			}
		}
	}
	return sortCampaignZohoEmailOutboundActionRecords(byActionID), nil
}

func ListCommittedAllFrankZohoInboundReplyRecords(root string) ([]FrankZohoInboundReplyRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreJobsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	byReplyID := make(map[string]FrankZohoInboundReplyRecord)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		jobID := strings.TrimSpace(entry.Name())
		if jobID == "" {
			continue
		}
		records, err := ListCommittedFrankZohoInboundReplyRecords(root, jobID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			existing, ok := byReplyID[record.ReplyID]
			if !ok || frankZohoInboundReplyRecordPreferred(record, existing) {
				byReplyID[record.ReplyID] = record
			}
		}
	}

	if len(byReplyID) == 0 {
		return nil, nil
	}
	records := make([]FrankZohoInboundReplyRecord, 0, len(byReplyID))
	for _, record := range byReplyID {
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].ReceivedAt.Equal(records[j].ReceivedAt) {
			return records[i].ReceivedAt.Before(records[j].ReceivedAt)
		}
		return records[i].ReplyID < records[j].ReplyID
	})
	return records, nil
}

func ListCommittedCampaignZohoEmailOutboundActionRecordsByCampaign(root, campaignID string) ([]CampaignZohoEmailOutboundActionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	if err := validateCampaignID(campaignID, "mission store campaign zoho email outbound action"); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(StoreJobsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	normalizedCampaignID := strings.TrimSpace(campaignID)
	byActionID := make(map[string]CampaignZohoEmailOutboundActionRecord)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		jobID := strings.TrimSpace(entry.Name())
		if jobID == "" {
			continue
		}
		records, err := ListCommittedCampaignZohoEmailOutboundActionRecords(root, jobID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if strings.TrimSpace(record.CampaignID) != normalizedCampaignID {
				continue
			}
			existing, ok := byActionID[record.ActionID]
			if !ok || campaignZohoEmailOutboundRecordPreferred(record, existing) {
				byActionID[record.ActionID] = record
			}
		}
	}

	return sortCampaignZohoEmailOutboundActionRecords(byActionID), nil
}

func LoadCommittedCampaignZohoEmailSendGateDecision(root string, campaign CampaignRecord) (CampaignZohoEmailSendGateDecision, error) {
	normalized := normalizeCampaignRecord(campaign)
	outboundRecords, err := ListCommittedAllCampaignZohoEmailOutboundActionRecords(root)
	if err != nil {
		return CampaignZohoEmailSendGateDecision{}, err
	}
	inboundReplyRecords, err := ListCommittedAllFrankZohoInboundReplyRecords(root)
	if err != nil {
		return CampaignZohoEmailSendGateDecision{}, err
	}
	return DeriveCampaignZohoEmailSendGateDecision(normalized, outboundRecords, inboundReplyRecords)
}

func DeriveCampaignZohoEmailSendGateDecision(campaign CampaignRecord, outboundRecords []CampaignZohoEmailOutboundActionRecord, inboundReplyRecords []FrankZohoInboundReplyRecord) (CampaignZohoEmailSendGateDecision, error) {
	normalizedCampaign := normalizeCampaignRecord(campaign)
	decision := CampaignZohoEmailSendGateDecision{
		CampaignID: strings.TrimSpace(normalizedCampaign.CampaignID),
	}

	for _, record := range outboundRecords {
		normalized := NormalizeCampaignZohoEmailOutboundAction(CampaignZohoEmailOutboundAction{
			ActionID:           record.ActionID,
			StepID:             record.StepID,
			CampaignID:         record.CampaignID,
			State:              CampaignZohoEmailOutboundActionState(record.State),
			Provider:           record.Provider,
			ProviderAccountID:  record.ProviderAccountID,
			FromAddress:        record.FromAddress,
			FromDisplayName:    record.FromDisplayName,
			Addressing:         record.Addressing,
			Subject:            record.Subject,
			BodyFormat:         record.BodyFormat,
			BodySHA256:         record.BodySHA256,
			PreparedAt:         record.PreparedAt,
			SentAt:             record.SentAt,
			VerifiedAt:         record.VerifiedAt,
			FailedAt:           record.FailedAt,
			ProviderMessageID:  record.ProviderMessageID,
			ProviderMailID:     record.ProviderMailID,
			MIMEMessageID:      record.MIMEMessageID,
			OriginalMessageURL: record.OriginalMessageURL,
			Failure:            record.Failure,
		})
		if normalized.CampaignID != decision.CampaignID {
			continue
		}
		switch normalized.State {
		case CampaignZohoEmailOutboundActionStateVerified:
			decision.VerifiedSuccessCount++
		case CampaignZohoEmailOutboundActionStateFailed:
			decision.FailureCount++
		case CampaignZohoEmailOutboundActionStatePrepared, CampaignZohoEmailOutboundActionStateSent:
			decision.AmbiguousOutcomeCount++
		default:
			return CampaignZohoEmailSendGateDecision{}, fmt.Errorf("campaign zoho email outbound action %q state %q is not supported for stop/failure derivation", normalized.ActionID, normalized.State)
		}
	}
	decision.AttributedReplyCount = campaignZohoEmailAttributedReplyCount(decision.CampaignID, outboundRecords, inboundReplyRecords)

	for _, stopCondition := range normalizedCampaign.StopConditions {
		if limit, ok := campaignZohoEmailVerifiedSendStopLimit(stopCondition); ok {
			if decision.VerifiedSuccessCount >= limit {
				decision.Allowed = false
				decision.Halted = true
				decision.TriggeredStopCondition = stopCondition
				decision.Reason = fmt.Sprintf("campaign zoho email stop_condition %q triggered after %d verified sends", stopCondition, decision.VerifiedSuccessCount)
				return decision, nil
			}
			continue
		}
		if limit, ok := campaignZohoEmailReplyStopLimit(stopCondition); ok {
			if decision.AttributedReplyCount >= limit {
				decision.Allowed = false
				decision.Halted = true
				decision.TriggeredStopCondition = stopCondition
				decision.Reason = fmt.Sprintf("campaign zoho email stop_condition %q triggered after %d attributed replies", stopCondition, decision.AttributedReplyCount)
				return decision, nil
			}
			continue
		}
		return CampaignZohoEmailSendGateDecision{}, fmt.Errorf("campaign zoho email stop_condition %q is not evaluable from committed outbound and inbound reply records", stopCondition)
	}

	decision.FailureThresholdMetric = normalizedCampaign.FailureThreshold.Metric
	decision.FailureThresholdLimit = normalizedCampaign.FailureThreshold.Limit
	switch normalizedCampaign.FailureThreshold.Metric {
	case "rejections", "verified_failed_sends":
	default:
		return CampaignZohoEmailSendGateDecision{}, fmt.Errorf("campaign zoho email failure_threshold.metric %q is not evaluable from committed outbound action records", normalizedCampaign.FailureThreshold.Metric)
	}
	if decision.FailureCount >= normalizedCampaign.FailureThreshold.Limit {
		decision.Allowed = false
		decision.Halted = true
		decision.Reason = fmt.Sprintf("campaign zoho email failure_threshold %q reached %d/%d counted failures", normalizedCampaign.FailureThreshold.Metric, decision.FailureCount, normalizedCampaign.FailureThreshold.Limit)
		return decision, nil
	}

	decision.Allowed = true
	return decision, nil
}

func campaignZohoEmailVerifiedSendStopLimit(stopCondition string) (int, bool) {
	normalized := strings.ToLower(strings.TrimSpace(stopCondition))
	if normalized == "stop after first verified send" {
		return 1, true
	}
	match := campaignZohoEmailVerifiedSendStopPattern.FindStringSubmatch(normalized)
	if len(match) != 2 {
		return 0, false
	}
	limit, err := strconv.Atoi(match[1])
	if err != nil || limit <= 0 {
		return 0, false
	}
	return limit, true
}

func CampaignZohoEmailStopConditionsRequireInboundReplies(stopConditions []string) bool {
	for _, stopCondition := range stopConditions {
		if _, ok := campaignZohoEmailReplyStopLimit(stopCondition); ok {
			return true
		}
	}
	return false
}

func campaignZohoEmailReplyStopLimit(stopCondition string) (int, bool) {
	normalized := strings.ToLower(strings.TrimSpace(stopCondition))
	if normalized == "stop after first reply" {
		return 1, true
	}
	match := campaignZohoEmailReplyStopPattern.FindStringSubmatch(normalized)
	if len(match) != 2 {
		return 0, false
	}
	limit, err := strconv.Atoi(match[1])
	if err != nil || limit <= 0 {
		return 0, false
	}
	return limit, true
}

func campaignZohoEmailAttributedReplyCount(campaignID string, outboundRecords []CampaignZohoEmailOutboundActionRecord, inboundReplyRecords []FrankZohoInboundReplyRecord) int {
	count := 0
	for _, record := range inboundReplyRecords {
		action, ok := attributedCampaignZohoEmailOutboundActionForReply(record, outboundRecords)
		if ok && strings.TrimSpace(action.CampaignID) == campaignID {
			count++
		}
	}
	return count
}

func sortCampaignZohoEmailOutboundActionRecords(byActionID map[string]CampaignZohoEmailOutboundActionRecord) []CampaignZohoEmailOutboundActionRecord {
	if len(byActionID) == 0 {
		return nil
	}
	records := make([]CampaignZohoEmailOutboundActionRecord, 0, len(byActionID))
	for _, record := range byActionID {
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		left := records[i]
		right := records[j]
		if !left.PreparedAt.Equal(right.PreparedAt) {
			return left.PreparedAt.Before(right.PreparedAt)
		}
		return left.ActionID < right.ActionID
	})
	return records
}

func campaignZohoEmailOutboundRecordPreferred(candidate, existing CampaignZohoEmailOutboundActionRecord) bool {
	candidateRank, candidateAt := campaignZohoEmailOutboundRecordRank(candidate)
	existingRank, existingAt := campaignZohoEmailOutboundRecordRank(existing)
	if candidateRank != existingRank {
		return candidateRank > existingRank
	}
	if !candidateAt.Equal(existingAt) {
		return candidateAt.After(existingAt)
	}
	if candidate.LastSeq != existing.LastSeq {
		return candidate.LastSeq > existing.LastSeq
	}
	leftPath := filepath.Join(strings.TrimSpace(candidate.JobID), strings.TrimSpace(candidate.ActionID))
	rightPath := filepath.Join(strings.TrimSpace(existing.JobID), strings.TrimSpace(existing.ActionID))
	return leftPath > rightPath
}

func frankZohoInboundReplyRecordPreferred(candidate, existing FrankZohoInboundReplyRecord) bool {
	if !candidate.ReceivedAt.Equal(existing.ReceivedAt) {
		return candidate.ReceivedAt.After(existing.ReceivedAt)
	}
	if candidate.LastSeq != existing.LastSeq {
		return candidate.LastSeq > existing.LastSeq
	}
	leftPath := filepath.Join(strings.TrimSpace(candidate.JobID), strings.TrimSpace(candidate.ReplyID))
	rightPath := filepath.Join(strings.TrimSpace(existing.JobID), strings.TrimSpace(existing.ReplyID))
	return leftPath > rightPath
}

func campaignZohoEmailOutboundRecordRank(record CampaignZohoEmailOutboundActionRecord) (int, time.Time) {
	switch CampaignZohoEmailOutboundActionState(strings.TrimSpace(record.State)) {
	case CampaignZohoEmailOutboundActionStateVerified:
		return 2, firstProjectedTime(record.VerifiedAt, record.SentAt, record.PreparedAt)
	case CampaignZohoEmailOutboundActionStateFailed:
		return 2, firstProjectedTime(record.FailedAt, record.PreparedAt)
	case CampaignZohoEmailOutboundActionStateSent:
		return 1, firstProjectedTime(record.SentAt, record.PreparedAt)
	case CampaignZohoEmailOutboundActionStatePrepared:
		return 0, firstProjectedTime(record.PreparedAt)
	default:
		return -1, firstProjectedTime(record.VerifiedAt, record.FailedAt, record.SentAt, record.PreparedAt)
	}
}
