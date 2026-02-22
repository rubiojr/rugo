package remote

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
		{"helpers.rugo", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsRemoteRequire(tt.path)
			if got != tt.remote {
				t.Errorf("IsRemoteRequire(%q) = %v, want %v", tt.path, got, tt.remote)
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
		{
			"github.com/user/repo@latest",
			"github.com", "user", "repo", "latest", "",
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
		{"latest", false},
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
		{"ABC1234", false}, // uppercase
		{"abc123", false},  // too short (6)
		{"main", false},    // not hex
		{"v1.0.0", false},  // not hex
		{"", false},        // empty
		{"abc12g4", false}, // non-hex char
	}
	for _, tt := range tests {
		if got := isSHA(tt.s); got != tt.want {
			t.Errorf("isSHA(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestFindEntryPoint(t *testing.T) {
	t.Run("finds repo-name.rugo", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "mylib.rugo"), []byte("def foo()\nend\n"), 0644)
		os.WriteFile(filepath.Join(dir, "README.md"), []byte("# mylib"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		got, err := FindEntryPoint(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != "mylib.rugo" {
			t.Errorf("got %q, want mylib.rugo", got)
		}
	})

	t.Run("falls back to main.rugo", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "main.rugo"), []byte("def foo()\nend\n"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		got, err := FindEntryPoint(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != "main.rugo" {
			t.Errorf("got %q, want main.rugo", got)
		}
	})

	t.Run("falls back to single Rugo file", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "helpers.rugo"), []byte("def foo()\nend\n"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		got, err := FindEntryPoint(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != "helpers.rugo" {
			t.Errorf("got %q, want helpers.rugo", got)
		}
	})

	t.Run("errors on multiple Rugo files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "a.rugo"), []byte("def a()\nend\n"), 0644)
		os.WriteFile(filepath.Join(dir, "b.rugo"), []byte("def b()\nend\n"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		_, err := FindEntryPoint(dir, r)
		if err == nil {
			t.Error("expected error for ambiguous entry point")
		}
	})

	t.Run("errors on no Rugo files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "README.md"), []byte("# mylib"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "mylib"}
		_, err := FindEntryPoint(dir, r)
		if err == nil {
			t.Error("expected error for no Rugo files")
		}
	})

	t.Run("subpath resolution", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "utils")
		os.MkdirAll(sub, 0755)
		os.WriteFile(filepath.Join(sub, "utils.rugo"), []byte("def foo()\nend\n"), 0644)

		r := &remotePath{Host: "github.com", Owner: "user", Repo: "repo", Subpath: "utils"}
		got, err := FindEntryPoint(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != "utils.rugo" {
			t.Errorf("got %q, want utils.rugo", got)
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

func TestRemotePathModuleKey(t *testing.T) {
	r := &remotePath{Host: "github.com", Owner: "user", Repo: "repo", Version: "v1.0.0", Subpath: "sub"}
	if got := r.moduleKey(); got != "github.com/user/repo" {
		t.Errorf("moduleKey() = %q, want github.com/user/repo", got)
	}
}

func TestAtomicInstall(t *testing.T) {
	t.Run("moves src to dest", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		dest := filepath.Join(dir, "dest")
		os.MkdirAll(src, 0755)
		os.WriteFile(filepath.Join(src, "file.txt"), []byte("hello"), 0644)

		if err := atomicInstall(src, dest); err != nil {
			t.Fatalf("atomicInstall: %v", err)
		}
		// dest should exist with content
		data, err := os.ReadFile(filepath.Join(dest, "file.txt"))
		if err != nil {
			t.Fatalf("reading dest: %v", err)
		}
		if string(data) != "hello" {
			t.Errorf("got %q, want %q", data, "hello")
		}
		// src should no longer exist
		if _, err := os.Stat(src); !os.IsNotExist(err) {
			t.Error("src should not exist after rename")
		}
	})

	t.Run("race lost cleans up src", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		dest := filepath.Join(dir, "dest")
		os.MkdirAll(src, 0755)
		os.WriteFile(filepath.Join(src, "file.txt"), []byte("loser"), 0644)
		// Simulate another process winning the race
		os.MkdirAll(dest, 0755)
		os.WriteFile(filepath.Join(dest, "file.txt"), []byte("winner"), 0644)

		if err := atomicInstall(src, dest); err != nil {
			t.Fatalf("atomicInstall should succeed when dest exists: %v", err)
		}
		// dest should retain the winner's content
		data, err := os.ReadFile(filepath.Join(dest, "file.txt"))
		if err != nil {
			t.Fatalf("reading dest: %v", err)
		}
		if string(data) != "winner" {
			t.Errorf("got %q, want %q (winner should be preserved)", data, "winner")
		}
		// src should be cleaned up
		if _, err := os.Stat(src); !os.IsNotExist(err) {
			t.Error("src should be cleaned up after race loss")
		}
	})
}

func TestIsLatest(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"latest", true},
		{"v1.0.0", false},
		{"main", false},
		{"", false},
		{"LATEST", false},
	}
	for _, tt := range tests {
		if got := isLatest(tt.version); got != tt.want {
			t.Errorf("isLatest(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input      string
		major      int
		minor      int
		patch      int
		prerelease string
		valid      bool
	}{
		{"v1.2.3", 1, 2, 3, "", true},
		{"v0.1.0", 0, 1, 0, "", true},
		{"v1.0.0-beta.1", 1, 0, 0, "beta.1", true},
		{"v2.3.4-rc1", 2, 3, 4, "rc1", true},
		{"1.2.3", 1, 2, 3, "", true},
		{"main", 0, 0, 0, "", false},
		{"v1.2", 0, 0, 0, "", false},
		{"", 0, 0, 0, "", false},
		{"v1.2.3.4", 0, 0, 0, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			sv := parseSemver(tt.input)
			if tt.valid {
				if sv == nil {
					t.Fatalf("parseSemver(%q) = nil, want valid", tt.input)
				}
				if sv.Major != tt.major || sv.Minor != tt.minor || sv.Patch != tt.patch || sv.Prerelease != tt.prerelease {
					t.Errorf("parseSemver(%q) = %d.%d.%d-%s, want %d.%d.%d-%s",
						tt.input, sv.Major, sv.Minor, sv.Patch, sv.Prerelease,
						tt.major, tt.minor, tt.patch, tt.prerelease)
				}
			} else if sv != nil {
				t.Errorf("parseSemver(%q) = non-nil, want nil", tt.input)
			}
		})
	}
}

func TestSemverLess(t *testing.T) {
	tests := []struct {
		a, b string
		less bool
	}{
		{"v1.0.0", "v2.0.0", true},
		{"v2.0.0", "v1.0.0", false},
		{"v1.1.0", "v1.2.0", true},
		{"v1.2.0", "v1.2.1", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0-alpha", "v1.0.0", true},     // pre-release < stable
		{"v1.0.0", "v1.0.0-alpha", false},     // stable > pre-release
		{"v1.0.0-alpha", "v1.0.0-beta", true}, // alphabetical pre-release
		{"v2.0.0-rc1", "v1.0.0", true},        // pre-release < stable even with higher major
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			a := parseSemver(tt.a)
			b := parseSemver(tt.b)
			if a == nil || b == nil {
				t.Fatalf("failed to parse: a=%v b=%v", a, b)
			}
			if got := semverLess(a, b); got != tt.less {
				t.Errorf("semverLess(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.less)
			}
		})
	}
}

func TestParseTagsFromLsRemote(t *testing.T) {
	t.Run("picks highest stable tag", func(t *testing.T) {
		output := "abc1234\trefs/tags/v0.1.0\ndef5678\trefs/tags/v1.0.0\nghi9012\trefs/tags/v1.0.0^{}\njkl3456\trefs/tags/v0.9.0\nmno7890\trefs/tags/v1.2.0\n"
		got := parseTagsFromLsRemote(output)
		if got != "v1.2.0" {
			t.Errorf("got %q, want v1.2.0", got)
		}
	})

	t.Run("stable beats pre-release", func(t *testing.T) {
		output := "abc1234\trefs/tags/v1.0.0\ndef5678\trefs/tags/v2.0.0-beta.1\n"
		got := parseTagsFromLsRemote(output)
		if got != "v1.0.0" {
			t.Errorf("got %q, want v1.0.0", got)
		}
	})

	t.Run("pre-release only", func(t *testing.T) {
		output := "abc1234\trefs/tags/v1.0.0-alpha\ndef5678\trefs/tags/v1.0.0-beta\n"
		got := parseTagsFromLsRemote(output)
		if got != "v1.0.0-beta" {
			t.Errorf("got %q, want v1.0.0-beta", got)
		}
	})

	t.Run("no semver tags", func(t *testing.T) {
		output := "abc1234\trefs/tags/release-2024\ndef5678\trefs/tags/old\n"
		got := parseTagsFromLsRemote(output)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("empty output", func(t *testing.T) {
		got := parseTagsFromLsRemote("")
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("skips dereferenced tags", func(t *testing.T) {
		output := "abc1234\trefs/tags/v1.0.0\ndef5678\trefs/tags/v1.0.0^{}\nghi9012\trefs/tags/v2.0.0\njkl3456\trefs/tags/v2.0.0^{}\n"
		got := parseTagsFromLsRemote(output)
		if got != "v2.0.0" {
			t.Errorf("got %q, want v2.0.0", got)
		}
	})
}
