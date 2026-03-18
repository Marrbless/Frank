package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemTool provides read/write/list/stat operations within the filesystem.
// All operations are sandboxed to the workspace directory using os.Root (Go 1.24+),
// which provides kernel-enforced path containment via openat() syscalls.
// This prevents symlink escapes, TOCTOU races, and path traversal attacks.
type FilesystemTool struct {
	root      *os.Root
	taskState *TaskState
}

// NewFilesystemTool opens an os.Root anchored at workspaceDir.
func NewFilesystemTool(workspaceDir string) (*FilesystemTool, error) {
	return NewFilesystemToolWithState(workspaceDir, nil)
}

// NewFilesystemToolWithState opens an os.Root anchored at workspaceDir and shares task state.
func NewFilesystemToolWithState(workspaceDir string, taskState *TaskState) (*FilesystemTool, error) {
	absDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("filesystem: resolve workspace path: %w", err)
	}
	root, err := os.OpenRoot(absDir)
	if err != nil {
		return nil, fmt.Errorf("filesystem: open workspace root: %w", err)
	}
	return &FilesystemTool{root: root, taskState: taskState}, nil
}

// Close releases the underlying os.Root file descriptor.
func (t *FilesystemTool) Close() error {
	return t.root.Close()
}

func (t *FilesystemTool) Name() string { return "filesystem" }
func (t *FilesystemTool) Description() string {
	return "Read, write, list, and stat files in the workspace"
}

func (t *FilesystemTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The filesystem operation to perform",
				"enum":        []string{"read", "write", "list", "stat"},
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The file or directory path (relative to workspace)",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write (required when action is 'write')",
			},
		},
		"required": []string{"action"},
	}
}

func (t *FilesystemTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	_ = ctx

	actionRaw, ok := args["action"]
	if !ok {
		return "", fmt.Errorf("filesystem: 'action' is required")
	}
	action, ok := actionRaw.(string)
	if !ok {
		return "", fmt.Errorf("filesystem: 'action' must be a string")
	}

	pathStr := ""
	if pathRaw, ok := args["path"]; ok && pathRaw != nil {
		switch v := pathRaw.(type) {
		case string:
			pathStr = v
		default:
			return "", fmt.Errorf("filesystem: 'path' must be a string")
		}
	}

	if pathStr == "" {
		switch action {
		case "list", "stat":
			pathStr = "."
		default:
			return "", fmt.Errorf("filesystem: 'path' is required for %s", action)
		}
	}

	cleanPath := filepath.Clean(pathStr)

	if action == "write" && t.taskState != nil {
		if cleanPath == "projects/current" || strings.HasPrefix(cleanPath, "projects/current/") {
			if !t.taskState.ProjectInitialized() {
				return "", fmt.Errorf("filesystem: current project not initialized for this task; call frank_new_project first")
			}
		}
	}

	switch action {
	case "read":
		b, err := t.root.ReadFile(cleanPath)
		if err != nil {
			return "", err
		}
		return string(b), nil

	case "write":
		contentRaw, ok := args["content"]
		if !ok {
			return "", fmt.Errorf("filesystem: 'content' is required for write")
		}
		content, ok := contentRaw.(string)
		if !ok {
			return "", fmt.Errorf("filesystem: 'content' must be a string")
		}

		dir := filepath.Dir(cleanPath)
		if dir != "." {
			if err := t.root.MkdirAll(dir, 0o755); err != nil {
				return "", err
			}
		}
		if err := t.root.WriteFile(cleanPath, []byte(content), 0o644); err != nil {
			return "", err
		}
		return "written", nil

	case "list":
		f, err := t.root.Open(cleanPath)
		if err != nil {
			return "", err
		}
		defer func() { _ = f.Close() }()

		info, err := f.Stat()
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", fmt.Errorf("filesystem: path is a file; use read or stat")
		}

		entries, err := f.ReadDir(-1)
		if err != nil {
			return "", err
		}

		var out strings.Builder
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				name += "/"
			}
			out.WriteString(name)
			out.WriteByte('\n')
		}
		return out.String(), nil

	case "stat":
		f, err := t.root.Open(cleanPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "exists=false\n", nil
			}
			return "", err
		}
		defer func() { _ = f.Close() }()

		info, err := f.Stat()
		if err != nil {
			return "", err
		}

		kind := "file"
		if info.IsDir() {
			kind = "dir"
		}

		return fmt.Sprintf("exists=true\nkind=%s\nname=%s\nsize=%d\n", kind, info.Name(), info.Size()), nil

	default:
		return "", fmt.Errorf("filesystem: unknown action %s", action)
	}
}
