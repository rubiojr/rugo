package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsRemoteRequire(t *testing.T) {
	tests := []struct {
		path   string
		remote bool
	}{
		// Remote paths
		{"github.com/user/repo", true},
		{"github.com/user/repo@v1.0.0", true},
		{"github.com/user/repo@main", true},
		{"github.com/user/repo/sub@v1.0.0", true},
		{"gitlab.com/org/lib", true},
		{"gitea.example.com/user/mod", true},
		{"localhost/user/repo", true},
		{"localhost:9418/user/repo", true},
		{"localhost/user/repo@v1.0.0", true},

		// Local paths
		{"helpers", false},
		{"lib/math", false},
		{"lib/utils", false},
		{"../shared/helpers", false},
		{"helpers.rg", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isRemoteRequire(tt.path)
			if got != tt.remote {
				t.Errorf("isRemoteRequire(%q) = %v, want %v", tt.path, got, tt.remote)
			}
		})
	}
}

func TestParseRemotePath(t *testing.T) {
	tests := []struct {
		input   string
		host    string
		owner   string
		repo    string
		version string
		subpath string
	}{
		{
			"github.com/user/repo",
			"github.com", "user", "repo", "", "",
		},
		{
			"github.com/user/repo@v1.2.0",
			"github.com", "user", "repo", "v1.2.0", "",
		},
		{
			"github.com/user/repo@main",
			"github.com", "user", "repo", "main", "",
		},
		{
			"github.com/user/repo@abc1234",
			"github.com", "user", "repo", "abc1234", "",
		},
		{
			"github.com/user/repo/sub@v1.0.0",
			"github.com", "user", "repo", "v1.0.0", "sub",
		},
		{
			"github.com/user/repo/deep/sub@main",
			"github.com", "user", "repo", "main", "deep/sub",
		},
		{
			"gitlab.com/org/lib",
			"gitlab.com", "org", "lib", "", "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, err := parseRemotePath(tt.input)
			if err != nil {
				t.Fatalf("parseRemotePath(%q) error: %v", tt.input, err)
			}
			if r.Host != tt.host {
				t.Errorf("host = %q, want %q", r.Host, tt.host)
			}
			if r.Owner != tt.owner {
				t.Errorf("owner = %q, want %q", r.Owner, tt.owner)
			}
			if r.Repo != tt.repo {
				t.Errorf("repo = %q, want %q", r.Repo, tt.repo)
			}
			if r.Version != tt.version {
				t.Errorf("version = %q, want %q", r.Version, tt.version)
			}
			if r.Subpath != tt.subpath {
				t.Errorf("subpath = %q, want %q", r.Subpath, tt.subpath)
			}
		})
	}
}

func TestParseRemotePathErrors(t *testing.T) {
	tests := []string{
		"github.com/user",
		"github.com",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := parseRemotePath(input)
			if err == nil {
				t.Errorf("parseRemotePath(%q) should fail", input)
			}
		})
	}
}

func TestRemotePathCloneURL(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"github.com", "https://github.com/user/repo.git"},
		{"gitlab.com", "https://gitlab.com/user/repo.git"},
		{"localhost", "http://localhost/user/repo.git"},
		{"localhost:9418", "http://localhost:9418/user/repo.git"},
	}
	for _, tt := range tests {
		r := &remotePath{Host: tt.host, Owner: "user", Repo: "repo"}
		if got := r.cloneURL(); got != tt.want {
			t.Errorf("cloneURL(%s) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

func TestRemotePathVersionLabel(t *testing.T) {
	tests := []struct {
		version string
		label   string
	}{
		{"v1.0.0", "v1.0.0"},
		{"main", "main"},
		{"", "_default"},
	}
	for _, tt := range tests {
		r := &remotePath{Version: tt.version}
		if got := r.versionLabel(); got != tt.label {
			t.Errorf("versionLabel(%q) = %q, want %q", tt.version, got, tt.label)
		}
	}
}

func TestRemotePathIsImmutable(t *testing.T) {
	tests := []struct {
		version   string
		immutable bool
	}{
		{"v1.0.0", true},
		{"v0.1.0-beta", true},
		{"abc1234", true},
		{"deadbeef", true},
		{"abcdef1234567890abcdef1234567890abcdef12", true},
		{"main", false},
		{"dev", false},
		{"", false},
	}
	for _, tt := range tests {
		r := &remotePath{Version: tt.version}
		if got := r.isImmutable(); got != tt.immutable {
			t.Errorf("isImmutable(%q) = %v, want %v", tt.version, got, tt.immutable)
		}
	}
}

func TestRemotePathDefaultNamespace(t *testing.T) {
	tests := []struct {
		repo    string
		subpath string
		ns      string
	}{
		{"rugo-utils", "", "rugo_utils"},
		{"mylib", "", "mylib"},
		{"rugo-str-utils", "", "rugo_str_utils"},
		{"repo", "str_utils", "str_utils"},
		{"repo", "deep/sub-mod", "sub_mod"},
	}
	for _, tt := range tests {
		r := &remotePath{Repo: tt.repo, Subpath: tt.subpath}
		if got := r.defaultNamespace(); got != tt.ns {
			t.Errorf("defaultNamespace(repo=%q, sub=%q) = %q, want %q", tt.repo, tt.subpath, got, tt.ns)
		}
	}
}

func TestSanitizeNamespace(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{"rugo-utils", "rugo_utils"},
		{"mylib", "mylib"},
		{"a-b-c", "a_b_c"},
	}
	for _, tt := range tests {
		if got := sanitizeNamespace(tt.in); got != tt.out {
			t.Errorf("sanitizeNamespace(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}

func TestIsSHA(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"abc1234", true},
		{"deadbeef", true},
		{"abcdef1234567890abcdef1234567890abcdef12", true},
		{"ABC1234", false},   // uppercase
		{"abc123", false},    // too short (6)
		{"main", false},      // not hex
		{"v1.0.0", false},    // not hex
		{"", false},          // empty
		{"abc12g4", false},   // non-hex char
	}
	for _, tt := range tests {
		if got := isSHA(tt.s); got != tt.want {
			t.Errorf("isSHA(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestFindEntryPoint(t *testing.T) {
	t.Run("finds repo-name.rg", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "mylib.rg"), []byte("def foo()\nend\n"), 0644)
		os.WriteFile(filepath.Join(dir, "README.md"), []byte("# mylib"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		got, err := findEntryPoint(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != "mylib.rg" {
			t.Errorf("got %q, want mylib.rg", got)
		}
	})

	t.Run("falls back to main.rg", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "main.rg"), []byte("def foo()\nend\n"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		got, err := findEntryPoint(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != "main.rg" {
			t.Errorf("got %q, want main.rg", got)
		}
	})

	t.Run("falls back to single .rg file", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "helpers.rg"), []byte("def foo()\nend\n"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		got, err := findEntryPoint(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != "helpers.rg" {
			t.Errorf("got %q, want helpers.rg", got)
		}
	})

	t.Run("errors on multiple .rg files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "a.rg"), []byte("def a()\nend\n"), 0644)
		os.WriteFile(filepath.Join(dir, "b.rg"), []byte("def b()\nend\n"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		_, err := findEntryPoint(dir, r)
		if err == nil {
			t.Error("expected error for ambiguous entry point")
		}
	})

	t.Run("errors on no .rg files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "README.md"), []byte("# mylib"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		_, err := findEntryPoint(dir, r)
		if err == nil {
			t.Error("expected error for no .rg files")
		}
	})

	t.Run("subpath resolution", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "utils")
		os.MkdirAll(sub, 0755)
		os.WriteFile(filepath.Join(sub, "utils.rg"), []byte("def foo()\nend\n"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "repo", Subpath: "utils"}
		got, err := findEntryPoint(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != "utils.rg" {
			t.Errorf("got %q, want utils.rg", got)
		}
	})
}

func TestNeedsFetch(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "cached")
	os.MkdirAll(existing, 0755)

	// Immutable + exists → no fetch
	r := &remotePath{Version: "v1.0.0"}
	if needsFetch(existing, r) {
		t.Error("immutable + cached should not need fetch")
	}

	// Immutable + missing → fetch
	if !needsFetch(filepath.Join(dir, "missing"), r) {
		t.Error("immutable + missing should need fetch")
	}

	// Mutable + exists → fetch (always re-fetch branches)
	r = &remotePath{Version: "main"}
	if !needsFetch(existing, r) {
		t.Error("mutable + cached should need fetch")
	}

	// No version + exists → fetch
	r = &remotePath{}
	if !needsFetch(existing, r) {
		t.Error("no version + cached should need fetch")
	}
}
