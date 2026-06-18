# Contributing to oops-guard

Thanks for your interest in contributing! oops-guard aims to be a small,
focused, dependency-light tool. The single most important design rule is below —
please read it before opening a PR.

## The golden rule: stay quiet on safe commands

oops-guard is only useful if people trust it not to nag. A rule that fires on an
everyday command is worse than no rule at all, because it trains people to mash
"yes" — and then they mash "yes" on the one that mattered.

**Every new rule must come with two tests: a dangerous case it catches, and a
realistic safe case it must leave alone.** If you can't write a tight safe case,
the rule is probably too broad.

## Getting started

```bash
git clone https://github.com/agenticraptor/oops-guard
cd oops-guard
go mod tidy        # fetch dependencies & populate go.sum
make build         # build into ./bin/oops-guard
make test          # run the unit tests
make demo          # watch it analyze a sample command
```

Requirements:

- Go 1.22 or newer
- (optional) `git` on your `PATH` — enables the uncommitted-change preview
- (optional) [`golangci-lint`](https://golangci-lint.run/) for `make lint`
- (optional) [`goreleaser`](https://goreleaser.com/) for `make snapshot`

## Development workflow

1. Fork the repo and create a feature branch from `main`.
2. Make your change, with tests.
3. Run the full check suite locally:
   ```bash
   make fmt vet test
   ```
4. Open a pull request. Fill in the PR template and link any related issue.

CI runs `gofmt`, `go vet`, `golangci-lint`, and the test suite on Linux, macOS,
and Windows. All checks must pass before review.

## Adding a detection rule

Most contributions are new rules. Here's the shape:

1. Add a `commandRule` (or `pipelineRule`) in `internal/analyzer/rules.go` — or a
   SQL pattern in `sql.go`. A rule is a pure function from a parsed `Command` to
   `[]Finding`; keep it free of filesystem access.
2. Pick the right `Severity`:
   - **Caution** — recoverable or scoped loss worth a glance.
   - **Danger** — irreversible loss of local work or data.
   - **Critical** — catastrophic, machine- or database-wide destruction.
3. If the rule deletes files, add an impact preview in `impact.go` so the
   warning can say *what specifically* is at stake — that precision is the whole
   point of the project.
4. Register it in `commandRules`/`pipelineRules` and add it to `Catalog()` in
   `catalog.go` so it shows in `oops-guard rules` and the docs.
5. Add a dangerous case to `TestDangerousCommands` **and** a safe case to
   `TestSafeCommands`.

## Coding guidelines

- **Keep dependencies minimal.** Prefer the standard library. New dependencies
  must be justified in the PR description.
- **Tolerant parsing.** The analyzer reads arbitrary command lines and must never
  panic on weird input — degrade to "no finding" instead.
- **The guard never runs the command and never writes to disk.** Impact previews
  are strictly read-only.
- **Speed matters.** The analyzer runs before every command in a hooked shell.
  Keep rules cheap and the filesystem walk bounded.
- **Format with `gofmt -s`** and keep `go vet` clean.

## Architecture at a glance

| Package | Responsibility |
|---------|----------------|
| `internal/analyzer` | Parse a command line and classify its danger (the core). |
| `internal/analyzer` (impact) | Read-only "what will be lost" preview: file counts, sizes, git state. |
| `internal/render` | Render an assessment to term / plain / JSON. |
| `internal/confirm` | Ask y/N (or type-to-confirm) on `/dev/tty`. |
| `internal/shell` | Embedded bash/zsh/fish integration snippets. |
| `internal/llm` | Tiny HTTP clients for Anthropic, OpenAI, Ollama. |
| `internal/explain` | The opt-in, plain-English model second opinion. |
| `internal/config` | Optional TOML config (threshold, allow-list, model). |
| `internal/cli` | Cobra commands: check, guard, init, explain, rules, doctor, config. |

## Good first issues

- New rules: `truncate -s 0` on an important file, `git update-ref -d`,
  `aws s3 rm --recursive`, `gcloud ... delete`, `npm/yarn cache clean --force`.
- A `--dry-run`/preview for the shell hooks that logs decisions without
  prompting, to help people tune their threshold.
- Windows PowerShell integration (`Remove-Item -Recurse -Force`, `Format-Volume`).
- More precise SQL parsing (multi-statement scripts, `WHERE 1=1`).

## Reporting bugs & requesting features

Use the [issue templates](https://github.com/agenticraptor/oops-guard/issues/new/choose).
For anything security-related, please follow [SECURITY.md](SECURITY.md) instead
of opening a public issue.

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).
