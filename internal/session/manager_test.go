package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionManagerSaveLoadRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)

	sess := sm.GetOrCreate("telegram-chat-42")
	sess.AddMessage("user", "hello")
	sess.AddMessage("assistant", "hi")
	if err := sm.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	filename, err := encodedSessionFilename(sess.Key)
	if err != nil {
		t.Fatalf("encodedSessionFilename() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "sessions", filename))
	if err != nil {
		t.Fatalf("ReadFile(saved session) error = %v", err)
	}
	var persisted Session
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("json.Unmarshal(saved session) error = %v", err)
	}
	if persisted.Key != sess.Key {
		t.Fatalf("persisted.Key = %q, want %q", persisted.Key, sess.Key)
	}

	reloaded := NewSessionManager(tmp)
	if err := reloaded.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	got := reloaded.GetOrCreate(sess.Key)
	if len(got.History) != 2 {
		t.Fatalf("len(got.History) = %d, want 2", len(got.History))
	}
	if got.History[0] != "user: hello" || got.History[1] != "assistant: hi" {
		t.Fatalf("got.History = %#v, want preserved round trip", got.History)
	}
}

func TestSessionManagerTraversalKeyCannotEscapeSessionsDir(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)
	key := ".." + string(os.PathSeparator) + "escape"

	path, err := sm.sessionFilePath(key)
	if err != nil {
		t.Fatalf("sessionFilePath() error = %v", err)
	}
	sessionsDir := filepath.Join(tmp, "sessions")
	rel, err := filepath.Rel(sessionsDir, path)
	if err != nil {
		t.Fatalf("filepath.Rel() error = %v", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		t.Fatalf("sessionFilePath() escaped sessions dir: rel=%q", rel)
	}

	sess := sm.GetOrCreate(key)
	sess.AddMessage("user", "hello")
	if err := sm.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat(encoded session path) error = %v", err)
	}
	if matches, err := filepath.Glob(filepath.Join(tmp, "*.json")); err != nil {
		t.Fatalf("Glob() error = %v", err)
	} else if len(matches) != 0 {
		t.Fatalf("root-level json files = %v, want none", matches)
	}
}

func TestSessionManagerSeparatorKeyCannotEscapeSessionsDir(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)
	key := "nested" + string(os.PathSeparator) + "child"

	path, err := sm.sessionFilePath(key)
	if err != nil {
		t.Fatalf("sessionFilePath() error = %v", err)
	}
	if filepath.Dir(path) != filepath.Join(tmp, "sessions") {
		t.Fatalf("filepath.Dir(path) = %q, want %q", filepath.Dir(path), filepath.Join(tmp, "sessions"))
	}

	sess := sm.GetOrCreate(key)
	sess.AddMessage("user", "hello")
	if err := sm.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(tmp, "sessions"))
	if err != nil {
		t.Fatalf("ReadDir(sessions) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].IsDir() {
		t.Fatal("saved session entry is a directory, want file")
	}
}

func TestSessionManagerDistinctKeysDoNotCollide(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)

	firstKey := "chat-1"
	secondKey := "chat-2"
	firstName, err := encodedSessionFilename(firstKey)
	if err != nil {
		t.Fatalf("encodedSessionFilename(firstKey) error = %v", err)
	}
	secondName, err := encodedSessionFilename(secondKey)
	if err != nil {
		t.Fatalf("encodedSessionFilename(secondKey) error = %v", err)
	}
	if firstName == secondName {
		t.Fatalf("encoded filenames collided: %q", firstName)
	}

	first := sm.GetOrCreate(firstKey)
	first.AddMessage("user", "one")
	if err := sm.Save(first); err != nil {
		t.Fatalf("Save(first) error = %v", err)
	}

	second := sm.GetOrCreate(secondKey)
	second.AddMessage("user", "two")
	if err := sm.Save(second); err != nil {
		t.Fatalf("Save(second) error = %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(tmp, "sessions"))
	if err != nil {
		t.Fatalf("ReadDir(sessions) error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
}

func TestSessionManagerLoadAllMissingDirectory(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)

	if err := sm.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v, want nil", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "sessions")); err != nil {
		t.Fatalf("Stat(sessions dir) error = %v", err)
	}

	got := sm.GetOrCreate("fresh-session")
	if len(got.History) != 0 {
		t.Fatalf("len(got.History) = %d, want 0", len(got.History))
	}
}

func TestSessionManagerLoadAllCorruptedFileReturnsExplicitError(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, "bad.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile(bad.json) error = %v", err)
	}

	sm := NewSessionManager(tmp)
	err := sm.LoadAll()
	if err == nil {
		t.Fatal("LoadAll() error = nil, want explicit corruption failure")
	}
	if !strings.Contains(err.Error(), "bad.json") {
		t.Fatalf("LoadAll() error = %q, want filename in error", err.Error())
	}
}

func TestSessionManagerLoadAllIgnoresAtomicTempFiles(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, "encoded.json.tmp-123"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile(temp) error = %v", err)
	}

	sm := NewSessionManager(tmp)
	if err := sm.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v, want temp artifact ignored", err)
	}
}

func TestSessionManagerLoadAllReadsLegacyFileInsideSessionsDir(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	want := Session{
		Key:     "legacy-session",
		History: []string{"user: legacy"},
	}
	data, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, want.Key+".json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(legacy) error = %v", err)
	}

	sm := NewSessionManager(tmp)
	if err := sm.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	got := sm.GetOrCreate(want.Key)
	if len(got.History) != 1 || got.History[0] != "user: legacy" {
		t.Fatalf("got.History = %#v, want legacy session history loaded", got.History)
	}
}

func TestSessionManagerLoadAllPrefersEncodedFileWhenLegacyAndEncodedExist(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	key := "legacy-session"
	legacy := Session{Key: key, History: []string{"user: legacy"}}
	encoded := Session{Key: key, History: []string{"user: encoded"}}

	legacyData, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent(legacy) error = %v", err)
	}
	encodedData, err := json.MarshalIndent(encoded, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent(encoded) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, key+".json"), legacyData, 0o644); err != nil {
		t.Fatalf("WriteFile(legacy) error = %v", err)
	}
	encodedName, err := encodedSessionFilename(key)
	if err != nil {
		t.Fatalf("encodedSessionFilename() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, encodedName), encodedData, 0o644); err != nil {
		t.Fatalf("WriteFile(encoded) error = %v", err)
	}

	sm := NewSessionManager(tmp)
	if err := sm.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	got := sm.GetOrCreate(key)
	if len(got.History) != 1 || got.History[0] != "user: encoded" {
		t.Fatalf("got.History = %#v, want encoded file to win", got.History)
	}
}

func TestSessionManagerSaveUsesAtomicWriter(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)

	original := sessionWriteFileAtomic
	t.Cleanup(func() { sessionWriteFileAtomic = original })

	var calledPath string
	var calledData []byte
	sessionWriteFileAtomic = func(path string, data []byte) error {
		calledPath = path
		calledData = append([]byte(nil), data...)
		return os.WriteFile(path, data, 0o644)
	}

	sess := sm.GetOrCreate("telegram-chat-42")
	sess.AddMessage("user", "hello")
	if err := sm.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	filename, err := encodedSessionFilename(sess.Key)
	if err != nil {
		t.Fatalf("encodedSessionFilename() error = %v", err)
	}
	wantPath := filepath.Join(tmp, "sessions", filename)
	if calledPath != wantPath {
		t.Fatalf("atomic writer path = %q, want %q", calledPath, wantPath)
	}
	if len(calledData) == 0 {
		t.Fatal("atomic writer data = empty, want marshaled session bytes")
	}
}

func TestSessionManagerSaveAtomicFailurePreservesExistingFile(t *testing.T) {
	tmp := t.TempDir()
	sm := NewSessionManager(tmp)

	sess := sm.GetOrCreate("telegram-chat-42")
	sess.AddMessage("user", "before")
	if err := sm.Save(sess); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	original := sessionWriteFileAtomic
	t.Cleanup(func() { sessionWriteFileAtomic = original })

	wantErr := errors.New("atomic write failed")
	sessionWriteFileAtomic = func(path string, data []byte) error {
		return wantErr
	}

	sess.History = []string{"user: after"}
	err := sm.Save(sess)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Save() error = %v, want %v", err, wantErr)
	}

	filename, err := encodedSessionFilename(sess.Key)
	if err != nil {
		t.Fatalf("encodedSessionFilename() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(tmp, "sessions", filename))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var persisted Session
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(persisted.History) != 1 || persisted.History[0] != "user: before" {
		t.Fatalf("persisted.History = %#v, want original content preserved", persisted.History)
	}
}
