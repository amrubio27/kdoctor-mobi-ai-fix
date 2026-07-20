# kdoctor · Android / KMP / CMP Health Scanner

[![CI](https://github.com/adkd/adkd/actions/workflows/ci.yml/badge.svg)](https://github.com/adkd/adkd/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> **A single CLI that audits Android, Kotlin Multiplatform and Compose Multiplatform projects, computes a 0–100 Health Score, and suggests AI-powered fixes.**

kdoctor runs [Detekt](https://detekt.dev/) under the hood, maps every finding to a curated rule catalog, and turns the result into actionable reports: rich console, JSON, SARIF, or the bundled HTML dashboard. It can also emit findings straight to the MobiAI Graph.

Inspired by [`react-doctor`](https://github.com/millionco/react-doctor), built for Kotlin teams.

---

## ✨ Features

| Feature | What it means |
|---|---|
| **Health Score 0–100** | One number that summarises the quality of your codebase. |
| **78 curated rules** | 11 live V1 rules + 53 planned + 14 default Detekt mappings, grouped by cluster (`compose-*`, `coroutine-*`, `security-*`, `architecture-*`, …). |
| **Multiple output formats** | Rich console, JSON schema v3, SARIF 2.1.0, HTML dashboard, or MobiAI Graph JSONL. |
| **Diff-aware scanning** | `kdoctor scan --diff main` reports only the findings introduced on the current branch. |
| **Baseline suppression** | `kdoctor scan --baseline baseline.xml` ignores known issues. |
| **AI Fixer** | `kdoctor fix --ai` generates context-aware patches via Claude Code / Cursor / Gemini / MobiAI (safe `suggest` mode by default). |
| **Team config** | `kdoctor.config.yaml` lets you disable rules, change severities, exclude paths, and set a quality gate. |
| **Single Go binary** | Ships as one executable; uses your existing Detekt or Gradle wrapper (JVM still required by Detekt/Gradle). |

---

## 🚀 Quick start

```bash
# 1. Install
#    - Pre-built binary: see Releases
#    - Or with Go:
go install github.com/adkd/adkd/cmd/kdoctor@latest

# 2. Bootstrap your project
cd my-android-project
kdoctor init        # creates kdoctor.config.yaml

# 3. Scan
kdoctor scan                    # rich console + Health Score
kdoctor scan --json             # CI-friendly JSON
kdoctor scan --sarif            # GitHub Code Scanning
kdoctor scan --diff main        # only new findings
```

---

## 📦 Installation

### Option A — Download a release binary

Grab the latest binary for your platform from the [Releases](https://github.com/adkd/adkd/releases) page and put it on your `PATH`.

### Option B — Install with Go

```bash
go install github.com/adkd/adkd/cmd/kdoctor@latest
```

Requires Go 1.22+.

### Option C — Build from source

```bash
git clone https://github.com/adkd/adkd.git
cd adkd
go build -o kdoctor ./cmd/kdoctor
```

### Requirements

- **Go** 1.22+ (only for building / `go install`).
- **Detekt CLI** or a Gradle wrapper with Detekt configured.
- **JDK** 17+ (used by Detekt/Gradle).

Check your environment with:

```bash
kdoctor doctor
```

---

## 🛠️ Usage

### Scan

```bash
kdoctor scan --project-dir ./my-app --prefer-standalone --detekt-bin /path/to/detekt
```

| Flag | Description |
|---|---|
| `--json` | Output JSON schema v3. |
| `--sarif` | Output SARIF 2.1.0. |
| `--out <path>` | Write report to a file. |
| `--project-dir <dir>` | Project to scan (default: current directory). |
| `--prefer-standalone` | Use the `detekt` binary instead of `./gradlew`. |
| `--detekt-bin <path>` | Explicit path to the Detekt binary. |
| `--diff <ref>` | Only report findings on lines changed since `<ref>`. |
| `--baseline <path>` | Suppress findings listed in a baseline file. |
| `--mobiai` | Emit findings to `.mobiai/graph/findings.jsonl`. |
| `--fail-below <N>` | Exit with non-zero code if Health Score `< N`. |

### Fix with AI

```bash
# Generate fixes.md without touching source code (default, safe)
kdoctor fix --ai --mode suggest

# Review each fix interactively
kdoctor fix --ai --mode interactive

# Auto-apply patches (use with care; still being hardened for v1.0.1)
kdoctor fix --ai --mode auto
```

### List rules

```bash
kdoctor rules
```

### HTML dashboard

```bash
kdoctor scan --json --out report.json
# Then in dashboard/
cd dashboard
npm install
npm run dev
```

---

## ⚙️ Configuration

`kdoctor init` creates a `kdoctor.config.yaml` file:

```yaml
projectType: android

paths:
  kotlin:
    - "app/src/main/**/*.kt"

rules:
  coroutine-global-scope: error
  sec-log-pii: error
  security: warning   # cluster-level override

excludes:
  - "**/generated/**"
  - "**/build/**"

score:
  failBelow: 80

aiFixer:
  provider: auto
  mode: suggest
```

- `rules` accepts rule IDs or cluster IDs. Rule-level severity wins over cluster-level.
- Use `off`, `disabled`, or `none` to disable a rule/cluster.
- `excludes` supports glob patterns (`**` for any depth).

---

## 🔁 CI / GitHub Actions

kdoctor ships with a fast CI gate in `.github/workflows/ci.yml`:

```yaml
name: CI Fast Gate
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23.x"
      - run: gofmt -l .
      - run: go vet ./...
      - run: go test -race -count=1 ./...
      - run: go build -o kdoctor ./cmd/kdoctor
```

Add kdoctor to your own workflow:

```yaml
- name: Quality gate
  run: |
    go install github.com/adkd/adkd/cmd/kdoctor@latest
    kdoctor scan --json --fail-below 80
```

---

## 🏥 Health Score

The score is computed from the mapped findings:

```
Health Score = 100 - errors*5 - warnings*2 - info*0.5
```

Use `--fail-below <N>` to break CI when the score drops below your threshold.

---

## 🆚 kdoctor vs. Detekt vs. Android Lint

| | kdoctor | Detekt | Android Lint |
|---|---|---|---|
| **Scope** | Android / KMP / CMP | Kotlin/JVM | Android-only |
| **Output** | Score, JSON, SARIF, HTML, MobiAI | SARIF, XML, HTML | XML, HTML |
| **Rule mapping** | Curated catalog + native Go detectors | Detekt rules only | Lint rules only |
| **AI fixes** | ✅ Built-in | ❌ | ❌ |
| **Diff-aware** | ✅ `--diff` | ❌ | ❌ |
| **Baseline** | ✅ `--baseline` | ✅ | ✅ |

---

## 🗺️ Roadmap

- [x] Phase 1 — Inspector CLI with Detekt SARIF
- [x] Phase 1.5 — 78-rule catalog + FixHint
- [x] Tier 2 — Gradle plugin + AI fixer scaffold
- [x] Tier 3 — HTML dashboard + team config overrides
- [x] Round-2 — Defensive hardening (rulemap, patchguard, diff/baseline paths, qualityprompt, provider cache)
- [x] Round-3 #9 — CI fast gate + gofmt/vet/race checks
- [ ] Round-3 #11 — `kdoctor fix --mode auto` patch apply + rollback
- [ ] Round-3 #12 — E2E tests on a real Android project
- [ ] Round-3 #13 — Makefile + Dockerfile
- [ ] Round-3 #14 — `kdoctor init` project bootstrapper
- [ ] v1.0.0 release — pre-built binaries + GitHub release notes

See [`docs/HANDOVER.md`](docs/HANDOVER.md) for the full continuity document.

---

## 🤝 Contributing

Contributions are welcome. Please make sure your changes pass:

```bash
gofmt -l .        # should produce no output
go vet ./...
go test -race -count=1 ./...
go build -o kdoctor ./cmd/kdoctor
```

Read [`docs/HANDOVER.md`](docs/HANDOVER.md) before making substantial changes — it contains the project’s conventions, gotchas, and current state.

---

## 📄 License

MIT — see [`LICENSE`](LICENSE).
