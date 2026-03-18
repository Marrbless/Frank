package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExecTool runs commands with a timeout.
// Safety model:
// - prefer array form: {"cmd": ["ls", "-la"]}
// - string form (shell) is disallowed to avoid shell injection
// - blacklist dangerous program names (rm, sudo, dd, mkfs, shutdown, reboot)
// - arguments containing absolute paths, ~, or .. are rejected
// - optional cwd enforces a working directory relative to the workspace
// - native frank_* helpers are handled internally and do not rely on shell scripts
type ExecTool struct {
	timeout    time.Duration
	allowedDir string
	taskState  *TaskState
}

func NewExecTool(timeoutSecs int) *ExecTool {
	return &ExecTool{timeout: time.Duration(timeoutSecs) * time.Second}
}

// NewExecToolWithWorkspace creates an ExecTool restricted to the provided workspace directory.
func NewExecToolWithWorkspace(timeoutSecs int, allowedDir string) *ExecTool {
	return &ExecTool{timeout: time.Duration(timeoutSecs) * time.Second, allowedDir: allowedDir}
}

// NewExecToolWithWorkspaceAndState creates an ExecTool restricted to the provided workspace directory
// and sharing per-task state.
func NewExecToolWithWorkspaceAndState(timeoutSecs int, allowedDir string, taskState *TaskState) *ExecTool {
	return &ExecTool{
		timeout:    time.Duration(timeoutSecs) * time.Second,
		allowedDir: allowedDir,
		taskState:  taskState,
	}
}

func (t *ExecTool) Name() string { return "exec" }
func (t *ExecTool) Description() string {
	return "Execute commands (array form only, restricted for safety) with native support for frank_* helpers"
}

func (t *ExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"cmd": map[string]interface{}{
				"type":        "array",
				"description": "Command as array [program, arg1, arg2, ...]. String form is disallowed for security.",
				"items": map[string]interface{}{
					"type": "string",
				},
				"minItems": 1,
			},
			"cwd": map[string]interface{}{
				"type":        "string",
				"description": "Optional working directory relative to the workspace",
			},
		},
		"required": []string{"cmd"},
	}
}

var dangerous = map[string]struct{}{
	"rm":       {},
	"sudo":     {},
	"dd":       {},
	"mkfs":     {},
	"shutdown": {},
	"reboot":   {},
}

func isDangerousProg(prog string) bool {
	base := filepath.Base(prog)
	base = strings.ToLower(base)
	_, ok := dangerous[base]
	return ok
}

func hasUnsafeArg(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~") || strings.Contains(s, "..") {
		return true
	}
	return false
}

func isFrankHelper(prog string) bool {
	switch prog {
	case "frank_new_project", "frank_finish", "frank_py_finish", "frank_py_run", "frank_sshd":
		return true
	default:
		return false
	}
}

func trimOutput(s string) string {
	return strings.TrimRight(s, "\n")
}

func (t *ExecTool) workspaceDir() string {
	if t.allowedDir != "" {
		return t.allowedDir
	}
	return "."
}

func (t *ExecTool) projectsRoot() string {
	return filepath.Join(t.workspaceDir(), "projects")
}

func (t *ExecTool) currentDir() string {
	return filepath.Join(t.projectsRoot(), "current")
}

func (t *ExecTool) archiveDir() string {
	return filepath.Join(t.projectsRoot(), "archive")
}

func (t *ExecTool) currentProjectName() string {
	b, err := os.ReadFile(filepath.Join(t.currentDir(), ".project_name"))
	if err != nil {
		return "current"
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "current"
	}
	return s
}

func sanitizeSlug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "task"
	}
	var out strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			out.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				out.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(out.String(), "-")
	if slug == "" {
		return "task"
	}
	return slug
}

func (t *ExecTool) archiveCurrent() error {
	cur := t.currentDir()
	info, err := os.Stat(cur)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("exec: projects/current is not a directory")
	}

	entries, err := os.ReadDir(cur)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.RemoveAll(cur)
	}

	if err := os.MkdirAll(t.archiveDir(), 0o755); err != nil {
		return err
	}

	name := t.currentProjectName()
	if name == "" || name == "current" {
		name = fmt.Sprintf("project-%s-carryover", time.Now().Format("20060102-150405"))
	}
	dst := filepath.Join(t.archiveDir(), name)
	_ = os.RemoveAll(dst)
	return os.Rename(cur, dst)
}

func (t *ExecTool) sweepWorkspaceStraysToCurrent() error {
	if t.allowedDir == "" {
		return nil
	}
	cur := t.currentDir()
	if err := os.MkdirAll(cur, 0o755); err != nil {
		return err
	}

	skip := map[string]struct{}{
		"AGENTS.md":    {},
		"USER.md":      {},
		"SOUL.md":      {},
		"HEARTBEAT.md": {},
		"projects":     {},
		"bin":          {},
		"memory":       {},
		"skills":       {},
	}

	entries, err := os.ReadDir(t.allowedDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if _, ok := skip[name]; ok {
			continue
		}
		src := filepath.Join(t.allowedDir, name)
		dst := filepath.Join(cur, name)
		_ = os.RemoveAll(dst)
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func (t *ExecTool) normalizeCurrentFileArg(arg string) (string, error) {
	if arg == "" {
		return "", fmt.Errorf("exec: missing file path")
	}
	if hasUnsafeArg(arg) {
		return "", fmt.Errorf("exec: argument '%s' looks unsafe", arg)
	}

	clean := filepath.Clean(arg)
	clean = strings.TrimPrefix(clean, "./")

	prefix := "projects/current"
	if clean == prefix {
		return "", fmt.Errorf("exec: expected a file under projects/current, got the directory itself")
	}
	if strings.HasPrefix(clean, prefix+"/") {
		clean = strings.TrimPrefix(clean, prefix+"/")
	}
	if strings.Contains(clean, string(filepath.Separator)) {
		clean = filepath.Base(clean)
	}
	if clean == "" || clean == "." {
		return "", fmt.Errorf("exec: invalid file path '%s'", arg)
	}
	return clean, nil
}

func (t *ExecTool) currentFiles() []string {
	cur := t.currentDir()
	entries, err := os.ReadDir(cur)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name() == ".project_name" {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func formatProjectResult(name string, files []string, valid string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "NAME=%s\n", name)
	fmt.Fprintf(&b, "REL=projects/current\n")
	if len(files) > 0 {
		fmt.Fprintf(&b, "FILES=%s\n", strings.Join(files, ","))
	}
	if valid != "" {
		fmt.Fprintf(&b, "VALID=%s\n", valid)
	}
	return trimOutput(b.String())
}

func (t *ExecTool) requireProjectInitialized() error {
	if t.taskState != nil && !t.taskState.ProjectInitialized() {
		return fmt.Errorf("exec: current project not initialized for this task; call frank_new_project first")
	}
	return nil
}

func (t *ExecTool) executeFrankHelper(ctx context.Context, prog string, argv []string) (string, error) {
	switch prog {
	case "frank_new_project":
		return t.handleFrankNewProject(argv)
	case "frank_finish":
		return t.handleFrankFinish(argv)
	case "frank_py_finish":
		return t.handleFrankPyFinish(ctx, argv)
	case "frank_py_run":
		return t.handleFrankPyRun(ctx, argv)
	case "frank_sshd":
		return t.handleFrankSSHD(ctx, argv)
	default:
		return "", fmt.Errorf("exec: unknown helper %s", prog)
	}
}

func (t *ExecTool) handleFrankNewProject(argv []string) (string, error) {
	if t.allowedDir == "" {
		return "", fmt.Errorf("exec: frank_new_project requires workspace-restricted exec tool")
	}

	if t.taskState != nil && t.taskState.ProjectInitialized() {
		if _, err := os.Stat(t.currentDir()); err == nil {
			return fmt.Sprintf("NAME=%s\nREL=projects/current", t.currentProjectName()), nil
		}
	}

	slug := "task"
	if len(argv) > 0 && strings.TrimSpace(argv[0]) != "" {
		slug = argv[0]
	}
	slug = sanitizeSlug(slug)

	if err := os.MkdirAll(t.projectsRoot(), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(t.archiveDir(), 0o755); err != nil {
		return "", err
	}
	if err := t.archiveCurrent(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(t.currentDir(), 0o755); err != nil {
		return "", err
	}

	name := fmt.Sprintf("project-%s-%s", time.Now().Format("20060102-150405"), slug)
	if err := os.WriteFile(filepath.Join(t.currentDir(), ".project_name"), []byte(name+"\n"), 0o644); err != nil {
		return "", err
	}

	if t.taskState != nil {
		t.taskState.MarkProjectInitialized()
	}

	return fmt.Sprintf("NAME=%s\nREL=projects/current", name), nil
}

func (t *ExecTool) handleFrankFinish(argv []string) (string, error) {
	if err := t.requireProjectInitialized(); err != nil {
		return "", err
	}
	if len(argv) < 1 {
		return "", fmt.Errorf("exec: frank_finish requires a file path")
	}
	if err := t.sweepWorkspaceStraysToCurrent(); err != nil {
		return "", err
	}

	rel, err := t.normalizeCurrentFileArg(argv[0])
	if err != nil {
		return "", err
	}
	abs := filepath.Join(t.currentDir(), rel)

	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("exec: %s is a directory", rel)
	}

	return formatProjectResult(t.currentProjectName(), t.currentFiles(), "projects/current/"+rel), nil
}

func (t *ExecTool) handleFrankPyFinish(ctx context.Context, argv []string) (string, error) {
	if err := t.requireProjectInitialized(); err != nil {
		return "", err
	}
	if len(argv) < 1 {
		return "", fmt.Errorf("exec: frank_py_finish requires a file path")
	}
	if err := t.sweepWorkspaceStraysToCurrent(); err != nil {
		return "", err
	}

	rel, err := t.normalizeCurrentFileArg(argv[0])
	if err != nil {
		return "", err
	}
	abs := filepath.Join(t.currentDir(), rel)

	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("exec: %s is a directory", rel)
	}

	cmd := exec.CommandContext(ctx, "python3", "-m", "py_compile", rel)
	cmd.Dir = t.currentDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return trimOutput(string(out)), fmt.Errorf("exec error: %w", err)
	}

	return formatProjectResult(t.currentProjectName(), t.currentFiles(), "projects/current/"+rel), nil
}

func (t *ExecTool) handleFrankPyRun(ctx context.Context, argv []string) (string, error) {
	if err := t.requireProjectInitialized(); err != nil {
		return "", err
	}
	if len(argv) < 1 {
		return "", fmt.Errorf("exec: frank_py_run requires a file path")
	}
	if err := t.sweepWorkspaceStraysToCurrent(); err != nil {
		return "", err
	}

	rel, err := t.normalizeCurrentFileArg(argv[0])
	if err != nil {
		return "", err
	}
	abs := filepath.Join(t.currentDir(), rel)

	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("exec: %s is a directory", rel)
	}

	cmd := exec.CommandContext(ctx, "python3", rel)
	cmd.Dir = t.currentDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return trimOutput(string(out)), fmt.Errorf("exec error: %w", err)
	}
	return trimOutput(string(out)), nil
}

func (t *ExecTool) handleFrankSSHD(ctx context.Context, argv []string) (string, error) {
	if len(argv) < 1 {
		return "", fmt.Errorf("exec: frank_sshd requires one of start|status|stop")
	}

	switch argv[0] {
	case "start":
		if err := exec.CommandContext(ctx, "pgrep", "-x", "sshd").Run(); err == nil {
			return "", nil
		}
		out, err := exec.CommandContext(ctx, "sshd").CombinedOutput()
		if err != nil {
			return trimOutput(string(out)), fmt.Errorf("exec error: %w", err)
		}
		return trimOutput(string(out)), nil

	case "status":
		lines := []string{}

		if out, err := exec.CommandContext(ctx, "pgrep", "-af", "sshd").CombinedOutput(); len(out) > 0 {
			lines = append(lines, trimOutput(string(out)))
			if err != nil && !errors.As(err, new(*exec.ExitError)) {
				return trimOutput(string(out)), fmt.Errorf("exec error: %w", err)
			}
		}

		port := "8022"
		if out, err := exec.CommandContext(ctx, "sshd", "-T").CombinedOutput(); err == nil || len(out) > 0 {
			for _, line := range strings.Split(string(out), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 && fields[0] == "port" {
					port = fields[1]
					break
				}
			}
		}

		lines = append(lines, "PORT="+port)
		return strings.Join(lines, "\n"), nil

	case "stop":
		out, err := exec.CommandContext(ctx, "pkill", "-x", "sshd").CombinedOutput()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
				return trimOutput(string(out)), nil
			}
			return trimOutput(string(out)), fmt.Errorf("exec error: %w", err)
		}
		return trimOutput(string(out)), nil

	default:
		return "", fmt.Errorf("exec: frank_sshd requires one of start|status|stop")
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	cmdRaw, ok := args["cmd"]
	if !ok {
		return "", fmt.Errorf("exec: 'cmd' argument required")
	}

	if _, ok := cmdRaw.(string); ok {
		return "", errors.New("exec: string commands are disallowed; use array form")
	}

	var argv []string
	switch v := cmdRaw.(type) {
	case []interface{}:
		if len(v) == 0 {
			return "", fmt.Errorf("exec: empty cmd array")
		}
		for _, a := range v {
			s, ok := a.(string)
			if !ok {
				return "", fmt.Errorf("exec: cmd array must contain strings only")
			}
			argv = append(argv, s)
		}
	case []string:
		if len(v) == 0 {
			return "", fmt.Errorf("exec: empty cmd array")
		}
		argv = append(argv, v...)
	default:
		return "", fmt.Errorf("exec: unsupported cmd type")
	}

	cwd := ""
	if cwdRaw, ok := args["cwd"]; ok && cwdRaw != nil {
		s, ok := cwdRaw.(string)
		if !ok {
			return "", fmt.Errorf("exec: 'cwd' must be a string")
		}
		cwd = s
	}

	cctx := ctx
	if t.timeout > 0 {
		var cancel context.CancelFunc
		cctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	prog := argv[0]
	if isFrankHelper(prog) {
		return t.executeFrankHelper(cctx, prog, argv[1:])
	}

	if isDangerousProg(prog) {
		return "", fmt.Errorf("exec: program '%s' is disallowed", prog)
	}
	for _, a := range argv[1:] {
		if hasUnsafeArg(a) {
			return "", fmt.Errorf("exec: argument '%s' looks unsafe", a)
		}
	}
	if cwd != "" && hasUnsafeArg(cwd) {
		return "", fmt.Errorf("exec: cwd '%s' looks unsafe", cwd)
	}

	cmd := exec.CommandContext(cctx, prog, argv[1:]...)
	switch {
	case cwd != "" && t.allowedDir != "":
		cmd.Dir = filepath.Join(t.allowedDir, filepath.Clean(cwd))
	case t.allowedDir != "":
		cmd.Dir = t.allowedDir
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return trimOutput(string(out)), fmt.Errorf("exec error: %w", err)
	}
	return trimOutput(string(out)), nil
}
