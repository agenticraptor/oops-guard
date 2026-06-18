# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-16

### Added

- Initial release. 🎉
- A deterministic, dependency-free command classifier that flags genuinely
  destructive commands across many tools:
  - **Filesystem:** `rm -r`/`-rf` (with a precise file-and-size preview),
    catastrophic targets (`/`, `~`, `$HOME`, system directories),
    `find … -delete`/`-exec rm`, and `shred`.
  - **Disks:** `dd of=/dev/…`, `mkfs.*`, `wipefs`/`blkdiscard`, and redirects
    onto a raw device.
  - **git:** `push --force`/`--mirror`, `reset --hard`, `clean -fdx`,
    discarding `checkout`/`restore`, `branch -D`, `stash clear`,
    and `filter-branch`.
  - **Databases:** `DROP DATABASE`/`DROP TABLE`, `TRUNCATE`, and `DELETE`/`UPDATE`
    with no `WHERE`, plus `redis-cli FLUSHALL`.
  - **Permissions:** `chmod -R 777` and `chmod`/`chown` on system paths.
  - **Infra & shell hazards:** `terraform destroy`, `docker volume rm`/`prune`,
    `kubectl delete --all`, `curl | sh`, and fork bombs.
- A precise, **read-only** impact preview: file counts, total size, git
  uncommitted-change counts, and notes (outside the current directory, symlink,
  wildcard, missing) — so warnings say *what specifically* is at stake.
- Severity levels — safe, caution, danger, and critical — with a configurable
  threshold that decides what interrupts your shell.
- Shell integration for **bash** (DEBUG trap), **zsh** (accept-line widget), and
  **fish** (Enter keybind), installed with `oops-guard init <shell>`. The hooks
  **fail open** — they block a command only on the explicit decline exit code, so
  a missing or erroring binary can never block your shell.
- Security-hardened git inspection: the impact preview overrides `core.fsmonitor`,
  `core.hooksPath`, and the pager and restricts git's protocols, so
  auto-inspecting an untrusted repository can't execute its config.
- A `/dev/tty` confirmation prompt: single-key y/N for danger, type-to-confirm
  for critical commands.
- Commands: `check`, `guard`, `init`, `explain`, `rules`, `doctor`, `config`,
  and `version`.
- An **opt-in** `explain` command that asks a model (Anthropic, OpenAI, or local
  Ollama) for a plain-English second opinion. The shell guard itself never needs
  a model or a network.
- Output formats for `check`: styled terminal, plain text, and JSON.
- Optional TOML configuration: severity threshold, an allow-list of trusted
  command patterns, and model settings.
- A single static binary with no runtime, daemon, or telemetry.

[Unreleased]: https://github.com/agenticraptor/oops-guard/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/agenticraptor/oops-guard/releases/tag/v0.1.0
