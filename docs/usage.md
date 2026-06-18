# Usage

oops-guard has two faces:

1. **The shell guard** — once installed, it sits in front of your shell and
   interrupts *only* when you're about to run something genuinely destructive.
   You don't run anything by hand; you just keep using your shell.
2. **The `check`/`explain` commands** — for inspecting a command on demand,
   scripting, or understanding a scary one-liner before you trust it.

## Commands

```text
oops-guard check [command...]     Analyze a command; print what's dangerous
oops-guard explain [command...]   check + a plain-English model second opinion
oops-guard init <bash|zsh|fish>   Print the shell hook to install
oops-guard guard -- <command>     Hook entry point (used by the shells)
oops-guard rules                  List every detection rule
oops-guard doctor                 Check your environment & whether the hook is installed
oops-guard config <init|path|show>  Manage the optional config file
oops-guard version                Print version information
```

## `check`

Classify a command and print the verdict. It reads your filesystem to build the
impact preview but **never** runs the command or changes anything.

```bash
oops-guard check 'rm -rf build dist'
oops-guard check 'git push --force origin main'
oops-guard check 'psql -c "DROP TABLE users"'
```

Flags:

| Flag | Description |
|------|-------------|
| `-f, --format` | `term` (default) · `plain` · `json` |
| `--no-color` | Disable colored output |
| `--exit-code` | Exit `2` if the command is dangerous (handy in scripts/CI) |
| `--no-impact` | Skip the filesystem/git preview (pure, instant) |

Quote the command, or pass it after the flags:

```bash
oops-guard check --format json 'dd if=img of=/dev/sda' | jq .max_severity
# "critical"
```

## `explain`

Everything `check` does, then — if a model is available — a short, plain-English
account of what the command does and what could go wrong. This is the **only**
command that can use a model, and it's opt-in.

```bash
export ANTHROPIC_API_KEY=sk-ant-...        # or OPENAI_API_KEY, or run Ollama
oops-guard explain 'tar xzf bundle.tgz -C / --strip-components=1'
```

Only the **command text** is sent to the provider you choose — never your files.
Use `--provider ollama` to keep everything on your machine.

## `init`

Print the integration snippet for your shell and add it to your startup file:

```bash
eval "$(oops-guard init zsh)"      # ~/.zshrc
eval "$(oops-guard init bash)"     # ~/.bashrc
oops-guard init fish | source      # ~/.config/fish/config.fish
```

See [shell-integration.md](shell-integration.md) for exactly how each hook works
and its caveats.

## `doctor`

A quick health check — version, detected shell, whether the hook looks
installed, whether `git` is available for richer previews, your config, and
whether a model key is present.

```bash
oops-guard doctor
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success / not dangerous / command allowed |
| `1` | A runtime error, or (from `guard`) the command was declined |
| `2` | `check --exit-code` on a dangerous command |
