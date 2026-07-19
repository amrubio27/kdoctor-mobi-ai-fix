package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestParseSupportsP: pure parser tests with whitespace-bounded flag
// detection. False-positive guards: "-pdf" must NOT match "-p",
// "--printed" must NOT match "--print".
func TestParseSupportsP(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"contains -p as flag", "Usage: claude [options]\n  -p, --print     prompt\n  --help          show help\n", true},
		{"contains --print only", "Flags:\n  --print            output to stdout\n", true},
		{"contains --non-interactive", "  --non-interactive  run without prompts\n", true},
		{"contains both -p and --print", "  -p  prompt\n  --print  print\n", true},
		{"no relevant flags", "  --file PATH    read prompt from file\n  --help         show help\n", false},
		{"false positive guard: -pdf", "  --pdf-output  render pdf\n  --file PATH    read prompt\n", false},
		{"false positive guard: --printed", "  --printed     legacy flag\n", false},
		{"comma-bounded matching", "Usage:,-p,--print", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSupportsP(tt.in)
			if got != tt.want {
				t.Errorf("parseSupportsP(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestMatchesFlag: micro-test for the boundary detection logic.
func TestMatchesFlag(t *testing.T) {
	tests := []struct {
		name  string
		s, fg string
		want  bool
	}{
		{"isolated at start", "-p", "-p", true},
		{"isolated at end", "x -p", "-p", true},
		{"in middle of whitespace", "x -p y", "-p", true},
		{"no whitespace before", "x-p", "-p", false},
		{"no whitespace after", "-py", "-p", false},
		{"newline boundary", "a\n-p\nb", "-p", true},
		{"comma boundary", "options,-p,--help", "-p", true},
		{"longer with prefix letter", "file-p", "-p", false},
		{"missing flag", "hello world", "-p", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFlag(tt.s, tt.fg)
			if got != tt.want {
				t.Errorf("matchesFlag(%q, %q) = %v, want %v", tt.s, tt.fg, got, tt.want)
			}
		})
	}
}

// TestParseVersion: extracts semver from common claude formats.
func TestParseVersion(t *testing.T) {
	tests := []struct {
		name   string
		vin    string
		hin    string
		wantOk string // expected parsed version
	}{
		{"Claude v1.2.3", "Claude v1.2.3\n", "", "1.2.3"},
		{"Claude 1.2.3", "Claude 1.2.3\n", "", "1.2.3"},
		{"v2.0.0 alone", "v2.0.0\n", "", "2.0.0"},
		{"patch zero explicit", "Claude v1.0.0\n", "", "1.0.0"},
		{"only in helpOutput", "", "claude v3.1.4 (latest)\n", "3.1.4"},
		{"empty both", "", "", "unknown"},
		{"no version string", "Claude Code\n", "", "unknown"},
		{"pre-release excluded", "Claude v1.0.0-beta\n", "", "unknown"}, // looksLikeVersion requires numeric only
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVersion(tt.vin, tt.hin)
			if got != tt.wantOk {
				t.Errorf("parseVersion(%q, %q) = %q, want %q", tt.vin, tt.hin, got, tt.wantOk)
			}
		})
	}
}

// TestLooksLikeVersion: micro-test for version-shape detection.
func TestLooksLikeVersion(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"1.2.3", true},
		{"0.1", true},
		{"10.20.30", true},
		{"1.2.3-beta", false}, // pre-release suffix has non-digit
		{"v1.2.3", false},     // 'v' prefix not stripped here
		{"1", false},          // single number
		{".1.2", false},
		{"1..2", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := looksLikeVersion(tt.in)
			if got != tt.want {
				t.Errorf("looksLikeVersion(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestLoadSaveCacheRoundtrip: write then read returns identical values.
func TestLoadSaveCacheRoundtrip(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "claude-version.json")
	original := &ClaudeCache{
		DetectedAt: time.Now().UTC().Truncate(time.Second),
		Version:    "1.2.3",
		SupportsP:  true,
	}
	if err := saveCache(tmp, original); err != nil {
		t.Fatalf("saveCache: %v", err)
	}

	got, err := loadCache(tmp, 24*time.Hour)
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	if got == nil {
		t.Fatal("expected cache, got nil")
	}
	if got.Version != original.Version || got.SupportsP != original.SupportsP {
		t.Errorf("roundtrip mismatch: got %+v want %+v", got, original)
	}
	if !got.DetectedAt.Equal(original.DetectedAt) {
		t.Errorf("DetectedAt drift: got %v want %v", got.DetectedAt, original.DetectedAt)
	}
}

// TestLoadCacheStaleReturnsNil: a cache older than ttl is treated as a miss.
func TestLoadCacheStaleReturnsNil(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "claude-version.json")
	stale := &ClaudeCache{
		DetectedAt: time.Now().UTC().Add(-25 * time.Hour),
		Version:    "1.0.0",
		SupportsP:  true,
	}
	if err := saveCache(tmp, stale); err != nil {
		t.Fatal(err)
	}
	got, err := loadCache(tmp, 24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("stale cache should return nil; got %+v", got)
	}
}

// TestLoadCacheMissingFileReturnsNilNoError: missing file is a cache miss.
func TestLoadCacheMissingFileReturnsNilNoError(t *testing.T) {
	got, err := loadCache(filepath.Join(t.TempDir(), "nope.json"), 24*time.Hour)
	if err != nil {
		t.Fatalf("missing file should NOT be an error; got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing file; got %+v", got)
	}
}

// TestSaveCacheAtomic: .tmp file should not survive if Rename succeeds.
// (If Rename was skipped, the tmp file would remain.)
func TestSaveCacheAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-version.json")
	if err := saveCache(path, &ClaudeCache{
		DetectedAt: time.Now().UTC(),
		Version:    "x",
		SupportsP:  true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf(".tmp file should not survive after successful rename; stat err=%v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("cache file should exist; stat err=%v", err)
	}
}

// TestEnsureCapabilities_HitsFreshCache: detector NOT called when cache valid.
func TestEnsureCapabilities_HitsFreshCache(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "cache.json")
	detectorCalled := 0
	fakeDetector := func() (string, string, error) {
		detectorCalled++
		return "", "", fmt.Errorf("should not be called")
	}
	if err := saveCache(tmp, &ClaudeCache{
		DetectedAt: time.Now().UTC(),
		Version:    "1.0.0",
		SupportsP:  true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ensureCapabilities(tmp, 24*time.Hour, fakeDetector); err != nil {
		t.Errorf("expected nil from valid cache; got %v", err)
	}
	if detectorCalled != 0 {
		t.Errorf("detector should NOT be called when cache is fresh; got %d calls", detectorCalled)
	}
}

// TestEnsureCapabilities_DetectsOnMiss: detector called, cache populated, result positive.
func TestEnsureCapabilities_DetectsOnMiss(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "cache.json")
	fakeDetector := func() (string, string, error) {
		return "Claude v1.2.3\n", "  -p  prompt\n  --help  help\n", nil
	}
	if err := ensureCapabilities(tmp, 24*time.Hour, fakeDetector); err != nil {
		t.Errorf("expected nil from positive detection; got %v", err)
	}
	// Verify persistence.
	c, err := loadCache(tmp, 24*time.Hour)
	if err != nil || c == nil {
		t.Fatalf("cache should be populated; got %+v err=%v", c, err)
	}
	if c.Version != "1.2.3" || !c.SupportsP {
		t.Errorf("cached fields wrong: %+v", c)
	}
}

// TestEnsureCapabilities_FailsOnNegative: detector reports no -p flag.
// Negative result is persisted AND returned as actionable error.
func TestEnsureCapabilities_FailsOnNegative(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "cache.json")
	fakeDetector := func() (string, string, error) {
		return "Claude v0.5.0\n", "  --help  help\n  --file PATH  file\n", nil
	}
	err := ensureCapabilities(tmp, 24*time.Hour, fakeDetector)
	if err == nil {
		t.Fatal("expected error when -p flag missing")
	}
	msg := err.Error()
	for _, want := range []string{
		"0.5.0",           // detected version surfaced
		"non-interactive", // the missing capability
		"v1.0.0+",         // the requirement
		tmp,               // cache path so user knows where to delete
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing fragment %q; full error: %s", want, msg)
		}
	}
	// Negative cache SHOULD also be persisted (Round-2 design decision).
	c, _ := loadCache(tmp, 24*time.Hour)
	if c == nil {
		t.Fatal("negative cache should also be persisted")
	}
	if c.SupportsP {
		t.Error("negative cache.SupportsP should be false")
	}
}

// TestEnsureCapabilities_NegativeCacheRespectedUntilTTL: a second call
// immediately after the first (within TTL) should NOT re-detect.
func TestEnsureCapabilities_NegativeCacheRespectedUntilTTL(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "cache.json")
	detectCount := 0
	firstDetector := func() (string, string, error) {
		detectCount++
		return "Claude v0.1\n", "  --help  help\n", nil
	}
	secondDetector := func() (string, string, error) {
		detectCount++
		return "Claude v2.0\n", "  -p  prompt\n", nil
	}

	if err := ensureCapabilities(tmp, 24*time.Hour, firstDetector); err == nil {
		t.Fatal("first call should fail (no -p flag)")
	}
	// Second call would succeed if detector was invoked, but cache
	// blocks re-detection within TTL.
	err := ensureCapabilities(tmp, 24*time.Hour, secondDetector)
	if err == nil {
		t.Error("expected second call to STILL fail because cache says supports=false")
	}
	if detectCount != 1 {
		t.Errorf("detector should have run exactly once (cached negative); got %d", detectCount)
	}
}

// TestEnsureCapabilities_DetectorErrorSurfaced: detector subprocess
// failure surfaces a wrapped error with cache-reset hint.
func TestEnsureCapabilities_DetectorErrorSurfaced(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "cache.json")
	fakeDetector := func() (string, string, error) {
		return "", "", fmt.Errorf("claude binary not found")
	}
	err := ensureCapabilities(tmp, 24*time.Hour, fakeDetector)
	if err == nil {
		t.Fatal("expected error when detector fails")
	}
	if !strings.Contains(err.Error(), "claude binary not found") {
		t.Errorf("error should preserve detector message; got %q", err)
	}
	if !strings.Contains(err.Error(), tmp) {
		t.Errorf("error should mention cache path so user can reset; got %q", err)
	}
}

// TestEnsureCapabilities_StaleCacheTriggersRedetect: stale cache is
// treated as miss and detector runs again.
func TestEnsureCapabilities_StaleCacheTriggersRedetect(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "cache.json")
	detectCount := 0
	fakeDetector := func() (string, string, error) {
		detectCount++
		return "Claude v1.0.0\n", "  -p  prompt\n", nil
	}
	// Pre-populate a stale-but-positive cache.
	if err := saveCache(tmp, &ClaudeCache{
		DetectedAt: time.Now().UTC().Add(-48 * time.Hour),
		Version:    "0.0.0",
		SupportsP:  true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ensureCapabilities(tmp, 24*time.Hour, fakeDetector); err != nil {
		t.Errorf("expected nil; got %v", err)
	}
	if detectCount != 1 {
		t.Errorf("detector should run once for stale cache; got %d", detectCount)
	}
	// And the cache should now reflect the fresh detection result.
	c, _ := loadCache(tmp, 24*time.Hour)
	if c == nil || c.Version != "1.0.0" {
		t.Errorf("cache after re-detection wrong: %+v", c)
	}
}

// TestFormatSupportsPMissing: the actionable error includes all expected fragments.
func TestFormatSupportsPMissing(t *testing.T) {
	msg := formatSupportsPMissing("0.5.0", "/tmp/cache.json")
	for _, want := range []string{
		"0.5.0", "non-interactive", "v1.0.0+", "/tmp/cache.json",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q", want)
		}
	}
}

// TestDefaultCachePath: returns absolute non-empty path ending in
// claude-version.json under .kdoctor/cache in the user's home dir.
func TestDefaultCachePath(t *testing.T) {
	p, err := defaultCachePath()
	if err != nil {
		t.Fatal(err)
	}
	if p == "" || !filepath.IsAbs(p) {
		t.Errorf("expected absolute non-empty path; got %q", p)
	}
	if !strings.HasSuffix(p, filepath.Join(".kdoctor", "cache", "claude-version.json")) {
		t.Errorf("expected suffix %q; got %q",
			filepath.Join(".kdoctor", "cache", "claude-version.json"), p)
	}
}
