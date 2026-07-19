package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ClaudeCacheTTL is the lifetime of the cached claude CLI capability
// detection. After this period, ensureCapabilities re-detects by
// invoking `claude --version` and `claude --help` again. The cost
// (~200ms of CLI startup per scan) is small but non-trivial at scale.
const ClaudeCacheTTL = 24 * time.Hour

// DetectionFunc is the injectable capability detector. The production
// provider wires this to defaultDetection which shells out to `claude`;
// unit tests pass a fake returning canned strings so the parser and
// cache logic can be exercised without a real `claude` binary.
type DetectionFunc func() (versionOutput, helpOutput string, err error)

// defaultDetection invokes `claude --version` and `claude --help`,
// forwarding their combined stdout. Individual failures are tolerated
// as long as at least one succeeds (the parser extracts version from
// whichever source is available), but a total failure surfaces as err.
var defaultDetection DetectionFunc = func() (string, string, error) {
	var vOut, hOut bytes.Buffer
	cmd := exec.Command("claude", "--version")
	cmd.Stdout = &vOut
	vErr := cmd.Run()
	cmd = exec.Command("claude", "--help")
	cmd.Stdout = &hOut
	hErr := cmd.Run()
	if vErr != nil && hErr != nil {
		return "", "", fmt.Errorf("`claude --version` and `claude --help` both failed: %v / %v", vErr, hErr)
	}
	return vOut.String(), hOut.String(), nil
}

// ClaudeCache is the persisted payload at defaultCachePath(). Cache
// misses are NOT a fatal error: the next scan regenerates.
type ClaudeCache struct {
	DetectedAt time.Time `json:"detected_at"`
	Version    string    `json:"version"`
	SupportsP  bool      `json:"supports_p"`
}

// ClaudeProvider implements Provider using the `claude` CLI tool.
// Round-2 task #7 added an explicit pre-check inside Fix that fails fast
// if the installed `claude` does not support non-interactive mode (the
// -p / --print / --non-interactive flag). This prevents the previous
// silent-degradation bug where an incompatible version produced an
// unrelated error and the caller fell back to a placeholder patch.
type ClaudeProvider struct{}

// Fix writes the prompt to a temp file and shells out to `claude`.
// Before invoking claude, ensureCapabilities verifies the CLI is
// compatible; an incompatible version produces an actionable error
// message including the detected version and a hint to upgrade.
func (p *ClaudeProvider) Fix(prompt string) (string, error) {
	cachePath, err := defaultCachePath()
	if err != nil {
		return "", err
	}
	if err := ensureCapabilities(cachePath, ClaudeCacheTTL, defaultDetection); err != nil {
		return "", err
	}

	tmpDir := os.TempDir()
	promptFile := filepath.Join(tmpDir, "kdoctor-prompt.md")
	if err := os.WriteFile(promptFile, []byte(prompt), 0644); err != nil {
		return "", fmt.Errorf("failed to write prompt to temp file: %w", err)
	}
	defer os.Remove(promptFile)

	cmd := exec.Command("claude", "--file", promptFile)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude execution failed: %w (stderr: %s)", err, errBuf.String())
	}
	return outBuf.String(), nil
}

// ensureCapabilities orchestrates cache lookup → fallback to detect →
// persist result. detect is injectable for unit testability. Result is
// an actionable error (with version and cache path) on negative, nil on
// positive, or a wrapped exec error if detection itself fails.
func ensureCapabilities(cachePath string, ttl time.Duration, detect DetectionFunc) error {
	if c, err := loadCache(cachePath, ttl); err == nil && c != nil {
		if !c.SupportsP {
			return fmt.Errorf(formatSupportsPMissing(c.Version, cachePath))
		}
		return nil
	}

	versionOut, helpOut, err := detect()
	if err != nil {
		return fmt.Errorf("claude CLI detection failed: %w. Cache at %s — delete to reset.", err, cachePath)
	}

	version := parseVersion(versionOut, helpOut)
	supports := parseSupportsP(helpOut)

	cache := &ClaudeCache{
		DetectedAt: time.Now().UTC(),
		Version:    version,
		SupportsP:  supports,
	}
	// Best-effort persistence; never block on a cache write failure.
	_ = saveCache(cachePath, cache)

	if !supports {
		return fmt.Errorf(formatSupportsPMissing(version, cachePath))
	}
	return nil
}

// formatSupportsPMissing returns the actionable error message shown when
// `claude <flag>` is missing. Includes detected version (or "unknown")
// and a cache-reset hint so the user knows how to force re-detection.
func formatSupportsPMissing(version, cachePath string) string {
	return fmt.Sprintf(
		"claude CLI (version %q) does not support non-interactive mode: "+
			"the -p, --print, or --non-interactive flag is missing from its --help output.\n"+
			"kdoctor requires claude v1.0.0+ with the -p flag for `kdoctor fix --ai`.\n"+
			"Upgrade: `npm update -g @anthropic-ai/claude-code` or see https://docs.anthropic.com/claude-code.\n"+
			"Cache at %s — delete this file to force re-detection on next run.",
		version, cachePath,
	)
}

// parseSupportsP scans claude --help output for the non-interactive
// flag. Returns true if ANY of the documented forms appear in a
// flag-like context (whitespace-bounded to avoid substring false
// positives such as "--pdf" matching "-p").
func parseSupportsP(helpOutput string) bool {
	if helpOutput == "" {
		return false
	}
	return matchesFlag(helpOutput, "-p") ||
		matchesFlag(helpOutput, "--print") ||
		matchesFlag(helpOutput, "--non-interactive")
}

// matchesFlag returns true if flag appears in s surrounded by whitespace
// or string boundary, so flag-like substrings ("-pdf") don't match a
// shorter flag ("-p") that happens to be a prefix.
func matchesFlag(s, flag string) bool {
	idx := strings.Index(s, flag)
	for idx >= 0 {
		beforeOK := idx == 0 || isFlagBoundary(s[idx-1])
		after := idx + len(flag)
		afterOK := after >= len(s) || isFlagBoundary(s[after])
		if beforeOK && afterOK {
			return true
		}
		idx = strings.Index(s[after:], flag)
		if idx >= 0 {
			idx += after
		}
	}
	return false
}

func isFlagBoundary(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == ',' || b == '|'
}

// parseVersion extracts a semver-like version from the version output,
// falling back to scanning helpOutput. Returns "unknown" on parse failure
// — callers should still surface this so the user knows detection ran
// but couldn't find a version string.
func parseVersion(versionOutput, helpOutput string) string {
	for _, src := range []string{versionOutput, helpOutput} {
		for _, raw := range strings.Split(src, "\n") {
			t := strings.TrimSpace(raw)
			// Strip a leading "Claude" / "claude" if present so we don't
			// accidentally pick up the product name as version.
			t = strings.TrimPrefix(t, "Claude")
			t = strings.TrimPrefix(t, "claude")
			t = strings.TrimSpace(t)
			// Now expect "vX.Y.Z" or "X.Y.Z" possibly followed by other text.
			t = strings.TrimPrefix(t, "v")
			fields := strings.Fields(t)
			if len(fields) == 0 {
				continue
			}
			if looksLikeVersion(fields[0]) {
				return fields[0]
			}
		}
	}
	return "unknown"
}

// looksLikeVersion reports whether s is "X.Y" or "X.Y.Z" with all
// numeric segments. Pre-release suffixes like "-beta" are excluded by
// design — they make version comparisons ambiguous for our purposes.
func looksLikeVersion(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 {
			return false
		}
		for i := 0; i < len(p); i++ {
			if p[i] < '0' || p[i] > '9' {
				return false
			}
		}
	}
	return true
}

// loadCache returns the cached entry if it exists AND is fresh
// (DetectedAt within ttl), otherwise nil. A non-existent file is not
// treated as an error (returns nil, nil).
func loadCache(path string, ttl time.Duration) (*ClaudeCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var c ClaudeCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if time.Since(c.DetectedAt) > ttl {
		return nil, nil // stale, treat as cache miss
	}
	return &c, nil
}

// saveCache writes c to path atomically: serialize to <path>.tmp first,
// then os.Rename onto <path>. This avoids torn writes if a crash occurs
// mid-write that would otherwise leave a partial JSON file.
func saveCache(path string, c *ClaudeCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// defaultCachePath returns the cross-platform path for the persisted
// cache. Order: $HOME via os.UserHomeDir (works on Windows, Linux,
// macOS). Falls back to a relative ".kdoctor" if HOME is unavailable.
func defaultCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve user home directory: %w", err)
	}
	return filepath.Join(home, ".kdoctor", "cache", "claude-version.json"), nil
}
