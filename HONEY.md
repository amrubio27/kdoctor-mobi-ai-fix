# Honey — AI Guardrails for kdoctor

Honey is the self-review and guardrails layer for any AI agent that works on
the kdoctor repository. Its goal is to reduce hallucinations, prevent
regressions, and keep the codebase maintainable.

This repository also uses **Honey for Devs** (`green-pt/honey-for-devs`) as an
AI style guide. The style rules live in:

- `.clinerules/honey.md` (for Cline, Claude Code, and other `.clinerules`-based agents)
- `.cursor/rules/honey.mdc` (for Cursor)

`HONEY.md` adds **kdoctor-specific guardrails** on top of those style rules.
Read this file before writing or deleting any code. If a proposed change
violates one of the hard hooks, stop and ask the user for explicit approval.

---

## 1. Plan before code

1. State the goal in one sentence.
2. List the files you will touch and why.
3. List the tests you will add or update.
4. State the validation command you will run.
5. Wait for the user to approve the plan before starting substantive work,
   unless the change is trivial (< 10 lines, no public API change, no test
   deletion).

## 2. Self-review before finalizing

Before declaring a task done, answer these questions out loud:

- Did I change any public function, type, or CLI flag? If yes, did I update
  all callers and the README/help text?
- Did I delete or modify any existing test? If yes, did I replace it with an
  equivalent or better test?
- Did I run `go vet ./...`, `go test ./...`, and `go build ./...`? Did they pass?
- Did I run `gofmt -l .`? Is the output empty?
- Did I update `docs/HANDOVER.md` if the task changes the architecture,
  roadmap, or validation steps?
- Are there hardcoded local paths (e.g., `D:/tools/detekt.cmd`,
  `C:/Program Files/...`) that should not be committed?

## 3. Hard hooks

Never do the following without explicit user approval:

1. **Push to a remote repository** (`git push`).
2. **Run commands with elevated privileges** (sudo, doas, RunAs, AppleScript).
3. **Delete tests without replacing them** with equivalent or better coverage.
4. **Change public APIs without updating all callers** and documentation.
5. **Commit changes the user did not ask for** or that are unrelated to the
   current task.
6. **Run scripts that could touch production environments**.
7. **Install global packages or modify system configuration**.

## 4. Validation checklist

Every non-trivial change must pass:

```bash
gofmt -l .                    # must be empty
go vet ./...                  # must exit 0
go test ./...                 # must pass
go build -o kdoctor.exe ./cmd/kdoctor  # must succeed
go build ./cmd/kdoctor-mcp   # must succeed
```

## 5. MCP integration

kdoctor exposes a stdio MCP server:

```bash
go run ./cmd/kdoctor-mcp
```

Tools exposed:
- `kdoctor_scan` — run a scan and return a JSON/SARIF report.
- `kdoctor_rules` — list the curated rule catalog.
- `kdoctor_init` — bootstrap `kdoctor.config.yaml` and `detekt.yml`.
- `kdoctor_doctor` — diagnose environment dependencies.
- `kdoctor_fix_suggest` — generate AI fix suggestions without applying them.

Use the MCP server when an AI agent needs structured access to kdoctor from a
host IDE (Cursor, Claude Code, Cline, etc.). Set `KDOCTOR_BIN` to the absolute
path of the `kdoctor` binary when it is not in PATH.

## 6. Context boundaries

- Keep changes minimal and focused.
- Reuse existing helpers in `internal/` instead of reimplementing them.
- Do not touch `D:/Programacion/RickMortyApp` or any user project directories
  outside this repository.
- Do not commit the compiled `kdoctor.exe` binary.
