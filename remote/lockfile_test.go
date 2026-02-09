package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockFileReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rugo.lock")

	// Write a lock file.
	lf := NewLockFile()
	lf.Set("github.com/user/repo", "v1.0.0", "abc1234def5678901234567890abcdef12345678")
	lf.Set("github.com/user/utils", "main", "9f8e7d6c5b4a321098765432109876fedcba9876")
	lf.Set("github.com/other/lib", "_default", "1111111222222233333334444444555555566666")

	if err := WriteLockFile(path, lf); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	// Read it back.
	lf2, err := ReadLockFile(path)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}

	if len(lf2.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(lf2.Entries))
	}

	// Verify lookup.
	e := lf2.Lookup("github.com/user/repo", "v1.0.0")
	if e == nil || e.SHA != "abc1234def5678901234567890abcdef12345678" {
		t.Errorf("lookup v1.0.0 failed: %+v", e)
	}

	e = lf2.Lookup("github.com/user/utils", "main")
	if e == nil || e.SHA != "9f8e7d6c5b4a321098765432109876fedcba9876" {
		t.Errorf("lookup main failed: %+v", e)
	}

	e = lf2.Lookup("github.com/other/lib", "_default")
	if e == nil || e.SHA != "1111111222222233333334444444555555566666" {
		t.Errorf("lookup _default failed: %+v", e)
	}

	// Missing entry.
	if lf2.Lookup("github.com/nobody/nothing", "v1.0.0") != nil {
		t.Error("expected nil for missing entry")
	}
}

func TestLockFileReadMissing(t *testing.T) {
	lf, err := ReadLockFile("/nonexistent/rugo.lock")
	if err != nil {
		t.Fatalf("ReadLockFile missing: %v", err)
	}
	if len(lf.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(lf.Entries))
	}
}

func TestLockFileCommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rugo.lock")

	content := `# rugo.lock — auto-generated, do not edit

github.com/user/repo v1.0.0 abc1234

# comment mid-file
github.com/other/lib main def5678

`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	lf, err := ReadLockFile(path)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}
	if len(lf.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(lf.Entries))
	}
}

func TestLockFileMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rugo.lock")

	content := "github.com/user/repo v1.0.0\n" // only 2 fields
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadLockFile(path)
	if err == nil {
		t.Error("expected error for malformed lock file")
	}
}

func TestLockFileSetUpdate(t *testing.T) {
	lf := NewLockFile()
	lf.Set("github.com/user/repo", "main", "aaa1111")
	lf.Set("github.com/user/repo", "main", "bbb2222")

	if len(lf.Entries) != 1 {
		t.Fatalf("expected 1 entry after update, got %d", len(lf.Entries))
	}
	e := lf.Lookup("github.com/user/repo", "main")
	if e.SHA != "bbb2222" {
		t.Errorf("expected updated SHA bbb2222, got %s", e.SHA)
	}
}

func TestLockFileEmptyRemovesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rugo.lock")

	// Create the file first.
	if err := os.WriteFile(path, []byte("# empty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	lf := NewLockFile()
	if err := WriteLockFile(path, lf); err != nil {
		t.Fatalf("WriteLockFile empty: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected lock file to be removed when empty")
	}
}
