# Changelog

All notable changes to kdoctor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.0.0] — 2026-07-19

### Initial Stable Release

`kdoctor` is a CLI tool (Go 1.22+) that audits the quality of Android / KMP / CMP
projects. It wraps `detekt` SARIF 2.1.0 output, maps Detekt rules into a curated
`kdoctor` catalog (78 rules: 11 V1 live + 53 V1 planned + 14 default-detekt
mappings Phase 1.5), computes a **Health Score 0-100** (errors -5, warnings -2,
info -0.5), and emits rich console / JSON / SARIF reports.

### Added (Tiers 1–3 + Phases 4–5)

- **Tier 1 #1** — Seeded `examples/bad-project` fixture + smoke test runner
  in `go test`. 11 V1 priority rules live in catalog.
- **Tier 1 #2** — Four Go-based regex detectors for rules without detekt
  equivalence: `sec-log-pii`, `compose-missing-key`,
  `sec-webview-javascript-enabled`, `coroutine-dispatchers-hardcoded`.
- **Tier 1 #3** — `--diff <ref>` filtering + `--baseline <path>` suppression
  for PR-scoped scans (`internal/core/diff`, `internal/core/baseline`).
- **Tier 2 #4** — Gradle plugin `kdoctor-gradle-plugin` exposing
  `kdoctorScan` task (non-invasive, user-opt-in).
- **Tier 2 #5** — `kdoctor fix --ai` LLM-driven fixer with Claude provider,
  patchguard validator, and quality-focused prompt templates. `--mode auto`
  extracts the code block, applies the patch, validates the patched file,
  and rolls back on validation failure.
- **Tier 3 #6** — HTML dashboard (vite + tailwind): cluster distribution charts,
  score trend, integration with MobiAI Graph via `--mobiai` flag.
- **Tier 3 #7** — `kdoctor.config.yaml` live overrides: cluster-aware severity,
  custom excludes, per-rule/cluster disable.

### Hardened (Round-2 Polish)

- **#1** — Rulemap: 7 round-1 tests restored verbatim + Bug 1 multi-vendor
  prefix-strip regression guard (`TestBug1MultiVendorPrefixStrip`).
- **#2** — Purged fake stubs (`WebViewSettings`, `Dispatchers`, `Log`,
  `items(...)`) in `examples/bad-project/BadCode.kt` that caused parse
  errors; regenerated `report.json` with 6 deterministic findings.
- **#2.1** — Added `examples/bad-project/README.md` + cluster-level override
  alive in `kdoctor.config.yaml` (`security: warning`) + ApplyOverrides
  precedence-test pinning in `rulemap`.
- **#3** — `examples/scoring-fixtures/bad.json` score band updated to
  `[75, 90]` (deterministic score 82 post-overrides) with durable rationale
  in `examples/scoring-fixtures/README.md`.
- **#4** — Patchguard rewritten as Kotlin-aware mini-lexer with state machine
  (7 modes: code, line/block comments, single/raw strings, char literals,
  templates); ignores braces/parens inside strings; 14 new edge tests
  preserving round-1 verbatim.
- **#5** — New `internal/core/pathutil` package: `NormalizePath` +
  `SuffixMatch` boundary-aware. `diff.FilterFindingsByDiffWithRoot` and
  `baseline.IsSuppressedWithRoot` for absolute-path matching; git-root
  auto-detection in `scan.go` with explicit stderr fallback warning.
- **#6** — `qualityprompt.BuildPromptWithContext` ±N context-line slicing
  (header + body + `<-- FINDING` marker inline); `--context-lines` CLI flag
  (default = 10) wired in `kdoctor fix`.
- **#7** — Claude provider: runtime verification of `-p` (or `--print` /
  `--non-interactive`) flag, 24-hour persistent cache at
  `~/.kdoctor/cache/claude-version.json`, pure parsers
  (`parseSupportsP`, `parseVersion`, `looksLikeVersion`); 15 new unit
  tests covering cache TTL, atomic write, detector injection, error UX.

### Release Prep & Public Documentation (Round-3)

- **#8** — Commit message audit + atomic history for v1.0.0; 14 commits
  landed from Tier 1 #1 through Round-3 #10.
- **#9** — GitHub Actions CI fast gate (`.github/workflows/ci.yml`):
  `gofmt` drift check, `go vet ./...`, `go test -race`, binary build,
  and `go mod tidy` drift check on push/PR to `main`.
- **#9.1** — CI hardening: pinned `mobiai@^1.0.0` in integration workflow,
  added `paths-ignore` for docs-only changes, replaced swallowers with
  `set -euo pipefail` and `::error::` annotations.
- **#9.2** — Fixed YAML header corruption introduced during #9.1 polish.
- **#10** — Public rewrite of `README.md`: hero with badges, feature grid,
  3-step quickstart, install options, usage sections, config examples,
  CI snippet, comparison table, and honest scoping of `fix --mode auto`.

### Stats

- 19 Go packages with tests.
- All tests PASS in the full suite (`go test -count=1 ./...`).
- Catalog: 78 rules total (`scripts/genschema/main.go`).
- Health Score formula: `100 - errors*5 - warnings*2 - info*0.5`.
- Project types supported: `kmp`, `cmp`, `jvm`, `android`, `compose`,
  `gradle`, `plain` (auto-detected or `--type=`).

### Validated

- `go vet ./...` exit 0
- `go test -count=1 ./...` all PASS
- `go build -o kdoctor.exe ./cmd/kdoctor` BUILD_OK
- End-to-end smoke test against `examples/bad-project` (Health Score: 82
  deterministic, 6 required findings matched).

### Notes

- Detekt SARIF 2.1.0 input validation in `sarif.Parse`.
- Detekt `--config` REPLACE-aware seeded config (`examples/bad-project/detekt.yml`).
- File path normalization handles Unix, Windows, mixed-slashes, and UNC
  via `internal/core/pathutil`.

---

## Future (post-v1.0.0)

See `docs/HANDOVER.md` §13 Round-3 release prep backlog (CI workflows,
README public rewrite, Makefile + Dockerfile, `kdoctor init` bootstrapper,
MobiAI Graph integration end-to-end).
