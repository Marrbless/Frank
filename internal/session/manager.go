package session

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/local/picobot/internal/missioncontrol"
)

// MaxHistorySize is the maximum number of messages kept in a session.
// Older messages are trimmed on save to keep the session file small
// and avoid blowing up the LLM context window.
// Important information should be persisted via write_memory, not session history.
const MaxHistorySize = 50

// Session holds a short chat history.
type Session struct {
	Key     string
	History []string
}

// SessionManager stores sessions in memory and persists to disk under workspace.
type SessionManager struct {
	mu        sync.RWMutex
	sessions  map[string]*Session
	workspace string
}

type loadedSession struct {
	session  *Session
	filename string
}

var sessionWriteFileAtomic = func(path string, data []byte) error {
	return missioncontrol.WriteStoreFileAtomicMode(path, data, 0o644)
}

func NewSessionManager(workspace string) *SessionManager {
	return &SessionManager{sessions: make(map[string]*Session), workspace: workspace}
}

func (sm *SessionManager) sessionsDir() string {
	return filepath.Join(sm.workspace, "sessions")
}

func encodedSessionFilename(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("session key cannot be empty")
	}
	return base64.RawURLEncoding.EncodeToString([]byte(key)) + ".json", nil
}

func isAtomicTempSessionFile(name string) bool {
	return strings.Contains(name, ".json.tmp-")
}

func ensurePathInsideDir(root, path string) error {
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return fmt.Errorf("resolve session path relative to %q: %w", cleanRoot, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("session path %q escapes sessions directory %q", cleanPath, cleanRoot)
	}
	return nil
}

func (sm *SessionManager) sessionFilePath(key string) (string, error) {
	filename, err := encodedSessionFilename(key)
	if err != nil {
		return "", err
	}
	dir := sm.sessionsDir()
	path := filepath.Join(dir, filename)
	if err := ensurePathInsideDir(dir, path); err != nil {
		return "", err
	}
	return path, nil
}

func (sm *SessionManager) GetOrCreate(key string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.sessions[key]; ok {
		return s
	}
	s := &Session{Key: key, History: make([]string, 0)}
	sm.sessions[key] = s
	return s
}

func (sm *SessionManager) Save(s *Session) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	// Trim history to the most recent messages
	s.trim()
	path := sm.sessionsDir()
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	fpath, err := sm.sessionFilePath(s.Key)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return sessionWriteFileAtomic(fpath, b)
}

func (sm *SessionManager) LoadAll() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	path := sm.sessionsDir()
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	loaded := make(map[string]loadedSession, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if isAtomicTempSessionFile(e.Name()) {
			continue
		}
		filePath := filepath.Join(path, e.Name())
		if err := ensurePathInsideDir(path, filePath); err != nil {
			return fmt.Errorf("load session %q: %w", e.Name(), err)
		}
		b, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read session %q: %w", e.Name(), err)
		}
		var s Session
		if err := json.Unmarshal(b, &s); err != nil {
			return fmt.Errorf("decode session %q: %w", e.Name(), err)
		}
		if s.Key == "" {
			return fmt.Errorf("decode session %q: missing session key", e.Name())
		}
		expectedFilename, err := encodedSessionFilename(s.Key)
		if err != nil {
			return fmt.Errorf("decode session %q: %w", e.Name(), err)
		}
		existing, ok := loaded[s.Key]
		if !ok {
			loaded[s.Key] = loadedSession{session: &s, filename: e.Name()}
			continue
		}
		switch {
		case existing.filename == expectedFilename && e.Name() != expectedFilename:
			continue
		case existing.filename != expectedFilename && e.Name() == expectedFilename:
			loaded[s.Key] = loadedSession{session: &s, filename: e.Name()}
		default:
			return fmt.Errorf("duplicate session records for key %q: %q and %q", s.Key, existing.filename, e.Name())
		}
	}
	sm.sessions = make(map[string]*Session, len(loaded))
	for key, loadedSession := range loaded {
		sm.sessions[key] = loadedSession.session
	}
	return nil
}

func (s *Session) AddMessage(role, content string) {
	s.History = append(s.History, role+": "+content)
}

// GetHistory returns the session history.
func (s *Session) GetHistory() []string {
	return s.History
}

// trim keeps only the last MaxHistorySize messages, discarding the oldest.
func (s *Session) trim() {
	if len(s.History) > MaxHistorySize {
		s.History = s.History[len(s.History)-MaxHistorySize:]
	}
}
