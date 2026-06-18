# Shell integration

The guard works by hooking the moment **between you pressing Enter and the shell
running your command**. Each shell exposes a different mechanism, so the hook is
slightly different in each — but the contract is the same: oops-guard inspects
the command, and if it's dangerous it prompts you on your terminal and can stop
the command from ever running.

Install the hook by adding one line to your shell's startup file:

| Shell | Add to | Line |
|-------|--------|------|
| zsh | `~/.zshrc` | `eval "$(oops-guard init zsh)"` |
| bash | `~/.bashrc` | `eval "$(oops-guard init bash)"` |
| fish | `~/.config/fish/config.fish` | `oops-guard init fish \| source` |

Then restart your shell (or `source` the file). Verify with `oops-guard doctor`.

> `oops-guard init <shell>` just prints the snippet — read it before you eval it
> if you like. The snippets live in
> [`internal/shell/hooks/`](../internal/shell/hooks).

## How each hook works

### zsh — `accept-line` widget

zsh lets you replace the widget bound to Enter. The hook wraps `accept-line` so
that pressing Return first runs `oops-guard guard` on the current buffer. If you
decline, the command is **not** executed and stays in your edit buffer so you can
fix or clear it. This is the cleanest "before you hit Enter" interception of the
three.

### bash — `DEBUG` trap + `extdebug`

bash has no Enter hook, so the integration uses a `DEBUG` trap with
`shopt -s extdebug`: when the trap returns non-zero, bash skips the command. The
hook restricts itself to **top-level, interactive** commands.

Caveats to be aware of:

- It inspects commands you type at the prompt, **not** commands run inside your
  functions or scripts.
- The `DEBUG` trap is a shared, single-owner mechanism in bash. Tools like
  **Atuin**, **Starship**, and anything built on **bash-preexec** also install a
  `DEBUG` trap, and whoever runs last wins — so loading oops-guard can disable
  them, or they can disable oops-guard, depending on order in your `~/.bashrc`.
  oops-guard installs its trap where you `eval` the snippet, so place that line
  **after** those tools if you want oops-guard to take effect (note this may then
  interfere with their command capture). bash has no clean way to chain
  `DEBUG` traps; **zsh and fish don't have this limitation**, so on bash with a
  preexec-based tool, prefer running oops-guard's checks via `oops-guard check`
  in your own wrapper, or use zsh/fish for the seamless guard.

### fish — Enter key binding

fish binds Enter (`\r` and `\n`) to a function that runs `oops-guard guard` on
the command line before calling the built-in execute. Declining repaints the
prompt with your command intact.

## Fail-safe by design

A guard that sits in front of every command must never be able to break your
shell, so every hook **fails open**:

- It first checks that `oops-guard` is actually installed; if not, the hook does
  nothing and your command runs normally.
- It blocks a command **only** when `oops-guard guard` exits with code `10` — the
  status that means "you looked at the warning and declined." Any other
  outcome — the binary erroring, timing out, a parse problem, anything — lets the
  command through.

In other words, the worst case if oops-guard misbehaves is that it stops
guarding; it can never wedge your terminal. (The bash `extdebug` mechanism can
cancel a command, which is exactly why this exit-code discipline matters there.)

## The confirmation prompt

When a command crosses your threshold, oops-guard writes the warning and reads
your answer **directly from `/dev/tty`**, not stdin — so it works even though the
hook's stdin may be the script being run. It also does its own minimal echo, so
it behaves correctly whether the shell left the terminal in cooked mode (bash,
fish) or raw mode (zsh's line editor).

- **danger** → a single-key `y/N`, defaulting to **No**.
- **critical** → you must type the full phrase `yes, do it`. The friction is the
  point: for `rm -rf /` or `DROP DATABASE`, a reflexive `y` shouldn't be enough.

If there's no terminal to prompt on (a non-interactive context), the guard
**does not block** — it surfaces the warning and allows the command, so it never
breaks scripts. The hook is only meant for interactive shells.

## Tuning & escape hatches

- Change what interrupts you with the `min_severity` setting — see
  [configuration.md](configuration.md).
- Allow-list commands you run intentionally (e.g. your own cleanup script) with
  the `allow` patterns — those are matched before anything is flagged, so they're
  the clean way to silence a command you trust.
- To turn the guard off for the rest of your current shell session, remove its
  hook:
  - bash: `trap - DEBUG`
  - zsh: `zle -A .accept-line accept-line`
  - fish: `bind \r execute; bind \n execute`

## Uninstalling

Remove the `oops-guard init …` line from your startup file and restart your
shell. Nothing else is left behind — there's no daemon and no background state.
