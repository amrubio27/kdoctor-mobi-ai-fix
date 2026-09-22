package detektrunner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Provisioning exists so that a third party can install kdoctor and get a full
// scan without first installing detekt, wiring Gradle, or reading any docs.
// Without it, 33 of the 53 live rules silently never fire.
//
// Everything here is stdlib on purpose: CI enforces a clean `go mod tidy`, so a
// new dependency would break the build gate.
const (
	// DetektVersion is pinned, not resolved at runtime: an auto-updating
	// analyser would silently change a project's Health Score between runs.
	DetektVersion = "1.23.8"

	// DetektSHA256 is the checksum published by Maven Central alongside the
	// artifact (…/detekt-cli-1.23.8-all.jar.sha256). Verification is
	// fail-closed; a mismatch never lands in the cache.
	DetektSHA256 = "3afe89a11120303c73c9bdda3d8fe558dd9070a6937d27819ddc04b275381245"

	// mavenCentralBase is the canonical origin. Maven Central is what Gradle
	// already talks to, so it tends to be reachable where other hosts are not.
	mavenCentralBase = "https://repo1.maven.org/maven2/io/gitlab/arturbosch/detekt/detekt-cli"

	downloadTimeout = 3 * time.Minute

	// ComposeRulesVersion is the last release of the detekt-compose plugin
	// (io.nlopez.compose.rules) built against detekt 1.23.x. Later versions
	// (0.5.x and up) compile against detekt 2.0.0-alpha and will not load
	// into the pinned CLI.
	ComposeRulesVersion = "0.4.22"

	composeRulesBase = "https://repo1.maven.org/maven2/io/nlopez/compose/rules"
)

var (
	// ErrProvisionDisabled means a download was required but the caller (or the
	// environment) forbade it.
	ErrProvisionDisabled = errors.New("detekt download disabled")

	// ErrChecksumMismatch means the downloaded artifact did not match the
	// pinned checksum. The partial file is discarded.
	ErrChecksumMismatch = errors.New("detekt jar checksum mismatch")
)

// ProvisionSource describes where a jar came from, for --verbose and doctor.
type ProvisionSource string

const (
	SourceEnv        ProvisionSource = "KDOCTOR_DETEKT_JAR"
	SourceCache      ProvisionSource = "cache"
	SourceDownloaded ProvisionSource = "downloaded"
)

// ProvisionOptions configures EnsureDetektJar.
type ProvisionOptions struct {
	// NoDownload forbids network access. A cache hit still succeeds.
	NoDownload bool
	// BaseURL overrides the artifact origin (tests, corporate mirrors).
	BaseURL string
	// ExpectedSHA256 overrides the pinned checksum, for mirrors that repackage.
	ExpectedSHA256 string
	// Progress receives human-readable status lines. Optional.
	Progress io.Writer
}

// EnsureDetektJar returns a local path to a verified detekt CLI jar,
// downloading it once into ~/.kdoctor/tools/ if needed.
func EnsureDetektJar(ctx context.Context, opts ProvisionOptions) (string, ProvisionSource, error) {
	if p := strings.TrimSpace(os.Getenv("KDOCTOR_DETEKT_JAR")); p != "" {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, SourceEnv, nil
		}
		return "", "", fmt.Errorf("KDOCTOR_DETEKT_JAR points at %q which is not a readable file", p)
	}

	target, err := DetektJarPath()
	if err != nil {
		return "", "", err
	}
	if fi, err := os.Stat(target); err == nil && fi.Size() > 0 {
		return target, SourceCache, nil
	}

	if opts.NoDownload || envTrue("KDOCTOR_NO_DOWNLOAD") {
		return "", "", fmt.Errorf("%w: no cached jar at %s", ErrProvisionDisabled, target)
	}

	if err := download(ctx, target, opts); err != nil {
		return "", "", err
	}
	return target, SourceDownloaded, nil
}

// DetektJarPath is the cache location for the pinned version. Versioned so a
// future bump does not silently reuse the old jar.
func DetektJarPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	version := detektVersion()
	return filepath.Join(home, ".kdoctor", "tools", "detekt", version,
		fmt.Sprintf("detekt-cli-%s-all.jar", version)), nil
}

func detektVersion() string {
	if v := strings.TrimSpace(os.Getenv("KDOCTOR_DETEKT_VERSION")); v != "" {
		return v
	}
	return DetektVersion
}

func expectedSHA(opts ProvisionOptions) string {
	if opts.ExpectedSHA256 != "" {
		return strings.ToLower(opts.ExpectedSHA256)
	}
	if v := strings.TrimSpace(os.Getenv("KDOCTOR_DETEKT_SHA256")); v != "" {
		return strings.ToLower(v)
	}
	// A pinned checksum only applies to the pinned version. If the user asked
	// for a different one we cannot vouch for it, so verification is skipped
	// rather than guaranteed to fail.
	if detektVersion() != DetektVersion {
		return ""
	}
	return DetektSHA256
}

func artifactURL(opts ProvisionOptions) string {
	base := opts.BaseURL
	if base == "" {
		base = strings.TrimSpace(os.Getenv("KDOCTOR_DETEKT_BASE_URL"))
	}
	if base == "" {
		base = mavenCentralBase
	}
	v := detektVersion()
	return fmt.Sprintf("%s/%s/detekt-cli-%s-all.jar", strings.TrimSuffix(base, "/"), v, v)
}

// download fetches the pinned detekt CLI jar.
func download(ctx context.Context, target string, opts ProvisionOptions) error {
	return fetch(ctx, artifactURL(opts), target, expectedSHA(opts), opts.Progress)
}

// fetch downloads url to target, verifying the digest while streaming and
// only renaming into place once it matches.
//
// Writing to a temp file first is what makes the cache trustworthy: a crash,
// a truncated response or a bad checksum can never leave a partial jar that a
// later run mistakes for a valid cache hit.
func fetch(ctx context.Context, url, target, wantSHA256 string, progress io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if progress != nil {
		fmt.Fprintf(progress, "Downloading %s (first run only)\n  to %s\n", url, target)
	}

	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), ".kdoctor-download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName) // no-op once the rename below succeeded
	}()

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if wantSHA256 != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if got != strings.ToLower(wantSHA256) {
			return fmt.Errorf("%w: expected %s, got %s (from %s)", ErrChecksumMismatch, wantSHA256, got, url)
		}
	}

	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("install %s: %w", target, err)
	}
	return nil
}

func envTrue(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// composePlugin describes one jar of the detekt-compose plugin.
//
// These 10 Compose rules are the main reason to reach for kdoctor over plain
// detekt, yet the catalog marked them live while nothing ever loaded the
// plugin that implements them -- they could not fire at all.
//
// Maven Central publishes no .sha256 for this artifact, only .sha1. The
// digests below were computed from downloads whose published SHA-1 matched,
// so they pin exactly the bytes Maven Central serves today.
type composePlugin struct {
	artifact string
	sha256   string
}

var composePlugins = []composePlugin{
	{artifact: "detekt", sha256: "437dd7d8731b13fb274099e8058dae6640f312165ed53171d50927e6c3d2468b"},
	{artifact: "common", sha256: "00a38c838b53209ea6be44d51a07edb02eacdb6cd523676125d021a5087ca11c"},
}

// EnsureComposePlugins returns local paths to the detekt-compose jars,
// downloading them on first use. Failure is never fatal: the caller runs
// detekt without the plugin and simply loses the Compose rules.
func EnsureComposePlugins(ctx context.Context, opts ProvisionOptions) ([]string, error) {
	if raw := strings.TrimSpace(os.Getenv("KDOCTOR_COMPOSE_PLUGINS")); raw != "" {
		return filepath.SplitList(raw), nil
	}
	version := composeVersion()
	paths := make([]string, 0, len(composePlugins))
	for _, pl := range composePlugins {
		target, err := composeJarPath(version, pl.artifact)
		if err != nil {
			return nil, err
		}
		if fi, statErr := os.Stat(target); statErr == nil && fi.Size() > 0 {
			paths = append(paths, target)
			continue
		}
		if opts.NoDownload || envTrue("KDOCTOR_NO_DOWNLOAD") {
			return nil, fmt.Errorf("%w: compose plugin not cached at %s", ErrProvisionDisabled, target)
		}
		url := fmt.Sprintf("%s/%s/%s/%s-%s.jar", composeRulesBase, pl.artifact, version, pl.artifact, version)
		sha := pl.sha256
		if version != ComposeRulesVersion {
			// Pinned digests only describe the pinned version.
			sha = ""
		}
		if err := fetch(ctx, url, target, sha, opts.Progress); err != nil {
			return nil, err
		}
		paths = append(paths, target)
	}
	return paths, nil
}

func composeVersion() string {
	if v := strings.TrimSpace(os.Getenv("KDOCTOR_COMPOSE_RULES_VERSION")); v != "" {
		return v
	}
	return ComposeRulesVersion
}

func composeJarPath(version, artifact string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".kdoctor", "tools", "compose-rules", version,
		fmt.Sprintf("%s-%s.jar", artifact, version)), nil
}
