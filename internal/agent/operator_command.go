package agent

import "strings"

var hotUpdateOperatorHelpAliases = []string{
	"HOT_UPDATE_HELP",
	"HELP HOT_UPDATE",
	"HELP V4",
}

type operatorCommandKind string

const (
	operatorCommandHotUpdateHelp operatorCommandKind = "hot_update_help"
	operatorCommandStatus        operatorCommandKind = "status"
	operatorCommandPause         operatorCommandKind = "pause"
	operatorCommandResume        operatorCommandKind = "resume"
	operatorCommandAbort         operatorCommandKind = "abort"
	operatorCommandApprove       operatorCommandKind = "approve"
	operatorCommandDeny          operatorCommandKind = "deny"
	operatorCommandSetStep       operatorCommandKind = "set_step"
)

type parsedOperatorCommand struct {
	Kind   operatorCommandKind
	JobID  string
	StepID string
}

func staticHotUpdateHelpAliases() []string {
	return append([]string(nil), hotUpdateOperatorHelpAliases...)
}

func formatStaticOperatorCommandAliases(aliases []string) string {
	if len(aliases) == 0 {
		return ""
	}
	return "  " + strings.Join(aliases, "\n  ")
}

func parseStaticOperatorCommand(content string) (parsedOperatorCommand, bool) {
	trimmed := strings.TrimSpace(content)
	if hotUpdateHelpCommandRE.MatchString(trimmed) {
		return parsedOperatorCommand{Kind: operatorCommandHotUpdateHelp}, true
	}

	matches := runtimeCommandRE.FindStringSubmatch(trimmed)
	if len(matches) == 3 {
		switch strings.ToLower(matches[1]) {
		case "status":
			return parsedOperatorCommand{Kind: operatorCommandStatus, JobID: matches[2]}, true
		case "pause":
			return parsedOperatorCommand{Kind: operatorCommandPause, JobID: matches[2]}, true
		case "resume":
			return parsedOperatorCommand{Kind: operatorCommandResume, JobID: matches[2]}, true
		case "abort":
			return parsedOperatorCommand{Kind: operatorCommandAbort, JobID: matches[2]}, true
		}
	}

	matches = approvalCommandRE.FindStringSubmatch(trimmed)
	if len(matches) == 4 {
		switch strings.ToLower(matches[1]) {
		case "approve":
			return parsedOperatorCommand{Kind: operatorCommandApprove, JobID: matches[2], StepID: matches[3]}, true
		case "deny":
			return parsedOperatorCommand{Kind: operatorCommandDeny, JobID: matches[2], StepID: matches[3]}, true
		}
	}

	matches = setStepCommandRE.FindStringSubmatch(trimmed)
	if len(matches) == 4 {
		return parsedOperatorCommand{Kind: operatorCommandSetStep, JobID: matches[2], StepID: matches[3]}, true
	}

	return parsedOperatorCommand{}, false
}
