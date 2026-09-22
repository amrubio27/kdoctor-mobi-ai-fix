---
name: kdoctor
description: "Use when the user wants to audit, review or measure the health of an Android / Kotlin Multiplatform / Compose Multiplatform codebase: Health Score 0-100, antipatterns in Compose, coroutines, lifecycle, architecture and security, with file:line findings. Pre-flight checks that the `kdoctor` binary is on PATH; if not, suggests the install one-liner without running it."
license: MIT
compatibility: [claude-code, cursor, copilot, codex, gemini]
platforms: [android, kmp]
---

# kdoctor — Android / KMP / Compose health audit

Static analysis that produces a **reproducible number** and **findings with file, line and column**, so an agent can act on them instead of guessing. Complements guidance skills: they say how to write code, kdoctor measures what is already written.

## When to invoke this skill

- "audita mi proyecto", "¿cómo está de salud este código?", "pásale el doctor"
- "encuentra antipatrones de Compose / corrutinas / arquitectura"
- Before a release or a PR review, to see what regressed.
- After a refactor, to compare the score against the previous run.
- When the user wants a shareable report (HTML or Markdown) of code quality.

**Skip** when the user asks how to *write* something (that is a guidance skill), when the question is about a single known file, or when they want a build/test run rather than static analysis.

## Pre-flight

Check **once** per conversation, not before every command:

1. Verify `kdoctor` is on PATH (`kdoctor --version`).
2. If it is **not** installed, tell the user once and stop:
   > "No encuentro `kdoctor`. Podés instalarlo con:
   > `irm https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.ps1 | iex` (Windows)
   > `curl -fsSL https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.sh | sh` (macOS/Linux)"

   Do **not** install it yourself. Wait for explicit confirmation.
3. The first scan downloads detekt (~50 MB) into `~/.kdoctor/tools/` and needs a **JDK 11-21**. Mention this once if the scan is slow to start. `kdoctor doctor` reports what is missing.

## Commands

### `kdoctor scan`

```bash
kdoctor scan                      # console summary
kdoctor scan --json               # structured, for agents (schema v3)
kdoctor scan --summary            # score + top clusters only
kdoctor scan --diff main          # only findings in what changed vs main
kdoctor scan --html               # self-contained report to share
kdoctor scan --sarif              # GitHub Code Scanning
kdoctor scan --fail-below 80      # non-zero exit for CI
```

Prefer `--json` when you are going to act on the results, and `--diff main` when reviewing a change: it keeps the agent focused on what the user just wrote instead of the whole backlog.

Findings carry `id`, `cluster`, `severity`, `file`, `line`, `column`, `message` and usually a `fixHint`. Clusters: `architecture`, `compose-performance`, `coroutines`, `lifecycle`, `security`, `testing`, `error-handling`, `naming`, `complexity`, `dead-code`, `magic-numbers`, `formatting`, `clean-code`, `kmp`, `memory`, `accessibility`.

A **partial scan** warning on stderr means detekt could not run, so only the native rules were evaluated. The message says why and how to fix it — relay it to the user rather than presenting the score as complete.

### `kdoctor init`

Generates `kdoctor.config.yaml` and a `detekt.yml` tuned for the stack. Run it when the user wants to customise severities or excludes. Use `--force` to overwrite.

```bash
kdoctor init --type=cmp     # android | kmp | cmp
```

### `kdoctor doctor`

Reports the environment: Java, detekt, Gradle, and how many rules the next scan will actually evaluate. Use it when a scan came back partial.

### `kdoctor-mcp`

An MCP stdio server exposing `kdoctor_scan`, `kdoctor_rules`, `kdoctor_init`, `kdoctor_doctor` and `kdoctor_fix_suggest`. Configure it with `KDOCTOR_BIN` pointing at the `kdoctor` binary.

## Recommended flow

1. If MobiAI Graph is available, narrow the blast radius first:
   `mobiai graph context "audit module app"` — then scan only what matters.
2. `kdoctor scan --diff main --json`
3. Parse `findings[]`, order by severity, and group by cluster. Lead with `security` and `architecture`: those are weighted highest and are the ones that bite later.
4. Fix, then re-scan and show the **delta** in score. A number that moves is more convincing than a number that is merely low.

## Reading the score

The Health Score is 0-100, penalised by severity and cluster weight, normalised by KLOC so a large project is not punished for being large. Security and architecture errors are deliberately **not** diluted by project size.

Treat it as a trend, not a grade: the useful question is "did this change make it better or worse?", not "is 72 good?".

## Working on the kdoctor repo itself

Read `HONEY.md` first. Key rules: plan before coding, self-review before declaring done, never delete tests without replacing them, and validate with `gofmt -l .`, `go vet ./...`, `go test ./...` and `go build ./...`.
