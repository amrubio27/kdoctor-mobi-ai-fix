package detektrunner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeJar stands in for the 50 MB artifact; provisioning logic does not care
// what the bytes are, only that they hash to what was promised.
var fakeJar = []byte("PK\x03\x04 not really a jar, but it hashes just fine")

func fakeJarSHA() string {
	sum := sha256.Sum256(fakeJar)
	return hex.EncodeToString(sum[:])
}

// mavenStub serves the artifact and counts how many times it was asked for, so
// tests can assert that the cache actually prevents a second request.
func mavenStub(t *testing.T, body []byte) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if !strings.HasSuffix(r.URL.Path, "-all.jar") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// isolateHome redirects the cache to a temp dir so tests never touch the real
// ~/.kdoctor and never depend on whether a previous run populated it.
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	t.Setenv("KDOCTOR_DETEKT_JAR", "")
	t.Setenv("KDOCTOR_NO_DOWNLOAD", "")
	t.Setenv("KDOCTOR_DETEKT_VERSION", "")
	t.Setenv("KDOCTOR_DETEKT_SHA256", "")
	return home
}

func TestEnsureDetektJarDownloadsAndVerifies(t *testing.T) {
	home := isolateHome(t)
	srv, hits := mavenStub(t, fakeJar)

	path, source, err := EnsureDetektJar(context.Background(), ProvisionOptions{
		BaseURL:        srv.URL,
		ExpectedSHA256: fakeJarSHA(),
	})
	if err != nil {
		t.Fatalf("provision failed: %v", err)
	}
	if source != SourceDownloaded {
		t.Fatalf("want source %q, got %q", SourceDownloaded, source)
	}
	if !strings.HasPrefix(path, filepath.Join(home, ".kdoctor")) {
		t.Fatalf("jar landed outside the isolated cache: %s", path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(fakeJar) {
		t.Fatal("cached jar content differs from what was served")
	}
	if *hits != 1 {
		t.Fatalf("want exactly 1 request, got %d", *hits)
	}
}

func TestEnsureDetektJarSecondCallHitsCache(t *testing.T) {
	isolateHome(t)
	srv, hits := mavenStub(t, fakeJar)
	opts := ProvisionOptions{BaseURL: srv.URL, ExpectedSHA256: fakeJarSHA()}

	if _, _, err := EnsureDetektJar(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	_, source, err := EnsureDetektJar(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if source != SourceCache {
		t.Fatalf("want source %q on second call, got %q", SourceCache, source)
	}
	if *hits != 1 {
		t.Fatalf("cache did not prevent a second download: %d requests", *hits)
	}
}

// A bad checksum must be fail-closed: nothing is installed, so the next run
// retries instead of silently reusing a corrupt or tampered artifact.
func TestEnsureDetektJarRejectsBadChecksum(t *testing.T) {
	isolateHome(t)
	srv, _ := mavenStub(t, fakeJar)

	_, _, err := EnsureDetektJar(context.Background(), ProvisionOptions{
		BaseURL:        srv.URL,
		ExpectedSHA256: strings.Repeat("ab", 32),
	})
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("want ErrChecksumMismatch, got %v", err)
	}

	target, err := DetektJarPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(target); statErr == nil {
		t.Fatal("a jar that failed verification was left in the cache")
	}
	// No stray temp files either.
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ".detekt-download-*"))
	if len(leftovers) != 0 {
		t.Fatalf("temp files left behind: %v", leftovers)
	}
}

func TestEnsureDetektJarRespectsNoDownload(t *testing.T) {
	isolateHome(t)
	srv, hits := mavenStub(t, fakeJar)

	_, _, err := EnsureDetektJar(context.Background(), ProvisionOptions{
		BaseURL:    srv.URL,
		NoDownload: true,
	})
	if !errors.Is(err, ErrProvisionDisabled) {
		t.Fatalf("want ErrProvisionDisabled, got %v", err)
	}
	if *hits != 0 {
		t.Fatalf("network was touched despite NoDownload: %d requests", *hits)
	}
}

func TestEnsureDetektJarPrefersExplicitEnvJar(t *testing.T) {
	isolateHome(t)
	srv, hits := mavenStub(t, fakeJar)

	explicit := filepath.Join(t.TempDir(), "my-detekt.jar")
	if err := os.WriteFile(explicit, fakeJar, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KDOCTOR_DETEKT_JAR", explicit)

	path, source, err := EnsureDetektJar(context.Background(), ProvisionOptions{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if source != SourceEnv || path != explicit {
		t.Fatalf("want explicit jar %q via %q, got %q via %q", explicit, SourceEnv, path, source)
	}
	if *hits != 0 {
		t.Fatalf("downloaded despite an explicit jar: %d requests", *hits)
	}
}
