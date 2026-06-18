<div align="center">

# 🛟 oops-guard

### Catch the dangerous command — *before* you hit enter.

`oops-guard` sits in front of your shell and watches for the commands you'll
regret: `rm -rf`, a force-push to `main`, `DROP TABLE`, `dd` to a disk. When it
sees one, it stops and tells you **exactly what would be lost** — how many files,
which database, whether you have uncommitted work — and asks. Everything else
runs untouched. It's a safety net, **not** a confirm-everything nag.

[![CI](https://github.com/agenticraptor/oops-guard/actions/workflows/ci.yml/badge.svg)](https://github.com/agenticraptor/oops-guard/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/agenticraptor/oops-guard?sort=semver)](https://github.com/agenticraptor/oops-guard/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/agenticraptor/oops-guard.svg)](https://pkg.go.dev/github.com/agenticraptor/oops-guard)
[![Go Report Card](https://goreportcard.com/badge/github.com/agenticraptor/oops-guard)](https://goreportcard.com/report/github.com/agenticraptor/oops-guard)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

> **Try it in one line — no install, no signup, no config:**
>
> ```bash
> go run github.com/agenticraptor/oops-guard/cmd/oops-guard@latest check 'rm -rf build dist'
> ```

<!--
  📸 Replace this block with a 15–20s screen-capture GIF: install the hook, then
  type `rm -rf build` and watch oops-guard catch it with the file count. The
  hero GIF is the single biggest driver of stars — record it once, drop it at
  docs/demo.gif, then uncomment:

  <p align="center"><img src="docs/demo.gif" alt="oops-guard demo" width="760"></p>
-->

```text
$ rm -rf build dist node_modules

  ☢ oops-guard  this looks destructive

  ✖ DANGER    Recursive force-delete
     Permanently removes build, dist, node_modules. rm does not use the
     trash or Recycle Bin, so this cannot be undone.
     build        — 1,204 files · 2.3 GB
     dist         — 392 files · 1.1 GB · git repo with 3 uncommitted changes
     node_modules — ≥ 20,000 files · 412 MB

  Proceed? [y/N] ▏
```

That `git repo with 3 uncommitted changes` line is the whole point. A blind
"are you sure?" can't tell you that you're about to delete work you never
committed. oops-guard can.

## The story you've lived

You meant to type `rm -rf ./build`. Your finger slipped and it was `rm -rf / build`.
Or you force-pushed the wrong branch. Or you ran a `DELETE` and forgot the
`WHERE`. The command obeyed instantly and perfectly — that's the horror of it.

`safe-rm` only knows about `rm`. A blanket "confirm every command" alias trains
you to mash `y` until the muscle memory betrays you on the one that mattered.
oops-guard is different: it stays **silent** on the thousands of safe commands
and speaks up **only** for the genuinely destructive few — with the specific
detail you need to make the call.

## Why you'll like it

- **It tells you what you'd actually lose.** Not "are you sure?" but
  *"1,204 files · 2.3 GB · 3 uncommitted changes."* For databases it names the
  table; for a force-push it explains whose history you'd rewrite.
- **It's specific, not naggy.** `rm file.txt`, `git push origin feature`,
  `DELETE … WHERE id = 5` all pass in silence. It catches the catastrophes, not
  your daily routine. Every rule ships with a test proving it stays quiet.
- **Tiered friction.** Plain `danger` asks a one-key `y/N`. A `critical` command
  (`rm -rf /`, `dd` to a disk, `DROP DATABASE`) makes you *type a phrase* — so a
  reflexive `y` can't wipe your laptop.
- **Works where you work.** One hook for **bash**, **zsh**, and **fish**, caught
  the instant you press Enter.
- **It can't brick your shell.** The hooks fail *open*: if oops-guard is ever
  missing, slow, or errors, your commands run normally. It only ever stops a
  command when **you** decline one. A safety net should never become the hazard.
- **No daemon, no network, no telemetry.** A single static binary that reads your
  command and (read-only) the files it would touch. Nothing leaves your machine.
- **Scriptable.** `oops-guard check --json` for tooling and CI gates.

## Install

### `go install`

```bash
go install github.com/agenticraptor/oops-guard/cmd/oops-guard@latest
```

### Pre-built binaries

Grab a binary for your OS/arch from the
[**Releases**](https://github.com/agenticraptor/oops-guard/releases) page.

### Homebrew (macOS / Linux)

```bash
brew install agenticraptor/tap/oops-guard
```

> Available once the Homebrew tap is published — see the note in
> [`.goreleaser.yaml`](.goreleaser.yaml) to enable it.

### From source

```bash
git clone https://github.com/agenticraptor/oops-guard
cd oops-guard
make install
```

## Quickstart

```bash
# 1. See what oops-guard thinks of a command — it never runs it:
oops-guard check 'git push --force origin main'

# 2. Install the shell guard (pick your shell), then restart your shell:
echo 'eval "$(oops-guard init zsh)"'  >> ~/.zshrc     # zsh
echo 'eval "$(oops-guard init bash)"' >> ~/.bashrc    # bash
echo 'oops-guard init fish | source'  >> ~/.config/fish/config.fish   # fish

# 3. That's it. Keep using your shell. oops-guard only speaks up when it matters.

# 4. Confirm everything's wired up:
oops-guard doctor
```

Full command reference, the rule catalog, and how each shell hook works live in
[**docs/**](docs):
[usage](docs/usage.md) ·
[rules](docs/rules.md) ·
[shell integration](docs/shell-integration.md) ·
[configuration](docs/configuration.md).

## How it works

```
  you press Enter
        │
        ▼
  shell hook (bash DEBUG trap · zsh accept-line · fish keybind)
        │  passes the command text to:
        ▼
  oops-guard guard ── parse pipeline ──► deterministic rules ──► findings
        │                                  (rm, git, dd, SQL,        │
        │                                   chmod, docker, …)        ▼
        │                                            read-only impact preview
        │                                            (file count · size · git state)
        ▼
   below threshold? ──► allow silently
        │
   at/above threshold ──► print the warning · ask on /dev/tty
        │
   approved ─► run it     declined ─► command never runs
```

The classifier is **deterministic and offline** — no model, no network — so it's
fast enough to run before every command you type. The impact preview only ever
**reads** the filesystem (and runs `git status`); it never executes your command
or changes a file. There's an optional [`oops-guard explain`](docs/usage.md#explain)
that asks a model for a plain-English second opinion, but the guard itself never
needs one.

## Privacy

oops-guard runs entirely on your machine and the shell guard makes **no network
connections** — no telemetry, no update checks. It reads your command and
performs read-only checks on the paths it would affect. The only feature that can
reach the network is `oops-guard explain`, which you invoke deliberately and
which sends the *command text* (never your files) to the model provider you pick;
use `--provider ollama` to keep even that fully local. See [SECURITY.md](SECURITY.md)
for the threat model — oops-guard is a safety net, not a sandbox.

## Contributing

Contributions are very welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). The best
contributions are **new detection rules**, each with a dangerous case it catches
*and* a safe case it leaves alone. Good first issues include `aws s3 rm
--recursive`, PowerShell support, and richer SQL parsing. Please also read our
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE) © oops-guard contributors.
