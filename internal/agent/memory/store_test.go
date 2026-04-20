package memory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMemoryAddAndRecent(t *testing.T) {
	s := NewMemoryStore(3)
	s.AddLong("L1")
	s.AddShort("two")
	s.AddShort("one")

	res := s.Recent(10)
	if len(res) != 3 {
		t.Fatalf("expected 3 items, got %d", len(res))
	}
	if res[0].Text != "one" || res[1].Text != "two" || res[2].Text != "L1" {
		t.Fatalf("unexpected recent order: %v", res)
	}
}

func TestShortLimit(t *testing.T) {
	s := NewMemoryStore(2)
	s.AddShort("c")
	time.Sleep(5 * time.Millisecond)
	s.AddShort("b")
	time.Sleep(5 * time.Millisecond)
	s.AddShort("a")

	res := s.Recent(10)
	if len(res) != 2 {
		t.Fatalf("expected 2 items due to limit, got %d", len(res))
	}
	if res[0].Text != "a" || res[1].Text != "b" {
		t.Fatalf("unexpected recent after limit: %v", res)
	}
}

func TestQueryByKeyword(t *testing.T) {
	s := NewMemoryStore(10)
	s.AddLong("apple pie recipe")
	s.AddShort("Remember the apple")

	res := s.QueryByKeyword("apple", 10)
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].Text != "Remember the apple" || res[1].Text != "apple pie recipe" {
		t.Fatalf("unexpected query order: %v", res)
	}
}

func TestWriteLongTermUsesAtomicWriter(t *testing.T) {
	tmp := t.TempDir()
	s := NewMemoryStoreWithWorkspace(tmp, 10)

	original := memoryWriteFileAtomic
	t.Cleanup(func() { memoryWriteFileAtomic = original })

	var calledPath string
	var calledData []byte
	memoryWriteFileAtomic = func(path string, data []byte) error {
		calledPath = path
		calledData = append([]byte(nil), data...)
		return os.WriteFile(path, data, 0o644)
	}

	if err := s.WriteLongTerm("hello"); err != nil {
		t.Fatalf("WriteLongTerm() error = %v", err)
	}

	wantPath := filepath.Join(tmp, "memory", "MEMORY.md")
	if calledPath != wantPath {
		t.Fatalf("atomic writer path = %q, want %q", calledPath, wantPath)
	}
	if string(calledData) != "hello" {
		t.Fatalf("atomic writer data = %q, want %q", string(calledData), "hello")
	}
}

func TestWriteFileUsesAtomicWriter(t *testing.T) {
	tmp := t.TempDir()
	s := NewMemoryStoreWithWorkspace(tmp, 10)

	original := memoryWriteFileAtomic
	t.Cleanup(func() { memoryWriteFileAtomic = original })

	var calledPath string
	memoryWriteFileAtomic = func(path string, data []byte) error {
		calledPath = path
		return os.WriteFile(path, data, 0o644)
	}

	if err := s.WriteFile("2026-01-15.md", "hello"); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	wantPath := filepath.Join(tmp, "memory", "2026-01-15.md")
	if calledPath != wantPath {
		t.Fatalf("atomic writer path = %q, want %q", calledPath, wantPath)
	}
}

func TestWriteLongTermAtomicFailurePreservesExistingContent(t *testing.T) {
	tmp := t.TempDir()
	s := NewMemoryStoreWithWorkspace(tmp, 10)

	if err := s.WriteLongTerm("before"); err != nil {
		t.Fatalf("WriteLongTerm(initial) error = %v", err)
	}

	original := memoryWriteFileAtomic
	t.Cleanup(func() { memoryWriteFileAtomic = original })

	wantErr := errors.New("atomic write failed")
	memoryWriteFileAtomic = func(path string, data []byte) error {
		return wantErr
	}

	err := s.WriteLongTerm("after")
	if !errors.Is(err, wantErr) {
		t.Fatalf("WriteLongTerm() error = %v, want %v", err, wantErr)
	}

	got, err := s.ReadLongTerm()
	if err != nil {
		t.Fatalf("ReadLongTerm() error = %v", err)
	}
	if got != "before" {
		t.Fatalf("ReadLongTerm() = %q, want %q", got, "before")
	}
}

func TestWriteFileAtomicFailurePreservesExistingContent(t *testing.T) {
	tmp := t.TempDir()
	s := NewMemoryStoreWithWorkspace(tmp, 10)

	if err := s.WriteFile("2026-01-15.md", "before"); err != nil {
		t.Fatalf("WriteFile(initial) error = %v", err)
	}

	original := memoryWriteFileAtomic
	t.Cleanup(func() { memoryWriteFileAtomic = original })

	wantErr := errors.New("atomic write failed")
	memoryWriteFileAtomic = func(path string, data []byte) error {
		return wantErr
	}

	err := s.WriteFile("2026-01-15.md", "after")
	if !errors.Is(err, wantErr) {
		t.Fatalf("WriteFile() error = %v, want %v", err, wantErr)
	}

	got, err := s.ReadFile("2026-01-15.md")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got != "before" {
		t.Fatalf("ReadFile() = %q, want %q", got, "before")
	}
}

func TestAppendTodaySyncFailureReturnsError(t *testing.T) {
	tmp := t.TempDir()
	s := NewMemoryStoreWithWorkspace(tmp, 10)

	original := memorySyncFile
	t.Cleanup(func() { memorySyncFile = original })

	wantErr := errors.New("sync failed")
	memorySyncFile = func(f *os.File) error {
		return wantErr
	}

	err := s.AppendToday("note 1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("AppendToday() error = %v, want %v", err, wantErr)
	}
}

func TestAppendLongTermRetryIsIdempotentAtTail(t *testing.T) {
	tmp := t.TempDir()
	s := NewMemoryStoreWithWorkspace(tmp, 10)

	if err := s.AppendLongTerm("LT1"); err != nil {
		t.Fatalf("AppendLongTerm(first) error = %v", err)
	}
	if err := s.AppendLongTerm("LT1"); err != nil {
		t.Fatalf("AppendLongTerm(retry) error = %v", err)
	}

	got, err := s.ReadLongTerm()
	if err != nil {
		t.Fatalf("ReadLongTerm() error = %v", err)
	}
	if strings.Count(got, "LT1") != 1 {
		t.Fatalf("ReadLongTerm() = %q, want LT1 appended once", got)
	}
}

func TestAppendLongTermConcurrentAppendsPreserveBothValues(t *testing.T) {
	tmp := t.TempDir()
	s := NewMemoryStoreWithWorkspace(tmp, 10)

	values := []string{"alpha", "beta"}
	errCh := make(chan error, len(values))
	var wg sync.WaitGroup
	for _, value := range values {
		value := value
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- s.AppendLongTerm(value)
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("AppendLongTerm() error = %v", err)
		}
	}

	got, err := s.ReadLongTerm()
	if err != nil {
		t.Fatalf("ReadLongTerm() error = %v", err)
	}
	for _, value := range values {
		if !strings.Contains(got, value) {
			t.Fatalf("ReadLongTerm() = %q, want value %q preserved", got, value)
		}
	}
}
