# Security Policy

## Supported versions

The latest released minor version receives security fixes. Please upgrade to the
most recent release before reporting an issue.

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Instead, use GitHub's private vulnerability reporting:
[**Report a vulnerability**](https://github.com/agenticraptor/oops-guard/security/advisories/new).
This keeps the report confidential between you and the maintainers until a fix is
ready.

Please include:

- A description of the issue and its impact.
- Steps to reproduce (a minimal proof of concept is ideal).
- Affected version(s), shell, and platform.

We aim to acknowledge reports within **72 hours** and to provide a remediation
timeline after triage. We will credit reporters in the release notes unless you
prefer to remain anonymous.

## Threat model — please read

oops-guard is a **safety net, not a sandbox.** It reduces the chance of an
expensive accident; it cannot stop a determined or obfuscated command. Keep this
in mind:

- **Detection is best-effort.** The rules recognize common destructive patterns.
  A command can be written so the rules don't match it (unusual quoting, an
  alias, an environment variable holding the dangerous part, a wrapper script).
  A missing warning is **never** a guarantee that a command is safe.
- **The guard inspects, it does not sandbox.** When you approve a command, it
  runs with your full privileges. oops-guard never adds isolation.
- **It fails open — by design.** The shell hooks check that the binary exists and
  block a command **only** on the explicit "you declined" exit code (`10`). If
  oops-guard is missing, errors, times out, or returns anything else, your
  command runs normally. A safety tool must never be able to brick your shell, so
  it never does.
- **The shell hook runs `oops-guard` on the command text before execution.** It
  reads the command string and, for the impact preview, performs **read-only**
  filesystem and `git` queries on the target paths. It never executes the command
  itself and never writes to your files.
- **Hardened git inspection.** The impact preview may run `git status` /
  `git clean -n` inside a directory you *named* (e.g. an `rm` target), which could
  be an untrusted repository. Because git executes some config keys on read
  (notably `core.fsmonitor`), oops-guard neutralizes those vectors on every
  invocation: it overrides `core.fsmonitor`, `core.hooksPath`, and the pager via
  command-line `-c` (which beats repo config), disables prompts and optional
  locks, and restricts git's allowed protocols. It only ever runs local,
  read-only queries — never a network operation, never your command.
- **No network, no telemetry, by default.** The shell guard makes no network
  calls. The only feature that can reach the network is `oops-guard explain`,
  which you invoke deliberately and which sends the *command text* (not your
  files) to the model provider you choose. Note that a command line can itself
  contain a secret (e.g. `mysql -p'…'`); `explain` will transmit it, so avoid
  `explain` on commands with inline credentials, or use `--provider ollama` to
  keep everything local.
- **Allow-list with care.** Patterns in `allow` silence the guard for matching
  commands. A broad pattern can switch off protection more than you intend.
