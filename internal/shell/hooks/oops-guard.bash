# oops-guard — bash integration.
#
# Catches a destructive command before it runs by inspecting it in a DEBUG
# trap. With `extdebug` enabled, a non-zero return from the trap cancels the
# command, so declining the prompt stops it from ever executing.
#
# Enable it by adding this to ~/.bashrc:
#
#     eval "$(oops-guard init bash)"
#
# Fails open: if oops-guard is missing or errors for any reason, your command
# runs normally. It blocks ONLY when you explicitly decline (exit code 10).
#
# Caveat: the DEBUG trap is a per-command hook. oops-guard restricts itself to
# top-level, interactive commands; it does not inspect commands run inside your
# functions or scripts.

if [ -n "${_OOPS_GUARD_BASH_LOADED:-}" ]; then
  return 0 2>/dev/null || true
fi
_OOPS_GUARD_BASH_LOADED=1

shopt -s extdebug

_oops_guard_check() {
  # Interactive shells only.
  case $- in *i*) ;; *) return 0 ;; esac
  # Skip subshells, completion, and anything running inside a function.
  [ "${BASH_SUBSHELL:-0}" -eq 0 ] || return 0
  [ -n "${COMP_LINE:-}" ] && return 0
  [ "${#FUNCNAME[@]}" -gt 1 ] && return 0
  # Fail open if the binary isn't on PATH (never block, never spam errors).
  command -v oops-guard >/dev/null 2>&1 || return 0

  local cmd="${BASH_COMMAND}"
  [ "${cmd}" = "${PROMPT_COMMAND:-}" ] && return 0
  case "${cmd}" in
    _oops_guard_* | oops-guard\ guard* | oops-guard\ init*) return 0 ;;
  esac

  oops-guard guard --shell bash -- "${cmd}"
  # Block ONLY on an explicit user-decline; any other status fails open.
  [ "$?" -eq 10 ] && return 1
  return 0
}

trap '_oops_guard_check' DEBUG
