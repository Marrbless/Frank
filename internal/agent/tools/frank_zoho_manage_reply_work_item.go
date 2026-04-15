package tools

import (
	"context"
	"fmt"
)

const frankZohoManageReplyWorkItemToolName = "frank_zoho_manage_reply_work_item"

const FrankZohoManageReplyWorkItemToolName = frankZohoManageReplyWorkItemToolName

type FrankZohoManageReplyWorkItemTool struct {
	taskState *TaskState
}

func NewFrankZohoManageReplyWorkItemTool() *FrankZohoManageReplyWorkItemTool {
	return &FrankZohoManageReplyWorkItemTool{}
}

func NewFrankZohoManageReplyWorkItemToolWithState(taskState *TaskState) *FrankZohoManageReplyWorkItemTool {
	return &FrankZohoManageReplyWorkItemTool{taskState: taskState}
}

func (t *FrankZohoManageReplyWorkItemTool) Name() string {
	return frankZohoManageReplyWorkItemToolName
}

func (t *FrankZohoManageReplyWorkItemTool) Description() string {
	return "Defer or ignore one committed Frank Zoho campaign reply work item by inbound_reply_id"
}

func (t *FrankZohoManageReplyWorkItemTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"inbound_reply_id": map[string]interface{}{
				"type":        "string",
				"description": "Committed Zoho inbound reply record linked to the reply work item",
			},
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Provider-specific triage mutation to apply",
				"enum":        []string{"defer", "ignore"},
			},
			"defer_until": map[string]interface{}{
				"type":        "string",
				"description": "RFC3339 timestamp required when action=defer",
			},
		},
		"required": []string{"inbound_reply_id", "action"},
	}
}

func (t *FrankZohoManageReplyWorkItemTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if t != nil && t.taskState != nil {
		result, skip, err := t.taskState.ManageFrankZohoCampaignReplyWorkItem(args)
		if err != nil {
			return "", err
		}
		if skip {
			return result, nil
		}
	}
	return "", fmt.Errorf("%s requires managed mission task state", t.Name())
}
