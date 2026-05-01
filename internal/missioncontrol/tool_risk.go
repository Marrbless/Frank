package missioncontrol

import (
	"sort"
	"strings"
)

type ToolRiskProfile struct {
	ToolName   string `json:"tool_name"`
	ReadOnly   bool   `json:"read_only"`
	Filesystem bool   `json:"filesystem"`
	Exec       bool   `json:"exec"`
	Network    bool   `json:"network"`
	Account    bool   `json:"account"`
	Write      bool   `json:"write"`
	SideEffect bool   `json:"side_effect"`
	Evidence   string `json:"evidence"`
}

func (p ToolRiskProfile) DangerousForLowAuthority() bool {
	return p.Filesystem || p.Exec || p.Network || p.Account || p.Write || p.SideEffect
}

func ClassifyKnownToolRisk(toolName string) (ToolRiskProfile, bool) {
	normalized := strings.TrimSpace(toolName)
	if strings.HasPrefix(normalized, "mcp_") {
		return ToolRiskProfile{}, false
	}
	profile, ok := knownToolRiskProfiles()[normalized]
	return profile, ok
}

func KnownToolRiskProfiles() []ToolRiskProfile {
	profiles := knownToolRiskProfiles()
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]ToolRiskProfile, 0, len(names))
	for _, name := range names {
		out = append(out, profiles[name])
	}
	return out
}

func knownToolRiskProfiles() map[string]ToolRiskProfile {
	return map[string]ToolRiskProfile{
		"message": {
			ToolName:   "message",
			Network:    true,
			Write:      true,
			SideEffect: true,
			Evidence:   "MessageTool sends outbound channel messages.",
		},
		"frank_zoho_send_email": {
			ToolName:   "frank_zoho_send_email",
			Network:    true,
			Account:    true,
			Write:      true,
			SideEffect: true,
			Evidence:   "FrankZohoSendEmailTool sends email through a Zoho account.",
		},
		"frank_zoho_manage_reply_work_item": {
			ToolName:   "frank_zoho_manage_reply_work_item",
			Network:    true,
			Account:    true,
			Write:      true,
			SideEffect: true,
			Evidence:   "FrankZohoManageReplyWorkItemTool mutates Zoho reply work item state.",
		},
		"filesystem": {
			ToolName:   "filesystem",
			Filesystem: true,
			Write:      true,
			SideEffect: true,
			Evidence:   "FilesystemTool can read, write, move, copy, and delete workspace files.",
		},
		"exec": {
			ToolName:   "exec",
			Exec:       true,
			Write:      true,
			SideEffect: true,
			Evidence:   "ExecTool runs workspace commands.",
		},
		"web": {
			ToolName: "web",
			ReadOnly: true,
			Network:  true,
			Evidence: "WebTool fetches URL content over HTTP.",
		},
		"web_search": {
			ToolName: "web_search",
			ReadOnly: true,
			Network:  true,
			Evidence: "WebSearchTool queries a remote search endpoint.",
		},
		"cron": {
			ToolName:   "cron",
			Write:      true,
			SideEffect: true,
			Evidence:   "CronTool mutates scheduled job state.",
		},
		"write_memory": {
			ToolName:   "write_memory",
			Filesystem: true,
			Write:      true,
			SideEffect: true,
			Evidence:   "WriteMemoryTool writes local memory files.",
		},
		"list_memory": {
			ToolName:   "list_memory",
			ReadOnly:   true,
			Filesystem: true,
			Evidence:   "ListMemoryTool reads local memory index state.",
		},
		"read_memory": {
			ToolName:   "read_memory",
			ReadOnly:   true,
			Filesystem: true,
			Evidence:   "ReadMemoryTool reads local memory file contents.",
		},
		"edit_memory": {
			ToolName:   "edit_memory",
			Filesystem: true,
			Write:      true,
			SideEffect: true,
			Evidence:   "EditMemoryTool mutates local memory files.",
		},
		"delete_memory": {
			ToolName:   "delete_memory",
			Filesystem: true,
			Write:      true,
			SideEffect: true,
			Evidence:   "DeleteMemoryTool deletes local memory files.",
		},
		"create_skill": {
			ToolName:   "create_skill",
			Filesystem: true,
			Write:      true,
			SideEffect: true,
			Evidence:   "CreateSkillTool writes local skill files.",
		},
		"list_skills": {
			ToolName:   "list_skills",
			ReadOnly:   true,
			Filesystem: true,
			Evidence:   "ListSkillsTool reads local skill metadata.",
		},
		"read_skill": {
			ToolName:   "read_skill",
			ReadOnly:   true,
			Filesystem: true,
			Evidence:   "ReadSkillTool reads local skill files.",
		},
		"delete_skill": {
			ToolName:   "delete_skill",
			Filesystem: true,
			Write:      true,
			SideEffect: true,
			Evidence:   "DeleteSkillTool deletes local skill files.",
		},
	}
}
