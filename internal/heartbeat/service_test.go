package heartbeat

import "testing"

func TestExtractHeartbeatTasksSkipsDefaultTemplate(t *testing.T) {
	content := `# Heartbeat

This file is checked periodically (every 60 seconds). Add tasks here that should run on a schedule.

## IMPORTANT RULES FOR HEARTBEAT PROCESSING

- After reviewing this file, take actions ONLY if there are explicit tasks listed below
- If there are no tasks (or all tasks are complete), do NOTHING

## Periodic Tasks

<!-- Add tasks below. The agent will process them on each heartbeat check. -->
<!-- Example:
- Check server status at https://example.com/health
- Summarize unread messages
-->
`

	if got := extractHeartbeatTasks(content); got != "" {
		t.Fatalf("extractHeartbeatTasks(default template) = %q, want empty", got)
	}
}

func TestExtractHeartbeatTasksKeepsExplicitTasks(t *testing.T) {
	content := `# Heartbeat

## Periodic Tasks

<!-- comments are ignored -->
- Check https://example.com/health.
- If it fails, message me on Telegram.
`

	got := extractHeartbeatTasks(content)
	want := "- Check https://example.com/health.\n- If it fails, message me on Telegram."
	if got != want {
		t.Fatalf("extractHeartbeatTasks() = %q, want %q", got, want)
	}
}

func TestExtractHeartbeatTasksSupportsPlainTaskFiles(t *testing.T) {
	content := `
- Check whether the gateway is alive.

<!-- skip this comment -->
- Message me only if it is not.
`

	got := extractHeartbeatTasks(content)
	want := "- Check whether the gateway is alive.\n- Message me only if it is not."
	if got != want {
		t.Fatalf("extractHeartbeatTasks() = %q, want %q", got, want)
	}
}
