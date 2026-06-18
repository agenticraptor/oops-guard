# Configuration

Configuration is **optional** — oops-guard works with no file at all. The file
exists only to let you change three things: the severity threshold that
interrupts your shell, an allow-list of commands you trust, and the model used
by `explain`.

## Location

The file lives at `~/.config/oops-guard/config.toml` (honoring `XDG_CONFIG_HOME`).
Run `oops-guard config path` to print the exact location, or set
`OOPS_GUARD_CONFIG` to override it.

Create a documented starter with:

```bash
oops-guard config init
```

## Reference

```toml
# Interrupt your shell when a command is at least this dangerous.
# One of: "caution", "danger", "critical".  Default: "danger".
min_severity = "danger"

# Command lines matching any of these regular expressions are never flagged.
# Use it to silence a pattern you run on purpose.
#   allow = ['^bin/clean-tmp\b', 'rm -rf .*/\.cache']
allow = []

[ai]
# Used only by `oops-guard explain`, never by the shell guard.
enabled  = false
provider = ""   # anthropic | openai | ollama (empty = auto-detect from env)
model    = ""   # empty = provider default
```

### `min_severity`

Controls what interrupts you in a hooked shell:

| Value | You get prompted for… |
|-------|------------------------|
| `caution` | everything oops-guard flags, including recoverable actions |
| `danger` *(default)* | irreversible and catastrophic commands |
| `critical` | only the catastrophic ones (`rm -rf /`, `dd` to a disk, `DROP DATABASE`) |

`check` always shows every finding regardless of this setting — the threshold
only affects the live shell guard.

### `allow`

A list of regular expressions matched against the **full command line**. A match
means oops-guard stays silent. Keep patterns tight: a broad pattern can switch
off protection more than you intend.

```toml
allow = [
  '^make clean\b',            # your Makefile's clean target
  '^bin/reset-fixtures\b',    # a script that legitimately uses rm
]
```

### `[ai]`

Only `oops-guard explain` ever consults a model. Provider auto-detection prefers
`ANTHROPIC_API_KEY`, then `OPENAI_API_KEY`, then a local Ollama. Override the
model per-invocation with `--provider` / `--model`, or pin it here.

**API keys are never stored in this file** — they're read from the environment
(`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`).

## Environment variables

| Variable | Effect |
|----------|--------|
| `OOPS_GUARD_CONFIG` | Full path to the config file (overrides the default) |
| `OOPS_GUARD_MODEL` | Default model for `explain` (overridden by `--model`) |
| `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` | Enable the respective cloud provider |
| `OLLAMA_HOST` | Where to reach a local Ollama (default `http://localhost:11434`) |
| `NO_COLOR` | Disable colored output everywhere |

## Inspecting the effective config

```bash
oops-guard config show
```

```text
config file: /Users/you/.config/oops-guard/config.toml
min_severity: danger
allow:        0 pattern(s)
ai.enabled:   false
```
