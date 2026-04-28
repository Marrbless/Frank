package heartbeat

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/local/picobot/internal/chat"
)

// StartHeartbeat starts a periodic check that reads HEARTBEAT.md and pushes
// its content into the agent's inbound chat hub for processing.
func StartHeartbeat(ctx context.Context, workspace string, interval time.Duration, hub *chat.Hub) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		log.Printf("heartbeat: started (every %v)", interval)
		for {
			select {
			case <-ctx.Done():
				log.Println("heartbeat: stopping")
				return
			case <-ticker.C:
				path := filepath.Join(workspace, "HEARTBEAT.md")
				data, err := os.ReadFile(path)
				if err != nil {
					// file doesn't exist or can't be read — skip silently
					continue
				}
				tasks := extractHeartbeatTasks(string(data))
				if tasks == "" {
					continue
				}

				// Push heartbeat content into the agent loop for processing
				log.Println("heartbeat: sending tasks to agent")
				hub.In <- chat.Inbound{
					Channel:  "heartbeat",
					ChatID:   "system",
					SenderID: "heartbeat",
					Content:  "[HEARTBEAT CHECK] Review and execute the explicit pending tasks from HEARTBEAT.md. If no task remains actionable, stop silently and do not send a status response.\n\n" + tasks,
				}
			}
		}
	}()
}

func extractHeartbeatTasks(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	start := 0
	for i, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line), "## Periodic Tasks") {
			start = i + 1
			break
		}
	}

	var tasks []string
	inComment := false
	for _, line := range lines[start:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if inComment {
			if strings.Contains(trimmed, "-->") {
				inComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "<!--") {
			if !strings.Contains(trimmed, "-->") {
				inComment = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		tasks = append(tasks, strings.TrimRight(line, " \t"))
	}

	return strings.TrimSpace(strings.Join(tasks, "\n"))
}
